# Esfinge

Cifra y descifra contraseñas, ficheros de credenciales y cualquier otro secreto con una clave que
solo conocen las dos partes. De **Webcafeína**.

Un binario suelto, sin nada que instalar. Abierto sin argumentos enseña menús; con argumentos es un
comando corriente que se deja meter en una tubería.

```
█████ █████ █████ █████ █   █ █████ █████
█     █     █       █   ██  █ █     █
████  █████ ████    █   █ █ █ █  ██ ████
█         █ █       █   █  ██ █   █ █
█████ █████ █     █████ █   █ █████ █████
Cifra y descifra secretos con una clave
```

## Aviso primero

**Sin la clave no hay forma de recuperar nada.** Esto no es una cuenta con «he olvidado mi
contraseña»: si se pierde la clave, el contenido se ha perdido, y no hay nadie —tampoco Webcafeína—
que pueda abrirlo. Guarda la clave antes de cifrar, no después.

## Instalación

Descarga el binario de tu sistema y ya está.

| Sistema | Fichero |
|---|---|
| macOS con chip Apple (M1 y posteriores) | `esfinge-darwin-arm64` |
| macOS con Intel | `esfinge-darwin-amd64` |
| Windows | `esfinge-windows-amd64.exe` |
| Linux | `esfinge-linux-amd64` |

En **macOS y Linux** hay que darle permiso de ejecución la primera vez:

```sh
chmod +x esfinge-darwin-arm64
```

En **macOS**, además, el sistema bloquea los programas descargados que no están firmados. Sale un
aviso diciendo que no se puede comprobar el desarrollador. Se quita con:

```sh
xattr -d com.apple.quarantine esfinge-darwin-arm64
```

En **Windows** basta con hacer doble clic o llamarlo desde PowerShell. Si Windows avisa de que es un
programa desconocido, «Más información» → «Ejecutar de todas formas».

Para tenerlo a mano en todo momento, cópialo a un sitio del `PATH` con el nombre `esfinge`:

```sh
sudo mv esfinge-darwin-arm64 /usr/local/bin/esfinge   # macOS y Linux
```

## Uso con menús

Abre el programa sin escribir nada más:

```sh
esfinge
```

Sale un menú: cifrar, descifrar, generar una contraseña, ayuda y salir.

**Se puede usar con el ratón o con el teclado, indistintamente.** Al pasar el puntero por encima, lo
que hay debajo se resalta; al hacer clic, se activa. Los botones —`[ Cifrar ]`, `[ Guardar en un
fichero ]`, `[ Volver ]`— hacen lo mismo que sus teclas, y clicar en un campo lo enfoca.

Con el teclado: flechas para moverse, `Intro` para elegir, `Tab` para cambiar de campo, `Esc` para
volver y `Q` para salir. En la pantalla de resultado, `C` copia y `G` guarda.

**Texto o fichero.** Dentro de cifrar y de descifrar hay dos botones, `[ Texto ]` y `[ Fichero ]`
(o `Ctrl+F`), que cambian de uno a otro. En modo fichero la ruta se puede dar **arrastrando el
fichero hasta la ventana del terminal**: Esfinge limpia las comillas y los espacios escapados que
cada terminal añade por su cuenta. El original nunca se toca, y el resultado se escribe al lado con
permisos `600` y sin pisar nada que ya exista.

**Copiar.** Al cifrar un texto, **el resultado se copia solo al portapapeles** y la pantalla lo dice
con un `✓`: lo siguiente que se hace con un texto cifrado es pegarlo en algún sitio, siempre. El
botón `[ Copiar ]` sigue ahí para repetirlo. Al descifrar no se copia nada por su cuenta: lo que sale
ahí es el secreto en claro, y dejarlo en el portapapeles sin que nadie lo pida es meterlo donde puede
leerlo cualquier cosa.

Por debajo usa el portapapeles del sistema y, además, manda la secuencia `OSC 52` al terminal, que es
lo único que funciona a través de SSH —el portapapeles del sistema copiaría en la máquina remota—. En
Linux hace falta `xclip`, `xsel` o `wl-copy`; si no hay ninguno, el botón lo dice y queda el de
guardar.

**No se pierde nada por error.** Si cierras o vuelves con algo en pantalla que no has guardado ni
copiado, Esfinge pregunta antes, y ofrece guardar y salir. Lo que ya está en un fichero no pregunta.

