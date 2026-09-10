# Estado

Última actualización: **2026-09-10**

## Dónde estamos

Esfinge es una **aplicación de escritorio** con ventana propia, más una línea de comandos que
comparte núcleo y formato. Va por la **2.18.1**. Funciona de punta a punta: cifra y descifra textos y
ficheros, genera contraseñas, **guarda contraseñas en una bóveda cifrada**, lleva un historial de qué
y cuándo, y se compila sola para macOS, Windows y Linux en GitHub Actions.

Desde la 2.12.0 Esfinge **deja de ser un cifrador sin estado**. La bóveda es la fase 1 de sustituir a
Dashlane, decidida con el cliente y con las fases 2 a 4 —autorrelleno, cuentas y compartir— sin
empezar a propósito: se hace la primera, se usa a diario, y solo entonces se decide si las otras
valen su coste.

La 1.x fue una herramienta de terminal con menús. Esos menús se retiraron: su público —el cliente—
tiene ahora una ventana. La línea de comandos se quedó, que es la que se mete en tuberías y scripts.

## Completado

- **Núcleo criptográfico** (`internal/cripto`): XChaCha20-Poly1305 con Argon2id, formato `ESF1`
  versionado, ficheros por segmentos con marca de final. Con tests que cubren la ida y vuelta, cada
  byte alterado, el truncado y la reordenación de segmentos.
- **Aplicación con ventana** (Wails, React 19 + TypeScript): cifrar y descifrar textos o tandas de
  ficheros, arrastrar y soltar, diálogos del sistema, historial e generador de contraseñas.
- **Actualizaciones dentro de la aplicación**: aviso, descarga comprobada con SHA256 y, en macOS y
  Windows, **reemplazo y reinicio sin que nadie arrastre nada** (ADR 0016). En Linux se le pasa al
  gestor de paquetes, que instala como root. Fue la única conexión que hacía el programa hasta la
  2.14.0; ahora son dos, con la de los iconos (ADR 0024). Las dos están dichas y se apagan en Ajustes,
  y `ESFINGE_SIN_RED` las apaga todas.
- **Menús del sistema en español** en los tres sistemas, construidos a mano porque los roles de Wails
  traen los rótulos en inglés escritos a fuego (ADR 0015). Con atajos ⌘1…⌘6 a las seis pantallas.
- **Apertura de un `.esf`** por doble clic o «Abrir con», que abre la pantalla que le toca según lo
  que lleve dentro: ficheros o texto.
- **Línea de comandos**: intacta desde la 1.5.0, con sus códigos de salida distintos por caso.
- **Color generado desde Go** (`internal/tema`), con el contraste de los dos temas medido en cada
  compilación.
- **Compilación automática** de los tres sistemas, más los seis binarios de la línea de comandos.
- **El vidrio del sistema en la barra lateral** —macOS siempre, Windows 11 con Mica—, con la zona de
  trabajo opaca (ADR 0017). Es donde el sistema pone la vibrancia. En Linux no lo hay, y ahí la
  ventana queda como estaba.
- **Los diálogos recuerdan su carpeta**, una para abrir y otra para guardar.
- **Las tandas se cifran en paralelo**, con tope: veinte ficheros pasaron de 4,42 s a 1,29 s en una
  máquina de cuatro núcleos (ADR 0018).
- **Estructura nativa en los tres sistemas**: barra lateral y contenido en todos, y luego lo de cada
  casa —macOS sin barra de título (ADR 0019), el panel de Fluent en Windows y la cabecera de GNOME en
  Linux (ADR 0020)—.
- **La marca, dentro de la ventana** (ADR 0021): el lockup de Esfinge en la barra lateral, la firma
  `▍ webcafeína` al fondo, la esfinge tenue en el historial vacío y la ficha de producto en Ajustes
  —que es lo que la ADR 0007 prometía y nunca se había construido—. Y el acento de la interfaz pasa a
  ser el oro del tocado en vez del azul del sistema.
