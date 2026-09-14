# ADR 0029 — El panel de la extensión es Esfinge

**Fecha:** 2026-09-14 · **Estado:** aceptada · **Extiende la [0021](0021-la-marca-en-la-interfaz.md)** ·
**Revisar cuando** se vea en un Mac, y al preparar las fichas de las tiendas

## Contexto

El panel de la extensión salió en la 2.17.0 **sin ninguna pasada de diseño**: seguía al navegador
con `color-scheme: light dark`, usaba la tipografía del sistema y tenía el CSS mínimo para que las
filas no se pisaran. Nadie lo había mirado en una captura hasta el 14 de septiembre, y lo que salió
al mirarlo no era de gusto:

- **El correo de cada cuenta se cortaba en «info@w…»**, que es lo único que distingue dos cuentas
  del mismo sitio.
- **Tres botones con rótulo en 320 px** envolvían según lo largo que fuera el título, así que cada
  fila tenía una forma distinta.
- **Los errores eran un párrafo** que mezclaba lo que hay que hacer con lo que dijo el navegador.
- **«Código» salía en todas las cuentas**, tuvieran o no segundo factor, porque el protocolo no lo
  decía.
- Y **la extensión no declaraba ningún icono**: en la barra del navegador salía la inicial genérica.

Había además una pregunta abierta desde el cierre anterior: **si el panel lleva marca**. La
[ADR 0021](0021-la-marca-en-la-interfaz.md) dice que la identidad va dentro de la ventana y
**nunca en las pantallas de trabajo**, y este panel es las dos cosas: sitio de trabajo —se abre para
rellenar y se cierra en tres segundos— y lo único de Esfinge que se ve dentro del navegador.

## Decisión

**El panel es la misma Esfinge en otro sitio, y la marca va solo donde la ventana la pone.**

### Los tokens de la ventana, importados tal cual

`panel.css` importa `frontend/src/tokens.css`, que genera `internal/tema`. Los mismos espaciados,
radios, tamaños de letra y colores. **No se inventa ni una escala ni una pareja de color**, y eso es
lo que hace innecesario medir nada nuevo: cada combinación que usa el panel es una de las que
`contraste_test.go` ya mide, y al lado de cada regla se dice cuál.

La regla que sale de ahí y hay que respetar al tocarlo: **una pareja que no esté en ese test no se
usa en el panel**. El caso que casi se cuela: el texto apagado sobre la superficie elevada no está
medido, así que al pasar el puntero por una fila el usuario sube a `--cuerpo`, que sí lo está.

### La marca, en la cabecera y en ningún otro sitio

La esfinge a trazo y el nombre, como el lockup de la barra lateral —importada **en crudo desde
`build/marca.svg`**, sin una segunda copia del dibujo—, y el sitio de la pestaña debajo. Es el
equivalente exacto de lo que la ventana hace: marca en el marco, no en el trabajo.

El oro, **solo como relleno del botón «Rellenar»**, con la piedra encima, como los botones primarios
de la ventana. En ningún otro sitio: ni en el foco —que es del acento, porque el oro como línea sobre
fondo claro da 1,37-1,68:1— ni en los iconos.

### Una forma de fila, y la acción principal con rótulo

Cada cuenta: título y usuario a la izquierda, **siempre en dos líneas** —si falta, «Sin usuario»—, y
a la derecha «Rellenar» con rótulo y dos botones de icono para copiar la contraseña y el código. Los
iconos llevan su nombre en `aria-label` y en `title`. Donde no hay código queda **un hueco del mismo
ancho**, para que el «Rellenar» de todas las filas caiga en la misma columna.

### Estados con jerarquía

Cuando no hay lista, un bloque con glifo, título, lo que hay que hacer y —aparte, pequeño y en
monoespaciada— lo que dijo el navegador tal cual. Y el resultado de cada gesto en una línea propia
debajo de la lista, con su marca de bien o mal.

### Y el protocolo dice si hay código

`Cuenta` gana `tieneCodigo`. **Solo el hecho**: la semilla no viaja. Lo que se revela a cambio ya lo
sabe quien ve la ventana, y hace falta de todas formas para rellenar el código solo, que es lo
siguiente.

### La segunda pasada (2.20.0)

Con el panel usado en el Mac, el cliente pidió que no pareciera tan básico y eligió estas mejoras entre
las que se le propusieron:

- **Un cuadro con la inicial en cada cuenta**, el mismo de la lista de la bóveda: `inicialDe`,
  `tinteDe` y `dominioDe` salen de `componentes.tsx` a `frontend/src/monograma.ts`, sin React, y los
  usan la ventana y el panel. **El color sale del sitio de la pestaña**, porque al panel no le llega
  el sitio guardado de cada entrada: coincide con la ventana salvo si una entrada se guardó con otro
  subdominio.
- **El icono de la web junto a su dirección, sin salir a internet**, como pide la ADR 0024: en Chrome,
  la dirección `_favicon` de la propia extensión, que lo sirve desde su caché y exige el permiso
  `favicon`; en Firefox, solo si ya viene incrustado. Si no, el cuadro con la inicial del sitio.
- **«● Abierta»** en la cabecera cuando la bóveda contesta.
- **«✓ Hecho» en la fila** al rellenar, con el botón girando mientras espera.
- **La esfinge grande y tenue detrás de los avisos**, como la del historial vacío. Solo está medida
  sobre el lienzo, así que **los avisos dejan de ir sobre tarjeta**.
- **La cuenta atrás del código** al copiarlo: un anillo que se vacía y los segundos al lado.
- **La firma «▍ webcafeína»** al pie, junto a la versión.
- **Atajos**: la primera cuenta con el foco al abrir, ↑ y ↓ entre cuentas, Intro rellena la de foco y
  Esc cierra.

