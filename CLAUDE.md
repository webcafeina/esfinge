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
make servidor     # el servidor de cuentas en local (http://localhost:8790), con buzón de pruebas
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

Y la extensión tiene lo suyo, en `navegador/`:

- `pnpm run comprobar` mira los tipos, comprueba que **el manifiesto declara lo que el código usa**
  —una palabra que faltó ahí costó cinco versiones publicadas— y corre las pruebas de Playwright.
- Ésas ejercitan **qué campo se rellena**, en un Chromium de verdad y contra el fuente compilado en
  memoria. No hay servidor que levantar: son páginas escritas a mano, y **media tabla son casos donde
  lo correcto es no rellenar nada**. Contra un DOM simulado no valdrían: `getComputedStyle`,
  `getBoundingClientRect` y `compareDocumentPosition` devolverían lo que se les hubiera enseñado.

Lo que **no** se puede comprobar aquí: la aplicación ensamblada. Eso se ve en el Mac.

## Decisiones tomadas con el cliente

No se cambian sin preguntar.

- **Wails** (Go + React + TypeScript), el stack de los demás proyectos de la casa.
- **Se conserva la línea de comandos** y se retiraron los menús de terminal de la 1.x.
- **Aspecto de aplicación del sistema**: tipografía, controles y formas de macOS y de Windows, sin
  identidad ajena (ADR 0007). La tipografía no se toca nunca.
- **Pero el acento sí es nuestro, y la marca está dentro de la ventana** (ADR 0021): el oro del
  tocado en vez del azul del sistema, el lockup de Esfinge arriba en la barra lateral, la firma
  `▍ webcafeína` abajo y la esfinge grande y tenue en el historial vacío. **Nunca en las pantallas de
  trabajo**: ahí se trabaja.
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
  en la carpeta de configuración del usuario, con permisos 600 y un botón de vaciar. **La bóveda no
  escribe en él**, y es una regla absoluta: `credenciales-dashlane.csv` ahí sería una señal de
  tráfico apuntando a lo que alguien acaba de exportar en claro.
- **Borrar una entrada no es para siempre** (ADR 0026): va a una papelera que **guarda el contenido**
  y se vacía a mano o sola a los treinta días. Se eligió con el cliente frente a la alternativa
  —«borrar es borrar»— por una razón que conviene no volver a discutir: en un gestor de contraseñas,
  perder una por un clic es peor que conservar treinta días una que se quiso tirar. El coste está en
  `docs/seguridad.md`, al lado del del historial de contraseñas anteriores, que es el mismo trato.
- **Y la bóveda calcula los códigos de un solo uso** (ADR 0025), lo que la convierte también en el
  autenticador. Con la consecuencia que hay que decir en voz alta y está en `docs/seguridad.md`:
  **una bóveda abierta entrega la contraseña y el segundo factor a la vez**. Se hace igual porque la
  alternativa realista no era tenerlos separados, era tenerlos juntos en Dashlane.
- **Y desde la fase 2 rellena los formularios del navegador** (ADR 0027 y 0028). Con **una** cuenta
  guardada del sitio se rellena sola, que es lo que se decidió —«solo, como Dashlane»—; con varias se
  elige en el panel de la extensión, **salvo que se sepa quién entra** —el usuario escrito en el
  formulario, el escondido que declara el sitio o el de la página anterior— y coincida con una sola:
  entonces ésa, que lo pidió el cliente con la 2.21.1. Y **en la página se dibuja lo mínimo** (ADR 0028, matizada por
  la 0031): un **filete** de oro con halo en el campo rellenado —sin anillo de piedra, que se leía como un
  borde negro—, que se va al escribir, y un **aviso
  de tres segundos**, «Rellenado por Esfinge», que no se puede pulsar y va en una sombra cerrada.
  Nada de desplegables, iconos fijos dentro del campo ni marcos: dibujar en la página de otro es la
  parte cara y arriesgada, y cada cosa más se decide **con el uso**, no con la intuición. Dos
  cosas que cambian y hay que decir en voz alta: **por el canal ya sale una contraseña de verdad** y
  **hay código nuestro en cada página `https` que se abra**.
  Y desde la 2.19.0 **rellena también el código de un solo uso** (ADR 0030), con el mismo freno que
  la contraseña y **sin fiarse nunca de la palabra «code»** —que es el código postal, el promocional y
  el CVC—. Con eso, **por el canal salen y en la página quedan escritos la contraseña y el segundo
  factor**, que es lo mismo que la ADR 0025 dijo de la bóveda abierta, ahora dentro del navegador.
- **Y desde la entrega 3 ofrece guardar y actualizar lo que se envía** (ADR 0032), con una **tarjeta en
  la página que se pulsa**: lo primero de Esfinge que se pulsa en la web de otro, y **la primera vez que
  el navegador escribe en la bóveda**. Se aprueba pulsando en el navegador, sin confirmar en la ventana;
  «Nunca en este sitio» va cifrado dentro de la bóveda y se deshace en Ajustes. Lo decidió el cliente con
  esas palabras y no se cambia sin preguntar. **Comprobado en su Mac con la 2.21.2.**
- **La extensión no escribe nada en la consola por defecto** (2026-09-14): ni trazas de por qué no
  ofrece guardar ni avisos de depuración. Se propuso para diagnosticar sitios que fallan y el cliente
  dijo que no, «de momento». Cuando un sitio no funciona, se le pide un fragmento para pegar en la
  consola —los campos, sin valores de contraseña—, como se hizo con Cloudflare, Google y Brevo.
- **Hay una bóveda de contraseñas**, local y cifrada, con clave de recuperación (ADR 0023). Es la
  fase 1 de sustituir a Dashlane, y **cambia lo que el producto es**: hasta ahora un fallo perdía un
  fichero; ahora puede perder todas las contraseñas de la empresa. Eso sube el listón de las pruebas
  y de la prisa antes de cada publicación. Lo que **no** hace: navegador, móvil, cuentas, servidor y
  compartir en equipo.
- **Al cifrar un texto se copia solo al portapapeles**; al descifrar no, porque ahí lo que sale es
  el secreto en claro.
- **Guardar usa el diálogo del sistema**, y los diálogos **recuerdan su carpeta**: una para abrir y
  otra para guardar, porque son gestos distintos. Ojo: Wails falla la llamada entera si el directorio
  por defecto ya no existe, así que se comprueba antes de proponerlo.
- **Sin firmar ni notarizar para macOS**: los 99 $/año de Apple no compensan para un cliente.
- **Comprueba actualizaciones sola**, una vez al día **y también con la ventana abierta**, y se
  descarga el instalador de su sistema comprobando el SHA256. Es la única conexión que hace el
  programa, se cuenta en Ajustes y se apaga ahí (ADR 0014).
- **Y se reemplaza a sí misma y se reinicia** en macOS y Windows, con un guion que espera a que el
  proceso muera. En Linux no: el `.deb` instala como root (ADR 0016). Firmar con Apple no tiene nada
  que ver con esto —evita el aviso de Gatekeeper, no habilita el reemplazo—, y darlo por hecho fue
  el error de la 0014.

- **Y va a haber cuentas** (ADR 0035, plan en `docs/cuentas.md`): la bóveda en todos los equipos con un
  servidor nuestro en **Cloudflare UE que no puede leer nada**, compartir **copias sin permisos** con otras
  cuentas, código por correo en cada equipo nuevo, bienvenida «En este ordenador / Con cuenta» reversible,
  **maestra «fuerte» obligatoria con cuenta**, y **la extensión como cliente completo de la cuenta, siempre
  por ella** —hecha en la 2.25.0 (ADR 0040): con cuenta, rellena, da códigos y guarda **sin la
  aplicación**, y copiar desde su panel no se borra solo del portapapeles—. La bienvenida sale **solo a quien estrena Esfinge sin nada** —quien ya tenía bóveda
  trabaja en local sin que se le pregunte—, y **la bóveda que hubiera en un equipo al entrar en una cuenta
  se aparta y no se borra nunca** (ADR 0039). **El registro libre, solo tras una auditoría externa**; hasta entonces, por invitación. Rompe
  dos promesas públicas —«sin servidores» y «Webcafeína no recibe nada»— que hay que cambiar antes de que
  nadie tenga cuenta.

