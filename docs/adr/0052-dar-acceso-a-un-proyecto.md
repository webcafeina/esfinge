# 0052 · Dar acceso a una bóveda de proyecto

**Fecha:** 2026-10-05 · **Estado:** aceptada, en construcción · **sin escribir todavía**

## Contexto

La [ADR 0051](0051-entregar-una-boveda.md) resolvió «el proyecto se acaba y hay que darle lo suyo al
cliente» con **una copia muerta**: su contraseña, su clave de recuperación, y a partir de ahí ya no depende
de nadie. El cliente lo eligió el 2026-10-02 con el coste delante —*«que se envíe una copia, como las
credenciales»*— y la ficha dejó escrito lo que quedaba fuera:

> **No se cambia sin preguntar**, y si algún día hace falta es una funcionalidad nueva —permisos,
> revocación, quién ve qué— y no un ajuste de ésta.

El 2026-10-05, tres días después, hizo falta: *«necesito que estas bóvedas de proyecto sí que se actualicen
en tiempo real a las personas que la tengan compartida. Solo las bóvedas de proyectos, lo que comparta de mi
bóveda no.»* Así que esto es esa funcionalidad nueva, con esos tres ejes, **y la 0051 no se deroga**:
entregar una copia sigue existiendo al lado, para cuando el proyecto de verdad se acaba.

Lo que lo hace caro no es el cifrado: es que **cinco piezas dan por hecho que una bóveda es de una persona**.
El Durable Object se nombra por la cuenta y el testigo lleva la cuenta dentro, que es *toda* la
autorización que hay; `ajustes["idBoveda:<ref>"]` rechaza con 409 cualquier otra bóveda; la clave de un
proyecto va envuelta con la clave de bóveda de **una** personal; y la lista de proyectos vive cifrada dentro
de esa personal, así que no hay ningún sitio donde pueda aparecer una bóveda que no sea tuya.

## Decisión

**Dos gestos distintos y las dos palabras separadas**: «**Dar acceso**» deja la bóveda viva en el equipo de
quien la recibe y se puede quitar; «**Entregar una copia**» es la 0051. **No se decide al crear el
proyecto**: cualquiera puede recibir acceso el día que haga falta.

| | |
|---|---|
| **Qué se comparte** | Solo **bóvedas de proyecto**. Una entrada suelta de la personal sigue siendo copia (ADR 0043) |
| **Qué es «en vivo»** | **Al abrirlo, y cada minuto mientras esté abierto** — el sondeo que ya existe. **Sin push** |
| **Permiso** | **Por persona: ver o editar** |
| **Hace falta cuenta** | **Sí**; a quien no la tiene se le invita, como hoy |
| **Recompartir** | **Sí, quien puede editar** |
| **La lista de quién tiene acceso** | **La ven todos los que la tienen** |
| **Última mano** | **«Nombre (correo)»**, y **no a la vista todo el rato** |
| **Al quitar el acceso** | **Se rota la clave** y se vuelve a envolver para los que quedan |

### La ranura va sellada hacia la identidad, no envuelta con la clave de quien recibe

Es la decisión técnica entera, y la evidente no sirve. Envolver la clave del proyecto **con la clave de
bóveda de esa persona** —que es lo que hace `boveda-principal` para el dueño (ADR 0050)— funciona… y deja
que **solo ella** pueda rehacer esa ranura. El día que haya que rotar la clave, el dueño no podría
reenvolver para los que quedan sin tenerlos delante.

Así que cada titular tiene una ranura con la clave del proyecto **sellada hacia su identidad pública**, que
es la primitiva de los sobres de compartir (ADR 0043) y ya está publicada en el servidor:

```
Tipo: "acceso:<idMiembro>"          // 8 bytes hex
Codificacion: "sobre-x25519-v1"
Contenedor: HPKE(clave del proyecto → identidad pública de esa persona)
```

**El tipo tiene que ser distinto por persona**, y eso no es estilo: el sello indexa los sobres **por su
tipo** (`sel.Huellas[tipo]`, `sel.Sobres[tipo]`), así que dos ranuras del mismo tipo no hacen que sobre una
— hacen que **la bóveda no abra**.

### Y la lápida de ranura, que no estaba prevista

**Quitar una ranura no es representable en la fusión.** `fundirSobres` une por tipo y conserva la que está
en un solo lado; no hay forma de decir «ésta se quitó». O sea que el dueño quita la ranura del revocado,
sube, y **el primer equipo con una copia de antes la resucita**. Hace falta una **lápida**: el tipo se queda
marcado como retirado, con su fecha, y lo retirado gana a lo vivo. Es el mismo patrón que las lápidas de las
entradas y por la misma razón — sin ellas, «quitada aquí» y «nunca existió allí» son indistinguibles.

**Eso convierte rotar la clave en lo que de verdad hace que revocar aguante**, y no en un extra: aunque una
ranura vuelva por una copia vieja, lleva dentro la clave de antes y ya no abre nada. El cliente eligió rotar
sin saber esto; esto lo confirma por una segunda razón.

