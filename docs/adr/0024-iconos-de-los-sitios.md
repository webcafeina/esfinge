# ADR 0024 — Los iconos de los sitios, y la segunda conexión

**Fecha:** 2026-09-09 · **Estado:** aceptada · **Matiza la [0014](0014-comprobacion-de-actualizaciones.md)** ·
**Revisar si** aparece sincronización, o si el cliente cambia de opinión sobre el ajuste

## Contexto

Con sesenta y cinco entradas dentro, la bóveda dejó de recorrerse con el ojo. La
2.13.0 puso un cuadro con la inicial y un color estable por sitio, que ayuda
mucho; el cliente pidió además **el icono real de cada web, como hace Dashlane**.

Y ahí está el problema: **el logo real hay que ir a buscarlo a la red**, y desde
la 2.1.0 la portada de Esfinge y `docs/seguridad.md` prometen que el programa
hace **una** conexión —una consulta al día a GitHub— con estas palabras: «Es lo
único que Esfinge hace fuera de tu ordenador».

## Decisión

**Se descargan, pidiéndoselos a cada sitio directamente, y el ajuste viene
encendido.**

Las tres partes las decidió el cliente con las consecuencias delante, y la
tercera después de que se le corrigiera un dato que se le había dado mal.

### Por qué a cada sitio y no a un servicio

Un servicio de iconos —el de Google es el habitual— recibiría **la lista completa
de sitios donde el cliente tiene cuenta**. En un gestor de contraseñas eso no se
puede hacer, y no hay atenuante: es entregarle a una empresa el índice de la
bóveda.

**Y conviene decir lo que ir directo *no* consigue**, porque al plantearlo se dijo
mal: no oculta la lista, **la reparte**. El nombre del sitio viaja en claro en la
consulta de DNS y en el saludo TLS, así que quien mire la red ve exactamente a
qué sitios se pregunta. Lo que se gana yendo directo es **no meter un tercero de
confianza dentro de un programa que guarda contraseñas**, que es una razón
distinta y suficiente por sí sola.

### Por qué encendido, y qué se hace a cambio

Lo pidió el cliente sabiendo lo anterior. La contrapartida es la costumbre de esta
casa desde la [ADR 0014](0014-comprobacion-de-actualizaciones.md): **si se hace,
se dice, y se deja apagar**. Así que la primera vez —una sola— la ventana explica
qué va a pasar, con el dato incómodo delante, y ofrece «No, gracias» al lado de
«De acuerdo». Y el interruptor vive en Ajustes, junto al de las versiones.

### Dónde se guardan: un fichero satélite cifrado

`boveda.esfinge.iconos`, sellado con la misma clave de bóveda y con
`PerfilLlave`, con un mapa de anfitrión a icono.

**No en una carpeta de iconos en claro**, que era lo evidente: una carpeta con
`banco.es.png` y `hacienda.es.png` **dice qué sitios hay en la bóveda**, que es
justo lo que la bóveda existe para ocultar. Hashear el nombre no salva nada: un
diccionario de dominios se invierte en segundos.

**Y tampoco dentro de la entrada**, que era lo cómodo —y que `Extra` incluso
habría protegido de las versiones viejas—, por tres cosas:

- **Cada guardado de la bóveda lee el fichero entero, copia el anterior entero y
  escribe el nuevo con `fsync`.** Sesenta y cinco iconos lo multiplican por diez,
  y esa cuenta se pagaría al confirmar **cada edición de una contraseña**.
- **La lista de la ventana viaja entera en cada tecla del buscador.** Con el icono
  dentro, teclear «ban» serían tres viajes de todos los iconos por el puente.
- Un icono es **dato derivado**: se tira y se vuelve a traer. Dentro de la bóveda
  pasaría a ser dato, y un dato hay que exportarlo, migrarlo y respetarlo siempre.

### Qué se acepta

Solo mapas de bits, y **se comprueba decodificándolos** con la biblioteca
estándar. Decodificar es lo que demuestra que es una imagen: el caso más común de
un sitio sin icono no es un 404, es un 200 con la página de error dentro.

Se mira la **cabecera antes de decodificar** —un PNG de treinta kilobytes puede
declarar 30.000×30.000 y eso son 3,6 GB de píxeles— y se reduce a 32×32 con un
promediador escrito a mano. Reducir es lo que decide si esto cabe: un
`apple-touch-icon` viene a 180×180 y pesa entre diez y cuarenta kilobytes.

Y re-codificar tiene un segundo efecto que vale por sí solo: **lo que se guarda no
es el fichero del sitio**, son píxeles re-emitidos por nuestro codificador.

## Alternativas descartadas

- **Un servicio de iconos.** Arriba: le entrega a un tercero el índice de la
  bóveda.
- **Analizar el HTML** de cada sitio para encontrar el `<link rel="icon">`. Sería
  lo completo, y obligaría a bajarse la portada de cada sitio de la bóveda —mucho
  más tráfico y mucha más superficie— y a meter un analizador de HTML en el
  binario. Sigue descartado, y ahora con un número al lado: sin él se cubren
  **nueve de doce** sitios de prueba, y los dos que faltan de verdad declaran su
  icono ahí.
