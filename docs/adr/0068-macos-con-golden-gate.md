# ADR 0068 · macOS con el diseño de macOS 27 Golden Gate

Fecha: 2026-11-20 · Estado: **propuesta** — falta que el cliente elija sobre capturas · Segunda fase
de nativizar la interfaz; matiza la [ADR 0007](0007-aspecto-del-sistema.md) y la
[0019](0019-estructura-del-sistema.md)

## Contexto

Con Windows ya como una aplicación de WinUI ([ADR 0064](0064-windows-con-fluent.md),
[0066](0066-el-marco-en-windows.md)), macOS se ha quedado siendo **la base**: la cabecera de
`estilos.css` dice literalmente «aspecto de aplicación de macOS», y sus medidas están tomadas sobre
capturas de **macOS 26** usando los semáforos como regla —controles de 28, radio 9, barra lateral de
225, filas de 38—.

Esa anatomía tiene dos años. **macOS 27 Golden Gate salió el 2026-09-14** y refina Liquid Glass:
barras laterales de borde a borde, barra de herramientas unificada, radios menores, controles más
bajos y vidrio en los controles flotantes.

Y hay un antecedente que vale más que cualquier lectura de la guía: **en Apolo esto ya se hizo, y se
hizo partiendo exactamente de esta anatomía**. Su [ADR 0028](../../../apolo/docs/adr/0028-macos-con-golden-gate.md)
lo dice con estas palabras: *«En macOS, Apolo seguía con la anatomía de Esfinge de 2025: controles de
28 con radio 9, barra lateral de 225 con la fila activa en oro»*. El cliente pidió allí llevarla a
Golden Gate, eligió sobre capturas y **después corrigió dos cosas en su Mac** — el oro de los iconos
de la barra lateral y el material de la ventana—.

## Decisión

Todo cuelga de `:root[data-sistema="macos"]` (`frontend/src/sistema-macos.css`), y **el marcado no
cambia**: lo vigila `TestElAspectoNoSeDecideEnElMarcado`. La base, la extensión y los escritorios sin
hoja propia no lo ven.

**Las medidas no se vuelven a leer del kit de Figma.** Están volcadas una por una en la ficha de
Apolo, **y allí ya pasaron por un Mac de verdad**. Releer el kit desde aquí habría gastado el asiento
de Figma —que corta a unas 20 consultas— en volver al punto de partida de esas correcciones. Lo que
sí hace falta mirar en Figma es lo que Esfinge tiene y Apolo no: la lista de la bóveda, las pestañas
de clases y el formulario de una entrada.

| Tema | Decisión |
|---|---|
| Escala | Texto 13/16 con la letra del sistema, 11 para las notas, título 15. **El «grande» baja de 15 a 13**: en macOS 27 lo que sube no es el tamaño, es el peso, y dejarlo en 15 hacía que una fila de la barra lateral pesara como un título |
| Formas | Radio 6 (chico 4, grande 10), frente a 9/6/14 de macOS 26 |
| Controles | 24 de alto. Botón con relleno gris **sin borde**, sin cambio al pasar el puntero y oscureciéndose al pulsar; campo con filete de medio píxel; desplegable como botón; casilla de 16 con radio 4; interruptor de 44 × 20 con el pomo en **píldora de 26 × 16**, más ancho que alto; deslizador con pista de 6 y botón en cápsula; segmentado en cápsula con el elegido levantado en blanco |
| Barra lateral | 240 de borde a borde, filas de 36 (el «large» del kit; con el «medium» se veían pequeñas en el Mac de Apolo), icono de 20, sin hueco entre filas y **sin resalte al pasar por encima** si no está elegida |
| Parte de arriba | **Barra de herramientas unificada** de 52, con el título en Headline 13 negrita y las órdenes en **cápsulas de vidrio de 36**, que es lo que distingue una barra de macOS 27 de una fila de botones. Se abomban al pulsarlas |
| Ajustes | **Filas agrupadas** como Ajustes del Sistema: 42 de alto, 10 de relleno, rótulo a la izquierda con su nota debajo en 11, control a la derecha a 16. Las filas seguidas forman un grupo con radio 10 y un filete sangrado 10. **El mismo marcado que las tarjetas de WinUI** (`TarjetaAjuste`), con su `grid-area` escrita |
| Grupos | Radio 10, el gris de la tarjeta y **sin borde**: en macOS un grupo se separa por su superficie |
| Paleta | `MacosClaro` y `MacosOscuro` en `internal/tema`, con los neutros del kit —`Window Background`, `Fills - Vibrant`, `Labels`— y **las 56 parejas medidas**: las seis paletas pasan AA |
| El marco | **No cambia** (ADR 0019): en macOS la barra de título no existe y los semáforos los pone el sistema. La barra que Esfinge dibuja en Windows está en el marcado de los cuatro y aquí la esconde el CSS |

