package api

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
)

// MinLargoClave es el largo mínimo de una clave sembrada (NIST SP 800-63B pide ≥ 8;
// se exige más porque son cuentas con acceso al clúster).
const MinLargoClave = 12

// Variables de entorno de la siembra: un par correo/clave por rol. Las claves
// nunca se versionan (deploy/.env está en .dockerignore y .gitignore).
var VariablesSiembra = []struct{ Rol, Correo, Clave string }{
	{"admin", "VIGIA_ADMIN_CORREO", "VIGIA_ADMIN_CLAVE"},
	{"analista", "VIGIA_ANALISTA_CORREO", "VIGIA_ANALISTA_CLAVE"},
}

// SembrarUsuarios crea o actualiza las cuentas definidas en el entorno (getenv)
// y devuelve los correos sembrados. Un rol sin variables se omite; uno con
// correo sin clave, o con clave corta, es un error. Lo usan cmd/usuarios y el
// coordinador al arrancar en modo servicio.
func SembrarUsuarios(ctx context.Context, repo almacen.Usuarios, getenv func(string) string) ([]string, error) {
	var sembrados []string
	for _, v := range VariablesSiembra {
		correo, clave := almacen.NormalizarCorreo(getenv(v.Correo)), getenv(v.Clave)
		if correo == "" && clave == "" {
			continue
		}
		if correo == "" || utf8.RuneCountInString(clave) < MinLargoClave {
			return sembrados, fmt.Errorf("usuarios: %s y %s: falta el correo o la clave tiene menos de %d caracteres",
				v.Correo, v.Clave, MinLargoClave)
		}
		h, err := auth.Hash(clave)
		if err != nil {
			return sembrados, err
		}
		if err := repo.GuardarUsuario(ctx, almacen.Usuario{Correo: correo, Hash: h, Rol: v.Rol, Creado: time.Now().UTC()}); err != nil {
			return sembrados, fmt.Errorf("usuarios: %s: %w", correo, err)
		}
		sembrados = append(sembrados, correo+" ("+v.Rol+")")
	}
	return sembrados, nil
}
