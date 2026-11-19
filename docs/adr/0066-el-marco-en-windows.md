# ADR 0066 — El marco en Windows lo dibuja Esfinge

**Fecha:** 2026-11-19 · **Estado:** aceptada · **Segunda mitad de la [0064](0064-windows-con-fluent.md)**,
**reabre la [0020](0020-windows-y-gnome.md)** · **Revisar si** aparece algún gesto de ventana que no
funcione

## Contexto

La [ADR 0064](0064-windows-con-fluent.md) vistió Windows de Fluent —paleta, escala, formas, iconos—
y dejó fuera dos cosas a propósito: **la anatomía de los formularios** y **el marco**. El cliente lo
miró y dijo exactamente lo que en Apolo ya había dicho una vez: *«los elementos se siguen pareciendo
mucho a la versión actual, no están adaptados a Windows»*. Tenía razón, y la causa era la anatomía:
un ajuste seguía siendo la casilla delante del rótulo, que es la gramática de macOS.

Decidió que las dos iban juntas y antes de publicar: *«no se publica la 2.70.0 hasta que la ventana
se vea de verdad como una de WinUI»*.

## Decisión

**En Windows, un ajuste es una SettingsCard y la barra de título la dibuja Esfinge.** En los demás
sistemas no cambia nada.

### La anatomía

| | |
|---|---|
| Componente | `TarjetaAjuste` + `Interruptor`, con **el mismo marcado en los cuatro sistemas** y una gramática por sistema en el CSS |
| En Windows | Texto a la izquierda, control a la derecha, tarjeta de **68 de alto, radio 4 y borde de 1 px**, y las tarjetas de una sección a **4 px** unas de otras |
| El bloque | **Deja de dibujarse**: si no, salen tarjetas dentro de una tarjeta. Su cabecera **sí** es una tarjeta cuando lleva el interruptor de la sección, que es lo que Configuración pone arriba de cada página |
| El interruptor | La píldora de **40 × 20** con el pomo de 12. **Encendida, el oro con la piedra encima** (ADR 0021), no el acento del sistema |
| Qué **no** entra | Una casilla dentro de un formulario —«La he apuntado en un sitio seguro», la red oculta— sigue siendo una casilla en los cuatro sistemas |
| Las pestañas de Ajustes | **Con su glifo al lado del nombre**, y el de cada sistema (pedido el 2026-11-19 al dar el visto bueno) |
| El control segmentado | En Windows deja de ser la píldora de macOS y pasa a ser **la SelectorBar de Fluent**: sin carril, la activa teñida y con su barra de acento debajo |

**Y la anatomía vale para los cuatro sistemas, no solo para Windows** (dicho por el cliente el
2026-11-19: *«hay que tener en cuenta de adaptar esta anatomía/estructura también en todos los
sistemas y entornos, incluido Mac»*). Eso es exactamente lo que `TarjetaAjuste` hace posible y lo que
convierte cada fase siguiente en escribir una hoja: **el marcado ya es el mismo**, así que macOS
tendrá su fila agrupada de los Ajustes del Sistema, GNOME su lista en tarjeta y KDE su FormLayout sin
tocar una línea de `App.tsx`. Mientras no llegue su fase, los tres se quedan con la gramática base,
que es la de hoy.

### El marco

| | |
|---|---|
| Quién decide | **`app.MarcoDe(sistema)`, y solo ahí**: lo lee `main.go` para poner `Frameless` —que es opción de arranque— y lo lee la interfaz por `Plataforma` |
| Las medidas | **32 de alto y botones de 46 × 32**, en Go, viajando a la interfaz como propiedades de CSS. Escritas también en la hoja serían la misma medida en dos sitios |
| Qué da Wails | Esconde la barra conservando el redimensionado y llama a `ExtendFrameIntoClientArea`, así que **la sombra y las esquinas de Windows 11 siguen siendo del sistema** |
| Qué ponemos | La banda, el nombre, los tres botones sobre `window.runtime` —`WindowMinimise`, `WindowToggleMaximise`, `WindowIsMaximised`, `Quit`—, el rojo `#c42b1c` del cierre y **el doble clic**, que Wails no trae |
| Qué falta | **El menú de Snap** al pasar por Maximizar, en un paso publicable aparte |

## Alternativas descartadas

**Montar la barra solo donde el marco es nuestro.** Es lo evidente y rompe dos cosas: la regla de la
0064 de que el marcado es el mismo en todos los sistemas, y —más caro en el día a día— **la
simulación por URL**, que cambia de aspecto sin recargar. Se dibuja siempre y la esconde el CSS; con
`display: none` no está en el árbol de accesibilidad, no entra en la tabulación y ninguna prueba la
encuentra, así que donde no se usa no existe.

