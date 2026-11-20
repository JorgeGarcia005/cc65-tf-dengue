package main

import (
	"context"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/mongo"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen/redis"
)

// plazoConexionBD acota la conexión inicial a MongoDB y Redis.
const plazoConexionBD = 30 * time.Second

// conectarBD abre MongoDB (si uriMongo no está vacío) y Redis (si dirRedis no
// está vacío) para el modo servicio. cerrar libera lo que se abrió y se llama
// después de Servidor.Cerrar, cuando la API ya no escribe.
func conectarBD(uriMongo, base, dirRedis string, logf func(string, ...any)) (bd baseDatos, cerrar func(), err error) {
	ctx, cancelar := context.WithTimeout(context.Background(), plazoConexionBD)
	defer cancelar()
	var cierres []func()
	cerrar = func() {
		for i := len(cierres) - 1; i >= 0; i-- {
			cierres[i]()
		}
	}
	if uriMongo != "" {
		a, err := mongo.Conectar(ctx, uriMongo, base)
		if err != nil {
			return baseDatos{}, func() {}, err
		}
		bd.Almacen = a
		cierres = append(cierres, func() {
			ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelar()
			if err := a.Cerrar(ctx); err != nil {
				logf("mongo: cierre: %v", err)
			}
		})
		logf("MongoDB conectado (base %s): modelos, entrenamientos y predicciones persistentes", base)
	} else {
		logf("sin -mongo: la API guarda modelos y entrenamientos en memoria")
	}
	if dirRedis != "" {
		c, err := redis.Conectar(ctx, dirRedis)
		if err != nil {
			cerrar()
			return baseDatos{}, func() {}, err
		}
		bd.Cache = c
		cierres = append(cierres, func() {
			if err := c.Cerrar(); err != nil {
				logf("redis: cierre: %v", err)
			}
		})
		logf("Redis conectado en %s: caché de predicciones", dirRedis)
	} else {
		logf("sin -redis: la API no usa caché de predicciones")
	}
	return bd, cerrar, nil
}