- **El formato `ESF1`, congelado con vectores fijos** (ADR 0022): once contenedores grabados una vez
  y no vueltos a generar, más los de la 1.5.0 sellados con el código de entonces. Antes lo único que
  decía congelarlo era un test que se miraba al espejo, y un cambio coherente en los dos sentidos
  habría pasado en verde dejando de abrir lo ya emitido.
- **La bóveda** (ADR 0023): local, cifrada, con clave de recuperación, cuatro clases de entrada,
  historial de contraseñas anteriores, importación desde Dashlane, Bitwarden, 1Password, LastPass y
  Chrome —cada gestor exporta **varios ficheros** y cada uno se reconoce por su forma—, exportación
  en claro para poder salir, bloqueo por inactividad, borrado del portapapeles y borrado de la bóveda
  entera pidiendo la maestra. Con su sección en la ventana, sus dos plazos en Ajustes y
  `esfinge boveda listar|ver|codigo|exportar` en la línea de comandos. **No toca `internal/cripto`**,
  así que los `.esf` y las claves ya emitidos siguen valiendo.
- **Los códigos de un solo uso** (ADR 0025): la bóveda calcula el código de seis cifras a partir de la
  semilla que ya guardaba, con su cuenta atrás en la ventana y con `esfinge boveda codigo` para los
  scripts. `internal/codigos`, sin dependencias, con los vectores de RFC 4226 y RFC 6238 enteros.
  Con eso se va la última cosa que obligaba a tener Dashlane abierto —y entra la consecuencia
  incómoda: **el segundo factor pasa a vivir al lado de la contraseña**, dicho tal cual en
  `docs/seguridad.md`.
- **La papelera** (ADR 0026): borrar deja de ser irreversible. Lo borrado se guarda entero, se
  restaura o se tira del todo una a una, se vacía a mano y **se va solo a los treinta días**. El
  coste está dicho en `docs/seguridad.md`: durante esos días la contraseña borrada sigue dentro del
  fichero, igual que las del historial de contraseñas anteriores.
- **La fase 2, entrega 1** (ADR 0027): la extensión del navegador consulta la
  bóveda por un **canal local** —un socket en tu carpeta, no un puerto—, apagado de fábrica y
  encendido en Ajustes. Enseña las cuentas del sitio de la pestaña y copia lo que se le pida.
  Comprobado en un Mac de verdad, en Firefox y en Chrome.
- **La fase 2, entrega 2** (ADR 0028): **rellena los formularios**. Con una cuenta guardada del
  sitio, sola al cargar la página; con varias, desde el panel. Y **en la página no se dibuja nada**:
  el guion lee el formulario, escribe en él y calla. Dos propiedades de la entrega 1 se rompen a
  conciencia y hay que decirlas: **por el canal ya sale una contraseña de verdad** —para escribirla
  en un campo hay que tenerla, y eso no hereda el borrado del portapapeles— y **hay código nuestro en
  cada página `https`**. A cambio, el verbo nuevo lleva su propio freno, más estrecho que el de
  preguntar, y la detección de campos está escrita en negativo: ante la duda, no se rellena.
  **Comprobado en un Mac, en Chrome y en Firefox** (2.18.1): rellena solo al cargar y acierta
  el formulario en Brevo y en Cloudflare, y el botón del panel escribe tras borrar los campos.
- **Pruebas de la interfaz** con Playwright contra el Go de verdad, en tema claro y oscuro, en una
  máquina sin entorno gráfico. Son **84**.

## En curso

**La fase 2 va por la entrega 2.** La 1 —el canal y la extensión que consulta y copia— se comprobó
en un Mac en las dos familias de navegador. La 2 —rellenar— está escrita y probada aquí hasta donde
se puede: **falta verla en sitios de verdad**, que es lo único que ninguna prueba puede decir.

