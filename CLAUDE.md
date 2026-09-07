# Esfinge

Herramienta de terminal de **Webcafeína** para cifrar y descifrar contraseñas y ficheros con una
clave. La usan dos perfiles: quien la encarga —desde la terminal, en tuberías y scripts— y un
cliente, que necesita menús, ratón y que la cosa se explique sola.

Un binario por plataforma, sin dependencias. Sin argumentos abre la interfaz de menús; con
argumentos es un comando pipeable.

## Cómo se compila

**Go no está en el `PATH` del sistema**: vive en `~/.local/go` porque se instaló sin `sudo`. El
`Makefile` ya lo da por hecho (`GO ?= $(HOME)/.local/go/bin/go`); en un shell suelto hace falta
`export PATH="$HOME/.local/go/bin:$PATH"`.

```sh
make comprobar    # go vet y toda la batería
make contraste    # mide las parejas de color de los dos temas
make esfinge      # binario de esta máquina
make instalar     # lo instala en esta máquina Linux
make paquetes     # el ZIP de macOS y el tar.gz de Linux, con instaladores
```

`make macos` y `make linux` por separado también valen: `publicar` ya no vacía `dist/`, solo borra
los binarios sueltos, para que uno no se lleve por delante el paquete del otro.

## Decisiones tomadas con el cliente

No se cambian sin preguntar.

- **Go**, binario único por plataforma. Nada de pedirle a nadie que instale un intérprete.
- **Cifrado suelto**, sin bóveda ni estado persistente. Texto y ficheros.
- **Interfaz híbrida**: menús sin argumentos, comando con ellos.
- **Identidad de ClickHouse completa, con su acento amarillo**, tomada de
  `~/sistemas-diseno-empresas/sistemas/clickhouse`. No se sustituye por el lima de Webcafeína: la
  marca está en el wordmark, la barra `▍` y el pie.
- **Español**, y **todas las frases empiezan en mayúscula**, aunque sean de una palabra. Va contra
  la costumbre de Go para los errores; manda lo que se ve en pantalla. Hay tests que lo vigilan en
  `internal/cripto/textos_test.go` e `internal/tui/textos_test.go`.
- **Ratón y botones** en igualdad con el teclado, no como añadido.
- **Guardar va a la carpeta de Descargas**, no al directorio de trabajo.
- **Al cifrar un texto se copia solo al portapapeles**; al descifrar no, porque ahí lo que sale es
  el secreto en claro.
- **Sin firmar ni notarizar para macOS**: los 99 $/año de Apple Developer no compensan para un
  cliente. El instalador quita la cuarentena con `xattr`.

## Tres trampas que ya costaron encontrarse

**El arranque de 5 segundos.** Bubble Tea llama a `lipgloss.HasDarkBackground()` en su `init()`,
antes de que corra una sola línea propia. En un terminal que no conteste al `OSC 11`, eso cuesta
cinco segundos, y `termenv.OSCTimeout` es una **constante**: no hay forma de tocarla desde fuera.
Escape documentado: `CI=1` o `TERM=dumb`. Bubble Tea marca ese `init` como provisional («will be
removed in v2»), así que al actualizar conviene volver a `internal/ui/estilos.go`.

**El ratón y el alto de la ventana.** Si la vista tiene más líneas que el terminal, el terminal la
desplaza y las coordenadas del ratón dejan de corresponderse con el mapa de zonas: los clics caen en
cualquier sitio. Por eso la pantalla va en tres bandas —cabecera fija, contenido desplazable, pie
fijo— y **nunca se emiten más líneas de las que caben**. Todo lo que tenga que estar siempre a mano
—conmutador Texto/Fichero, botones, mensajes de error— va en las bandas fijas, no en el contenido.
`TestLaVistaNuncaSePasaDelAlto` recorre 40 combinaciones de tamaño.

**El amarillo como texto.** `#faff69` sobre blanco da 1,07:1. En tema claro queda reservado a
rellenos y el texto usa una variante oscurecida hasta AA, con el mismo procedimiento que
`readableAccent` del paquete `design-tokens`. Todo lo que se dibuja en primer plano usa `Acento`,
nunca `Relleno`. `make contraste` mide las parejas reales de los dos temas y falla el build si una
no cumple.

## Lo que nunca se ha probado

Los binarios de **macOS y Windows se compilan pero no se ejecutan** desde aquí: no hay Wine ni Mac.
Sin verificar en un Mac real: el arrastrar-y-soltar desde el Finder, `pbcopy`, el diálogo de
Gatekeeper y si el terminal atiende la petición de agrandar la ventana.
