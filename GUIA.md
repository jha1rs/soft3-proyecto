# Guía de ejecución

Todo corre en un PC con Windows, VirtualBox y Vagrant. Los comandos se escriben en **PowerShell**, dentro de la carpeta del proyecto (`D:\proyecto\soft3-proyecto`).

Tiempo aproximado: 30 minutos para crear las máquinas la primera vez y 1 hora de pruebas.

## Qué hay en cada máquina

| VM | CPU | RAM | Papel |
|---|---|---|---|
| vm1 | 2 | 3 GB | Docker Compose (fase 1) y servidor de Kubernetes. Siempre aloja PostgreSQL |
| vm2 | 2 | 1,5 GB | Segundo nodo de Kubernetes |
| vm3 | 2 | 1,5 GB | Tercer nodo de Kubernetes |
| jmeter | 2 | 1,5 GB | Genera la carga. Desde aquí se controla todo |

Con las cuatro encendidas usan 8 CPU virtuales y 7,5 GB de RAM, pensado para que Windows conserve sus ~5,5 GB y quede margen. Durante las pruebas hay que cerrar todo lo demás en el PC. Si se cambian los valores en el `Vagrantfile`, aplicar con `vagrant reload <vm>`.

## Requisitos (una sola vez)

Programas en `D:\soft3-proyecto`, máquinas virtuales en `D:\VirtualBox VMs` e imágenes de Vagrant en `D:\vagrant.d`. Todo en PowerShell como administrador.

1. Virtualización activada (Administrador de tareas → Rendimiento → CPU → *Virtualización: Habilitado*).
2. Apagar Hyper-V y reiniciar. En *Seguridad de Windows → Seguridad del dispositivo → Aislamiento del núcleo*, desactivar *Integridad de memoria*.

   ```powershell
   bcdedit /set hypervisorlaunchtype off
   ```

3. Carpetas e instalación:

   ```powershell
   New-Item -ItemType Directory -Force -Path "D:\soft3-proyecto", "D:\VirtualBox VMs", "D:\vagrant.d", "D:\proyecto"
   winget install -e --id Microsoft.VCRedist.2015+.x64
   winget install -e --id Oracle.VirtualBox --location "D:\soft3-proyecto\VirtualBox"
   winget install -e --id Hashicorp.Vagrant --location "D:\soft3-proyecto\Vagrant"
   winget install -e --id Git.Git --location "D:\soft3-proyecto\Git"
   ```

   Comprobar que quedaron en D (si alguno da `False`, instalarlo a mano eligiendo esa carpeta) y reiniciar:

   ```powershell
   Test-Path "D:\soft3-proyecto\VirtualBox\VBoxManage.exe"
   Test-Path "D:\soft3-proyecto\Vagrant\bin\vagrant.exe"
   Test-Path "D:\soft3-proyecto\Git\bin\git.exe"
   ```

4. VirtualBox en el PATH y datos en D:

   ```powershell
   $p = [Environment]::GetEnvironmentVariable("Path", "User")
   [Environment]::SetEnvironmentVariable("Path", "$p;D:\soft3-proyecto\VirtualBox", "User")
   [Environment]::SetEnvironmentVariable("VAGRANT_HOME", "D:\vagrant.d", "User")
   ```

   Cerrar y abrir PowerShell de nuevo:

   ```powershell
   VBoxManage setproperty machinefolder "D:\VirtualBox VMs"
   VBoxManage --version      # 7.2.x
   vagrant --version         # 2.4.9 o mas reciente
   ```

5. Proyecto e imagen de Ubuntu:

   ```powershell
   cd D:\proyecto
   git clone https://github.com/jha1rs/soft3-proyecto.git
   cd soft3-proyecto
   vagrant box add bento/ubuntu-24.04 --provider virtualbox
   ```

## Paso 1. Crear vm1 y la VM de JMeter

```powershell
vagrant up vm1 jmeter
```

La primera vez descarga Ubuntu (1,5 GB), Docker, k3s y JMeter. Tarda entre 15 y 25 minutos.

## Paso 2. Fase 1: Docker Compose

```powershell
vagrant ssh vm1 -c "bash /vagrant/scripts/modo-compose.sh"
```

Construye la imagen de la API (también la guarda en `build/libreria-api.tar` para Kubernetes), levanta la base y la API, y muestra un libro y `Libros en la base: 10000`.

```powershell
vagrant ssh jmeter -c "bash /vagrant/pruebas/correr.sh compose 1"
```

Unos 5 minutos. Cada línea muestra una carga terminada con su throughput (`/s`), tiempo medio (`Avg`) y errores (`Err`).

