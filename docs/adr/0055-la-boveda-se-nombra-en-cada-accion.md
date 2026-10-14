# 0055 · La bóveda se nombra en cada acción

**Fecha:** 2026-10-14 · **Estado:** aceptada · **Matiza la [0054](0054-los-agentes-por-mcp.md)**

## Contexto

La [ADR 0054](0054-los-agentes-por-mcp.md) decidió que un agente actúa **sobre la bóveda que esté abierta y
ninguna otra**, y que conmutar es administrar y lo hace la persona en la ventana. La razón era buena: que un
agente al que alguien le cuele una instrucción no pueda meterse en la bóveda de un cliente.

Una semana de uso con varias bóvedas la ha puesto del revés. El cliente, el 2026-10-14:

> *«Ahora es muy molesto tener que ir cambiando de bóveda y que diga todo el rato que no puede hacer algo.
> Debería ser orgánico, que el operador del agente diga en qué bóveda quiere hacer la acción, y si no concreta
> ninguna que el agente diga cuál está abierta y si lo quiere ahí para confirmar.»*

**El fallo de fondo no era que faltara una funcionalidad: era que la bóveda es un modo.** Un modo hay que
recordarlo, y lo que un agente no recuerda entre llamadas lo paga quien le pide las cosas. Es la misma familia
que los arreglos del 13 de octubre —escribir sin mirar dónde— con el signo cambiado: allí el agente escribía
en la bóveda equivocada por no mirar, y aquí no escribe en ninguna por no poder.

## Decisión

**La bóveda es un dato de cada acción, no un modo.** Todas las herramientas que tocan la bóveda aceptan
`boveda` (opcional, por su nombre).

- **Con nombre**: Esfinge conmuta a ésa, hace la acción y **devuelve la ventana a donde estaba**.
- **Sin nombre**: la abierta, y sigue mandando lo de la 2.52.0 — mirar `esfinge_estado` **antes de escribir,
  siempre**, decirlo en voz alta y **preguntar** si es la de un proyecto y nadie la ha nombrado.

Y una herramienta más, `esfinge_bovedas`, porque para decir «en Zeri's» hay que saber que existe.

Cuatro cosas de cómo se mueve, y ninguna es un detalle:

1. **Cambia, hace y vuelve**, elegido por el cliente frente a quedarse donde el agente lo dejó. Lo que ve en la
   ventana sigue siendo lo suyo.
2. **La vuelta es tras un respiro de diez segundos, no tras cada llamada.** Tres acciones seguidas en la misma
   bóveda son una ida y una vuelta: cada conmutación rearranca la sincronización, el goteo de iconos y el aviso
   a la ventana.
3. **No se devuelve a nadie a la fuerza.** Si alguien ha conmutado mientras tanto, la vuelta no se hace.
4. **Conmutar por un agente no cuenta como actividad ni marca el proyecto como usado.** Lo primero es la regla
   absoluta de la casa —lo que se repite solo no mantiene la bóveda abierta—; lo segundo es que la lista de
   proyectos la ordena lo que abre la persona, y reordenarla por detrás es mentirle.

**Sin permiso en la ventana**, elegido por el cliente teniendo delante la alternativa —preguntar una vez por
bóveda y sesión—. Abrir otra no pide contraseña: con la personal desbloqueada la clave ya está en memoria, que
es lo que la [ADR 0050](0050-una-boveda-por-proyecto.md) decidió y lo que `conLaPersonal` usa desde entonces.

## Alternativas descartadas

**Un verbo `abrir` en vez de un argumento.** Es lo que parecía natural y es peor: el agente tendría que
acordarse de haber conmutado y de volver, y **lo que un agente tiene que recordar entre llamadas es justo lo
que se le olvida**. Una acción acabaría en la bóveda de otro cliente sin que nada lo dijera. Pegada a la
acción no hay estado que perder.

