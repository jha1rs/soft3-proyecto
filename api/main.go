// API REST de una libreria: catalogo de libros, busqueda, estadisticas y pedidos.
// Hecha en Go solo con la libreria estandar y el driver de PostgreSQL (lib/pq),
// para que la imagen sea pequena (~8 MB), arranque en menos de un segundo y use poca memoria.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

func env(clave, porDefecto string) string {
	if v := os.Getenv(clave); v != "" {
		return v
	}
	return porDefecto
}

func main() {
	salud := flag.Bool("salud", false, "consulta /listo y termina (para el healthcheck de Docker)")
	flag.Parse()
	puerto := env("PORT", "8080")

	if *salud {
		cliente := http.Client{Timeout: 3 * time.Second}
		r, err := cliente.Get("http://127.0.0.1:" + puerto + "/listo")
		if err != nil || r.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		env("DB_HOST", "localhost"), env("DB_PORT", "5432"), env("DB_USER", "libreria"),
		env("DB_PASSWORD", "libreria"), env("DB_NAME", "libreria"))
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	pool, _ := strconv.Atoi(env("DB_POOL", "10"))
	db.SetMaxOpenConns(pool)
	db.SetMaxIdleConns(pool)
	db.SetConnMaxLifetime(30 * time.Minute)

	nombre, _ := os.Hostname()
	app := &App{db: db, replica: nombre}

	srv := &http.Server{
		Addr:              ":" + puerto,
		Handler:           app.rutas(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Apagado ordenado: Kubernetes manda SIGTERM al bajar replicas
	apagado := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, os.Interrupt)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
		close(apagado)
		os.Exit(0) // por si llega mientras todavia espera la base
	}()

	// El servidor HTTP arranca de una vez: /salud responde aunque la base aun no este lista
	// (asi Kubernetes no reinicia la replica), y /listo dice cuando ya se puede atender.
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	log.Printf("API escuchando en :%s (replica %s, pool %d)", puerto, nombre, pool)

	// La base puede tardar en estar lista: se reintenta hasta 2 minutos
	for i := 0; ; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			log.Printf("conectado a la base de datos")
			break
		}
		if i == 60 {
			log.Fatalf("no se pudo conectar a la base: %v", err)
		}
		log.Printf("esperando la base de datos: %v", err)
		time.Sleep(2 * time.Second)
	}
	<-apagado
}
