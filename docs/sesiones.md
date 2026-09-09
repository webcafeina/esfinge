# Sesiones

Bitácora. Una entrada por sesión, la más reciente arriba. Sirve para retomar exactamente donde se
dejó aunque se pierda la conversación.

Plantilla al final.

## 2026-09-09 · Y aun arreglado, no aparecía ninguno

- **2.14.2.** El cliente actualizó a la 2.14.1 —con el filtro ya corregido— y seguía sin ver un solo
  icono. Segundo fallo, distinto del primero y también mío.
- **El goteo guardaba y avisaba solo al terminar la tanda entera.** Diez segundos de espera más
  veinte entre cada uno son **casi cuatro minutos** antes de que se guardara nada, y quien cerraba la
  bóveda antes no se llevaba ninguno **ni siquiera de los ya descargados**. Ahora se guarda y se
  avisa **uno a uno**: cada icono que llega se queda y se ve.
- Y el ritmo, más razonable: tres segundos para empezar y cinco entre cada uno, veinticinco por
  sesión. Sigue sin ser una ráfaga —que es lo que la ADR quiere evitar— pero se ve avanzar.
- **Lo que de verdad hay que aprender de estos dos fallos seguidos:** las piezas estaban probadas una
  a una y **la tubería no lo estaba**. El descargador tenía ocho pruebas, el almacén cuatro, los
  frenos tres… y no había ni una que fuera de la bóveda al almacén pasando por la red. Ahora la hay,
  con su costura para inyectar un descargador y acortar los tiempos —sin eso costaría minutos y no se
  ejecutaría nunca—.
- De escribirla salieron dos cosas más: las pruebas del descargador usaban un servidor **en claro** y
  un ayudante que repetía lo que hace `De()` con otro esquema, así que **lo que se probaba era el
  ayudante**; ahora son servidores con TLS y se llama al camino de verdad. Y todo lo que devuelve el
  descargador es ya un `ErrNoHay` con el detalle detrás: para quien llama, «404», «no era una imagen»
  y «no contestó» son lo mismo.
- Verificado: `make comprobar`, todo con `-race`, **74 pruebas de interfaz**, y la medida contra doce
  dominios reales repetida —sigue en nueve—.

## 2026-09-09 · La descarga de iconos no funcionaba en absoluto

- **2.14.1.** El cliente actualizó y dijo que **ninguna entrada tenía icono**. Tenía razón, y no era
  que los sitios no dieran: **no se llegaba a pedir ni uno**.
- **La causa, y es de las buenas:** el filtro de direcciones privadas estaba en `DialContext`, que
  recibe la dirección **tal como se pidió** —o sea, el nombre sin resolver—. `ParseIP("github.com")`
  da nulo, la regla «si no se sabe qué es, no se va» se cumplía, y **se rechazaban todos los sitios
  del mundo** con el mensaje «github.com es una dirección de una red privada». El filtro vive ahora
  en `Control`, que corre **después** de resolver y recibe la IP de verdad.
- **Por qué las pruebas no lo vieron**, que es lo que hay que recordar: las que hablaban con un
  servidor llevaban el filtro aflojado, y la única que lo ejercitaba de verdad usaba `127.0.0.1`,
  que **sí** es una dirección y por eso se rechazaba bien. Pasaba por el motivo correcto y por la
  razón equivocada. La prueba nueva pide por un **nombre** y exige que el rechazo hable de la
  dirección resuelta.
- Se encontró en un minuto **probando el descargador contra doce dominios reales** desde aquí, que es
  algo que no se me había ocurrido hacer y que estaba a una prueba de usar y tirar.
- **Y con eso funcionando, medir cambió dos decisiones que estaban tomadas.** Con las tres rutas
  previstas salían **4 de 12**. Añadiendo `/favicon.ico` —leído como el contenedor que es— y un
  navegador de verdad en la cabecera —tres sitios contestaban 403— subió a 6. Y leyendo el **mapa de
  bits de 32 bits** que llevan dentro Google, Amazon y Netflix, a **9 de 12**. Las dos cosas estaban
  descartadas en la ADR por buenos argumentos; los datos las desmintieron y la ficha lo dice.
- Solo se lee la variante de 32 bits sin comprimir, que es la única que no necesita paletas ni
  descompresión, y la aritmética se comprueba contra el tamaño real antes de tocar un byte: son datos
  de un tercero.
- Verificado: `make comprobar`, todo con `-race`, **74 pruebas de interfaz** y siete nuevas en Go,
  entre ellas la de un ICO con números inventados que no puede tumbar el programa.

## 2026-09-09 · Los iconos de los sitios, y la segunda conexión

- **2.14.0**, segunda mitad de lo que pidió el cliente. Ahora cada entrada enseña el icono real del
  sitio, pedido **al propio sitio** y nunca a un intermediario, con el ajuste **encendido**, decidido
  por el cliente (ADR 0024).
- **Se le corrigió un dato antes de decidir, y era importante.** Al plantear las opciones se dijo que
  ir directo hacía que «el que se entera es el sitio». Eso se queda corto: el nombre viaja en claro
  en la consulta de DNS y en el saludo TLS, así que **quien mire la red ve la lista entera igual**.
  Lo que se gana yendo directo es no meter un tercero de confianza, que es una razón distinta. Está
  escrito así en la ventana, en `docs/seguridad.md` y en la ADR.
- Como viene encendido, **la ventana lo dice una vez** antes de que ocurra, con el «No, gracias» al
  lado. Es la costumbre de la ADR 0014: si se hace, se dice, y se deja apagar.
- **El grueso del trabajo no es bajar un PNG: es no fiarse de él.** El destino lo elige un CSV que
  alguien importó, así que hay filtro de direcciones privadas **en el momento de conectar** —el
  nombre no vale, cualquier dominio resuelve a lo que quiera—, tope de saltos, prohibición de bajar a
  texto claro, tope de bytes, la cabecera mirada **antes** de decodificar —un PNG de 30 KB puede
  declarar 30.000×30.000, que son 3,6 GB— y re-codificado propio, para que lo que se guarde no sean
  bytes de un tercero.
- **Los iconos van en un fichero satélite cifrado, no dentro de la bóveda.** Dentro habrían
  multiplicado por diez un fichero que se lee, se copia y se reescribe entero al confirmar **cada
  edición de una contraseña**, y habrían viajado por el puente en **cada tecla del buscador**. Y en
  claro no podían ir: la lista de dominios es justo lo que la bóveda oculta.
- **Dos cosas que estaban mal desde antes y que esto obligó a arreglar:** `ESFINGE_SIN_RED` solo la
  miraba la línea de comandos —la ventana no la consultaba nunca, así que quien la ponía apagaba
  media red— y **la bóveda no tenía cerrojo**, con dos gorrutinas ya tocándola. Las dos eran ventanas
  estrechas que nadie había pillado y que el trabajo de fondo volvía anchas.
- Y una que se decidió a conciencia: **al sitio no se le dice quién pregunta**. A GitHub se le manda
  «Esfinge/versión» porque es lo que se compara; a un sitio cualquiera sería contarle que quien
  pregunta usa este gestor, en esta versión, con los fallos que esa versión tenga.
- Verificado: `make comprobar`, todo el paquete de la bóveda con `-race`, **74 pruebas de interfaz** y
  quince nuevas en Go, entre ellas las ocho del descargador. `/favicon.ico` **no** está en la lista de
  rutas: la biblioteca estándar de Go no sabe decodificar ICO.
- **Lo que no se ha comprobado:** nada de esto se ha ejecutado contra sitios de verdad. Cuántos de los
  sesenta y cinco dan icono con solo tres rutas conocidas lo dirá el uso.

## 2026-09-09 · Cada entrada con su cuadro, y la lista ordenada

- **2.13.0**, primera mitad de lo que pidió el cliente: que cada entrada se reconozca sin leerla.
  Cuadro con la inicial y color propio, iconos de trazo en las pestañas, la clase al final de la
  fila **solo en «Todo»** —dentro de «Tarjetas» todas son tarjetas— y la lista ordenada.
- **Ocho tintes, generados en Go y medidos uno a uno.** Es el único color propio que entra en una
  pantalla de trabajo, y la ADR 0021 decía que ahí no entra ninguno; entra porque el problema
  cambió, con sesenta y cinco entradas dentro. La condición es la de siempre: **32 parejas nuevas
  en `make contraste`** —la letra sobre cada cuadro y cada cuadro contra la lista, en los dos
  temas—. Salen entre 8,7:1 y 11,3:1.
- El tinte lo elige el dominio con **FNV-1a y no con el hash del motor**: el color de un sitio tiene
  que ser el mismo mañana y en la otra máquina, o la lista cambia de colores sola y se lee como un
  fallo.
- **Y los emoji fuera.** `ICONO_TIPO` usaba 🔑📝💳🪪, que es exactamente lo que el comentario de
  `Icono` prohíbe desde que existe: «los emoji son de color y desentonan». Cuatro trazos nuevos con
  el mismo molde que los seis de la barra lateral, y `Icono` pasa a estar exportada.
- **Dos fallos que solo se vieron en la captura**, no en las pruebas: el selector de orden **se
  salía del panel** y quedaba cortado —ahora la fila se parte en dos antes que cortar un control— y
  **«Hacienda» salía con una «A»**, porque la letra se sacaba del dominio y no del nombre que se lee
  justo al lado. Ahora la letra sale del nombre y **el color del dominio**, que son dos cosas: el
  color identifica el sitio y la letra tiene que cuadrar con lo que se ve.
