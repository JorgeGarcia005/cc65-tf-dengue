package api

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Límite de intentos fallidos de login por IP (ADR-0007): ventana deslizante.
const (
	MaxFallosLogin  = 5
	VentanaFallos   = time.Minute
	maxIPsAntesPoda = 1024
)

// limitador cuenta los intentos fallidos por IP en la última ventana. Con
// MaxFallosLogin fallos dentro de la ventana, el siguiente intento se rechaza
// hasta que el fallo más viejo salga de ella.
type limitador struct {
	mu      sync.Mutex
	fallos  map[string][]time.Time // en orden de llegada
	max     int
	ventana time.Duration
	ahora   func() time.Time
}

func nuevoLimitador() *limitador {
	return &limitador{fallos: map[string][]time.Time{}, max: MaxFallosLogin, ventana: VentanaFallos, ahora: time.Now}
}

// vigentes quita los fallos fuera de la ventana; borra la IP si no queda ninguno.
// Requiere l.mu.
func (l *limitador) vigentes(ip string, t time.Time) []time.Time {
	f := l.fallos[ip]
	i := 0
	for i < len(f) && !f[i].After(t.Add(-l.ventana)) {
		i++
	}
	f = f[i:]
	if len(f) == 0 {
		delete(l.fallos, ip)
	} else {
		l.fallos[ip] = f
	}
	return f
}

// Bloqueado dice si la IP agotó sus intentos y cuánto falta para el próximo.
func (l *limitador) Bloqueado(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.ahora()
	f := l.vigentes(ip, t)
	if len(f) < l.max {
		return false, 0
	}
	return true, f[len(f)-l.max].Add(l.ventana).Sub(t)
}

// Fallo anota un intento fallido. Si hay muchas IPs, poda las que ya no tienen
// fallos vigentes para que la memoria no crezca sin límite.
func (l *limitador) Fallo(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.ahora()
	if len(l.fallos) >= maxIPsAntesPoda {
		for otra := range l.fallos {
			l.vigentes(otra, t)
		}
	}
	l.fallos[ip] = append(l.vigentes(ip, t), t)
}

// Exito borra los fallos de la IP.
func (l *limitador) Exito(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fallos, ip)
}

// ipCliente toma el host de RemoteAddr. No se mira X-Forwarded-For: sin un
// proxy de confianza delante, cualquiera lo falsifica (ADR-0007).
func ipCliente(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