### Lo que se lleva a la galería, porque es de marca y no de sistema

**La fila activa de la barra lateral.** Hoy va **rellena de oro** ([ADR 0021](0021-la-marca-en-la-interfaz.md)).
En el Golden Gate de Apolo la elegida es gris con radio 8 y **el oro se va a los iconos**, en un tono
propio (`#d99a00`) que el cliente eligió entre cuatro capturas sabiendo que baja de 3:1. Son dos
lecturas distintas de la misma regla —«el oro rellena»— y **la decide él**, no esta hoja. Mientras no
elija, se queda como está.

## Alternativas descartadas

- **Quedarse en macOS 26.** Es lo que hay y no costaría nada, pero deja macOS siendo «la base con
  otro nombre» justo cuando Windows ya tiene gramática propia — y el cliente pidió nativizar los
  cuatro, no tres.
- **Volver a leer el kit de Figma entero.** Veinte consultas para llegar a lo que la ficha de Apolo
  ya lista, y además **sin** las correcciones que salieron de verlo en un Mac.
- **Mover las medidas en la base** en vez de en una hoja. Mueve los cuatro sistemas a la vez y
  rompería el panel de la extensión, que importa esos mismos tokens.

## Consecuencias

- La base deja de ser «macOS» de hecho aunque su cabecera lo siga diciendo: pasa a ser el neutro que
  ven la extensión y los escritorios sin hoja. **Arreglar esa cabecera y repartir el bloque que
  queda es un paso aparte**, apuntado en `deuda.md`.
- Las parejas de color pasan de cuatro juegos a seis, y el test de contraste las mide todas.
- El vidrio de los controles flotantes **no se puede juzgar aquí**: `backdrop-filter` se ve en el
  Chromium de las pruebas, pero el material de la ventana sólo existe en un Mac.

## Verificación

**Hecho:**

- `make contraste` mide **las seis paletas** y las dos nuevas pasan AA, con los estados derivados
  contra la peor superficie de su propia paleta —la fila elegida de la barra lateral en claro, el
  botón con el puntero encima en oscuro—.
- `make tokens` regenera los tokens con el bloque `[data-sistema="macos"]` en sus tres selectores, y
  `navegador/src/tokens.css` **no cambia**: la extensión sigue viendo sólo la base.
- `make capturas-macos` retrata el recorrido entero en los dos temas, con vidrio y sin él.

**Lo que falta, y se dice por su nombre:**

- **Que el cliente elija** sobre esas capturas, y en particular el tratamiento de la barra lateral.
  Por eso esta ficha está en «propuesta» y no en «aceptada».
- **`macos.spec.ts`**, que mida la anatomía —alto de control y de fila, radios, el ancho de la barra
  lateral, la píldora del interruptor y la agrupación de las filas de ajuste— como `windows.spec.ts`
  mide la de Fluent.
- **Un Mac.** El material de la ventana, si el pomo del interruptor sale píldora o elipse, y si las
  cápsulas de la barra se leen como vidrio o como botones grises. Nada de eso se ve desde este VPS.
