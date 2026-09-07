# Deuda y cabos sueltos

Última actualización: **2026-09-07**

Lo que sabemos que está a medias, mal o sin comprobar. Los bloqueantes primero. Lo saldado se tacha
y se queda.

## Sin comprobar

Lo más caro de esta lista no es lo que está mal, es lo que no sabemos si lo está.

| Elemento | Severidad | Impacto | Estado |
|---|---|---|---|
| La aplicación nunca se ha ejecutado en un escritorio | Alta | Todo lo visual y todos los diálogos son una suposición hasta que alguien la abra | Abierto · depende del humano |
| El `.deb` no se puede instalar aquí | Media | Se puede inspeccionar con `dpkg -c`, pero instalarlo exige permisos que esta máquina no da | Abierto |
| El instalador de Windows | Media | Se compila, pero nadie lo ha ejecutado | Abierto |
| El portapapeles en Windows y Linux | Baja | Va por la API del navegador dentro del webview; en macOS está comprobado | Abierto |

## Técnica

| Elemento | Severidad | Impacto | Estado |
|---|---|---|---|
| El `main.go` de la aplicación vive en la raíz, no en `cmd/` | Baja | Rompe la organización idiomática de Go. Lo impone Wails, que busca el paquete main junto a `wails.json` | Aceptado · [ADR 0006](adr/0006-de-terminal-a-ventana.md) |
| La interfaz construida se copia a `internal/interfaz/dist` | Baja | Un paso más en la compilación. `go:embed` no puede salir del directorio de su paquete | Aceptado |
| El servidor de desarrollo publica los métodos por reflexión | Baja | Si un método cambia de firma, el fallo sale en tiempo de ejecución y no al compilar | Aceptado · solo existe tras la etiqueta `dev` |
| No hay pruebas de la línea de comandos | Media | `internal/cli` no tiene tests: se comprueba a mano en cada cambio | Abierto |
| Cifrar una tanda deriva la clave una vez por fichero | Baja | Cincuenta ficheros son veinticinco segundos. Es el precio de que cada contenedor lleve su sal, y se puede paralelizar | Abierto → [siguiente.md](siguiente.md) |

## De producto

| Elemento | Severidad | Impacto | Estado |
|---|---|---|---|
| Gatekeeper avisa en macOS y Windows | Media | El cliente ve un aviso de programa no identificado. Firmar cuesta 99 $/año y se decidió no hacerlo | Aceptado · [ADR 0012](adr/0012-sin-firmar.md) |
| El icono del documento `.esf` no está comprobado | Baja | El Finder puede enseñar un icono genérico | Abierto |
| No hay barra de menús propia | Baja | Sin atajos de teclado ni órdenes en la barra del sistema | Abierto → [siguiente.md](siguiente.md) |

## Saldada

- ~~El arrastrar y soltar no hacía nada~~ → el modo «zona» de Wails exige declarar una propiedad CSS
  que no se estaba declarando (2026-09-07, 2.0.1).
- ~~Los símbolos de SF Symbols salían como cuadro vacío fuera de macOS~~ → `@supports` no sabe
  detectar fuentes; se volvió a caracteres normales (2026-09-07, 2.0.1).
- ~~El arranque tardaba cinco segundos en terminales que no contestan al OSC 11~~ → lo causaba el
  `init()` de Bubble Tea, que se fue con la interfaz de terminal (2026-09-07, 2.0.0).
