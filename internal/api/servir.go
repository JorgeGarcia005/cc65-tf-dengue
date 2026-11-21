package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// NuevoHTTP crea el http.Server con plazos: ReadHeaderTimeout evita que un
// cliente lento retenga conexiones (Slowloris) y los demás acotan cada petición.
func NuevoHTTP(h http.Handler) *http.Server {
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// Servir atiende en ln hasta que ctx se cancele (p. ej. por SIGTERM con
// signal.NotifyContext) y entonces cierra ordenadamente: deja de aceptar
// conexiones y espera hasta plazo a que terminen las peticiones en curso.
func Servir(ctx context.Context, srv *http.Server, ln net.Listener, plazo time.Duration) error {
	errServe := make(chan error, 1)
	go func() { errServe <- srv.Serve(ln) }()
	select {
	case err := <-errServe:
		return err // falló antes de pedir el cierre
	case <-ctx.Done():
	}
	ctxCierre, cancelar := context.WithTimeout(context.Background(), plazo)
	defer cancelar()
	errCierre := srv.Shutdown(ctxCierre)
	if err := <-errServe; !errors.Is(err, http.ErrServerClosed) {
		return errors.Join(err, errCierre)
	}
	return errCierre
}
