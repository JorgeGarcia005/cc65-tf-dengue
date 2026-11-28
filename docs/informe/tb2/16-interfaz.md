# 16. Parte D: interfaz web

La Parte D es **Vigía**, una aplicación de una sola página (SPA) hecha con React 19, TypeScript y Vite, que consume la API de la sección 12. Esta sección describe a quién está dirigida (16.1), cómo se navega según el rol (16.2), cada vista (16.3), cómo se configura un entrenamiento desde la interfaz (16.4), cómo se muestra el clúster en tiempo real (16.5) y las medidas de accesibilidad y diseño adaptable (16.6). Las capturas son de las pruebas de punta a punta con Playwright, que recorren la SPA desplegada con un navegador real y fallan si algo no se ve como se espera.

## 16.1 Usuario y decisiones de diseño

**Usuario final.** Un analista de salud pública de una DIRESA o del CDC Perú, que tiene que decidir dónde, cuándo y para qué grupos de edad reforzar camas, vigilancia y cerco epidemiológico (la pregunta de la sección 2). No necesita saber cómo se entrenó el modelo, sino qué dice y cuánto confiar en eso. Por eso la vista principal responde la pregunta y el mapa de riesgo es la herramienta de trabajo; el clúster, que interesa a quien opera el sistema, queda en un panel aparte, solo para el administrador.

**Decisiones de diseño.**

- **Ninguna cifra en el código del front.** Todo número que muestra la SPA viene de la API, que a su vez lo toma del modelo publicado o de `reports/evaluacion.json`. Cada gráfico indica el archivo del que salen sus datos. Es la misma regla del informe (AGENTS.md §3.6).
- **Lenguaje del usuario.** «Casos esperados con signos de alarma», «semana epidemiológica» y «grupo de edad» en lugar de «probabilidad positiva», «índice de semana» o «*bin*»; los grupos de edad son las etapas de vida del MINSA.
- **Honestidad sobre los datos.** Las combinaciones con pocos casos se marcan como «datos insuficientes» con un rayado, en lugar de pintarlas con un riesgo poco confiable; la vista de evaluación muestra la calibración y la matriz de confusión, no solo la exactitud.
- **Estilo sobrio.** Tipografía serifa para títulos y monoespaciada para cifras, paleta de pocos colores con una escala secuencial para el riesgo y sin animaciones innecesarias.

## 16.2 Menú y navegación por rol

Las vistas y los roles que pueden verlas están en una sola tabla del código, de la que salen el menú y las guardas de ruta:

<!-- generado:archivo:web/src/rutas.tsx#vistas -->
```tsx
// web/src/rutas.tsx
// Matriz de vistas × rol (ADR-0007 y plan TB2): el analista de salud pública ve
// riesgo, predicción y evaluación; el clúster es solo del admin.
export const VISTAS: Vista[] = [
  { ruta: '/', titulo: 'Dónde y cuándo reforzar', menu: 'Inicio', roles: ['analista', 'admin'], elemento: <Inicio /> },
  { ruta: '/mapa', titulo: 'Mapa de riesgo', menu: 'Mapa', roles: ['analista', 'admin'], elemento: <Mapa /> },
  { ruta: '/prediccion', titulo: 'Predicción de un caso', menu: 'Predicción', roles: ['analista', 'admin'], elemento: <Prediccion /> },
  { ruta: '/evaluacion', titulo: 'Evaluación del modelo', menu: 'Evaluación', roles: ['analista', 'admin'], elemento: <Evaluacion /> },
  { ruta: '/admin', titulo: 'Clúster y entrenamiento en vivo', menu: 'Clúster', roles: ['admin'], elemento: <Admin /> },
]
```
<!-- /generado -->

