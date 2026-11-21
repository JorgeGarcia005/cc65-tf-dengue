package contrato

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// ProxyLento abre un proxy TCP hacia destino que demora `demora` cada
// respuesta del servidor. Simula un Redis lento para probar que los plazos de
// la petición se respetan con el cliente real (PR-40 ronda 2, H5). Devuelve la
// dirección del proxy; se cierra al terminar el test.
func ProxyLento(t *testing.T, destino string, demora time.Duration) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var conexiones []net.Conn
	cerrar := func(cs ...net.Conn) {
		mu.Lock()
		conexiones = append(conexiones, cs...)
		mu.Unlock()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			cli, err := ln.Accept()
			if err != nil {
				return
			}
			srv, err := net.Dial("tcp", destino)
			if err != nil {
				cli.Close()
				continue
			}
			cerrar(cli, srv)
			wg.Add(2)
			go func() { defer wg.Done(); _, _ = io.Copy(srv, cli); srv.Close() }()
			go func() {
				defer wg.Done()
				buf := make([]byte, 32<<10)
				for {
					n, err := srv.Read(buf)
					if n > 0 {
						time.Sleep(demora)
						if _, errW := cli.Write(buf[:n]); errW != nil {
							break
						}
					}
					if err != nil {
						break
					}
				}
				cli.Close()
			}()
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		mu.Lock()
		for _, c := range conexiones {
			c.Close()
		}
		mu.Unlock()
		wg.Wait()
	})
	return ln.Addr().String()
}
