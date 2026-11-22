package auth

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestClaveIdaYVuelta(t *testing.T) {
	h, err := hashCon("una clave larga", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if !Comparar(h, "una clave larga") {
		t.Fatal("la clave correcta no verifica")
	}
	if Comparar(h, "una clave larg") || Comparar(h, "") {
		t.Fatal("una clave distinta verificó")
	}
}

func TestClaveSalDistinta(t *testing.T) {
	a, _ := hashCon("misma", 1000)
	b, _ := hashCon("misma", 1000)
	if a == b {
		t.Fatal("dos hashes de la misma clave son iguales: la sal no es aleatoria")
	}
}

func TestClaveFormatoPorDefecto(t *testing.T) {
	h, err := Hash("clave por defecto")
	if err != nil {
		t.Fatal(err)
	}
	partes := strings.Split(h, "$")
	if len(partes) != 4 || partes[0] != "pbkdf2-sha256" || partes[1] != "600000" {
		t.Fatalf("formato inesperado: %q", h)
	}
	sal, err1 := b64Clave.DecodeString(partes[2])
	clave, err2 := b64Clave.DecodeString(partes[3])
	if err1 != nil || err2 != nil || len(sal) != 16 || len(clave) != 32 {
		t.Fatalf("sal de %d bytes y clave de %d bytes, quería 16 y 32", len(sal), len(clave))
	}
	if !Comparar(h, "clave por defecto") {
		t.Fatal("no verifica con las iteraciones por defecto")
	}
}

func TestClaveGuardadoInvalido(t *testing.T) {
	for _, g := range []string{"", "texto", "pbkdf2-sha256$x$AA$AA", "bcrypt$10$AA$AA",
		"pbkdf2-sha256$10$@@$AA", "pbkdf2-sha256$0$AAAA$AAAA", "pbkdf2-sha256$99999999$AAAA$AAAA"} {
		if Comparar(g, "x") {
			t.Fatalf("aceptó un hash inválido: %q", g)
		}
	}
	// Hashes reales manipulados: clave truncada a 1 byte (iter 1), clave o sal de
	// otro largo y saltos de línea; ninguno debe verificar.
	sal16 := make([]byte, 16)
	truncada, _ := pbkdf2Hex("x", sal16, 1, 1)
	larga, _ := pbkdf2Hex("x", sal16, 1, 320)
	salCorta, _ := pbkdf2Hex("x", make([]byte, 4), 1, 32)
	buena, _ := pbkdf2Hex("x", sal16, 1, 32)
	if !Comparar(buena, "x") {
		t.Fatal("el hash bien formado de control no verifica")
	}
	for nombre, g := range map[string]string{"truncada": truncada, "larga": larga, "sal corta": salCorta,
		"salto de línea": buena[:20] + "\n" + buena[20:]} {
		if Comparar(g, "x") {
			t.Fatalf("aceptó un hash con %s: %q", nombre, g)
		}
	}
}

// pbkdf2Hex arma un hash con largos arbitrarios, como lo haría un atacante.
func pbkdf2Hex(clave string, sal []byte, iter, largo int) (string, error) {
	k, err := pbkdf2.Key(sha256.New, clave, sal, iter, largo)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iter, b64Clave.EncodeToString(sal), b64Clave.EncodeToString(k)), err
}

func TestClaveSenueloNoVerifica(t *testing.T) {
	if Comparar(HashSenuelo(), "") || Comparar(HashSenuelo(), "senuelo") {
		t.Fatal("el señuelo no debe verificar ninguna clave")
	}
}