## Trampas que ya costaron encontrarse

**Todo método exportado de `*App` queda expuesto a la interfaz.** Wails los enlaza por `Bind` y el
servidor de desarrollo los publica por reflexión, sin listas que mantener — que es cómodo hasta que
se exporta algo que no debería poder pedirse desde la ventana. Por eso `comprobarAlArrancar` va en
minúscula y `ApuntarAAPI` es función y no método: dejar que la interfaz apunte la actualización a
donde quiera sería abrir una puerta por comodidad. Desde la bóveda **hay una lista blanca que lo
vigila** (`TestLoQueCruzaElPuenteEstaEnLaLista`): con contraseñas dentro, un método de más puede ser
la clave maestra saliendo por ahí, y acordarse dejó de ser defensa suficiente.

**Un solo flujo de eventos para todos los oyentes, y esto no es una optimización.** El navegador solo
abre **seis conexiones** contra el mismo origen y un flujo de eventos no termina nunca: con un
`EventSource` por suscripción, a partir del sexto oyente **toda llamada al puente se queda esperando
para siempre**, sin error, sin petición en la red y sin nada que mirar. Con cinco la aplicación
funcionaba; la bóveda trajo el sexto y el síntoma fue un botón de copiar que no hacía nada. Solo pasa
en el navegador: en la ventana los reparte Wails por dentro.

**En las preferencias, el cero es «no lo he dicho».** `GuardarPreferencias` recibe el objeto entero,
así que un guardado a medias —mandar solo `{"buscarActualizaciones":true}`, que es lo que hace una
prueba— llega con todos los números a cero al deserializar. Si el cero significara «nunca», ese
descuido **apagaría el bloqueo de la bóveda y el borrado del portapapeles en silencio**. Por eso
«nunca» viaja como `-1` (`app.Nunca`) y el cero conserva lo que hubiera. Vale para cualquier campo
numérico que se añada.

**Y desde la 2.14.0 son dos salidas a la red, no una** (ADR 0024): la de versiones y **el icono de
cada sitio de la bóveda, pedido al propio sitio**. Tres cosas que hay que tener presentes al tocar
eso:

- **El destino lo elige el mundo exterior**, no nosotros: sale de un CSV que alguien importó. Por eso
  `internal/iconos` filtra las direcciones privadas, limita los saltos de redirección y prohíbe bajar
  a `http://`, que si no el filtro se esquiva con un `Location:`.
- **Y ese filtro va en `Control`, no en `DialContext`.** Es la diferencia entre filtrar y no hacer
  nada: `DialContext` recibe **el nombre sin resolver**, así que `ParseIP("github.com")` da nulo y la
  regla «si no se sabe qué es, no se va» rechaza todos los sitios del mundo. La 2.14.0 salió así y no
  descargó ni un icono. `Control` corre después de resolver y recibe la IP de verdad, que es lo que
  hay que mirar porque cualquier dominio público resuelve a donde quiera.
- **Ir directo no oculta la lista de sitios, la reparte**: el nombre viaja en claro en el DNS y en el
  saludo TLS. Lo que se gana es no meter un tercero de confianza. Está dicho así en la ventana, en
  `docs/seguridad.md` y en la ADR, y no se debe escribir de otra forma.
- **El goteo guarda y avisa icono a icono, no al terminar la tanda.** Guardando al final, el primero
  no aparecía hasta pasados minutos y quien cerraba antes no se llevaba ninguno. Y hay una prueba que
  recorre **la tubería entera** —de la bóveda al almacén pasando por la red—, que es la que faltaba:
  las piezas estaban probadas una a una y por eso sobrevivieron dos fallos seguidos que el cliente vio
  a la primera.
- **El goteo no llama a `Actividad()`.** Si lo hiciera, la bóveda no se cerraría nunca mientras baja
  iconos y el bloqueo por inactividad dejaría de significar lo que dice.

**Y eso ya es una regla y no un detalle de un caso: lo que se repite solo no cuenta como actividad.**
Apareció con el goteo de iconos y ha vuelto con el código de un solo uso, que la ventana vuelve a
pedir cada treinta segundos mientras haya una entrada abierta. Con una sola de esas dos cosas tocando
el reloj, una bóveda abierta encima de la mesa **no se cierra nunca**. La prueba que lo vigila pide el
código treinta veces con el reloj corriendo y comprueba que se cierra igual.

**Cualquier cadena de letras es base32 válida, así que una semilla mal copiada no se detecta.** Lo que
sí se puede detectar —y hay que hacerlo a mano— es que le falte o le sobre un carácter: **el
descifrador de Go no comprueba el largo cuando no hay relleno**, y un grupo final de un solo carácter
no le parece un error, así que devuelve los bytes anteriores como si nada y la semilla entra entera.
Los restos posibles de un grupo de ocho son 0, 2, 4, 5 y 7. Con el `0`, el `1`, el `8` y el `9` —que
no están en ese alfabeto— pasa lo mismo pero al revés: ésos sí los caza el descifrador.

**Y lo que está en la papelera no cuenta para el índice de duplicados del importador.** Desde que la
papelera guarda la entrada entera (ADR 0026), sin esa línea el importador reconoce lo borrado y
volver a pasar el CSV lo da por repetido: se ve como «la borré, la reimporté y no ha vuelto», con la
única copia escondida en la papelera y a punto de caducar. Vale para cualquier índice nuevo que se
construya recorriendo `cont.Entradas`: **casi siempre hay que saltarse la papelera**, y los sitios
donde ya se hace son `Buscar`, `Cuantas`, `Exportar` y ése.

**Los campos sensibles de una entrada estaban escritos en dos sitios, y solo uno estaba completo.**
Lo que viaja a la ventana en la lista y lo que se limpia al mandar una entrada a la papelera son la
misma lista y se mantenían por separado: la de borrar quitaba la contraseña, el TOTP y el historial
—el secreto de **una credencial**— y dejaba enteros el texto de una nota segura, el número de una
tarjeta y el de un documento, que son el secreto entero de las otras tres clases. Se quedaban dentro
del fichero para siempre. Ahora hay una sola función (`vaciarLoSensible`), y la prueba borra **una
entrada de cada clase** y vuelve a abrir el fichero para mirar lo guardado, no lo que quedó en
memoria.

**Y `ESFINGE_SIN_RED` es ahora un freno de verdad** (`internal/red`). Antes lo miraba solo la línea de
comandos y la ventana no lo consultaba nunca, así que quien lo ponía apagaba media red.

**Ahora hay red en el binario del cliente.** Hasta la 2.0.3 no la había: el único `net/http` estaba
tras la etiqueta `dev`. Es una petición GET al día a `api.github.com`, y está documentada en
`docs/seguridad.md` porque la portada prometía lo contrario. Cualquier conexión nueva pasa por ahí
antes que por el código.

**«Comprobar al arrancar» no es «comprobar una vez al día», y así estuvo hasta la 2.10.4.** La
comprobación la disparaba solo `Arrancar`, sin ningún reloj: quien dejaba la ventana abierta no se
enteraba nunca de una versión nueva, mientras la portada y la ADR 0014 prometían lo contrario. Lo
descubrió el cliente diciendo que no le salía la banda de aviso, no una prueba. Ahora `vigilar` deja
un reloj que se asoma cada hora, y **la puerta de las 24 horas sigue siendo quien decide**: asomarse
a menudo no es preguntar a menudo.

Y el detalle que costó una prueba en rojo: **hay que reservar el turno, no solo preguntar por él**
(`ReservarComprobacion`). Preguntando con `TocaMirar` y anotando al volver de la red, entre las dos
cosas cabe toda la ida y vuelta a GitHub, y por ese hueco pasan varias comprobaciones a la vez. Con
la comprobación solo al arrancar no había dos; con el reloj, sí. Lo mismo vale para el cierre: cuando
las dos ramas de un `select` están listas, **Go elige al azar**, así que el bucle vuelve a preguntar
por `ctx.Err()` antes de trabajar.

