// Command usuarios crea o actualiza en MongoDB las cuentas de la SPA (ADR-0007)
// a partir de variables de entorno; ninguna clave queda en el repositorio:
//
//	VIGIA_ADMIN_CORREO, VIGIA_ADMIN_CLAVE         cuenta con rol admin
//	VIGIA_ANALISTA_CORREO, VIGIA_ANALISTA_CLAVE   cuenta con rol analista
//
// Las claves deben tener al menos 12 caracteres. El coordinador hace lo mismo
// al arrancar en modo servicio si encuentra estas variables.
//
//	go run ./cmd/usuarios -mongo mongodb://localhost:27017
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/api"
)

func main() {
	uri := flag.String("mongo", envO("MONGO_URI", "mongodb://localhost:27017"), "URI de MongoDB (por defecto $MONGO_URI)")
	base := flag.String("base", mongo.BasePorDefecto, "base de datos")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Uso: usuarios [-mongo URI] [-base NOMBRE]\n\n"+
			"Lee VIGIA_ADMIN_CORREO/VIGIA_ADMIN_CLAVE y VIGIA_ANALISTA_CORREO/VIGIA_ANALISTA_CLAVE\n"+
			"y crea o actualiza esas cuentas (claves de ≥ %d caracteres, guardadas con PBKDF2).\n\n", api.MinLargoClave)
		flag.PrintDefaults()
	}
	flag.Parse()
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	a, err := mongo.Conectar(ctx, *uri, *base)
	if err != nil {
		log.Fatal(err)
	}
	defer a.Cerrar(context.Background())
	if err := ejecutar(ctx, a, os.Getenv, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func ejecutar(ctx context.Context, repo almacen.Usuarios, getenv func(string) string, salida io.Writer) error {
	sembrados, err := api.SembrarUsuarios(ctx, repo, getenv)
	if err != nil {
		return err
	}
	if len(sembrados) == 0 {
		return fmt.Errorf("no hay cuentas que crear: define VIGIA_ADMIN_* o VIGIA_ANALISTA_* (ver -h)")
	}
	for _, s := range sembrados {
		fmt.Fprintln(salida, "usuario listo:", s)
	}
	return nil
}

func envO(clave, defecto string) string {
	if v := os.Getenv(clave); v != "" {
		return v
	}
	return defecto
}
