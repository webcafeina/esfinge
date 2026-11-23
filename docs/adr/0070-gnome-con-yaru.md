# ADR 0070 · GNOME se ve como GNOME: libadwaita con Yaru

**Fecha**: 2026-11-23
**Estado**: aceptada
**Fase**: F4 del plan de nativizar la interfaz, después de Windows (0064, 0065, 0066) y macOS

## Contexto

Hasta aquí, Linux era **macOS con ~55 líneas de parches**: la cabecera centrada,
las filas de 36, los radios a 12 y poco más. La ADR 0020 lo dejó dicho —«en KDE
se verá correcta aunque no de la casa»— y en GNOME pasaba lo mismo con otro
nombre: se reconocía como «una aplicación», no como una aplicación de Ubuntu.

Windows (0064) y macOS (0066) ya tienen su paleta, su hoja, su juego de iconos y
su gramática de formularios. Esto es lo mismo para GNOME, y con ello **la base
deja de ser macOS y pasa a ser la base neutra**: la que ven el panel de la
extensión y un escritorio que no reconocemos.

El cliente lo pidió el 2026-11-19 con el antecedente de Apolo, donde ya se hizo
entero, y con una condición que ha vuelto a valer aquí: *«revisa que todos los
elementos vayan acorde al estándar más reciente, para evitar más errores de este
tipo»*.

## Decisión

**GNOME se viste con libadwaita 1.10 y el tema Yaru**, que es lo que trae Ubuntu
26.10. Ubuntu no cambia ninguna medida de libadwaita, sólo los colores, así que
las medidas son las de libadwaita y los colores los de Yaru.

Cinco piezas:

1. **Paleta** `LinuxClaro` / `LinuxOscuro` en `internal/tema/paleta.go`, con los
   neutros de Yaru y sus colores con significado —el verde de Canonical, no el de
   Apple—. Salen en `tokens.css` bajo `[data-sistema="linux"]`. Las **~140
   parejas de cada tema están medidas**: `make contraste` recorre los ocho temas.
2. **Hoja** `frontend/src/sistema-linux.css`, importada detrás de la de macOS y
   delante de la futura de KDE. El bloque de Linux que quedaba en `estilos.css`
   **se repartió, no se borró**.
3. **Iconos** de Yaru (`frontend/src/iconos-yaru.ts`), 21, cosechados por
   `herramientas/iconos/yaru.py` desde un commit fijado.
4. **Anatomía**: la lista en tarjeta de Ubuntu para los ajustes, la AdwHeaderBar
   con el título centrado, el AdwToggleGroup para el segmentado y el
   AdwPreferencesPage con su AdwClamp de 600.
5. **Pruebas** `frontend/e2e/gnome.spec.ts`, que miden, y
   `TestLosIconosDeYaruNoSeHanAdaptado`, que respalda la licencia.

**Esta hoja viste GNOME y todo el Linux que no es KDE** —Xfce, Cinnamon,
COSMIC—: son escritorios de GTK y lo que se parece a una aplicación suya es
libadwaita. KDE refina encima, en su fase.

### Lo que no se cambia sin preguntar

- **La escala va en puntos** —11 pt el cuerpo, 9 pt el chico—, no en píxeles. En
  GNOME el tamaño de la interfaz lo pone el usuario y el escritorio lo escala:
  escrito en píxeles, Esfinge sería la única ventana que no crece cuando alguien
  sube la letra del sistema. Lo vigila una prueba que mide los `14.6667px`.
- **El marco lo dibuja el sistema.** La barra de título de GTK, con su sombra y
  sus esquinas. La barra que Esfinge dibuja en Windows está en el marcado de los
  cuatro sistemas y aquí **la esconde el CSS**, que es la regla de la 0064.
- **El oro sólo en el icono de la fila elegida.** En GNOME una barra lateral no
  se colorea: lo que dice «estás aquí» es el gris. El acento de marca se queda en
  el icono (ADR 0021) y en el botón de acción.
- **Los iconos no se adaptan.** Ni se juntan trazados, ni se recortan, ni se
  reescalan. Es lo que permite llevar obra CC BY-SA dentro de un programa que no
  es código abierto, y lo que lo sostiene es una prueba, no una promesa.

### Lo que eligió el cliente sobre capturas (2026-11-23)

Tres cosas se le presentaron con sus opciones dibujadas, que es como se decide el
aspecto en este proyecto.

**Las pestañas de Ajustes van en una fila, con el glifo encima del rótulo.** Son
seis y a 11 pt con el glifo al lado piden **749 px**; la columna de una página de
preferencias son los **600** del AdwClamp y la zona de contenido da **736** con
la ventana por defecto, así que salían **dos filas de tres** y eso fue lo
primero que preguntó. Se le dieron tres salidas: dejarlo en dos filas, dejar que
la fila se salga del ancho de la página, o **subir el glifo**, que es la forma
estrecha del AdwViewSwitcher —la de la barra de abajo de las ventanas de GNOME—.
Eligió la tercera, y descartó a propósito tocar la ventana: cabría subiendo el
ancho por defecto de 980 a 1000, pero para que fuese **siempre** un renglón
habría que subir el **mínimo** a ~1000, y eso impide poner Esfinge en media
pantalla de un monitor de 1920 —960 justos—. Con el glifo encima caben sin tocar
nada, y **con la ventana en su mínimo se parten en tres y tres**, no en cinco y
una.

