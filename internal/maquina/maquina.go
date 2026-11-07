// Package maquina describe el equipo donde corre un benchmark, para que cada
// reporte en reports/*.json diga en qué CPU se midió.
package maquina

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Describir devuelve SO, arquitectura, versión de Go, CPUs y, si se puede
// averiguar, el modelo de CPU (Linux: /proc/cpuinfo; Windows: el registro).
func Describir() map[string]any {
	m := map[string]any{
		"so": runtime.GOOS, "arquitectura": runtime.GOARCH, "go": runtime.Version(),
		"num_cpu": runtime.NumCPU(), "gomaxprocs": runtime.GOMAXPROCS(0),
	}
	if cpu := modeloCPU(); cpu != "" {
		m["cpu"] = cpu
	}
	return m
}

// Host devuelve el nombre del equipo en minúsculas y apto para un nombre de archivo.
func Host() string {
	h, _ := os.Hostname()
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, strings.ToLower(h))
}

func modeloCPU() string {
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		return cpuDeProcCPUInfo(string(b))
	}
	if runtime.GOOS == "windows" {
		out, err := exec.Command("reg", "query",
			`HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0`, "/v", "ProcessorNameString").Output()
		if err == nil {
			return cpuDeRegQuery(string(out))
		}
	}
	return ""
}

func cpuDeProcCPUInfo(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.HasPrefix(l, "model name") {
			if _, v, ok := strings.Cut(l, ":"); ok {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func cpuDeRegQuery(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if _, v, ok := strings.Cut(l, "REG_SZ"); ok && strings.Contains(l, "ProcessorNameString") {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
