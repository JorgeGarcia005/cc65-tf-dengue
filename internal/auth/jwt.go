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

// La cabecera es siempre la misma: se precalcula y se exige tal cual al decodificar.
var cabeceraB64 = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

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
	datos := cabeceraB64 + "." + base64.RawURLEncoding.EncodeToString(carga)
	return datos + "." + base64.RawURLEncoding.EncodeToString(f.firma(datos)), rec, nil
}

// Verificar comprueba estructura, algoritmo, firma (en tiempo constante),
// vigencia, emisor, audiencia y rol, en ese orden.
func (f *Firmador) Verificar(token string) (Reclamos, error) {
	partes := strings.Split(token, ".")
	if len(partes) != 3 {
		return Reclamos{}, ErrMalformado
	}
	cab, err := base64.RawURLEncoding.DecodeString(partes[0])
	if err != nil {
		return Reclamos{}, ErrMalformado
	}
	var c struct{ Alg, Typ string }
	if err := json.Unmarshal(cab, &c); err != nil {
		return Reclamos{}, ErrMalformado
	}
	if c.Alg != "HS256" {
		return Reclamos{}, fmt.Errorf("%w: %q", ErrAlgoritmo, c.Alg)
	}
	firma, err := base64.RawURLEncoding.DecodeString(partes[2])
	if err != nil {
		return Reclamos{}, ErrMalformado
	}
	if !hmac.Equal(firma, f.firma(partes[0]+"."+partes[1])) {
		return Reclamos{}, ErrFirma
	}
	carga, err := base64.RawURLEncoding.DecodeString(partes[1])
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

func (f *Firmador) firma(datos string) []byte {
	m := hmac.New(sha256.New, f.secreto)
	m.Write([]byte(datos))
	return m.Sum(nil)
}
