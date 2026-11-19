# ADR 0064 · Windows con Fluent

Fecha: 2026-11-19 · Estado: aceptada · Primera fase de nativizar la interfaz; matiza la
[ADR 0007](0007-aspecto-del-sistema.md) y la [0020](0020-windows-y-gnome.md)

## Contexto

Hasta hoy Esfinge se veía **igual en los tres sistemas salvo nueve detalles**: todo el CSS por
sistema eran ~140 líneas al final de `estilos.css`, y su propio comentario decía la verdad del
asunto —*«lo de arriba es macOS, que es donde se juzga esto; aquí van las diferencias»*—. O sea que
Windows y Linux eran macOS con parches.

Lo pidió el cliente el 2026-11-19 con el antecedente de **Apolo**, donde la nativización se hizo
entera entre octubre y noviembre: una fase por sistema, con su ficha y su versión publicada, los
juegos de iconos de cada escritorio y las decisiones tomadas **sobre galerías de capturas**. Y con
una lección que conviene tener delante antes de empezar: allí la primera vuelta —sólo los
controles— **no bastó**, y lo dijo él mirándolo: *«se sigue pareciendo a la que está… la estructura
visual de la ventana debe ser la nativa de Fluent/WinUI»*.

Windows va primero, y no es casual: es donde la interfaz de Esfinge es hoy más ajena, porque la base
es macOS medida con capturas de verdad. En Apolo, además, ver Windows terminado fue lo que disparó
la fase de macOS.

## Decisión

**En Windows, Esfinge habla Fluent 2 / WinUI 3: su paleta, su escala, sus formas y sus iconos.**

| Tema | Decisión |
|---|---|
| Paleta | Los neutros de **WinUI 3** aplanados sobre el gris de Mica, en dos paletas propias (`WindowsClaro`, `WindowsOscuro`) con **sus 66 parejas medidas** |
| Acento | **El oro de la marca, no el de Windows** (ADR 0021). Que la ventana se vista de Fluent no cambia de quién es |
| Escala | Body **14/20**, Caption 12, Subtitle 20; controles de **32**; radios **4 y 8**; líneas de **1 px** |
| Tipografía | **Segoe UI Variable**, con la familia óptica *Display* en los títulos. Ninguna tipografía viaja dentro de Esfinge: si no está, se cae a Segoe UI y luego a la del sistema |
| Iconos | **Fluent System Icons** (MIT), cosechados de un commit fijado, **sin juntar ni reescalar los trazados** |
| Navegación | El NavigationView: 240 de ancho, filas de 40, la activa **teñida** del acento con su **barra de 3 × 16** al lado |
| Campos | La línea de abajo marcada, que **al enfocar se vuelve del acento**: es lo más reconocible de un campo de Windows 11, y aquí sustituye al anillo de macOS |
| El marco | **No entra aquí.** Lo decide la [ADR 0066](0066-el-marco-en-windows.md) sobre capturas de las dos opciones |

Y tres cosas de método que valen para las tres fases que vienen:

- **El color se genera y las medidas se escriben.** `internal/tema` emite ahora un bloque por
  sistema —`:root[data-sistema="windows"]`— y las medidas viven en `frontend/src/sistema-windows.css`.
  Una medida no se puede medir en contraste y viene en familia con el relleno que la acompaña:
  partir eso en dos ficheros sería partir una decisión en dos.
- **Los tokens se publican en dos ficheros.** El panel de la extensión importaba los de la ventana,
  y allí `data-sistema` no existe ni puede existir: recibiría bloques que nunca puede activar,
  creciendo con cada sistema. Ahora hay `navegador/src/tokens.css` con sólo la base.
- **El marcado es el mismo en todos los sistemas.** Lo único que cambia es el CSS, y lo vigila una
  prueba que compara el árbol del DOM entre dos aspectos. El día que haya un `<div>` que sólo exista
  en un sistema, mantener cuatro aspectos deja de ser cambiar CSS y pasa a ser mantener cuatro
  interfaces.

## Alternativas descartadas