**Preguntar la forma de un ajuste con `:has()` en la base.** Habría ahorrado una clase y habría hecho
que **la base** dependiera de un selector que el WebKit de un macOS anterior al 12.3 no entiende. La
forma la dice el componente; `:has()` se usa solo en la hoja de Windows, donde el motor es WebView2.

**Sin marco también en Linux.** GTK3 sin decoraciones no dibuja marco —ni sombra, ni esquinas, ni
borde de redimensionado— y Wails no expone con qué volver a decorarla: `pkg/runtime/window.go` no
tiene `WindowSetDecorated` ni da el `GtkWindow*`. Haría falta cgo propio, compilable solo en CI, y en
X11 no funcionaría. Se decide en su fase, sobre capturas y con el precio delante.

**Llevar la cuenta de si la ventana está maximizada.** También se maximiza arrastrando arriba y con
el teclado, así que una cuenta propia acabaría ofreciendo lo contrario de lo que hace. Se le pregunta
a la ventana.

**El rojo del cierre derivado de la paleta.** Es la única pieza que no es nuestra ni se deriva, y lo
es porque **ese color significa algo que la gente ya sabe leer**. El blanco encima da 5,7:1.

## Consecuencias

- **La regla de la 0064 sigue en pie sin excepciones**: el marcado es el mismo en los cuatro
  sistemas, barra de título incluida.
- **Los botones de ventana no piden ni un método nuevo** en la lista blanca del puente: las cuatro
  llamadas las exporta el runtime de Wails. Dibujar nuestra barra no abre ninguna puerta entre la
  interfaz y la aplicación.
- **`.fila-ajuste` no muere**, al contrario de lo que decía el plan: se queda siendo lo que es, la
  casilla de un formulario. Lo que se migró son los diez ajustes de Ajustes y del servidor de un
  proyecto.
- **La regla del marco está en dos idiomas** —Go y la simulación de `plataforma.ts`— porque pedir el
  aspecto de Windows desde aquí tiene que traer también su marco, o no saldría en ninguna captura.
  Lo vigila `TestElMarcoEsElMismoEnLosDosLados`, que es la salida de siempre cuando dos listas no se
  pueden juntar.
- Y lo que cuesta: **el menú de Snap no está**, así que pasar el ratón por Maximizar no ofrece las
  mitades de pantalla. En Windows 11 ese gesto lo usa mucha gente, y la tecla de Windows con las
  flechas sigue funcionando porque eso lo hace el sistema.

## Verificación

**Comprobado aquí**, con el aspecto y el marco pedidos por la URL, en los dos temas:

- **La anatomía**: la tarjeta mide 68, radio 4 y borde de 1 px, y **el control cae a la derecha del
  rótulo** —que es lo único que distingue las dos gramáticas, porque el marcado es el mismo—. La
  píldora mide 40 × 20, el pomo existe y **se mueve** al encenderla, y encendida es `--relleno` con
  `--sobre-acento` encima.
- **El marco**: la barra mide 32, los tres botones 46 × 32 y están a la derecha del nombre; la banda
  declara `--wails-draggable: drag` y **los botones `no-drag`**, que es lo que impide que pulsar
  «Cerrar» arrastre la ventana. Con un runtime de mentira puesto antes de cargar la página, **cada
  botón llama a lo suyo y el doble clic maximiza**, contando las llamadas y no el último argumento. Y
  maximizada, el botón del medio ofrece restaurar.
- **Y que donde la barra la dibuja el sistema la nuestra no existe**: ni visible, ni en el árbol de
  accesibilidad.
- **Cuatro mutaciones** las ponen rojas: quitarle a la tarjeta la colocación de Windows (el control
  vuelve a la izquierda), apagar el pomo, cambiar el alto del marco en un solo lado y borrar una
  entrada de la tabla. La primera que escribí **no valía** —cambiaba el ancho de las columnas, y las
  piezas van con su `grid-area` escrita, así que no movía nada—.

- **Las cinco pestañas de Ajustes caben en una fila con sus iconos**, medido **en Windows**, que es
  el caso que aprieta: su cuerpo es de 14 y el de la base de 13, y la columna está topada en 560 sin
  crecer con la ventana. Antes pedían 570. Los 24 px salieron de darle a Windows su propio control
  segmentado —quitar el carril y apretar el relleno de 12 a 10—, no de recortar el de todos: **se
  midió antes y después**, y de paso lo que estaba mal era que ese control siguiera siendo una
  píldora de macOS.

**No comprobado, y hace falta su Windows 11 ARM**: la tipografía, que es la mitad de parecer nativo;
Mica detrás; que la ventana arrastre, se redimensione por los bordes y conserve la sombra; y si la
densidad se lee bien al lado de una aplicación del sistema.
