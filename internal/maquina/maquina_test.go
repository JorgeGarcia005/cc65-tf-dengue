package maquina

import "testing"

func TestCPUDeProcCPUInfo(t *testing.T) {
	s := "processor\t: 0\nvendor_id\t: GenuineIntel\nmodel name\t: Intel(R) Xeon(R) Processor @ 2.10GHz\nflags\t: fpu\n"
	if got := cpuDeProcCPUInfo(s); got != "Intel(R) Xeon(R) Processor @ 2.10GHz" {
		t.Fatalf("cpu = %q", got)
	}
	if got := cpuDeProcCPUInfo("processor\t: 0\n"); got != "" {
		t.Fatalf("sin model name: cpu = %q", got)
	}
}

func TestCPUDeRegQuery(t *testing.T) {
	s := "\r\nHKEY_LOCAL_MACHINE\\HARDWARE\\DESCRIPTION\\System\\CentralProcessor\\0\r\n" +
		"    ProcessorNameString    REG_SZ    AMD Ryzen 7 5800X 8-Core Processor             \r\n\r\n"
	if got := cpuDeRegQuery(s); got != "AMD Ryzen 7 5800X 8-Core Processor" {
		t.Fatalf("cpu = %q", got)
	}
	if got := cpuDeRegQuery("ERROR: no se encontró\r\n"); got != "" {
		t.Fatalf("sin valor: cpu = %q", got)
	}
}

func TestHostEsNombreDeArchivo(t *testing.T) {
	for _, r := range Host() {
		if !(r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			t.Fatalf("Host() = %q contiene %q", Host(), r)
		}
	}
}
