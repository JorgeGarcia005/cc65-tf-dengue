package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMiddleware(t *testing.T) {
	f := firmadorFijo(t, time.Now())
	admin, _, _ := f.Firmar("admin@vigia.pe", RolAdmin)
	analista, _, _ := f.Firmar("ana@diresa.pe", RolAnalista)
	viejo := firmadorFijo(t, time.Now().Add(-2*time.Hour))
	vencido, _, _ := viejo.Firmar("admin@vigia.pe", RolAdmin)

	var visto Reclamos
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec, ok := ReclamosDe(r.Context())
		if !ok {
			t.Error("el handler protegido no recibió reclamos")
		}
		visto = rec
		w.WriteHeader(http.StatusNoContent)
	})
	soloAdmin := f.Autenticar(Requiere(RolAdmin)(final))
	ambos := f.Autenticar(Requiere(RolAdmin, RolAnalista)(final))

	casos := []struct {
		nombre, cabecera string
		h                http.Handler
		codigo           int
		sub              string
		rol              Rol
	}{
		{"sin token", "", soloAdmin, 401, "", ""},
		{"esquema Basic", "Basic YTpi", soloAdmin, 401, "", ""},
		{"token basura", "Bearer abc.def.ghi", soloAdmin, 401, "", ""},
		{"token vencido", "Bearer " + vencido, soloAdmin, 401, "", ""},
		{"Bearer sin token", "Bearer ", soloAdmin, 401, "", ""},
		{"analista en ruta admin", "Bearer " + analista, soloAdmin, 403, "", ""},
		{"admin en ruta admin", "Bearer " + admin, soloAdmin, 204, "admin@vigia.pe", RolAdmin},
		{"analista en ruta compartida", "Bearer " + analista, ambos, 204, "ana@diresa.pe", RolAnalista},
		{"bearer en minúsculas", "bearer " + admin, ambos, 204, "admin@vigia.pe", RolAdmin},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			visto = Reclamos{}
			req := httptest.NewRequest("GET", "/x", nil)
			if c.cabecera != "" {
				req.Header.Set("Authorization", c.cabecera)
			}
			w := httptest.NewRecorder()
			c.h.ServeHTTP(w, req)
			if w.Code != c.codigo {
				t.Fatalf("código = %d, quería %d (cuerpo %s)", w.Code, c.codigo, w.Body)
			}
			if c.codigo == 204 {
				if visto.Sub != c.sub || visto.Rol != c.rol {
					t.Fatalf("reclamos = %q/%q, quería %q/%q", visto.Sub, visto.Rol, c.sub, c.rol)
				}
				return
			}
			var cuerpo map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &cuerpo); err != nil || cuerpo["error"] == "" {
				t.Fatalf("cuerpo de error inválido: %q", w.Body)
			}
			if c.codigo == 401 && !strings.HasPrefix(w.Header().Get("WWW-Authenticate"), "Bearer") {
				t.Fatal("401 sin WWW-Authenticate: Bearer")
			}
			if c.nombre == "token vencido" && !strings.Contains(w.Header().Get("WWW-Authenticate"), `error="invalid_token"`) {
				t.Fatalf("token vencido sin error=invalid_token: %q", w.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestMiddlewareRequiereSinAutenticar(t *testing.T) {
	// Requiere sin Autenticar delante (error de cableado) responde 401, nunca deja pasar.
	h := Requiere(RolAdmin)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("no debía llegar")
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	if w.Code != 401 {
		t.Fatalf("código = %d, quería 401", w.Code)
	}
}

func TestMiddlewareRolesNoSeAlteran(t *testing.T) {
	f := firmadorFijo(t, time.Now())
	admin, _, _ := f.Firmar("admin@vigia.pe", RolAdmin)
	roles := []Rol{RolAdmin}
	h := f.Autenticar(Requiere(roles...)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})))
	roles[0] = RolAnalista // el llamador cambia su slice después de armar el middleware
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer "+admin)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("la política cambió con el slice del llamador: código %d", w.Code)
	}
}

func TestMiddlewareConcurrente(t *testing.T) {
	f := firmadorFijo(t, time.Now())
	admin, _, _ := f.Firmar("admin@vigia.pe", RolAdmin)
	analista, _, _ := f.Firmar("ana@diresa.pe", RolAnalista)
	h := f.Autenticar(Requiere(RolAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec, ok := ReclamosDe(r.Context())
		if !ok || rec.Rol != RolAdmin || rec.Sub != "admin@vigia.pe" {
			t.Errorf("reclamos en el contexto: %+v %v", rec, ok)
		}
	})))
	casos := []struct {
		cabecera string
		codigo   int
	}{{"Bearer " + admin, 200}, {"Bearer " + analista, 403}, {"Bearer x.y.z", 401}, {"", 401}}
	var wg sync.WaitGroup
	for i := 0; i < 128; i++ {
		c := casos[i%len(casos)]
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/x", nil)
			if c.cabecera != "" {
				req.Header.Set("Authorization", c.cabecera)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != c.codigo {
				t.Errorf("%q: código %d, quería %d", c.cabecera, w.Code, c.codigo)
			}
		}()
	}
	wg.Wait()
}