De escribirla salió, sin buscarla, una de la entrega 1: **el freno de preguntas no frenaba nada**. El
contador vivía en la conexión y la extensión abre una conexión por petición, así que el tope de
sesenta por minuto que impedía reconstruir la lista de sitios de la bóveda con un diccionario de
dominios no se alcanzaba jamás. La prueba estaba en verde porque le pasaba un contador hecho a mano a
sesenta llamadas seguidas, que es el caso que no ocurre. Es la misma familia que los iconos.

La 2.15.0 cerró los códigos de un solo uso —comprobados contra Dashlane el mismo
día— y la 2.16.0, la papelera. En medio salió, sin buscarla, una que llevaba desde la 2.12.0:
**borrar una nota segura o una tarjeta dejaba su contenido dentro del fichero**, porque la lista de
campos sensibles estaba escrita en dos sitios y solo uno estaba completo.

**Con la bóveda ya no queda nada de Dashlane por traer al escritorio.** Lo siguiente es la fase 2, y
su puerta sigue siendo de uso y no técnica.

La 2.12.0 salió con la bóveda y de usarla salieron cinco versiones seguidas de
correcciones —2.12.1 a 2.12.5—, todas de cosas que **solo aparecen usando la aplicación en un Mac**:
pegar con ⌘V, copiar de un campo de contraseña, el filtro del diálogo de abrir, el importador que
solo entendía una forma de fichero y una negrita que partía los avisos en columnas. Ninguna se
habría encontrado desde esta máquina. Los tres sistemas tienen ya su estructura. La de macOS está probada en un Mac; la de
GNOME se puede mirar aquí, porque el servidor de desarrollo corre en Linux; la de Windows no la ha
visto nadie.

## Comprobado en un Mac de verdad

- Que la comprobación de versiones funciona: con la última instalada no sale la banda, y «Buscar
  ahora» dice que ya se está al día.
- La descarga de una actualización, con su barra.
- El doble clic en un `.esf`, en los dos momentos —con Esfinge cerrada y con Esfinge abierta— y con
  las dos clases de contenedor: el que lleva un fichero abre la pantalla de ficheros y el que lleva
  un texto abre la de texto, con la línea puesta.
- **Copiar y pegar con los atajos de ⌘**, que era lo que más riesgo tenía: al construir los menús a
  mano se perdieron los selectores nativos y esas acciones pasan por código propio.
- **La estructura de macOS** de la 2.6.x: barra lateral, título sin banda y el resaltado del ratón
  respetando la fila activa.
- **El disco del DMG**, después de tres intentos y de que llegaran capturas de discos de verdad.
- **Usar la contraseña generada como clave**, y que **cambiar de sección ya no borra lo escrito**
  (2.9.0), que era la deuda que dejó abierta la versión anterior.
- **El vidrio, por fin, en la 2.10.0.** «Ahora sí se ve». Y comparado con la barra lateral del Finder
  al lado, **se ve igual**: el desenfoque que parecía excesivo es el que macOS 26 pone en todas las
  barras laterales del sistema. No hay nada que calibrar.
- **La actualización desde dentro, y que Gatekeeper no aparece.** Se ha usado de verdad, varias
  versiones seguidas: se descarga, se reemplaza y se reinicia sola sin preguntar nada. Confirma lo
  que sostenía la ADR 0016: la cuarentena la pone quien descarga, y aquí descarga Go. El aviso de
  programa no identificado es cosa **solo de la primera instalación**.
- **El icono del documento `.esf` en el Finder** (2.10.1): sale, y sin tener que forzar la caché de
  LaunchServices. De mirarlo puesto salió la corrección de la 2.10.2: la placa estaba centrada en 636
  y la hoja tiene su centro en 512, así que caía 124 px baja. Se había bajado a propósito, creyendo
  que macOS pone ahí el emblema de los documentos; no era cierto.
