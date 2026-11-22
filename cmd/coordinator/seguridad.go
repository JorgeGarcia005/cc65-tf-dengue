package main

import (
	"context"
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
// almacén que usará la API). Sin JWT_SECRETO devuelve nil y el login responde
// 503 (en #61 pasa a ser obligatorio, cuando las rutas exijan token).
func prepararSeguridad(ctx context.Context, alm almacen.Almacen, getenv func(string) string, logf func(string, ...any)) (*auth.Firmador, error) {
	var f *auth.Firmador
	if secreto := getenv("JWT_SECRETO"); secreto != "" {
		var err error
		if f, err = auth.NuevoFirmador([]byte(secreto), DuracionToken); err != nil {
			return nil, fmt.Errorf("JWT_SECRETO: %w", err)
		}
	} else {
		logf("sin JWT_SECRETO: POST /api/v1/auth/login responde 503")
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
