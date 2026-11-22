// Package mongo implementa almacen.Almacen sobre MongoDB con el driver
// oficial (ADR-0003, ADR-0006). Colecciones: modelos, entrenamientos,
// predicciones; los índices se crean al conectar (operación idempotente).
package mongo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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
	usuarios                        *drv.Collection
}

var (
	_ almacen.Almacen  = (*Almacen)(nil)
	_ almacen.Usuarios = (*Almacen)(nil)
)

// Conectar abre el cliente, comprueba la conexión y crea los índices.
func Conectar(ctx context.Context, uri, base string) (*Almacen, error) {
	cli, err := drv.Connect(options.Client().ApplyURI(uri).
		SetServerSelectionTimeout(5 * time.Second).SetTimeout(10 * time.Second))
	if err != nil {
		return nil, fmt.Errorf("mongo: %w", limpiar(err, uri))
	}
	if err := cli.Ping(ctx, nil); err != nil {
		_ = cli.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo: sin conexión a %s: %w", destino(uri), limpiar(err, uri))
	}
	db := cli.Database(base)
	a := &Almacen{cliente: cli, modelos: db.Collection("modelos"),
		entrenamientos: db.Collection("entrenamientos"), predic: db.Collection("predicciones"),
		usuarios: db.Collection("usuarios")}
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
		{a.modelos, []drv.IndexModel{
			{Keys: bson.D{{Key: "publicado", Value: -1}}},
			{Keys: bson.D{{Key: "entrenamientos.id", Value: 1}}},
		}},
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

// GuardarModelo es un solo UpdateOne con upsert, atómico sobre el documento:
// inserta los campos del modelo si la versión es nueva ($setOnInsert), agrega
// los resúmenes de entrenamiento ($push) y adelanta publicado ($max).
func (a *Almacen) GuardarModelo(ctx context.Context, m almacen.Modelo) error {
	resumenes := m.Entrenamientos
	if resumenes == nil {
		resumenes = []almacen.ResumenEntrenamiento{}
	}
	cambio := bson.D{
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "pesos", Value: m.Pesos}, {Key: "esquema", Value: m.Esquema}, {Key: "umbral", Value: m.Umbral},
			{Key: "ajuste_prior", Value: m.AjustePrior}, {Key: "generado", Value: m.Generado},
		}},
		{Key: "$max", Value: bson.D{{Key: "publicado", Value: m.Publicado}}},
		{Key: "$push", Value: bson.D{{Key: "entrenamientos", Value: bson.D{{Key: "$each", Value: resumenes}}}}},
	}
	_, err := a.modelos.UpdateOne(ctx, bson.D{{Key: "_id", Value: m.Version}}, cambio, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("mongo: modelos: %w", err)
	}
	return nil
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

// GuardarUsuario crea o reemplaza la cuenta. El _id (correo normalizado) es
// único por definición en MongoDB: no hace falta otro índice.
func (a *Almacen) GuardarUsuario(ctx context.Context, u almacen.Usuario) error {
	u, err := almacen.PrepararUsuario(u)
	if err != nil {
		return err
	}
	return reemplazar(ctx, a.usuarios, u.Correo, u)
}

// Usuario busca por correo normalizado.
func (a *Almacen) Usuario(ctx context.Context, correo string) (almacen.Usuario, bool, error) {
	var u almacen.Usuario
	err := a.usuarios.FindOne(ctx, bson.D{{Key: "_id", Value: almacen.NormalizarCorreo(correo)}}).Decode(&u)
	if errors.Is(err, drv.ErrNoDocuments) {
		return almacen.Usuario{}, false, nil
	}
	if err != nil {
		return almacen.Usuario{}, false, fmt.Errorf("mongo: usuarios: %w", err)
	}
	return u, true, nil
}

// Cerrar desconecta el cliente.
func (a *Almacen) Cerrar(ctx context.Context) error { return a.cliente.Disconnect(ctx) }

// Borrar elimina la base (solo para los tests de integración).
func (a *Almacen) Borrar(ctx context.Context) error { return a.modelos.Database().Drop(ctx) }

