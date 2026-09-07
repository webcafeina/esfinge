# Esfinge

Cifra y descifra contraseñas y ficheros con una clave. De **Webcafeína**.

## Protocolo de sesión

La documentación viva está en [docs/](docs/). Se mantiene con disciplina, no con un control
automático: un validador de documentación se acaba sorteando, y lo que hay que sostener es el hábito.

**Al empezar**, leer [docs/estado.md](docs/estado.md). Dice dónde está el proyecto y cuál es la
siguiente acción concreta.

**Al tomar una decisión** que costaría volver a discutir, o que deja el código raro sin explicación,
o que descarta lo que parecía la opción evidente: una ficha en [docs/adr/](docs/adr/) y su línea en
[docs/decisiones.md](docs/decisiones.md). Las secciones son fijas —Contexto, Decisión, Alternativas
descartadas, Consecuencias, Verificación— y la última es la que más se agradece: dice qué se
comprobó de verdad y **qué no**.

**Al encontrar algo a medias o mal**, aunque no se arregle: a [docs/deuda.md](docs/deuda.md), con su
severidad y su impacto. Lo que no está escrito solo lo sabe quien lo dejó así.

**Al cerrar**, actualizar [docs/estado.md](docs/estado.md) y añadir la entrada en
[docs/sesiones.md](docs/sesiones.md), que tiene su plantilla al final.

Lo que se cierra no se borra: se tacha y se queda, con la fecha.

Dos caras sobre el mismo núcleo y el mismo formato de contenedor:

- **La aplicación** (`cmd/esfinge-gui`), con ventana propia, para el cliente.
- **La línea de comandos** (`cmd/esfinge`), para tuberías y scripts.

Lo cifrado por una lo abre la otra. El formato `ESF1` no ha cambiado desde la 1.x.

## Cómo se compila

**Go no está en el `PATH` del sistema**: vive en `~/.local/go` porque se instaló sin `sudo`. El
`Makefile` ya lo da por hecho (`GO ?= $(HOME)/.local/go/bin/go`); en un shell suelto hace falta
`export PATH="$HOME/.local/go/bin:$PATH"`.

```sh
make comprobar    # vet, tests de Go y tipos de la interfaz
make contraste    # mide las parejas de color de los dos temas
make tokens       # regenera frontend/src/tokens.css desde Go
make e2e          # mueve la interfaz de verdad contra el Go de verdad
make esfinge      # la línea de comandos, para esta máquina
make publicar     # comprueba y compila la línea de comandos para los seis objetivos
make app          # la aplicación con ventana (necesita wails; ver abajo)
make dmg          # la imagen de disco de macOS (solo en un Mac, con create-dmg)
make ventana-dmg  # dibuja cómo quedará esa ventana, sin necesidad de Mac
make ayuda        # todos los objetivos
```

**La aplicación con ventana no se puede compilar en esta máquina.** Falta `webkit2gtk` y
`pkg-config`, y no hay `sudo` sin contraseña. La compila **GitHub Actions** en los tres sistemas
(`.github/workflows/compilar.yml`, se dispara a mano). La línea de comandos sí cruza de plataforma
desde aquí, porque no usa cgo.

**Publicar es empujar una etiqueta `v*`.** Eso dispara `.github/workflows/publicar.yml`, que compila
en los tres sistemas, arma el DMG, el instalador de Windows y el `.deb`, y cuelga todo de la
publicación de GitHub. Los empaquetados viven en `empaquetado/`, uno por sistema.

## Cómo se prueba lo que no se puede ejecutar

Aquí no hay entorno gráfico, así que la interfaz se ejercita en un navegador contra el mismo Go que
llevará la ventana:

- `internal/app` es lo que la ventana puede pedir. Los diálogos del escritorio entran por la
  interfaz `Sistema`, que en producción implementa Wails (`escritorio.go`) y en desarrollo un
  servidor HTTP (`dev.go`, tras la etiqueta `dev` para que no acabe en el binario del cliente).
- `frontend/src/puente.ts` llama a Wails si está y, si no, por HTTP. La interfaz no distingue.
- `make e2e` levanta los dos servidores y recorre cifrar, descifrar, generar, tandas de ficheros e
  historial, en tema claro y oscuro.

