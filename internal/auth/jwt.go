// Package auth implementa la seguridad de la API con la biblioteca estándar
// (ADR-0007): JWT HS256, contraseñas con PBKDF2-SHA256 y middlewares de
// autenticación y autorización por rol.
package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Emisor y Audiencia que se firman y se exigen en cada token.
const (
	Emisor          = "cc65-tf"
	Audiencia       = "vigia"
	MinBytesSecreto = 32
)

// Rol de un usuario. Solo existen los dos de ADR-0007.
type Rol string

const (
	RolAdmin    Rol = "admin"
	RolAnalista Rol = "analista"
)

// Valido indica si r es uno de los roles conocidos.
func (r Rol) Valido() bool { return r == RolAdmin || r == RolAnalista }

// Errores de verificación. Se comparan con errors.Is.
var (
	ErrMalformado   = errors.New("token malformado")
	ErrAlgoritmo    = errors.New("algoritmo de firma no permitido")
	ErrFirma        = errors.New("firma inválida")
	ErrVencido      = errors.New("token vencido")
	ErrNoVigente    = errors.New("token aún no vigente")
	ErrReclamos     = errors.New("reclamos inválidos")
	ErrSecretoCorto = fmt.Errorf("el secreto JWT debe tener al menos %d bytes", MinBytesSecreto)
)

// Reclamos del token. Comparables con ==.
type Reclamos struct {
	Sub string `json:"sub"`
	Rol Rol    `json:"rol"`
	Iss string `json:"iss"`
	Aud string `json:"aud"`
	Iat int64  `json:"iat"`
	Nbf int64  `json:"nbf"`
	Exp int64  `json:"exp"`
}

// b64 es base64url sin relleno en modo estricto (RFC 7515 §2): rechaza bits
// sobrantes distintos de cero.
var b64 = base64.RawURLEncoding.Strict()

// La cabecera es siempre la misma: se precalcula y Verificar la exige tal cual.
var cabeceraB64 = b64.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

// Firmador emite y verifica tokens. Es inmutable tras construirlo, así que es
// seguro usarlo desde muchas goroutines.
type Firmador struct {
	secreto  []byte
	duracion time.Duration
	ahora    func() time.Time
}

// NuevoFirmador copia el secreto (≥ 32 bytes) y fija la vida de cada token.
func NuevoFirmador(secreto []byte, duracion time.Duration) (*Firmador, error) {
	if len(secreto) < MinBytesSecreto {
		return nil, ErrSecretoCorto
	}
	if duracion <= 0 {
		return nil, errors.New("la duración del token debe ser positiva")
	}
	return &Firmador{secreto: bytes.Clone(secreto), duracion: duracion, ahora: time.Now}, nil
}

// Firmar emite un token para sub con el rol dado.
func (f *Firmador) Firmar(sub string, rol Rol) (string, Reclamos, error) {
	if sub == "" || !rol.Valido() {
		return "", Reclamos{}, fmt.Errorf("%w: sub vacío o rol %q desconocido", ErrReclamos, rol)
	}
	t := f.ahora().Unix()
	rec := Reclamos{Sub: sub, Rol: rol, Iss: Emisor, Aud: Audiencia, Iat: t, Nbf: t,
		Exp: t + int64(f.duracion/time.Second)}
	carga, err := json.Marshal(rec)
	if err != nil {
		return "", Reclamos{}, err
	}
	datos := cabeceraB64 + "." + b64.EncodeToString(carga)
	return datos + "." + b64.EncodeToString(f.firma(datos)), rec, nil
}

// Verificar comprueba estructura, cabecera, firma (en tiempo constante),
// vigencia, emisor, audiencia y rol, en ese orden. La cabecera tiene que ser
// exactamente la que emite Firmar y el base64url se decodifica en modo
// estricto (sin bits sobrantes ni saltos de línea): cada token válido tiene una
// sola representación.
func (f *Firmador) Verificar(token string) (Reclamos, error) {
	if strings.ContainsAny(token, "\r\n") {
		return Reclamos{}, ErrMalformado
	}
	partes := strings.Split(token, ".")
	if len(partes) != 3 {
		return Reclamos{}, ErrMalformado
	}
	if partes[0] != cabeceraB64 {
		return Reclamos{}, errorDeCabecera(partes[0])
	}
	firma, err := b64.DecodeString(partes[2])
	if err != nil {
		return Reclamos{}, ErrMalformado
	}
	if !hmac.Equal(firma, f.firma(partes[0]+"."+partes[1])) {
		return Reclamos{}, ErrFirma
	}
	carga, err := b64.DecodeString(partes[1])
	if err != nil {
		return Reclamos{}, ErrMalformado
	}
	var rec Reclamos
	if err := json.Unmarshal(carga, &rec); err != nil {
		return Reclamos{}, ErrMalformado
	}
	ahora := f.ahora().Unix()
	switch {
	case ahora >= rec.Exp:
		return Reclamos{}, ErrVencido
	case ahora < rec.Nbf:
		return Reclamos{}, ErrNoVigente
	case rec.Iss != Emisor || rec.Aud != Audiencia || rec.Sub == "" || !rec.Rol.Valido():
		return Reclamos{}, ErrReclamos
	}
	return rec, nil
}

// errorDeCabecera clasifica una cabecera distinta de la fija: un "alg" explícito
// que no es HS256 es ErrAlgoritmo; cualquier otra diferencia (typ, crit, claves
// en mayúscula, null, base64 inválido) es ErrMalformado.
func errorDeCabecera(parte string) error {
	cab, err := b64.DecodeString(parte)
	if err != nil {
		return ErrMalformado
	}
	var c map[string]json.RawMessage
	if err := json.Unmarshal(cab, &c); err != nil || c == nil {
		return ErrMalformado
	}
	var alg string
	if raw, ok := c["alg"]; ok && json.Unmarshal(raw, &alg) == nil && alg != "HS256" {
		return fmt.Errorf("%w: %q", ErrAlgoritmo, alg)
	}
	return ErrMalformado
}

func (f *Firmador) firma(datos string) []byte {
	m := hmac.New(sha256.New, f.secreto)
	m.Write([]byte(datos))
	return m.Sum(nil)
}