**El Objective-C de `vidrio_darwin.go` no se puede probar aquí, y eso ya costó una versión rota.** La
2.9.1 añadió un diagnóstico que compilaba en el Mac de la publicación y **cerraba la aplicación al
arrancar**; hubo que revertirlo. Tres trampas que no avisan al compilar: devolver el `UTF8String` de
una cadena autoliberada deja un puntero colgando; `alphaComponent` **lanza excepción** sobre un color
de patrón; y `valueForKey:` puede no existir para leer una propiedad que sí se puede escribir. Que
el trabajo de macOS pase en verde solo dice que compila, **no que arranque**.

**Wails deja el vidrio de macOS a medias, y son tres cosas, no una.** Crea el `NSVisualEffectView`
con mezcla «BehindWindow» pero (1) **no pone la ventana como no opaca**, y una `NSWindow` opaca
compone como opaca aunque su color tenga alfa cero; (2) desde macOS 12 se suma el
`underPageBackgroundColor` del `WKWebView`, que tapa igual; y (3) —la que costó tres versiones—
**nunca le pone el material** a esa vista, que se queda con `AppearanceBased`, obsoleto desde macOS
10.14 y que hoy se dibuja plano. Las tres las remata `vidrio_darwin.go` con cgo (ADR 0017), y **ese
fichero no se puede compilar en esta máquina**: lo comprueba el trabajo de macOS de la publicación.

**Y el atajo que se tardó demasiado en usar: el código de Wails está aquí**, en
`~/go/pkg/mod/github.com/wailsapp/wails/v2@v2.15.0`. Las tres carencias de arriba se leen en
`internal/frontend/desktop/darwin/WailsContext.m` en dos minutos. Antes de razonar sobre lo que Wails
«debería» hacer en una plataforma que no se puede ejecutar aquí, se mira lo que hace.

**Cuando ni leyendo se resuelve, hay compilación con inspector.** `compilar.yml` tiene una entrada
`inspector` en su disparador manual (`gh workflow run compilar.yml -f inspector=true`) que compila
con `wails build -devtools`: en el paquete resultante se abre el Web Inspector y se pueden probar
hipótesis en vivo, en el sistema de verdad, sin publicar una versión por cada una. **No es para
publicar**, y está hecho para que no se pueda confundir: el artefacto se llama `-CON-INSPECTOR`, dura
siete días en vez de treinta y la versión lleva sufijo —lo que además hace que la comprobación de
actualizaciones se calle, porque no son tres números—. Existe porque adivinar costó tres versiones y
una aplicación que no arrancaba.

**Bajo vidrio, la barra lateral no lleva fondo propio.** El material del sistema *es* el fondo, como
en cualquier barra lateral nativa. Aquí se pintó un tinte por delante y se bajó su alfa dos veces
—0,82 y 0,55— persiguiendo un síntoma cuya causa estaba en el material; con la causa arreglada, el
tinte solo puede volver a apagarlo. Hay una prueba de interfaz que lo vigila.

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

**El servidor de desarrollo aísla su carpeta de configuración** (`cmd/dev`, la opción `-config`).
Hasta la bóveda escribía el historial y las preferencias en la carpeta de verdad de quien desarrolla,
que era molesto y poco más. Con una bóveda dentro deja de serlo: las pruebas crearían una con una
contraseña maestra que está escrita en el fichero de pruebas, en el mismo sitio donde va la de
verdad. La carpeta es nueva en cada arranque del servidor, así que **la bóveda de las pruebas
sobrevive entre pruebas y entre temas** pero no entre tandas: el fichero de pruebas la crea o la
abre, según lo que encuentre.

**React no ve un `campo.value = …`, y eso rompió pegar durante versiones.** React sustituye la
propiedad `value` **del elemento concreto** por un accesor que mantiene su registro interno del
valor; al asignar directamente, ese registro se pone al día antes de tiempo y el evento `input` que
se dispara después no le parece un cambio, así que **no llama a `onChange`**. En pantalla la clave
estaba puesta y para la aplicación el campo seguía vacío: el botón de cifrar no se activaba. Como los
menús se construyen a mano (ADR 0015), **⌘V pasa por `ordenes.ts`** y por ahí pasa todo lo que se
pega en la ventana. Se escribe con el accesor del prototipo (`ponerValor`), que no toca el registro.
Nunca se notó porque la prueba del menú ejercitaba «seleccionar todo» y no pegar.

**Y de un campo de contraseña el navegador se niega a copiar.** WebKit y Chromium lo bloquean a
propósito: `execCommand("copy")` dice que sí y el portapapeles se queda como estaba. Eso dejaba la
clave recién fabricada por «Generar una» sin forma de salir de ahí, que es justo la que no está
apuntada en ningún otro sitio. Se copia por Go —que además arma el borrado del portapapeles— y el
campo tiene un ojo para destaparla, **dentro del campo y a la derecha**: fuera competía con «Generar
una» por el mismo sitio y se leía como otra acción del formulario. Su nombre accesible es lo único
por lo que se puede localizar, así que no puede faltar.

**Un filtro de fichero no significa lo mismo en los tres sistemas.** En Windows y en GTK es una
lista desplegable: proponer los `.esf` primero es una comodidad y no impide elegir otra cosa. En
macOS **no**: lo que se manda es la única lista de extensiones que el panel deja seleccionar y todo
lo demás sale en gris. Encima Wails les quita el `*.` de delante antes de dárselos al panel
(`WailsContext.m`), así que el `*.*` de «todos los ficheros» —idiomático en Windows, donde funciona—
llegaba convertido en una extensión llamada literalmente `*`, que no tiene ningún fichero: **el
diálogo se abría sin dejar elegir nada**. En macOS no se manda ningún filtro, y entonces Wails llama
a `setAllowsOtherFileTypes:true`. Lo aguanta `filtrosPara`, que toma el sistema como argumento para
poder comprobar los tres desde aquí.

Y la otra mitad: **un filtro nunca puede impedir elegir**, ni donde hay desplegable. Un `.esf` puede
ser un `.txt` con la línea `ESF1.…` dentro, y una exportación de contraseñas llega con la extensión
que le dé la gana al gestor que la escribió.

**Un gestor de contraseñas no exporta un CSV: exporta varios, y la misma columna cambia de
significado entre ellos.** Dashlane saca cinco —credenciales, notas, tarjetas, documentos e
información personal— y ahí `number` es una tarjeta o un pasaporte según el fichero, `type` es la
clase de tarjeta o la de documento, y `name` es el título de una cuenta o el nombre de una persona.
Por eso `FormaDeLaCabecera` decide **primero** qué se está leyendo y cada forma tiene su tabla de
alias, que pisa a la común. Con una sola tabla no se puede: se acierta en un fichero y se falla en el
otro. Y el tipo de cada entrada se deduce de los campos que vengan rellenos, no de lo que el fichero
diga de sí mismo.

Con ello va una regla: **cada clase de entrada se identifica por lo suyo** (`huellaDeCuenta`). La
huella de una credencial es «sitio + usuario», y una tarjeta no tiene ninguno de los dos: con esa
huella todas las tarjetas del mundo son la misma y importar cinco marcaba cuatro como duplicadas.

**Los avisos y las notas no pueden ser contenedores flexibles.** Lo fueron desde el principio, para
colocar el glifo delante, y con `display: flex` cada trozo del párrafo se convierte en un elemento
por su cuenta: **un `<strong>` en medio de una frase se sale a una columna aparte** y la frase se lee
en vertical, partida en pedazos. No se notó mientras todos los avisos fueron texto pelado, y se vio
mirando una captura, no en una prueba en verde. Lo que se quería es una sangría francesa
—`padding-left` más `text-indent` negativo—, y hay una prueba de interfaz que lo vigila.

**Una respuesta que llega tarde puede deshacer lo que ya se decidió.** Las preferencias se leen más
de una vez —al montar Ajustes y otra vez al terminar de buscar actualizaciones— y una lectura pedida
**antes** de un cambio puede llegar **después**: trae lo de antes, se aplica encima, y el cambio
siguiente parte de ahí y borra el anterior sin que nada lo diga. Se veía como que bajar el bloqueo a
cinco minutos y acto seguido el portapapeles a diez dejaba el bloqueo otra vez en quince. Se resuelve
contando los cambios locales (`cambiosHechos`) y **descartando la lectura si ha habido alguno desde
que se pidió**. Vale para cualquier pantalla que lea y escriba lo mismo.