### La bóveda no se muda de sitio

Se queda en el Durable Object de su dueño. Lo que se añade es una tabla `miembros`, y que **el Worker valide
la sesión de quien llama contra su propio objeto y después hable con el del dueño diciendo qué cuenta es** —
la rendija ya estaba abierta y escrita: el objeto confía en el Worker, igual que en `recibir()`, que no pide
sesión. Así no hay objeto nuevo, ni migración de lo que ya existe.

**El permiso se hace cumplir en el servidor**, no en la pantalla, y se dice con las palabras con que Esfinge
dice lo de Touch ID: *quien solo puede ver tiene la clave para leerla; lo que impide que escriba es el
servidor, no el cifrado*.

### Y el reparto de la clave va por el buzón que ya existe

Dar acceso no inventa transporte: es un sobre HPKE como el de compartir una entrada, con otra carga de unos
cientos de bytes. Con eso vienen gratis **la huella enseñada antes**, **que espere a que lo acepten** —si
no, cualquiera que sepa tu correo te mete un proyecto en la lista— y **la invitación** a quien todavía no
tiene cuenta.

## Alternativas descartadas

**Un Durable Object por bóveda compartida**, con su lista de miembros. Es lo que haría cualquiera y obliga a
**mudar las bóvedas que ya existen** la primera vez que se comparten, con el riesgo de una migración por
cada proyecto en vez de una por cuenta. Dejarlas donde están cuesta una tabla y un salto más del Worker.

**Envolver la ranura con la clave de bóveda de quien recibe**, como el dueño. Más simétrico y **mata la
revocación**: nadie salvo esa persona puede rehacer su ranura.

**Empujar los cambios en vivo** por WebSocket desde el servidor. El cliente lo descartó al preguntárselo
—le vale al abrir y cada minuto— y conviene no abrirlo a la ligera: una conexión que no termina nunca por
bóveda abierta es la misma familia del fallo de los seis flujos de eventos, donde a partir del sexto oyente
toda llamada se quedaba esperando para siempre, sin error.

**Que quien recibe pueda borrar el proyecto para todos.** Se le ofreció al cliente y lo dejó fuera. Puede
dejar de verlo, archivarlo y entregar una copia a un tercero; cargárselo, no.

**Borrar en remoto lo que ya se bajó.** Se puede intentar y **no se puede prometer** —basta con no abrir
Esfinge—, y media promesa en un gestor de contraseñas es peor que ninguna. Lo que hay es lo que ya vale para
un equipo perdido, dicho igual de claro.

**Guardar la última mano como la identidad de la bóveda.** No sirve: la identidad es **una por bóveda** y la
fusión elige una de las dos cuando chocan. Quién cambió qué tiene que ser un miembro, no una identidad.

## Consecuencias

**«En vivo» es hasta un minuto, y un proyecto cerrado no se entera de nada.** Es lo que ya pasa entre los
dos Macs del cliente, dicho ahora para dos personas. La pantalla lo dice en vez de dejar creer que es
instantáneo.

**Hay personas distintas escribiendo la misma bóveda, y la fusión no sabe quién es quién.** Converge —para
eso están el orden total del desempate y el historial de contraseñas— pero el perdedor de un choque de notas
pierde su texto sin enterarse, igual que hoy entre dos equipos. La última mano es lo único que lo hace
visible después.

**El freno de subidas deja de ser solo del dueño.** Se cuenta por quien escribe, o un miembro podría agotar
el cupo de la cuenta que le dio acceso.

**El servidor sabe con quién compartes**, y ahora también **quién tiene acceso a qué proyecto**: una tabla
de miembros es exactamente eso. Va a `docs/seguridad.md` y a la política, como fue la lista de sitios.

**Y una bóveda compartida no cría identidad**, así que desde dentro no se comparte nada. Regalar la
identidad es regalar la firma (ADR 0051), y con varias personas sería peor: todas firmarían igual.

## Verificación

**Lo que se va a comprobar, y cada prueba mutada:**

- Tres accesos a la vez abren los tres, y **mutar el tipo a uno fijo pone la bóveda en `ErrManipulada`** —
  que es lo que demuestra que el tipo por titular no es estilo.
- **Quitar el acceso aguanta una copia vieja**: se revoca y se funde después contra el fichero de antes del
  revocado; su ranura no vuelve viva, y si vuelve no abre. Esta prueba no existiría sin haber leído
  `fundirSobres`.
- El **403** del que solo puede ver, contra el Worker de verdad levantado en local.
- Que la última mano **no se despegue de su cambio** al fundir.
- Las cruzadas Go↔TypeScript **con datos de las secciones nuevas** en el generador al azar: sin eso, los 400
  casos no las tocan y la cruzada pasa sin comprobar nada.

**Lo que no se podrá comprobar aquí**, y se dice desde el principio: **todo lo que es con dos cuentas de
verdad**. Que lo que guarda uno aparezca en el otro al abrirlo, que quitar el acceso se note, y que la
extensión del otro rellene con lo compartido.
