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

// Largos máximos del login, en bytes: acotan el trabajo de PBKDF2 y el tamaño
// de la búsqueda antes de tocar la base. SembrarUsuarios exige los mismos.
const (
	MaxBytesCorreo = 254
	MaxBytesClave  = 256
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
	switch {
	case s.op.Firmador == nil:
		responderError(w, http.StatusServiceUnavailable, "el login no está configurado (falta JWT_SECRETO)")
		return
	case s.op.Usuarios == nil:
		responderError(w, http.StatusServiceUnavailable, "el login no está configurado (no hay almacén de usuarios)")
		return
	}
	ip := ipCliente(r)
	liberar, espera := s.limite.Reservar(ip)
	if liberar == nil {
		w.Header().Set("Retry-After", strconv.Itoa(max(1, int(math.Ceil(espera.Seconds())))))
		responderError(w, http.StatusTooManyRequests, "demasiados intentos fallidos; espera antes de volver a probar")
		return
	}
	resultado := intentoNeutro
	defer func() { liberar(resultado) }() // también ante un pánico: el cupo no queda tomado
	var p pedidoLogin
	if !leerJSON(w, r, &p, false) {
		return
	}
	correo := almacen.NormalizarCorreo(p.Correo)
	if correo == "" || p.Clave == "" || len(correo) > MaxBytesCorreo || len(p.Clave) > MaxBytesClave {
		responderError(w, http.StatusBadRequest, fmt.Sprintf("correo y clave son obligatorios (máximo %d y %d bytes)", MaxBytesCorreo, MaxBytesClave))
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
	if !(auth.Comparar(hash, p.Clave) && existe && auth.Rol(u.Rol).Valido()) {
		resultado = intentoFallido
		responderError(w, http.StatusUnauthorized, MensajeCredenciales)
		return
	}
	tok, rec, err := s.op.Firmador.Firmar(u.Correo, auth.Rol(u.Rol))
	if err != nil {
		s.op.Log("api: login: firmar: %v", err)
		responderError(w, http.StatusInternalServerError, "no se pudo emitir el token")
		return
	}
	resultado = intentoExitoso
	responder(w, http.StatusOK, RespuestaLogin{Token: tok, Correo: rec.Sub, Rol: string(rec.Rol), Expira: rec.Exp})
}
