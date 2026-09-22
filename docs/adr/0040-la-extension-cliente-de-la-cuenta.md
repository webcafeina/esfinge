# ADR 0040 — La extensión, cliente de la cuenta (fase E)

**Fecha:** 2026-09-22 · **Estado:** aceptada; E1 y E2 hechas, falta la E3 en su Mac · **Continúa la [0035](0035-las-cuentas.md)**,
que decidió con el cliente que la extensión fuera un cliente completo de la cuenta · **Revisar cuando** llegue
la auditoría

## Contexto

Hoy la extensión no sabe nada de la bóveda: todo lo pide a la aplicación por el canal nativo (ADR 0027).
Sin la aplicación abierta y la bóveda abierta en ella, no rellena ni guarda. La ADR 0035 decidió con el
cliente que, **con cuenta, la extensión lo haga todo sola** —rellenar, códigos, guardar y actualizar—
**yendo siempre por la cuenta**, aunque la aplicación esté abierta, y con su propio desbloqueo. Sin
cuenta, sigue como hoy. El plan está en `docs/cuentas.md`, «La extensión, cliente de la cuenta».

El 2026-09-22, con la A3 comprobada en sus dos Macs, el cliente eligió el orden: **primero la E y después
la auditoría externa**, que así revisa también la parte más expuesta.

## Decisión

**Una segunda implementación del formato de la bóveda, en TypeScript, que se vigila contra la de Go.** El
contrato es `docs/formato-boveda.md`; si una de las dos no lo cumple, las pruebas cruzadas lo dicen.

Por entregas, cada una publicable sin romper lo anterior:

- **E1, el núcleo, sin nada visible.** En `navegador/src/nucleo/`: el contenedor `ESF1` (modo único y de
  texto, que es todo lo que usa la bóveda), abrir y guardar la bóveda con su sello, la forma canónica, la
  fusión, el código de un solo uso, y las derivaciones de la cuenta —clave de acceso y posesión—. **No lo
  importa todavía ningún punto de entrada de la extensión**, así que lo publicado no cambia.
  - **Argon2id con `hash-wasm`** y **XChaCha20-Poly1305 con `@noble/ciphers`**, a versión exacta:
    WebCrypto no trae ninguno de los dos. SHA-256, HMAC y HKDF, de WebCrypto.
  - Se prueba **contra los vectores fijos de Go** (`internal/cripto/testdata/`), leídos de su sitio, sin
    copiarlos: abrir, y sellar con la misma sal y el mismo nonce da los mismos bytes (ADR 0022).
  - Y **cruzado con Go**: una bóveda escrita por uno la abre el otro, y la fusión de escenarios al azar da
    lo mismo byte a byte en los dos. Lo lanza una prueba de Go que ejecuta el código de la extensión con
    Node, empaquetado al vuelo.
- **E2, la extensión con cuenta.** Entrar, desbloquear y bloquear en el panel; la bóveda abierta solo en
  `storage.session`; sincronizar con `alarms`; y rellenar, códigos, guardar y actualizar contra la bóveda
  del navegador cuando hay cuenta. Con **`VERSION_DEL_AVISO` en 2**, la política de privacidad, las fichas
  de las tiendas y `docs/seguridad.md` al día, y la **prueba con la extensión cargada de verdad** que era
  deuda alta.
- **E3, en su Mac**: rellenar y guardar con la aplicación cerrada, y un cambio en el navegador que aparece
  en la aplicación.

## Alternativas descartadas

- **Argon2id en JavaScript puro** (`@noble/hashes` lo trae). Evitaría el WebAssembly y el
  `'wasm-unsafe-eval'` de la CSP, pero es varias veces más lento, y desbloquear pide al menos un Argon2id
  de 64 MiB y tres pasadas. Se queda como plan B si una tienda rechaza el WebAssembly.
- **Que la extensión siga pidiéndolo todo a la aplicación también con cuenta.** Es lo que hay hoy y es lo
  que el cliente decidió dejar atrás: obliga a tener la aplicación abierta.