- **Y uno que encontró una prueba**: ordenar por «cambiada la última» salía en orden aleatorio,
  porque la fecha se guarda con precisión de segundo y una importación deja las sesenta y cinco
  entradas con la misma. Lleva desempate por nombre.
- Verificado: `make comprobar`, `make contraste` con las 32 parejas nuevas y **74 pruebas de
  interfaz** —cuatro más—, la tanda repetida dos veces.

## 2026-09-09 · La bóveda, separada por clases

- **2.12.8.** Con sesenta y cinco entradas dentro, el listado único deja de navegarse. Ahora hay
  cinco pestañas encima de la lista —Todo, Credenciales, Notas, Tarjetas e Identidades— y el
  buscador sigue valiendo dentro de cada una.
- **Las pestañas están siempre, aunque estén vacías.** Que la de tarjetas exista es lo que dice que
  se pueden guardar tarjetas; esconderla haría que la interfaz cambiara de forma según lo que hubiera
  dentro, que es peor que una pestaña vacía. Cuando lo está, dice **por qué** —«todavía no hay
  ninguna tarjeta»— en vez de un «no hay nada» que deja pensando si se ha roto algo.
- Dos detalles que salieron de mirarlo puesto: **«Nueva» crea de la clase que se esté mirando**
  —estando en Tarjetas, una tarjeta— y el pie cuenta las de la pestaña y el total, que son dos
  números distintos y los dos hacen falta.
- El filtro se aplica en la ventana y no en Go: la lista ya ha cruzado el puente entera —viene sin
  secretos— y volver a pedirla por cada pestaña sería un viaje para nada.
- Verificado: `make comprobar` y **70 pruebas de interfaz**, dos más, y la tanda repetida dos veces.
  La prueba del filtro comprueba las dos mitades: que cada pestaña enseña lo suyo **y esconde lo
  demás**, que es la que se olvida.

## 2026-09-09 · «¿Están todas?», contestado por el programa

- **2.12.7.** Después de importar, el cliente preguntó si en su `credentials.csv` había 65 entradas.
  No se podía saber: la respuesta estaba en un fichero de su Mac, y ni él ni yo teníamos forma de
  contestarla desde donde estábamos.
- **Eso es un hueco del programa, no de la pregunta.** Es la segunda vez que «¿me lo he traído todo?»
  se queda sin respuesta —la primera fue con los cinco ficheros de Dashlane—, y quien tiene el dato
  es justamente el que acaba de leer el CSV. Ahora la ventana dice cuántas filas traía el fichero,
  cuántas venían vacías y si la cuenta cierra.
- Contar las líneas por fuera no vale, y por eso lo tiene que decir él: **una nota con saltos de
  línea ocupa varias líneas y es una sola fila**. Hay una prueba con ese caso exacto.
- Verificado: `make comprobar` y **68 pruebas de interfaz**, la tanda repetida dos veces.

## 2026-09-09 · Reimportar duplicaba la bóveda entera

- **2.12.6.** El cliente pasó dos veces su `credentials.csv` y se encontró con **130 entradas donde
  había 65**. Los repetidos se marcaban —esa parte funcionaba— pero **se metían igual**, y marcar no
  sirve de nada cuando lo que hay que hacer es no meterlos.
- La regla de antes venía de un argumento correcto —«dos contraseñas distintas para la misma cuenta
  significan que una está mal, y adivinar cuál no es cosa de un importador»— aplicado donde no tocaba.
  Ese argumento vale para una cuenta **con otra contraseña**; para una entrada idéntica no hay nada
  que decidir. Ahora son dos preguntas distintas: si es la misma entrada no se mete, y si es la misma
  cuenta con otro secreto entra marcada.
- Con eso **pasar dos veces el mismo fichero no cambia nada**, que es la prueba que faltaba y que se
  echó de menos de la peor forma posible.
- **Y de perseguir una prueba frágil salió otro fallo de verdad**, que llevaba oculto detrás: las
  preferencias se leen más de una vez, y **una lectura pedida antes de un cambio puede llegar
  después**, traer lo viejo y aplicarlo encima; el cambio siguiente parte de ahí y borra el anterior.
  Se veía como que bajar el bloqueo a cinco minutos y acto seguido el portapapeles a diez dejaba el
  bloqueo otra vez en quince. La prueba lo enseñó cuatro veces sin que yo supiera leerlo hasta que
  puse a imprimir las peticiones y las respuestas en orden.
- La lección de método: **una prueba que falla una de cada cinco veces está diciendo algo**, y lo
  barato —subirle el tiempo de espera— es lo que la calla. Aquí decía dos cosas, y las dos eran del
  programa.
- Verificado: `make comprobar` y **68 pruebas de interfaz**, con la tanda entera repetida **seis
  veces seguidas** en verde.

## 2026-09-09 · Borrar la bóveda, y una negrita que partía los avisos

- **2.12.5.** Se puede borrar la bóveda desde el programa, que hasta ahora exigía ir al Finder y
  borrar dos ficheros a mano —y quien borraba solo el primero se dejaba una bóveda entera detrás, en
  el `.anterior`—.
- Está construida para costar: **pide la contraseña maestra**, ofrece exportar antes, necesita dos
  pulsaciones y va apartada con un filete de aviso. Y conviene ser honesto sobre qué protege la
  contraseña: **no de quien quiera hacer daño** —quien puede abrir Esfinge puede borrar el fichero
  desde el Finder— sino de un clic mal dado y de que lo haga quien no es el dueño con la bóveda
  abierta encima de la mesa. Es la misma razón por la que cambiar la maestra pide la de antes.
- Borra los dos ficheros **y los temporales**, que llevan una bóveda entera dentro. Hay una prueba
  que recorre el directorio y no deja pasar nada con «boveda» en el nombre.
- **Y de mirar una captura salió un fallo de CSS que llevaba desde el principio**: los avisos y las
  notas eran contenedores flexibles —para colocar el glifo delante— así que **un `<strong>` en medio
  de una frase se convertía en una columna aparte** y la frase se leía partida en trozos verticales.
  No se había notado mientras todos los avisos fueron texto pelado. Ahora es una sangría francesa,
  que es lo que se quería desde el principio, y hay una prueba que vigila que eso no vuelva a ser
  flexible.
- **Tres pruebas frágiles arregladas de verdad, no subiéndoles el tiempo de espera**: la de los
  plazos competía con la llamada que carga las preferencias —mientras está en vuelo la lista enseña
  su valor por defecto—, y la de pegar reintentaba comparando por igualdad, con lo que dos pegados
  dejaban el texto duplicado y la comparación no se cumplía nunca. Las dos fallaban unas veces sí y
  otras no, y ninguna era un fallo del programa.
- Verificado: `make comprobar` y **68 pruebas de interfaz**, con la tanda entera repetida tres veces
  seguidas en verde, que es lo que distingue arreglar una prueba frágil de esconderla.

## 2026-09-09 · Dashlane no exporta un CSV, exporta cinco

- **2.12.4.** El cliente importó `credentials.csv`, salieron 65 entradas y en Dashlane había más
  cosas. No había ningún error por medio: **lo que faltaba eran los otros cuatro ficheros**, y el
  importador los rechazaba enteros diciendo que no reconocía ninguna columna.
- **La causa no era que faltaran alias, era que faltaba distinguir la forma del fichero.** Una misma
  columna significa cosas distintas en cada uno: `number` es el número de una tarjeta en
  `payments.csv` y el de un pasaporte en `ids.csv`; `type` es la clase de tarjeta allí y la clase de
  documento aquí; `name` es el título de una cuenta en uno y el nombre de una persona en otro. Con
  una sola tabla de alias eso no se resuelve: se acierta en un fichero y se falla en el otro.
- Ahora `FormaDeLaCabecera` decide primero qué se está leyendo —por las columnas que **solo** salen
  en una clase de fichero— y cada forma tiene su tabla, que pisa a la común. Con eso entran
  credenciales, notas seguras, tarjetas y documentos, y el tipo de cada entrada se deduce **de los
  campos que vengan rellenos**, no de lo que diga el fichero de sí mismo.
- **Y de paso, dos cosas que estaban mal y no se veían:**
  - **Todas las tarjetas eran duplicadas entre sí.** La huella de una entrada era «sitio + usuario»,
    y una tarjeta no tiene ni lo uno ni lo otro: importar cinco marcaba cuatro como repetidas. Lo
    mismo con las notas seguras. Cada clase se identifica ahora por lo suyo, y una tarjeta escrita
    con espacios se reconoce como la misma que sin ellos.
  - **Lo exportado no volvía a entrar entero.** El CSV de salida solo llevaba los campos de una
    credencial. Ahora lleva todos y se reconoce como propio, porque una bóveda de la que no se puede
    salir del todo es una trampa a medias.
- **La ventana no decía nada de esto**, que es la mitad del fallo: ahora avisa de que algunos
  gestores exportan varios ficheros y hay que traerlos uno a uno. Y si aun así uno no se reconoce, el
  error **dice qué columnas traía**, para que añadirlo sea cosa de cinco minutos en vez de un
  callejón sin salida.
- **Un tercer fallo, encontrado por una prueba y no por una persona:** cambiar dos ajustes seguidos
  perdía el primero. Go recibe el objeto entero, así que cada cambio manda también lo que no se ha
  tocado; leyéndolo del estado de React, el segundo cambio parte del valor de antes porque entre uno
  y otro todavía no se ha vuelto a dibujar. En pantalla los dos se veían puestos. En un ajuste que
  apaga el bloqueo de la bóveda, eso no puede quedarse así.