**Escribir en el fichero de la otra bóveda sin abrirla**, dejando la ventana quieta. Más limpio en apariencia y
roto por debajo: **la sincronización va con la bóveda abierta**, así que lo escrito ahí no subiría hasta que
alguien la abriera — y quien tenga acceso compartido (ADR 0052) no lo vería. Con la conmutación de verdad eso
sale gratis, porque `alCerrarLaBoveda` ya vacía lo pendiente antes de parar la sincronización.

**Permiso en la ventana por bóveda.** Se ofreció y el cliente lo descartó por la fricción, que es real y ya le
hizo pedir «No volver a preguntar» el 8 de octubre. Si algún día molesta lo contrario, el enganche es el mismo
sitio.

**Varias bóvedas abiertas a la vez.** Es lo que la ADR 0050 descartó y sostiene media aplicación: el reloj, el
candado, el goteo de iconos y el canal del navegador siguen siendo uno.

## Consecuencias

**Lo que esto cuesta, y hay que decirlo con todas las letras.** Un agente al que alguien le cuele una
instrucción **puede ahora inventariar la bóveda de cualquier cliente**, no solo la que tuvieras abierta. Las
contraseñas siguen pidiendo aprobación en la ventana y los códigos también; lo que se abre sin preguntar es el
inventario: qué cuentas hay, de qué sitios y cuáles están mal. **La bóveda abierta era un cerrojo de hecho y
deja de serlo.** Está dicho así en `docs/seguridad.md`, con las mismas palabras que lo del portapapeles.

**Y la lista de proyectos es la lista de clientes.** La ADR 0050 la protegía a propósito —los ficheros se
llaman por una referencia al azar para que un listado de la carpeta no la delate— y `esfinge_bovedas` se la
da al agente. No hay forma de dar lo que se pide sin esto.

De ahí salen los dos límites que sí se mantienen: **lo archivado no se ofrece** —archivar es decir «con esto
ya no se trabaja»— y **lo que no está en este equipo tampoco**, porque bajarlo es administrar.

**La tarjeta de aprobación dice ahora en qué bóveda.** Desde que la acción puede ocurrir en una que no estás
mirando, aprobar sin saber dónde es aprobar a ciegas: «la contraseña de GitHub» no es la misma pregunta en la
tuya que en la de un cliente.

**Administrar sigue fuera**: crear, archivar, borrar, entregar o compartir una bóveda, no. Abrir una que ya
existe no crea ni destruye nada.

## Verificación

- **Que se actúa donde se dijo**: se crea con `boveda` desde la personal y se **abre el fichero del proyecto**
  para mirar lo guardado, no lo que quedó en memoria.
- **La tubería entera**: los bytes de MCP por donde entran de verdad → socket → traductor → canal → el fichero
  de otro proyecto → la vuelta.
- **Que vuelve, y que no tira de nadie**: mutadas las dos. La segunda **cazó un fallo de diseño**: la primera
  versión comparaba «en qué bóveda estamos» y eso no distingue «no se ha movido nadie» de «se fue y volvió al
  mismo sitio». Lo dijo la prueba, no la lectura. Ahora es un contador de conmutaciones.
- **Que no cuenta como actividad**: treinta idas y vueltas con el reloj corriendo y la bóveda cerrándose igual.
  Mutado.
- **Que una dormida se dice como tal** y no como que no existe, porque el arreglo es distinto y el agente no
  puede hacerlo.
- **Que el campo llega**: lo vigila `TestLoQueDeclaraElEsquemaLlegaALaPeticion`, que arma los argumentos desde
  el propio esquema. Es la guarda que nació del día que `sitios` se perdió cinco días en el traductor.

**Lo que no se ha comprobado**: con una bóveda **compartida de solo ver**, que escribir conteste su motivo
propio. El camino existe —`porQueNoSeEscribe` ya lo decide— y la lista ya dice `seEscribe: false`, pero montar
dos cuentas con una bóveda compartida para esta prueba es el mismo coste que ya bloquea otras dos filas de
`deuda.md`. Y **que esto deje de ser molesto**, que lo dice el cliente usándolo.
