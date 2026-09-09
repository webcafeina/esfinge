# ADR 0021 — La marca entra en la ventana, y el acento es el oro de la esfinge

**Fecha:** 2026-09-09 · **Estado:** aceptada · **Matiza la [0007](0007-aspecto-del-sistema.md)** ·
**Revisar si** el cliente pide separar la identidad del producto de la de la casa

## Contexto

Hasta la 2.10.4, la ventana **no tenía una sola marca en ningún píxel**: ni logotipo, ni nombre de
producto, ni «Webcafeína». No era un olvido. La [0007](0007-aspecto-del-sistema.md) lo decidió así
después de que la 1.x vistiera Esfinge con la identidad de ClickHouse: en un terminal una identidad
fuerte ayuda, en una ventana con convenciones hace que la aplicación se sienta ajena. La frase era
«la marca queda en el icono y en *Acerca de*».

Dos cosas hicieron falta para volver sobre ella.

**La primera es que ese «Acerca de» nunca se construyó.** `menu.go` cablea «Acerca de Esfinge» a
`OrdenIrAAjustes` —navega a Ajustes, no abre ningún diálogo— y Ajustes empezaba con «Tienes
instalada la versión X»: sin nombre, sin autoría, sin logotipo. La mitad de la promesa de la 0007
llevaba desde la 2.0.0 sin cumplirse, y nadie lo había notado porque estaba escrita como si ya
existiera.

**La segunda es que el cliente lo pidió**, con una condición que manda sobre todo lo demás: «de una
forma orgánica, bonita y no porque sí».

## Decisión

**La marca vive en el chrome y en los vacíos. Nunca en las pantallas de trabajo.**

- **Barra lateral**: el lockup —la esfinge y «Esfinge»— debajo del hueco de los semáforos, y la
  firma `▍ webcafeína` al fondo del todo. Ninguna de las dos es interactiva.
- **Historial vacío**: la esfinge grande y tenue, teñida con `--filete`. Es el único sitio donde la
  marca se permite ser grande, y solo porque ahí no hay nada que estorbar. Con una entrada,
  desaparece.
- **Ajustes**: la ficha de producto que la 0007 prometía. Sustituye a la línea de la versión.
- **Nada en Cifrar, Descifrar ni Generar.** Ahí se trabaja.

**Y el acento de la interfaz pasa a ser el oro del tocado**, `#f2c14e`, en lugar del azul del
sistema. Ésta es la parte que de verdad matiza la 0007, y por eso está aquí y no colada en un commit.

**La tipografía no se toca.** Sigue siendo la del sistema en los tres. Es la mitad de lo que hace que
una aplicación parezca nativa y cambiarla se nota más que ninguna otra cosa.

### La regla: el oro rellena, la piedra escribe

No es una preferencia. Sale de medir, antes de dibujar nada:

| pareja | ratio | |
|---|---|---|
| blanco sobre el oro `#f2c14e` | **1,68:1** | imposible |
| oro oscurecido hasta que el blanco cumple | `#9a6c1b` | ya es marrón: se pierde la marca |
| **piedra `#2b2b31` sobre el oro** | **8,38:1** | cómodo |
| *referencia*: blanco sobre el azul que había | 4,65:1 | |

Con el blanco descartado, lo que se movió fue la tinta y no el fondo: `SobreAcento` deja de ser
blanco y pasa a ser la piedra. Y resulta que eso **es la gramática del propio icono** —tocado dorado
sobre placa oscura—, así que la interfaz acabó hablando el idioma que el dibujo ya hablaba. De ahí
que la decisión se sienta orgánica: no se le puso una identidad encima, se le sacó la que tenía.

La consecuencia es que `Acento` —el rol de *texto*— deja de ser «el acento en versión legible». Es la
tinta fuerte del tema: la piedra en claro, el blanco en oscuro. **En oscuro el oro sí se leería**
—9,93:1 sobre el lienzo— y aun así no se usa: una regla que vale en un tema y no en el otro deja de
ser una regla.

### Y la trampa que casi se cuela

`--relleno` se usaba para dos cosas que el azul cumplía y el oro no: **rellenar superficies con texto
encima**, y **dibujar líneas e indicadores finos sobre fondo claro**. Lo segundo pide 3:1 y el oro da
entre **1,37 y 1,68:1** según la superficie. Cuatro sitios tuvieron que pasar a `--acento`: el filete
de foco, el borde de la zona de soltar, el relleno de la barra de progreso y la barra de acento de la
fila activa en Windows.

**Lo grave es cómo se habría colado**: `make contraste` mide **parejas de tokens, no sitios**. Habría
pasado en verde con el foco de los campos invisible en tema claro. Lo vigila ahora una prueba de
interfaz, no el test de color.

En el foco quedan las dos cosas: el filete de piedra dice dónde estás, y el halo dorado es la marca.
Un halo puede permitirse no cumplir 3:1 porque no es él quien informa.

## Alternativas descartadas

- **Una cabecera de marca en la barra de herramientas.** Era el hueco fácil —la mitad derecha está
  vacía— y habría metido el logotipo en todas las pantallas, incluidas aquellas donde lo que importa
  es un secreto en claro. La marca no tiene por qué verse mientras se trabaja.
- **Mantener el azul del sistema y limitar el oro a un logotipo.** Es tenerlo a medias: una ventana
  azul con un sello dorado en la esquina no es identidad, es una pegatina.
- **Un bronce para el texto de acento**, oscureciendo el oro hasta que se lea (`#9a6c1b`). Conviven
  entonces dos oros distintos, el que rellena y el que escribe, y la ventana se ensucia. Se descartó
  con el cliente delante.
