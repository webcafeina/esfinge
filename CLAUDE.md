# Esfinge

Cifra y descifra contraseñas y ficheros con una clave. De **Webcafeína**.

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
make publicar     # la línea de comandos para los seis objetivos
make app          # la aplicación con ventana (necesita wails; ver abajo)
make ayuda        # todos los objetivos
```

**La aplicación con ventana no se puede compilar en esta máquina.** Falta `webkit2gtk` y
`pkg-config`, y no hay `sudo` sin contraseña. La compila **GitHub Actions** en los tres sistemas
(`.github/workflows/compilar.yml`, se dispara a mano o con una etiqueta `v*`). La línea de comandos
sí cruza de plataforma desde aquí, porque no usa cgo.

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

**Doble clic en un `.esf` en macOS.** La asociación está declarada en `wails.json` y el Finder la
respeta, pero el fichero llega por un evento de Apple que Wails v2 no expone. En Windows y Linux
llega como argumento y funciona. Ahí hay trabajo pendiente si se quiere cerrar del todo.

## Lo que nunca se ha probado

La aplicación **solo se ha ejecutado en la cabeza de CI, no en un escritorio de verdad**. Sin
verificar en un Mac: el arrastrar y soltar desde el Finder, el diálogo de guardar, el portapapeles
del sistema, el aviso de Gatekeeper —que con una `.app` sin firmar es más aparatoso que con un
binario de terminal— y si el doble clic en un `.esf` hace algo útil.