Y el corolario: **mientras las preferencias no hayan llegado, sus controles van desactivados**. Se
dibujan con su valor de siempre para que la pantalla no dé un salto, pero dejarlos pulsables hace que
el clic no haga nada en silencio, porque `cambiar` no tiene de dónde partir.

**Los ocho cuadros de la bóveda son el único color propio en una pantalla de trabajo**, y entran
matizando la ADR 0021 con una condición: **las 32 parejas están medidas** en `contraste_test.go` —la
letra sobre cada cuadro y cada cuadro contra la lista, en los dos temas—. El tinte lo elige el
dominio con **FNV-1a**, nunca con el hash del motor: tiene que dar lo mismo mañana y en otra máquina,
o la lista cambia de colores sola. Y **la letra sale del nombre, no del dominio**: sacándola del
dominio, «Hacienda» salía con la «A» de agenciatributaria.gob.es.

**Los glifos de los gestores no son sus logotipos**, y no por descuido: nombrar un producto con el
que se interopera es legítimo —el nombre va al lado en texto— pero calcar la marca de otra empresa
dentro del binario es otra cosa. Son formas inspiradas, en nuestro trazo, que además es lo único que
se lee a quince píxeles.

Y de ahí una regla del control segmentado: **`compacto` no viene con `conIconos`**. Las clases de la
bóveda se quedan sin rótulo cuando la ventana se estrecha porque una llave o una tarjeta se adivinan;
los gestores lo conservan siempre, porque cinco marcas ajenas sin su nombre no las reconoce nadie.

**En las pruebas, «Cifrar» es dos cosas.** Nombra la sección de la barra lateral y el botón que
cifra, así que los selectores se acotan: `seccion()` mira dentro de `.lateral` y `accion()` dentro de
`.contenido`. Sin acotar, Playwright encuentra dos y falla por modo estricto.

**Las secciones se esconden, no se desmontan.** Desde la 2.9.0 cada una se monta la primera vez que
se visita y luego se oculta, para que cambiar de pantalla no borre lo escrito. Tres cosas que eso
arrastra: los identificadores de los campos llevan el nombre de la pantalla —`clave-cifrar`,
`clave-descifrar`— porque si no habría dos elementos con el mismo `id`; el `[hidden]` necesita un
`display: none !important` en el CSS, porque el `display: flex` de `.panel` le gana por ser de autor;
y lo que se cargaba al montarse hay que recargarlo **al entrar**, que es lo que hace el historial.
En las pruebas, cualquier selector por clase dentro de un panel se acota con `:visible`, o encuentra
también los escondidos.

**El formato `ESF1` está congelado con vectores fijos, y no se regeneran** (ADR 0022). Viven en
`internal/cripto/testdata/` y comprueban dos caminos: que lo grabado **se abre**, y que sellar con la
misma sal y el mismo nonce **da los mismos bytes**. El segundo es el que importa: sin él, cambiar el
orden de bytes de los parámetros o el offset del contador de segmento **a la vez al escribir y al
leer** deja todos los demás tests en verde y deja de abrir lo ya emitido, en silencio. Se comprobó
que es así antes de darlo por bueno.

Por eso **`azar` es una variable y no una función** en `clave.go`: es la costura que permite
reproducir un contenedor byte a byte. Y por eso **el programa que generó los vectores se borró**: no
hay bandera `-actualizar` que apretar cuando un test se pone rojo. Si se pone rojo, o el cambio está
mal o toca subir la versión del contenedor y grabar vectores nuevos **al lado**.

**El color se genera, no se escribe.** `internal/tema` es la fuente de verdad y produce
`frontend/src/tokens.css` con `make tokens`. Editar el CSS a mano no sirve: hay un test que compara
el fichero con lo que dice Go y falla. Y `make contraste` mide las parejas reales de los dos temas.

**El oro rellena, la piedra escribe** (ADR 0021). Sobre el oro de marca, el blanco da **1,68:1** y no
hay arreglo: oscurecerlo hasta que el blanco cumpla lo convierte en un marrón y se pierde la marca.
Así que `SobreAcento` es la piedra `#2b2b31` —8,38:1— y `Acento`, el rol de *texto*, es la tinta
fuerte del tema. En oscuro el oro sí se leería, y aun así no se usa: una regla que solo vale en un
tema no es una regla.

**Y la trampa que casi se cuela con eso: `make contraste` mide parejas de tokens, no sitios.** El oro
vale para rellenar superficies con texto encima, pero como línea sobre fondo claro da 1,37-1,68:1 y
es invisible. El filete de foco, el borde de la zona de soltar, la barra de progreso y la barra de
acento de Windows van de `--acento` por eso, y si alguien los devuelve a `--relleno` **el test de
color seguirá en verde**. Lo vigila una prueba de interfaz, que enfoca un campo y comprueba que el
borde cambia.

Y un daño colateral que conviene conocer: sin color que gastar, **el botón discreto va subrayado**.
Antes su texto era azul y el color solo decía «esto se pulsa».

Los azules del sistema siguen en `paleta.go` sin usarse, a propósito: son el porqué de
`RellenoLegible` —el `#007aff` de macOS da 3,6:1 con blanco— y borrarlos dejaría esa función
pareciendo un adorno.

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
suscribe antes de preguntar**: al revés queda un hueco por el que el fichero se pierde.

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

**Y el del documento tampoco, que es el mismo error cometido dos veces.** `build/esf.png` —el icono
de los `.esf`, cuyo nombre lo fija `iconName` en `wails.json` y que Wails busca en `build/<nombre>.png`—
era una **copia byte a byte de `build/appicon.png`** hasta la 2.10.1. Ahora lo dibuja
`build/documento.svg`: hoja vertical con la esquina de arriba a la derecha doblada, el dorso del
papel a la vista y la marca sobre una placa oscura, que es la forma de un documento en macOS. La
placa ocupa poco más de la mitad del ancho y va **centrada en la hoja**: con ella más grande el papel
no se veía, y bajada a la mitad inferior se notaba caída. **Lo que dice «documento» es el papel, no
el emblema.**

**El icono es la placa; la marca es la silueta; la silueta es una.** Dentro de la ventana no se usa
`icono.svg`: lleva su placa y sus colores dentro, así que no se puede teñir y en miniatura se lee
como un pegote. Lo que se usa es `build/marca.svg` —la esfinge a trazo, en `currentColor`— importada
en línea con `?raw` desde `build/`, **sin copiarla a `frontend/`**. Dos reglas que ese fichero tiene
que cumplir: nada de `id`, `clipPath` ni `mask`, porque se inserta hasta tres veces en la misma
página y los identificadores chocarían; y el grosor del trazo va como atributo, para poder afinarlo
desde CSS en cada tamaño. Maciza no vale: en monocromo el rostro y el tocado se funden en un borrón.

**Y dos cosas que el trazo obliga y el relleno perdona**, las dos vistas en la ventana y no en el
código: el contorno del tocado **va abierto por abajo**, porque su tramo `h246` queda tapado por el
rostro cuando está relleno y a trazo aparece como una raya cruzando la cara; y **el ojal baja 40**,
porque los doce píxeles que lo separaban de la barbilla se los come medio grosor de trazo.

**El lockup no puede ir al tamaño de las filas.** `--texto-grande` es exactamente el de la
navegación: puesto ahí, el nombre del producto parece una sección más. Va al tamaño de título.

**Y la barra lateral se hunde al pasar por encima, no se levanta** (`--barra-encima`, que existe solo
para eso). `--boton-encima` aclara, que es lo correcto para un botón suelto y lo contrario de lo que
hace una fila de barra lateral.

**La casilla de Ajustes se dibuja a mano**, y no por gusto: la del navegador se pinta con el acento
del sistema y `accent-color` tampoco sirve, porque el tic lo dibuja blanco y eso son 1,68:1 sobre el
oro. Dibujada, el tic es de piedra.

**El icono del documento llega a cada sistema por un camino distinto, y a Linux no llegaba solo.**
De `build/esf.png` salen dos: Wails arma el `esf.icns` del paquete de macOS y el `esf.ico` que el
instalador de Windows copia y registra (`File "..\esf.ico"` en su plantilla NSIS). Pero
`packageApplicationForLinux` **devuelve `nil`**: en Linux Wails no toca las asociaciones, y las pone
el `.deb`. Ahí el escritorio busca el icono del tipo **por su nombre y en su carpeta** —el tipo con
la barra cambiada por un guion, en `hicolor/<tamaño>/mimetypes/`, no en `apps/`—, así que hace falta
instalar `application-x-esfinge.png` aparte. Hasta la 2.10.3, `esfinge-mime.xml` apuntaba al icono de
la aplicación y un `.esf` se veía igual que Esfinge.

