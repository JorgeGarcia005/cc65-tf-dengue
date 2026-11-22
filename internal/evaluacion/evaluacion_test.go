package evaluacion

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml"
)

var deptosPrueba = []string{"LORETO", "MADRE DE DIOS", "PIURA", "TUMBES", "UCAYALI"}

// sintetico arma un conjunto con el esquema real de features y etiquetas que
// dependen del departamento, la edad y la semana.
func sintetico(t *testing.T, n int, semilla uint64) (*ml.Dataset, *features.Esquema, []ingest.Registro) {
	t.Helper()
	r := rand.New(rand.NewPCG(semilla, 7))
	regs := make([]ingest.Registro, n)
	for i := range regs {
		regs[i] = ingest.Registro{
			Departamento: deptosPrueba[r.IntN(len(deptosPrueba))], Ano: 2015 + r.IntN(10),
			Semana: 1 + r.IntN(52), EdadAnios: float64(r.IntN(90)), Sexo: "FM"[r.IntN(2)],
		}
		if r.IntN(50) == 0 {
			regs[i].Departamento = "TACNA" // cae en OTROS
		}
		riesgo := 0.08 + 0.10*float64(regs[i].Semana%13)/13 + 0.05*float64(regs[i].EdadAnios)/90
		if regs[i].Departamento == "LORETO" {
			riesgo += 0.1
		}
		if r.Float64() < riesgo {
			regs[i].Clase = 1
		}
	}
	esq, err := features.Ajustar(regs, 2011, 100)
	if err != nil {
		t.Fatal(err)
	}
	x, y := make([]float64, n*esq.D()), make([]float64, n)
	for i, rg := range regs {
		esq.Llenar(rg, x[i*esq.D():(i+1)*esq.D()])
		y[i] = rg.Objetivo()
	}
	ds, err := ml.NuevoDataset(x, y, esq.D())
	if err != nil {
		t.Fatal(err)
	}
	return ds, esq, regs
}

// pesosDe entrena unas pocas épocas para tener puntajes realistas.
func pesosDe(ds *ml.Dataset) []float64 {
	pos, neg := ml.PesosBalanceados(ds.Y)
	s := ml.NuevoSecuencial(ds, ml.Config{TamBloque: 256, Lambda: 1e-4, PesoPos: pos, PesoNeg: neg})
	return ml.Entrenar(s, ds.D, ml.Entrenamiento{Epocas: 30, Tasa: 1}).W
}

func entradaPrueba(t *testing.T, n int) Entrada {
	t.Helper()
	ds, esq, _ := sintetico(t, n, 1)
	cluster, err := os.ReadFile("../../reports/bench_cluster_deez_gold.json")
	if err != nil {
		t.Fatal(err)
	}
	api, err := os.ReadFile("../../reports/bench_api_deez.json")
	if err != nil {
		t.Fatal(err)
	}
	pos, neg := ml.PesosBalanceados(ds.Y)
	return Entrada{Test: ds, Pesos: pesosDe(ds), Esquema: esq, Umbral: 0.5, AjustePrior: math.Log(pos / neg),
		VersionModelo: "prueba", BenchCluster: cluster, BenchAPI: api, MinCasos: 5, TopN: 10,
		PuntosROC: 50, Deciles: 10}
}

func TestROCAUCIgualMannWhitney(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 3))
	y, p := make([]float64, 5000), make([]float64, 5000)
	for i := range y {
		p[i] = math.Round(r.Float64()*40) / 40 // muchos empates a propósito
		if r.Float64() < p[i] {
			y[i] = 1
		}
	}
	curva, auc, err := ROC(y, p, 0, 30)
	if err != nil {
		t.Fatal(err)
	}
	mw, _ := ml.AUC(y, p)
	if math.Abs(auc-mw) > 1e-12 {
		t.Fatalf("AUC trapecio %.15f ≠ Mann-Whitney %.15f", auc, mw)
	}
	if len(curva) > 30 || curva[0] != (PuntoROC{0, 0, 1}) {
		t.Fatalf("curva mal reducida: %d puntos, primero %+v", len(curva), curva[0])
	}
	if u := curva[len(curva)-1]; u.FPR != 1 || u.TPR != 1 {
		t.Fatalf("la curva no termina en (1,1): %+v", u)
	}
	for k := 1; k < len(curva); k++ {
		if curva[k].FPR < curva[k-1].FPR || curva[k].TPR < curva[k-1].TPR {
			t.Fatalf("curva no monótona en %d", k)
		}
	}
	if _, _, err := ROC([]float64{1, 1}, []float64{0.2, 0.3}, 0, 10); err == nil {
		t.Fatal("una sola clase debería fallar")
	}
}