Lo que **no** se puede comprobar aquí: la aplicación ensamblada. Eso se ve en el Mac.

## Decisiones tomadas con el cliente

No se cambian sin preguntar.

- **Wails** (Go + React + TypeScript), el stack de los demás proyectos de la casa.
- **Se conserva la línea de comandos** y se retiraron los menús de terminal de la 1.x.
- **Aspecto de aplicación del sistema**, no una identidad propia: tipografía y controles de macOS y
  Windows. La marca queda en el icono y en «Acerca de».
- **Y estructura del sistema**: barra lateral a la izquierda, título en la barra de herramientas y
  formularios en tarjetas, sin barra de título propia (ADR 0019). Las medidas salen de capturas de
  macOS 26 que están en `referencias/` —carpeta ignorada por git—, usando los semáforos como regla:
  miden 12 pt clavados y con eso se saca la escala de cualquier captura.
- **Cada sistema, con su forma y con su marco** (ADR 0020): la estructura de fondo es la misma en los
  tres, y lo que cambia son densidades, radios y quién dibuja la barra de título —solo macOS se queda
  sin ella—. Windows lleva el panel de Fluent con su barra de acento a la izquierda; Linux, la
  cabecera centrada de GNOME. Cuelga todo de `data-sistema`, que pone la interfaz preguntando a Go.
- **La barra de menús se construye entera** (`menu.go`), en español y en los tres sistemas: los roles
  de Wails traen los rótulos en inglés escritos a fuego en su Objective-C y no hay forma de
  traducirlos. Como sin roles no hay selectores nativos, las acciones de edición las hace la ventana
  con una orden (ADR 0015).
- **Español**, y **todas las frases empiezan en mayúscula**, aunque sean de una palabra. Va contra
  la costumbre de Go para los errores; manda lo que se ve en pantalla. Lo vigila
  `internal/cripto/textos_test.go`.
- **El historial guarda solo qué y cuándo**: nunca el contenido, la clave ni el texto cifrado. Vive
  en la carpeta de configuración del usuario, con permisos 600 y un botón de vaciar.
- **Al cifrar un texto se copia solo al portapapeles**; al descifrar no, porque ahí lo que sale es
  el secreto en claro.
- **Guardar usa el diálogo del sistema**, y los diálogos **recuerdan su carpeta**: una para abrir y
  otra para guardar, porque son gestos distintos. Ojo: Wails falla la llamada entera si el directorio
  por defecto ya no existe, así que se comprueba antes de proponerlo.
- **Sin firmar ni notarizar para macOS**: los 99 $/año de Apple no compensan para un cliente.
- **Comprueba actualizaciones sola**, una vez al día, y se descarga el instalador de su sistema
  comprobando el SHA256. Es la única conexión que hace el programa, se cuenta en Ajustes y se apaga
  ahí (ADR 0014).
- **Y se reemplaza a sí misma y se reinicia** en macOS y Windows, con un guion que espera a que el
  proceso muera. En Linux no: el `.deb` instala como root (ADR 0016). Firmar con Apple no tiene nada
  que ver con esto —evita el aviso de Gatekeeper, no habilita el reemplazo—, y darlo por hecho fue
  el error de la 0014.

## Trampas que ya costaron encontrarse

**Todo método exportado de `*App` queda expuesto a la interfaz.** Wails los enlaza por `Bind` y el
servidor de desarrollo los publica por reflexión, sin listas que mantener — que es cómodo hasta que
se exporta algo que no debería poder pedirse desde la ventana. Por eso `comprobarAlArrancar` va en
minúscula y `ApuntarAAPI` es función y no método: dejar que la interfaz apunte la actualización a
donde quiera sería abrir una puerta por comodidad.

**Ahora hay red en el binario del cliente.** Hasta la 2.0.3 no la había: el único `net/http` estaba
tras la etiqueta `dev`. Es una petición GET al día a `api.github.com`, y está documentada en
`docs/seguridad.md` porque la portada prometía lo contrario. Cualquier conexión nueva pasa por ahí
antes que por el código.

