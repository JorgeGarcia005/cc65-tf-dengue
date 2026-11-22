package api

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Límite de intentos fallidos de login por IP (ADR-0007): ventana deslizante.
const (
	MaxFallosLogin = 5
	VentanaFallos  = time.Minute
	// Con muchas IPs, las vencidas se podan como mucho una vez cada media
	// ventana (no en cada fallo) y el mapa nunca pasa de maxIPsLimitador.
	maxIPsAntesPoda = 1024
	maxIPsLimitador = 8192
)

// Resultado de un intento de login, para liberar su reserva.
type resultadoIntento int

const (
	intentoFallido resultadoIntento = iota // credenciales incorrectas: cuenta como fallo
	intentoExitoso                         // borra los fallos de la IP
	intentoNeutro                          // error interno o entrada inválida: no cuenta
)

// limitador cuenta, por IP, los fallos de la última ventana y los intentos en
// curso. Reservar comprueba y reserva en la misma sección crítica: como un
// intento dura cientos de milisegundos (PBKDF2), si solo se contaran los fallos
// ya cerrados, una ráfaga concurrente pasaría entera antes del primer fallo
// (TOCTOU). Así nunca hay más de MaxFallosLogin intentos fallidos o en curso.
type limitador struct {
	mu         sync.Mutex
	fallos     map[string][]time.Time // en orden de llegada
	enCurso    map[string]int
	max        int
	ventana    time.Duration
	podarDesde int // tamaño del mapa a partir del cual se poda
	maxIPs     int // tope duro del mapa de fallos
	ultimaPoda time.Time
	ahora      func() time.Time
}

func nuevoLimitador() *limitador {
	return &limitador{fallos: map[string][]time.Time{}, enCurso: map[string]int{}, max: MaxFallosLogin,
		ventana: VentanaFallos, podarDesde: maxIPsAntesPoda, maxIPs: maxIPsLimitador, ahora: time.Now}
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

// Reservar decide si la IP puede intentar ahora. Si puede, reserva un cupo y
// devuelve la función que lo libera con el resultado del intento (llamarla
// exactamente una vez). Si no, devuelve cuánto esperar: hasta que salga de la
// ventana el fallo que hace sobrar, o 1 s si los cupos están ocupados por
// intentos que aún no terminan.
func (l *limitador) Reservar(ip string) (liberar func(resultadoIntento), espera time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.ahora()
	f := l.vigentes(ip, t)
	if len(f)+l.enCurso[ip] >= l.max {
		if len(f) >= l.max {
			return nil, f[len(f)-l.max].Add(l.ventana).Sub(t)
		}
		return nil, time.Second
	}
	l.enCurso[ip]++
	var una sync.Once
	return func(r resultadoIntento) { una.Do(func() { l.liberar(ip, r) }) }, 0
}

func (l *limitador) liberar(ip string, r resultadoIntento) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.enCurso[ip]--; l.enCurso[ip] <= 0 {
		delete(l.enCurso, ip)
	}
	switch r {
	case intentoExitoso:
		delete(l.fallos, ip)
	case intentoFallido:
		t := l.ahora()
		l.podar(t, ip)
		l.fallos[ip] = append(l.vigentes(ip, t), t)
	}
}

// podar mantiene acotado el mapa de fallos. Con muchas IPs, barre las vencidas
// a lo sumo una vez cada media ventana; si aun así está lleno y la IP es
// nueva, desaloja una entrada cualquiera (sus fallos se pierden: se prefiere
// memoria acotada a recordar a todas las IPs). Requiere l.mu.
func (l *limitador) podar(t time.Time, ip string) {
	if len(l.fallos) >= l.podarDesde && t.Sub(l.ultimaPoda) >= l.ventana/2 {
		l.ultimaPoda = t
		for otra := range l.fallos {
			l.vigentes(otra, t)
		}
	}
	if _, existe := l.fallos[ip]; !existe && len(l.fallos) >= l.maxIPs {
		for otra := range l.fallos {
			delete(l.fallos, otra)
			break
		}
	}
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
