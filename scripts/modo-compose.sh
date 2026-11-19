#!/usr/bin/env bash
# Fase 1. Se ejecuta en vm1:   vagrant ssh vm1 -c "bash /vagrant/scripts/modo-compose.sh"
# Apaga Kubernetes, construye la imagen de la API, la guarda en build/ y levanta Docker Compose.
set -euo pipefail

if systemctl is-active --quiet k3s; then
  echo "== Apagando Kubernetes (k3s) =="
  sudo systemctl stop k3s
fi
sudo /usr/local/bin/k3s-killall.sh >/dev/null 2>&1 || true

for ip in 192.168.56.12 192.168.56.13; do
  if ping -c1 -W1 "$ip" >/dev/null 2>&1; then
    echo "AVISO: $ip esta encendida y le quita CPU al PC. Apagarla con: vagrant halt vm2 vm3"
  fi
done

echo "== Encendiendo Docker =="
sudo systemctl start docker.socket docker.service

cd /vagrant/docker
sudo docker compose down -v --remove-orphans >/dev/null 2>&1 || true

echo "== Construyendo la imagen de la API =="
sudo docker compose build api
sudo docker save libreria-api:1.1 -o /vagrant/build/libreria-api.tar

echo "== Levantando base y API =="
sudo docker compose up -d
for i in $(seq 1 60); do
  curl -sf http://localhost:8080/listo >/dev/null && break
  sleep 2
done
sudo docker compose ps
echo
curl -s http://localhost:8080/api/libros/1; echo
sudo docker compose exec -T db psql -U libreria -d libreria -tAc "SELECT 'Libros en la base: ' || COUNT(*) FROM libros"
echo "Listo. Siguiente: vagrant ssh jmeter -c \"bash /vagrant/pruebas/correr.sh compose 1\""
