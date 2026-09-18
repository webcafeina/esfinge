# ADR 0033 — La extensión, pública en las tiendas de Chrome y de Firefox

**Fecha:** 2026-09-14 · **Estado:** aceptada · **Continúa la [0027](0027-el-canal-con-el-navegador.md)
y la [0032](0032-guardar-desde-la-pagina.md)** · **Revisar cuando** las dos tiendas la aprueben, y si
cambian sus normas de datos

## Contexto

Con la entrega 3 comprobada en el Mac del cliente (2.21.2), la extensión seguía instalándose a mano:
en Firefox como **complemento temporal** —desaparece al cerrar Firefox— y en Chrome **descomprimida
con el modo de desarrollador**, reinstalando con cada versión. La entrega 5 la lleva a las tiendas para
que se instale y se actualice sola.

Antes de preguntar nada, dos agentes leyeron la documentación oficial de las dos tiendas. De ahí
salieron requisitos que obligan a tocar el código y no solo a rellenar fichas.

## Decisión

Decidido con el cliente, en dos rondas de preguntas con opciones:

| Pregunta | Elegido |
|---|---|
| Chrome | **Pública** en la Chrome Web Store |
| Firefox | **Pública** en addons.mozilla.org |
| Quién sube cada versión | **Sola, al publicar**, desde GitHub Actions |
| Cuentas | **Webcafeína, con info@webcafeina.com** |
| Aviso y consentimiento de datos | **En el panel, la primera vez que se abre** |
| Datos de comerciante en la ficha de Chrome (DSA) | **Sí, los de Webcafeína** —nombre legal, dirección y teléfono públicos— |
| Política de privacidad y soporte | **Una web del proyecto con GitHub Pages** |
| Windows | **Arreglarlo antes de publicar** ([ADR 0034](0034-el-canal-con-el-navegador-en-windows.md)) |

Se hace en cuatro bloques: Windows, la extensión lista para las tiendas, la web y las fichas con la subida
automática.

### El aviso de datos, en el panel la primera vez

**Chrome exige un aviso visible y una aceptación activa dentro de la extensión**, aunque los datos no
salgan del equipo (política de datos de usuario, endurecida el 01-08-2026). El panel enseña «Esfinge y
tus datos» —qué lee, que va **solo a Esfinge en este ordenador**, qué guarda y los enlaces a la política
de privacidad y al soporte— con «Aceptar y empezar».

**Y hasta aceptarlo la extensión no hace nada**, que es lo que da sentido al aviso. Se hace cumplir en tres
sitios, porque uno solo no basta:

- **El panel** no pregunta nada antes de aceptar: su primera pregunta empareja el navegador y haría salir
  el aviso de permiso en Esfinge.
- **El trabajador de fondo** no lanza el puente al refrescar el icono y contesta `sin-consentimiento` a
  cualquier puerto, venga del panel o de una página.
- **El guion de la página** no mira formularios, no escucha envíos —que es donde se leen las contraseñas—
  ni contesta al panel. Si se acepta con la página abierta, empieza sin recargar.

El icono dice «Esfinge · Abre el panel para empezar», con el «!» del ámbar ya medido, **antes que
cualquier otro estado**. Se guarda en `storage.local` **con la versión del aviso**: si lo que dice cambia,
sube `VERSION_DEL_AVISO` y se vuelve a pedir, que es lo que pide Chrome cuando cambian las prácticas.

### Los manifiestos

- **Fuera `activeTab`**: no lo usaba ningún código, y las dos tiendas piden permisos mínimos.
- **Firefox 140 como mínimo y `data_collection_permissions`**, obligatorio para extensiones nuevas desde
  el 03-11-2025. Mozilla dice expresamente que **lo que se manda a una aplicación nativa cuenta**. Se
  declaran `authenticationInfo` (usuario y contraseña), `personallyIdentifyingInfo` (el correo) y
  `browsingActivity` (la dirección de la pestaña). Sin `websiteContent`: de la página solo sale lo del
  formulario de entrar. La clave solo existe desde Firefox 140; la 128 dejó de tener soporte en 2025.
- **La versión, pasada de verdad** al empaquetar en `publicar.yml`: sin ella salía de `git describe`, y en
  un lanzamiento a mano sin etiqueta el manifiesto habría llevado el primer número de un hash.

### Sin `innerHTML`

Los ocho `innerHTML` metían dibujos nuestros —la marca, la silueta y los iconos—, pero son lo primero que
marca la revisión de Mozilla (`UNSAFE_VAR_ASSIGNMENT`). Ahora pasan por `dibujar` (`dibujo.ts`), que los
lee con `DOMParser` como SVG y los inserta como nodos. `web-ext lint` queda sin errores y con un solo aviso:
la declaración de datos no existe en Firefox para Android antes de la 142, y la extensión no funciona en
Android —no hay native messaging—.