El fichero que genera `G` —o el botón— se escribe **en la carpeta de Descargas**, con permisos `600`
y un nombre con la fecha y la hora, para que un guardado no se coma el anterior. La pantalla te dice
la ruta completa. Descargas es el sitio que todo el mundo sabe abrir y no depende de desde dónde se
haya arrancado Esfinge; en Linux se respeta `XDG_DOWNLOAD_DIR`, y si no hay carpeta de descargas por
ninguna parte se usa la carpeta personal.

**El pie de la ventana también se pulsa.** Los recordatorios de abajo —`C Copiar`, `G Guardar`,
`Intro Volver`— se ven como botones, así que se comportan como botones.

**La ventana se agranda sola.** La pantalla más alta —cifrar, con sus tres campos— necesita 30
filas, y un terminal viene con 24. Al abrir la interfaz, Esfinge le pide al terminal que se agrande
hasta 30, y al salir lo devuelve al tamaño que tenía. Es una petición: la atienden iTerm2,
Terminal.app, gnome-terminal, konsole, kitty, alacritty y Windows Terminal, pero hay terminales que
la traen desactivada a propósito. Cuando no la atienden, el contenido se desplaza —la vista sigue
al campo que tiene el foco, la rueda del ratón manda, y abajo se indica cuánto queda fuera—.
`--sin-redimensionar` deja la ventana en paz.

Un efecto secundario del ratón: mientras Esfinge está abierto, el terminal le cede los clics, así que
para **seleccionar texto con el ratón** hay que mantener pulsada una tecla — `⌥ Option` en macOS,
`Shift` en Windows y en la mayoría de los Linux—. Para no depender de eso, el botón de guardar deja
el resultado en un fichero.

## Uso desde la terminal

### Cifrar un secreto corto

```sh
esfinge cifrar
```

Pregunta el secreto y la clave, y devuelve una línea así:

```
ESF1.RVNGMQEAAAEAAAAAAAMEW77N6LiWIRQM4fm0l8P6mt8PbFAz4GJC-C0icDTG7h7iJWpBo
```

Esa línea se puede pegar en un correo, en un chat, en un `.env` o dentro de una URL sin escapar
nada: no lleva `/`, ni `+`, ni `=`. **Hay que copiarla entera, con el `ESF1.` de delante.**

### Descifrar

```sh
esfinge descifrar
```

Pide el texto y la clave. No hay que decirle si le estás dando un texto o un fichero: lo distingue
solo.

### Cifrar un fichero de credenciales

```sh
esfinge cifrar    -i credenciales.env -o credenciales.env.esf
esfinge descifrar -i credenciales.env.esf -o credenciales.env
```

Aguanta ficheros de cualquier tamaño sin cargarlos enteros en memoria. El fichero cifrado se escribe
con permisos `600` —solo lo lee su dueño— y de forma atómica: si el proceso se corta a mitad, no
queda un fichero medio escrito con el nombre del bueno.

Sin `-o`, cifrar añade `.esf` al nombre y descifrar se lo quita. No sobrescribe nada sin que se lo
pidas con `--forzar`.

### En un script, sin que pregunte nada

```sh
export CLAVE='…'
echo -n 'el secreto' | esfinge cifrar --clave-env CLAVE > secreto.esf
esfinge descifrar --clave-env CLAVE -i secreto.esf
```

También `--clave-fichero ruta`. La clave **nunca** se pasa como argumento de la línea de comandos:
ahí quedaría en el historial del shell y la vería cualquiera con un `ps`.

Cuando la salida no es un terminal sale el dato pelado, sin colores ni marca, para que se pueda
encadenar. Los códigos de salida distinguen qué ha pasado:

| Código | Qué ha pasado |
|---|---|
| `0` | bien |
| `1` | error general |
| `2` | el comando está mal escrito |
| `3` | la clave no es correcta |
| `4` | el contenedor está dañado, cortado o no es de Esfinge |

### Generar contraseñas

```sh
esfinge generar                          # hexadecimal, 192 bits
esfinge generar --bytes 32 -n 5          # cinco de 256 bits
esfinge generar --alfabeto alnum         # letras y números
esfinge generar --alfabeto simbolos      # con símbolos, y avisa
```

Por defecto **hexadecimal**, y es a propósito. Una contraseña con `/` parte una cadena de conexión
—`postgres://usuario:pa/ss@host` deja de ser una URL— y el error que sale por el otro lado no
menciona la contraseña por ninguna parte, así que se pierden horas buscando donde no es. El alfabeto
`simbolos` está disponible, y avisa cada vez.

