# Deuda y cabos sueltos

Última actualización: **2026-09-10**

Lo que sabemos que está a medias, mal o sin comprobar. Los bloqueantes primero. Lo saldado se tacha
y se queda.

## Sin comprobar

Lo más caro de esta lista no es lo que está mal, es lo que no sabemos si lo está.

| Elemento | Severidad | Impacto | Estado |
|---|---|---|---|
| La aplicación no se ha ejecutado en Windows ni en Linux | Media | En macOS está probada de sobra —diálogos, arrastrar y soltar, doble clic en un `.esf`, menús, estructura y vidrio—, pero en los otros dos sistemas todo lo visual sigue siendo una suposición | Abierto · depende del humano. Bajó de Alta a Media cuando la 2.10.0 cerró el frente de macOS |
| El `.deb` no se puede instalar aquí | Media | Se puede inspeccionar con `dpkg -c`, pero instalarlo exige permisos que esta máquina no da | Abierto |
| El instalador de Windows | Media | Se compila, pero nadie lo ha ejecutado | Abierto |
| Los iconos con la bóveda de verdad | Baja | Medido contra doce dominios conocidos: **nueve dan icono**. Los que faltan lo declaran en el HTML, que sigue descartado. Cuántos de los 65 del cliente lo darán es otra cosa, y de eso depende si algún día hay que leer el HTML ([ADR 0024](adr/0024-iconos-de-los-sitios.md)) | Abierto · depende del humano |
| El portapapeles en Windows y Linux | Baja | Va por la API del navegador dentro del webview; en macOS está comprobado. Desde la bóveda, además, **el borrado pasa por Go** (`Sistema.PonerEnPortapapeles`), que en Wails usa la API del sistema | Abierto |
| La bóveda con datos de verdad | Baja | **Comprobado (2026-09-09)**: en un Mac, con una exportación real de Dashlane que entra entera —los cuatro ficheros— y con la clave de recuperación abriendo la bóveda, que era lo único que ninguna prueba podía decir. **Y (2026-09-10) el código de un solo uso coincide con el de Dashlane**, con los dos abiertos a la vez, que era lo mismo para los segundos factores: los vectores del RFC dicen que el algoritmo está bien, no que la semilla importada sea la buena. Lo que queda no es una comprobación sino **uso**: vivir con ella lo bastante para saber si los plazos estorban y si se abre Esfinge o se sigue abriendo Dashlane. De eso depende seguir o parar ([ADR 0023](adr/0023-la-boveda.md)) | Abierto · depende del humano |
| **Nadie de fuera ha auditado esto** | **Alta** | Para un cifrador puntual era una nota al pie; para un gestor de contraseñas publicado en GitHub es la primera pregunta que hará cualquiera. Está dicho en `docs/seguridad.md` | Abierto · decisión de producto |

## Técnica

