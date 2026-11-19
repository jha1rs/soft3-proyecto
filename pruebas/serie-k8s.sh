#!/usr/bin/env bash
# Corre 1, 2 y 3 replicas en Kubernetes, una tras otra. Se ejecuta en la VM de JMeter.
#   bash /vagrant/pruebas/serie-k8s.sh k8s1n        (con 1 nodo)
#   bash /vagrant/pruebas/serie-k8s.sh k8s2n        (despues de: vagrant up vm2)
#   bash /vagrant/pruebas/serie-k8s.sh k8s3n "3"    (solo 3 replicas)
# Para cada configuracion guarda donde quedo cada pod y el consumo de CPU y memoria.
set -euo pipefail

ESC=$1
REPLICAS=${2:-"1 2 3"}
export KUBECONFIG=/vagrant/build/kubeconfig
cd /vagrant

# kubectl de la misma version que el cluster (se instala la primera vez)
if ! command -v kubectl >/dev/null; then
  VER=$(cut -d+ -f1 build/k3s-version)
  curl -fsSL -o /tmp/kubectl "https://dl.k8s.io/release/$VER/bin/linux/amd64/kubectl"
  sudo install -m 755 /tmp/kubectl /usr/local/bin/kubectl
fi

NODOS=$(kubectl get nodes --no-headers | awk '$2 == "Ready"' | wc -l)
echo "== $ESC: $NODOS nodo(s) listos =="
kubectl get nodes
mkdir -p resultados/ubicacion resultados/monitoreo

for N in $REPLICAS; do
  echo "== Escalando a $N replica(s) =="
  # Se baja a 0 y se sube a N para que las replicas se repartan desde cero
  kubectl -n libreria scale deployment/api --replicas=0
  kubectl -n libreria wait --for=delete pod -l app=api --timeout=60s || true
  kubectl -n libreria scale deployment/api --replicas="$N"
  kubectl -n libreria rollout status deployment/api --timeout=180s
  sleep 5
  kubectl -n libreria get pods -o wide | tee "resultados/ubicacion/${ESC}_${N}rep.txt"

  # Consumo de CPU y memoria cada 10 s mientras dura la prueba
  ARCH="resultados/monitoreo/${ESC}_${N}rep.txt"
  : > "$ARCH"
  ( while true; do
      { echo "=== $(date +%H:%M:%S)"; kubectl top nodes; kubectl -n libreria top pods; } >> "$ARCH" 2>&1
      sleep 10
    done ) &
  MONITOR=$!

  bash pruebas/correr.sh "$ESC" "$N" || true
  kill "$MONITOR" 2>/dev/null || true
done
echo "== Serie $ESC terminada =="