**Wails deja el vidrio de macOS a medias.** Crea el `NSVisualEffectView` con mezcla «BehindWindow»
pero **no pone la ventana como no opaca**, y una `NSWindow` opaca compone como opaca aunque su color
tenga alfa cero: el material no tiene nada que mezclar y sale un **gris plano**. Desde macOS 12 se
suma el `underPageBackgroundColor` del `WKWebView`, que tapa igual. Las dos cosas las remata
`vidrio_darwin.go` con cgo (ADR 0017), y **ese fichero no se puede compilar en esta máquina**: lo
comprueba el trabajo de macOS de la publicación.

**El vidrio del sistema obliga a que el CSS pinte el fondo.** Al pedir una ventana translúcida
—macOS siempre, Windows 11 con Mica— el webview deja pasar la luz, así que el color lo pone la
página. Donde no hay vidrio, un `body` transparente no enseña el escritorio: enseña un agujero. De
ahí que la interfaz pregunte con `Vidrio()` y ponga `data-vidrio="si"`, y que **todo el CSS del
efecto cuelgue de ese atributo** (ADR 0017). Hay una prueba de interfaz que lo vigila.

Y el corolario, que costó una versión: **lo que no declara fondo se vuelve transparente y enseña el
material**. Al reestructurar, la barra de herramientas se quedó sin declararlo y aparecía una banda de
otro color encima del contenido. Por eso el fondo se pone en la columna entera (`.zona`) y no en cada
pieza. El material va en la barra lateral, que es donde lo pone macOS.

**Las tandas se cifran en paralelo, con tope.** La mitad de los núcleos, máximo cuatro: cada
derivación ya usa cuatro hilos por dentro y 64 MiB mientras dura, así que pasarse es pisarse (ADR
0018). Los resultados conservan el orden de entrada y el progreso se cuenta al **terminar** cada
fichero. Desde aquí, `go test -race` deja de ser una cortesía.

**`TitleBarHidden`, no `TitleBarHiddenInset`.** El preajuste «Inset» activa `UseToolbar`, y entonces
macOS dibuja su propia banda de barra de herramientas justo donde va nuestro título: se ve un fondo
que no cuadra con el resto de la ventana. La barra de herramientas la dibujamos nosotros.

**En la barra lateral, la fila activa no se resalta al pasar por encima**, y hace falta escribirlo:
`.lateral button` **empata en peso** con la regla general de `button:hover`, que vive más abajo en el
fichero y por eso ganaba. Los selectores de la barra lateral llevan `nav` para desempatar, y hay una
prueba que lo vigila porque esos empates vuelven solos.

**Sin barra de título, arrastrar la ventana deja de ser gratis.** Con `TitleBarHiddenInset` el
contenido llega hasta arriba y ya no hay nada que agarrar: hay que declarar las zonas con
`--wails-draggable: drag` —la barra lateral y la de herramientas— y desmarcar los botones con
`no-drag`. Si se olvida, la ventana se queda clavada en la pantalla.

**En las pruebas, «Cifrar» es dos cosas.** Nombra la sección de la barra lateral y el botón que
cifra, así que los selectores se acotan: `seccion()` mira dentro de `.lateral` y `accion()` dentro de
`.contenido`. Sin acotar, Playwright encuentra dos y falla por modo estricto.

**El color se genera, no se escribe.** `internal/tema` es la fuente de verdad y produce
`frontend/src/tokens.css` con `make tokens`. Editar el CSS a mano no sirve: hay un test que compara
el fichero con lo que dice Go y falla. Y `make contraste` mide las parejas reales de los dos temas.
El azul de botón del sistema no cumple AA con texto blanco encima —3,6:1—, así que `RellenoLegible`
lo oscurece hasta que se lee.

**`go:embed` no puede salir del directorio de su paquete.** Por eso la interfaz construida se copia
a `internal/interfaz/dist`, y no se embebe directamente desde `frontend/dist`.

**Un `error` nulo devuelto por reflexión no supera una aserción de tipo.** En `dev.go` hay que mirar
el tipo declarado (`tipo.Out(i)`), no el valor: preguntándole al valor se acaba tomando el error por
resultado y devolviendo `null` cuando todo ha ido bien.