func TestCalibracionDeciles(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 5))
	n := 20003
	y, pc := make([]float64, n), make([]float64, n)
	for i := range y {
		pc[i] = r.Float64()
		if r.Float64() < pc[i] {
			y[i] = 1
		}
	}
	bins, err := Calibracion(y, pc, 10)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for k, b := range bins {
		total += b.Casos
		if k > 0 && b.Predicho < bins[k-1].Predicho {
			t.Fatal("los tramos no están ordenados por riesgo")
		}
		// Modelo perfectamente calibrado (~2000 casos por tramo): observado ≈ predicho.
		if math.Abs(b.Predicho-b.Observado) > 0.04 || b.IC95Inf > b.Observado || b.IC95Sup < b.Observado {
			t.Fatalf("tramo %d: predicho %.3f, observado %.3f, IC [%.3f, %.3f]", k, b.Predicho, b.Observado, b.IC95Inf, b.IC95Sup)
		}
	}
	if total != n {
		t.Fatalf("los tramos suman %d filas, quería %d", total, n)
	}
	if inf, sup := Wilson(0, 10); inf != 0 || sup <= 0 {
		t.Fatalf("Wilson(0,10) = [%v, %v]", inf, sup)
	}
}

func TestCruzadoConteosYNiveles(t *testing.T) {
	ds, esq, regs := sintetico(t, 3000, 2)
	pc := make([]float64, ds.N)
	for i := range pc {
		pc[i] = 0.1
	}
	cubo, err := Cruzar(ds, ds.Y, pc, esq, 30)
	if err != nil {
		t.Fatal(err)
	}
	// Conteo directo desde los registros para una celda concreta.
	quiero, pos := 0, 0
	for _, r := range regs {
		if r.Departamento == "LORETO" && r.Semana == 10 && ingest.TramoEdad(r.EdadAnios) == 0 {
			quiero++
			pos += r.Clase
		}
	}
	celda := buscar(t, cubo, "LORETO", 10, ingest.TramosEdad[0].Nombre)
	if celda.Casos != quiero || celda.Positivos != pos || celda.Insuficiente != (quiero < 30) {
		t.Fatalf("LORETO/10/niño = %+v, quería %d casos y %d positivos", celda, quiero, pos)
	}
	if n := buscar(t, cubo, TodosDepartamentos, TodasSemanas, TodasEtapas).Casos; n != ds.N {
		t.Fatalf("total nacional = %d, quería %d", n, ds.N)
	}
	// Cada nivel «todos» es la suma de sus partes.
	suma := 0
	for _, c := range cubo.Celdas {
		if c.Departamento == "PIURA" && c.Semana != TodasSemanas && c.Etapa == TodasEtapas {
			suma += c.Casos
		}
	}
	if total := buscar(t, cubo, "PIURA", TodasSemanas, TodasEtapas).Casos; suma != total {
		t.Fatalf("PIURA: semanas suman %d, total %d", suma, total)
	}
	if buscar(t, cubo, features.OtrosDepartamentos, TodasSemanas, TodasEtapas).Casos == 0 {
		t.Fatal("TACNA debía caer en OTROS")
	}
}

func TestCasosEsperados(t *testing.T) {
	in := entradaPrueba(t, 4000)
	rep, err := Generar(in)
	if err != nil {
		t.Fatal(err)
	}
	var suma float64
	for i := 0; i < in.Test.N; i++ {
		suma += ml.Calibrar(ml.Probabilidad(in.Pesos, in.Test.Fila(i)), in.AjustePrior)
	}
	nac := buscar(t, rep.Riesgo, TodosDepartamentos, TodasSemanas, TodasEtapas)
	if math.Abs(nac.CasosEsperados-suma) > 1e-9 || math.Abs(nac.Fraccion-1) > 1e-12 {
		t.Fatalf("esperados nacionales %.12f ≠ suma fila a fila %.12f", nac.CasosEsperados, suma)
	}
	var porDepto float64
	for _, c := range rep.Riesgo.Celdas {
		if c.Departamento != TodosDepartamentos && c.Semana == TodasSemanas && c.Etapa == TodasEtapas {
			porDepto += c.CasosEsperados
		}
	}
	if math.Abs(porDepto-suma) > 1e-9 {
		t.Fatalf("la suma por departamento %.12f ≠ nacional %.12f", porDepto, suma)
	}
}

