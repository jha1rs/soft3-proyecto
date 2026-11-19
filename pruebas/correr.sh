#!/usr/bin/env bash
# Corre todas las cargas de UNA configuracion. Se ejecuta en la VM de JMeter.
#   bash /vagrant/pruebas/correr.sh <escenario> <replicas>
#   bash /vagrant/pruebas/correr.sh compose 1
#   bash /vagrant/pruebas/correr.sh k8s2n 3
# Escenarios: compose | k8s1n | k8s2n | k8s3n
#
# Duracion: unos 5 minutos por configuracion. Para mas rigor:  REPETICIONES=2 bash ...
set -euo pipefail

ESC=$1; REP=$2
HOST=${HOST:-vm1}
if [ "$ESC" = "compose" ]; then PORT=8080; else PORT=30080; fi

CARGAS=${CARGAS:-"10 25 50 100"}   # usuarios concurrentes
REPETICIONES=${REPETICIONES:-1}
RAMPUP=${RAMPUP:-10}               # segundos de subida (se descartan en el analisis)
DURACION=${DURACION:-60}           # segundos totales de cada prueba
PAUSA=${PAUSA:-10}
export JVM_ARGS=${JVM_ARGS:-"-Xms512m -Xmx768m"}

cd /vagrant
PLAN=jmeter/plan.jmx
mkdir -p resultados/logs

if ! curl -sf --max-time 5 "http://$HOST:$PORT/listo" >/dev/null; then
  echo "La API no responde en http://$HOST:$PORT/listo"
  exit 1
fi

echo "== $ESC con $REP replica(s): calentamiento de 20 s =="
jmeter -n -t "$PLAN" -JHOST="$HOST" -JPORT="$PORT" -JUSERS=10 -JRAMPUP=2 -JDURATION=20 \
  -l /tmp/calentamiento.csv -j /tmp/calentamiento.log > /dev/null
rm -f /tmp/calentamiento.csv
sleep 5

for R in $(seq 1 "$REPETICIONES"); do
  for U in $CARGAS; do
    NOMBRE="${ESC}_${REP}rep_${U}u_r${R}"
    rm -f "resultados/$NOMBRE.csv"
    jmeter -n -t "$PLAN" -JHOST="$HOST" -JPORT="$PORT" \
      -JUSERS="$U" -JRAMPUP="$RAMPUP" -JDURATION="$DURACION" \
      -l "resultados/$NOMBRE.csv" -j "resultados/logs/$NOMBRE.jmeter.log" > "resultados/logs/$NOMBRE.log"
    echo "$(date +%H:%M:%S) $NOMBRE  $(grep 'summary =' "resultados/logs/$NOMBRE.log" | tail -1 | sed 's/summary = *//')"
    sleep "$PAUSA"
  done
done