- Verificado: `make comprobar`, **66 pruebas de interfaz** y siete pruebas nuevas del importador, con
  las cabeceras reales de los cinco ficheros de Dashlane.
- **Lo que sigue sin saberse:** si `personalinfo.csv` —direcciones, teléfonos, fechas de nacimiento—
  merece entrar. No son secretos, son datos de autorrelleno, y esa es una decisión de producto y no
  de importador.

## 2026-09-09 · Tres cosas que solo aparecen usando la aplicación

- **2.12.3: el diálogo de abrir no dejaba elegir ningún fichero en macOS.** El filtro «todos los
  ficheros» llevaba el patrón `*.*`, que es lo idiomático en Windows y ahí funciona. Wails le quita
  el `*.` de delante antes de pasárselo al `NSOpenPanel`, así que llegaba como **una extensión
  llamada literalmente `*`**, que no tiene ningún fichero: el panel se abría y estaba todo en gris.
  En macOS los filtros no son una lista desplegable, son la única lista que el panel acepta.
- **Estuvo así desde que existen los diálogos.** No se vio nunca porque en macOS los ficheros se
  arrastran a la ventana y ese camino no pasa por ahí; salió a la primera con importar de otro
  gestor, que es lo primero que no tiene arrastrar y soltar. Está anotado en `CLAUDE.md` junto a lo
  que sí se ha comprobado en el Mac, para que no vuelva a parecer que ese diálogo estaba visto.
- El arreglo: en macOS **no se manda ningún filtro** —y entonces Wails llama a
  `setAllowsOtherFileTypes:true`—; en Windows `*.*` y en GTK `*`, que ahí sí son listas desplegables.
  La regla vive en `filtrosPara`, que **toma el sistema como argumento** para poder comprobar los
  tres desde esta máquina; sin esa costura, la regla de macOS no se puede probar en ninguna parte.
- Y la segunda mitad, que es la lección: **un filtro nunca puede impedir elegir**. Un `.esf` puede
  ser un `.txt` con la línea `ESF1.…` dentro, y una exportación de contraseñas llega con la extensión
  que le dé la gana al gestor que la escribió. Hay una prueba que lo vigila para los tres sistemas.

## 2026-09-09 · Dos bugs del campo de la clave, contados por el cliente

- **2.12.1**, antes de que llegara a probar la bóveda. Los dos eran de la pantalla de cifrar y los
  dos llevaban versiones ahí.
- **Pegar una clave con ⌘V no activaba el botón de cifrar.** React mantiene su propio registro del
  valor de cada campo en un accesor **del elemento**: al hacer `campo.value = …` ese registro se pone
  al día antes de tiempo, y el evento `input` que se dispara después no le parece un cambio, así que
  no llama a `onChange`. La clave se veía puesta y para la aplicación el campo seguía vacío. Como los
  menús se construyen a mano, **⌘V pasa por `ordenes.ts`**, así que esto afectaba a todos los campos
  de la ventana.
- **La clave generada con «Generar una» no se podía copiar.** De un campo de contraseña el navegador
  se niega a copiar, a propósito y sin decirlo: `execCommand("copy")` devuelve que sí y el
  portapapeles se queda como estaba. Y esa clave es precisamente la que no está apuntada en ningún
  otro sitio. Ahora se copia por Go, que además arma el borrado pasado el plazo.
- **El campo de la clave tiene un ojo** para verla y volver a taparla, que es lo que pidió el
  cliente. Tapada de partida, siempre: quien teclea con alguien detrás no tiene que acordarse de
  esconderla primero. Destapada va en monoespaciada, para distinguir un cero de una O.
- **2.12.2**: ese ojo pasa a ir **dentro del campo**, a la derecha, en vez de ser un botón con rótulo
  encima. Lo pidió el cliente al verlo y tiene razón: fuera competía con «Generar una» por el mismo
  sitio y se leía como otra acción del formulario, cuando no es una acción sino una propiedad de lo
  que se está mirando. El icono lleva nombre accesible —«Ver la clave» / «Ocultar la clave»—, que es
  además por donde lo localiza la prueba: si alguien deja el dibujo sin nombre, se pone roja.
- **Lo que enseña esta pareja**: la prueba que había del menú ejercitaba «seleccionar todo» —lo fácil
  de mirar— y no pegar, que es lo que la gente hace todos los días. Las tres pruebas nuevas van por
  el camino de verdad, y la de pegar **se comprobó en rojo** quitando el arreglo antes de darla por
  buena: falla exactamente donde lo notó el cliente, con el valor en el campo y el botón apagado.
- Verificado: `make comprobar` y **66 pruebas de interfaz** en los dos temas, tres más.

## 2026-09-09 · La bóveda: Esfinge deja de ser un cifrador sin estado

- **2.12.0.** El cliente quiere sustituir Dashlane poco a poco. Se valoró entero —bóveda,
  autorrelleno, cuentas, servidor y compartir— y la conclusión fue que **eso no es ampliar Esfinge,
  es construir Dashlane**. Se hace la fase 1 y se para ahí: si la bóveda no se usa a diario, las
  otras tres no se empiezan.
- **Antes de nada, la fase 0: congelar el formato** (ADR 0022). El único test que decía congelar
  `ESF1` se miraba al espejo —ciframos y desciframos en el momento— así que un cambio coherente en
  los dos sentidos habría pasado en verde dejando de abrir lo ya emitido, en silencio. Ahora hay once
  vectores grabados una vez, tres rotos a propósito, y contenedores sellados **con el código de la
  1.5.0 de verdad**, sacado con `git archive`. Se comprobó que el test se pone rojo ante un cambio de
  ese tipo antes de darlo por bueno.
- **La bóveda no toca `internal/cripto`** (ADR 0023), y ésa es la decisión de la que cuelga todo lo
  demás: es un JSON legible cuyos campos cifrados son líneas `ESF1.` corrientes. La idea evidente
  —un contenedor cuyo contenido sea la bóveda— no vale, porque la clave de recuperación exige dos
  entradas independientes al mismo secreto y en la cabecera de 55 bytes solo cabe una sal. Lo que se
  gana: **los `.esf` y las claves ya emitidos siguen valiendo**, y las ranuras quedan como lista
  abierta para Touch ID o un servidor, sin migrar nada.
- La contraseña maestra **no cifra la bóveda**: cifra la clave que la cifra. Cambiarla es volver a
  envolver 32 bytes, y hay un test que comprueba que **el cuerpo queda byte a byte idéntico**.
- Piezas: escritura atómica con `fsync` en su propio paquete, importación de Dashlane, Bitwarden,
  1Password, LastPass y Chrome —por nombre de columna, no por posición—, exportación en claro para
  poder salir, bloqueo por inactividad, borrado del portapapeles, sección en la ventana, dos plazos
  en Ajustes y `esfinge boveda listar|ver|exportar`.
- **Tres fallos encontrados por el camino, y ninguno era de la bóveda:**
  - **Una conexión de eventos por oyente.** El navegador solo abre seis contra el mismo origen y un
    flujo de eventos no termina nunca: con el sexto oyente **toda llamada al puente se quedaba
    esperando para siempre**, sin error y sin petición en la red. Con cinco funcionaba. El síntoma
    fue un botón de copiar que no hacía nada, y costó encontrarlo porque no había nada que mirar.
  - **Un guardado a medias apagaba los dos relojes.** `GuardarPreferencias` recibe el objeto entero,
    así que un `{"buscarActualizaciones":true}` —que es lo que manda una prueba de hace versiones—
    dejaba los plazos a cero al deserializar; con el cero significando «nunca», eso apagaba el
    bloqueo de la bóveda en silencio. Ahora «nunca» es `-1` y el cero conserva lo que hubiera.
  - **Una errata en la clave de recuperación se contaba como «no abre».** El comentario prometía la
    distinción y el código se comía el error de la suma de control. La prueba que había le preguntaba
    a `Normalizar`, no al camino por el que pasa la persona.
- Y el servidor de desarrollo **aísla ya su carpeta de configuración**: sin eso, estas pruebas
  dejarían una bóveda con una contraseña que está escrita en el fichero de pruebas en la carpeta de
  verdad de quien desarrolla.
- Verificado: `make comprobar` con `-race`, `make contraste`, y **60 pruebas de interfaz** en los dos
  temas —diez más—. Medido y no estimado: 20.000 entradas se guardan en 92 ms y se abren en 276 ms.
- **Lo que no se ha comprobado, y es lo que importa ahora**: nada de la bóveda se ha usado con datos
  de verdad ni en un Mac. Y sigue sin haber auditoría externa, que para un cifrador era una nota al
  pie y para un gestor de contraseñas es la primera pregunta que hará cualquiera.

## 2026-09-09 · La identidad entra en la ventana

- **2.11.0.** La aplicación no tenía una sola marca en ningún píxel, y ahora la tiene: lockup de
  Esfinge en la barra lateral, firma `▍ webcafeína` al fondo, la esfinge tenue en el historial vacío
  y la ficha de producto en Ajustes. **El acento pasa a ser el oro del tocado** en vez del azul del
  sistema (ADR 0021, que matiza la 0007).
- **Se midió antes de dibujar, y la medida decidió el diseño.** Blanco sobre el oro da 1,68:1 y no
  tiene arreglo: oscurecer el oro hasta que el blanco cumpla lo deja en un marrón. Piedra sobre oro
  da 8,38:1. De ahí la regla —**el oro rellena, la piedra escribe**— que resultó ser la gramática del
  propio icono: tocado dorado sobre placa oscura. La identidad no se le puso encima, se le sacó.
