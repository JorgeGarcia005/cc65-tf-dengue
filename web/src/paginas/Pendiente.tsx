// Marcador de las vistas que llegan en sus propios issues (#66–#69).
export default function Pendiente({ titulo, issue, texto }: { titulo: string; issue: number; texto: string }) {
  return (
    <section className="tarjeta" aria-labelledby="titulo-vista">
      <h1 id="titulo-vista" className="titulo">
        {titulo}
      </h1>
      <p className="bajada">{texto}</p>
      <p className="nota">En construcción · issue #{issue}</p>
    </section>
  )
}
