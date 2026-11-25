# ADR 0072 · Lo que un paquete de Linux dice de sí mismo, y en qué idioma

**Fecha**: 2026-11-25
**Estado**: aceptada

## Contexto

El cliente quiso instalar Esfinge de cero en su VM de Ubuntu **con la interfaz gráfica**, para ver el
instalador y sus textos. Es la primera vez que se mira ese camino: hasta hoy se instalaba con `apt` o
con el `.deb` suelto desde una terminal, y lo que una tienda o un instalador gráfico enseñan no lo
había revisado nadie.

Salieron dos cosas.

**Una, nuestra: el paquete no llevaba metadatos de AppStream.** Son los que lee un centro de software
—el App Center de Ubuntu, GNOME Software, Discover—, y sin ellos Esfinge sale ahí como una entrada
pelada, sin nombre, sin descripción y sin icono; ya instalada, **no aparece como aplicación
instalada**. El `.desktop` no cubre eso: lo lee el menú de aplicaciones, no la tienda.

**Dos, de quien lo enseña: la ventana que abre un doble clic sobre el `.deb` convierte la descripción
a ASCII** y pone `?` donde hay un acento. El cliente leyó **«contrase?as» en la primera línea de un
gestor de contraseñas**. Lo nuestro es correcto —la política de Debian pide los ficheros de control
en UTF-8 (5.1) y eso es lo que llevamos, medido byte a byte en el paquete publicado— y le pasa igual
a **23 de los 966 paquetes instalados en este servidor**, `gh` entre ellos con su apóstrofo
tipográfico. Pero esa línea es lo primero que lee quien instala, y la propia política avisa de que en
muchos sitios es lo único que se ve (3.4.1).

## Decisión

**1 · El paquete lleva su ficha de AppStream** (`empaquetado/linux/esfinge.metainfo.xml`), y **la
llevan los tres empaquetados**: el `.deb` y el `.rpm` por `nfpm.yaml`, la AppImage por su guion.

**2 · El inglés va por defecto y el español es la traducción, diciendo en el inglés que el programa
está en español.** Vale para la ficha y para la descripción del paquete. Es la misma regla que el
listado de Chocolatey —cuyo moderador devolvió una ficha escrita sólo en español en noviembre— y lo
que necesita saber quien lo lee antes de instalarlo.

**3 · La primera línea de la descripción va en ASCII puro**, y es **literalmente la misma frase** que
el `<summary>` por omisión de la ficha. El español entero queda debajo.

### Lo que no se cambia sin preguntar

- **El identificador es de DNS invertido y es el mismo que el del puente con el navegador**
  (`com.webcafeina.esfinge`). Lo que lo ata al lanzador es `<launchable>`, que existe para eso, así
  que **`esfinge.desktop` y los ocho iconos se quedan como están**.
- **El fichero instalado se llama como el identificador.** Si no, el catálogo no lo recoge.
- **El resumen y el `<summary>` son la misma frase**, y hay una prueba que las compara: son dos
  sitios diciendo lo mismo, o sea una regla en dos listas.

## Alternativas descartadas

**Dejar la descripción en español, que era la opción evidente.** El fallo es del instalador y nuestro
texto cumple la política; `apt`, GDebi y el terminal lo enseñan bien. Se descartó porque el coste lo
paga el cliente de un gestor de contraseñas leyendo «contrase?as» en la primera línea. **Lo eligió él
entre tres opciones, con las tres pantallas delante.**

**Quitarle los acentos sólo a la primera línea, manteniéndolo todo en español.** Dejaba limpio lo que
siempre se ve, pero obligaba a renunciar a la palabra «contraseñas» justo donde más dice.

**Renombrar el lanzador y los iconos a DNS invertido**, que es lo que pide la especificación para el
identificador. Arrastra el `Icon=`, las ocho rutas de `hicolor` y el `StartupWMClass`, y en este
proyecto los iconos ya se han confundido **dos veces**. `<launchable>` lo resuelve sin tocar nada.