### El código fuente, reproducible

Mozilla exige el código de verdad de lo que va empaquetado con Vite, con instrucciones, y **lo compila y
compara byte a byte**. `herramientas/fuente-de-la-extension.sh` arma el zip con lo que sigue git en
`navegador/`, **los cuatro ficheros de fuera que la extensión importa** —`build/marca.svg`,
`build/icono-barra.svg`, `frontend/src/monograma.ts` y `frontend/src/tokens.css`— y `COMPILAR.md` con Node
22, pnpm 11.20.0 —fijado con `packageManager`— y las órdenes.

### La web del proyecto

`web/`, publicada en GitHub Pages por `.github/workflows/web.yml` en `https://webcafeina.github.io/esfinge/`,
que es adonde apuntan los enlaces del aviso. Tres páginas en español: **portada** —qué hace, descargas y la
extensión—, **política de privacidad** y **soporte**.

- **La política de privacidad dice lo mismo que el aviso del panel**, que `docs/seguridad.md` y que lo que
  se declara en las tiendas: qué lee la extensión, que va solo a Esfinge en este ordenador, qué guarda en
  el navegador y en memoria —y cuánto—, para qué es cada permiso, las dos conexiones de la aplicación y
  que Webcafeína no recibe ningún dato. Lleva un comentario que lo recuerda.
- **Los colores, espaciados y radios son los tokens de la ventana**, copiados al armarla, y **cada pareja
  de color es una de las ya medidas**, dicha al lado. La **estructura** —escala de títulos, cuerpo a 16 px,
  88 px entre secciones, botones en píldora de 36 px— sale del sistema `ollama` del catálogo de sistemas de
  diseño, «documentación primero». Solo la estructura: paleta y letra son las de Esfinge.
- **Sin captura de la ventana**: las del README son de la 2.0.3 y no enseñan la bóveda.
- `herramientas/armar-web.sh` la arma y **falla si hay un enlace roto** a un fichero de la propia web.

### Las fichas y la subida automática

- **`docs/tiendas/ficha.md`**: los textos de las dos fichas, el propósito único, la justificación de cada
  permiso, las respuestas de «Privacy practices» de Chrome y las notas para los revisores de Mozilla.
  **`docs/tiendas/amo-metadata.json`**, lo mismo para la primera subida automática a Firefox.
- **Las imágenes** (`docs/tiendas/imagenes/`): cinco capturas de 1280×800 y el mosaico de 440×280,
  compuestas por `navegador/herramientas/imagenes-de-tienda.mjs` **con las capturas de verdad** del panel,
  la marca en el campo y la tarjeta, y textos en blanco y `#d8d8de` sobre la piedra, parejas ya medidas.
- **Cada publicación genera también** el zip de Chrome **sin `key`** —el identificador lo asigna la
  tienda— y el zip de código fuente.
- **El trabajo `tiendas`** de `publicar.yml`, después de la publicación de GitHub y sin frenarla:
  1. **Compila el código fuente en una carpeta limpia y lo compara byte a byte** con el paquete, que es lo
     que hará Mozilla.
  2. **Firefox**: `web-ext sign --channel=listed --approval-timeout=0` con el código fuente y la metadata.
  3. **Chrome**: `herramientas/tienda-chrome.mjs`, API v2 con **cuenta de servicio** —un token firmado
     cada vez, sin tokens de refresco que caduquen a los siete días—, sin dependencias.
  4. **Sin los secretos, avisa y se salta.**
- **`docs/tiendas/pasos.md`**: lo que hace el cliente —las cuentas, la primera subida a Chrome a mano, los
  secretos— con lo que está por confirmar marcado.

### Pendiente en esta misma decisión

- **Los pasos del cliente** (`docs/tiendas/pasos.md`).
- ~~**El identificador de Chrome**, que asigna la tienda~~: `jfkkegampjamnnlopobepjoanebemegp`, añadido a
  `extensionesDeChrome` el 2026-09-15, calculado de la clave pública de la consola. Queda que la web cambie
  «Muy pronto» por los enlaces.
- **Dejar de colgar los zip de la extensión** en las publicaciones de GitHub cuando las dos fichas estén
  en marcha.

## Alternativas descartadas

**La política de privacidad en webcafeina.com.** Con la marca de la casa, pero cada cambio sería un paso
del cliente, y esta política tiene que cambiar a la vez que el código.

**No listadas en las dos tiendas.** Se actualizan solas igual y no se exponen, pero el cliente la quiere
a la vista.

