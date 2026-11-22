package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Parámetros de PBKDF2-HMAC-SHA256 (ADR-0007; OWASP 2023 recomienda 600 000).
const (
	Iteraciones    = 600_000
	maxIteraciones = 10_000_000 // tope al leer un hash: evita que uno manipulado congele el login
	bytesSal       = 16
	bytesClave     = 32
	prefijo        = "pbkdf2-sha256"
)

// Hash deriva la clave con una sal aleatoria y devuelve
// "pbkdf2-sha256$<iter>$<sal b64>$<clave b64>".
func Hash(clave string) (string, error) { return hashCon(clave, Iteraciones) }

func hashCon(clave string, iter int) (string, error) {
	sal := make([]byte, bytesSal)
	if _, err := rand.Read(sal); err != nil {
		return "", err
	}
	k, err := pbkdf2.Key(sha256.New, clave, sal, iter, bytesClave)
	if err != nil {
		return "", err
	}
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("%s$%d$%s$%s", prefijo, iter, b64.EncodeToString(sal), b64.EncodeToString(k)), nil
}

// Comparar indica si clave corresponde al hash guardado. La comparación final es
// en tiempo constante; un hash mal formado nunca verifica.
func Comparar(guardado, clave string) bool {
	partes := strings.Split(guardado, "$")
	if len(partes) != 4 || partes[0] != prefijo {
		return false
	}
	iter, err := strconv.Atoi(partes[1])
	if err != nil || iter < 1 || iter > maxIteraciones {
		return false
	}
	b64 := base64.RawStdEncoding
	sal, err1 := b64.DecodeString(partes[2])
	esperado, err2 := b64.DecodeString(partes[3])
	if err1 != nil || err2 != nil || len(sal) == 0 || len(esperado) == 0 {
		return false
	}
	k, err := pbkdf2.Key(sha256.New, clave, sal, iter, len(esperado))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(k, esperado) == 1
}

var senuelo = sync.OnceValue(func() string {
	// Clave aleatoria que nadie conoce: Comparar contra ella cuesta lo mismo que
	// contra un usuario real, así que un correo inexistente no responde más rápido.
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	h, err := Hash(base64.RawStdEncoding.EncodeToString(b))
	if err != nil {
		panic("auth: no se pudo crear el hash señuelo: " + err.Error())
	}
	return h
})

// HashSenuelo devuelve un hash válido que no corresponde a ninguna clave conocida.
// El login lo usa cuando el correo no existe (ADR-0007).
func HashSenuelo() string { return senuelo() }
