# ADR 0038 — Sincronizar la bóveda: la bóveda entera, con versión, y la fusión a tres bandas

**Fecha:** 2026-09-18 · **Estado:** aceptada, sin interfaz todavía · **Continúa la [0035](0035-las-cuentas.md)** ·
**Matiza la [0026](0026-la-papelera.md)** (el rastro de lo borrado) · **Revisar cuando** se use en dos equipos de verdad

## Contexto

Con cuenta, la misma bóveda vive en varios equipos, y cada uno puede cambiarla sin conexión. Hay que
juntar esos cambios **sin perder nada** y sin que el servidor pueda leer ni fabricar nada. Y el riesgo
número uno de todo el plan es éste: **un error al juntar se copia a todos los equipos de la cuenta**.

## Decisión

### Qué viaja

**La bóveda entera, tal como está en el disco**, cifrada igual. Con dos cambios que prepara
`Boveda.PrepararSubida`:

- **Sin las ranuras de este equipo** —Touch ID, PIN—, que no se suben nunca. Quitarlas obliga a volver a
  sellar: si no, el otro equipo echaría en falta su huella y diría que está manipulada.
- **Con la versión del servidor sellada dentro** (`sello.Sincro`), cifrada con la clave de bóveda. Así un
  servidor no puede devolver una versión vieja haciéndola pasar por nueva: la versión que dice tiene que
  coincidir con la de dentro, y la de dentro no la puede escribir nadie sin la clave.

### Cómo se sincroniza (`internal/sincro`)

Una pasada es: **bajar, fundir y, si queda algo que el servidor no tiene, subir sobre la versión que se
bajó** (`If-Match`). Si otro equipo sube en medio, el servidor contesta que la versión ya no es la última
y se vuelve a empezar. Cada equipo recuerda la última versión vista y sus bytes —la **base** de la fusión—
junto al fichero de la bóveda (`<ruta>.base` y `<ruta>.sincro`, permisos 600).

- **El turno se reserva**, y una pasada pedida mientras hay otra en marcha la hace la que está en marcha.
- **La serie del fichero sale del mismo cerrojo que la subida** y que la fusión. Leída después, un
  guardado de otro hilo —el navegador guardando una contraseña— se daría por subido sin estarlo.
- **Un servidor que vuelve atrás no se funde ni se pisa**: una versión menor que la ya vista, o ninguna
  bóveda después de haberla tenido, paran la sincronización y se dicen.
- **Nada de esto cuenta como actividad**: el paquete no conoce el reloj del bloqueo.
- Se sincroniza al arrancar, tres segundos después de cada guardado —juntando los seguidos— y cada cinco
  minutos, con espera creciente tras un fallo de red.

### Cómo se funde (`Boveda.Fundir`)

**A tres bandas**, contra la base, entrada a entrada por su identificador:

| Aquí | Allí | Queda |
|---|---|---|
| igual que la base | cambiada | la de allí |
| cambiada | igual que la base | la de aquí |
| cambiada | cambiada | **campo a campo**; si un mismo campo cambió en los dos, la «mayor» |
| borrada | sin tocar | borrada |
| borrada | cambiada | **la cambiada**: la edición gana al borrado |
| no está (sin base) | está | está, salvo que aquí haya su lápida y no haya cambiado desde entonces |

- **La «mayor»** es la de más revisiones, luego la de fecha más reciente, luego la de huella mayor: un
  orden que **da lo mismo lo mire quien lo mire**.
- **Ninguna contraseña se pierde en un choque**: la que no gana va al historial de la entrada, con los
  historiales de los dos lados. Lo que sí se puede perder es una **nota** editada a la vez en dos sitios.
- **El orden es el del servidor**, con lo nuevo de aquí al final.
- Los **sitios excluidos** se funden como conjunto; las **secciones que esta versión no conoce**, enteras;
  las **ranuras**, a tres bandas y, si no, la más reciente.
- **Una fusión que se llevaría más de la mitad de las entradas no se aplica sola**: se pregunta. Y
  cualquier fusión que borre algo deja antes una copia del fichero en `<ruta>.antes-de-fundir`.