| Elemento | Severidad | Impacto | Estado |
|---|---|---|---|
| El `main.go` de la aplicación vive en la raíz, no en `cmd/` | Baja | Rompe la organización idiomática de Go. Lo impone Wails, que busca el paquete main junto a `wails.json` | Aceptado · [ADR 0006](adr/0006-de-terminal-a-ventana.md) |
| La interfaz construida se copia a `internal/interfaz/dist` | Baja | Un paso más en la compilación. `go:embed` no puede salir del directorio de su paquete | Aceptado |
| El servidor de desarrollo publica los métodos por reflexión | Baja | Si un método cambia de firma, el fallo sale en tiempo de ejecución y no al compilar | Aceptado · solo existe tras la etiqueta `dev` |
| No hay pruebas de la línea de comandos | Media | `internal/cli` no tenía ni un test: se comprobaba a mano en cada cambio | **Parcialmente saldada (2026-09-09)**: `conSalida` sí tiene pruebas —escribe con 0600, no pisa sin `--forzar`, no deja el fichero a medias, acepta `/dev/null`—, porque se le movieron las tripas a `internal/escritura` y hacer eso sin red es como se rompen las cosas en silencio. El resto de subcomandos sigue sin cubrir |
| ~~Los códigos de un solo uso (TOTP) no se calculan~~ | Media | La bóveda guardaba la semilla y la traía al importar, pero nadie calculaba el código de seis cifras: se enseñaba la semilla, que no sirve para entrar en ningún sitio | **Saldada en la 2.15.0**: `internal/codigos`, sin dependencias, con los vectores de RFC 4226 y RFC 6238 enteros. En la ventana con su cuenta atrás y en `esfinge boveda codigo` ([ADR 0025](adr/0025-los-codigos-de-un-solo-uso.md)). Lo que no puede decir ninguna prueba: si abre una cuenta de verdad |
| `personalinfo.csv` de Dashlane no se importa | Baja | Direcciones, teléfonos y fechas de nacimiento. No son secretos sino datos de autorrelleno, así que meterlos en una bóveda es una decisión de producto por tomar y no un fallo del importador. Si se intenta, el fichero se rechaza diciendo qué columnas trae | Abierto |
| ~~La papelera no se puede vaciar desde la ventana~~ | Media | Borrar es borrado suave —hace falta para sincronizar después— así que una entrada borrada se quedaba en el fichero para siempre, y encima el borrado **no tenía vuelta atrás**: dos clics y la contraseña se había ido | **Saldada en la 2.16.0**: papelera de verdad ([ADR 0026](adr/0026-la-papelera.md)), con restaurar, borrar del todo, vaciar a mano y vaciado solo a los treinta días. El coste, dicho en `docs/seguridad.md`: durante esos días la contraseña borrada sigue dentro del fichero |
| El JSON exterior de la bóveda no va autenticado en su conjunto | Baja | Quien pueda escribir el fichero no puede leer nada ni fabricar una bóveda que abra, pero sí estropearla o revertirla a una copia vieja. `comprobarCoherencia` lo **detecta** con un sello por dentro; no lo impide | Aceptado · [ADR 0023](adr/0023-la-boveda.md) |
| Cifrar una tanda deriva la clave una vez por fichero | Baja | Es el precio de que cada contenedor lleve su sal, y no se va a quitar: compartir la derivación entre ficheros sería compartir la sal | Aceptado. Lo que sí se hizo es paralelizarlo, con tope de la mitad de los núcleos y máximo cuatro: veinte ficheros pasaron de 4,42 s a 1,29 s ([ADR 0018](adr/0018-tandas-en-paralelo.md)) |

## De producto

| Elemento | Severidad | Impacto | Estado |
|---|---|---|---|
| ~~La comprobación de actualizaciones solo ocurre al arrancar~~ | Media | `comprobarAlArrancar` la llamaba `Arrancar` **una sola vez** y no había ningún reloj: con Esfinge abierta no volvía a mirar nunca, mientras la portada y la [ADR 0014](adr/0014-comprobacion-de-actualizaciones.md) prometían «una vez al día». Lo dijo el cliente: nunca le había salido la banda de aviso | **Saldada en la 2.10.4**: `vigilar` deja un reloj que se asoma cada hora y aplica la misma puerta de las 24 h. Tres pruebas con `-race`: que avisa sin reiniciar, que el techo de una petición al día se mantiene, y que el reloj se para al cerrar la ventana (2026-09-08) |
| `frontend/public/icono-256.png` no lo usa nadie | Baja | Se genera, se copia a `dist/` y se embebe en el binario, y no lo referencia ni una línea de código. La marca de dentro de la ventana sale de `build/marca.svg`, así que este PNG solo vale como favicon —que en un webview de Wails tampoco se pinta—. O se usa o sobra | Abierto |
| `VersionVista` se escribe y no se lee | Baja | Se guarda en las preferencias «para no repetir el mismo aviso», y no hay ni un sitio que la consulte: el descarte de la banda es solo de la interfaz y no sobrevive al reinicio. O se usa o sobra | Abierto |
| Gatekeeper avisa en macOS y Windows | Baja | El cliente ve un aviso de programa no identificado. Firmar cuesta 99 $/año y se decidió no hacerlo | Aceptado · [ADR 0012](adr/0012-sin-firmar.md). Bajó de Media a Baja al comprobarse que es **solo de la primera instalación**: al actualizarse desde dentro no aparece, porque la cuarentena la pone quien descarga y ahí descarga Go (2026-09-08) |
| ~~El icono del documento `.esf` era el de la aplicación~~ | Baja | Un fichero cifrado y el programa que lo abre se veían igual: `build/esf.png` era una copia byte a byte de `build/appicon.png`, y en Linux el `.deb` apuntaba al icono de la aplicación | **Saldada (2026-09-08)**: `build/documento.svg` dibuja una hoja con la esquina doblada y la marca sobre una placa. **Comprobado en el Mac**, y la marca se centró en la hoja tras verlo puesto (2.10.2). En Windows sale del mismo PNG sin tocar nada; en Linux hizo falta instalar `application-x-esfinge.png` en `mimetypes/`, comprobado con `dpkg -c` (2.10.3). Sin ver todavía en Windows ni en GNOME de verdad |
| ~~No hay barra de menús propia~~ | Baja | Sin atajos de teclado ni órdenes en la barra del sistema | **Saldada**: se construye entera y en español en los tres sistemas, con atajos ⌘1…⌘6. Los roles de Wails no servían porque traen los rótulos en inglés escritos a fuego, así que las acciones de edición las hace la ventana con una orden ([ADR 0015](adr/0015-menus-en-espanol.md)). Comprobado en el Mac, que era donde más riesgo había (2026-09-07) |