**El fondo del DMG y los nombres de los iconos.** El Finder centra cada icono en la posición que le
da `create-dmg` y **escribe su nombre debajo**: con iconos de 96 px, el pie del nombre queda unos 64
px por debajo del centro. Todo lo que el fondo dibuje ahí queda tapado, y eso no se ve hasta montar
la imagen en un Mac. `make ventana-dmg` la dibuja antes, leyendo las posiciones de
`empaquetado/macos/armar-dmg.sh` para que fondo y guion no se separen.

**Lo que una extensión no declara en su manifiesto no existe, y no da error: da `undefined`.** El
código usaba `api.storage.local` y el manifiesto no pedía el permiso `storage`, así que la excepción
saltaba en la primera línea del trabajador de fondo, nadie la recogía y **desde fuera parecía que el
puente con Esfinge no contestaba**. Se persiguieron el socket, los manifiestos de native messaging, el
orden de bytes y el modelo de mensajes de cada navegador: **cinco versiones publicadas por una palabra
que faltaba en una lista**. Lo vio en dos segundos la consola de la extensión —«Inspeccionar» en
`about:debugging`—, que es lo primero que hay que abrir cuando una extensión no hace nada, antes de
razonar sobre nada. Lo vigila ahora `navegador/herramientas/permisos.mjs`, que corre en
`make comprobar` y compara lo que el código usa con lo que el manifiesto declara.

**Y entre el panel y el trabajador de fondo se habla por un puerto, no con `sendMessage`.** Con un
mensaje suelto hay que prometer que la respuesta llega después, y **eso no se promete igual en los
dos navegadores**: Chrome quiere `return true` y una retrollamada; Firefox quiere que el oyente
devuelva la promesa. Con la forma de Chrome, Firefox contesta «Promised response from onMessage
listener went out of scope» y al panel no le llega nada. Un puerto no promete nada —la respuesta es
otro mensaje— y es idéntico en los dos, sin detectar cuál es. Costó una versión publicada.

**En una extensión, `chrome.*` no es lo mismo en los dos navegadores, y `browser` sí.** En Chrome,
`chrome.*` devuelve promesas desde MV3; en Firefox existe solo por compatibilidad y es de
retrollamada, así que `await chrome.runtime.sendMessage(…)` allí recibe `undefined` y lo siguiente
revienta con un error de tipo. Escrito con `chrome.*` funciona en Chrome y **muere en silencio en
Firefox**. Se usa `globalThis.browser ?? globalThis.chrome`, que da promesas en los dos.

Y con ello dos reglas de la extensión que salieron del mismo fallo: **todo lo que arranca un panel va
con red debajo**, porque una excepción sin recoger deja el panel en blanco y eso no le dice nada a
quien lo mira ni a quien lo va a arreglar; y **el trabajador de fondo se compila aparte, en una sola
pieza y sin `import`**, porque en cuanto comparte un módulo con el panel el empaquetador saca un
trozo común, mete un `import` en el trabajador y eso obliga a declararlo como módulo en el
manifiesto —que es justo la clase de detalle que funciona en un navegador y no en el otro—.

**Un freno que vive en la conexión no frena nada, si cada pregunta trae una conexión.** El canal
limitaba a sesenta preguntas por minuto para que nadie reconstruyera la lista de sitios de la bóveda
con un diccionario de dominios —que es justo lo que la ADR 0024 decidió cifrar en el disco—, y el
contador se creaba en `conversar`, o sea **uno por conexión**. La extensión abre **un puerto nativo
por petición**, porque con MV3 el trabajador se muere solo cada pocos minutos y uno de larga vida se
cae igual: cada pregunta llegaba por un proceso nuevo con el contador a cero y el tope no se
alcanzaba nunca. La prueba estaba en verde porque le pasaba **un contador hecho a mano** a sesenta
llamadas seguidas, que es el caso que no ocurre. Ahora los frenos cuelgan del `Servidor` —no hay
dónde poner uno por conexión aunque se quiera— y la prueba abre una conexión por pregunta. **La
regla general:** antes de escribir un contador, preguntarse cuánto vive la cosa donde se guarda.

**Un guion de contenido no puede ser un módulo, y el que lo sea no da error.** Se declara en el
manifiesto y el navegador lo carga como guion suelto: si el empaquetador saca un trozo común con el
panel y le mete un `import`, revienta en la primera línea, en la página de otro y sin que nadie lo
vea. Por eso `pagina.ts` se compila aparte y en una sola pieza, exactamente igual que el trabajador
de fondo y por una razón todavía menos negociable.

**En Firefox, los `host_permissions` de MV3 no se conceden al instalar.** Hay que darlos en el panel
de extensiones. Sin ellos, `sender.tab.url` llega `undefined` —sin error, como siempre—, el origen
viaja vacío y Esfinge contesta que ahí no se rellena: desde fuera parece un fallo de Esfinge y no un
permiso que falta. Es el fallo mudo de `storage` otra vez con otra cara, así que se convierte en una
frase que dice dónde darlo, y `herramientas/permisos.mjs` comprueba que el manifiesto declara
anfitriones si el código lee esa propiedad.

**`tabs.connect` sin `frameId` abre el puerto a TODAS las tramas de la pestaña**, y contesta cada
guion que haya en la página. Eso convierte «pedir algo a la página» en una conversación con varias
voces, y creerse la primera es un fallo con un síntoma rarísimo: en la 2.18.0, el relleno automático
funcionaba en Brevo y en Cloudflare y **el botón «Rellenar» del panel decía que allí no había ningún
formulario**, porque el `iframe` del captcha contestaba antes que la trama de verdad. Dos reglas
salen de ahí: **el oyente del panel vive dentro del guardián de tramas** —una trama de otro origen no
rellena, así que tampoco tiene nada que contestar; antes el guardián solo protegía el relleno
automático, que era la mitad del trabajo— y **el panel espera un momento tras el primer «no»** por si
alguien acierta, en vez de quedarse con el primero que habla.

**Y no todo lo que evita repetirse vale para un clic.** `yaRellenados` existe para que el observador
de la página no entre en un tira y afloja con quien está tecleando; aplicado también al botón del
panel, borrar los campos y pulsarlo **no escribía nada y contestaba «Rellenado.»**. Un clic es una
persona pidiéndolo, y por eso ese camino insiste. Vale para cualquier freno que se añada: preguntarse
si distingue al programa de la persona.

**`Buscar` devuelve las entradas ya pasadas por `SinSecretos`, y eso incluye la semilla del código.**
Parece obvio con la contraseña y no lo es con lo demás: al añadir `tieneCodigo` al protocolo del
navegador se leyó `e.TOTP != ""` sobre lo que devuelve `Buscar`, y **todas las cuentas salían sin
segundo factor**. `SinSecretos` vacía la semilla sin dejar marca de que la hubiera. Lo cazó la prueba
antes de publicar, no la vista. Para saber algo de un campo sensible —aunque solo sea si está— se mira
la entrada entera con `Ver`, y se hace solo con las que ya se van a usar.

**Y el icono de la barra dice el estado de cada pestaña** (ADR 0031): número de cuentas, ✓ si ha
rellenado, candado si la bóveda está cerrada, «!» si algo falla. Se pone al día al cambiar de pestaña
y **cada minuto**, y dos cosas de ahí no se pueden olvidar: **el refresco no se empareja** —con
`pedir`, la ventana de Esfinge avisaría de «un navegador pide permiso» cada minuto sin que nadie
tocara nada—, y **gasta del mismo freno de sesenta preguntas** que el panel y las páginas, así que
agrupa y no repite. Todo lo que decide está en `insignia.ts`, que es una función pura y está probada
entera; lo que la envuelve, no.