- ~~**`/favicon.ico`**~~ → **entró después de medir**, y es la corrección más
  interesante de esta ficha. El argumento para dejarlo fuera era bueno —la
  biblioteca estándar de Go no lo sabe decodificar— pero al probar contra sitios
  reales solo cuatro de doce daban icono, y varios de los que faltaban lo tenían
  justo ahí. Se lee como **el contenedor que es**: se saca la imagen más grande
  del índice y, si es un PNG, se le pasa al decodificador de siempre.
- ~~**Descodificar el mapa de bits que llevan algunos ICO**~~ → **también entró
  después de medir**. Google, Amazon y Netflix sirven exactamente eso, y los tres
  lo hacen con la misma variante: **32 bits sin comprimir**. Se lee solo ésa, que
  es la única que no necesita paletas ni descompresión, y todo lo demás se deja
  pasar. La aritmética se comprueba contra el tamaño real antes de tocar un byte.
- **`golang.org/x/image` para reescalar.** Sería un módulo nuevo de verdad en un
  binario que hoy tiene cuatro dependencias directas. El promediador son cuarenta
  líneas.
- **Guardar el icono tal como lo sirve el sitio**, sin reducir. Multiplicaría por
  veinte el tamaño y guardaría bytes de un tercero sin tocar.

## Consecuencias

- **Esfinge deja de hacer una sola conexión.** `docs/seguridad.md`, el README y el
  documento de paquete de `internal/actualizacion` decían «una» y ya no es cierto.
- **Quien mire la red ve la lista de sitios de la bóveda.** Es lo que se compra a
  cambio de los iconos, y está escrito en la ventana antes de que ocurra.
- **`ESFINGE_SIN_RED` pasa a ser un freno de verdad.** Antes lo miraba solo la
  línea de comandos —la ventana no lo consultaba nunca—, así que quien lo ponía
  creyendo que apagaba la red apagaba la mitad. Ahora vive en `internal/red` y lo
  consultan las dos salidas.
- **La bóveda gana un cerrojo.** Ya había dos gorrutinas tocándola —la ventana y
  el tic del bloqueo, que llama a `Cerrar()`— y era una carrera estrecha que nadie
  había pillado. Con descargas de fondo se volvía ancha.
- **El goteo no toca `Actividad()`.** Si lo hiciera, una bóveda estaría abierta
  indefinidamente mientras baja iconos, y el bloqueo por inactividad dejaría de
  significar lo que dice.

## Verificación

- **Ocho pruebas del descargador**, y son las que importan: que una página de
  error no pasa por icono; que una imagen que dice medir 30.000 píxeles no se
  abre; que lo que llega se lee con tope; que **no se toca ninguna dirección
  privada** —incluida la de metadatos de las nubes y la franja del operador— ni
  por nombre ni por IP; que una redirección no puede bajar a texto claro ni pasar
  de tres saltos; y que **el sitio no se entera de que quien pregunta es Esfinge**.
- Del almacén: que el fichero **no contiene ningún dominio en claro**, que se
  mezcla en vez de sustituir, que una caché rota no rompe nada y que **un sitio
  sin icono no se vuelve a preguntar** hasta pasado un mes.
- De los frenos: que con `ESFINGE_SIN_RED` y con el ajuste apagado no se arranca
  el goteo, y que los anfitriones de una intranet o de una red privada **no entran
  siquiera en la lista de candidatos**.
- Todo el paquete de la bóveda con `-race`, ahora que hay trabajo de fondo.

## Lo que enseñó medirlo contra sitios de verdad

La primera versión **no funcionó en absoluto**, y ninguna prueba lo vio: el filtro
de direcciones privadas estaba en `DialContext`, que recibe **el nombre sin
resolver**, así que `ParseIP("github.com")` daba nulo, la regla «si no se sabe qué
es, no se va» se cumplía y **se rechazaban todos los sitios del mundo**. Lo dijo
el cliente —ninguna entrada tenía icono— y se encontró en un minuto probando el
descargador contra doce dominios reales. El filtro vive ahora en `Control`, que
corre después de resolver.

Las pruebas no lo cazaron por una razón que conviene recordar: las que hablaban
con un servidor llevaban el filtro aflojado, y la que sí lo ejercitaba usaba
`127.0.0.1`, que **sí** es una dirección. Pasaba por el motivo correcto y por la
razón equivocada. Hay una prueba nueva que pide por un nombre y comprueba que el
rechazo habla de la dirección resuelta.

Y con el camino ya funcionando, la medida decidió dos cosas que estaban
descartadas de antemano —el `.ico` y su mapa de bits—: de **4 de 12** se pasó a
**9 de 12**.

**Lo que sigue sin comprobarse:** cuántos de los sesenta y cinco sitios del
cliente dan icono. Doce dominios conocidos no son una bóveda real.