## Saldada

- ~~Los vectores fijos del formato no estaban en el repositorio~~ → el `.gitignore` lleva `*.esf`, que
  es lo razonable en un programa que produce ficheros con esa extensión, y **se tragó exactamente los
  quince vectores**, que tienen esa extensión. Todo lo que promete la [ADR 0022](adr/0022-vectores-fijos.md)
  era cierto solo en el disco de desarrollo, y los tres `v15-*` —sellados con el compilador de la
  1.5.0, con sal y nonce de verdad— **no se pueden volver a fabricar**: se habrían perdido con la
  máquina. Lo decía GitHub en rojo desde la 2.14.0 (2026-09-10, 2.16.1).
- ~~Publicar no comprobaba nada~~ → los tests los corría solo el flujo «Compilar», en paralelo y sin
  que nadie dependiera de él, así que **una versión salía igual con las pruebas en rojo**, y salieron
  seis. Ahora «Publicar» los corre como puerta: si fallan, no se compila ni se publica. Y «Compilar»
  deja de dispararse en las etiquetas, que era trabajo duplicado y el correo de fallo que llegaba al
  cliente (2026-09-10, 2.16.1).
- ~~Lo borrado no bloquea ya su propia reimportación~~ → lo trajo la papelera de la 2.16.0: con la
  entrada entera guardada dentro, el índice de duplicados del importador la reconocía y volver a
  pasar el CSV la daba por repetida. Se habría visto como «la borré, la reimporté y no ha vuelto»,
  con la única copia escondida en la papelera y a punto de caducar. El índice ignora lo que está en
  la papelera (2026-09-10, 2.16.0).
- ~~Borrar una nota segura o una tarjeta dejaba su contenido dentro del fichero~~ → el borrado suave
  limpiaba la contraseña, el TOTP y el historial, que es el secreto de **una credencial**. El texto de
  una nota segura, el número de una tarjeta, su código de verificación y el número de un documento
  —que son el secreto entero de esas otras tres clases— se quedaban tal cual, para siempre y sin
  forma de sacarlos. Había dos listas de campos sensibles en dos sitios y solo una estaba completa;
  ahora hay una (2026-09-10, 2.15.0). **Lo que arregló esto duró unas horas y quedó sustituido por la
  papelera de la 2.16.0**, que guarda lo borrado a propósito porque ya se puede vaciar; el fallo de
  fondo —dos listas de campos sensibles— sigue arreglado.
- ~~El goteo de iconos no guardaba nada hasta terminar la tanda~~ → casi cuatro minutos antes de que
  apareciera el primero, y quien cerraba la bóveda antes no se llevaba ninguno ni de los ya
  descargados. Se guarda y se avisa uno a uno (2026-09-09, 2.14.2).
- ~~La descarga de iconos no funcionaba en absoluto~~ → el filtro de direcciones privadas estaba en
  `DialContext`, que recibe el nombre sin resolver: `ParseIP("github.com")` da nulo y se rechazaban
  todos los sitios del mundo. La 2.14.0 salió así y no descargó ni un icono. Lo dijo el cliente
  (2026-09-09, 2.14.1).
- ~~Un repositorio de terceros roto dejaba sin publicar~~ → la máquina de GitHub trae listas de apt
  que esta compilación no necesita, y `apt-get update` falla entero si cualquiera sirve un índice
  caducado. Un problema en un servidor de Google dejó sin publicar la 2.14.0. Ahora se quitan antes
  de mirar (2026-09-09, 2.14.0).