### Colores

El tema se elige solo según el fondo del terminal. Se puede forzar con `--tema claro` o
`--tema oscuro`, dejarlo fijo con la variable `ESFINGE_TEMA`, y apagar el color del todo con
`NO_COLOR`; sin color, los estados se distinguen por su símbolo (`✓`, `!`, `✕`).

### Si tarda cinco segundos en arrancar

Para saber si el fondo del terminal es claro u oscuro hay que preguntárselo con una secuencia de
escape y esperar la respuesta. Casi todos contestan al instante —Windows Terminal, Terminal.app,
iTerm2, gnome-terminal, konsole, alacritty, kitty, mintty—, pero alguno no contesta nunca, y
entonces la librería espera cinco segundos antes de rendirse. Se nota sobre todo en `esfinge
generar`, que debería ser instantáneo.

La consulta la lanza Bubble Tea al cargarse, antes de que Esfinge ejecute una sola línea propia, así
que no se puede desactivar desde dentro del programa. Si te pasa, cualquiera de estas dos lo quita:

```sh
export CI=1          # la librería deja de preguntar; el color se mantiene
export TERM=dumb     # también lo quita, pero pierdes el color
```

Con la salida redirigida a un fichero o a una tubería no ocurre: ahí no se pregunta nada.

## Cómo está hecho

- **Cifrado:** XChaCha20-Poly1305, con nonce de 24 bytes al azar y la cabecera autenticada.
- **Derivación de la clave:** Argon2id con 64 MiB, 3 pasadas y paralelismo 4. Los parámetros viajan
  dentro del contenedor, de modo que subir el coste más adelante no rompe lo ya cifrado.
- **Ficheros:** troceados en segmentos de 64 KiB, cada uno con su etiqueta, su número de orden y una
  marca en el último. Eso es lo que hace que un fichero cortado por la mitad se detecte en vez de
  descifrarse a medias y en silencio.
- **Formato:** `ESF1`, versionado. `esfinge descifrar` avisa si le llega un contenedor de una versión
  posterior en vez de fallar de cualquier manera.

## Desarrollo

Hace falta Go 1.24 o posterior. En esta máquina está en `~/.local/go`, que es lo que el `Makefile`
da por hecho; con Go en el `PATH`, `make GO=go`.

```sh
make comprobar    # go vet y toda la batería de pruebas
make contraste    # mide el contraste de los dos temas y lo lista
make esfinge      # binario para esta máquina
make instalar     # compila e instala en esta máquina Linux
make publicar     # los seis binarios en dist/, con SHA256SUMS
make macos        # el ZIP con instalador de doble clic
make linux        # el tar.gz con instalador para Linux
make ayuda        # los objetivos disponibles
```

`make instalar` deja el binario en `/usr/local/bin` si puede escribir ahí y, si no, en
`~/.local/bin`; avisa cuando esa carpeta no está en el `PATH` y cuando falta una herramienta de
portapapeles.

Las pruebas cubren la ida y vuelta del cifrado, la detección de cada byte alterado, el truncado, la
reordenación de segmentos, el recorrido completo de los menús y el contraste de las dos paletas.

### Sobre el diseño

La identidad visual es la de **ClickHouse**, tomada del catálogo de sistemas de diseño de la casa
(`~/sistemas-diseno-empresas/sistemas/clickhouse`): lienzo casi negro, tres superficies, filete de
1 px como única separación y un amarillo eléctrico como acento único. La marca de Webcafeína está en
el rótulo, la barra `▍` y el pie.

ClickHouse es un sistema **solo oscuro**, así que la variante clara está derivada. Su amarillo
`#faff69` sobre blanco da 1,07:1 y es ilegible como texto —el mismo problema que el lima `#B1F100`
de la marca—, de modo que en tema claro queda reservado a rellenos y el texto usa una versión
oscurecida hasta cumplir AA, con el mismo procedimiento que `readableAccent` del paquete
`design-tokens`. Por eso el foco de un campo y la barra de marca usan el acento y nunca el relleno.

`make contraste` mide las parejas que la interfaz usa de verdad, cada una con su mínimo y su
propósito, en los dos temas. Si una falla, falla `go test` y no hay binario.

---

▍ **webcafeína**