**El acento de Windows en vez del oro.** Es lo que haría una aplicación de Microsoft, y es
exactamente lo que la ADR 0021 descartó: Esfinge dejaría de reconocerse. En Apolo el cliente eligió
lo mismo.

**Copiar los hexadecimales de WinUI tal cual.** Los estados y los textos se derivan con
`AcentoLegible` contra la peor superficie **de su paleta**, y aquí se vio por qué: el divisor de
Fluent da **1,1996:1** sobre su lienzo y el suelo de esta casa para una línea de separación es 1,2.
Son cuatro diezmilésimas y da igual — lo que no se baja es el umbral. Derivándolo, se arregla solo.

**Dibujar nuestros iconos «al estilo Fluent».** Se descartó por lo de siempre: un parecido razonable
no lo reconoce nadie. Los de Fluent son los que esa persona ve en el resto de su sistema.

**Juntar los trazados de cada icono en una sola `d`**, que es lo que hace cualquier optimizador.
En Apolo eso dejó a Ubuntu sin la montaña de un icono durante cinco versiones —la `m` inicial del
segundo trazado pasa a ser relativa— y **volvió dos veces**. Aquí entran como vienen, uno por
`<path>`. De paso, no juntarlos es lo que hace que el mismo cosechador sirva para Yaru (CC BY-SA) y
Breeze (LGPL) sin adaptar nada, que es otra conversación.

**Un parámetro de tamaño en el componente del icono.** El tamaño lo decide el CSS (`--icono`);
pasarlo también por props sería la misma decisión en dos sitios.

## Consecuencias

- **El contraste pasa de 132 medidas a 264**, y seguirá creciendo: son las mismas parejas medidas en
  cada juego de neutros. El nombre del tema entra en el error, porque un fallo tiene que decir en
  qué sistema se queda corto un color.
- **`estilos.css` deja de ser macOS y pasa a ser la base neutra**: lo que ven el panel de la
  extensión y los escritorios que no tienen hoja propia. macOS se mudará a la suya en su fase.
- **Hay una dependencia nueva, y es de datos, no de código**: los trazados de Fluent, con su licencia
  MIT íntegra en la cabecera del módulo. No se descarga nada al compilar; lo que se publica es el
  fichero cosechado.
- **El filete de Windows queda un punto más marcado que el de Fluent** (`#d3d3d3` en vez de
  `#e5e5e5`), porque la derivación va a pasos del 8 % y se pasa del suelo. Está en `deuda.md`.
- Y lo que hay que decir en voz alta: **la tipografía no se puede juzgar desde aquí.** En este
  servidor no hay Segoe UI, así que las capturas enseñan las formas, las densidades y los colores,
  pero no la letra. Eso sólo se ve en Windows.

## Verificación

**Comprobado aquí**, con el aspecto pedido por la URL (`?sistema=windows`), en los dos temas:

- **264 parejas de contraste**, las 66 de cada tema por las cuatro paletas.
- **Seis pruebas que miden**, no que opinan: la escala (14/20), los controles (32 de alto, radio 4),
  la tarjeta (radio 8), el panel (240 de ancho, filas de 40), la fila activa **comparada contra el
  token** —no contra un hexadecimal escrito a mano— con su barra de 3 × 16, la identidad del trazado
  de cada icono contra el módulo cosechado, y que el botón de acción sigue siendo oro con piedra
  encima.
- **Y que el marcado es el mismo** que en la base, mirando el árbol del DOM sin entrar en los `<svg>`
  —ahí es distinto a propósito, porque los iconos son otros—.
- Dos mutaciones las ponen rojas: quitarle a Windows su altura de control (dice 28) y que el
  componente deje de mirar el sistema (sale el trazado dibujado de la casa).

**No comprobado, y hace falta su máquina**: la tipografía —que es la mitad de parecer nativo—, Mica
detrás de la ventana, y si la densidad se lee bien al lado de una aplicación de Windows. Eso se ve
en la VM de Windows 11 ARM del cliente, con la versión publicada.