**La insignia del icono solo admite texto, y lo que no es texto va dibujado en el icono.** El número,
el ✓ y el «!» son texto y los pinta el navegador; el candado no se le puede pasar en blanco —el emoji
🔒 lo pinta el sistema en color—, así que va dibujado en `build/icono-barra-cerrado.svg` imitando la
insignia. Costó tres versiones llegar al tamaño bueno: **dentro del icono no se puede salir del cuadro
como la insignia de verdad**, así que la placa tiene que llegar a los bordes para parecer igual de
grande.

**Un `stroke-width` puesto en el `svg` no llega a un grupo que trae el suyo escrito.** La marca
(`build/marca.svg`) lleva `stroke-width="58"` en su grupo, y un atributo de presentación le gana a lo
que hereda. Por eso en la ventana el `stroke-width: 34` del historial vacío, puesto en el `svg`, **no
hace nada** y lo que se ve es el 58; y por eso en el panel, donde se puso en el grupo, sí adelgazó las
líneas y la esfinge salió con trazos finos al lado de unos ojos y un ojal gordos (2.20.0). Si se quiere
otro grosor, se pone en el grupo y se mira en una captura.

**Y en Chrome de macOS, un botón con `font: inherit` puede salir con otra letra.** «Rellenar» salía
más pequeño que en Firefox, porque Chrome da a los botones su aspecto nativo y ese aspecto trae su
letra. En el panel los botones llevan `appearance: none` y la familia, el tamaño y el interlineado
fijados; hay una prueba que comprueba que el botón mide lo mismo que el texto de al lado.

**Los colores de la extensión que no pinta nuestro CSS se miden aparte**, en
`internal/tema/extension_test.go`: la silueta contra las barras de Chrome y Firefox, las insignias y el
aviso de la página. Y esa prueba **lee los SVG y el TypeScript**: si alguien cambia un color allí y no
en la medición, se pone roja en vez de seguir midiendo el color viejo.

**Y el panel de la extensión no puede usar una pareja de colores que no esté medida** (ADR 0029).
Importa los tokens de la ventana tal cual, y cada combinación es una de las de `contraste_test.go`,
dicha al lado de cada regla. Si hace falta una nueva, se añade primero ahí. La que casi se cuela: el
texto apagado sobre la superficie elevada no está medido, así que al pasar el puntero por una fila el
usuario sube a `--cuerpo`.

**Unas casillas de código no siempre llevan `maxlength="1"`, y seis campos declarados pueden ser una
sola cosa.** El formulario de segundo factor de Cloudflare son seis casillas que declaran **todas**
`one-time-code`, y ninguna tiene `maxlength="1"`: la primera admite seis cifras para el autorrelleno
del sistema. La 2.19.0 no lo detectaba por los dos lados —seis declarados era «no sé cuál», y sin
`maxlength="1"` no eran casillas—, y lo vio el cliente. Ahora **las casillas se miran antes que el
campo declarado** y cuenta como casilla también un `pattern` de una cifra. Lo que lo destapó fue
**pedir la forma de los campos por la consola**, sin valores, en vez de adivinarla.

**Y leer el código de una biblioteca no es ejecutarlo.** De leer el `otp-field` de Base UI salió que
escribir sus casillas seguidas no le valdría, y se escribió un rodeo. La prueba contra el paquete de
verdad dijo que sí le valía —React atiende cada `input` al momento— y el rodeo se quitó. Leer dice qué
hace el código; **cómo se comporta junto al resto solo lo dice ejecutarlo**, y cuando se puede montar
el componente de verdad en una prueba, eso manda sobre la deducción.

**Y el origen de una página lo pone el trabajador de fondo, no la página.** Lo que llegue por un
puerto llamado «pagina» en el campo `origen` se tira y se pone `sender.tab.url`. Sin esa línea,
cualquier página que consiguiera hablar por ese puerto pediría las cuentas de un banco diciendo que
es el banco.

**Y la contraseña de un envío no vuelve a la página.** Al pulsar «Entrar» la página cambia y el guion
que la leyó muere, así que tiene que esperar en algún sitio: en la memoria del trabajador de fondo
(`ofertasPendientes`), dos minutos y nunca en `storage`. La tarjeta de la página siguiente recibe el
sitio, el usuario y las cuentas, **y manda solo la decisión**; quien guarda es el trabajador, **con el
origen del envío** y no con el de la página donde está la tarjeta. Pasársela a la tarjeta «para que la
devuelva» parece más sencillo y es poner una contraseña en una página donde no se escribió.

Y con ello un detalle que se olvida fácil: **a los tres segundos de enviar no se descarta nada** aunque
el formulario siga ahí. Puede ser la página de antes esperando a que el sitio conteste; solo al cargar
la página nueva un formulario de contraseña visible dice que la contraseña era mala.

**Y en las páginas de dos pasos, en la de la contraseña ya no se sabe quién entra.** Google pide el
correo en una página y la contraseña en otra, y la segunda no tiene campo de usuario: con una sola
cuenta guardada, la 2.21.0 **rellenaba su contraseña aunque se acabara de teclear otro correo**, y al
enviar la tarjeta llegaba sin usuario y ofrecía **actualizar la cuenta de otro**. Lo vio el cliente. Por
eso el trabajador recuerda **el usuario que había** en la página de solo usuario (`usuariosEscritos`,
cinco minutos, mismo sitio), y **se rellena la cuenta de ese usuario o ninguna** (`identidad.ts`).
**Cuenta lo ponga quien lo ponga** —la primera versión solo contaba lo tecleado, y el cliente preguntó
lo evidente: ¿y si el correo lo pone Google solo?—, y se lee también con el clic de «Siguiente», porque
un sitio puede poner el valor sin disparar ningún evento. **Y Google no declara `username` en la página de
la contraseña**: lleva el correo en un `type="email"` escondido con `autocomplete="off"`. Se supo con un
diagnóstico por la consola del cliente —los campos, sin valores de contraseña—, que es lo que hay que
pedir antes de escribir una regla para un sitio que no se puede abrir desde aquí.

**Y el cambio de contraseña casi nunca se declara.** Brevo pone la actual y la nueva sin `autocomplete`,
llamadas `currentPassword` y `newPassword`, y hasta la 2.21.1 eso no ofrecía nada: con dos contraseñas
distintas y sin declarar, no se sabe cuál es la nueva. Ahora el nombre del campo cuenta, **si lo dice de
uno y no del otro** —`newPassword` y `confirmNewPassword` distintas es un error al teclear—. Y un sitio que
cambia sin cambiar de página se mira varias veces tras enviar, no una.

**Nada de la extensión funciona antes del aviso de datos, y se hace cumplir en tres sitios** (ADR 0033):
el panel no pregunta hasta aceptarlo, el trabajador de fondo contesta `sin-consentimiento` a cualquier
puerto y no lanza el puente al refrescar el icono, y el guion de la página no arranca. Con uno solo no
basta: los oyentes de envío de la página leen contraseñas sin preguntar a nadie. **Si cambia lo que dice
el aviso, sube `VERSION_DEL_AVISO`**, y con él la política de privacidad y lo declarado en las tiendas.

**La política de privacidad de la web tiene que decir lo mismo que el aviso del panel** (ADR 0033), y las
dos lo mismo que lo declarado en las tiendas. Viven en `web/privacidad.html` y `navegador/src/panel.html`,
y las enlazan las fichas de Chrome y Firefox: cambiar lo que la extensión hace con los datos sin tocar las
dos es publicar una política falsa.

**Y en la extensión no hay `innerHTML`**: los dibujos pasan por `dibujar` (`dibujo.ts`). Eran nuestros e
inofensivos, pero es lo primero que marca la revisión de Mozilla. Y **Mozilla compila el código fuente y
lo compara byte a byte**: si la extensión importa un fichero nuevo de fuera de `navegador/`, hay que
añadirlo a `herramientas/fuente-de-la-extension.sh` o la versión de Firefox se rechaza.

**Y a la tienda de Chrome no se sube el paquete de siempre**: el suyo va **sin `key`**, porque el
identificador lo asigna la tienda (`publicar.yml` lo genera como `…-chrome-tienda.zip`). El de desarrollo
la conserva, para que quien cargue la extensión a mano tenga el identificador que ya conoce Esfinge. Y
**la primera ficha de Chrome se crea a mano**: la API v2 no crea fichas (`docs/tiendas/pasos.md`).

