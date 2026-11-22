package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/almacen"
	"github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth"
)

// Largos máximos del login: acotan el trabajo de PBKDF2 y el tamaño de la
// búsqueda antes de tocar la base.
const (
	maxLargoCorreo = 254
	maxLargoClave  = 256
)

// MensajeCredenciales es la única respuesta a un login fallido: no revela si
// el correo existe (ADR-0007).
const MensajeCredenciales = "correo o clave incorrectos"

type pedidoLogin struct {
	Correo string `json:"correo"`
	Clave  string `json:"clave"`
}

// RespuestaLogin es el contrato que espera la SPA (#62).
type RespuestaLogin struct {
	Token  string `json:"token"`
	Correo string `json:"correo"`
	Rol    string `json:"rol"`
	Expira int64  `json:"expira"` // segundos Unix
}

func (s *Servidor) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if s.op.Firmador == nil || s.op.Usuarios == nil {
		responderError(w, http.StatusServiceUnavailable, "el login no está configurado (falta JWT_SECRETO)")
		return
	}
	ip := ipCliente(r)
	if bloqueado, espera := s.limite.Bloqueado(ip); bloqueado {
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(espera.Seconds()))))
		responderError(w, http.StatusTooManyRequests, "demasiados intentos fallidos; espera antes de volver a probar")
		return
	}
	var p pedidoLogin
	if !leerJSON(w, r, &p, false) {
		return
	}
	correo := almacen.NormalizarCorreo(p.Correo)
	if correo == "" || p.Clave == "" || len(correo) > maxLargoCorreo || len(p.Clave) > maxLargoClave {
		responderError(w, http.StatusBadRequest, fmt.Sprintf("correo y clave son obligatorios (máximo %d y %d caracteres)", maxLargoCorreo, maxLargoClave))
		return
	}
	ctx, cancelar := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancelar()
	u, existe, err := s.op.Usuarios.Usuario(ctx, correo)
	if err != nil {
		s.op.Log("api: login: %v", err)
		responderError(w, http.StatusInternalServerError, "no se pudo verificar el usuario")
		return
	}
	// Con o sin usuario se calcula un PBKDF2 completo: el tiempo de respuesta no
	// revela si el correo existe.
	hash := auth.HashSenuelo()
	if existe {
		hash = u.Hash
	}
	valida := auth.Comparar(hash, p.Clave) && existe && auth.Rol(u.Rol).Valido()
	if !valida {
		s.limite.Fallo(ip)
		responderError(w, http.StatusUnauthorized, MensajeCredenciales)
		return
	}
	s.limite.Exito(ip)
	tok, rec, err := s.op.Firmador.Firmar(u.Correo, auth.Rol(u.Rol))
	if err != nil {
		s.op.Log("api: login: firmar: %v", err)
		responderError(w, http.StatusInternalServerError, "no se pudo emitir el token")
		return
	}
	responder(w, http.StatusOK, RespuestaLogin{Token: tok, Correo: rec.Sub, Rol: string(rec.Rol), Expira: rec.Exp})
}
