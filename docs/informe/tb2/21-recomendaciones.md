# 21. Recomendaciones

**Para el uso del sistema en salud pública**

1. **Planificar antes de la temporada.** Usar las prioridades por departamento, semana y edad (sección 18) para programar camas y personal y activar el cerco epidemiológico en las semanas previas a la temporada alta, no cuando los servicios ya están saturados.
2. **Distinguir volumen de riesgo.** En los departamentos que destacan por cantidad de casos, reforzar la capacidad hospitalaria; en los que destacan por riesgo por caso, la detección temprana de los signos de alarma y la derivación oportuna.
3. **Reentrenar con cada año de datos.** El modelo se reentrena desde el panel en el clúster con los mismos parámetros; hacerlo con cada actualización del conjunto de datos del MINSA mantiene las prioridades al día.
4. **No usar el modelo para decisiones clínicas individuales.** Su poder de discriminación es moderado; la decisión sobre un paciente debe seguir los criterios clínicos de la guía de la OMS.

**Para mejorar el modelo**

5. **Incorporar variables clínicas.** Con acceso a comorbilidades, serotipo, infección previa o resultados de laboratorio (por ejemplo, de la ficha de investigación epidemiológica, con los resguardos de privacidad que correspondan), el poder predictivo mejoraría de forma importante.
6. **Agregar variables del entorno.** Precipitación, temperatura e índices aédicos por semana y departamento podrían anticipar mejor los brotes; se integran como columnas nuevas del gold sin cambiar la arquitectura.
7. **Evaluar por año.** Validar el modelo entrenando con los años anteriores y probando con el siguiente mediría mejor su capacidad de anticipar una temporada nueva que el conjunto de prueba aleatorio actual.

**Para el sistema**

8. **Desplegar con TLS.** Para un uso real fuera de una red local, poner la SPA, la API y el WebSocket detrás de un *proxy* con `https` y `wss` (sección 15.3) y administrar el secreto JWT con un gestor de secretos.
9. **Llevar el clúster a varias máquinas.** El protocolo ya funciona sobre TCP entre contenedores; desplegar los nodos en máquinas distintas permitiría medir el efecto de la latencia de red real que la sección 11.4 estima, y escalar a conjuntos de datos mayores.
10. **Mantener la equivalencia como regla.** Cualquier cambio en el cálculo del gradiente o en el protocolo debe mantener las pruebas de equivalencia bit a bit y la verificación con Spin: son las que permiten distinguir una optimización de un error.