Y el panel pasa de 340 a **360 px** de ancho, porque el cuadro le quita espacio al correo.

**Corregido en la 2.20.1: el marco de la fila, solo con el teclado.** En la 2.20.0 la primera cuenta
recibía el foco al abrir el panel, y el navegador le pintaba el marco sin que nadie hubiera tocado
nada: se leía como una cuenta seleccionada. Ahora nada tiene el foco al abrir; el marco aparece con la
primera flecha o el tabulador y se va con el ratón, e **Intro sin haber usado las flechas rellena la
primera cuenta**, que era para lo que servía aquel foco.

**Corregido en la 2.20.3, vistas las dos en Firefox y en Chrome:**

- **La esfinge tenue de los avisos, entera y con su trazo de serie.** Salía desplazada fuera de su
  caja, que la recortaba por abajo, y con el trazo forzado a 34 sobre el grupo: líneas finas al lado
  de unos ojos y un ojal rellenos, gordos, que se leían como un dibujo mal construido. Ahora mide
  112 px como la del historial vacío de la ventana, con su trazo de 58, entera dentro de la caja; y
  **el texto del aviso tiene reservado su ancho**, porque con la esfinge entera el texto largo le
  pasaba por encima.
- **«Rellenar» con la misma letra en los dos navegadores.** En Chrome de macOS salía más pequeño que
  en Firefox. Los botones ya no heredan la letra: llevan familia, tamaño e interlineado fijados y sin
  aspecto nativo.

## Alternativas descartadas

**Un panel neutro, sin marca**, siguiendo al navegador como hasta ahora. Era lo más fácil de defender
leyendo la ADR 0021 al pie de la letra, y es como estaba: una página suelta que no se reconocía como
Esfinge y en la que había que inventar cada decisión de espaciado. La 0021 no prohíbe la marca en las
superficies de Esfinge; la pone en el marco y la quita del trabajo, y eso es lo que se hace aquí.

**Una paleta propia del panel.** Habría que medirla aparte y mantener dos fuentes de verdad del color.
Con la de la ventana ya medida, no hay nada que ganar.

**Tres botones con rótulo.** Es lo más legible botón a botón y lo que había. En 340 px, lo que se
sacrificaba para que cupieran era el correo, que es justo lo que no puede faltar.

**Que pulsar la fila entera rellene**, como hacen algunos gestores. Ahorra un botón, pero esconde la
acción principal detrás de un gesto que hay que adivinar, y un clic por descuido escribe en el
formulario.

## Consecuencias

- **El panel depende de `frontend/src/tokens.css`.** Si la ventana cambia un token, el panel cambia
  con ella, que es lo que se quiere; y si alguien añade una pareja nueva en el panel, tiene que
  añadirla antes a `contraste_test.go`.
- **340 px en vez de 320.** Los navegadores dejan hasta 800.
- **Los iconos de la extensión se generan** con `make icono` desde `build/icono.svg`, como todos los
  demás, y viven en `navegador/iconos/`. El empaquetador **falla si el manifiesto nombra uno que no
  está**, porque un icono que falta no da error en el navegador: pone la inicial genérica.

## Verificación

**Comprobado aquí:**

- **Capturas de cada estado en claro y en oscuro** (`pnpm run capturas`, que deja
  `navegador/capturas/`): una cuenta, varias, ninguna, cargando, bóveda cerrada, sin permiso, sin
  Esfinge, origen no válido y el resultado tras rellenar. De mirarlas salió un fallo que ninguna
  prueba habría visto: el «Rellenar» de la fila sin código caía desplazado.
- **Seis pruebas del panel compilado** (`navegador/pruebas/panel.spec.ts`) con una `chrome` de
  mentira: una fila por cuenta y el código solo donde lo hay; los botones alineados, medido; un
  título con HTML dentro se escribe como texto; la instrucción y el detalle del navegador van aparte;
  y **el fallo de la 2.18.0 reproducido** —una trama dice que no antes que la buena— con el sí
  ganando.
- **`tieneCodigo` sin la semilla**, en Go, mirando el JSON. Y aquí la prueba cazó un fallo mío
  antes de publicar: `Buscar` devuelve las entradas pasadas por `SinSecretos`, que vacía la semilla
  sin dejar marca, así que todas las cuentas salían sin código.

**Y usado en un Mac (2026-09-14, 2.19.0 y 2.19.1), en Firefox y en Chrome**: el cliente lo ha
probado de punta a punta y dice que funciona todo. **No ha comentado nada en concreto** de las dos
cosas de abajo, así que siguen apuntadas como sin comprobar en vez de darlas por buenas.

**Comprobado aquí en la segunda pasada (2.20.0):** capturas de todos los estados en los dos temas, y
ocho pruebas más del panel: el tinte del cuadro coincide con `tinteDe` para el sitio; **el icono de
la web nunca se pide a internet** —se comprueba que no sale ninguna petición—; «Abierta» solo con la
bóveda abierta; «✓ Hecho» y su vuelta; el teclado; la esfinge tenue y la firma.

**Sin comprobar:**

- **Cómo se ve dentro de un navegador de verdad, en un Mac.** Las capturas son del panel compilado en
  Chromium, con la tipografía de esta máquina: en macOS la letra es San Francisco y los anchos
  cambian. Lo primero que hay que mirar allí es si el correo sigue cabiendo.
- **El icono de 16 px en la barra**, que a ese tamaño puede no leerse.
- **La extensión cargada de verdad**: estas pruebas mueven el panel, no el guion de página ni el
  trabajador de fondo. Eso sigue en `docs/deuda.md`.
