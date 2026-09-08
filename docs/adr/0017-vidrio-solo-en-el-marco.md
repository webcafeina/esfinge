# ADR 0017 — El vidrio del sistema va en la barra lateral, no en la zona de trabajo

**Fecha:** 2026-09-07 · **Estado:** aceptada · **Revisar si** Wails ofrece translucidez en Linux

## Contexto

La barra y el pie fingían el vidrio con `backdrop-filter`, que desenfoca lo que hay **dentro** de la
página. El de verdad toma lo que hay **detrás de la ventana**, y eso solo lo puede dar el sistema:
en macOS con `NSVisualEffectView`, en Windows 11 con Mica. Wails los expone.

> **Corrección (2026-09-07):** aquí se dijo que Linux no tenía nada, y **es falso**: Wails ofrece
> `linux.Options.WindowIsTranslucent`, que llama a `SetWindowTransparency` sobre la ventana GTK. No se
> usa por otra razón —sin desenfoque del compositor, la transparencia enseña el escritorio a pelo, y
> eso no es vibrancia sino un agujero— pero la opción existe.

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

**La barra lateral no lleva tinte: el material del sistema *es* su fondo.**

> **Corrección (2026-09-07), y es la tercera de esta ficha.** Aquí se decía que el tinte era 0,55,
> un número en `internal/tema` (`alfaDelVidrio`). Ese número ya no existe, y la historia de cómo se
> eligió es el mejor aviso que tiene este documento.
>
> Se puso primero en **0,82**, calibrado contra una simulación con un degradado saturado detrás y
> **sin desenfoque**. En un Mac no se notaba. Se razonó que el problema era el tinte —macOS no enseña
> el escritorio sino un material ya desenfocado y desaturado, así que dejar pasar el 18 % de eso es
> no tener efecto— y se bajó a **0,55**. En un Mac **tampoco se notaba**: «sigue sin apreciarse, solo
> es un gris».
>
> Las dos veces se eligió el número mirando algo, y las dos veces se estaba arreglando el problema
> equivocado. La causa estaba en el material, no en lo que había por delante (abajo). Y con la causa
> resuelta el tinte sobra por definición: en una barra lateral de macOS no hay ninguna capa entre el
> material y el texto. Cualquier alfa que se ponga ahí reproduce exactamente el síntoma del que se
> venía.

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

**Y una tercera, que es la que costó tres versiones y la que de verdad explicaba el gris.** Wails
crea el `NSVisualEffectView`, le pone la mezcla y el estado, y **nunca le pone el material**. Se
comprobó leyendo su código, que está en el caché de módulos de esta máquina: en
`internal/frontend/desktop/darwin/WailsContext.m` solo hay `setBlendingMode` y `setState`, y
`setMaterial` no aparece en todo el directorio. Sin material, la vista se queda con el de por
defecto, `NSVisualEffectMaterialAppearanceBased`, que Apple dejó **obsoleto en macOS 10.14** y que en
macOS moderno se dibuja como una superficie plana.

Es decir: había vidrio, y estaba desenfocando nada. `vidrio_darwin.go` le pone
`NSVisualEffectMaterialSidebar`, que es el de las barras laterales del Finder y de Correo, y fuerza
`NSVisualEffectStateActive` para que el efecto no se apague al perder el foco.

Las tres son una línea de AppKit cada una y no hay forma de pedirlas desde la API de Wails, así que
las hace `vidrio_darwin.go` con cgo, en el arranque. Si algún día Wails las hace, ese fichero sobra
entero.

**Cómo se escribe ese fichero después de haberlo roto.** La 2.9.1 metió aquí un diagnóstico que
compilaba en verde y **cerraba la aplicación al arrancar**; hubo que revertirlo y publicar la 2.9.2
para devolverle la herramienta al cliente. Las reglas que salieron de aquello: nada de devolver
cadenas a Go —un `UTF8String` autoliberado deja un puntero colgando—, nada de `valueForKey:` —se
puede escribir una propiedad que no se deja leer—, nada de `alphaComponent` —lanza excepción sobre un
color de patrón—. Solo asignaciones a propiedades públicas, preguntando antes si existen.

## Consecuencias

- **Las parejas que mide `make contraste` siguen midiéndose contra el color opaco de la barra.** Un
  fondo translúcido no se puede medir, y fingir que sí sería peor que no medirlo. Con el tinte
  retirado esto es más cierto que antes: bajo vidrio, el contraste de la barra lateral lo sostiene el
  material del sistema, que es lo que hace cualquier barra lateral nativa, y no un color nuestro.
- Linux se queda como estaba, y hay una prueba de interfaz que lo vigila: sin el atributo, el `body`
  no puede quedar transparente.
- En Windows 10 no hay Mica: la ventana sale opaca, sin error y sin aviso.

## Verificación

- Prueba de interfaz en los dos temas: sin `data-vidrio`, el fondo del `body` **no** es transparente.
  Ésa es la garantía de que Linux no se rompe.
- Prueba de interfaz en los dos temas, nueva: **con** `data-vidrio`, el fondo de `.lateral` es
  transparente y el de `.zona` no. Existe porque el tinte ya ha vuelto dos veces, y la tercera que lo
  haga que falle una prueba y no un cliente.
- `make contraste` en verde después de quitar el token.
- Compila para macOS, Windows y Linux.

- Ajustes dice si la ventana está usando el vidrio del sistema. No es adorno: la primera vez que el
  efecto no se vio, no había forma de distinguir «no llega la señal» de «el tinte tapa demasiado».

**Comprobado en un Mac (2.10.0, 2026-09-07): el vidrio se ve.** Y puesto al lado de la barra lateral
del Finder, **se ve igual**: el desenfoque, que a primera vista parecía excesivo, es el que macOS 26
pone en todas las barras laterales del sistema. Como el radio del desenfoque no es ajustable en
AppKit —lo fija el material— la comparación con una aplicación de Apple es la única forma de saber si
sobra o si es el estándar. Aquí era el estándar, y no había nada que calibrar.

**Lo que sigue sin poder comprobarse aquí:** el Objective-C de `vidrio_darwin.go`. En esta máquina no
hay clang ni SDK de macOS, así que ni siquiera compila; lo compila el trabajo de macOS de la
publicación, y **que ese trabajo pase en verde solo dice que compila, no que arranque** —lo aprendimos
con la 2.9.1—.

**Y la lección que costó tres versiones:** las tres veces se dedujo la causa razonando sobre lo que
Wails «debería» hacer, y las tres se falló. El código de Wails estaba todo el tiempo en el caché de
módulos de esta máquina, y la causa se leyó en dos minutos el día que se fue a mirar. Cuando el fallo
está en una plataforma que no se puede ejecutar aquí, **leer la biblioteca va antes que razonar sobre
ella**. Y si ni así, Wails admite `wails build -devtools`, que deja el inspector en la aplicación de
verdad: cada hipótesis pasa a ser una prueba en vivo en lugar de una versión publicada.
