# ADR 0017 — El vidrio del sistema va en el marco, no en la zona de trabajo

**Fecha:** 2026-09-08 · **Estado:** aceptada · **Revisar si** Wails ofrece translucidez en Linux

## Contexto

La barra y el pie fingían el vidrio con `backdrop-filter`, que desenfoca lo que hay **dentro** de la
página. El de verdad toma lo que hay **detrás de la ventana**, y eso solo lo puede dar el sistema:
en macOS con `NSVisualEffectView`, en Windows 11 con Mica. Wails los expone; Linux no tiene nada.

Pedirlos obliga a una cosa que no es evidente: **el fondo deja de pintarlo el sistema y pasa a
pintarlo el CSS**. Un webview transparente sobre una ventana sin efecto no enseña el escritorio,
enseña un agujero.

## Decisión

Vidrio **solo en la barra y el pie**. La zona de trabajo se queda opaca.

Es lo que hacen las aplicaciones del sistema, y aquí hay una razón de más: es donde se lee y se
teclea, y un fondo que cambia según lo que haya detrás de la ventana no es sitio para un campo de
texto ni para un secreto en claro.

La interfaz **pregunta a Go si hay vidrio** (`App.Vidrio()`) y pone `data-vidrio="si"` en la raíz.
Todo el CSS del efecto cuelga de ese atributo, así que donde no lo hay la ventana queda exactamente
como antes. `MarcarVidrio` va como función y no como método, por la trampa ya conocida de que lo que
se exporta como método de `*App` cruza el puente.

El tinte es **0,55**, un número en `internal/tema` (`alfaDelVidrio`).

**Llegar ahí costó equivocarse, y conviene que quede escrito.** Se puso en 0,82 tras simular la
ventana con un degradado saturado detrás, sin desenfoque, donde el texto del pie se lavaba. Probado
en un Mac, el efecto **no se notaba**: macOS no enseña el escritorio, enseña un material ya
desenfocado y desaturado, así que dejar pasar el 18 % de eso es no tener efecto. La simulación
describía un caso que el sistema nunca produce, y calibrar contra ella fue el error.

## Alternativas descartadas

- **La ventana entera translúcida.** Más vistoso y peor: el contraste deja de poder medirse —depende
  del escritorio de cada uno— y `make contraste` existe justamente para que eso no pase.
- **Dejar el `backdrop-filter` de CSS.** Funciona igual en los tres sistemas y no es lo mismo: no ve
  lo que hay detrás de la ventana, que es el efecto entero.
- **Acrylic en vez de Mica** en Windows. Desenfoca en tiempo real y se parece más a macOS, pero
  Microsoft lo desaconseja para ventanas de trabajo porque consume y distrae.

## Lo que Wails deja a medias, y hay que rematar

Pedir `WindowIsTranslucent` no basta. Wails crea el `NSVisualEffectView` con mezcla «BehindWindow»,
que es lo correcto, pero **nunca pone la ventana como no opaca**: le cambia el color de fondo a
transparente y ya. Una `NSWindow` con `opaque = YES` compone como opaca por mucho que su color tenga
alfa cero, así que el material no tiene nada detrás que mezclar y **se dibuja como un gris plano**.
Ése era el síntoma: vidrio puesto, efecto ninguno.

Y hay una segunda capa: desde macOS 12, `WKWebView` pinta su `underPageBackgroundColor` por debajo
de la página aunque `drawsBackground` esté a `NO`. Si no se aclara, tapa el material igual.

Las dos son una línea de AppKit cada una y no hay forma de pedirlas desde la API de Wails, así que
las hace `vidrio_darwin.go` con cgo, en el arranque. Si algún día Wails las hace, ese fichero sobra
entero.

## Consecuencias

- **Las parejas que mide `make contraste` siguen midiéndose contra el color opaco de la barra.** Un
  fondo translúcido no se puede medir, y fingir que sí sería peor que no medirlo.
- Linux se queda como estaba, y hay una prueba de interfaz que lo vigila: sin el atributo, el `body`
  no puede quedar transparente.
- En Windows 10 no hay Mica: la ventana sale opaca, sin error y sin aviso.

## Verificación

- Prueba de interfaz en los dos temas: sin `data-vidrio`, el fondo del `body` **no** es transparente.
  Ésa es la garantía de que Linux no se rompe.
- `make contraste` en verde con el token nuevo.
- Compila para macOS, Windows y Linux.

- Ajustes dice si la ventana está usando el vidrio del sistema. No es adorno: la primera vez que el
  efecto no se vio, no había forma de distinguir «no llega la señal» de «el tinte tapa demasiado».

**Lo que no se ha comprobado:** el Objective-C de `vidrio_darwin.go`. En esta máquina no hay clang ni
SDK de macOS, así que ni siquiera compila aquí; lo compila el trabajo de macOS de la publicación, y
si estuviera mal la publicación fallaría. Cómo queda el material tampoco: la simulación de aquí no
tiene el desenfoque del sistema y ya engañó una vez.
