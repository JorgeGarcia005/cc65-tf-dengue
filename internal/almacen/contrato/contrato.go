// Package contrato tiene las pruebas que toda implementación de
// almacen.Almacen y almacen.Cache debe pasar: la de memoria en los tests
// unitarios y las de Mongo y Redis en los de integración.
package contrato

import (
	"context"
	"fmt"
	"slices"
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
	clave := fmt.Sprintf("pred:test:%d", time.Now().UnixNano())
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