- **Un hallazgo que casi se cuela, y es el que más vale de la sesión.** `--relleno` servía para dos
  cosas que el azul cumplía y el oro no: rellenar superficies y dibujar líneas finas. El oro como
  línea sobre fondo claro da 1,37-1,68:1 y es invisible. **`make contraste` habría pasado en verde
  con el foco de los campos roto**, porque mide parejas de tokens y no sitios. Cuatro reglas pasaron
  a `--acento` y hay una prueba de interfaz que lo vigila.
- Un daño colateral que se vio mirando la captura, no razonando: **«Generar una» dejó de parecer un
  botón** al perder el azul. Lleva subrayado, que devuelve la señal sin gastar color.
- **La marca es una pieza nueva, no una copia**: `build/marca.svg`, la esfinge a trazo y en
  `currentColor`. Se probó maciza y era un borrón —en monocromo el rostro y el tocado se funden— y se
  probó con `evenodd` y salía una herradura. A trazo se lee, y rima con los iconos de la barra
  lateral. El grosor salió de mirarla a 18, 20, 24, 48 y 96 px.
- Verificado: `make contraste` con ocho parejas nuevas, dos tests de Go reescritos —uno afirma ahora
  que el blanco sobre el oro **no** llega, que es el supuesto que sostiene todo—, y **50 pruebas de
  interfaz** en los dos temas, seis de ellas nuevas. Capturas de la portada regeneradas.
- Dos fallos míos en las pruebas nuevas, los dos por leer demasiado pronto: `.focus()` no dispara
  `:focus-visible`, y el borde va con transición, así que hubo que medirlo con `poll`. Y el historial
  se carga al entrar, o sea que preguntar en el acto si «Vaciar» está activo es la trampa que este
  fichero ya había pisado dos veces.
- **2.11.1, con las seis correcciones que salieron de mirarlo puesto.** Ninguna se habría visto sin
  abrir la ventana, y una de ellas era un fallo de dibujo con causa exacta:
  - El lockup usaba **el mismo tamaño que las filas** de navegación y el nombre parecía una sección
    más. Ahora va al tamaño de título.
  - La firma pasa a `Webcafeína ▍ 2.11.1`: el glifo de la casa separa el nombre de la versión, que
    antes solo estaba en Ajustes.
  - **La fila de la barra lateral se hunde al pasar por encima, no se levanta.** Usaba
    `--boton-encima`, que aclara: correcto para un botón suelto, al revés para una barra lateral. De
    ahí `--barra-encima`, un color propio.
  - La casilla de Ajustes se dibuja entera: la del navegador va en azul del sistema, y `accent-color`
    tampoco sirve porque el tic lo pinta blanco y sobre el oro son 1,68:1.
  - **La silueta tenía una raya cruzando la cara**: el tramo `h246` del contorno del tocado, que
    relleno queda tapado por el rostro y a trazo se dibuja. Contorno abierto por abajo.
  - Y el ojal tocaba la barbilla: doce píxeles de separación que medio grosor de trazo —29— se comía
    entera. Bajado 40, y el `viewBox` ajustado al dibujo.
- **2.11.2, de preguntar si todo esto llegaba a Windows y a Linux.** Llegaba —lockup, firma, hover y
  marca son de los tres— **menos una cosa: la barra lateral de Windows se quedaba sin una gota de
  oro**, con la fila activa en gris y la barra de acento en piedra. La decisión original era correcta
  a medias: Fluent no rellena la selección, pero **la tiñe** con el acento diluido. De ahí
  `--relleno-tenue`, y ahora el oro entra como relleno mientras la señal la sigue dando la barra.
- Y una trampa de método: al forzar `data-sistema` desde una prueba para mirar los tres sistemas hay
  que **esperar a que la aplicación fije el suyo**, o su promesa lo sobrescribe y se acaba juzgando
  otro creyendo que es éste. Pasó, y la captura decía que el arreglo no funcionaba.
- **Queda por juzgar con el ojo**: la pastilla dorada de la fila activa, que es lo más visible, y si
  el ámbar de los avisos se estorba con el oro. Medido no es visto — y estas siete correcciones lo
  demuestran.

## 2026-09-09 · La banda sale sola

- **Confirmado en el Mac: la banda de versión nueva aparece con la ventana abierta**, sin reiniciar y
  sin tocar nada. Es la prueba buena del reloj de la 2.10.4, y la que no se podía hacer aquí: una
  comprobación que ocurre una vez al día no se observa en una compilación en verde.
- Con eso se cierra un frente que llevaba abierto desde la 2.1.0 **sin que nadie lo supiera**: la
  ficha decía «comprueba una vez al día» y lo implementado era «comprueba al arrancar». La ADR 0014
  ya lleva su corrección y ahora también su verificación.
- De paso, dos frases de `CLAUDE.md` que se habían quedado atrás: seguían dando por no comprobado el
  icono del `.esf` en el Finder, visto el día anterior.
- **Queda una sola cosa en todo el proyecto**, y solo la puede hacer el humano: abrir la aplicación en
  Windows y en GNOME de verdad, y mirar ahí la estructura de cada sistema y el icono de los `.esf`.

## 2026-09-08 · El aviso de versión nueva no llegaba nunca

- **2.10.4, y salió de una confusión de nombres.** El cliente llevaba toda la sesión diciendo que «no
  salta el aviso», y yo lo entendí como el de Gatekeeper. Al explicarle qué era Gatekeeper, aclaró:
  se refería a **la banda de versión nueva**, que no le había salido jamás.
- **El fallo era real.** `comprobarAlArrancar` la llamaba `Arrancar` una sola vez y **no había ningún
  reloj en todo el código**: solo se comprobaba al abrir la ventana. Quien deja Esfinge abierta —lo
  normal en una herramienta así— no se enteraba nunca. La portada y la ADR 0014 decían «comprueba una
  vez al día»; lo que hacía era «comprueba al arrancar, y como mucho una vez al día». Esa frase la
  escribí yo y no era cierta.
- Ahora `vigilar` deja un reloj que se asoma cada hora y aplica **la misma puerta de las 24 horas**,
  así que el techo de una petición al día no se mueve: asomarse a menudo no es preguntar a menudo.
- **Dos fallos míos los encontraron las pruebas, no yo.** Preguntar con `TocaMirar` y anotar al volver
  de la red deja el hueco de toda la petición en medio: con el reloj, cuatro comprobaciones donde
  debía haber una. De ahí `ReservarComprobacion`, que decide y anota **sin soltar el cerrojo**. Y al
  cerrar la ventana el reloj seguía: cuando las dos ramas de un `select` están listas Go elige al
  azar, así que el bucle vuelve a mirar `ctx.Err()` antes de trabajar.
- Verificado: tres pruebas nuevas con `-race` —avisa sin reiniciar, respeta el techo diario, se para
  con la ventana—, toda la suite con `-race`, y las 38 de interfaz.
- **La lección, que es la de la sesión entera por tercera vez:** esto lo encontró el cliente usando la
  aplicación, no una prueba ni una lectura del código. Las pruebas cubrían el arranque, que era
  justamente el único caso que funcionaba.

**Estado al cerrar:** 2.10.4 publicada. Nada a medias en el código y ningún frente abierto de los que
venían de atrás. Queda **una sola comprobación**, y solo la puede hacer el humano: abrir la
aplicación en Windows y en GNOME de verdad, y mirar ahí la estructura de cada sistema y el icono de
los `.esf`.

**Lo que se llevó esta sesión, y vale más que las versiones:** tres veces se dio por bueno algo que
no se había comprobado, y las tres se cayeron en cuanto el cliente miró, preguntó o usó la
aplicación.

1. Que macOS baja el emblema de los documentos a la mitad inferior. Lo corrigió él al ver el icono
   puesto: caía 124 px por debajo del centro.
2. Que el icono del documento llegaría solo a los tres sistemas. Lo destapó su pregunta de si valía
   para Windows y Linux; sin ella, el fallo quedaba arreglado en macOS y vivo en Linux.
3. Que la comprobación de actualizaciones era «una vez al día». Era «al arrancar», y la frase la
   había escrito yo en la portada y en la ADR 0014.

Ninguna de las tres la encontró una prueba ni una lectura del código. Y la tercera es la que más
duele, porque **las pruebas cubrían el arranque, que era justamente el único caso que funcionaba**:
la prueba confirmaba la parte buena y no preguntaba por el resto.

## 2026-09-08 · El icono del documento, y Gatekeeper respondido

- **2.10.1.** Dos frentes que llevaban semanas como «sin comprobar», y uno de los dos escondía un
  fallo.
- **El icono del `.esf` no faltaba: era el de la aplicación.** `build/esf.png` resultó ser una copia
  **byte a byte** de `build/appicon.png`, así que el Finder sí enseñaba un icono propio… el del
  programa. Un documento cifrado y la aplicación que lo abre se veían igual. Es el mismo error que ya
  se corrigió en el icono del volumen del DMG, cometido dos veces.
- Ahora lo dibuja `build/documento.svg`, con la forma del sistema: hoja vertical, esquina de arriba a
  la derecha doblada con el dorso del papel a la vista, y la marca sobre una placa oscura. **Con el
  logo tal cual y no en negativo**, que fue la decisión al dibujar el disco.
