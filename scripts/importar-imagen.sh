#!/usr/bin/env bash
# Solo si se cambia el codigo de la API y se reconstruye la imagen (modo-compose.sh).
# Importa build/libreria-api.tar en el nodo donde se ejecute:
#   vagrant ssh vm1 -c "bash /vagrant/scripts/importar-imagen.sh"   (y lo mismo en vm2, vm3)
set -euo pipefail
sudo k3s ctr images import /vagrant/build/libreria-api.tar
