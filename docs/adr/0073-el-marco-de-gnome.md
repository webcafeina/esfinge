# ADR 0073 · En GNOME el marco lo dibuja Esfinge

**Fecha**: 2026-11-27
**Estado**: aceptada
**Reabre** la [ADR 0020](0020-cada-sistema-su-forma.md) en su parte de Linux, y extiende la
[ADR 0066](0066-la-barra-de-titulo-de-windows.md) a un segundo sistema.

## Contexto

El cliente instaló la 2.79.1 de cero en su Ubuntu 26.10 ARM y mandó capturas. Entre lo que señaló:
«mira si la barra es como está planteada o aún no».

**Estaba como se planteó, y lo planteado estaba mal.** Con el marco del sistema, Esfinge enseñaba
**dos barras apiladas**: la de GTK, con «Esfinge» y los botones de ventana, y debajo la nuestra, con
el título de la sección. Ninguna aplicación de GNOME tiene dos. Él lo vio comparando con **Apolo**,
que dibuja una sola: el nombre sobre la barra lateral, el título de la sección sobre el contenido y
los botones de ventana al final.

La ADR 0020 había decidido lo contrario para Linux con esta razón escrita: «GTK3 sin decoraciones no
dibuja marco —ni sombra, ni esquinas, ni borde de redimensionado— y Wails no expone con qué volver a
decorarla». **Se midió en el fuente de Wails 2.15.0 y dos tercios de eso son falsos**:

- **El redimensionado lo trae Wails hecho.** `internal/frontend/desktop/linux/frontend.go` enciende
  `window.wails.flags.enableResize` cuando la ventana es `Frameless`, y su runtime hace los cuatro
  cantos y las cuatro esquinas con un borde de 6 px y sus cursores.
- **Las esquinas y la sombra puede dibujarlas el CSS**, porque Wails expone
  `linux.Options.WindowIsTranslucent`, que pone la ventana transparente.

Lo único cierto era que GTK, sin decorar, no pone sombra ni esquinas **por su cuenta**.

## Decisión

**En GNOME la barra de título la dibuja Esfinge**, y es **una sola** que ocupa la fila de arriba:

1. **El nombre de la aplicación sobre la barra lateral**, centrado y en negrita, con el botón de
   plegar al extremo de esa mitad —donde ya estaba—.
2. **El título de la sección sobre el contenido**, centrado, que es lo que ya hacía.
3. **Los tres botones de ventana al final**, como **círculos de Yaru del mismo tamaño**: lo que los
   distingue es su glifo. El rojo de cerrar es de Windows y aquí no se pone.

### Lo que no se cambia sin preguntar

- **Quién dibuja el marco se decide en un solo sitio**, `app.MarcoDe`, que lee `main.go` para pasarle
  `Frameless` a Wails y la interfaz por `Plataforma`. Escrito dos veces, una ventana acaba con dos
  barras o con ninguna — que es literalmente lo que esta ficha viene a arreglar.
- **La barra de título no es una banda más: es una capa transparente** sobre la fila que ya existe,
  con `pointer-events: none` y sólo los botones recuperándolo. Montarla como una fila de la rejilla
  volvería a dar dos barras; sin el `pointer-events` taparía la cabeza de la barra lateral entera,
  incluido el botón de plegar.
- **El marcado sigue siendo el mismo en los cuatro sistemas** (ADR 0064) y lo único que cambia es
  dónde cae cada pieza.

## Alternativas descartadas

**Dejarlo con dos barras.** Es lo que había. Se descartó al verlo al lado de Apolo: son unos 45 px de
cromo de más y el nombre de la ventana repetido.

**Quitar la nuestra y quedarnos con la de GTK.** Una sola barra sin tocar el marco, pero el título de
la sección tendría que bajar al contenido y eso cambia la cabeza de todas las pantallas de todos los
sistemas, no sólo de GNOME.

**Esquinas redondeadas y sombra, ahora.** Exigen poner la ventana transparente
(`WindowIsTranslucent`) y que el CSS pinte **todo** el fondo. Se deja fuera a propósito: aquí no hay
forma de ver una ventana, y equivocarse ahí no es un detalle cosmético, es una ventana negra o en
blanco. Con la ventana opaca el marco funciona entero y lo que falta son las esquinas y la sombra.
Está en `deuda.md` con lo que haría falta medir.

**Cosechar los iconos de ventana de Yaru.** Los tres botones ya llevan su SVG dentro del componente,
así que vestirlos de GNOME no pide ni un icono nuevo ni tocar el marcado. Si al verlos en una máquina
los glifos no convencen, cambiarlos es un paso aparte.

## Consecuencias

- **La ventana de GNOME pierde la sombra y las esquinas del sistema** hasta que se haga lo de arriba:
  GTK no las dibuja en una ventana sin decorar.
- **El doble clic para maximizar deja de funcionar.** Lo traía la barra de GTK; la nuestra tiene el
  manejador, pero vive en la capa que no recibe clics. Está en `deuda.md`.
- **Maximizada, la ventana no cuadra sus esquinas**, porque hoy no hay ninguna redondeada. Cuando las
  haya, habrá que quitarlas al maximizar.
- **Catorce reglas de la hoja de Windows hubo que acotar antes**: colgaban de `[data-barra="propia"]`
  sin nombrar su sistema, que era cierto mientras Windows fuera el único con barra propia. A GNOME se
  le habrían aplicado la banda de Windows y sus botones cuadrados.

## Verificación

**Lo que se comprobó, y con qué.**

- **El fuente de Wails 2.15.0**, no la memoria: `SetDecorated(!Frameless)` en `linux/window.go`, el
  `enableResize` de `linux/frontend.go` y el redimensionado por cantos y esquinas de su runtime
  (`runtime/desktop/main.js`), y `WindowIsTranslucent` en `pkg/options/linux`.
- **Una prueba de interfaz que mide la barra**: que el nombre cae centrado **sobre la barra lateral**
  y no sobre el contenido, que los tres botones miden lo mismo, que **ninguno se queda sin su
  círculo** y que van en orden pegados al canto, y que la capa no recibe clics.
- **Las 24 pruebas de Windows en verde** tras acotar sus catorce reglas, que es lo que dice que no se
  ha roto el sistema que ya funcionaba.
- **Dos guardianes nuevos en Go**: que toda variable que lee una hoja la escriba alguien —cazó un
  `--sutil` inventado **en esta misma hoja**, a los pocos minutos de escribirlo— y que cada regla de
  una hoja nombre su sistema, que es lo que impide que vuelvan las catorce.
- **Capturas de las dos temas**, miradas.

**Lo que no se ha comprobado, y sólo se ve en una máquina.** Que GTK esconda la barra de verdad; que
el arrastre y el redimensionado de Wails funcionen en GTK3; cómo queda la ventana sin sombra ni
esquinas; y si los glifos de los tres botones tienen el peso de los de Yaru. Aquí no hay entorno
gráfico: lo único que se puede ver es lo que pasa dentro del webview.