- Un ajuste tras mirarlo: la placa ocupaba dos tercios del ancho y el papel casi no se veía, con lo
  que volvía a parecer el icono de la aplicación. Bajada a poco más de la mitad. **Lo que dice
  «documento» es el papel, no el emblema.** Comprobado a 16, 32, 48, 64 y 128 px.
- **Verificado sin Mac, abriendo el artefacto**: en el paquete, `esf.icns` (270.907 bytes) y
  `iconfile.icns` (113.518 bytes) ya son ficheros distintos, y el `Info.plist` declara la extensión,
  el nombre, `CFBundleTypeIconFile: esf` y el rol. Que el Finder lo enseñe ya es cosa del humano.
- **Gatekeeper no salta al actualizarse desde dentro**, confirmado tras varias actualizaciones de
  verdad. Confirma el razonamiento de la ADR 0016: la cuarentena la pone quien descarga, y aquí
  descarga Go. No hubo nada que quitar del LÉEME —ya acotaba el aviso a la primera vez—; lo que
  faltaba era decir lo contrario, que las actualizaciones no vuelven a preguntar. La deuda baja de
  Media a Baja.
- **Comprobado en el Mac: el Finder lo enseña**, y sin forzar la caché de LaunchServices. Pero de
  verlo puesto salió una corrección, la **2.10.2**: la placa estaba centrada en y=636 y la hoja tiene
  su centro en 512, así que caía 124 px baja. La había bajado a propósito, dando por hecho que macOS
  pone el emblema en la mitad inferior de los documentos; no era cierto, y mirándolo se notaba.
  Centrada, quedan 282 px de margen arriba y abajo.
- **2.10.3, de preguntar si el icono también valía para Windows y Linux.** En Windows sí, y solo, y
  no había nada que hacer: Wails genera `build/windows/esf.ico` desde el mismo `build/esf.png` y su
  plantilla NSIS lo copia al directorio de instalación y registra ahí la asociación. Además no está
  en git, así que se regenera en cada compilación y no puede quedarse viejo.
- **En Linux no llegaba.** `packageApplicationForLinux` de Wails **devuelve `nil`**: allí las
  asociaciones las pone nuestro `.deb`, y `esfinge-mime.xml` apuntaba a `<icon name="esfinge"/>`, que
  es el de la aplicación. O sea, el mismo fallo recién corregido en macOS, vivo en el otro sistema.
  Ahora el paquete instala `application-x-esfinge.png` en `hicolor/<tamaño>/mimetypes/` —por nombre y
  carpeta, que es como lo busca el escritorio— y el XML apunta ahí.
- Verificado sin escritorio Linux, con `dpkg -c` sobre el `.deb` publicado: los ocho tamaños de
  `application-x-esfinge.png` están en `mimetypes/`, y el `esfinge.xml` que los nombra también.

## 2026-09-08 · La red para lo que no se puede probar aquí

- **`compilar.yml` acepta pedir compilación con inspector**: una entrada booleana en el disparador
  manual que se traduce en `wails build -devtools`
  (`gh workflow run compilar.yml -f inspector=true`). Es lo último que quedaba de la deuda que dejó
  el vidrio: adivinar en un sistema que no se puede ejecutar aquí costó tres versiones y una
  aplicación que no arrancaba.
- **Marcada por tres sitios para que no se confunda con una normal**: el artefacto sale como
  `-CON-INSPECTOR`, dura siete días en vez de treinta, y la versión lleva sufijo. Lo del sufijo no es
  cosmético: una versión que no son tres números hace que la comprobación de actualizaciones se calle
  —está escrito así a propósito en `internal/actualizacion`—, que es justo lo que quiere una
  compilación de diagnóstico. Al dispararse por etiqueta `inputs` viene vacío, así que una
  publicación no puede salir con inspector; y publicar va por `publicar.yml`, que ni tiene la entrada.
- Leyendo el código de Wails antes de escribir: el flag `-devtools` solo añade la etiqueta del mismo
  nombre, y en macOS imprime un aviso de que el paquete usa APIs privadas y no pasaría la App Store.
  No publicamos ahí, pero es una razón más para que no se confunda con un paquete de publicación.
- **Verificado disparándola de verdad**, no solo validando el YAML: una red de seguridad sin probar
  no es una red.
- De paso, se pusieron al día los documentos vivos: varias entradas llevaban `2026-09-08` cuando se
  escribieron el 7 —el git lo confirma—, y `deuda.md` tenía tres apuntes que los hechos ya
  desmentían: que la aplicación nunca se hubiera ejecutado en un escritorio, que no hubiera barra de
  menús propia, y que la derivación por fichero siguiera pendiente cuando se aceptó y lo que se hizo
  fue paralelizar.

## 2026-09-07 · El vidrio, resuelto leyendo el código de Wails en vez de adivinando

- **2.9.2**: revertido entero el diagnóstico de la 2.9.1, que cerraba la aplicación al arrancar.
  Publicada y confirmada por el cliente: «ya arranca». **Restaurar la herramienta fue primero**; el
  porqué, después.
- **2.10.0, y la causa del gris de tres versiones.** El código de Wails está en el caché de módulos de
  esta máquina, y en `internal/frontend/desktop/darwin/WailsContext.m` se lee que crea el
  `NSVisualEffectView`, le pone `setBlendingMode` y `setState`… y **nunca `setMaterial`**. Se queda
  con `NSVisualEffectMaterialAppearanceBased`, obsoleto desde macOS 10.14 y que hoy se dibuja plano.
  Había vidrio y estaba desenfocando nada.
- **Y encima lo tapábamos.** La barra lateral pintaba un tinte por delante del material, cuyo alfa se
  había bajado dos veces (0,82 → 0,55) persiguiendo el síntoma. Retirado: en una barra lateral de
  macOS el material *es* el fondo. `alfaDelVidrio` y `--barra-vidrio` ya no existen.
- Verificado aquí: `make comprobar`, `make contraste` en AA, cruce a darwin sin cgo y **38 pruebas de
  interfaz** en los dos temas —dos nuevas: que bajo vidrio `.lateral` queda transparente y `.zona` no—.
  El Objective-C, como siempre, solo lo comprueba el trabajo de macOS, **y eso solo dice que compila**.
- De paso: Playwright se había actualizado a 1.63 con `^`, y pedía un navegador que no estaba en el
  caché. Las 40 pruebas fallaban en 3 ms por eso, no por el cambio. Descargado el que toca.
- **Comprobado en el Mac: «ahora sí se ve».** El desenfoque pareció excesivo a primera vista, pero
  puesto al lado de la barra lateral del Finder **se ve igual**: es el estándar de macOS 26. Como el
  radio del desenfoque no es ajustable —lo fija el material—, comparar con una aplicación de Apple era
  la única forma de distinguir «nos hemos pasado» de «esto es el sistema». No se tocó nada.
- **La lección, que costó tres versiones y una aplicación rota:** las tres veces deduje la causa
  razonando sobre lo que Wails «debería» hacer en una plataforma que no puedo ejecutar aquí. Su código
  estaba todo el tiempo en `~/go/pkg/mod/`, y la causa se leyó en dos minutos el día que fui a mirar.
  **Leer la biblioteca va antes que razonar sobre ella.**
- Queda anotado, ya sin urgencia, en `docs/deuda.md` y con su forma concreta en `siguiente.md`
  (Media): darle a `compilar.yml` una entrada `workflow_dispatch` para pedir `wails build -devtools`.
  El siguiente problema de macOS que no se reproduzca aquí lo agradecerá.
- **Al cerrar se puso al día `siguiente.md`**, que se había quedado atrás: se cerraron la barra de
  menús, el montaje del DMG y el reemplazo automático del binario —que además estaba **mal
  descartado**, porque se dio por hecho que exigía firmar con Apple y no tiene nada que ver (ADR
  0016)—, y se añadieron el vidrio y las estructuras de Windows y GNOME pendientes de ver en máquinas
  de verdad.

**Estado al cerrar:** 2.10.0 publicada e instalada por el cliente, sin nada a medias en el código y
sin frentes abiertos de los que venían de atrás. Lo que espera son comprobaciones suyas, ninguna
bloqueante.

## 2026-09-07 · Vidrio, carpetas recordadas y tandas en paralelo

- **2.5.0**, tres cosas que llevaban días anotadas en `siguiente.md` y no dependían de nadie.
- **El vidrio del sistema** en la barra y el pie, con la zona de trabajo opaca (ADR 0017). Lo que no
  era evidente: pedirlo hace que el fondo lo tenga que pintar el CSS, y donde no hay vidrio —Linux,
  o un Windows sin Mica— un `body` transparente no enseña el escritorio, enseña un agujero. De ahí
  el atributo `data-vidrio` y una prueba que vigila que sin él nada cambia.
- El tinte se ajustó **mirando**: se simuló la ventana con un degradado saturado detrás y a 0,72 el
  texto apagado del pie se lavaba. Subido a 0,82. La simulación no tiene el desenfoque del sistema,
  así que es un caso peor que el real.
- **Los diálogos recuerdan su carpeta**, una para abrir y otra para guardar. Con una trampa que
  habría dejado el diálogo sin abrir: Wails **falla la llamada entera** si el directorio por defecto
  ya no existe, así que se comprueba antes de proponerlo.
- **Las tandas, en paralelo** con tope de la mitad de los núcleos, máximo cuatro (ADR 0018). Medido
  antes y después en la misma máquina: **veinte ficheros de 4,42 s a 1,29 s**. El tope no es
  prudencia vaga: cada derivación ya usa cuatro hilos y 64 MiB por dentro.