**En Windows el navegador no mira una carpeta, mira el registro** (ADR 0034): una clave bajo `HKCU` cuyo
valor por defecto es la ruta **absoluta** del manifiesto. Los manifiestos van en `%APPDATA%\Esfinge`, uno
por familia —Firefox y Chrome llevan campos distintos—, y la clave de Chrome la comparten Brave, Vivaldi
y Opera, que no documentan cuál leen. Dos cosas para tocarlo: **las rutas se calculan para `sistema`, no
para `runtime.GOOS`**, así que la tabla de Windows se prueba desde aquí; y **lo único que se ejecuta en
un Windows es la máquina de GitHub al publicar**, que prueba el registro de verdad y que el instalador
lleve el puente, no que un navegador lo lance.

**Y el corolario pequeño, que costó un commit el mismo día: `make comprobar | tail` no dice si
`make` ha fallado.** El código de salida de una tubería es el del **último** mandato, así que
`make comprobar 2>&1 | tail -3 && git commit` compromete igual con `go vet` en rojo: lo que se mira
es el `tail`. Se arregla de dos formas y las dos valen: guardar la salida en un fichero y mirar `$?`,
o `set -o pipefail` antes. Vale para cualquier comprobación que se lea por una tubería, que aquí son
todas porque salen largas.

**Un aviso que no detiene nada deja de leerse, y aquí costó seis versiones.** Los tests los corría
solo el flujo «Compilar», en paralelo con «Publicar» y sin que nadie dependiera de él: una versión
salía aunque estuvieran en rojo. Y estuvieron en rojo de la 2.14.0 a la 2.16.0 —el `.gitignore` con
su `*.esf` se había tragado **los quince vectores fijos del formato**, que tienen esa extensión, así
que la promesa de la ADR 0022 era cierta solo en este disco— mientras GitHub mandaba un correo de
fallo al lado de una publicación en verde. Lo contó el cliente, no una prueba. Ahora **«Publicar»
corre las comprobaciones como puerta** y «Compilar» solo se dispara a mano. Dos reglas que salen de
ahí: lo que hay que respetar tiene que **parar** algo, y al añadir un `.gitignore` conviene
preguntarse qué ficheros del repositorio tienen esa extensión —los vectores llevan ahora su
excepción, y `TestLosVectoresEstanEnElRepositorio` explica dónde mirar si vuelven a faltar—.

**Y la regla que sale de eso, que ya ha costado tres publicaciones: una dependencia de la compilación
no puede colgar de un intermediario.** NSIS se bajaba de Chocolatey en cada publicación, y el día que
su servidor devolvió 503 durante media hora se cayeron **dos publicaciones seguidas** con todo lo
demás en verde. Antes había pasado igual con un repositorio de apt caducado.

Y una corrección que conviene leer, porque el primer arreglo fue una suposición: se dio por hecho que
la imagen de Windows de GitHub **traía NSIS preinstalado**, y no lo trae —lo dijo el registro del
propio flujo, «NSIS no viene con la máquina»—. Ahora se baja **del proyecto NSIS, con su suma
comprobada**: una descarga sin comprobar es peor que la dependencia que sustituye. La regla, entonces,
no es «usa lo que ya está» sino **«si hay que bajar algo, del origen y con su suma»**.

Y la otra mitad de esa regla, que costó la cuarta publicación caída por lo mismo: **del origen, con
su suma, y reintentando**. La 2.18.0 se cayó con todo lo nuestro en verde porque
`go install …/wails` no consiguió hablar con `sum.golang.org` —«net/http: TLS handshake timeout»—.
Un apretón de manos que expira no es un fallo de la compilación, es un segundo malo, y tratarlo como
un fallo tira una publicación y hace que el correo de «ha fallado» deje de significar algo. Lo
envuelve `herramientas/reintentar.sh`, que repite con espera creciente. **Lo que no se hace nunca es
bajar la guardia para que no vuelva a fallar**: nada de `GOFLAGS` saltándose la base de datos de
sumas ni de `--insecure`. Se reintenta exactamente lo mismo.

**Y una que solo se descubre publicando: la máquina de GitHub trae repositorios de apt que no son
nuestros.** `apt-get update` **falla entero** si cualquiera de ellos sirve un índice caducado, así que
un problema en un servidor de Google puede dejar sin publicar una versión de Esfinge —pasó con la
2.14.0—. Los dos flujos quitan esas listas antes de mirar: esta compilación solo necesita GTK y
WebKit de los archivos de Ubuntu.

**Cuatro cosas que solo se descubren compilando de verdad**, todas encontradas en GitHub Actions:

- El `main.go` de la aplicación **tiene que estar en la raíz**, junto a `wails.json`. Wails genera
  los enlaces buscando el paquete main ahí; en `cmd/` falla con «no Go files».
- `fileAssociations` va **dentro de `info`** en `wails.json`. Fuera no da error: simplemente el
  paquete sale sin asociaciones, y eso solo se ve mirando el `Info.plist` del artefacto.
- En Linux, Wails busca `webkit2gtk-4.0` y Ubuntu reciente solo trae la 4.1: hace falta compilar con
  `-tags webkit2_41`.
- Los artefactos de GitHub **no conservan el bit de ejecución**. Una `.app` descargada de ahí no
  arranca hasta que se le devuelve con `chmod +x Contents/MacOS/Esfinge`.

**El servidor de cuentas vive en `servidor/`** (ADR 0036): un Durable Object por cuenta **con la bóveda
dentro, en trozos**, y no en R2, para que comprobar la versión y escribir sean una transacción. Cuatro
cosas que ya costaron algo al escribirlo:

- **`workerd` no implementa las jurisdicciones** y revienta al pedir `jurisdiction("eu")`. Por eso sale de
  la variable `JURISDICCION`, vacía en las pruebas y `eu` en los dos Workers, y **una prueba lee
  `wrangler.jsonc`** y se pone roja si alguien la quita. La de D1 se elige **al crear la base** y no se
  cambia: no se puede crear desde la integración de Cloudflare, que solo da una «preferencia».
- **Sin sus secretos no contesta nada** (`503`): un secreto que falta es `undefined`, y firmar con la
  palabra «undefined» funcionaría sin que nada avisara. `PIMIENTA` **no se cambia nunca**: deja fuera a
  todas las cuentas.
- **El 8787 lo ocupa otro programa** en la máquina de desarrollo; `make servidor` usa el 8790.
- **Cloudflare debilita el `ETag` al comprimir**: la versión `"17"` llega como `W/"17"`. Solo se ve contra
  el servidor desplegado, nunca en local, y el cliente tiene que leer las dos formas.
- **Está desplegado desde el 2026-09-18**: producción en `https://esfinge-cuentas.webcafeina.com`, por
  invitación (`admision`, con `@webcafeina.com`). Desplegar es lanzar a mano `servidor.yml`; los secretos
  los pone el cliente en el panel y **la `PIMIENTA` de producción la guarda él**.
- **pnpm frena las versiones recién publicadas** y, si se le deja, las mete solo en
  `minimumReleaseAgeExclude`. No se acepta: se fijan versiones con una semana, que es lo que ese freno
  pide.

**La fusión de la bóveda la vigila una prueba de tres equipos al azar, y es la que manda** (ADR 0038).
Al escribirla cazó dos fallos que ninguna prueba caso a caso veía: **cada equipo conservaba su orden de
entradas**, así que la bóveda fundida nunca era igual a la del servidor y se la pasaban sin fin —ahora
manda el orden del servidor—; y **un borrado volvía a matar una entrada** que otro equipo había decidido
conservar porque la editó. Tres reglas que salen de ahí y de lo demás:

- **Las fechas tienen resolución de un segundo**, así que nada que importe puede decidirse solo por ellas:
  la revisión desempata antes, y las ranuras se funden a tres bandas contra la base. Una prueba que haga
  dos cosas en el mismo segundo puede pasar por el empate sin ejercitar la regla: la de la lápida no
  cazaba nada hasta que el borrado se movió una hora.
- **La serie del fichero sale del mismo cerrojo que la subida y que la fusión** (`PrepararSubida`,
  `Fusion.Serie`). Leída después, un guardado del navegador en medio se daría por subido sin estarlo.
- **La forma canónica de una entrada no es `json.Marshal`**: claves en orden y sin el escape de HTML de
  Go, porque la extensión tendrá que sacar los mismos bytes en TypeScript (`docs/formato-boveda.md`).

