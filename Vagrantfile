# -*- mode: ruby -*-
# vi: set ft=ruby :
#
# 4 maquinas virtuales en el mismo PC (VirtualBox):
#   vm1    -> Docker Compose (fase 1) y servidor de Kubernetes (k3s). Siempre aloja la base.
#   vm2    -> segundo nodo de Kubernetes (se crea en la fase de 2 nodos)
#   vm3    -> tercer nodo de Kubernetes (se crea en la fase de 3 nodos)
#   jmeter -> genera la carga
#
# Orden de uso (ver GUIA.md):
#   vagrant up vm1 jmeter
#   vagrant up vm2          (cuando vm1 ya esta en modo Kubernetes; se une sola al cluster)
#   vagrant up vm3
#
# La carpeta del proyecto queda dentro de cada VM en /vagrant.

# Recursos pensados para un PC de 4 nucleos / 8 hilos y 16 GB de RAM.
# Con las 4 VM encendidas: 8 CPU virtuales y 12 GB de RAM.
NODOS = {
  "vm1" => { ip: "192.168.56.11", cpus: 2, ram: 4096 },
  "vm2" => { ip: "192.168.56.12", cpus: 2, ram: 3072 },
  "vm3" => { ip: "192.168.56.13", cpus: 2, ram: 3072 },
}
JMETER = { ip: "192.168.56.20", cpus: 2, ram: 2048 }

K3S_TOKEN = "proyecto-final-libreria"
JMETER_VERSION = "5.6.3"

HOSTS = NODOS.map { |n, c| "#{c[:ip]} #{n}" }.push("#{JMETER[:ip]} jmeter").join("\n")

# ---- Comun a todas: nombres de las maquinas y nada corriendo en segundo plano ----
BASE = <<~SHELL
  set -eux
  sed -i '/192\\.168\\.56\\./d' /etc/hosts
  cat >> /etc/hosts <<'EOF'
  #{HOSTS}
  EOF
  # Que las actualizaciones automaticas no consuman CPU durante las pruebas
  systemctl disable --now unattended-upgrades apt-daily.timer apt-daily-upgrade.timer || true
  mkdir -p /vagrant/build
SHELL

# Interfaz de la red privada 192.168.56.x (la que usa Kubernetes entre nodos)
IFACE = %q(IFACE=$(ip -o -4 addr show | awk -v ip="$IP" '$4 ~ "^"ip"/" {print $2}'))

# ---- vm1: Docker + servidor k3s (instalado, pero apagado hasta la fase 2) ----
SERVIDOR = <<~SHELL
  set -eux
  IP="$1"
  #{IFACE}
  if ! command -v docker >/dev/null; then
    curl -fsSL https://get.docker.com | sh
  fi
  usermod -aG docker vagrant
  systemctl disable --now docker.service docker.socket   # lo enciende scripts/modo-compose.sh

  if ! command -v k3s >/dev/null; then
    curl -sfL https://get.k3s.io | INSTALL_K3S_CHANNEL=stable INSTALL_K3S_SKIP_START=true \
      INSTALL_K3S_SKIP_ENABLE=true K3S_TOKEN=#{K3S_TOKEN} sh -s - server \
      --node-ip "$IP" --advertise-address "$IP" --tls-san "$IP" --flannel-iface "$IFACE" \
      --disable traefik --disable servicelb --write-kubeconfig-mode 644
  fi
  # Los nodos y la VM de JMeter usan exactamente la misma version
  k3s --version | awk 'NR==1 {print $3}' > /vagrant/build/k3s-version
SHELL

# ---- vm2 y vm3: agente k3s que se une solo al servidor de vm1 ----
AGENTE = <<~SHELL
  set -eux
  IP="$1"
  #{IFACE}
  if ! curl -sk --max-time 5 https://192.168.56.11:6443 >/dev/null; then
    echo "vm1 no esta en modo Kubernetes. Ejecutar primero: vagrant ssh vm1 -c 'bash /vagrant/scripts/modo-k8s.sh'"
    exit 1
  fi
  if ! command -v k3s >/dev/null; then
    curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION="$(cat /vagrant/build/k3s-version)" \
      K3S_URL=https://192.168.56.11:6443 K3S_TOKEN=#{K3S_TOKEN} sh -s - agent \
      --node-ip "$IP" --flannel-iface "$IFACE"
  fi
  # Importa la imagen de la API (construida en vm1), asi no se necesita Docker Hub
  for i in $(seq 1 60); do k3s ctr version >/dev/null 2>&1 && break; sleep 2; done
  k3s ctr images import /vagrant/build/libreria-api.tar
SHELL

# ---- jmeter: Java y JMeter ----
JMETER_SH = <<~SHELL
  set -eux
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -q
  apt-get install -y -q openjdk-17-jre-headless curl python3
  if [ ! -d /opt/apache-jmeter-#{JMETER_VERSION} ]; then
    curl -fsSL -o /tmp/jmeter.tgz https://archive.apache.org/dist/jmeter/binaries/apache-jmeter-#{JMETER_VERSION}.tgz
    tar xzf /tmp/jmeter.tgz -C /opt
    rm /tmp/jmeter.tgz
  fi
  ln -sf /opt/apache-jmeter-#{JMETER_VERSION}/bin/jmeter /usr/local/bin/jmeter
  # Mas puertos locales para abrir muchas conexiones seguidas
  printf 'net.ipv4.ip_local_port_range = 10240 65000\\nnet.ipv4.tcp_tw_reuse = 1\\n' > /etc/sysctl.d/90-jmeter.conf
  sysctl --system >/dev/null
SHELL

Vagrant.configure("2") do |config|
  config.vm.box = "bento/ubuntu-24.04"
  config.vm.box_check_update = false
  config.vm.synced_folder ".", "/vagrant"

  config.vm.provider "virtualbox" do |vb|
    vb.linked_clone = true     # crea las VM copiando solo las diferencias: mas rapido y menos disco
    vb.customize ["modifyvm", :id, "--nictype1", "virtio", "--nictype2", "virtio", "--audio-driver", "none"]
  end

  NODOS.each do |nombre, c|
    config.vm.define nombre, autostart: false do |nodo|
      nodo.vm.hostname = nombre
      nodo.vm.network "private_network", ip: c[:ip]
      nodo.vm.provider "virtualbox" do |vb|
        vb.name   = "libreria-#{nombre}"
        vb.cpus   = c[:cpus]
        vb.memory = c[:ram]
      end
      nodo.vm.provision "base", type: "shell", inline: BASE
      if nombre == "vm1"
        nodo.vm.provision "servidor", type: "shell", inline: SERVIDOR, args: [c[:ip]]
      else
        nodo.vm.provision "agente", type: "shell", inline: AGENTE, args: [c[:ip]]
      end
    end
  end

  config.vm.define "jmeter", autostart: false do |jm|
    jm.vm.hostname = "jmeter"
    jm.vm.network "private_network", ip: JMETER[:ip]
    jm.vm.provider "virtualbox" do |vb|
      vb.name   = "libreria-jmeter"
      vb.cpus   = JMETER[:cpus]
      vb.memory = JMETER[:ram]
    end
    jm.vm.provision "base", type: "shell", inline: BASE
    jm.vm.provision "jmeter", type: "shell", inline: JMETER_SH
  end
end
