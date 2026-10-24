package ml

import "sync"

// Pool calcula el gradiente con un worker pool persistente.
//
// Patrón y sincronización:
//
//   - Los workers se crean una vez (NuevoPool) y viven hasta Cerrar: no se paga
//     el costo de crear goroutines en cada época.
//   - Un canal sin búfer reparte índices de bloque (fan-out). Cada bloque lo
//     procesa exactamente un worker.
//   - Cada worker acumula el bloque en su propio búfer privado (local) y al
//     terminar lo copia a parciales[k], el slot del bloque k. Ningún worker
//     escribe en memoria que otro lea o escriba durante la época, así que no
//     hace falta mutex sobre los acumuladores. Acumular en el búfer local y no
//     directo en el slot evita el false sharing: los slots de bloques vecinos
//     comparten líneas de caché y escribirlos fila a fila desde dos núcleos
//     hacía al pool más lento que la versión secuencial.
//   - Un sync.WaitGroup es la barrera de fin de época: Gradiente espera a que
//     los B bloques terminen antes de reducir.
//   - La reducción la hace un solo goroutine, en orden de índice de bloque
//     (ReducirEnOrden). Por eso el resultado es idéntico bit a bit al de
//     Secuencial, para cualquier número de workers.
//
// Relaciones happens-before que hacen correcto el acceso a memoria:
//   - p.w se copia antes de enviar el primer índice por el canal; cada worker lo
//     lee después de recibir, por lo que ve la copia completa.
//   - wg.Done() de cada worker ocurre después de escribir su slot; wg.Wait()
//     retorna después de todos los Done, así que la reducción ve todos los slots.
//
// Un Pool no admite llamadas concurrentes a Gradiente; un mutex las serializa.
type Pool struct {
	ds        *Dataset
	cfg       Config
	workers   int
	bloques   int
	tareas    chan int
	parciales []Parcial
	w         []float64
	wg        sync.WaitGroup
	mu        sync.Mutex
	cerrar    sync.Once
	cerrado   bool
}

// NuevoPool arranca workers goroutines (mínimo 1) sobre el dataset.
func NuevoPool(ds *Dataset, cfg Config, workers int) *Pool {
	cfg = cfg.normalizada()
	workers = max(workers, 1)
	bloques := NumBloques(ds.N, cfg.TamBloque)
	p := &Pool{
		ds:        ds,
		cfg:       cfg,
		workers:   workers,
		bloques:   bloques,
		tareas:    make(chan int),
		parciales: make([]Parcial, bloques),
		w:         make([]float64, ds.D+1),
	}
	for k := range p.parciales {
		p.parciales[k] = NuevoParcial(ds.D)
	}
	for range workers {
		go p.trabajar()
	}
	return p
}

func (p *Pool) trabajar() {
	local := NuevoParcial(p.ds.D) // búfer privado de este worker
	for k := range p.tareas {
		ParcialBloque(p.ds, p.cfg, p.w, k, &local)
		local.copiarEn(&p.parciales[k])
		p.wg.Done()
	}
}

// Workers devuelve la cantidad de goroutines del pool.
func (p *Pool) Workers() int { return p.workers }

// Gradiente implementa Calculador. Entra en pánico si el pool está cerrado.
func (p *Pool) Gradiente(w []float64) Gradiente {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cerrado {
		panic("ml: Gradiente sobre un Pool cerrado")
	}
	copy(p.w, w)
	p.wg.Add(p.bloques)
	for k := 0; k < p.bloques; k++ {
		p.tareas <- k
	}
	p.wg.Wait()
	return Finalizar(ReducirEnOrden(p.parciales, p.ds.D), p.cfg, p.w)
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