- Verificado: `make comprobar`, `make contraste`, **`go test -race` sobre todo**, que aquí importa
  porque es la primera vez que se cifra desde varias gorrutinas, y 26 pruebas de interfaz en los dos
  temas.
- **2.5.1**, al probarlo: el vidrio **no se notaba**. La cañería estaba bien —Wails mete un
  `NSVisualEffectView` detrás de la ventana—; el error fue calibrar el tinte contra una simulación
  sin desenfoque. macOS no enseña el escritorio, enseña un material ya suavizado, así que dejar pasar
  el 18 % de eso es no tener efecto. Bajado a 0,55.
- Y Ajustes pasa a decir si la ventana usa el vidrio del sistema: la primera vez no había forma de
  distinguir «no llega la señal» de «el tinte tapa demasiado», y eso costó una versión.
- **2.5.2, y aquí estaba el fallo de verdad**: seguía sin verse, «solo un gris». Ese gris plano es la
  pinta exacta de un `NSVisualEffectView` cuya ventana sigue siendo opaca. Wails crea el efecto con
  mezcla «BehindWindow» pero **nunca pone la ventana como no opaca**, y una `NSWindow` opaca compone
  como opaca aunque su color tenga alfa cero. Se remata desde `vidrio_darwin.go` con cgo, junto con
  el `underPageBackgroundColor` del `WKWebView`, que desde macOS 12 tapa igual.
- Lección: bajar el tinte dos veces sin entender el síntoma era el camino equivocado. El «gris plano»
  no era un tinte de más, era el material sin nada que mezclar.

## 2026-09-07 · Cambiar de sección deja de borrar lo escrito

- **2.9.0**, y era la deuda que dejó la versión anterior: cada sección se desmontaba al salir, así
  que escribir el secreto, ir a Generar a por una clave y volver dejaba el campo vacío. Justo el
  camino que la propia aplicación propone desde que existe «Usar como clave».
- Ahora cada sección **se monta la primera vez que se visita y luego se esconde**. Montarlas todas de
  golpe habría sido más simple y peor: Generar saca una contraseña nada más montarse, y una
  herramienta que cifra no debería fabricar un secreto que nadie ha pedido por si acaso.
- Tres cosas que arrastró el cambio, y ninguna era evidente:
  - **Los identificadores de los campos** pasan a llevar el nombre de la pantalla. Con cifrar y
    descifrar montadas a la vez, dos `id="clave"` dejan la etiqueta apuntando a cualquiera.
  - **`hidden` no esconde nada por sí solo** aquí: el `display: flex` de `.panel` le gana por ser de
    autor. Hace falta un `display: none !important`.
  - **Lo que se cargaba al montarse hay que recargarlo al entrar.** El historial enseñaba lo que
    había la primera vez que se miró, no lo recién cifrado.
- Las pruebas costaron más que el cambio, y por una razón que conviene recordar: **los paneles
  escondidos siguen en el DOM**, así que cualquier selector por clase encontraba dos. Van acotados
  con `:visible`.
- Dos pruebas propias resultaron mentirosas y se rehicieron: una contaba el historial **antes de que
  cargara** —veía cero y daba por hecho un vaciado que no había ocurrido— y otra leía la contraseña
  generada mientras React montaba los efectos dos veces en desarrollo, comparando la primera contra
  la segunda.
- Verificado: 36 pruebas en los dos temas, tres tandas seguidas, **y comprobado en el Mac**: lo
  escrito sobrevive al cambiar de sección.

## 2026-09-07 · La contraseña generada, usable como clave

- **2.8.0**: en Generar aparece «Usar como clave», que lleva a Cifrar con la contraseña puesta y de
  paso la copia; y en Cifrar, un «Generar una» junto al campo de la clave. Nada de esto toca Go: las
  dos llamadas ya existían.
- **Lo que mandó el diseño fue el riesgo**, no la comodidad. Una clave generada al azar no se
  recuerda, así que cuando la clave viene del generador el aviso de Cifrar **cambia de texto** y dice
  que esa clave no está guardada en ninguna parte. **Sustituye al genérico en vez de sumarse**: dos
  avisos diciendo lo mismo se leen menos que uno, y además hay una prueba que exige que haya
  exactamente uno.
- **Encontrado de paso, y es anterior a este cambio**: cambiar de sección borra lo escrito, porque
  cada panel se desmonta al salir. Se nota justo aquí —lo natural es escribir el secreto y luego ir a
  por la clave— pero pasaba igual antes. Comprobado volviendo atrás el código y midiéndolo. A la
  deuda.

## 2026-09-07 · El icono del disco montado

- **2.7.1**: el volumen del DMG usaba el icono de la aplicación, así que el disco montado y lo que
  hay dentro se veían **exactamente igual** —en la barra lateral del Finder no había forma de
  distinguirlos—. Ahora es un disco con la marca encima.
- El `.icns` se arma en el trabajo de macOS con `iconutil`, que solo existe allí; aquí se rasterizan
  los PNG del `.iconset` con `make icono`, y el guion vuelve al icono de antes si algo falla.
- Los nombres de los ficheros del `.iconset` los impone `iconutil`: si falta uno o se llama distinto,
  se niega a construir el fichero.
- **2.7.2, con dos correcciones del humano mirando su Mac**: el disco estaba **apaisado** y los del
  sistema van **de pie**; y la marca repintada en dorado sobre gris quedaba desvaída, así que va el
  icono de la aplicación **tal cual**, que lleva su propio fondo oscuro y se sostiene contra la cara
  clara del disco. Buscar la forma en la web no sirvió —los resultados hablan de iconos de
  aplicación, no de volúmenes—: lo resolvió quien tenía el sistema delante.
- **2.7.3, y aquí estaba lo de fondo**: con capturas de discos de verdad —una unidad del sistema y
  los volúmenes de FUSE y de Affinity— se vio que **un disco no se dibuja de frente**. Va en
  perspectiva, mirado desde arriba: carcasa que se estrecha hacia el fondo, placa en color con el
  logo grande arriba, banda con su piloto abajo. Y casi cuadrado, no alargado.
- Tres intentos: apaisado, vertical y plano, y por fin en perspectiva. Los dos primeros salieron de
  imaginar cómo era; el tercero, de mirar tres discos de verdad. La diferencia entre uno y otro
  método está en el resultado.
- **2.7.4**: preguntado si el disco valía también para Windows y Linux, la respuesta es que no —el
  icono de volumen existe porque en macOS se **monta** un disco, y ni el instalador de Windows ni el
  `.deb` montan nada—, pero de mirarlo salió una carencia de verdad: el `.deb` instalaba **un solo
  tamaño de icono**, el de 512. Ahora van los ocho del tema hicolor. En Windows no había nada que
  arreglar: Wails ya genera el `icon.ico` del instalador desde `appicon.png`.

## 2026-09-07 · Windows y GNOME, cada uno a lo suyo

- **2.7.0**: los tres sistemas tienen ya su estructura. La de fondo es la misma —navegación a un
  lado, contenido al otro, que es como se organizan los tres escritorios modernos— y lo que cambia
  son las formas, las densidades y quién dibuja el marco (ADR 0020).
- **El marco lo dibuja el sistema** en Windows y Linux; solo macOS se queda sin barra de título.
  Hacerlo a mano en los tres se parecería más, pero cerrar, maximizar y redimensionar pasarían a ser
  código nuestro y **aquí no hay dónde probarlo**.
- Windows lleva el panel de Fluent, con **la barra de acento a la izquierda de la fila activa**, que
  es lo que distingue un `NavigationView` de una barra lateral cualquiera. Linux va a lo GNOME:
  título centrado en negrita con su línea debajo.
- Aparece `App.Plataforma()` y un `data-sistema` en la raíz, igual que el `data-vidrio`. Con eso, el
  CSS de cada sistema cuelga de un sitio y no se mezcla.
- **Corregido un error que había escrito en la 0017**: dije que Wails no ofrecía translucidez en
  Linux, y sí la ofrece. No se usa por otra razón —sin desenfoque del compositor, la transparencia de
  GTK enseña el escritorio a pelo— pero la ficha decía algo falso.
- Verificado: 28 pruebas en los dos temas, dos tandas seguidas —y ahora ejercitan la variante de
  GNOME, porque el servidor de desarrollo corre en Linux—, más las tres variantes miradas una a una.
  **Sin comprobar**: Windows y GNOME de verdad.

## 2026-09-07 · La estructura de una aplicación de macOS

- **2.6.0**: la ventana deja de ser una página web dentro de un marco. Barra lateral con la
  navegación, título en la barra de herramientas, formularios en tarjetas agrupadas y sin barra de
  título propia —los semáforos caen sobre la barra lateral, como en Finder— (ADR 0019).
- **Las medidas no se inventaron.** El humano dejó cuatro capturas de macOS 26 en `referencias/`, y
  de ahí salen: los semáforos miden 12 pt clavados, así que sirven de regla para sacar la escala de
  cualquier captura. Medido: barra lateral **224,6 pt**, paso entre filas **38,5 pt**, semáforos a
  **21 pt** del borde. Esa técnica vale para cualquier captura futura.
- Dos cosas se vieron al primer render y no antes: las filas de la barra lateral salían con una
  **línea separadora** que era la sombra del botón de acción heredada, y los **emoji de color**
  desentonaban —el sistema usa trazo monocromo—. Los iconos se dibujan ahora en SVG, que además
  esquiva lo de los SF Symbols, que no se pueden detectar desde CSS.
