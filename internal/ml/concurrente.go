package ml

import "sync"

// Pool calcula parciales por bloque con un worker pool persistente.
//
// Patrón y sincronización:
//
//   - Los workers se crean una vez (NuevoPool) y viven hasta Cerrar: no se paga
//     el costo de crear goroutines en cada época.
//   - Un canal sin búfer reparte índices de bloque (fan-out). Cada bloque lo
//     procesa exactamente un worker.
//   - Cada worker acumula el bloque en su propio búfer privado (local) y al
//     terminar lo copia a salida[k], el slot del bloque k. Ningún worker
//     escribe en memoria que otro lea o escriba durante la pasada, así que no
//     hace falta mutex sobre los acumuladores. Acumular en el búfer local y no
//     directo en el slot evita el false sharing: los slots de bloques vecinos
//     comparten líneas de caché y escribirlos fila a fila desde dos núcleos
//     hacía al pool más lento que la versión secuencial.
//   - Un sync.WaitGroup es la barrera de fin de pasada: se espera a que los B
//     bloques terminen antes de devolver o reducir.
//   - La reducción la hace un solo goroutine, en orden de índice de bloque
//     (ReducirEnOrden). Por eso el resultado es idéntico bit a bit al de
//     Secuencial, para cualquier número de workers.
//
// El pool no queda atado a un dataset (ADR-0005): Parciales recibe el dataset
// de cada pasada, así un nodo del clúster atiende varios shards con los mismos
// workers. Gradiente usa el dataset base que recibió NuevoPool.
//
// Relaciones happens-before que hacen correcto el acceso a memoria:
//   - p.ds, p.w y p.salida se escriben antes de enviar el primer índice por el
//     canal; cada worker los lee después de recibir, por lo que ve la pasada
//     completa y nunca la anterior.
//   - wg.Done() de cada worker ocurre después de escribir su slot; wg.Wait()
//     retorna después de todos los Done, así que el llamador ve todos los slots.
//
// Un Pool no admite pasadas concurrentes; un mutex las serializa.
type Pool struct {
	base    *Dataset // dataset de Gradiente; nil si el pool solo se usa con Parciales
	cfg     Config
	workers int
	tareas  chan int
	slots   []Parcial // slots reutilizados por Gradiente (sin reservar por época)

	// Estado de la pasada en curso; lo escribe pasada() con mu tomado.
	ds     *Dataset
	salida []Parcial
	w      []float64

	wg      sync.WaitGroup
	mu      sync.Mutex
	cerrar  sync.Once
	cerrado bool
}

// NuevoPool arranca workers goroutines (mínimo 1). ds es el dataset base de
// Gradiente; puede ser nil si el pool solo se va a usar con Parciales.
func NuevoPool(ds *Dataset, cfg Config, workers int) *Pool {
	p := &Pool{
		base:    ds,
		cfg:     cfg.normalizada(),
		workers: max(workers, 1),
		tareas:  make(chan int),
	}
	if ds != nil {
		p.slots = nuevosParciales(NumBloques(ds.N, p.cfg.TamBloque), ds.D)
	}
	for range p.workers {
		go p.trabajar()
	}
	return p
}

func nuevosParciales(n, d int) []Parcial {
	ps := make([]Parcial, n)
	for k := range ps {
		ps[k] = NuevoParcial(d)
	}
	return ps
}

func (p *Pool) trabajar() {
	var local Parcial // búfer privado de este worker
	for k := range p.tareas {
		if len(local.Grad) != p.ds.D+1 {
			local = NuevoParcial(p.ds.D)
		}
		ParcialBloque(p.ds, p.cfg, p.w, k, &local)
		local.copiarEn(&p.salida[k])
		p.wg.Done()
	}
}

// Workers devuelve la cantidad de goroutines del pool.
func (p *Pool) Workers() int { return p.workers }

// pasada calcula los parciales de todos los bloques de ds en salida y espera a
// que terminen. Requiere p.mu tomado.
func (p *Pool) pasada(ds *Dataset, w []float64, salida []Parcial) {
	if p.cerrado {
		panic("ml: uso de un Pool cerrado")
	}
	validarPesos(w, ds.D)
	p.ds, p.salida = ds, salida
	p.w = append(p.w[:0], w...) // copia: el llamador puede modificar w después
	p.wg.Add(len(salida))
	for k := range salida {
		p.tareas <- k
	}
	p.wg.Wait()
}

// Gradiente implementa Calculador sobre el dataset base. Entra en pánico si el
// pool está cerrado o se creó sin dataset base.
func (p *Pool) Gradiente(w []float64) Gradiente {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.base == nil {
		panic("ml: Gradiente sobre un Pool sin dataset base (usar Parciales)")
	}
	p.pasada(p.base, w, p.slots)
	return Finalizar(ReducirEnOrden(p.slots, p.base.D), p.cfg, p.w)
}

// Parciales calcula un Parcial por bloque de ds, en orden de bloque y sin
// reducirlos. El slice devuelto es del llamador: la siguiente pasada no lo
// modifica. Es lo que un nodo envía al coordinador, que reduce los bloques de
// todos los shards en orden global (ADR-0005).
func (p *Pool) Parciales(ds *Dataset, w []float64) []Parcial {
	p.mu.Lock()
	defer p.mu.Unlock()
	salida := nuevosParciales(NumBloques(ds.N, p.cfg.TamBloque), ds.D)
	p.pasada(ds, w, salida)
	return salida
}

// Cerrar termina los workers. Es seguro llamarlo más de una vez.
func (p *Pool) Cerrar() {
	p.cerrar.Do(func() {
		p.mu.Lock()
		p.cerrado = true
		close(p.tareas)
		p.mu.Unlock()
	})
}
