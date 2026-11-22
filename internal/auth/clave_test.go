package auth

import (
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
}

func TestClaveSenueloNoVerifica(t *testing.T) {
	if Comparar(HashSenuelo(), "") || Comparar(HashSenuelo(), "senuelo") {
		t.Fatal("el señuelo no debe verificar ninguna clave")
	}
}