- **Usar el icono de aplicación dentro de la ventana.** Lleva su placa y sus colores dentro, así que
  no se puede teñir; y en miniatura se lee como un pegote. De ahí `build/marca.svg`, que es otra
  pieza y no una copia.
- **Una silueta maciza** para esa marca. Probada y desechada mirándola: en monocromo el rostro y el
  tocado son del mismo color, se funden y sale un borrón. A trazo se leen los dos, y además rima con
  los iconos de la barra lateral, que ya son de trazo (0019).

## Consecuencias

- **`Relleno`, `RellenoVivo` y `SobreAcento` son ahora idénticos en los dos temas.** Lo único que
  cambia entre claro y oscuro es `Acento`. Con el azul no era así, y es deliberado: el icono del Dock
  no cambia con la apariencia del sistema, así que el acento tampoco.
- **El botón discreto va subrayado.** Antes su texto era azul y el color solo decía «esto se pulsa»;
  con la tinta como acento, «Generar una» se leía como una etiqueta. El subrayado devuelve la señal
  sin gastar color, que es lo único que aquí no se puede gastar.
- **En Windows la fila activa no se pinta de oro.** Fluent usa fondo tenue y barra de acento, y eso
  se respeta (0020). El oro aparece allí en el botón de acción y en la ficha, no en la navegación.
- **`RellenoLegible` se queda sin trabajo** con esta paleta: el oro con la piedra encima ya cumple y
  la función lo devuelve intacto. No se borra —es la red por si el acento vuelve a cambiar— y hay un
  test que comprueba justamente que **no** lo toque.
- **Los azules del sistema siguen en `paleta.go`**, sin usarse. Son el porqué de `RellenoLegible`, y
  borrarlos dejaría a esa función pareciendo un adorno.
- El ámbar de los avisos y el oro de la marca **son vecinos**. Conviven en la pantalla de cifrar. No
  se ha tocado: los estados siguen siendo los del sistema, y hay que mirarlo puesto.

## Lo que cambió al verlo puesto (2.11.1)

Seis cosas, y ninguna se habría visto sin abrir la ventana. Van aquí porque son
la misma decisión afinada, no otra.

- **El lockup se confundía con el menú.** El nombre usaba `--texto-grande`, que
  es *exactamente* el tamaño de las filas de navegación: solo cambiaba el peso.
  Pasa al tamaño de título, con la esfinge más grande y más aire debajo.
- **La firma lleva la versión**: `Webcafeína ▍ 2.11.0`. El glifo de la casa pasa
  de abrir la marca a separar el nombre del número. La versión estaba solo en
  Ajustes, que es un sitio al que hay que ir, y es lo primero que se pregunta
  cuando algo va raro. La misma pieza se usa en la ficha de Ajustes.
- **La fila de la barra lateral se hunde al pasar por encima, no se levanta.**
  Usaba `BotonEncima`, que es la respuesta correcta para un botón suelto y la
  equivocada para una barra lateral: en claro salía casi blanca sobre una barra
  que ya es casi blanca, y en oscuro directamente aclaraba. De ahí `BarraEncima`,
  un color propio y no un reciclado.
- **La casilla de Ajustes se dibuja entera.** La del navegador se pinta con el
  acento del sistema —azul— y no hay forma de que respete el nuestro; y con
  `accent-color` tampoco valdría, porque el navegador dibuja el tic en blanco y
  eso son 1,68:1 sobre el oro. Dibujada, el tic es de piedra y la casilla mide
  18 px en vez de la miniatura de serie.
- **La silueta tenía una raya cruzándole la cara.** Es el tramo `h246` del
  contorno del tocado, a la altura y=478: relleno queda tapado por el rostro, y a
  trazo se dibuja. Se sustituye por un salto, y el contorno queda en dos trazos
  abiertos que se encuentran en la punta.
- **Y el ojal tocaba la barbilla.** El rostro acaba en y=742 y el ojal empezaba
  en 754: doce de separación, que relleno se leen y a trazo no, porque medio
  grosor ya son 29. Bajado 40, y el `viewBox` ajustado al dibujo para que la
  marca ocupe la caja que se le da.

## Verificación

- **`make contraste`** con ocho parejas nuevas, entre ellas la fila activa, la firma, la barra de
  progreso y el borde de la zona de soltar. Todas pasan.
- Tests de Go reescritos: `TestSobreElOroEscribeLaPiedra` afirma que **el blanco sobre el oro no
  llega**, que es el supuesto que sostiene la decisión; y `TestElAzulDelSistemaSeguiriaSinCumplir`
  conserva el porqué de `RellenoLegible` ahora que el azul ya no se usa.
- **Seis pruebas de interfaz nuevas**, en los dos temas: que la marca está y **que la barra lateral
  sigue teniendo cinco botones** —si el lockup fuera interactivo rompería el localizador de media
  suite—; que la firma queda debajo de Ajustes; que sobre el oro se escribe con tinta oscura y no
  blanca; que **el foco de un campo cambia de color al enfocarlo**, que es la red contra la trampa de
  arriba; que el historial vacío enseña la esfinge y deja de enseñarla al haber una entrada; y que
  Ajustes dice qué es esto, de qué versión y de quién.
- La marca se miró a 18, 20, 24, 48, 96 y 112 px antes de elegir el grosor del trazo, y la ventana
  entera en los dos temas antes de dar nada por bueno.

**Lo que no se ha comprobado:** cómo queda en un Mac y en un Windows de verdad. Y en concreto **la
pastilla dorada de la fila activa**, que es el cambio más visible de todos: cumple de sobra y es la
gramática del icono, pero es lo primero que hay que juzgar delante de la ventana.