func TestPrioridades(t *testing.T) {
	rep, err := Generar(entradaPrueba(t, 6000))
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Prioridades) == 0 || len(rep.Prioridades) > 10 {
		t.Fatalf("%d prioridades", len(rep.Prioridades))
	}
	for k, c := range rep.Prioridades {
		if c.Insuficiente || c.Semana == TodasSemanas || c.Etapa == TodasEtapas ||
			c.Departamento == TodosDepartamentos || c.Departamento == features.OtrosDepartamentos {
			t.Fatalf("prioridad %d no es una combinación concreta: %+v", k, c)
		}
		if k > 0 && c.CasosEsperados > rep.Prioridades[k-1].CasosEsperados {
			t.Fatal("prioridades no ordenadas por casos esperados")
		}
	}
}

func TestGenerarDeterminista(t *testing.T) {
	in := entradaPrueba(t, 3000)
	a, err := Generar(in)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Generar(in)
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if !bytes.Equal(ja, jb) {
		t.Fatal("dos corridas con la misma entrada difieren")
	}
	if a.Escalabilidad.Mediciones[0].Nodos != 1 || a.Escalabilidad.Mediciones[0].Speedup != 1 {
		t.Fatalf("escalabilidad mal leída: %+v", a.Escalabilidad.Mediciones[0])
	}
	if a.LatenciaAPI.LimiteP95 != 100 || len(a.LatenciaAPI.Corridas) == 0 {
		t.Fatalf("latencia mal leída: %+v", a.LatenciaAPI)
	}
}

func TestGenerarAUCReferenciaDistinta(t *testing.T) {
	in := entradaPrueba(t, 2000)
	in.AUCReferencia = 0.123
	if _, err := Generar(in); err == nil {
		t.Fatal("un AUC de referencia distinto debía fallar")
	}
}

func buscar(t *testing.T, c Cubo, d string, s int, e string) Celda {
	t.Helper()
	for _, x := range c.Celdas {
		if x.Departamento == d && x.Semana == s && x.Etapa == e {
			return x
		}
	}
	t.Fatalf("no existe la celda %s/%d/%s", d, s, e)
	return Celda{}
}

func TestGenerarDimensionesDistintas(t *testing.T) {
	for _, delta := range []int{-2, 2} {
		in := entradaPrueba(t, 500)
		d := in.Test.D + delta
		x := make([]float64, in.Test.N*d)
		ds, err := ml.NuevoDataset(x, in.Test.Y, d)
		if err != nil {
			t.Fatal(err)
		}
		in.Test = ds
		if _, err := Generar(in); err == nil {
			t.Fatalf("D del test %+d respecto del esquema debía fallar sin pánico", delta)
		}
		if _, err := Cruzar(ds, ds.Y, make([]float64, ds.N), in.Esquema, 1); err == nil {
			t.Fatalf("Cruzar con D %+d debía fallar", delta)
		}
	}
	in := entradaPrueba(t, 500)
	if _, err := Cruzar(in.Test, in.Test.Y, make([]float64, 3), in.Esquema, 1); err == nil {
		t.Fatal("pc de otro largo debía fallar")
	}
	if _, err := Cruzar(nil, nil, nil, in.Esquema, 1); err == nil {
		t.Fatal("dataset nulo debía fallar")
	}
}

func TestPrioridadesVaciasSeSerializanComoLista(t *testing.T) {
	in := entradaPrueba(t, 500)
	in.MinCasos = 1_000_000 // ninguna celda alcanza
	rep, err := Generar(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{-1, 0} {
		if p := Prioridades(rep.Riesgo, n); p == nil || len(p) != 0 {
			t.Fatalf("Prioridades(%d) = %v, quería lista vacía", n, p)
		}
	}
	b, _ := json.Marshal(rep)
	if !bytes.Contains(b, []byte(`"prioridades":[]`)) {
		t.Fatal(`sin celdas suficientes, "prioridades" debe ser [] y no null`)
	}
}

func TestCruzarRiesgoNacionalCero(t *testing.T) {
	ds, esq, _ := sintetico(t, 300, 4)
	cubo, err := Cruzar(ds, ds.Y, make([]float64, ds.N), esq, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(cubo); err != nil {
		t.Fatalf("con riesgo nacional 0 el cubo no se serializa: %v", err)
	}
}
