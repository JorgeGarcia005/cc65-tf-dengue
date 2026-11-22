package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

type claveCtx struct{}

// ReclamosDe devuelve los reclamos que Autenticar dejó en el contexto.
func ReclamosDe(ctx context.Context) (Reclamos, bool) {
	r, ok := ctx.Value(claveCtx{}).(Reclamos)
	return r, ok
}

// ConReclamos devuelve un contexto con rec (para tests y para el ticket del WebSocket).
func ConReclamos(ctx context.Context, rec Reclamos) context.Context {
	return context.WithValue(ctx, claveCtx{}, rec)
}

// Autenticar exige "Authorization: Bearer <jwt>" válido: si falta o no verifica
// responde 401 con WWW-Authenticate; si verifica, deja los reclamos en el contexto.
func (f *Firmador) Autenticar(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok, ok := tokenBearer(r.Header.Get("Authorization"))
		if !ok {
			noAutenticado(w, `Bearer realm="vigia"`, "falta el token de acceso")
			return
		}
		rec, err := f.Verificar(tok)
		if err != nil {
			noAutenticado(w, `Bearer realm="vigia", error="invalid_token"`, "token inválido o vencido")
			return
		}
		next.ServeHTTP(w, r.WithContext(ConReclamos(r.Context(), rec)))
	})
}

// Requiere deja pasar solo a los roles dados: sin reclamos en el contexto → 401;
// rol no incluido → 403. Se coloca detrás de Autenticar.
func Requiere(roles ...Rol) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec, ok := ReclamosDe(r.Context())
			if !ok {
				noAutenticado(w, `Bearer realm="vigia"`, "falta el token de acceso")
				return
			}
			if !slices.Contains(roles, rec.Rol) {
				escribirError(w, http.StatusForbidden, "tu rol no tiene acceso a este recurso")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func tokenBearer(cab string) (string, bool) {
	esquema, tok, ok := strings.Cut(cab, " ")
	if !ok || !strings.EqualFold(esquema, "Bearer") {
		return "", false
	}
	tok = strings.TrimSpace(tok)
	return tok, tok != ""
}

func noAutenticado(w http.ResponseWriter, desafio, msg string) {
	w.Header().Set("WWW-Authenticate", desafio)
	escribirError(w, http.StatusUnauthorized, msg)
}

// escribirError usa el mismo formato que internal/api: {"error": "..."}.
func escribirError(w http.ResponseWriter, codigo int, msg string) {
	b, _ := json.Marshal(map[string]string{"error": msg})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(codigo)
	_, _ = w.Write(append(b, '\n'))
}