- En las pruebas apareció una colisión nueva: **«Cifrar» nombra a la vez la sección y el botón que
  cifra**, así que los selectores se acotan a `.lateral` o a `.contenido`.
- Verificado: 26 pruebas de interfaz actualizadas y en verde en los dos temas, dos tandas seguidas;
  `make comprobar` y `make contraste`. Capturas de la portada rehechas.
- **2.6.1**, con lo primero que dijo el Mac: el título se veía con fondo y el resaltado del ratón
  pisaba la fila activa.
  - Lo del fondo no era CSS sino macOS: `TitleBarHiddenInset` activa `UseToolbar` y el sistema dibuja
    **su propia banda** justo donde va nuestro título. Con `TitleBarHidden` desaparece.
  - Lo del hover era un **empate de especificidad**: `.lateral button` pesa lo mismo que la regla
    general de `button:hover`, que va más abajo en el fichero y por eso ganaba. Los selectores llevan
    ahora `nav` para desempatar, y tiene prueba: esos empates vuelven solos en cuanto alguien añade
    una regla al final.
  - De paso, la prueba del interruptor de Ajustes **se prepara su propio punto de partida**. Daba por
    hecho que arrancaba encendido, y el fichero de preferencias del servidor de desarrollo es de
    verdad: una tanda interrumpida lo dejaba apagado y a partir de ahí fallaba siempre.
- **2.6.2**: el fondo del título seguía ahí, y esta vez sí era CSS mío. Bajo el vidrio, **lo que no
  declara fondo se vuelve transparente y enseña el material**; la barra de herramientas se había
  quedado sin declararlo al reestructurar, así que se veía una banda encima del contenido. El fondo
  pasa a la columna entera.
- De paso, el reparto del vidrio se rehace por columnas: **el material va en la barra lateral**, que
  es donde lo pone macOS, y la zona de trabajo se queda opaca. Puede que además resuelva lo de que el
  vidrio no se apreciaba: la barra lateral es una superficie grande, y hasta ahora el material solo
  tenía dos franjas finas donde asomar.
- **Sin comprobar, y es lo que decide lo siguiente**: cómo queda en un Mac. De eso depende si Windows
  y Linux se hacen igual.

## 2026-09-07 · Lo que dijo el Mac

- **Copiar y pegar funcionan** con los atajos de ⌘. Era el trozo con más riesgo de todo lo escrito
  el día anterior: al construir los menús a mano (ADR 0015) se perdieron los selectores nativos y
  esas acciones pasan por código propio, con el pegar repartido entre Go —que lee el portapapeles— y
  la interfaz —que coloca el texto en el cursor—.
- Queda abierto lo del aviso de Gatekeeper al actualizarse desde dentro.

## 2026-09-07 · Cierre de la jornada

De la 2.0.3 a la **2.4.0** en una sesión, todas publicadas y todas con su ficha cuando la decisión lo
merecía. El orden en que salieron cuenta bien lo que pasó: primero la distribución, luego enterarse
de que hay versión nueva, luego traérsela, luego instalarla sin que nadie arrastre nada, y por el
camino dos cosas que se daban por cerradas y no lo estaban.

**Lo que se corrigió de decisiones anteriores**, que es lo que más vale conservar:

- La **0014** daba por hecho que reemplazarse a sí misma exige firmar con Apple. No es cierto: la
  firma evita el aviso de Gatekeeper, no habilita el reemplazo. Lo arregla la **0016**.
- La **2.0.1** dio por cerrado el doble clic en un `.esf` porque enganchó `Mac.OnFileOpen`. Estaba a
  medias: nadie escuchaba el evento. Cerrado en la **2.3.1**.
- El `SHA256SUMS` publicado tuvo **dos fallos seguidos**: primero solo cubría los binarios de la
  línea de comandos, y luego se incluía a sí mismo con un resumen calculado a medio escribir.

**Al volver, lo primero es leer las preguntas abiertas de `estado.md`**: hay dos comprobaciones que
solo puede hacer el humano en su Mac —copiar y pegar con los menús nuevos, y si el aviso de
Gatekeeper desaparece al actualizarse desde dentro— y lo que venga después depende de lo que salga
de ahí.

## 2026-09-07 · Un .esf se abre en la pantalla que le toca

- **2.4.0**: abrir un `.esf` llevaba siempre a la pantalla de ficheros, y eso es un lío cuando lo que
  lleva dentro es un texto: ahí lo que se quiere ver es el secreto, no otro fichero al lado.
- Un `.esf` puede ser dos cosas y la extensión no lo dice: un fichero cifrado, o la línea `ESF1.…`
  que sale de cifrar un texto y que alguien guardó. **Los dos empiezan por la misma magia**; lo que
  los separa es el byte siguiente —la versión en el binario, el punto en el de texto—. De ahí sale
  `cripto.FormaDe`, que es donde tiene que vivir porque es el formato quien lo sabe.
- `AperturaDe` mira dentro y decide la pantalla. Con varios ficheros van todos como ficheros: en la
  pantalla de texto no cabe más que uno y elegir cuál sería adivinar.
- Verificado: seis casos de la detección de formato, cinco de la apertura, y dos pruebas de interfaz
  —una fabrica un `.esf` de texto de verdad, cifrando y guardando, y comprueba que al abrirlo sale
  en la pantalla de texto y se descifra desde ahí—. 24 pruebas de interfaz en los dos temas, dos
  tandas seguidas.

## 2026-09-07 · El doble clic en un .esf, arreglado de verdad

- **2.3.1**: la 2.0.1 dio por cerrado el doble clic en un `.esf` porque enganchó `Mac.OnFileOpen`.
  Estaba a medias: el evento se emitía y **la interfaz no lo escuchaba**. Con la ventana ya abierta
  el doble clic no hacía nada, y al abrir la aplicación con un fichero había una carrera —si la
  ventana preguntaba antes de que macOS entregara el fichero, se perdía—.
- Son dos momentos distintos y hay que tratarlos distinto: lo que llega **antes** de que haya alguien
  escuchando se guarda, y lo que llega **después** se avisa. Y la interfaz **se suscribe antes de
  preguntar**, que al revés deja el hueco por el que se colaba el fallo.
- De paso: varios ficheros a la vez —macOS manda un evento por cada uno— y la pantalla de descifrar
  se rehace si llega otro con ella ya abierta.
- Verificado: cuatro pruebas de Go de los dos momentos, con `-race`, y una de interfaz que recorre el
  camino entero. Una prueba anterior dependía del orden —el servidor de desarrollo es uno solo y se
  acuerda de la novedad—, y se ha quitado esa dependencia; tres tandas seguidas en verde.
- **Sin comprobar**: si el Finder enseña el icono propio del documento. Anotado en la deuda.

## 2026-09-07 · Actualizarse de verdad, sin arrastrar nada

- Probada la 2.2.0 en el Mac: la descarga va, pero al instalar salía la ventana de arrastrar a
  Aplicaciones. Es decir, el trabajo se lo acababa haciendo el usuario, que es lo que se quería
  quitar.
- **La 0014 estaba mal en un punto y hay que decirlo**: daba por hecho que reemplazarse exige firmar
  con Apple. No es cierto. La firma evita el aviso de Gatekeeper; el reemplazo solo necesita permiso
  de escritura donde vive la aplicación y hacer el cambiazo desde fuera del proceso. Corregido en la
  0016.
- **2.3.0**: Esfinge se sustituye y se reinicia sola. Lo hace un guion que ella escribe, lanza fuera
  de su grupo de procesos —para que cerrar la ventana no se lo lleve por delante— y que espera a que
  el proceso muera antes de tocar nada. En macOS monta el DMG, saca el `.app` con `ditto` y lo
  cambia; en Windows lanza el NSIS en silencio; en Linux no, porque el `.deb` instala como root.
- Quién puede hacerlo no lo decide el sistema sino **si se puede escribir donde vive la aplicación**,
  y eso se comprueba escribiendo: los permisos de `Stat` no valen en macOS, que tiene listas de
  control de acceso.
- El botón dice ahora «Instalar y reiniciar» o «Abrir el instalador» según lo que se pueda hacer en
  esa máquina, con el modo viajando dentro de la novedad.
- Verificado: tests del reparto, compilación para los tres sistemas, y las 22 pruebas de interfaz.
  Una de ellas era inestable —la orden del menú se emitía antes de que el navegador enganchara el
  flujo de eventos— y se hizo robusta reintentando; se comprobó con tres tandas seguidas.
- **Sin verificar, y es lo que importa**: el cambiazo de verdad. Aquí no hay Mac ni Windows.

## 2026-09-07 · La barra de menús, en español

- Probada la 2.1.0 en el Mac: la comprobación funciona —con la última instalada no sale la banda y
  «Buscar ahora» dice que ya se está al día—, que es exactamente lo que tenía que pasar.
- **2.2.0**: la barra de menús del sistema salía en inglés. No se podía traducir: los roles de Wails
  llevan los rótulos escritos a fuego en su Objective-C y no son los que localiza macOS. Se construye
  entera (ADR 0015), igual en los tres sistemas.
- Eso obliga a hacer las acciones de edición por nuestra cuenta, porque sin roles no hay selectores
  nativos: el menú manda una orden y la ventana la ejecuta sobre el campo que tiene el foco. El pegar
  necesita las dos mitades —el portapapeles lo lee Go, el texto lo coloca la interfaz—, que es el
  camino de más riesgo de todo esto: si falla, falla pegar una contraseña.