**El aviso en una pestaña al instalar.** Más visible, pero el cliente prefirió no abrir nada por su cuenta.
El precio, dicho: **hasta abrir el panel la extensión no rellena nada**, y eso puede no entenderse; lo
compensa el «!» del icono.

**Reutilizar el permiso que ya pide Esfinge en su ventana.** No añade pantallas, pero Chrome pide que el
aviso esté en la propia extensión y la revisión podía rechazarla.

**Seguir con `innerHTML`.** Es inofensivo con dibujos propios, pero cada aviso es un motivo para que un
revisor se pare, y con `https://*/*` y `nativeMessaging` la revisión ya va a ser a fondo.

## Consecuencias

- **Una extensión recién instalada no hace nada hasta que se abre su panel** y se acepta el aviso.
- **Los datos de la empresa quedan a la vista en la ficha de Chrome**: nombre legal, dirección y teléfono.
- **Cada versión sube el código fuente a Mozilla**, y si no se compila igual, la rechaza. Una dependencia
  que no dé el mismo resultado —o un fichero de fuera de `navegador/` que se importe sin añadirlo al
  guion— rompe la publicación en Firefox.
- **Las revisiones pueden tardar**: la de Chrome, días o semanas, con `https://*/*` y `nativeMessaging`. Al
  enviarla (2026-09-15), la consola avisó de «Permisos de host amplios» y propuso `activeTab` o una lista de
  sitios. **Se descartó**: `activeTab` solo da acceso tras pulsar el botón, así que dejaría de rellenar sola y
  de ofrecer guardar, y una lista de sitios no existe para un gestor de contraseñas.

## Verificación

**Comprobado aquí:**

- **El aviso**, en un Chromium de verdad con el panel compilado: sale sin aceptar, **sin una sola petición
  al trabajador de fondo** antes de aceptar —se registran—, aceptar guarda la versión y trae las cuentas,
  Intro acepta sin marco de foco al abrir, los enlaces van a la web en otra pestaña, y ya aceptado no sale.
- **El icono**: el estado pendiente gana a todos, también a las páginas sin https y a una rellenada.
- **Solo vale la versión del aviso de ahora**, y nada que no sea una aceptación.
- **Capturas del aviso** en claro y en oscuro, miradas.
- **La web**, armada con su comprobación de enlaces y mirada en escritorio y a 400 px, en claro y en
  oscuro: sin errores en la consola y sin desbordar a lo ancho. De mirarla salió quitar la captura de la
  ventana, que era de la 2.0.3.
- `permisos.mjs` en verde sin `activeTab`; los manifiestos compilados con la versión, Firefox 140 y la
  declaración de datos.
- **`web-ext lint` 10.6.0**: cero errores y el aviso de Android.
- **Reproducible**: el zip de código fuente, descomprimido en una carpeta limpia e instalado desde cero,
  compila un `dist/firefox` **idéntico byte a byte** al del repositorio.

- **Las imágenes de las fichas**, miradas una a una. De mirarlas salieron dos arreglos: el pie del panel
  decía «Extensión 2.19.0» —la versión de mentira de las capturas— y la tarjeta tapaba a medias el texto
  de la página de ejemplo.
- `publicar.yml` válido, `tienda-chrome.mjs` sin errores de sintaxis y `amo-metadata.json` válido.

**Y en las tiendas de verdad**: la 2.22.1 fue a Firefox con `web-ext` el 2026-09-15, y **las dos fichas se
aprobaron y son públicas el 2026-09-18** —la de Chrome, tras avisar de revisión a fondo por los permisos de
host amplios—. La 2.22.2 subió sola a las dos, **y a Chrome por la API v2 con la cuenta de servicio a la
primera**: `uploadState: SUCCEEDED` y `state: PENDING_REVIEW`. Fichas:
https://addons.mozilla.org/es-ES/firefox/addon/esfinge/ y
https://chromewebstore.google.com/detail/esfinge/jfkkegampjamnnlopobepjoanebemegp.

**Comprobado por el cliente en su Mac con la 2.22.0 instalada a mano** (2026-09-14): «todo parece
correcto», con el aviso de datos.

**Sin comprobar:**

- **Nada de la subida a las tiendas contra las tiendas de verdad**: las direcciones de la API v2 de
  Chrome salen de su documentación, y que AMO acepte `es-ES` como idioma de la ficha en la primera
  subida está por confirmar. La primera publicación con los secretos puestos es la prueba.

- **El trabajador de fondo y el guion de página con la extensión cargada**: el bloqueo de sus puertos y
  que la página empiece al aceptar sin recargar. Es la deuda alta de siempre.
- Que la compilación sea igual con el Node 24 de los revisores de Mozilla.
- Que las dos tiendas aprueben lo declarado.