// partes separa una URI mongodb:// en esquema, userinfo, hosts y consulta.
func partes(uri string) (esquema, info, hosts, consulta string, ok bool) {
	esquema, resto, ok := strings.Cut(uri, "://")
	if !ok {
		return "", "", "", "", false
	}
	autoridad, despues := resto, ""
	if i := strings.IndexAny(resto, "/?"); i >= 0 {
		autoridad, despues = resto[:i], resto[i:]
	}
	if at := strings.LastIndex(autoridad, "@"); at >= 0 {
		info, autoridad = autoridad[:at], autoridad[at+1:]
	}
	if _, q, hay := strings.Cut(despues, "?"); hay {
		consulta = q
	}
	return esquema, info, autoridad, consulta, true
}

// destino es lo único de la URI que se muestra en un diagnóstico: esquema y
// hosts, sin usuario, contraseña, base ni opciones (PR-40 H8, rondas 1 y 2).
func destino(uri string) string {
	esquema, _, hosts, _, ok := partes(uri)
	if !ok {
		return "(URI no válida)"
	}
	return esquema + "://" + hosts
}

// opcionesSensibles se ocultan siempre, aunque su valor sea corto.
var opcionesSensibles = map[string]bool{
	"tlscertificatekeyfilepassword":   true,
	"sslclientcertificatekeypassword": true,
	"authmechanismproperties":         true,
	"password":                        true,
}

// limpiar quita de un error del driver la URI completa y los valores que
// pueden ser secretos: usuario, contraseña y los valores de la consulta (los
// sensibles siempre; los demás si tienen al menos 4 caracteres, para no
// desfigurar direcciones o números del mensaje).
func limpiar(err error, uri string) error {
	_, info, _, consulta, ok := partes(uri)
	msg := err.Error()
	if !ok {
		return err
	}
	msg = strings.ReplaceAll(msg, uri, destino(uri))
	var secretos []string
	if info != "" {
		usuario, clave, _ := strings.Cut(info, ":")
		secretos = append(secretos, clave, usuario)
	}
	for _, par := range strings.Split(consulta, "&") {
		clave, valor, _ := strings.Cut(par, "=")
		if valor == "" || (len(valor) < 4 && !opcionesSensibles[strings.ToLower(clave)]) {
			continue
		}
		secretos = append(secretos, valor)
		if dec, err := url.QueryUnescape(valor); err == nil && dec != valor {
			secretos = append(secretos, dec)
		}
		// authMechanismProperties=CLAVE:valor,…: oculta también cada valor.
		for _, kv := range strings.Split(valor, ",") {
			if _, v, hay := strings.Cut(kv, ":"); hay && v != "" {
				secretos = append(secretos, v)
			}
		}
	}
	for _, s := range secretos {
		if s != "" {
			msg = strings.ReplaceAll(msg, s, "***")
		}
	}
	return errors.New(msg)
}

// ModeloDeEntrenamiento busca el modelo publicado por ese entrenamiento.
func (a *Almacen) ModeloDeEntrenamiento(ctx context.Context, id string) (almacen.Modelo, bool, error) {
	var m almacen.Modelo
	err := a.modelos.FindOne(ctx, bson.D{{Key: "entrenamientos.id", Value: id}}).Decode(&m)
	if errors.Is(err, drv.ErrNoDocuments) {
		return almacen.Modelo{}, false, nil
	}
	if err != nil {
		return almacen.Modelo{}, false, fmt.Errorf("mongo: modelos: %w", err)
	}
	return m, true, nil
}

// CrearEntrenamiento inserta; un _id repetido devuelve almacen.ErrDuplicado.
func (a *Almacen) CrearEntrenamiento(ctx context.Context, e almacen.Entrenamiento) error {
	_, err := a.entrenamientos.InsertOne(ctx, e)
	if drv.IsDuplicateKeyError(err) {
		return almacen.ErrDuplicado
	}
	if err != nil {
		return fmt.Errorf("mongo: entrenamientos: %w", err)
	}
	return nil
}

// EntrenamientosEnCurso lista los que tienen estado en_curso.
func (a *Almacen) EntrenamientosEnCurso(ctx context.Context) ([]almacen.Entrenamiento, error) {
	cur, err := a.entrenamientos.Find(ctx, bson.D{{Key: "estado", Value: almacen.EnCurso}})
	if err != nil {
		return nil, fmt.Errorf("mongo: entrenamientos: %w", err)
	}
	var en []almacen.Entrenamiento
	if err := cur.All(ctx, &en); err != nil {
		return nil, fmt.Errorf("mongo: entrenamientos: %w", err)
	}
	return en, nil
}