- De paso, atajos a las cinco pantallas (⌘1 a ⌘5), que antes no había forma de alcanzar sin ratón.
- Verificado: prueba de interfaz en los dos temas que recorre el camino entero desde que Go manda la
  orden, y prueba de Go de que salen por el mismo canal de eventos que el progreso. **Sin verificar**:
  los menús dibujados de verdad, que no hay Mac ni Windows aquí.

## 2026-09-07 · Que la aplicación se entere de sus propias versiones

- **2.1.0**: Esfinge comprueba una vez al día si hay versión nueva, se descarga el instalador de su
  sistema comprobando el SHA256 mientras baja, y lo abre. No se reemplaza a sí misma: eso exige
  firmar con Apple, que está descartado desde la 0012. Ninguno de los tres sistemas pide desinstalar
  antes.
- Paquete nuevo `internal/actualizacion`, sin dependencias fuera de la biblioteca estándar y sin
  saber nada de Wails, para poder usarlo también desde la línea de comandos y probarlo entero contra
  un servidor de mentira.
- **Lo primero que hubo que arreglar no estaba en el plan**: el `SHA256SUMS` que se publicaba solo
  cubría los seis binarios de la línea de comandos. El DMG, el instalador y el `.deb` salían sin
  resumen, así que no había contra qué comparar una descarga. Ahora se rehace en el trabajo de
  publicar, sobre todos los adjuntos.
- Aparecen dos cosas que no existían: un fichero de preferencias —junto al historial, con sus mismos
  permisos— y una quinta pestaña, **Ajustes**, que es donde se cuenta con todas las letras qué se
  envía y dónde se apaga. Eso es la contrapartida de haber elegido comprobación automática: la
  portada decía que nada salía del ordenador y ha habido que matizarlo, en el README y en
  `seguridad.md`.
- En la línea de comandos el aviso va por la salida de error, nunca por la estándar, **solo si esa
  salida es un terminal** —en una tubería o un cron no se escribe ni se pregunta— y con un plazo de
  cortesía: si la red no contesta, el comando no espera.
- Una trampa del diseño de Wails que conviene recordar: **todo método exportado de `*App` queda
  expuesto a la interfaz**, así que `ApuntarAAPI` tuvo que ser función y no método, para que la
  ventana no pueda apuntar la comprobación a donde quiera.
- Verificado: 21 tests de Go nuevos contra un `httptest.Server` —incluido que con el interruptor
  apagado no se hace **ni una** petición, que se cuenta en vez de suponerse—, `make comprobar` en
  verde con y sin `-tags dev`, `-race` limpio, y 10 pruebas de interfaz en los dos temas contra una
  API de mentira.
- Publicada la 2.1.0, se vio otro fallo del `SHA256SUMS`: se incluía **a sí mismo**, con el resumen
  del fichero a medio escribir —la redirección lo crea antes de que corra `find`—. Un
  `sha256sum -c` daba FAILED justo en el fichero que sirve para confiar. Corregido en el flujo, que
  además ahora lo verifica, y reemplazado en la publicación ya hecha: las demás líneas eran buenas,
  comprobado descargando el `.deb`.
- Queda abierto lo que no se puede ver desde aquí: una actualización de verdad, y si al descargar el
  DMG desde Go la copia instalada se libra del aviso de Gatekeeper.

## 2026-09-07 · Infraestructura: documentos, repositorio e instaladores

- Se montó este sistema de documentos vivos siguiendo la convención de Tempero, con trece decisiones
  escritas hacia atrás a partir de lo que ya estaba en `CLAUDE.md` y en los mensajes de los commits.
- `gitleaks` sobre el historial completo: 11 commits, sin filtraciones. Era el requisito para hacer
  público el repositorio.
- Portada del repositorio con capturas, tabla de descargas y distintivos; licencia propietaria que
  permite leer y auditar el código pero no reutilizarlo. Lo técnico que estaba en el README se mudó
  a `docs/`.
- Instaladores de los tres sistemas: DMG con fondo de marca y LÉEME dentro
  (`empaquetado/macos/armar-dmg.sh`, la misma receta para `make dmg` y para el flujo), instalador
  NSIS para Windows, y `.deb` con `nfpm` que lleva las dos caras —ventana y línea de comandos—, su
  lanzador y la asociación de los `.esf`. Los instaladores de doble clic de la época del ZIP se
  retiraron: los sustituye el DMG.
- Flujo `publicar.yml`: se dispara con una etiqueta `v*`, compila en los tres sistemas y cuelga todo
  de la publicación de GitHub.
- Verificado: `make comprobar` en verde, `gitleaks` sin filtraciones, los tres YAML válidos.
- **2.0.1** publicada: el flujo entero de punta a punta, con sus adjuntos. Salió en verde con un
  fallo escondido: Chocolatey instala NSIS pero no lo deja en el `PATH`, y `wails -nsis` avisa de que
  no encuentra `makensis` y **sale con código cero**. El trabajo pasó y la publicación se quedó sin
  instalador de Windows. **2.0.2** lo arregla: se añade NSIS al `PATH` y se comprueba que el fichero
  existe, que es la misma lección que ya estaba escrita para `create-dmg`.
- Del `.dmg` publicado, leído desde Linux con 7-Zip: `Esfinge.app` con el binario universal —x86_64
  y arm64—, el enlace a `/Applications`, el `LÉEME.txt` idéntico al del repositorio, el fondo en
  `.background/fondo-dmg.tiff` con las dos resoluciones, el icono de volumen y un `.DS_Store` de 10
  KB, que es la señal de que `create-dmg` sí habló con el Finder y guardó la colocación.
- Del `.deb`: los dos binarios con su bit de ejecución, el lanzador, la asociación de los `.esf`, el
  icono, el copyright y las dependencias del webview.
- El instalador de Windows costó tres intentos y cada uno enseñó algo distinto: que Chocolatey no
  deja NSIS en el `PATH`, que `GITHUB_PATH` quiere la ruta como la entiende Windows y no la de Git
  bash —con la de bash el fichero está y la comprobación pasa, pero wails sigue sin encontrarlo—, y
  que Git bash convierte los argumentos que empiezan por «/», así que `makensis /VERSION` hay que
  ejecutarlo en PowerShell. Comprobado del `.exe` publicado: es un NSIS-3 Unicode y lleva dentro la
  aplicación, el `esf.ico`, el instalador de WebView2 por si el sistema no lo trae, y en su cabecera
  las claves que registran el `.esf` —`Software\Classes`, `DefaultIcon`, `shell\open\command`—.
- **2.0.3**: montado el DMG en el Mac, la imagen estaba bien salvo el `LÉEME.txt`, cuyo nombre caía
  encima del aviso de Gatekeeper del fondo. El Finder centra cada icono en la posición que se le da
  y **escribe el nombre debajo**, unos 64 px más abajo con iconos de 96: ese espacio no se puede
  usar para dibujar. La ventana pasa a 660×470 y el LÉEME baja a su propia banda, con el aviso a su
  derecha en vez de debajo.
- De ahí sale `make ventana-dmg`, que dibuja la ventana con los iconos donde los pondrá el Finder
  **leyendo las posiciones del propio `armar-dmg.sh`**, para que no puedan separarse. Es la forma de
  ver esto sin un Mac, que era justo lo que faltaba.

## 2026-09-07 · La aplicación de escritorio, y arreglarla con lo que dijo el Mac

- **2.0.0**: se retiraron los menús de terminal (1.830 líneas y 1.492 de test) y se construyó la
  aplicación con Wails. Repositorio creado en GitHub y compilación de los tres sistemas.
- Cuatro intentos hasta que compiló, y las cuatro cosas solo se ven compilando de verdad: dónde
  tiene que vivir el `main.go`, que las asociaciones de fichero van dentro de `info` en `wails.json`,
  qué versión de webkit busca Wails en Linux, y que los artefactos de GitHub pierden el bit de
  ejecución.
- **2.0.1**, con lo que salió de abrirla en un Mac: el arrastrar y soltar no hacía nada —el modo
  «zona» exige declarar una propiedad CSS que no se declaraba—, el doble clic en un `.esf` abría la
  ventana vacía —Wails sí lo entrega, por `Mac.OnFileOpen`—, y el aspecto «no parecía nativo», que
  obligó a rehacerlo con barra translúcida, radios mayores y bloques agrupados.
- El generador pasó a medir en caracteres además de en bits, ligados entre sí.
- Verificado: `make comprobar`, 16 pruebas de interfaz en los dos temas, y los tres sistemas
  compilando en verde.

## 2026-09-04 · De cero a herramienta de terminal

- Se construyó Esfinge entero: núcleo criptográfico, línea de comandos e interfaz de menús.
- Se instaló Go en `~/.local/go`, sin tocar el sistema.
- De la 1.0 a la 1.5.0 en una sesión, con lo que fue saliendo al probarlo: el truncado que culpaba a
  la clave, el ratón que no iba en la pantalla de cifrar —la vista se salía del alto y las
  coordenadas dejaban de cuadrar—, el guardado que se pisaba a sí mismo, y el arranque de cinco
  segundos por el `init()` de Bubble Tea.
- Instaladores de doble clic para macOS y Linux.

---

## Plantilla

```
## AAAA-MM-DD · <título>

- Qué se hizo o se decidió.
- Qué se verificó, y con qué.
- Qué queda abierto (→ mover a siguiente.md o deuda.md si procede).
```
