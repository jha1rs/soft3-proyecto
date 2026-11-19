#!/usr/bin/env bash
# Regenera k8s/01-db-init.yaml a partir de db/init.sql (solo si se cambia la carga de datos)
set -euo pipefail
cd "$(dirname "$0")/.."
{
  echo "# Copia de db/init.sql (generada con scripts/generar-configmap.sh). Postgres lo ejecuta al arrancar vacio."
  echo "apiVersion: v1"
  echo "kind: ConfigMap"
  echo "metadata:"
  echo "  name: db-init"
  echo "  namespace: libreria"
  echo "data:"
  echo "  init.sql: |"
  sed 's/^/    /; s/^    $//' db/init.sql
} > k8s/01-db-init.yaml
echo "k8s/01-db-init.yaml actualizado"
