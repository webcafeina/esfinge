# Estado

Última actualización: **2026-09-09**

## Dónde estamos

Esfinge es una **aplicación de escritorio** con ventana propia, más una línea de comandos que
comparte núcleo y formato. Va por la **2.12.7**. Funciona de punta a punta: cifra y descifra textos y
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
  gestor de paquetes, que instala como root. Es la única conexión que hace el programa, dicha y
  apagable en Ajustes.
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
  `esfinge boveda listar|ver|exportar` en la línea de comandos. **No toca `internal/cripto`**, así que
  los `.esf` y las claves ya emitidos siguen valiendo.
- **Pruebas de la interfaz** con Playwright contra el Go de verdad, en tema claro y oscuro, en una
  máquina sin entorno gráfico. Son **68**.

## En curso

Nada a medias. La 2.12.0 salió con la bóveda y de usarla salieron cinco versiones seguidas de
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
- **La banda de versión nueva sale sola** (2026-09-09), con la ventana abierta y sin tocar nada. Es
  la prueba buena de la 2.10.4: antes la comprobación ocurría **solo al arrancar** y esa banda no
  había aparecido nunca, aunque la portada llevara desde la 2.1.0 prometiendo «una vez al día».
- Y de preguntar si ese icono valía para los otros dos sistemas salió la 2.10.3: en **Windows** sí,
  del mismo PNG y sin tocar nada, pero en **Linux** no llegaba y seguía saliendo el de la aplicación.
  El `.deb` instala ahora `application-x-esfinge.png` en `hicolor/<tamaño>/mimetypes/`, verificado
  aquí con `dpkg -c`. Verlo puesto en un escritorio de verdad sigue pendiente.

## Siguiente acción concreta

**Vivir con la bóveda una semana.** Los datos ya están dentro y la clave de recuperación ya se ha
usado: lo que queda no es una comprobación, es uso. La puerta de decisión del plan es de costumbre y
no de código —si no se abre Esfinge para buscar una contraseña, las fases 2 a 4 no se empiezan— y hay
dos cosas concretas que solo dirá el día a día: si el bloqueo a los quince minutos molesta o
tranquiliza, y si **los códigos de un solo uso** hacen falta ya, porque hoy se guarda la semilla pero
no se calcula el código, y con eso los segundos factores se quedan en Dashlane.

Y de paso, juzgar el oro de la 2.11.0 puesto: la identidad entró en la ventana y todo lo que la
sostiene está medido, pero medido no es visto.

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