- ~~`ESFINGE_SIN_RED` solo la miraba la línea de comandos~~ → la ventana no la consultaba nunca, así
  que quien la ponía creyendo que apagaba la red apagaba la mitad. Con una segunda salida eso dejó de
  ser aceptable: vive en `internal/red` y hay una prueba por cada salida (2026-09-09, 2.14.0).
- ~~La bóveda no tenía cerrojo~~ → ya había dos gorrutinas tocándola: la que atiende a la ventana y
  el tic del bloqueo, que llama a `Cerrar()` justo mientras `Guardar` puede estar serializando. Era
  una ventana estrecha que nadie había pillado, y el trabajo de fondo la volvía ancha (2026-09-09,
  2.14.0).
- ~~Reimportar el mismo fichero duplicaba la bóveda entera~~ → los repetidos se marcaban pero se
  metían igual, así que pasar dos veces el `credentials.csv` dejaba 130 entradas donde había 65. Lo
  contó el cliente después de hacerlo. Ahora una entrada idéntica no se mete, y solo entra —marcada—
  la cuenta que ya estaba **con otra contraseña** (2026-09-09, 2.12.6).
- ~~Una lectura de preferencias que llegaba tarde deshacía un cambio~~ → la respuesta de una lectura
  pedida antes de un cambio llegaba después y se aplicaba encima, con lo que el cambio siguiente
  partía de lo viejo y borraba el anterior. Cambiar dos ajustes seguidos dejaba el primero como
  estaba (2026-09-09, 2.12.6).
- ~~Un `<strong>` dentro de un aviso partía la frase en columnas~~ → los avisos y las notas eran
  contenedores flexibles, para colocar el glifo delante, así que cada trozo del párrafo se convertía
  en un elemento por su cuenta. Era así desde el principio y no se notó mientras todos fueron texto
  pelado. Ahora es una sangría francesa (2026-09-09, 2.12.5).
- ~~No se podía borrar la bóveda desde el programa~~ → había que ir al Finder y borrar dos ficheros,
  y quien borraba solo el primero se dejaba una bóveda entera en el `.anterior` (2026-09-09, 2.12.5).
- ~~Las tarjetas y los documentos de Dashlane no se podían importar~~ → el importador tenía una sola
  tabla de alias, y una misma columna significa cosas distintas en cada uno de los cinco ficheros que
  exporta Dashlane. Sus `payments.csv` e `ids.csv` se rechazaban enteros. Ahora se reconoce la forma
  del fichero antes de mapear nada (2026-09-09, 2.12.4).
- ~~Todas las tarjetas se marcaban como duplicadas entre sí~~ → la huella de una entrada era «sitio
  más usuario» y una tarjeta no tiene ninguno de los dos, así que todas tenían la misma. Lo mismo con
  las notas seguras. Cada clase se identifica ahora por lo suyo (2026-09-09, 2.12.4).
- ~~Lo exportado no volvía a entrar entero~~ → el CSV de salida solo llevaba los campos de una
  credencial, así que exportar y reimportar perdía tarjetas y documentos (2026-09-09, 2.12.4).
- ~~Cambiar dos ajustes seguidos perdía el primero~~ → Go recibe el objeto entero y el segundo cambio
  partía del estado de React de antes de que se redibujara, así que deshacía el primero. En pantalla
  los dos se veían puestos. Lo encontró una prueba (2026-09-09, 2.12.4).
- ~~El diálogo de abrir no dejaba elegir nada en macOS~~ → el filtro «todos los ficheros» iba con el
  patrón `*.*`, que es lo idiomático en Windows; Wails le quita el `*.` de delante antes de dárselo
  al `NSOpenPanel`, así que llegaba como una extensión llamada `*` y el panel lo dejaba todo en gris.
  En macOS no se manda ya ningún filtro. Estuvo así desde que existen los diálogos y no se vio nunca
  porque allí los ficheros se arrastran; lo encontró el cliente al ir a importar de Dashlane, que es
  lo primero que no tiene arrastrar y soltar (2026-09-09, 2.12.3).
- ~~Pegar con ⌘V no activaba el botón de cifrar~~ → React no ve un `campo.value = …`: mantiene su
  propio registro del valor en un accesor del elemento, así que al asignar directamente el evento
  `input` posterior no le parece un cambio y no llama a `onChange`. La clave se veía en pantalla y
  para la aplicación el campo seguía vacío. Afectaba a **todos** los campos, porque los menús son
  propios y ⌘V pasa por `ordenes.ts`. Lo dijo el cliente (2026-09-09, 2.12.1).