De ahí salió además una corrección que no era de medidas: **los rótulos del
conmutador pierden la negrita**. En libadwaita la negrita es de los botones, y un
conmutador de vistas no es un grupo de botones sino navegación —como las filas de
la barra lateral, que por lo mismo tampoco la llevan—. Los demás segmentados de
Esfinge sí son grupos de botones y se quedan en negrita.

**El icono de «Generar» es `star-new`**, una estrella con un más. Yaru no tiene
ninguna chispa —se buscó y no existe—, así que había que elegir metáfora y se
eligió sobre una rejilla de seis candidatos dibujados. Lo primero que hubo fue
`view-refresh`, la flecha de recargar, y su problema es que dice «otra vez» y se
lee como «recargar la página».

**Y los grupos de Ajustes se quedan sin glifo**, que era lo que estaba en duda:
en Ubuntu un grupo de preferencias es un título a secas y debajo su lista, y el
icono junto al título es de la tarjeta de Windows. Se le enseñó el antes y el
después y el argumento en contra —el glifo ayuda a localizar el bloque cuando hay
varios en la misma pestaña—, y eligió el estándar.

## Alternativas descartadas

**Seguir con macOS parcheado.** Es lo que había, y el coste no es estético: cada
parche nuevo se escribía contra una base que decía ser otra cosa, y el comentario
que lo encabezaba —«lo de arriba es macOS»— llevaba dos fases siendo falso.

**Un solo aspecto para todo Linux.** Habría sido más barato y deja KDE con la
cabecera centrada de GNOME, que allí no existe. El cliente eligió los cuatro
aspectos.

**Dibujar nosotros los iconos de GNOME.** Se descartó en el plan: lo que hace que
un escritorio se reconozca es su propio juego, y un parecido razonable no lo
reconoce nadie. El coste es la atribución, que está en `TERCEROS.md` y en
Ajustes.

**Juntar los trazados de cada icono en una sola `d`.** Es lo cómodo y es lo que
en Apolo dejó a Ubuntu sin media montaña durante cinco versiones; aquí además
sería **adaptar**, con lo que eso arrastra en CC BY-SA.

## Consecuencias

- Ocho paletas medidas en vez de seis: `make contraste` pasa de ~700 a ~950
  comprobaciones y sigue tardando centésimas.
- `TERCEROS.md` es nuevo, y la pestaña de Acerca de dice de quién son los iconos.
- Los juegos de Fluent y de Yaru tienen ahora **los mismos 21 nombres**: al
  cosechar los de Yaru se vio que `texto`, `ficheros` y `tema` no estaban en
  ninguno de los dos —nacieron después de la fase de Windows—, así que en Windows
  esas filas mezclaban iconos del escritorio con dibujados a mano.
- **Lo que falta por ver en una máquina**: todo esto se ha decidido sobre
  capturas. En la Ubuntu del cliente hay que mirar el marco de GTK alrededor, la
  letra real —aquí no está Ubuntu Sans, así que las capturas salen con la letra
  de respaldo— y si los iconos se reconocen a su tamaño de verdad.

## Verificación

**Lo que se comprobó.** `make comprobar`, `make e2e` y las once pruebas de
`gnome.spec.ts`, que miden la escala en puntos, los controles de 34, los radios,
la barra lateral de 220, la fila elegida contra el token, el botón de plegar
dentro de su banda y sin tocar la primera fila, la cabecera de 46 con el título
centrado de verdad, la lista en tarjeta tocándose, el interruptor con su pomo, el
segmentado por `aria-selected`, el reparto de las pestañas sin fila coja, que la
regla general de botón no pisa a los botones compuestos, y que el dibujo que
llega al DOM es el del módulo **con su transformación**.

**Cada una mutada para verla roja.** Ocho mutaciones; siete cayeron a la primera
y **una se quedó verde**: la de los iconos comparaba el DOM con el módulo, o sea
el mismo fichero consigo mismo. De ahí salieron las dos correcciones que hacen
que esa familia valga algo: la prueba de interfaz mide ahora un icono **que lleva
grupo y transformación** —con `cifrar`, que es un trazado suelto, pasaba en verde
aunque se dejaran de pintar los hijos— y la promesa de la licencia la respalda
una prueba aparte, en Go, contra **los SVG de origen guardados en
`internal/tema/testdata/yaru`**.

**Y lo que no encontró ninguna prueba, sino mirar las capturas**: el botón de
plegar montado encima de «Cifrar» y comiéndose sus clics; el interruptor sin
pomo, porque la base lo deja a `display: none` y lo enciende cada hoja; la lista
de ajustes partida en dos tarjetas por el hueco del bloque; la fila del proyecto
corrida, porque la regla general de botón pesaba más que `.abrir-proyecto`; el
título descentrado 19 px por un relleno asimétrico; y tres iconos que a 16 px
parecían distintos y en una rejilla a 48 eran la misma página. **Seis de siete
arreglos de esta fase salieron de mirar, no de medir.**

**Lo que no se ha comprobado**: nada de esto se ha visto en una Ubuntu. El marco
de GTK, la tipografía de verdad y el tamaño real de los iconos sólo se pueden
mirar allí.