**Doble clic en un `.esf` en macOS: hay dos momentos, y confundirlos es lo que abría la ventana
vacía.** El fichero no llega como argumento sino por un evento de Apple, que Wails **sí** entrega en
`options.Mac.OnFileOpen` (en Windows y Linux llega por `os.Args`). Puede llegar **antes** de que la
interfaz esté escuchando —abrir la aplicación con doble clic— o **después** —doble clic con Esfinge
ya abierta—. El primero hay que guardarlo, porque no hay a quién avisar; el segundo hay que
avisarlo, porque nadie va a volver a preguntar. `AlAbrirCon` distingue los dos, y la interfaz **se
suscribe antes de preguntar**: al revés queda un hueco por el que el fichero se pierde. Sin
comprobar todavía: si el Finder enseña el icono del documento.

**Un `.esf` no dice qué lleva dentro por la extensión.** Puede ser un fichero cifrado o la línea
`ESF1.…` que sale de cifrar un texto y que alguien guardó. Los dos empiezan por la misma magia; lo
que los separa es el byte siguiente —la versión en el binario, el punto en el de texto—, y eso es
`cripto.FormaDe`. `AperturaDe` lo mira para abrir la pantalla que toca: en la de texto, con la línea
puesta; en la de ficheros, con las rutas. Abrir un texto en la pantalla de ficheros era un lío,
porque ahí lo que se quiere ver es el secreto, no otro fichero al lado.

**Un icono de disco no se dibuja de frente.** Costó tres intentos: apaisado, luego vertical pero
plano, y solo al tercero salió. Los discos de macOS van **en perspectiva**, mirados un poco desde
arriba, con la carcasa estrechándose hacia el fondo, una **placa en color con el logo grande** en la
cara de arriba y una **banda con su piloto** debajo. Y son casi cuadrados, no alargados. Las
capturas de referencia están en `referencias/`, que no se sube.

**El icono del volumen no es el de la aplicación.** Con el mismo icono, el disco montado y lo que hay
dentro se ven igual y en la barra lateral del Finder no se distinguen. El volumen lleva un disco con
la marca encima (`build/darwin/disco.svg`), y su `.icns` lo arma `armar-dmg.sh` con `iconutil`, que
**solo existe en macOS**: aquí solo se pueden rasterizar los PNG del `.iconset` con `make icono`.

**El fondo del DMG y los nombres de los iconos.** El Finder centra cada icono en la posición que le
da `create-dmg` y **escribe su nombre debajo**: con iconos de 96 px, el pie del nombre queda unos 64
px por debajo del centro. Todo lo que el fondo dibuje ahí queda tapado, y eso no se ve hasta montar
la imagen en un Mac. `make ventana-dmg` la dibuja antes, leyendo las posiciones de
`empaquetado/macos/armar-dmg.sh` para que fondo y guion no se separen.

**Cuatro cosas que solo se descubren compilando de verdad**, todas encontradas en GitHub Actions:

- El `main.go` de la aplicación **tiene que estar en la raíz**, junto a `wails.json`. Wails genera
  los enlaces buscando el paquete main ahí; en `cmd/` falla con «no Go files».
- `fileAssociations` va **dentro de `info`** en `wails.json`. Fuera no da error: simplemente el
  paquete sale sin asociaciones, y eso solo se ve mirando el `Info.plist` del artefacto.
- En Linux, Wails busca `webkit2gtk-4.0` y Ubuntu reciente solo trae la 4.1: hace falta compilar con
  `-tags webkit2_41`.
- Los artefactos de GitHub **no conservan el bit de ejecución**. Una `.app` descargada de ahí no
  arranca hasta que se le devuelve con `chmod +x Contents/MacOS/Esfinge`.

## Lo que nunca se ha probado

Comprobado ya en un Mac de verdad: el arrastrar y soltar desde el Finder, el diálogo de guardar, el
doble clic en un `.esf` y la imagen de disco, que se monta y se arrastra sin más.

Sin verificar todavía: si el Finder enseña el icono propio en los ficheros `.esf`, y qué tan
aparatoso resulta el aviso de Gatekeeper con una `.app` sin firmar la primera vez.
