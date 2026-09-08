# Deuda y cabos sueltos

Última actualización: **2026-09-08**

Lo que sabemos que está a medias, mal o sin comprobar. Los bloqueantes primero. Lo saldado se tacha
y se queda.

## Sin comprobar

Lo más caro de esta lista no es lo que está mal, es lo que no sabemos si lo está.

| Elemento | Severidad | Impacto | Estado |
|---|---|---|---|
| La aplicación no se ha ejecutado en Windows ni en Linux | Media | En macOS está probada de sobra —diálogos, arrastrar y soltar, doble clic en un `.esf`, menús, estructura y vidrio—, pero en los otros dos sistemas todo lo visual sigue siendo una suposición | Abierto · depende del humano. Bajó de Alta a Media cuando la 2.10.0 cerró el frente de macOS |
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
| Cifrar una tanda deriva la clave una vez por fichero | Baja | Es el precio de que cada contenedor lleve su sal, y no se va a quitar: compartir la derivación entre ficheros sería compartir la sal | Aceptado. Lo que sí se hizo es paralelizarlo, con tope de la mitad de los núcleos y máximo cuatro: veinte ficheros pasaron de 4,42 s a 1,29 s ([ADR 0018](adr/0018-tandas-en-paralelo.md)) |

## De producto

| Elemento | Severidad | Impacto | Estado |
|---|---|---|---|
| ~~La comprobación de actualizaciones solo ocurre al arrancar~~ | Media | `comprobarAlArrancar` la llamaba `Arrancar` **una sola vez** y no había ningún reloj: con Esfinge abierta no volvía a mirar nunca, mientras la portada y la [ADR 0014](adr/0014-comprobacion-de-actualizaciones.md) prometían «una vez al día». Lo dijo el cliente: nunca le había salido la banda de aviso | **Saldada en la 2.10.4**: `vigilar` deja un reloj que se asoma cada hora y aplica la misma puerta de las 24 h. Tres pruebas con `-race`: que avisa sin reiniciar, que el techo de una petición al día se mantiene, y que el reloj se para al cerrar la ventana (2026-09-08) |
| `VersionVista` se escribe y no se lee | Baja | Se guarda en las preferencias «para no repetir el mismo aviso», y no hay ni un sitio que la consulte: el descarte de la banda es solo de la interfaz y no sobrevive al reinicio. O se usa o sobra | Abierto |
| Gatekeeper avisa en macOS y Windows | Baja | El cliente ve un aviso de programa no identificado. Firmar cuesta 99 $/año y se decidió no hacerlo | Aceptado · [ADR 0012](adr/0012-sin-firmar.md). Bajó de Media a Baja al comprobarse que es **solo de la primera instalación**: al actualizarse desde dentro no aparece, porque la cuarentena la pone quien descarga y ahí descarga Go (2026-09-08) |
| ~~El icono del documento `.esf` era el de la aplicación~~ | Baja | Un fichero cifrado y el programa que lo abre se veían igual: `build/esf.png` era una copia byte a byte de `build/appicon.png`, y en Linux el `.deb` apuntaba al icono de la aplicación | **Saldada (2026-09-08)**: `build/documento.svg` dibuja una hoja con la esquina doblada y la marca sobre una placa. **Comprobado en el Mac**, y la marca se centró en la hoja tras verlo puesto (2.10.2). En Windows sale del mismo PNG sin tocar nada; en Linux hizo falta instalar `application-x-esfinge.png` en `mimetypes/`, comprobado con `dpkg -c` (2.10.3). Sin ver todavía en Windows ni en GNOME de verdad |
| ~~No hay barra de menús propia~~ | Baja | Sin atajos de teclado ni órdenes en la barra del sistema | **Saldada**: se construye entera y en español en los tres sistemas, con atajos ⌘1…⌘5. Los roles de Wails no servían porque traen los rótulos en inglés escritos a fuego, así que las acciones de edición las hace la ventana con una orden ([ADR 0015](adr/0015-menus-en-espanol.md)). Comprobado en el Mac, que era donde más riesgo había (2026-09-07) |

## Saldada

- ~~El arrastrar y soltar no hacía nada~~ → el modo «zona» de Wails exige declarar una propiedad CSS
  que no se estaba declarando (2026-09-07, 2.0.1).
- ~~Los símbolos de SF Symbols salían como cuadro vacío fuera de macOS~~ → `@supports` no sabe
  detectar fuentes; se volvió a caracteres normales (2026-09-07, 2.0.1).
- ~~El arranque tardaba cinco segundos en terminales que no contestan al OSC 11~~ → lo causaba el
  `init()` de Bubble Tea, que se fue con la interfaz de terminal (2026-09-07, 2.0.0).

| ~~Cambiar de sección borra lo escrito~~ | Media | Cada sección se desmontaba al salir y con ella se iba lo escrito | **Saldada en la 2.9.0**: las secciones se quedan montadas desde la primera visita y se esconden en vez de quitarse (2026-09-07) |

| ~~Medir el vidrio sin romper la aplicación~~ | Alta | El diagnóstico de la 2.9.1 dejaba la aplicación cerrándose sola al arrancar. Tres cosas de ese Objective-C pueden reventar y ninguna avisa al compilar: devolver el `UTF8String` de una cadena autoliberada, que `alphaComponent` lanza excepción sobre un color de patrón, y que `valueForKey:@"drawsBackground"` puede no existir para lectura. Revertido en la 2.9.2 | **Saldada en la 2.10.0, y no midiendo sino leyendo**: el código de Wails está en el caché de módulos de esta máquina, y ahí se ve que nunca le pone material al `NSVisualEffectView`. No hacía falta instrumentar la aplicación del cliente para averiguarlo (2026-09-07) |

| ~~Adivinar en macOS sale caro y no hay banco de pruebas~~ | Media | Tres versiones seguidas —2.5.1, 2.9.0 y 2.9.1— salieron con un cambio de macOS que no se podía probar aquí, y las tres fallaron: dos no hicieron nada visible y la tercera cerró la aplicación, dejando al cliente sin herramienta hasta la 2.9.2. El ciclo «publicar para ver qué pasa» lo sufre él, no yo | **Saldada (2026-09-08)**: `compilar.yml` acepta `inspector` en su disparador manual y compila con `wails build -devtools`, que deja el Web Inspector en el paquete. Cada hipótesis pasa a costar una línea en una consola en vez de una versión publicada. Marcada por tres sitios para que no se confunda con una compilación normal: nombre del artefacto, retención y sufijo en la versión |