### Lo que cambia en la bóveda

Todo **opcional y dentro de lo cifrado**, así que una bóveda de la 2.22.2 se abre igual y una Esfinge
vieja conserva lo nuevo sin entenderlo. **ESF1 no se toca.**

- **`Entrada.Revision`**, que pone la bóveda al guardar —nunca quien edita— y sube de uno en uno.
- **Lápidas** (`lapidas`: identificador → fecha): borrar del todo —a mano, al vaciar la papelera o a los
  treinta días— deja una, **que dura seis meses**. Sin ellas, «borrada aquí» y «todavía no ha llegado
  allí» son lo mismo. Pasado ese plazo, un equipo que vuelva con la entrada la subirá como nueva: es el
  coste.
- **`sello.Sincro`**, arriba.
- **`AbrirEnMemoria`**, que abre sin escribir nada: `AbrirBytes` guarda si hay papelera caducada, y con la
  bóveda de otro equipo eso pisaría el fichero de aquí.

## Alternativas descartadas

- **Sincronizar entrada a entrada**, con el servidor sabiendo cuántas hay y cuándo cambia cada una. Sería
  más fino y el servidor aprendería mucho más de cada bóveda. Entera, solo sabe cuánto ocupa.
- **Fundir con las fechas** (`cambiada`) en vez de con la base: tienen resolución de un segundo y
  dependen del reloj de cada equipo. Solo desempatan, y la revisión va antes que ellas.
- **Que la edición no gane al borrado.** Lo que se pierde al revés es una contraseña editada; así, una
  entrada borrada vuelve y se puede volver a borrar.
- **Conservar el orden de cada equipo.** Era lo natural y **lo tumbó la prueba de los tres equipos**: la
  bóveda fundida nunca era igual a la del servidor y los equipos se la pasaban sin fin.

## Consecuencias

- Con dos equipos editando la misma entrada, **la nota puede perder un lado**; la contraseña, no.
- **Una entrada borrada puede volver** si otro equipo la editó a la vez, o si pasa más de seis meses sin
  conectarse.
- El servidor ve **cuánto ocupa la bóveda y cuándo cambia**, no qué hay dentro.
- **Quien borre la bóveda tiene que borrar también `.base`, `.sincro` y `.antes-de-fundir`**: la lección
  de la caché de iconos. Es de la A2, que es donde se borra desde la ventana.

## Verificación

**Comprobado aquí**, en `internal/boveda` e `internal/sincro`:

- **Tres equipos haciendo cosas al azar** —crear, cambiar la contraseña y la nota, papelera, restaurar,
  borrar del todo, excluir sitios— y sincronizándose cuando les toca, con **16 semillas**: todos acaban
  igual que el servidor, nadie se queda subiendo sin fin y **ninguna contraseña se pierde** salvo las de
  entradas borradas del todo a propósito. Al escribirla cazó dos fallos: el del orden, y un borrado que
  volvía a matar lo que otro equipo había decidido conservar.
- Cada fila de la tabla de arriba, con su prueba; y **se rompieron a propósito** el historial de la
  contraseña perdedora y la regla de la lápida ya vista, y las pruebas lo cazaron. La de la lápida **no lo
  cazaba** al principio —la edición y el borrado caían en el mismo segundo y pasaba por el empate— y se
  endureció.
- **Una bóveda escrita por el código de la 2.22.2** (`testdata/boveda-2.22.2.esfinge`, que no se
  regenera) se abre entera y se sincroniza.
- Otro equipo que sube en medio de una pasada, un servidor que vuelve atrás de tres formas, un guardado
  que se cuela, dos pasadas a la vez, y el vigilante.
- **La tubería entera contra el servidor de verdad**, levantado en local: una cuenta creada desde un
  equipo, otro que entra con su código y baja la bóveda, los dos trabajando a la vez en la misma entrada, y
  los dos acaban igual; lo del servidor no lleva nada en claro. Corre en `make comprobar` y en la puerta
  de `publicar.yml`.

**Sin comprobar**: dos equipos de verdad, con la red que se va y viene y la tapa del portátil cerrada a
medias. Eso es de la A2, en el Mac.