- **Portar la fusión sin pruebas cruzadas**, confiando en dos baterías de pruebas escritas por separado.
  Dos fusiones que no coinciden se pasan la bóveda sin fin entre equipos, que es el riesgo número uno de
  las cuentas (ADR 0038).

## Consecuencias

- **Dos implementaciones que mantener**: cualquier cambio del formato o de la fusión se hace en las dos, y
  las pruebas cruzadas se ponen rojas si se olvida una.
- La extensión gana **dos dependencias de ejecución**, que Mozilla revisa y que entran en la compilación
  reproducible.
- Con la E2, **la extensión se conecta a internet** y guarda la bóveda cifrada en el navegador: cambian el
  aviso del panel, la política y lo declarado en las dos tiendas.

## Verificación

**E1, hecha el 2026-09-22**, sin publicar: no cambia nada de lo que se instala —el núcleo no entra en ningún
paquete de la extensión, comprobado buscándolo en `dist/`—.

- **El ESF1 de TypeScript contra los vectores fijos de Go**: los seis de modo único y de texto se abren y
  se sellan byte a byte igual, incluido el de 64 MiB. Cambiando el orden de bytes de un parámetro al
  sellar, siete pruebas se ponen rojas.
- **Cruzado con Go** (`cruzada_test.go` en `internal/boveda`, `internal/codigos` e `internal/cuenta`, con
  `ESFINGE_CRUZADA=1`; corre en `make comprobar` y en la puerta de publicación):
  - la forma canónica de entradas con `<`, `&`, U+2028, controles, emoji y campos desconocidos;
  - **la fusión con 400 escenarios al azar por pasada**, hechos para chocar —mismos identificadores en los
    dos lados, fechas y revisiones que empatan, lápidas de entradas vivas, ranuras del mismo segundo—: diez
    pasadas seguidas, 4.000 escenarios, **sin una diferencia**. Y se comprobó que la prueba caza de verdad
    rompiendo la extensión tres veces: la edición que ya no gana al borrado (85 casos distintos), el
    desempate al revés (15) y U+2028 sin escapar (219);
  - una bóveda de Go abierta en la extensión con la maestra y con la clave de recuperación tecleada de
    cualquier manera, cambiada allí y vuelta a abrir en Go; una nueva de la extensión abierta en Go; lo que
    sube una lo funde la otra, en los dos sentidos, y una versión que no cuadra no se funde en ninguno;
  - 202 códigos de un solo uso con semillas al azar, los tres algoritmos, de seis a diez cifras y semillas
    mal copiadas; la clave de acceso a la cuenta; y el correo normalizado.
- **Lo que no coincide y no se arregla, dicho**: Go conserva el texto de un número (`1.50`) y JavaScript no;
  y Go compara un campo desconocido por sus bytes, con el HTML escapado, y la extensión por su forma
  canónica. Solo afectaría a campos que hoy no existe ninguna versión que escriba.

**E2, hecha el 2026-09-22** (se publica con la 2.25.0):

- **Lo que se decidió al construirla**, que la ADR no decía:
  - **Con cuenta, el trabajador de fondo contesta los mismos verbos del canal** (`nucleo/fuente.ts`), con
    las reglas y frases de la aplicación: el panel, el guion de las páginas y la tarjeta de guardar no
    cambian. Lo de la cuenta —entrar, el código, desbloquear, bloquear, salir— solo se atiende por el
    puerto del panel.
  - **Copia el panel**, porque el trabajador de fondo no puede tocar el portapapeles, y **no se borra
    solo**: el panel lo dice. Sin pedir `clipboardWrite`, que en Chrome avisa al actualizar y desactiva
    la extensión hasta aceptarlo; basta el clic de la persona.
  - **Quince minutos sin tocarla** y se cierra, como la aplicación por defecto, y al cerrar el navegador.
    Sin ajuste todavía.
  - **Desbloquear con una contraseña que no abre** prueba la cuenta (la nueva, cambiada en otro equipo),
    como la aplicación desde la 2.24.1.
  - **Una cola en la bóveda de TypeScript**: sin cerrojos, dos guardados cruzados compartían serie y el
    segundo no se subía, y una fusión pisaba lo guardado en medio. Una prueba lo vigila.
  - **La lista de sufijos públicos, de `tldts`**, cruzada con la de Go en 30 casos. Lo que no es ASCII se
    rechaza mirando lo escrito: el navegador convierte solo `bücher.de` a punycode y Go no.
  - **`VERSION_DEL_AVISO` en 2**, con el aviso del panel, la política de la web, su copia para Mozilla,
    la portada, las fichas de las dos tiendas y `docs/seguridad.md` al día.
