package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func escribirPrueba(t *testing.T, ruta, contenido string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestRegionArchivoCompleto: el archivo entero, con su ruta en un comentario
// del lenguaje y el bloque marcado con el lenguaje correcto.
func TestRegionArchivoCompleto(t *testing.T) {
	t.Chdir(t.TempDir())
	escribirPrueba(t, "deploy/Dockerfile", "FROM golang:1.24 AS compilacion\nRUN go build ./...\n")
	got, err := generarRegion("archivo:deploy/Dockerfile")
	want := "```dockerfile\n# deploy/Dockerfile\nFROM golang:1.24 AS compilacion\nRUN go build ./...\n```\n"
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

// TestRegionArchivoFragmento: solo lo que hay entre los marcadores
// informe:NOMBRE y /informe:NOMBRE, sin los marcadores (también los de otros
// fragmentos anidados) y con la sangría común quitada. Un nombre que es prefijo
// de otro no lo confunde.
func TestRegionArchivoFragmento(t *testing.T) {
	t.Chdir(t.TempDir())
	escribirPrueba(t, "deploy/compose.yml", strings.Join([]string{
		"services:",
		"  # informe:nodo",
		"  node:",
		"    image: cc65-tf",
		"  # informe:nodos",
		"    replicas: 3",
		"  # /informe:nodos",
		"  # /informe:nodo",
		"  web: {}",
	}, "\n"))
	got, err := generarRegion("archivo:deploy/compose.yml#nodo")
	want := "```yaml\n# deploy/compose.yml\nnode:\n  image: cc65-tf\n  replicas: 3\n```\n"
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
	escribirPrueba(t, "spin/barrera.pml", "/* informe:canales */\nchan tarea[N] = [1] of { byte };\n/* /informe:canales */\n")
	got, err = generarRegion("archivo:spin/barrera.pml#canales")
	if err != nil || got != "```promela\n/* spin/barrera.pml */\nchan tarea[N] = [1] of { byte };\n```\n" {
		t.Fatalf("promela: %q, %v", got, err)
	}
}

func TestRegionArchivoErrores(t *testing.T) {
	t.Chdir(t.TempDir())
	escribirPrueba(t, "a.sh", "# informe:abierto\necho hola\n")
	for _, nombre := range []string{"archivo:no-existe.sh", "archivo:a.sh#otro", "archivo:a.sh#abierto"} {
		if _, err := generarRegion(nombre); err == nil {
			t.Errorf("%s: se esperaba error", nombre)
		}
	}
}

// TestValoresEnLinea: <!--v:EXPR-->…<!--/v--> se reescribe con el valor de
// EXPR sobre los datos, con las mismas funciones que las plantillas.
func TestValoresEnLinea(t *testing.T) {
	datos := struct {
		N   int
		AUC float64
	}{1070123, 0.678912}
	in := "Hay <!--v:miles .N-->?<!--/v--> casos y un AUC de <!--v:cuatro .AUC--><!--/v-->.\n"
	got, err := expandirValores([]byte(in), datos)
	want := "Hay <!--v:miles .N-->" + miles(1070123) + "<!--/v--> casos y un AUC de <!--v:cuatro .AUC-->0,6789<!--/v-->.\n"
	if err != nil || string(got) != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
	if _, err := expandirValores([]byte("<!--v:.NoExiste--><!--/v-->"), datos); err == nil {
		t.Error("un campo que no existe debía dar error")
	}
	if _, err := expandirValores([]byte("<!--v:.N--><!--/v-->"), nil); err == nil {
		t.Error("sin datos debía dar error")
	}
}
