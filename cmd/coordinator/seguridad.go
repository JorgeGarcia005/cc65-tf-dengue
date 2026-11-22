package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/api"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
)

// DuracionToken es la vida de un JWT del login (ADR-0007).
const DuracionToken = 60 * time.Minute

// prepararSeguridad arma el firmador de JWT con $JWT_SECRETO y siembra las
// cuentas definidas en $VIGIA_ADMIN_* y $VIGIA_ANALISTA_* en alm (el mismo
// almacén que usará la API). JWT_SECRETO es obligatorio (#61): sin él las
// rutas protegidas no podrían verificar tokens, y el servicio no arranca.
func prepararSeguridad(ctx context.Context, alm almacen.Almacen, getenv func(string) string, logf func(string, ...any)) (*auth.Firmador, error) {
	secreto := getenv("JWT_SECRETO")
	if secreto == "" {
		return nil, errors.New("JWT_SECRETO es obligatorio en modo servicio: cópialo en deploy/.env (ver deploy/.env.example)")
	}
	f, err := auth.NuevoFirmador([]byte(secreto), DuracionToken)
	if err != nil {
		return nil, fmt.Errorf("JWT_SECRETO: %w", err)
	}
	repo, ok := alm.(almacen.Usuarios)
	if !ok {
		return f, nil
	}
	sembrados, err := api.SembrarUsuarios(ctx, repo, getenv)
	if err != nil {
		return nil, err
	}
	for _, s := range sembrados {
		logf("usuario sembrado: %s", s)
	}
	return f, nil
}
