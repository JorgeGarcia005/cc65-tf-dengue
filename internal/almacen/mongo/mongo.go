// Package mongo implementa almacen.Almacen sobre MongoDB con el driver
// oficial (ADR-0003, ADR-0006). Colecciones: modelos, entrenamientos,
// predicciones; los índices se crean al conectar (operación idempotente).
package mongo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	drv "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
)

// BasePorDefecto es la base de datos del sistema.
const BasePorDefecto = "cc65_dengue"

// DiasRegistroPredicciones es el TTL del registro de predicciones.
const DiasRegistroPredicciones = 90

// Almacen guarda en MongoDB. Es seguro para uso concurrente (el cliente del
// driver mantiene su propio pool de conexiones).
type Almacen struct {
	cliente                         *drv.Client
	modelos, entrenamientos, predic *drv.Collection
}

var _ almacen.Almacen = (*Almacen)(nil)

// Conectar abre el cliente, comprueba la conexión y crea los índices.
func Conectar(ctx context.Context, uri, base string) (*Almacen, error) {
	cli, err := drv.Connect(options.Client().ApplyURI(uri).
		SetServerSelectionTimeout(5 * time.Second).SetTimeout(10 * time.Second))
	if err != nil {
		return nil, fmt.Errorf("mongo: %w", limpiar(err, uri))
	}
	if err := cli.Ping(ctx, nil); err != nil {
		_ = cli.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo: sin conexión a %s: %w", sinCredenciales(uri), limpiar(err, uri))
	}
	db := cli.Database(base)
	a := &Almacen{cliente: cli, modelos: db.Collection("modelos"),
		entrenamientos: db.Collection("entrenamientos"), predic: db.Collection("predicciones")}
	if err := a.crearIndices(ctx); err != nil {
		_ = cli.Disconnect(context.Background())
		return nil, err
	}
	return a, nil
}

func (a *Almacen) crearIndices(ctx context.Context) error {
	indices := []struct {
		col    *drv.Collection
		models []drv.IndexModel
	}{
		{a.modelos, []drv.IndexModel{{Keys: bson.D{{Key: "publicado", Value: -1}}}}},
		{a.entrenamientos, []drv.IndexModel{
			{Keys: bson.D{{Key: "inicio", Value: -1}}},
			{Keys: bson.D{{Key: "estado", Value: 1}}},
		}},
		{a.predic, []drv.IndexModel{
			{Keys: bson.D{{Key: "fecha", Value: -1}},
				Options: options.Index().SetExpireAfterSeconds(DiasRegistroPredicciones * 24 * 3600)},
			{Keys: bson.D{{Key: "version_modelo", Value: 1}, {Key: "consulta.departamento", Value: 1}}},
		}},
	}
	for _, i := range indices {
		if _, err := i.col.Indexes().CreateMany(ctx, i.models); err != nil {
			return fmt.Errorf("mongo: índices de %s: %w", i.col.Name(), err)
		}
	}
	return nil
}

func reemplazar(ctx context.Context, col *drv.Collection, id string, doc any) error {
	_, err := col.ReplaceOne(ctx, bson.D{{Key: "_id", Value: id}}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("mongo: %s: %w", col.Name(), err)
	}
	return nil
}

// GuardarModelo inserta o reemplaza el modelo con esa versión.
func (a *Almacen) GuardarModelo(ctx context.Context, m almacen.Modelo) error {
	return reemplazar(ctx, a.modelos, m.Version, m)
}

// ModeloVigente devuelve el de publicado más reciente.
func (a *Almacen) ModeloVigente(ctx context.Context) (almacen.Modelo, bool, error) {
	var m almacen.Modelo
	err := a.modelos.FindOne(ctx, bson.D{}, options.FindOne().SetSort(bson.D{{Key: "publicado", Value: -1}})).Decode(&m)
	if errors.Is(err, drv.ErrNoDocuments) {
		return almacen.Modelo{}, false, nil
	}
	if err != nil {
		return almacen.Modelo{}, false, fmt.Errorf("mongo: modelos: %w", err)
	}
	return m, true, nil
}

// GuardarEntrenamiento inserta o reemplaza por ID.
func (a *Almacen) GuardarEntrenamiento(ctx context.Context, e almacen.Entrenamiento) error {
	return reemplazar(ctx, a.entrenamientos, e.ID, e)
}

// Entrenamiento busca por ID.
func (a *Almacen) Entrenamiento(ctx context.Context, id string) (almacen.Entrenamiento, bool, error) {
	var e almacen.Entrenamiento
	err := a.entrenamientos.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&e)
	if errors.Is(err, drv.ErrNoDocuments) {
		return almacen.Entrenamiento{}, false, nil
	}
	if err != nil {
		return almacen.Entrenamiento{}, false, fmt.Errorf("mongo: entrenamientos: %w", err)
	}
	return e, true, nil
}

// RegistrarPrediccion inserta un documento con _id generado por Mongo.
func (a *Almacen) RegistrarPrediccion(ctx context.Context, r almacen.RegistroPrediccion) error {
	if _, err := a.predic.InsertOne(ctx, r); err != nil {
		return fmt.Errorf("mongo: predicciones: %w", err)
	}
	return nil
}

// Cerrar desconecta el cliente.
func (a *Almacen) Cerrar(ctx context.Context) error { return a.cliente.Disconnect(ctx) }

// Borrar elimina la base (solo para los tests de integración).
func (a *Almacen) Borrar(ctx context.Context) error { return a.modelos.Database().Drop(ctx) }

// credenciales separa la parte usuario:clave@ de una URI mongodb://…
func credenciales(uri string) (esquema, info, resto string, ok bool) {
	esquema, despues, hay := strings.Cut(uri, "://")
	if !hay {
		return "", "", uri, false
	}
	fin := strings.IndexAny(despues, "/?")
	if fin < 0 {
		fin = len(despues)
	}
	at := strings.LastIndex(despues[:fin], "@")
	if at < 0 {
		return esquema, "", despues, false
	}
	return esquema, despues[:at], despues[at:], true
}

// sinCredenciales reemplaza usuario y contraseña de una URI por *** para
// poder mostrarla en logs y errores (PR-40 H8).
func sinCredenciales(uri string) string {
	esquema, _, resto, ok := credenciales(uri)
	if !ok {
		return uri
	}
	return esquema + "://***" + resto
}

// limpiar quita de un error del driver la URI y cualquier rastro del usuario
// o la contraseña.
func limpiar(err error, uri string) error {
	_, info, _, ok := credenciales(uri)
	if !ok {
		return err
	}
	msg := strings.ReplaceAll(err.Error(), uri, sinCredenciales(uri))
	usuario, clave, _ := strings.Cut(info, ":")
	for _, s := range []string{clave, usuario} {
		if s != "" {
			msg = strings.ReplaceAll(msg, s, "***")
		}
	}
	return errors.New(msg)
}
