#!/usr/bin/env bash
# Fases 2 a 4. Se ejecuta en vm1:   vagrant ssh vm1 -c "bash /vagrant/scripts/modo-k8s.sh"
# Apaga Docker Compose, enciende k3s, importa la imagen y despliega la aplicacion.
set -euo pipefail

if systemctl is-active --quiet docker; then
  echo "== Apagando Docker Compose =="
  (cd /vagrant/docker && sudo docker compose down -v --remove-orphans) || true
  sudo systemctl stop docker.socket docker.service
fi
sudo iptables -P FORWARD ACCEPT

if [ ! -f /vagrant/build/libreria-api.tar ]; then
  echo "Falta build/libreria-api.tar. Ejecutar antes scripts/modo-compose.sh (construye la imagen)."
  exit 1
fi

echo "== Encendiendo Kubernetes (k3s) =="
sudo systemctl start k3s
for i in $(seq 1 60); do
  kubectl get nodes >/dev/null 2>&1 && break
  sleep 2
done

# Despues de apagar y encender k3s, el nodo puede tardar unos minutos en volver a Ready
# (recrea la red de los pods). Sin esto, los pods nuevos quedan esperando sin avisar.
echo "== Esperando a que vm1 este Ready =="
kubectl wait --for=condition=Ready node/vm1 --timeout=600s

sudo k3s ctr images import /vagrant/build/libreria-api.tar >/dev/null
# kubeconfig para controlar el cluster desde la VM de JMeter
sed 's/127.0.0.1/192.168.56.11/' /etc/rancher/k3s/k3s.yaml > /vagrant/build/kubeconfig

kubectl apply -f /vagrant/k8s/
echo "== Esperando la base (carga los 10.000 libros al arrancar) =="
kubectl -n libreria rollout status deployment/db --timeout=300s
kubectl -n libreria rollout status deployment/api --timeout=300s

kubectl get nodes -o wide
kubectl -n libreria get pods -o wide
curl -s http://localhost:30080/api/libros/1; echo
echo "Listo. Siguiente: vagrant ssh jmeter -c \"bash /vagrant/pruebas/serie-k8s.sh k8s1n\""
