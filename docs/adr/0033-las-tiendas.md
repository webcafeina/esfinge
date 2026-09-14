# ADR 0033 — La extensión, pública en las tiendas de Chrome y de Firefox

**Fecha:** 2026-09-14 · **Estado:** aceptada, en curso · **Continúa la [0027](0027-el-canal-con-el-navegador.md)
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

### Pendiente en esta misma decisión

- **La web** (`web/`, GitHub Pages): portada, privacidad y soporte. Los enlaces del aviso ya apuntan a
  `https://webcafeina.github.io/esfinge/`.
- **Las fichas** (`docs/tiendas/`), sus imágenes, **el trabajo `tiendas`** en `publicar.yml` —Firefox con
  `web-ext sign --channel=listed`, Chrome con la API v2 y una cuenta de servicio— y **los pasos del
  cliente**: las cuentas, la primera subida a Chrome a mano —la API v2 no crea fichas— y los secretos.
- **El identificador de Chrome**, que asigna la tienda: se añade a `extensionesDeChrome` cuando se sepa.

## Alternativas descartadas

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
- **Las revisiones pueden tardar**: la de Chrome, días o semanas, con `https://*/*` y `nativeMessaging`.

## Verificación

**Comprobado aquí:**

- **El aviso**, en un Chromium de verdad con el panel compilado: sale sin aceptar, **sin una sola petición
  al trabajador de fondo** antes de aceptar —se registran—, aceptar guarda la versión y trae las cuentas,
  Intro acepta sin marco de foco al abrir, los enlaces van a la web en otra pestaña, y ya aceptado no sale.
- **El icono**: el estado pendiente gana a todos, también a las páginas sin https y a una rellenada.
- **Solo vale la versión del aviso de ahora**, y nada que no sea una aceptación.
- **Capturas del aviso** en claro y en oscuro, miradas.
- `permisos.mjs` en verde sin `activeTab`; los manifiestos compilados con la versión, Firefox 140 y la
  declaración de datos.
- **`web-ext lint` 10.6.0**: cero errores y el aviso de Android.
- **Reproducible**: el zip de código fuente, descomprimido en una carpeta limpia e instalado desde cero,
  compila un `dist/firefox` **idéntico byte a byte** al del repositorio.

**Sin comprobar:**

- **El trabajador de fondo y el guion de página con la extensión cargada**: el bloqueo de sus puertos y
  que la página empiece al aceptar sin recargar. Es la deuda alta de siempre.
- Que la compilación sea igual con el Node 24 de los revisores de Mozilla.
- Que las dos tiendas aprueben lo declarado.
