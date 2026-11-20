#!/usr/bin/env bash
# Verifica spin/barrera.pml (issue #31): el modelo no tiene errores y cada
# mutante da un contraejemplo. Falla (código 1) si algo no es lo esperado o si
# una búsqueda quedó incompleta. Escribe el resultado en reports/spin.json, del
# que cmd/informe toma la tabla del informe (AGENTS.md §3.6).
#
#   cd spin && bash verificar.sh
#
# Requiere spin y gcc (Debian/Ubuntu: sudo apt-get install -y spin gcc).
set -euo pipefail
cd "$(dirname "$0")"
raiz=$(cd .. && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
cp barrera.pml "$tmp/"
cd "$tmp"

estado=0
filas=()
printf '%-48s %-10s %12s %7s %s\n' "corrida" "chequeo" "estados" "errores" "esperado"

# correr <nombre> <defines de spin> <esperado: 0 | >0>
correr() {
	local nombre=$1 defs=$2 esperado=$3 hubo=0
	spin -a $defs barrera.pml >/dev/null
	for chequeo in seguridad termina; do
		local salida errores estados ok=true
		if [ "$chequeo" = seguridad ]; then
			gcc -O2 -w -DNOCLAIM -o pan-seg pan.c
			salida=$(./pan-seg 2>&1 || true)
		else
			gcc -O2 -w -o pan pan.c
			salida=$(./pan -a -N termina 2>&1 || true)
		fi
		errores=$(sed -n 's/.*errors: \([0-9][0-9]*\).*/\1/p' <<<"$salida" | head -1)
		estados=$(sed -n 's/^ *\([0-9.e+]*\) states, stored.*/\1/p' <<<"$salida" | head -1)
		if ! [[ "$errores" =~ ^[0-9]+$ ]]; then
			echo "  ERROR: no se pudo leer el número de errores de pan ($nombre, $chequeo)"
			estado=1; ok=false; errores=-1
		fi
		# Una búsqueda truncada no prueba nada, aunque diga errors: 0.
		if [ "$esperado" = 0 ] && grep -Eq 'Search not completed|max search depth too small|out of memory' <<<"$salida"; then
			echo "  ERROR: búsqueda incompleta ($nombre, $chequeo)"
			estado=1; ok=false
		fi
		if [ "$esperado" = 0 ] && [ "$errores" != 0 ]; then estado=1; ok=false; fi
		printf '%-48s %-10s %12s %7s %s\n' "$nombre" "$chequeo" "$estados" "$errores" "$esperado"
		filas+=("{\"corrida\":\"$nombre\",\"defines\":\"$defs\",\"chequeo\":\"$chequeo\",\"estados\":${estados:-0},\"errores\":$errores,\"esperado\":\"$esperado\",\"ok\":$ok}")
		rm -f ./*.trail
		if [ "$esperado" = ">0" ] && [ "$errores" -gt 0 ]; then hubo=1; break; fi
	done
	if [ "$esperado" = ">0" ] && [ $hubo -eq 0 ]; then
		echo "  ERROR: el mutante no dio contraejemplo"
		estado=1
	fi
}

correr "modelo N=3 (canal de capacidad N)"            ""                             0
correr "modelo N=3, canal de capacidad 1"              "-DCAP=1"                      0
correr "modelo N=4, 1 época, sin sospechas"            "-DN=4 -DEPOCAS=1 -DSOSPECHAS=0" 0
correr "mutante: envío bloqueante (capacidad 1)"       "-DMUT_BLOQUEANTE -DCAP=1"     ">0"
correr "mutante: reasigna solo el shard vencido"       "-DMUT_UNA_PASADA"             ">0"
correr "mutante: plazo global"                         "-DMUT_PLAZO_GLOBAL"           ">0"
correr "mutante: primer nodo sano (N=4)"               "-DMUT_PRIMER_SANO -DN=4 -DEPOCAS=1 -DSOSPECHAS=0" ">0"
version=$(spin -V)
echo "spin: $version"

mkdir -p "$raiz/reports"
{
	printf '{\n  "generado": "%s",\n  "spin": "%s",\n  "gcc": "%s",\n  "ok": %s,\n  "corridas": [\n    ' \
		"$(date -Iseconds)" "$version" "$(gcc -dumpfullversion)" "$([ $estado -eq 0 ] && echo true || echo false)"
	(IFS=$'\n'; echo "${filas[*]}") | paste -sd ',' | sed 's/},{/},\n    {/g'
	printf '  ]\n}\n'
} > "$raiz/reports/spin.json"
echo "escrito reports/spin.json"

if [ $estado -eq 0 ]; then echo "RESULTADO: todo como se esperaba"; else echo "RESULTADO: hay diferencias"; fi
exit $estado
