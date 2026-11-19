"""Tabla resumen de todas las pruebas de resultados/ (solo libreria estandar).

    python3 /vagrant/pruebas/resumir.py

Descarta los segundos de subida de usuarios y promedia las repeticiones.
"""
import csv
import glob
import os
import re
import statistics
from collections import defaultdict

RAMPUP_MS = 10_000
patron = re.compile(r"(compose|k8s\dn)_(\d+)rep_(\d+)u_r(\d+)\.csv$")
grupos = defaultdict(list)

for ruta in sorted(glob.glob(os.path.join(os.path.dirname(__file__), "..", "resultados", "*.csv"))):
    m = patron.search(ruta)
    if not m:
        continue
    with open(ruta, newline="") as f:
        filas = list(csv.DictReader(f))
    if not filas:
        continue
    inicio = min(int(r["timeStamp"]) for r in filas)
    filas = [r for r in filas if int(r["timeStamp"]) >= inicio + RAMPUP_MS]
    fin = max(int(r["timeStamp"]) + int(r["elapsed"]) for r in filas)
    segundos = (fin - inicio - RAMPUP_MS) / 1000
    tiempos = sorted(int(r["elapsed"]) for r in filas)
    errores = sum(r["success"].lower() != "true" for r in filas)
    grupos[(m[1], int(m[2]), int(m[3]))].append((
        len(filas) / segundos,
        statistics.mean(tiempos),
        tiempos[int(len(tiempos) * 0.95) - 1],
        100 * errores / len(filas),
    ))

print(f"{'escenario':<9} {'rep':>3} {'usuarios':>8} {'pet/s':>8} {'medio ms':>9} {'p95 ms':>8} {'err %':>6}")
with open(os.path.join(os.path.dirname(__file__), "..", "resultados", "resumen.csv"), "w", newline="") as f:
    w = csv.writer(f)
    w.writerow(["escenario", "replicas", "usuarios", "throughput", "t_medio_ms", "p95_ms", "errores_pct"])
    for (esc, rep, u), valores in sorted(grupos.items()):
        prom = [statistics.mean(v[i] for v in valores) for i in range(4)]
        print(f"{esc:<9} {rep:>3} {u:>8} {prom[0]:>8.1f} {prom[1]:>9.1f} {prom[2]:>8.0f} {prom[3]:>6.2f}")
        w.writerow([esc, rep, u] + [round(x, 2) for x in prom])
print("\nGuardado en resultados/resumen.csv")