- ~~La clave generada en Cifrar no se podía copiar~~ → de un campo `type="password"` el navegador se
  niega a copiar, sin decirlo: `execCommand("copy")` devuelve que sí y no copia nada. Ahora se copia
  por Go —que además arma el borrado del portapapeles— y el campo tiene un ojo para destaparla
  (2026-09-09, 2.12.1).
- ~~Un `EventSource` por oyente en el puente~~ → el navegador solo abre **seis conexiones** contra el
  mismo origen y un flujo de eventos no termina nunca, así que a partir del sexto oyente **toda
  llamada al puente se quedaba esperando para siempre**, sin error y sin petición en la red. Con
  cinco la aplicación funcionaba; la bóveda trajo el sexto. Ahora hay una sola fuente para todos
  (2026-09-09, 2.12.0).
- ~~Un guardado de preferencias a medias apagaba el bloqueo de la bóveda~~ → `GuardarPreferencias`
  recibe el objeto entero, así que un objeto incompleto llegaba con los plazos a cero; con el cero
  significando «nunca», eso apagaba en silencio el bloqueo y el borrado del portapapeles. Ahora
  «nunca» es `-1` y el cero conserva lo que hubiera (2026-09-09, 2.12.0).
- ~~El portapapeles no se limpiaba nunca~~ → desde la 2.8.0, «Usar como clave» copiaba una contraseña
  generada en claro y ahí se quedaba, cosa que `docs/seguridad.md` reconocía sin resolver. Ahora lo
  borra un reloj de Go —no un temporizador del webview, que se pausa y muere al recargar— y **nunca
  pisa lo que se haya copiado después** (2026-09-09, 2.12.0).
- ~~El arrastrar y soltar no hacía nada~~ → el modo «zona» de Wails exige declarar una propiedad CSS
  que no se estaba declarando (2026-09-07, 2.0.1).
- ~~Los símbolos de SF Symbols salían como cuadro vacío fuera de macOS~~ → `@supports` no sabe
  detectar fuentes; se volvió a caracteres normales (2026-09-07, 2.0.1).
- ~~El arranque tardaba cinco segundos en terminales que no contestan al OSC 11~~ → lo causaba el
  `init()` de Bubble Tea, que se fue con la interfaz de terminal (2026-09-07, 2.0.0).

| ~~Cambiar de sección borra lo escrito~~ | Media | Cada sección se desmontaba al salir y con ella se iba lo escrito | **Saldada en la 2.9.0**: las secciones se quedan montadas desde la primera visita y se esconden en vez de quitarse (2026-09-07) |

| ~~Medir el vidrio sin romper la aplicación~~ | Alta | El diagnóstico de la 2.9.1 dejaba la aplicación cerrándose sola al arrancar. Tres cosas de ese Objective-C pueden reventar y ninguna avisa al compilar: devolver el `UTF8String` de una cadena autoliberada, que `alphaComponent` lanza excepción sobre un color de patrón, y que `valueForKey:@"drawsBackground"` puede no existir para lectura. Revertido en la 2.9.2 | **Saldada en la 2.10.0, y no midiendo sino leyendo**: el código de Wails está en el caché de módulos de esta máquina, y ahí se ve que nunca le pone material al `NSVisualEffectView`. No hacía falta instrumentar la aplicación del cliente para averiguarlo (2026-09-07) |

| ~~Adivinar en macOS sale caro y no hay banco de pruebas~~ | Media | Tres versiones seguidas —2.5.1, 2.9.0 y 2.9.1— salieron con un cambio de macOS que no se podía probar aquí, y las tres fallaron: dos no hicieron nada visible y la tercera cerró la aplicación, dejando al cliente sin herramienta hasta la 2.9.2. El ciclo «publicar para ver qué pasa» lo sufre él, no yo | **Saldada (2026-09-08)**: `compilar.yml` acepta `inspector` en su disparador manual y compila con `wails build -devtools`, que deja el Web Inspector en el paquete. Cada hipótesis pasa a costar una línea en una consola en vez de una versión publicada. Marcada por tres sitios para que no se confunda con una compilación normal: nombre del artefacto, retención y sufijo en la versión |
