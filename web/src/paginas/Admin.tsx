import Pendiente from './Pendiente'

export default function Admin() {
  return (
    <Pendiente
      titulo="Clúster y entrenamiento en vivo"
      issue={69}
      texto="Coordinador y nodos en vivo por WebSocket: pérdida, pesos, bitácora del protocolo y caída de un nodo."
    />
  )
}
