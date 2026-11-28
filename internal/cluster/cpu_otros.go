//go:build !(linux || darwin || freebsd || netbsd || openbsd)

package cluster

// cpuProceso no está disponible fuera de Unix (p. ej. un nodo en Windows sin
// Docker): el nodo no informa su CPU y el panel lo muestra como «—».
func cpuProceso() (int64, bool) { return 0, false }