- **La bóveda con datos de verdad, en el Mac** (2026-09-09): una exportación de Dashlane entra
  entera —credenciales, notas seguras, tarjetas y documentos, que son cuatro ficheros— y **la clave
  de recuperación abre la bóveda**, que era lo único que ninguna prueba podía decir. Costó cuatro
  versiones: el filtro del diálogo de abrir en macOS, el importador que solo entendía una forma de
  fichero, las tarjetas que salían todas duplicadas y lo exportado que no volvía a entrar entero.
- **El código de un solo uso es el mismo que el de Dashlane** (2026-09-10, 2.15.0), con los dos
  programas abiertos uno al lado del otro. Es la comprobación que ninguna prueba de aquí podía hacer,
  y cierra las dos mitades a la vez: que **la semilla se importó bien** —que era lo que de verdad
  estaba en duda, porque los vectores del RFC solo dicen que el algoritmo está bien— y que el
  cálculo concuerda con el de un gestor que lleva años en producción.
- **El canal con el navegador, de punta a punta** (2026-09-10, 2.17.6): con Firefox y un sitio de
  verdad, la extensión reconoce el sitio, enseña la cuenta guardada y **copia la contraseña al
  portapapeles**. Comprobado además lo que hace que eso sea aceptable: **el portapapeles se borra
  solo** pasado el plazo de Ajustes, igual que copiando desde la ventana. Y el **código de un solo
  uso** también, contra un sitio con segundo factor de verdad. **Y en Chrome igual** (2.17.7), con el
  identificador fijado por la clave pública del manifiesto: las dos familias de navegador funcionan.
  Era lo único que no se podía ejercitar aquí —no hay navegador con el que probar
  `connectNative`— y por eso costó **seis versiones**: el puente, el socket, el emparejamiento de
  dominios y los manifiestos estaban bien desde el principio, y lo que faltaba era **una palabra en
  una lista**, el permiso `storage` del manifiesto de la extensión. Sin él esa API no da error: es
  `undefined`, y el fallo se veía como si el puente no contestara.
- **La banda de versión nueva sale sola** (2026-09-09), con la ventana abierta y sin tocar nada. Es
  la prueba buena de la 2.10.4: antes la comprobación ocurría **solo al arrancar** y esa banda no
  había aparecido nunca, aunque la portada llevara desde la 2.1.0 prometiendo «una vez al día».
- Y de preguntar si ese icono valía para los otros dos sistemas salió la 2.10.3: en **Windows** sí,
  del mismo PNG y sin tocar nada, pero en **Linux** no llegaba y seguía saliendo el de la aplicación.
  El `.deb` instala ahora `application-x-esfinge.png` en `hicolor/<tamaño>/mimetypes/`, verificado
  aquí con `dpkg -c`. Verlo puesto en un escritorio de verdad sigue pendiente.

## Siguiente acción concreta

**Probar el camino de la extensión con la extensión de verdad cargada.** Es lo primero de la entrega
3 y no un adorno: la entrega 2 salió con la detección de campos probada en un Chromium auténtico y
**con lo que la envuelve sin probar por nadie**, y ahí aparecieron los dos únicos fallos —el oyente
del panel contestando también desde las tramas de otro origen, y `yaRellenados` bloqueando un relleno
pedido a mano—. Los dos los vio el cliente a la primera; ninguno estaba en la parte probada.

La salida está descrita desde el plan de la fase 2: **Playwright puede levantar Chromium con la
extensión cargada** (`launchPersistentContext` con `--load-extension`) y se le puede escribir el
manifiesto de native messaging dentro de ese perfil, con un host de mentira que conteste JSON
preparado. Con eso se ejercita panel → guion → trabajador → puente sin necesidad de Esfinge.

Después, la **entrega 3: guardar y actualizar desde la página**. Al enviar un formulario con una
cuenta que no está en la bóveda, ofrecer guardarla; si está con otra contraseña, ofrecer
actualizarla, con el historial de contraseñas anteriores que ya existe.

