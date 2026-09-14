# ADR 0034 — El canal con el navegador, en Windows

**Fecha:** 2026-09-14 · **Estado:** aceptada · **Continúa la [0027](0027-el-canal-con-el-navegador.md)** ·
**Revisar cuando** alguien lo pruebe en un Windows de verdad

## Contexto

La extensión funcionaba en macOS y Linux y **no podía funcionar en Windows** por dos huecos apuntados
en `docs/deuda.md` desde la fase 2:

- **Esfinge no decía a los navegadores dónde está el puente.** En macOS y Linux eso es un fichero JSON
  en una carpeta de cada navegador. En Windows no hay carpeta: `dondeMiraCadaNavegador` devolvía nulo.
- **El instalador no llevaba `esfinge-puente.exe`**, que es lo que el navegador lanza.

El socket ya funcionaba allí desde la 2.17 (`escuchar_windows.go`) y `rutaDelPuente` ya buscaba el `.exe`
junto a Esfinge.

Antes de publicar la extensión en las tiendas, el cliente eligió **arreglar Windows primero**, sabiendo
que nadie tiene un Windows a mano para probarlo.

## Decisión

### El manifiesto, apuntado desde el registro

En Windows el navegador mira **una clave de registro cuyo valor por defecto es la ruta absoluta del
manifiesto** (documentación de Chrome, Edge y MDN). Así que Esfinge escribe los manifiestos en su propia
carpeta —`%APPDATA%\Esfinge\NativeMessagingHosts\`, **uno por familia**, porque Firefox lleva
`allowed_extensions` y Chrome `allowed_origins`— y **una clave por navegador instalado**, solo en `HKCU`:

| Navegador | Clave bajo `HKCU\Software\` | Señal de que está |
|---|---|---|
| Firefox | `Mozilla\NativeMessagingHosts` | `%APPDATA%\Mozilla\Firefox` |
| Chrome | `Google\Chrome\NativeMessagingHosts` | `%LOCALAPPDATA%\Google\Chrome\User Data` |
| Chromium | `Chromium\NativeMessagingHosts` | `%LOCALAPPDATA%\Chromium\User Data` |
| Edge | `Microsoft\Edge\NativeMessagingHosts` | `%LOCALAPPDATA%\Microsoft\Edge\User Data` |
| Brave | la suya **y** la de Chrome | `%LOCALAPPDATA%\BraveSoftware\Brave-Browser\User Data` |
| Vivaldi | la de Chrome | `%LOCALAPPDATA%\Vivaldi\User Data` |
| Opera | la de Chrome | `%APPDATA%\Opera Software\Opera Stable` |

**Brave, Vivaldi y Opera no documentan qué clave leen**, y lo que se encuentra se contradice. Brave va
en las dos claves; Vivaldi y Opera en la de Chrome, que es lo que hace KeePassXC hoy.

Apagar el canal borra las claves y los ficheros, igual que borra los ficheros en los otros dos sistemas.

### Probable desde Linux

- **El registro va detrás de una interfaz** (`registro`): el de verdad en `registro_windows.go`, con
  `golang.org/x/sys/windows/registry`, y nulo en los demás sistemas.
- **Las rutas se calculan para un sistema dado, no para el que corre** (`sistema` es una variable), como
  `filtrosPara`. La tabla de Windows se prueba entera desde Linux con un registro de mentira.
- **Y las pruebas de siempre ya no se saltan en Windows**: fuerzan un sistema de carpetas y corren igual.

### El instalador lleva el puente

`build/windows/installer/project.nsi` es la plantilla de Wails v2.15.0 con **`File "esfinge-puente.exe"`**
tras los ficheros de la aplicación. Wails usa la del proyecto tal cual si existe. El puente lo compila
`publicar.yml` para el subsistema gráfico (`-H windowsgui`, sin consola que parpadee) en esa carpeta, y
después **se comprueba que el instalador lo lleva** listándolo con `7z`.

## Alternativas descartadas

**`HKLM`, desde el instalador.** Valdría para todos los usuarios del equipo, pero lo escribiría un
instalador con permisos de administrador y no se podría apagar desde Ajustes sin pedirlos otra vez. Y
rompería la simetría con macOS y Linux, donde todo va en la carpeta de la persona.

**Solo la clave de Chrome para todos los de su familia.** Edge la lee como respaldo, y Vivaldi y Opera
parece que también. Pero Edge prefiere la suya y Brave puede no leer la de Chrome, así que se escribe
cada una.

**Un manifiesto junto al ejecutable, en `Program Files`.** No hace falta que esté en ningún sitio
concreto, pero ahí no puede escribir una aplicación sin permisos de administrador.

## Consecuencias

- **Esfinge escribe en el registro de la persona** al encender el canal, y lo borra al apagarlo. Si se
  desinstala con el canal encendido, las claves quedan apuntando a un fichero que ya no existe. Es
  inofensivo —el navegador no encuentra el puente—, pero queda.
- **La clave de Chrome la comparten Chrome, Brave, Vivaldi y Opera**: apagar el canal la quita para
  todos, que es lo que se quiere.
- **`golang.org/x/sys` pasa a ser dependencia directa.** Ya estaba, indirecta, por Wails.

## Verificación

**Comprobado aquí (Linux):**

- La tabla entera con un registro de mentira: Firefox y Chrome con su clave y su fichero; los siete
  navegadores con las cinco claves esperadas; Brave en las dos; sin variables de entorno, las carpetas
  del perfil; y **sin registro no se da por avisado a nadie**.
- `GOOS=windows go vet ./...`, el binario de pruebas y el puente para Windows, compilados.

**Comprobado en la máquina Windows de GitHub, en cada publicación:**

- Las pruebas de manifiestos, y **una que escribe, lee y borra una clave en el registro de verdad**.
- Que el instalador lleva `esfinge-puente.exe`.

**Sin comprobar, y no lo puede decir ninguna máquina de GitHub:**

- **Que Chrome, Edge o Firefox lancen el puente de verdad** y la extensión rellene en Windows.
- Qué clave leen de verdad Brave, Vivaldi y Opera.
- Que el puente compilado con `-H windowsgui` conserve la entrada y la salida estándar cuando lo lanza
  cada navegador. Es lo que dice la documentación del subsistema y lo que hace KeePassXC, pero no se ha
  visto.
