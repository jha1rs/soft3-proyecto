package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
)

type App struct {
	db      *sql.DB
	replica string
}

type Libro struct {
	ID        int     `json:"id"`
	ISBN      string  `json:"isbn"`
	Titulo    string  `json:"titulo"`
	Autor     string  `json:"autor"`
	Pais      string  `json:"pais_autor"`
	Categoria string  `json:"categoria"`
	Anio      int     `json:"anio"`
	Precio    float64 `json:"precio"`
	Stock     int     `json:"stock"`
}

const selectLibro = `
SELECT l.id, l.isbn, l.titulo, a.nombre, a.pais, c.nombre, l.anio, l.precio, l.stock
FROM libros l
JOIN autores a ON a.id = l.autor_id
JOIN categorias c ON c.id = l.categoria_id`

func (a *App) rutas() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /salud", a.salud)
	m.HandleFunc("GET /listo", a.listo)
	m.HandleFunc("GET /api/libros", a.listarLibros)
	m.HandleFunc("GET /api/libros/buscar", a.buscarLibros)
	m.HandleFunc("GET /api/libros/{id}", a.obtenerLibro)
	m.HandleFunc("GET /api/categorias/estadisticas", a.estadisticas)
	m.HandleFunc("POST /api/pedidos", a.crearPedido)
	m.HandleFunc("GET /api/pedidos/{id}", a.obtenerPedido)
	// Cada respuesta dice que replica la atendio (sirve para ver el balanceo)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Replica", a.replica)
		m.ServeHTTP(w, r)
	})
}

func responder(w http.ResponseWriter, estado int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	json.NewEncoder(w).Encode(v)
}

func fallar(w http.ResponseWriter, estado int, msg string) {
	responder(w, estado, map[string]string{"error": msg})
}

func entero(r *http.Request, clave string, porDefecto, min, max int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(clave))
	if err != nil {
		return porDefecto
	}
	return int(math.Max(float64(min), math.Min(float64(max), float64(v))))
}

// GET /salud: el proceso esta vivo (no toca la base)
func (a *App) salud(w http.ResponseWriter, r *http.Request) {
	responder(w, http.StatusOK, map[string]string{"estado": "ok", "replica": a.replica})
}

// GET /listo: el proceso puede atender (la base responde)
func (a *App) listo(w http.ResponseWriter, r *http.Request) {
	if err := a.db.PingContext(r.Context()); err != nil {
		fallar(w, http.StatusServiceUnavailable, "base de datos no disponible")
		return
	}
	responder(w, http.StatusOK, map[string]string{"estado": "listo"})
}

func escanearLibros(filas *sql.Rows) ([]Libro, error) {
	defer filas.Close()
	libros := []Libro{}
	for filas.Next() {
		var l Libro
		if err := filas.Scan(&l.ID, &l.ISBN, &l.Titulo, &l.Autor, &l.Pais, &l.Categoria, &l.Anio, &l.Precio, &l.Stock); err != nil {
			return nil, err
		}
		libros = append(libros, l)
	}
	return libros, filas.Err()
}

// GET /api/libros?pagina=1&tamano=20&categoria=3
func (a *App) listarLibros(w http.ResponseWriter, r *http.Request) {
	pagina := entero(r, "pagina", 1, 1, 100000)
	tamano := entero(r, "tamano", 20, 1, 100)
	categoria := entero(r, "categoria", 0, 0, 1000)

	var filas *sql.Rows
	var err error
	if categoria > 0 {
		filas, err = a.db.QueryContext(r.Context(),
			selectLibro+` WHERE l.categoria_id = $1 ORDER BY l.id LIMIT $2 OFFSET $3`,
			categoria, tamano, (pagina-1)*tamano)
	} else {
		// Paginacion por rango de id en vez de OFFSET: con OFFSET la base recorre y descarta
		// todas las filas anteriores (en la pagina 450, unas 9.000). Con el indice de la llave
		// primaria salta directo al primer libro de la pagina: ~10 veces menos trabajo.
		// Los id de los libros son consecutivos, asi que el resultado es el mismo.
		filas, err = a.db.QueryContext(r.Context(),
			selectLibro+` WHERE l.id > $1 ORDER BY l.id LIMIT $2`, (pagina-1)*tamano, tamano)
	}
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	libros, err := escanearLibros(filas)
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	responder(w, http.StatusOK, map[string]any{"pagina": pagina, "tamano": tamano, "libros": libros})
}

// GET /api/libros/buscar?q=jardin
func (a *App) buscarLibros(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 3 {
		fallar(w, http.StatusBadRequest, "la busqueda necesita al menos 3 caracteres")
		return
	}
	filas, err := a.db.QueryContext(r.Context(),
		selectLibro+` WHERE l.titulo ILIKE '%' || $1 || '%' OR a.nombre ILIKE '%' || $1 || '%'
		ORDER BY l.id LIMIT 20`, q)
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	libros, err := escanearLibros(filas)
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	responder(w, http.StatusOK, map[string]any{"consulta": q, "libros": libros})
}

