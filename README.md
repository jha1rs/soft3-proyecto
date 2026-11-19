# Librería: pruebas de carga en Docker Compose y Kubernetes

Proyecto final de *Principios y prácticas de Desarrollo de Software Orientado a Objetos*.

Una API REST de librería (Go + PostgreSQL, 10.000 libros) desplegada con Docker Compose y con
Kubernetes (k3s) en 1, 2 y 3 nodos, probada con JMeter. Todo corre en máquinas virtuales creadas
con Vagrant y VirtualBox.

| Carpeta | Contenido |
|---|---|
| `api/` | Código de la API y `Dockerfile` |
| `db/init.sql` | Crea las tablas y carga los datos (12 categorías, 1.000 autores, 10.000 libros) |
| `docker/` | `docker-compose.yml` (fase 1) |
| `k8s/` | Manifiestos de Kubernetes (fases 2 a 4) |
| `jmeter/plan.jmx` | Plan de pruebas |
| `pruebas/` | Scripts que corren las pruebas y resumen los resultados |
| `scripts/` | Scripts para vm1: pasar entre Docker Compose y Kubernetes |
| `resultados/` | CSV de JMeter, ubicación de los pods y consumo de CPU y memoria |
| `notebook/analisis.ipynb` | Análisis con tablas y gráficas (Google Colab) |
| `Vagrantfile` | Las 4 máquinas virtuales |

Instrucciones completas en [GUIA.md](GUIA.md).