Luego el **código de un solo uso rellenado también** (entrega 4), que en Go está desde la 2.15.0, y
las **tiendas** (entrega 5), donde Chrome consigue su identificador definitivo y donde el permiso que
se pide —`https://*/*` y un guion en todas las páginas— pasa por la revisión más estricta que dan las
dos.

**Y lo que solo dice el uso**, que no es una tarea sino una escucha: si el relleno acierta en un sitio
difícil —un banco—, si rellenar solo resulta demasiado —y entonces hace falta un interruptor en
Ajustes, no afinar la detección a ciegas— y si el clic de más molesta cuando hay varias cuentas del
mismo sitio, que es de lo que depende el desplegable dentro del campo.

Y sigue abierta la deuda de Windows, que la entrega 2 no toca: **el manifiesto del navegador va al
registro y no se escribe**, y el instalador no copia `esfinge-puente`.

**Y lo que sigue siendo verdad aunque la fase 2 esté empezada:** la puerta del plan era no empezarla
hasta usar la bóveda a diario, y se empezó con un día. Fue una decisión tomada a sabiendas, no un
descuido, y conviene recordarla si la entrega 2 se hace larga: lo que la justifica es que la bóveda
se use, no que la extensión avance.

De paso sigue pendiente juzgar el oro de la 2.11.0 puesto, y ver **cuántos de los 65 sitios acaban
con icono real**: de ese número depende si hay que leer el HTML de los que faltan.

Después, y sin prisa, sigue pendiente abrir la aplicación en **Windows** y en **GNOME** de verdad.

Para lo que venga: la **compilación con inspector** ya existe.
`gh workflow run compilar.yml -f inspector=true` da un paquete con el Web Inspector abierto, en el
sistema de verdad, para probar hipótesis en vivo sin publicar una versión por cada una. Va marcado
por tres sitios —nombre del artefacto, retención y sufijo en la versión— para que no se confunda con
una compilación normal.

## Bloqueantes

Ninguno técnico. Lo pendiente son comprobaciones que solo puede hacer el humano.

## Preguntas abiertas para el humano

Queda una, y es de código publicado que no se puede ejercitar aquí: en esta máquina no hay Windows ni
GNOME con la aplicación puesta, y lo visual no se afirma desde una compilación en verde.

- **¿Se usa la bóveda?** Es la única pregunta que decide las fases 2 a 4, y no la contesta ninguna
  prueba: la contesta el uso. Con ella hay que mirar tres cosas que aquí no se pueden ver: si el
  CSV de Dashlane se importa entero y bien, si el bloqueo a los quince minutos molesta o tranquiliza,
  y si copiar una contraseña con el borrado a los treinta segundos llega a tiempo de pegarla.
- **¿Cómo queda el oro puesto, en un Mac?** Es la pregunta de la 2.11.0, y sobre todo por **la
  pastilla dorada de la fila activa**: es el cambio más visible de todos, cumple de sobra —8,38:1— y
  es la gramática del icono, pero eso lo dice el cálculo y no el ojo. Si canta, el repliegue está
  pensado: fondo dorado tenue y texto de tinta, como hace Fluent. Mirar también si el ámbar de los
  avisos y el oro se estorban en la pantalla de cifrar, que son vecinos.
- **¿Cómo queda la aplicación en Windows y en GNOME?** La de macOS está juzgada en un Mac; las otras
  dos se escribieron a partir de las convenciones de cada sistema y nadie las ha visto corriendo. Hay
  tres cosas que mirar de una vez: **la estructura** —el panel de Fluent con su barra de acento, la
  cabecera de GNOME—, **el icono de los `.esf`** en el explorador de ficheros, y ahora **el lockup**,
  que ahí sube hasta el borde porque no hay semáforos que esquivar.
