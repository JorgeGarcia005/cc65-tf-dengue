#!/usr/bin/env bash
# Verifica spin/barrera.pml (issue #31): el modelo correcto no tiene errores y
# cada mutante da un contraejemplo. Sale con 1 si algo no es lo esperado.
#
#   cd spin && bash verificar.sh
#
# Requiere spin y gcc (Debian/Ubuntu: sudo apt-get install -y spin gcc).
set -euo pipefail
cd "$(dirname "$0")"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
cp barrera.pml "$tmp/"
cd "$tmp"

estado=0
printf '%-44s %-10s %14s %8s %s\n' "corrida" "chequeo" "estados" "errores" "esperado"

# correr <nombre> <defines de spin> <opciones de pan> <esperado: 0 | >0>
correr() {
	local nombre=$1 defs=$2 extra=$3 esperado=$4
	spin -a $defs barrera.pml >/dev/null
	for chequeo in seguridad termina; do
		if [ "$chequeo" = seguridad ]; then
			gcc -O2 -w -DNOCLAIM -o pan-seg pan.c
			salida=$(./pan-seg $extra 2>&1)
		else
			gcc -O2 -w -o pan pan.c
			salida=$(./pan -a $extra -N termina 2>&1)
		fi
		errores=$(sed -n 's/.*errors: \([0-9]*\).*/\1/p' <<<"$salida" | head -1)
		estados=$(sed -n 's/^ *\([0-9.e+]*\) states, stored.*/\1/p' <<<"$salida" | head -1)
		printf '%-44s %-10s %14s %8s %s\n' "$nombre" "$chequeo" "$estados" "$errores" "$esperado"
		if [ "$esperado" = 0 ] && [ "$errores" != 0 ]; then estado=1; fi
		if [ "$esperado" = ">0" ] && [ "$errores" != 0 ]; then return 0; fi
		rm -f ./*.trail
	done
	if [ "$esperado" = ">0" ]; then
		echo "  ERROR: el mutante no dio contraejemplo"
		estado=1
	fi
}

correr "modelo (canal de capacidad N)"            ""                       ""   0
correr "modelo, canal de capacidad 1, sin asserts" "-DCAP=1"                "-A" 0
correr "mutante: envío bloqueante (capacidad 1)"   "-DMUT_BLOQUEANTE -DCAP=1" ""  ">0"
correr "mutante: reasigna solo el shard vencido"   "-DMUT_UNA_PASADA"       ""   ">0"
correr "mutante: plazo global"                     "-DMUT_PLAZO_GLOBAL"     ""   ">0"
echo "spin: $(spin -V)"
if [ $estado -eq 0 ]; then echo "RESULTADO: todo como se esperaba"; else echo "RESULTADO: hay diferencias"; fi
exit $estado