## Paso 3. Fase 2: Kubernetes con 1 nodo

```powershell
vagrant ssh vm1 -c "bash /vagrant/scripts/modo-k8s.sh"
vagrant ssh jmeter -c "bash /vagrant/pruebas/serie-k8s.sh k8s1n"
```

`modo-k8s.sh` apaga Docker, enciende k3s y despliega la base y la API. `serie-k8s.sh` corre 1, 2 y 3 réplicas una tras otra (unos 17 minutos) y guarda dónde quedó cada pod y el consumo de CPU y memoria.

## Paso 4. Fase 3: Kubernetes con 2 nodos

```powershell
vagrant up vm2
vagrant ssh jmeter -c "bash /vagrant/pruebas/serie-k8s.sh k8s2n"
```

vm2 se une sola al clúster y carga la imagen de la API. La serie comprueba al inicio cuántos nodos hay listos.

## Paso 5. Fase 4: Kubernetes con 3 nodos

```powershell
vagrant up vm3
vagrant ssh jmeter -c "bash /vagrant/pruebas/serie-k8s.sh k8s3n"
```

## Paso 6. Resumen y notebook

```powershell
vagrant ssh jmeter -c "python3 /vagrant/pruebas/resumir.py"
```

Muestra la tabla de todas las configuraciones y la guarda en `resultados/resumen.csv`.

Luego subir el proyecto a GitHub (con los resultados), abrir `notebook/analisis.ipynb` en Google Colab, poner el enlace del repositorio en `REPO_URL` y ejecutar todo.

```powershell
git add .
git commit -m "Resultados de las pruebas"
git push
```

## Paso 7. Apagar

```powershell
vagrant halt
```

Para borrar las VM al terminar el proyecto: `vagrant destroy -f`.

## Opciones de las pruebas

Se cambian con variables antes del comando, dentro de la VM de JMeter:

```powershell
vagrant ssh jmeter -c "REPETICIONES=2 bash /vagrant/pruebas/serie-k8s.sh k8s1n"
vagrant ssh jmeter -c "CARGAS='10 50 100 200' DURACION=90 bash /vagrant/pruebas/correr.sh compose 1"
```

| Variable | Por defecto | Qué es |
|---|---|---|
| `CARGAS` | `10 25 50 100` | Usuarios concurrentes |
| `DURACION` | 60 | Segundos de cada prueba |
| `RAMPUP` | 10 | Segundos de subida (se descartan) |
| `REPETICIONES` | 1 | Veces que se repite cada carga |
| `PAUSA` | 10 | Segundos entre pruebas |

Si se usa la misma opción en todas las configuraciones, los resultados siguen siendo comparables.

## Probar la API a mano

Desde Windows, con vm1 encendida:

```powershell
curl.exe http://192.168.56.11:8080/api/libros/1                 # Docker Compose
curl.exe http://192.168.56.11:30080/api/libros/1                # Kubernetes
curl.exe "http://192.168.56.11:30080/api/libros?pagina=2&tamano=5"
curl.exe http://192.168.56.11:30080/api/categorias/estadisticas
curl.exe -X POST -H "Content-Type: application/json" -d '{\"libro_id\": 5, \"cantidad\": 3}' http://192.168.56.11:30080/api/pedidos
```

## Si algo falla

| Síntoma | Qué hacer |
|---|---|
| `vagrant up` dice que VT-x no está disponible | Activar la virtualización en la BIOS y apagar Hyper-V |
| Las VM van muy lentas (tortuga verde en VirtualBox) | Hyper-V sigue activo: `bcdedit /set hypervisorlaunchtype off`, desactivar Integridad de memoria y reiniciar |
| `vagrant up vm2` dice que vm1 no está en modo Kubernetes | Ejecutar antes el paso 3 (`modo-k8s.sh`) |
| `correr.sh` dice que la API no responde | Revisar que el modo de vm1 coincida con la prueba: `compose` usa el puerto 8080 y `k8s*` el 30080 |
| Pods de la API en `ErrImageNeverPull` | La imagen no está en ese nodo: `vagrant ssh vm2 -c "bash /vagrant/scripts/importar-imagen.sh"` |
| `kubectl top` dice `metrics not available` | Esperar un minuto después de encender k3s |
| Se reinicia vm1 o el PC | `vagrant up vm1` y volver a ejecutar el `modo-*.sh` que corresponda. vm2 y vm3 se vuelven a unir solas |
| Se cambia el código de la API | `modo-compose.sh` la reconstruye; luego `modo-k8s.sh` e `importar-imagen.sh` en vm2 y vm3 |