- **Sin sesión**, cualquier ruta lleva al login, que recuerda a dónde iba el usuario y lo devuelve ahí después de entrar, siempre que su rol pueda verla.
- **El menú solo muestra lo que el rol puede ver**: el analista no ve «Clúster».
- **Si el analista escribe `/admin` en la barra de direcciones**, ve la pantalla 403 propia de la SPA dentro del marco de la aplicación (@fig:analista-403), y la API también le negaría los datos: la guarda del front es comodidad, la autorización real está en el servidor (sección 15.1).
- **El login** muestra un mensaje genérico ante cualquier error de credenciales (@fig:login), igual que la API, y avisa cuando hay demasiados intentos.
- **Un token vencido** (401 en cualquier petición) cierra la sesión y lleva al login.

<!-- captura: Inicio de sesión con el mensaje genérico tras una clave errada {#fig:login} [capturas/e2e-01-login.png] -->

<!-- captura: El analista intenta entrar al panel de administración y recibe la pantalla 403 {#fig:analista-403} [capturas/e2e-02-analista-403.png] -->

## 16.3 Vistas

**Inicio: dónde y cuándo reforzar** (@fig:inicio). Responde la pregunta de impacto antes que cualquier otra cosa: las combinaciones de departamento, semana epidemiológica y grupo de edad con más **casos esperados con signos de alarma** (casos notificados × riesgo calibrado, que es lo que llena camas), las semanas pico a nivel nacional y los grupos de edad de mayor riesgo, junto con las métricas del modelo y su versión. Cada fila lleva al mapa con esos filtros aplicados.

<!-- captura: Inicio con las prioridades por casos esperados, las semanas pico y las métricas del modelo {#fig:inicio} [capturas/e2e-03-inicio.png] -->

**Mapa de riesgo** (@fig:mapa y @fig:mapa-departamento). Mapa coroplético del Perú por departamento, con filtros de semana epidemiológica (un deslizador con botón para reproducir las semanas en secuencia), grupo de edad y métrica (riesgo calibrado o casos esperados). Al elegir un departamento se abre un panel con su curva por semana y la comparación entre la tasa observada y el riesgo calibrado. Los filtros quedan en la URL, de modo que una vista se puede compartir con un enlace.

<!-- captura: Mapa de riesgo en la semana pico {#fig:mapa} [capturas/e2e-04-mapa.png] -->

<!-- captura: Mapa con el panel de un departamento: curva por semana y tasa observada frente a riesgo calibrado {#fig:mapa-departamento} [capturas/e2e-05-mapa-departamento.png] -->

**Predicción de un caso** (@fig:prediccion). Formulario con departamento, año, semana, edad y sexo; valida los campos antes de enviar y muestra los errores de la API en español. El resultado incluye el riesgo calibrado, la clase, la **contribución de cada variable** al logit (qué empuja el riesgo hacia arriba y qué hacia abajo; la suma coincide con el logit, y un test lo comprueba), la latencia medida en el navegador y si la respuesta salió de Redis.

<!-- captura: Predicción de un caso con su riesgo, las contribuciones de cada variable, la latencia y el estado de la caché {#fig:prediccion} [capturas/e2e-06-prediccion.png] -->

**Evaluación del modelo** (@fig:evaluacion). Curva ROC con su AUC, curva de calibración por deciles, matriz de confusión con las métricas en el conjunto de prueba, *speedup* y eficiencia del clúster por número de nodos y latencia de la API. Es la evaluación experimental de la sección 17, presentada al usuario.

<!-- captura: Evaluación: curva ROC, calibración y matriz de confusión {#fig:evaluacion} [capturas/e2e-07-evaluacion.png] -->

## 16.4 Configuración del entrenamiento desde la interfaz

El panel de administración (@fig:admin-entrenando) permite lanzar un entrenamiento distribuido eligiendo las **épocas**, la **tasa de aprendizaje** y, en modo demostración, una **pausa entre épocas** que hace visible el avance en vivo. El formulario valida los valores antes de enviarlos (épocas entero positivo, tasa mayor que cero) y la API los vuelve a validar (sección 12.2); si ya hay un entrenamiento en curso, la API responde 409 y el panel muestra su mensaje de error.

Al lanzarlo, el panel muestra la época actual sobre el total, la pérdida, los nodos sanos sobre los registrados y el p95 de la API (sección 13.3), y lo actualiza con cada evento del WebSocket. Al terminar, el modelo nuevo se publica: la API empieza a usarlo de inmediato (el puntero atómico de la sección 12.1) y el inicio, el mapa y la predicción lo muestran con su nueva versión.

<!-- captura: Panel de administración durante un entrenamiento: época, pérdida, nodos sanos y p95 de la API {#fig:admin-entrenando} [capturas/e2e-08-admin-entrenando.png] -->

## 16.5 Tiempo real

El panel de administración recibe los eventos del clúster por WebSocket (sección 15.3): cada mensaje del protocolo (asignación, época, gradiente, caída, reasignación, fin), la pérdida y los pesos de cada época. Con ellos muestra:

- la **topología** del clúster, con el estado, la carga, el uso de CPU y la latencia de cada nodo (@fig:admin-metricas, sección 9.6);
- la **curva de pérdida** en vivo y las **barras de los pesos** del modelo, que cambian época a época;
- un **mapa que se repinta** cada cierto número de épocas con el modelo en entrenamiento;
- la **bitácora** del protocolo, donde se ve la caída de un nodo y la reasignación de su *shard* (@fig:admin-caida, sección 9.4). En modo demostración, un botón «Tumbar» corta la conexión de un nodo para comprobarlo en vivo.

**Un solo reductor para todo.** El estado del panel se calcula con una función pura, `reducir(estado, evento)`, que usan igual el WebSocket en vivo y el **modo réplica**: un entrenamiento grabado como archivo `.jsonl` se puede reproducir en el panel sin clúster, a la velocidad que se elija. Tests de Vitest comprueban el reductor con secuencias de eventos conocidas.

**Reconexión.** Si la conexión se corta por cualquier motivo distinto de un cierre normal, la SPA reintenta con espera exponencial y una variación aleatoria (para que varios paneles no reconecten a la vez), y siempre con un ticket nuevo. Si el token venció o el rol no tiene acceso, deja de reintentar y lo avisa.

**El tiempo real no frena el entrenamiento.** Los eventos se publican en un *hub* que descarta mensajes de un cliente lento en lugar de esperar, y el entrenamiento observado produce el mismo modelo bit a bit que el no observado (@tab:observador).

## 16.6 Accesibilidad y diseño adaptable

- **Teclado.** Enlace «Saltar al contenido» al inicio de cada página, foco visible en todos los controles (`:focus-visible`) y cada departamento del mapa es un elemento enfocable que se elige con Enter o Espacio.
- **Lectores de pantalla.** Cada sección tiene un título asociado (`aria-labelledby`), el resultado de la predicción se anuncia al cambiar (`aria-live`), la bitácora es una región de registro (`role="log"`) y los gráficos tienen etiquetas que describen lo que muestran.
- **Color.** El riesgo se codifica con una escala secuencial de un solo tono y, además, con el valor numérico en la tabla y en la información emergente; los «datos insuficientes» se distinguen por un rayado, no solo por color.
- **Movimiento.** Con `prefers-reduced-motion`, la SPA desactiva las transiciones y animaciones.
- **Diseño adaptable.** La disposición se reorganiza en una columna en pantallas angostas: la @fig:responsive muestra la SPA a 375 px de ancho, el de un teléfono.

<!-- captura: La SPA a 375 px de ancho {#fig:responsive} [capturas/e2e-12-responsive-375.png] -->

**Pruebas.** La SPA tiene pruebas unitarias con Vitest (guardas de ruta, cliente de la API, reductor del panel, reconexión del WebSocket, colores del mapa) y pruebas de punta a punta con Playwright contra el sistema desplegado, que recorren los flujos de cada rol y toman las capturas de esta sección. Su salida:

<!-- evidencia: evidencias/e2e.txt -->
