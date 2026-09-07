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
- **Español**, y **todas las frases empiezan en mayúscula**, aunque sean de una palabra. Va contra
  la costumbre de Go para los errores; manda lo que se ve en pantalla. Lo vigila
  `internal/cripto/textos_test.go`.
- **El historial guarda solo qué y cuándo**: nunca el contenido, la clave ni el texto cifrado. Vive
  en la carpeta de configuración del usuario, con permisos 600 y un botón de vaciar.
- **Al cifrar un texto se copia solo al portapapeles**; al descifrar no, porque ahí lo que sale es
  el secreto en claro.
- **Guardar usa el diálogo del sistema.**
- **Sin firmar ni notarizar para macOS**: los 99 $/año de Apple no compensan para un cliente.

## Trampas que ya costaron encontrarse

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

**Doble clic en un `.esf` en macOS.** No llega como argumento, sino por un evento de Apple. Wails v2
**sí** lo entrega, en `options.Mac.OnFileOpen`, que es lo que usa `main.go`; en Windows y Linux el
fichero llega por `os.Args` y no hace falta. Lo que queda sin comprobar es si el Finder enseña el
icono del documento.

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