**Capturas en la ficha.** AppStream las quiere como URL absoluta, así que harían falta imágenes
publicadas y mantenidas **a cada fase de la nativización**. Referenciar enlaces que no existen deja
la ficha peor que sin ellas. En `deuda.md`.

**`<releases>` con el historial de versiones.** Tendría que llevar la versión dentro y **nfpm no
expande el entorno dentro de los ficheros que empaqueta**, así que haría falta un paso de sustitución
más en el flujo de publicar — justo donde este proyecto se ha caído tres veces seguidas. La versión
que la tienda enseña sale igual del propio paquete.

## Consecuencias

- Quien haga `apt show esfinge` con el sistema en español **lee el inglés primero**. Es el precio, y
  se eligió a sabiendas.
- **En el `.rpm` sale una línea con un punto suelto**: el ` .` que separa los dos párrafos es como se
  hace en un fichero de control, y esa descripción la comparten los dos paquetes. Se acepta porque el
  `.deb` es el que se instala a mano.
- **El español de debajo sigue saliendo con `?`** en ese instalador. Esto no lo arregla: hace que lo
  primero que se lee no dependa de quien lo enseña.
- **Una corrección que conviene que conste**: se dijo aquí que la política de Debian exige la
  descripción en inglés, y **es falso**. Leídas las secciones 5.6.13 y 3.4, no dicen nada del idioma;
  lo único que aplica es que los ficheros de control van en UTF-8. El inglés es **una elección
  nuestra**, no una norma, y tampoco existe ninguna regla sobre el punto final del resumen.

## Verificación

**Lo que se comprobó, y con qué.**

- **Los bytes del paquete publicado**, con `cat -A`: el control, el lanzador y la ficha llevan la eñe
  y las vocales acentuadas en UTF-8 correcto. O sea que el dato nunca fue el problema.
- **El control que genera `nfpm`**, construyendo el `.deb` y el `.rpm` **aquí** con su versión fijada,
  no suponiendo cómo formatearía la descripción: resumen en la primera línea, continuaciones con su
  espacio y el ` .` como separador de párrafo.
- **La ficha extraída del `.deb` publicado**, con `appstreamcli validate --pedantic`: conforme, con un
  único aviso, `releases-info-missing`, que es el que se dejó fuera a propósito.
- **Que el `?` es una conversión y no una pérdida**: `iconv -f UTF-8 -t ASCII//TRANSLIT` da exactamente
  «contrase?as», y sin transliterar da «contraseas». Lo que el cliente ve es lo primero.
- **Que no es nuestro**: 23 de 966 paquetes instalados aquí llevan caracteres no ASCII en su
  descripción, así que ese instalador los rompe todos.
- **Seis pruebas**, en `internal/adjuntos/`, cada afirmación comparando **dos fuentes distintas** —el
  identificador contra el del puente, el lanzador y el icono contra lo que el paquete instala, el
  nombre del fichero contra el identificador, el resumen contra el `<summary>`— y **todas mutadas para
  verlas rojas**.
- **Y una nació en verde**, que es lo que hay que aprenderse: preguntaba si `esfinge.metainfo.xml`
  **aparecía en** el guion de la AppImage, y aparece también en el comentario que lo explica, así que
  comentar la orden de copia no la ponía roja. Es la trampa del tap de Homebrew otra vez. Ahora afirma
  **la orden**, sobre el fichero sin comentarios.

**Lo que no se ha comprobado.** Aquí no hay entorno gráfico, así que **ninguna tienda ha enseñado esta
ficha**: no se sabe si el App Center, GNOME Software o Discover la leen para un `.deb` local, ni si el
icono llega a salir en esa ventana. Tampoco **qué programa** abre el `.deb` en su Ubuntu 26.10 —el
doble clic lo abrió algo, y de ahí viene el `?`, pero no se identificó—. Eso se mira en su VM.