// GET /api/libros/{id}
func (a *App) obtenerLibro(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		fallar(w, http.StatusBadRequest, "id invalido")
		return
	}
	filas, err := a.db.QueryContext(r.Context(), selectLibro+` WHERE l.id = $1`, id)
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	libros, err := escanearLibros(filas)
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(libros) == 0 {
		fallar(w, http.StatusNotFound, "libro no encontrado")
		return
	}
	responder(w, http.StatusOK, libros[0])
}

// GET /api/categorias/estadisticas: libros, precio promedio y unidades por categoria
func (a *App) estadisticas(w http.ResponseWriter, r *http.Request) {
	filas, err := a.db.QueryContext(r.Context(), `
		SELECT c.nombre, COUNT(l.id), ROUND(AVG(l.precio), 2), COALESCE(SUM(l.stock), 0)
		FROM categorias c LEFT JOIN libros l ON l.categoria_id = c.id
		GROUP BY c.id, c.nombre ORDER BY c.nombre`)
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer filas.Close()
	type fila struct {
		Categoria string  `json:"categoria"`
		Libros    int     `json:"libros"`
		Promedio  float64 `json:"precio_promedio"`
		Unidades  int     `json:"unidades_en_stock"`
	}
	resultado := []fila{}
	for filas.Next() {
		var f fila
		if err := filas.Scan(&f.Categoria, &f.Libros, &f.Promedio, &f.Unidades); err != nil {
			fallar(w, http.StatusInternalServerError, err.Error())
			return
		}
		resultado = append(resultado, f)
	}
	responder(w, http.StatusOK, resultado)
}

type Pedido struct {
	ID             int64   `json:"id"`
	LibroID        int     `json:"libro_id"`
	Cantidad       int     `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
	Descuento      float64 `json:"descuento"`
	Total          float64 `json:"total"`
}

// POST /api/pedidos  {"libro_id": 10, "cantidad": 2}
// Descuenta el inventario y registra el pedido en una sola transaccion.
// Regla de negocio: 10 % de descuento desde 3 unidades.
func (a *App) crearPedido(w http.ResponseWriter, r *http.Request) {
	var p Pedido
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&p); err != nil {
		fallar(w, http.StatusBadRequest, "JSON invalido")
		return
	}
	if p.LibroID < 1 || p.Cantidad < 1 || p.Cantidad > 10 {
		fallar(w, http.StatusBadRequest, "libro_id debe ser positivo y cantidad entre 1 y 10")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	// El UPDATE con la condicion de stock evita vender unidades que no hay,
	// aunque lleguen pedidos simultaneos por varias replicas.
	err = tx.QueryRowContext(r.Context(),
		`UPDATE libros SET stock = stock - $2 WHERE id = $1 AND stock >= $2 RETURNING precio`,
		p.LibroID, p.Cantidad).Scan(&p.PrecioUnitario)
	if errors.Is(err, sql.ErrNoRows) {
		var existe bool
		a.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM libros WHERE id = $1)`, p.LibroID).Scan(&existe)
		if !existe {
			fallar(w, http.StatusNotFound, "libro no encontrado")
		} else {
			fallar(w, http.StatusConflict, "no hay unidades suficientes")
		}
		return
	}
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}

	bruto := p.PrecioUnitario * float64(p.Cantidad)
	if p.Cantidad >= 3 {
		p.Descuento = math.Round(bruto*0.10*100) / 100
	}
	p.Total = bruto - p.Descuento

	err = tx.QueryRowContext(r.Context(),
		`INSERT INTO pedidos (libro_id, cantidad, precio_unitario, descuento, total)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		p.LibroID, p.Cantidad, p.PrecioUnitario, p.Descuento, p.Total).Scan(&p.ID)
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	responder(w, http.StatusCreated, p)
}

// GET /api/pedidos/{id}
func (a *App) obtenerPedido(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		fallar(w, http.StatusBadRequest, "id invalido")
		return
	}
	var p Pedido
	err = a.db.QueryRowContext(r.Context(),
		`SELECT id, libro_id, cantidad, precio_unitario, descuento, total FROM pedidos WHERE id = $1`, id).
		Scan(&p.ID, &p.LibroID, &p.Cantidad, &p.PrecioUnitario, &p.Descuento, &p.Total)
	if errors.Is(err, sql.ErrNoRows) {
		fallar(w, http.StatusNotFound, "pedido no encontrado")
		return
	}
	if err != nil {
		fallar(w, http.StatusInternalServerError, err.Error())
		return
	}
	responder(w, http.StatusOK, p)
}
