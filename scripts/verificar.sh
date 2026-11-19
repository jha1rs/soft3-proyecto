#!/usr/bin/env bash
# Comprueba que la API responde bien. Se ejecuta en vm1:
#   vagrant ssh vm1 -c "bash /vagrant/scripts/verificar.sh"         (Docker Compose, puerto 8080)
#   vagrant ssh vm1 -c "bash /vagrant/scripts/verificar.sh 30080"   (Kubernetes)
PORT=${1:-8080}
B="http://localhost:$PORT"

if [ "$PORT" = "8080" ]; then
  sudo docker compose -f /vagrant/docker/docker-compose.yml ps
else
  kubectl -n libreria get pods -o wide
fi
echo

printf '%-36s %s\n' "RUTA" "CODIGO  TIEMPO"
for u in /salud /listo /api/libros/1 '/api/libros?pagina=450&tamano=2' '/api/libros?categoria=3&tamano=1' \
         '/api/libros/buscar?q=jardin' /api/categorias/estadisticas /api/libros/99999; do
  printf '%-36s ' "$u"
  curl -s -o /dev/null -w '%{http_code}    %{time_total}s\n' "$B$u"
done
echo

echo "Pedido de 3 unidades (debe tener descuento):"
curl -s -X POST -H 'Content-Type: application/json' -d '{"libro_id":5,"cantidad":3}' "$B/api/pedidos"; echo
echo "Pagina 450 (debe empezar en id 899):"
curl -s "$B/api/libros?pagina=450&tamano=2" | head -c 120; echo
echo "Replica que respondio:"
curl -s -D - -o /dev/null "$B/salud" | grep -i x-replica