**Juntar dos bóvedas no puede comparar solo identificadores** (ADR 0039, 2.24.2). Dos importaciones del
mismo gestor dan las mismas cuentas con identificadores distintos, y «Juntar» al entrar en la cuenta dejó
**cada cuenta dos veces en los dos Macs del cliente**, propagadas por la sincronización. Y **tampoco puede
comparar «igual en todo»**: fue el primer arreglo (2.24.2) y no encontró ni una, porque dos importaciones
de versiones distintas difieren en lo que la ventana no enseña. La misma cuenta es clase, título, usuario
y secreto (`claveDeCuenta`), y lo demás se junta (`juntarEn`, sobre una `copiaHonda`: la copia plana de Go
comparte listas y mapas, y contar habría escrito en la bóveda). Quitar las repetidas se queda **la de
identificador menor**, para que dos equipos limpiando a la vez no se queden sin ninguna. Cualquier
limpieza que corra en varios equipos tiene que elegir igual en todos.

**Desde la E1, la bóveda existe dos veces: en Go y en `navegador/src/nucleo/`** (ADR 0040), y **cualquier
cambio del formato, de la fusión o de los códigos se hace en las dos**. Lo vigilan las pruebas cruzadas
(`cruzada_test.go` en `internal/boveda`, `codigos` y `cuenta`), que mandan lo mismo a los dos lados y
exigen los mismos bytes; como las de la cuenta, un `go test` suelto se las salta y las activa
`ESFINGE_CRUZADA=1` en `make comprobar` y en la puerta. Tres cosas que salieron al escribirlas:

- **La forma canónica no es `JSON.stringify`**: Go escapa U+2028 y U+2029 aunque no escape el HTML, y
  ordena las claves por bytes UTF-8. `canon.ts` la escribe a mano, y una prueba cruzada la vigila.
- **El TOTP de Go se desborda a diez cifras** (el divisor es un `uint32`), y la extensión lo imita: tiene
  que dar el mismo código que la ventana, aunque sea raro.
- **Un `*/` dentro de un comentario lo cierra**, y `internal/*/…` es justo eso. Y `readFileSync(0)` da
  `EAGAIN` con una tubería grande: la entrada estándar se lee como flujo.

**Y desde la E2, la extensión con cuenta no le pide nada a la aplicación** (ADR 0040): el trabajador de
fondo guarda la bóveda cifrada en `storage.local`, la clave de la abierta en `storage.session` —que se va
al cerrar el navegador y que los guiones de las páginas no leen— y **contesta los mismos verbos del canal**
(`nucleo/fuente.ts`), así que el panel, las páginas y la tarjeta no distinguen. Cinco cosas para tocarlo:

- **Lo de la cuenta —entrar, el código, desbloquear, salir— solo por el puerto del panel.** Una página no
  puede pedir eso; su puerto ni lo mira.
- **La bóveda de TypeScript pasa sus cambios por una cola** (`_exclusivo`). Sin cerrojos, entre dos
  `await` se cuela cualquiera: dos guardados cruzados compartían serie y el segundo no se subía nunca, y
  una fusión pisaba lo que la tarjeta guardara en medio. Hay una prueba que lo caza quitando la cola.
- **El trabajador se muere cada pocos minutos y la bóveda se rehace al despertar** (`laBoveda`), con la
  clave de `storage.session`. Nada que haga falta después puede vivir solo en una variable: **la entrada
  a medias vivía en una y se perdía mientras se iba al correo a por el código** —lo vio el cliente la
  primera vez—; ahora va en `storage.session`, diez minutos.
- **Solo cuenta como actividad lo que llega del panel.** El relleno automático, el refresco del icono y
  la sincronización no, o no se cerraría nunca: la misma regla de la aplicación.
- **La compilación de pruebas** (`ESFINGE_CUENTAS_PRUEBAS`) apunta al servidor local y sale en
  `dist/pruebas`; **las tres configuraciones de Vite tienen que mandar ahí**. La del guion de las páginas
  no lo hacía, y **Chromium rechaza sin decir nada una extensión cuyo manifiesto nombra un fichero que no
  está**. En `navegador/pruebas-reales` la extensión corre de verdad, sin pantalla: la alarma se hace
  sonar desde el propio trabajador (`worker.evaluate`), y **abrir el panel es actividad**, así que una
  prueba del bloqueo tiene que esperar a que se cierre antes de abrirlo. La tarjeta de guardar no se puede
  pulsar desde ahí: va en una sombra cerrada.

**Las pruebas de la cuenta y de la sincronización hablan con el servidor de verdad**, levantado en local
por `herramientas/con-servidor.sh` con el entorno `local` del Worker —frenos holgados, porque todo llega
desde 127.0.0.1: con los de verdad, la cuarta cuenta de la tanda chocaba con el tope de tres altas por IP y
día—. Un `go test` suelto se las salta; `make comprobar` y `publicar.yml` no.

**Desde la A2, el servidor de desarrollo arranca en la bienvenida** (ADR 0039): sin bóveda y sin modo
elegido, la bienvenida tapa la ventana entera. Por eso las pruebas de siempre **eligen local antes de
empezar** (`beforeEach` con `ElegirModoLocal` en `esfinge.spec.ts`), y las de la cuenta
(`cuentas.spec.ts`) tienen sus propias dos ventanas —5174 y 5175, cada una con su Go, con `ESFINGE_GO`
diciéndole a Vite a cuál hablar— y el servidor de cuentas de verdad en el 8792. Tres cosas que salieron
al escribirlas:

- **`fill(await codigo())` lee el buzón antes de esperar al campo**: el argumento se calcula primero, y se
  cogía el código anterior. Se espera a que salga el campo del código y después se lee el buzón.
- **Una captura justo después de `emulateMedia` sale a mitad de la transición de los botones**: en oscuro
  parecían blancos con letra blanca. Se espera un momento antes de retratar.
- **La hoja de estilos viste los campos por su tipo** (`text`, `email`, `password`): uno sin `type`, o de
  un tipo que no esté en la lista, sale con el aspecto del navegador.

## Lo que nunca se ha probado

Y ojo con qué se ha comprobado de verdad: **el diálogo de abrir no**, porque en macOS los ficheros se
arrastran a la ventana y ese camino no pasa por él. Ahí estuvo escondido el fallo del filtro hasta
que llegó importar de otro gestor, que es lo primero que no tiene arrastrar y soltar.

Comprobado ya en un Mac de verdad: el arrastrar y soltar desde el Finder, el diálogo de guardar, el
doble clic en un `.esf`, la imagen de disco —que se monta y se arrastra sin más—, la estructura de la
barra lateral y, desde la 2.10.0, **el vidrio**: se ve, y comparado con el Finder al lado se ve
igual, así que el desenfoque es el estándar de macOS 26 y no hay nada que calibrar.

Y **la actualización desde dentro**, usada de verdad varias versiones seguidas: se descarga, se
reemplaza y se reinicia sola. **Gatekeeper no aparece al actualizar**, lo que confirma el
razonamiento de la ADR 0016 —la cuarentena la pone quien descarga, y ahí descarga Go, no un
navegador—. El aviso de programa no identificado es cosa solo de la primera instalación.

Y **el icono del documento `.esf`**, que el Finder enseña sin necesidad de forzar la caché de
LaunchServices. De verlo puesto salió su corrección: la marca caía baja en la hoja.

Y **la banda de versión nueva sale sola**, con la ventana abierta y sin tocar nada: la prueba buena
del reloj de la 2.10.4, porque hasta entonces solo se comprobaba al arrancar.

Y **el código de un solo uso es el mismo que el de Dashlane**, con los dos programas abiertos uno al
lado del otro (2.15.0). Es la comprobación que cierra el asunto y que ninguna prueba de aquí podía
hacer: los vectores del RFC dicen que el algoritmo está bien, **no** que la semilla que salió de
Dashlane sea la que espera el servicio.

Sin verificar todavía, y es lo único que queda en todo el proyecto: **cómo quedan las estructuras de
Windows y de GNOME** en máquinas de verdad, y ahí mismo el icono de los `.esf`, que en Windows lo
pone el instalador y en Linux el `.deb`.