- **Comprobado aquí, con la extensión cargada de verdad** (`navegador/pruebas-reales`, en `make comprobar`
  y en la puerta): un Chromium sin pantalla con la extensión compilada para pruebas, contra el servidor de
  cuentas levantado aquí y **sin la aplicación**: entrar con el código, rellenar sola en un sitio guardado
  y no en otro, recibir lo que se guarda en otro equipo, guardar desde el navegador y que suba, bloquear
  —y que entonces no rellene— y desbloquear, el cierre a los quince minutos sin tocarla, y el cierre cuando
  otro equipo la olvida, que lo dice y ofrece volver a entrar. Tres roturas a propósito cazadas: no pasar
  por la cuenta, no subir y un plazo de bloqueo de 1.500 minutos. Capturas del panel en los dos temas,
  miradas.
- **Y por piezas**: los verbos (qué se ofrece, que una contraseña no sale hacia otro sitio —rompiéndolo,
  cazado—, el freno de rellenos), los dominios cruzados con Go y la cola de la bóveda.

**Corregido en la 2.25.1, lo primero que vio el cliente al probarla**: para leer el código hay que ir al
correo, el panel se cierra y al abrirlo otra vez se empezaba de cero, con otro correo. La entrada a medias
vivía en una variable del trabajador de fondo, que se duerme a los pocos segundos. Ahora va en
`storage.session` diez minutos —lo que dura el código—, el panel la retoma al abrirse, y «Volver» la
cancela. La prueba con la extensión cargada cierra el panel en ese punto, como una persona.

**Comprobado por el cliente en su Mac con la 2.25.1 (2026-09-22), cargada a mano en Chrome**: entrar con
el código cerrando el panel para ir al correo, rellenar con la aplicación cerrada, guardar desde la
tarjeta y que llegue a sus Macs, lo de los Macs en la extensión, bloquear y desbloquear, copiar, y
«Chrome en Mac» en la lista de equipos. «Funciona todo perfecto».

**2.25.2, lo que pidió después**: un botón para **sincronizar a mano**, en la aplicación y en la
extensión, sin quitar la pasada de cada minuto. En la aplicación va en la línea de sincronización de la
bóveda y **no espera** los tres segundos de después de guardar (`Vigilante.Ya`, que corta también esa
espera si ya estaba en marcha); en la extensión va en el panel, y además **abrir el panel pide una
pasada**, como la aplicación al volver a su ventana. Al hacerlo salió una carrera: el botón del panel
volvía en el acto si ya había una pasada, con lo de antes; ahora espera a la que está en marcha.

**2.25.3**: el botón pasa a ser **dos flechas en círculo que giran** mientras sincroniza, en la aplicación y
en la extensión, que lo pidió el cliente: con la palabra y nada que se moviera no se sabía si había hecho
algo.

**Comprobado por el cliente en su Mac con la 2.25.3**: las flechas giran y sincronizan, en la aplicación
y en la extensión. «Funciona perfecto».

**Sin comprobar todavía**: el resto de la E3 —en su Mac, con Chrome y Firefox de verdad y la aplicación cerrada—,
**Firefox con la extensión cargada** (la prueba es solo Chromium), y que las tiendas acepten el
WebAssembly y la conexión nueva.
