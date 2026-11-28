//go:build linux || darwin || freebsd || netbsd || openbsd

package cluster

import "syscall"

// cpuProceso devuelve el tiempo de CPU (usuario + sistema) que lleva el
// proceso, en µs, y true. Con dos lecturas alrededor del cálculo, el nodo sabe
// cuánta CPU usó su worker pool (#91). Los nodos corren en contenedores Linux.
func cpuProceso() (int64, bool) {
	var r syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &r); err != nil {
		return 0, false
	}
	return r.Utime.Nano()/1e3 + r.Stime.Nano()/1e3, true
}
