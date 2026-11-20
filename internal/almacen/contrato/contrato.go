// Package contrato tiene las pruebas que toda implementación de
// almacen.Almacen y almacen.Cache debe pasar: la de memoria en los tests
// unitarios y las de Mongo y Redis en los de integración.
package contrato

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/features"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest"
)

func esquema(t *testing.T) *features.Esquema {
	t.Helper()
	regs := []ingest.Registro{
		{Departamento: "LORETO", Ano: 2015, Semana: 3, EdadAnios: 20, Sexo: 'F'},
		{Departamento: "PIURA", Ano: 2020, Semana: 30, EdadAnios: 40, Sexo: 'M'},
	}
	e, err := features.Ajustar(regs, 2011, 1)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// Almacen prueba una implementación vacía de almacen.Almacen.
func Almacen(t *testing.T, a almacen.Almacen) {
	ctx := context.Background()
	t.Run("ModeloVigenteEsElUltimoPublicado", func(t *testing.T) {
		if _, ok, err := a.ModeloVigente(ctx); err != nil || ok {
			t.Fatalf("almacén vacío: ok=%v err=%v", ok, err)
		}
		base := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
		esq := esquema(t)
		for i, v := range []string{"aaaaaaaaaaaa", "cccccccccccc", "bbbbbbbbbbbb"} {
			m := almacen.Modelo{Version: v, Pesos: make([]float64, esq.D()+1), Esquema: esq,
				Umbral: 0.5, AjustePrior: 1.9, Generado: "x", Publicado: base.Add(time.Duration([]int{0, 2, 1}[i]) * time.Hour)}
			m.Pesos[0] = float64(i) + 0.125
			if err := a.GuardarModelo(ctx, m); err != nil {
				t.Fatal(err)
			}
		}
		m, ok, err := a.ModeloVigente(ctx)
		if err != nil || !ok || m.Version != "cccccccccccc" || m.Pesos[0] != 1.125 {
			t.Fatalf("vigente: %+v ok=%v err=%v", m, ok, err)
		}
		if m.Esquema == nil || !slices.Equal(m.Esquema.Nombres, esq.Nombres) || m.Esquema.DesvEdad != esq.DesvEdad {
			t.Fatalf("esquema no conservado: %+v", m.Esquema)
		}
		if err := m.Esquema.Preparar(); err != nil {
			t.Fatalf("el esquema leído no se puede preparar: %v", err)
		}
		// PR-40 H7: ni quien guardó ni quien leyó puede alterar el modelo guardado.
		mediaOriginal, depOriginal := m.Esquema.MediaEdad, m.Esquema.Departamentos[0]
		esq.MediaEdad, esq.Departamentos[0] = -1, "ALTERADO"
		m.Esquema.MediaEdad, m.Esquema.Departamentos[0], m.Pesos[0] = -2, "ALTERADO", -3
		otra, _, _ := a.ModeloVigente(ctx)
		if otra.Esquema.MediaEdad != mediaOriginal || otra.Esquema.Departamentos[0] != depOriginal || otra.Pesos[0] != 1.125 {
			t.Fatalf("el almacén comparte el modelo con quien lo guardó o lo leyó: media %v, depto %q, peso %v",
				otra.Esquema.MediaEdad, otra.Esquema.Departamentos[0], otra.Pesos[0])
		}
	})

	t.Run("EntrenamientoSeReemplazaPorID", func(t *testing.T) {
		if _, ok, err := a.Entrenamiento(ctx, "no-existe"); err != nil || ok {
			t.Fatalf("inexistente: ok=%v err=%v", ok, err)
		}
		ini := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
		e := almacen.Entrenamiento{ID: "ent-x-1", Estado: almacen.EnCurso, Epocas: 3, Tasa: 2, Lambda: 1e-4, Inicio: ini}
		if err := a.GuardarEntrenamiento(ctx, e); err != nil {
			t.Fatal(err)
		}
		fin := ini.Add(time.Minute)
		e.Estado, e.Epoca, e.Perdida, e.Historia, e.Fin, e.VersionModelo = almacen.Terminado, 3, 0.5, []float64{0.69, 0.6, 0.5}, &fin, "dddddddddddd"
		e.Nodos = []cluster.EstadoNodo{{Nombre: "node-1", Remoto: "10.0.0.2:5000", Workers: 8, Sano: true, Carga: 2}}
		e.Reasignaciones = 1
		if err := a.GuardarEntrenamiento(ctx, e); err != nil {
			t.Fatal(err)
		}
		got, ok, err := a.Entrenamiento(ctx, "ent-x-1")
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
		if got.Estado != almacen.Terminado || !slices.Equal(got.Historia, e.Historia) || got.Fin == nil ||
			!got.Fin.Equal(fin) || !got.Inicio.Equal(ini) || len(got.Nodos) != 1 || got.Nodos[0] != e.Nodos[0] ||
			got.Reasignaciones != 1 || got.VersionModelo != e.VersionModelo {
			t.Fatalf("leído %+v, guardado %+v", got, e)
		}
		got.Historia[0] = 99 // no debe alterar lo guardado
		again, _, _ := a.Entrenamiento(ctx, "ent-x-1")
		if again.Historia[0] != 0.69 {
			t.Fatal("el almacén comparte el slice de historia con quien lo lee")
		}
	})

	// PR-40 H3: crear no sobrescribe un entrenamiento existente.
	t.Run("CrearEntrenamientoNoSobrescribe", func(t *testing.T) {
		ini := time.Date(2026, 11, 17, 13, 0, 0, 0, time.UTC)
		e := almacen.Entrenamiento{ID: "ent-dup", Estado: almacen.EnCurso, Epocas: 3, Inicio: ini}
		if err := a.CrearEntrenamiento(ctx, e); err != nil {
			t.Fatal(err)
		}
		otro := e
		otro.Epocas = 9
		if err := a.CrearEntrenamiento(ctx, otro); !errors.Is(err, almacen.ErrDuplicado) {
			t.Fatalf("segundo CrearEntrenamiento con el mismo id: %v, se esperaba ErrDuplicado", err)
		}
		got, _, _ := a.Entrenamiento(ctx, "ent-dup")
		if got.Epocas != 3 {
			t.Fatalf("la colisión alteró el existente: %d épocas", got.Epocas)
		}
	})

	// PR-40 H1: lo necesario para reconciliar al arrancar.
	t.Run("EnCursoYModeloDeEntrenamiento", func(t *testing.T) {
		ini := time.Date(2026, 11, 17, 14, 0, 0, 0, time.UTC)
		for i, est := range []string{almacen.EnCurso, almacen.Terminado, almacen.EnCurso} {
			e := almacen.Entrenamiento{ID: fmt.Sprintf("ent-rec-%d", i), Estado: est, Inicio: ini}
			if err := a.GuardarEntrenamiento(ctx, e); err != nil {
				t.Fatal(err)
			}
		}
		en, err := a.EntrenamientosEnCurso(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, e := range en {
			if strings.HasPrefix(e.ID, "ent-rec-") {
				ids = append(ids, e.ID)
			}
		}
		slices.Sort(ids)
		if !slices.Equal(ids, []string{"ent-rec-0", "ent-rec-2"}) {
			t.Fatalf("en curso: %v", ids)
		}
		if _, ok, err := a.ModeloDeEntrenamiento(ctx, "ent-rec-0"); err != nil || ok {
			t.Fatalf("sin modelo: ok=%v err=%v", ok, err)
		}
		esq := esquema(t)
		// PR-40 ronda 2, H1: el modelo guarda el resumen completo del entrenamiento
		// (7 épocas: no múltiplo de PersistirCada), para reconstruirlo al reiniciar.
		fin := ini.Add(time.Minute)
		resA := almacen.ResumenEntrenamiento{ID: "ent-rec-0", Epoca: 7, Perdida: 0.51,
			Historia: []float64{0.69, 0.65, 0.6, 0.57, 0.55, 0.53, 0.51}, Reasignaciones: 1, Fin: fin,
			Nodos: []cluster.EstadoNodo{{Nombre: "node-1", Remoto: "10.0.0.2:5000", Workers: 8, Sano: true, Carga: 2}}}
		m := almacen.Modelo{Version: "ffffffffffff", Pesos: make([]float64, esq.D()+1), Esquema: esq, Umbral: 0.5,
			Entrenamientos: []almacen.ResumenEntrenamiento{resA}, Publicado: ini}
		if err := a.GuardarModelo(ctx, m); err != nil {
			t.Fatal(err)
		}
		got, ok, err := a.ModeloDeEntrenamiento(ctx, "ent-rec-0")
		if err != nil || !ok || got.Version != "ffffffffffff" {
			t.Fatalf("modelo de ent-rec-0: %+v ok=%v err=%v", got, ok, err)
		}
		if r, ok := got.Resumen("ent-rec-0"); !ok || !mismoResumen(r, resA) {
			t.Fatalf("resumen leído %+v, guardado %+v", r, resA)
		}

		// PR-40 ronda 2, H11: otro entrenamiento que produce la misma versión no
		// borra el vínculo del primero.
		resB := almacen.ResumenEntrenamiento{ID: "ent-rec-2", Epoca: 3, Perdida: 0.6, Historia: []float64{0.69, 0.65, 0.6},
			Nodos: []cluster.EstadoNodo{}, Fin: fin.Add(time.Second)}
		m2 := m
		m2.Entrenamientos, m2.Publicado = []almacen.ResumenEntrenamiento{resB}, ini.Add(10*time.Hour)
		if err := a.GuardarModelo(ctx, m2); err != nil {
			t.Fatal(err)
		}
		for id, want := range map[string]almacen.ResumenEntrenamiento{"ent-rec-0": resA, "ent-rec-2": resB} {
			got, ok, err := a.ModeloDeEntrenamiento(ctx, id)
			r, okR := got.Resumen(id)
			if err != nil || !ok || !okR || got.Version != "ffffffffffff" || !mismoResumen(r, want) {
				t.Fatalf("tras la segunda publicación, %s: ok=%v err=%v resumen %+v", id, ok, err, r)
			}
		}
		vig, _, _ := a.ModeloVigente(ctx)
		if vig.Version != "ffffffffffff" || !vig.Publicado.Equal(ini.Add(10*time.Hour)) || len(vig.Entrenamientos) != 2 {
			t.Fatalf("vigente tras republicar: %s %v, %d resúmenes", vig.Version, vig.Publicado, len(vig.Entrenamientos))
		}
	})

	t.Run("RegistrarPrediccionesConcurrentes", func(t *testing.T) {
		var wg sync.WaitGroup
		errs := make(chan error, 16)
		for i := range 16 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- a.RegistrarPrediccion(ctx, almacen.RegistroPrediccion{
					Fecha: time.Now().UTC(), VersionModelo: "eeeeeeeeeeee", Riesgo: 0.1, Clase: i % 2,
					Consulta: almacen.ConsultaNormalizada{Departamento: "LORETO", Ano: 2024, Semana: i + 1, EdadAnios: 30, Sexo: "F"},
				})
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
	})
}

// Cache prueba una implementación de almacen.Cache. esperar avanza el reloj
// (real o simulado) más allá de un TTL dado.
func Cache(t *testing.T, c almacen.Cache, esperar func(time.Duration)) {
	ctx := context.Background()
	clave := fmt.Sprintf("test:cache:%d", time.Now().UnixNano())
	if _, ok, err := c.Obtener(ctx, clave); err != nil || ok {
		t.Fatalf("clave nueva: ok=%v err=%v", ok, err)
	}
	if err := c.Guardar(ctx, clave, []byte(`{"riesgo":0.1}`), time.Hour); err != nil {
		t.Fatal(err)
	}
	if v, ok, err := c.Obtener(ctx, clave); err != nil || !ok || string(v) != `{"riesgo":0.1}` {
		t.Fatalf("obtener: %q ok=%v err=%v", v, ok, err)
	}
	if err := c.Guardar(ctx, clave, []byte(`{"riesgo":0.2}`), time.Hour); err != nil {
		t.Fatal(err)
	}
	if v, _, _ := c.Obtener(ctx, clave); string(v) != `{"riesgo":0.2}` {
		t.Fatalf("no se reemplazó: %q", v)
	}
	corta := clave + ":ttl"
	if err := c.Guardar(ctx, corta, []byte("x"), time.Second); err != nil {
		t.Fatal(err)
	}
	esperar(1500 * time.Millisecond)
	if _, ok, err := c.Obtener(ctx, corta); err != nil || ok {
		t.Fatalf("la clave debía expirar: ok=%v err=%v", ok, err)
	}
}

func mismoResumen(a, b almacen.ResumenEntrenamiento) bool {
	return a.ID == b.ID && a.Epoca == b.Epoca && a.Perdida == b.Perdida && slices.Equal(a.Historia, b.Historia) &&
		slices.Equal(a.Nodos, b.Nodos) && a.Reasignaciones == b.Reasignaciones && a.Fin.Equal(b.Fin)
}
