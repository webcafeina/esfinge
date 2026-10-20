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
- **Y desde el 2026-09-29 la bóveda guarda también datos personales** (ADR 0047): nombre, correo,
  teléfono, fecha de nacimiento y **la dirección en sus nueve trozos**, que es el quinto fichero de
  Dashlane. Es **la primera clase
  que no guarda un secreto**, y se hace igual porque la alternativa realista no era tenerlo fuera, era
  tenerlo en Dashlane. Lo eligió el cliente frente a la opción barata —cada fila como nota segura—
  sabiendo que **su fichero son dos filas** y que **Esfinge no rellena nada de eso**. Con ello, dos cosas
  que se dicen en voz alta: lo que Esfinge dice de sí mismo ya no es «solo secretos», y **las pestañas de
  clases se han quedado sin rótulo salvo la activa**, porque seis no caben.
- **Y va a guardar llaves de acceso** (ADR 0048), que es la sexta clase y la primera de las cuatro
  entregas de `docs/passkeys.md`. Lo que hay que saber sin abrir la ficha: **no hay contador de firmas**
  —se firma siempre con cero, como todos los gestores, y por eso esta clase no necesita regla de fusión
  propia—; **no se crean a mano**, porque una llave la emite el sitio; **la clave privada no se enseña, no
  se copia y no cruza el puente**, y cuando llegue la P2 lo que saldrá será la firma ya hecha; y **no salen
  en el CSV en claro**, sino aparte, en un contenedor ESF1 con una clave que no es la maestra.
  **Y una llave lleva dos fechas, y dicen cosas distintas** (2026-10-01): `confirmada` es que **el sitio la
  nombró** —lo único que prueba que la tiene registrada— y `usada`, que **se firmó con ella**, que solo prueba
  que alguien la eligió. **No se mezclan**, porque **ninguna cubre a la otra**: cada camino de
  entrada rellena una. El sitio nombra la llave **cuando ya sabe quién eres** —usuario y contraseña primero y
  la llave después, o sea de segundo factor—, y en el «entrar solo con la llave» no nombra ninguna y la única
  señal es `usada`. Mezclarlas era lo barato y habría hecho pasar la señal débil por la fuerte.
  **Y ojo con cómo se llegó aquí** (corregido el 2026-10-02): esto se escribió diciendo que la señal fuerte
  «casi nunca llega», a partir de una ficha que seguía en blanco tras entrar — y la causa era que **la
  extensión de entonces no escribía ninguno de los dos campos**. Una ficha vacía no prueba que el sitio no
  nombre la llave: prueba que nadie lo apuntó. Una llave sin ninguna de las dos **no se da por mala**: puede ser que no se haya entrado aún.
  `usada` guarda **la última** vez, y por eso cada inicio de sesión escribe en la bóveda — no se reescribe si
  la fecha no ha cambiado, que las fechas van a segundos.
- **Y desde la P2 de las llaves de acceso hay código de Esfinge dentro de cada página `https`** (ADR 0048).
  Es lo más caro de todo el proyecto y conviene saber por qué antes de tocar `navegador/src/mundo.ts`:
  `navigator.credentials` solo existe en el mundo de la página, así que un fallo ahí **no deja el relleno a
  medias, deja el sitio sin poder entrar**, también a quien no use Esfinge en esa cuenta. Todo está escrito
  en negativo —en marco ajeno no se instala nada, sin acuse del puente no se instala nada, si otro gestor ya
  parcheó no se instala nada, y ante cualquier duda se cede llamando al original con el mismo `this`—, y
  **el `shim` cede en la misma vuelta del bucle de eventos** cuando aquí no hay nada: no es una
  optimización, es lo que impide que esperar a un trabajador dormido se coma la activación de usuario. Tres
  cosas más que no se cambian sin preguntar: **el banner sale también con la bóveda cerrada** y para eso el
  navegador se queda **la lista de dominios con llave** en `storage.session` —lo único de la bóveda que
  guarda, y relaja por escrito la regla de la cabecera de `fondo.ts`—; **en esa lista va el `rpId` y no su
  dominio registrable**, porque una llave de `accounts.google.com` no sirve en `mail.google.com`; y **hay un
  interruptor en Ajustes**, encendido de fábrica, que apaga **las tres puertas** —ofrecer, apuntar y
  firmar—. Lo que ese interruptor **no** cubre, y está en `deuda.md`: la extensión con cuenta, que no le
  pregunta nada a la ventana.
  **Y desde la P3 las crea** (2026-09-30), lo que trae dos cosas que hay que decir en voz alta: **se le dice
  al sitio que este equipo tiene un autenticador de plataforma aunque no tenga ninguno** —es lo que los
  sitios preguntan para ofrecer llaves, y sin eso Esfinge no serviría justo donde más falta hace; solo se
  dice cuando Esfinge de verdad puede crear, y ese método **sí puede esperar** la bandera porque no consume
  la activación de usuario—; y **una llave creada en Esfinge y perdida es una cuenta perdida**, que es la
  razón de que la exportación cifrada se hiciera en la P1. El AAGUID va a ceros a propósito: con
  `fmt: "none"` es lo que dice la especificación, y el coste es que el sitio dirá «una llave de acceso» y no
  «Esfinge». Y **no se crea nada** sin `-7`, ni si el sitio dice que ya tiene una llave nuestra, ni con un
  `rpId` que no case con el origen.
- **Y guarda redes wifi** (ADR 0049), que es la séptima clase. Lo que hay que saber sin abrir la ficha:
  **lo que la hace útil es el código QR** —un invitado apunta el móvil y se conecta sin que nadie dicte la
  clave—, el generador **está escrito aquí** (`internal/qr`) y no es una dependencia, por lo de siempre y
  porque además **recibe la contraseña en claro**; **el código se enseña a la vista**, que es la primera vez
  que Esfinge enseña un secreto sin que nadie lo pida y por eso la ficha lo dice con todas las letras; y
  **la contraseña en texto sigue oculta con su ojo** —el código lo lee una cámara apuntada a propósito y el
  texto lo lee quien pase por detrás—. Los colores del código **no salen de la paleta**: negro sobre blanco
  a mano, o en tema oscuro no lo lee ninguna cámara.
- **Y va a haber una bóveda por proyecto** (ADR 0050, empezada el 2026-10-02): la última gran funcionalidad.
  Lo que hay que saber sin abrir la ficha, porque es lo que no se puede volver a discutir: **un proyecto se
  abre con la clave de bóveda de la personal, no con la contraseña maestra**. Suena a lo mismo —el cliente
  pidió «su misma maestra»— y no lo es: envolviéndolo con la maestra, **el día que se recupere los proyectos
  siguen envueltos con la vieja, que ya nadie sabe, y se pierden todos**, y además cambiar la maestra pasaría
  a ser una transacción sobre N ficheros. La clave de bóveda no cambia nunca. De ahí salen gratis dos cosas
  que el cliente ya había aceptado pagar: **una sola entrada en el llavero del sistema** —un diálogo tras cada
  actualización y no uno por bóveda, porque la ranura de Touch ID va solo en la personal— y **una sola
  ceremonia de clave de recuperación**. Lo demás que no se cambia sin preguntar: **una bóveda abierta a la
  vez** —es lo que hace que `App.boveda()` no cambie de firma y que el reloj, el candado, el goteo de iconos y
  el canal del navegador sigan siendo uno—; **la lista de proyectos va dentro del cuerpo cifrado de la
  personal y sin sus claves** —la ranura de cada fichero es lo que abre, así que guardarlas sería amontonar
  las llaves de todos los proyectos por nada—; y **los ficheros se llaman por una referencia al azar**, que un
  listado de la carpeta no puede ser la lista de clientes. **Y esa ranura SÍ se sube**, al contrario que la
  del sistema: es lo único que permite que el otro Mac abra el proyecto, y meterla en `ranurasLocales` es
  media línea que parece lo prudente y no se nota hasta llegar al segundo equipo.
- **Y con un proyecto abierto, el nombre del proyecto es el título** (2026-10-02, tras verlo él):
  antetítulo «Proyecto», el nombre en el `<h1>`, la fila de la barra lateral con el nombre debajo de
  «Bóveda» y **un botón de salir siempre a la vista**. Estaba dicho —el nombre iba en un rótulo
  pequeño al lado— y **no bastó**: lo que más se lee seguía diciendo «Bóveda» y el cliente leyó la
  pantalla como su bóveda personal. Y salir **no estaba en ninguna parte**:
  `VolverALaBovedaPersonal` existía en Go y en el puente y no la llamaba nadie, así que la única
  forma era bloquear la bóveda y desbloquear. **Y salir devuelve tu bóveda abierta y lleva a la lista de
  proyectos** (2026-10-05): antes pedía la maestra y entonces **«Salir del proyecto» y «Cerrar la bóveda»
  hacían exactamente lo mismo** —lo preguntó el cliente—. La razón que había escrita para pedirla no se
  sostenía: decía que la personal no puede quedarse abierta porque dos bóvedas a la vez es lo que se
  descartó, y eso vale para tenerlas abiertas **a la vez**, no para abrir una después de cerrar la otra. No
  abre ninguna puerta: la clave de la personal está en memoria mientras hay un proyecto abierto y
  `conLaPersonal` ya abre ese fichero con ella cada vez que toca la lista de proyectos. Ahora **cerrar
  cierra todo y olvida la clave**, y dentro de un proyecto ese botón se llama **«Cerrar todo»**. El de salir
  mantiene el segundo clic, porque salir de la bóveda de un cliente no es un clic cualquiera. En la barra lateral, el nombre va en un
  rótulo `aria-hidden` y en el `title`: **el nombre accesible del botón sigue siendo «Bóveda»**.
- **Y una bóveda de proyecto se entrega, que es mandar una copia** (ADR 0051): con su contraseña y su clave de
  recuperación, **sin la identidad dentro** —es la semilla con la que se firman los envíos, así que regalarla
  es regalar la firma— y **la contraseña se dice por otro camino**, que lo pone la pantalla. Se puede entregar
  cuando se quiera y tantas veces como haga falta. **Lo que no hay es acceso continuo**, tipo carpeta
  compartida: el cliente lo preguntó el 2026-10-02 y lo dejó así con el coste delante —lo entregado es una
  foto fija y si cambia algo hay que volver a entregarla—, con estas palabras: «que se envíe una copia, como
  las credenciales». Es la misma decisión que la ADR 0043 para una entrada suelta, una escala más arriba, y
  **no se cambia sin preguntar**. Con ello, archivar **borra el fichero de este equipo** y deja el del
  servidor, y borrar **pide la maestra** y no va a ninguna papelera.
- **Y tres días después hizo falta lo contrario, así que también se da acceso** (ADR 0052, 2026-10-05): una
  bóveda de proyecto puede quedarse **viva** en el equipo de otra persona, con sus cambios yendo y viniendo.
  **La 0051 no se deroga**: entregar una copia sigue al lado, para cuando el proyecto de verdad se acaba, y
  son **dos gestos con dos nombres** —«Dar acceso» y «Entregar una copia»—. Lo pidió así: *«necesito que estas
  bóvedas de proyecto sí que se actualicen en tiempo real a las personas que la tengan compartida. Solo las
  bóvedas de proyectos, lo que comparta de mi bóveda no.»* Lo que no se cambia sin preguntar: **«en vivo» es
  al abrir y cada minuto**, con el sondeo que ya existe y **sin push**; **el permiso es por persona**, ver o
  editar, y **lo impone el servidor, no el cifrado** —quien puede ver tiene la clave, lo que no puede es
  subir, y eso se dice con esas palabras igual que lo de Touch ID—; **hace falta cuenta**; **quien puede
  editar puede dar acceso a otros**; **la lista de quién tiene acceso la ven todos**; y **al quitar el acceso
  se rota la clave**. Lo que **no** se hace: borrar en remoto lo que ya se bajó —no se puede prometer— ni
  dejar que quien recibe borre la bóveda para todos.
  **Y tres cosas técnicas que cuestan caro si se tocan a ciegas.** La ranura de cada titular va **sellada
  hacia su identidad pública**, no envuelta con su clave de bóveda: envolviéndola solo esa persona podría
  rehacerla, y entonces rotar al revocar exigiría tenerla delante. **El tipo lleva dentro a quién es**
  (`acceso:<titular>`) y eso no es estilo: el sello indexa los sobres por tipo, así que dos ranuras del mismo
  tipo no hacen que sobre una, hacen que **la bóveda no abra**. Y **quitar una ranura no es representable en
  la fusión**, que une por tipo y conserva la que está en un solo lado: hace falta una **lápida**, o el primer
  equipo con una copia de antes resucita el acceso del revocado.
  **Y desde la C6 la extensión también abre bóvedas compartidas**, con una asimetría escrita a propósito:
  **en el navegador, con permiso de solo ver no se ofrece guardar; en la ventana sí se puede editar**. La
  razón es la pantalla —en la ventana hay dónde explicar que lo escrito no sube, y en la página de otro la
  tarjeta la saca Esfinge por su cuenta—, está en `porQueNoSeEscribe` en los dos lados y **no se iguala «por
  consistencia» sin preguntar**. Del sellado, el espejo lleva **solo abrir**: sellar es de quien da el acceso.
- **Y desde la 2.44.0 un agente de IA puede usar la bóveda, por MCP** (ADR 0054). Lo pidió el cliente, y
  **cambia el modelo de amenaza** como lo hizo la extensión, con uno que no existía: lo que un agente recibe
  **entra en la conversación de un modelo** y queda donde ésa se guarde, y **quien controle su contexto puede
  pedir cosas en tu nombre** —no hace falta atacar a Esfinge, basta con hablarle al que tiene la llave—. De
  ahí sale casi todo lo que no se cambia sin preguntar: **el agente actúa sin ver** —la contraseña va al
  portapapeles y él recibe «copiado»—, con **una sola excepción elegida a sabiendas**, el código de un solo
  uso, que caduca en treinta segundos; **cada secreto se aprueba en la ventana**, con válvula de cinco minutos
  y veinte usos que **no cubre lo que sí se le enseña** ni **lo que no se deshace** —cambiar una contraseña o
  una semilla—; **borrar pregunta** y crear y editar van directos, como el navegador; y hay **registro dentro
  del cuerpo cifrado** de lo que se le ha dado, porque *qué le has dado a quién* dice tanto como lo que
  guardas. **Administrar no**: ni exportar, ni importar, ni la maestra, ni recuperación, ni borrar la bóveda,
  ni cuentas, ni proyectos, ni compartir, ni abrir la bóveda.
  **Y el portapapeles no es una frontera contra un agente con terminal**, que puede leerlo: se acepta, **y se
  dice en la pantalla** con las palabras de Touch ID —lo que protege es que cada uso se apruebe y quede
  apuntado—, no solo en la documentación.
  **Puerta propia** (`internal/agente`) y no el canal del navegador, por tres razones que no son de estilo:
  allí **todo exige un origen `https` con dominio registrable** y un agente **no tiene pestaña**; el testigo
  de emparejamiento **no tiene ámbito** y daría los catorce verbos, incluido firmar llaves de acceso; y los
  frenos **cuelgan del `Servidor`**, así que se comerían los de la extensión y los del refresco del icono.
  **Y MCP está escrito a mano**, sin SDK: son cinco métodos de JSON-RPC, y un SDK sería la primera dependencia
  **dentro de la frontera de seguridad**.
  **Y se instala arrastrando un paquete `.mcpb`** a Claude Desktop, o con `claude mcp add` en Claude Code. Lo
  preguntó el cliente comparando con **Cronos**, que se instala desde Extensiones pegando una URL: allí se
  puede porque es un servidor **remoto**, y **Esfinge no puede serlo** porque la bóveda está en el equipo. El
  paquete lo arma `internal/mcpb` —**sin la herramienta de
  Anthropic**, que es un zip y una dependencia más en el camino de publicar ya costó cuatro publicaciones— y
  cuelga de la *release*, uno por sistema. El binario va **dentro**, con `${__dirname}`, y Claude Desktop le
  añade el `.exe` en Windows por su cuenta. **Y lo arma también la ventana**, con un botón en Ajustes: el
  servidor ya está dentro de la aplicación, así que mandar a alguien a la página de descargas era pedirle que
  acertara con su sistema y con la versión.
  **Y en Ajustes hay cuatro caminos y no uno** (2026-10-08), porque el genérico acierta en la mitad de los
  sitios y en la otra mitad falla callado: **Claude Code necesita `--scope user`** —sin eso el servidor queda
  registrado solo en la carpeta donde se pegó la orden, y la bóveda es una por persona, no una por
  repositorio— y **VS Code lee `servers` donde los demás leen `mcpServers`**. Los dos fallos son mudos. Lo
  primero lo vio el cliente el primer día que lo instaló.
  **Y hay una fila «MCP» en la barra lateral**, encima de Ajustes, que abre ese bloque: lo que no se
  encuentra no se usa. No es una pantalla —lo pidió así—, y **se queda marcada ella** para que el clic no
  parezca no haber hecho nada. El glifo es **nuestro**, como los de los gestores: el símbolo de MCP es marca
  de la Linux Foundation.

- **Y desde la ADR 0055 la bóveda se nombra en cada acción, en vez de ser un modo** (2026-10-14). La 0054
  decía que un agente actúa sobre la abierta y ninguna otra; una semana de uso con varias la puso del revés
  —«es muy molesto tener que ir cambiando de bóveda y que diga todo el rato que no puede hacer algo»—. El
  fallo de fondo era **que fuera un modo**: un modo hay que recordarlo, y lo que un agente no recuerda entre
  llamadas lo paga quien le pide las cosas. Ahora todas las herramientas aceptan **`boveda` por su nombre** y
  Esfinge **conmuta, hace y devuelve la ventana a donde estaba**; sin nombre, la abierta y la regla de la
  2.52.0 —mirar antes de escribir **siempre** y preguntar si es la de un proyecto que nadie ha nombrado—. Lo
  que no se cambia sin preguntar: **sin permiso en la ventana** —lo eligió él frente a preguntar una vez por
  bóveda, y abrir otra no pide contraseña porque la clave ya está en memoria desde la 0050—; **vuelve tras un
  respiro y no tras cada llamada**, porque cada conmutación rearranca la sincronización y el goteo de iconos;
  **no devuelve a nadie a la fuerza** si la persona ha conmutado mientras tanto; y **conmutar por un agente
  no cuenta como actividad ni marca el proyecto como usado**, que son la regla absoluta del reloj y el orden
  de una lista que es de la persona. Y lo que cuesta, dicho en voz alta y escrito en `docs/seguridad.md`: **la
  bóveda abierta era un cerrojo de hecho y deja de serlo** —un agente con una instrucción colada puede
  inventariar la de cualquier cliente; los secretos siguen pidiendo aprobación— y **la lista de proyectos es
  la lista de clientes**, que la 0050 protegía a propósito. **Y la tarjeta de aprobación dice en qué bóveda**
  está pasando: aprobar sin saber dónde es aprobar a ciegas.

- **Y lo que salió de usarlo de verdad el 2026-10-08**, que no se cambia sin preguntar:
  **«No volver a preguntar»** es el tercer botón de la tarjeta —permiso que **no caduca**— y lo pidió al
  aprobar una por una: **lo que cansa se aprueba sin mirar**, que es peor que no preguntar porque da la
  apariencia de control sin el control. Su alcance es **el de la válvula y ni un verbo más**, y lo eligió con
  la alternativa delante —«de todos modos iría a la papelera»—: una frontera es más fácil de sostener que dos
  parecidas. Mientras está puesto, la ventana **lo dice arriba y en todas las pestañas**, con el botón de
  quitarlo al lado: ésa fue la condición con la que se dio.
  **Y Ajustes se recorre por pestañas** —Cuenta · Bóveda · Navegador · Agentes · Actualizaciones—, porque
  ordenarlo en bloques lo hizo legible y no encontrable. **Lo que espera respuesta no vive en ninguna
  pestaña**: lo que pide un agente, lo que pide un navegador y el aviso de los iconos salen arriba, pase lo
  que pase, porque son estado y caducan. **Y hay una fila «MCP»** en la barra lateral que abre esa sección,
  con el **símbolo oficial** —lo pidió él tras ver el nuestro: un glifo inventado para un protocolo no lo
  reconoce nadie—.

- **Y la bóveda se abre también en un servidor, sin nadie delante** (ADR 0057, 2026-10-20). Lo pidió el
  cliente por los agentes que trabajan en el VPS: allí los secretos viven en ficheros `.env` **en claro**
  —111 variables solo en uno— y al leerlos **entran en la conversación de un modelo**, que es lo que la 0054
  vino a evitar en el Mac. Y aquí **no hay ventana donde aprobar nada, y nada la sustituye**: lo que queda es
  acotar el alcance, dejar rastro y que el secreto no pase por la conversación. Lo que no se cambia sin
  preguntar: **el VPS es otra cuenta**, no otro equipo tuyo —si lo fuera, entrar le bajaría tu bóveda
  personal entera—; el permiso es de **solo ver** y lo impone el servidor; **nada corre de forma
  permanente**, así que allí **no existe el canal de agentes** ni hay bóveda abierta entre órdenes; y **se
  guarda la clave de acceso de su cuenta, nunca la maestra**, para que vuelva a entrar solo cuando la sesión
  caduque —frente a entrar a mano cada 90 días, cuyo fallo es mudo—. Lo que cuesta, con las palabras de
  Touch ID y dicho en la pantalla donde se activa: **quien entre en el servidor abre las bóvedas que se le
  marquen**; `esfinge boveda ejecutar` solo consigue que **el camino cómodo no se las enseñe** —el secreto va
  al entorno del proceso hijo, la bóveda se cierra antes de lanzarlo y queda apuntado— y no es una frontera
  contra quien tenga terminal allí. Dos cosas técnicas: **la llave va fuera de la carpeta de las bóvedas**
  (regla de la 0044), y **quitarle el acceso borra su copia** (0053), porque rotar la clave corta lo que
  llega y no lo que ya está en el disco.

- **Y se publica siempre, sin esperar a ninguna tienda** (2026-10-02). Chrome no admite un paquete nuevo
  mientras revisa el anterior, así que publicar durante una revisión deja **ese** paquete fuera de la tienda
  hasta la publicación siguiente — no es un fallo, es un aviso del flujo. Se hace igual: **el cliente para la
  revisión a mano en la consola** si esa versión traía algo de la extensión que le corre prisa. Lo decidió él
  con la alternativa delante —esperar a la aprobación, que son medio día o más por versión— y no se cambia
  sin preguntar. Lo que **sí** conviene decirle al publicar: si esa versión cambiaba `navegador/` y el paquete
  no subió, porque entonces la decisión de cancelar es suya y necesita saberlo.
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
  aplicación**, y copiar desde su panel no se borra solo del portapapeles—. **Con una excepción elegida
  por el cliente**: si se olvida la contraseña maestra, **recuperar se hace en la aplicación**, porque
  exige volver a envolver la bóveda y eso el núcleo del navegador no lo sabe hacer. El panel lo dice. La bienvenida sale **solo a quien estrena Esfinge sin nada** —quien ya tenía bóveda
  trabaja en local sin que se le pregunte—, y **la bóveda que hubiera en un equipo al entrar en una cuenta
  se aparta y no se borra nunca** (ADR 0039). **El registro, por invitación** mientras no se decida abrirlo. Rompe
  dos promesas públicas —«sin servidores» y «Webcafeína no recibe nada»— que hay que cambiar antes de que
  nadie tenga cuenta.
- **Y compartir es mandar una copia a la cuenta de otra persona** (ADR 0043), en la ventana y en el panel
  de la extensión. Tres cosas que no se cambian: **se manda una copia y al llegar es suya** —si luego
  cambia aquí, hay que volver a mandarla—; **lo que llega espera en un buzón** hasta que alguien lo acepta,
  que si no cualquiera que sepa tu correo te escribe dentro de la bóveda; y **la huella se enseña siempre y
  antes**, porque es lo único que protege del servidor en el primer envío y una huella que nadie mira no
  protege nada.
- **Y a quien todavía no tiene cuenta se le invita** (ADR 0043, B3): el servidor le manda un correo **con
  quién le invita y un botón** —lo eligió así el cliente—, **el sobre no sale** y se queda una nota dentro
  de la bóveda de quien lo manda, que lo suelta cuando esa dirección publica sus llaves. Lo que **no** se
  hace, y no se cambia sin preguntar: **mandar la contraseña por correo**, ni siquiera dentro de un enlace.
  Dos consecuencias que hay que decir en voz alta: **Webcafeína le manda correo a alguien que no es cliente
  suyo, con tu dirección dentro**, y **esa copia puede no llegar nunca** —hace falta que abras Esfinge
  mientras la nota viva, treinta días—.
- **Y la bóveda se abre con Touch ID o Windows Hello, sabiendo que es un cerrojo y no una llave**
  (ADR 0044). Sin firmar, el sistema no puede atar ese secreto a Esfinge: protege de quien se sienta
  delante de tu ordenador desbloqueado, **no de un programa que corra como tú**. Se hace igual, **se dice
  con esas palabras en la pantalla donde se activa** y no solo en la documentación, y **no se debe escribir
  de otra forma**. Tres cosas que decidió el cliente el 2026-09-24 y no se cambian sin preguntar: **no se
  firma el Mac** —«de momento»—, **se hace en los tres sistemas** aunque el de Windows se escriba a ciegas,
  y **no hay PIN**: «siempre será la contraseña maestra», así que lo único que la sustituye es la huella y
  donde no la haya la pantalla lo dice en vez de ofrecer algo que no está. Se descartó con ello un PIN como
  desbloqueo rápido de sesión, que daba la misma protección real **sin una línea de código nativo**.
  Y una decisión técnica de la misma familia: **el secreto va al llavero del sistema, nunca a un fichero
  al lado de la bóveda**, porque entonces quien copiara la carpeta abriría la bóveda sin poner el dedo —eso
  no es un cerrojo peor, es quitar la puerta—.
  **Y la huella se pide sola al llegar a la pantalla** (2026-09-25, tras probarlo él): un botón más no es
  desbloquear con el sistema. La huella manda en la pantalla, **late mientras espera** y la maestra queda
  debajo separada por un «o». El coste lo eligió sabiéndolo: **para teclear la maestra hay que cancelar el
  diálogo primero**. El dibujo es nuestro —el glifo de Touch ID es de Apple—, igual que los glifos de los
  gestores.
  **Y la bóveda lo ofrece sola la primera vez**, en una tarjeta arriba: el interruptor vive en Ajustes y
  ahí no entra quien no sabe que existe. **Solo donde hay biometría** —donde no la hay no se dice nada, que
  es ruido sobre algo que no se puede arreglar—, **una sola vez** —se apunta al contestar, en las
  preferencias, que son locales como la ranura— y **diciendo que es un cerrojo**, porque ofrecerlo sin
  decirlo sería venderlo. **Y «una vez» es una vez por bóveda**: lo que se guarda es **de qué bóveda se
  trata**, no un sí/no, y lo decide Go y no la pantalla. Con un sí/no, quien entra en una cuenta —que
  **trae otra bóveda**, sin la ranura, porque es local— se quedaba sin Touch ID **y sin que nadie se lo
  mencionara**. La ranura es de la bóveda; la marca tenía que serlo también. **Y en la pantalla de desbloquear va como casilla** —«Abrir con Touch ID a partir
  de ahora»— y no como botón: **activarlo exige la bóveda abierta**, que es lo que impide encenderlo sin
  saber la maestra, así que se marca, se teclea la maestra y queda activado al abrir. Marcarla **no** cuenta
  como haber contestado: eso lo hacen los dos botones de la tarjeta. **Y la casilla no activa, lo pide**: lo
  hace la tarjeta de dentro, que es la única de las dos pantallas que sigue ahí para decir que ha fallado.

## Trampas que ya costaron encontrarse

**El paquete `.mcpb` no se actualiza con Esfinge, y los clientes se quedan con la lista de herramientas.**
Lleva el servidor **dentro**, que es lo que impide que se rompa al mover la aplicación, y el precio es el
otro lado: al actualizar Esfinge ese servidor se queda como estaba. Y como un cliente MCP pide
`tools/list` **al conectar** y se la queda, una herramienta nueva no aparece ni reiniciando la
conversación. Lo encontró el cliente pidiendo crear una cuenta con su sitio: el agente dijo que su versión
no tenía ese campo, y **tenía razón**. Desde la 2.48.0 el binario del paquete **le cede el turno al de la
Esfinge instalada** —lo lanza con la misma entrada y salida, y manda el de la aplicación porque es con
quien se va a hablar—, y si no lo encuentra sigue él. En **Claude Code esto nunca pasó**: allí la orden
apunta al binario de dentro de la aplicación.

**Lo que un agente escribe en la bóveda tiene que avisar a la ventana, como lo que escribe el navegador.**
`EventoBovedaCambiada` existe desde el canal del navegador —que lo manda en ocho sitios— y su comentario
dice literalmente «alguien de fuera ha escrito en la bóveda»: **un agente es exactamente eso** y no lo
mandaba. El cliente guardó una entrada en la bóveda de un proyecto, el agente dijo la verdad —que estaba
guardada— y **él no la veía** hasta cerrar y abrir. Por eso decir dónde ha ido y avisar van ahora **en una
sola función** que usan los tres verbos: lo que se escribe por separado se olvida por separado, y la vez
anterior se olvidó justo el que no se había pensado (`borrar`).

**Y una descripción de herramienta con basura dentro compila.** Al añadir una frase a las descripciones, un
script dejó escrito literalmente `"' + aviso + '"` dentro de la de `esfinge_editar`: es una cadena Go
válida, así que **nada se puso rojo** y lo habría leído el modelo. Se vio mirando el fichero. Ahora hay una
prueba que busca restos de interpolación en todas, y de paso encontró una herramienta que escribía sin su
aviso.

**Y comparar las dos puntas no prueba que estén unidas.** El arreglo de arriba se quedó a medias cinco
días: el esquema declaraba `sitios` y `etiquetas`, el núcleo sabía aplicarlas, y **el traductor de en medio
no las copiaba** —`ponerArgumentos` recorre los campos de texto uno a uno y ahí no estaban las listas—. El
agente las mandaba, **nadie daba error** y la entrada nacía sin sitio. Y la guarda que se escribió entonces
no lo vio porque comparaba **el esquema contra el núcleo**, o sea los dos extremos. Ahora hay una prueba que
arma los argumentos **a partir del propio esquema** y exige que cada campo declarado llegue a la petición.
Lo encontró **el agente del cliente usando el producto**, al no poder verificar lo que acababa de crear.

**Y lo que no está en el esquema no existe para el modelo.** `crear` y `editar` aceptaban `sitios` y
`etiquetas` desde el principio —el código los aplica— y **no estaban declarados en las herramientas**, así
que el agente creaba credenciales sin sitio: credenciales que no se rellenan solas, que es para lo que
existe la bóveda. El núcleo sabía hacerlo y nadie se lo había dicho al modelo. Lo vigila
`TestLoQueElNucleoAplicaEstaEnElEsquema`, que **lee el fuente** —la lista de campos es un `switch`, no se
saca por reflexión— y compara **por herramienta**: la primera versión las juntaba en un saco y pasaba con
`sitios` fuera de `crear`.

**Un registro que no dice qué se dio no contesta la pregunta para la que existe.** La línea ponía el sitio,
quién y «copiada», y con eso no se sabe si salió la contraseña, el código o si la entrada se borró. Y hay
una distinción que ahí no puede perderse: **la contraseña se copia y el agente no la ve; el código se le
enseña**, y entonces está en la conversación de su modelo para siempre.

**Lo que el binario tiene que hacer solo, la prueba no se lo puede dar hecho.** El servidor MCP se empareja
al arrancar, y **no lo hacía nunca**: solo `estado` contestaba y todo lo demás decía `sin-emparejar`. Lo
tapaban todas las pruebas a la vez, porque **le ponían el testigo a mano** antes de empezar —incluida la que
se llama «la tubería entera», que hablaba con `Atender` directamente—. Lo encontró **el paseo a mano**:
arrancar el binario y hablarle por su entrada estándar como lo haría un cliente de verdad. Con él salieron
otros dos: **el nombre del cliente no llegaba a la ventana** —«Un agente» en vez de «Claude Code»— y un campo
se llamaba `repetidas` llevando dentro las reutilizadas. No es la trampa del ayudante que reimplementa
producción: el ayudante **hacía el paso que faltaba**. La regla general: si una prueba prepara un estado
—un testigo, una sesión, un permiso— que en producción consigue el propio programa, **ese paso no lo prueba
nadie**.

**`tools/list` se contesta con Esfinge cerrada, y por eso el catálogo no vive en `internal/app`.** Claude
Code pide la lista **al abrir la sesión** y se la queda: contestar vacío porque la aplicación no está deja al
agente sin herramientas **toda la sesión**, y volver a abrir Esfinge no lo arregla. El catálogo son datos de
`internal/agente`, que es lo que importa el binario pequeño, y hay una prueba que se pone roja si alguien lo
mueve.

**Las pruebas no apartaban la carpeta de configuración en Windows, y eran siete listas.** `conReloj` ponía
`XDG_CONFIG_HOME`, `HOME` y `USERPROFILE`, y allí `os.UserConfigDir` lee **`AppData`**, que nadie tocaba: en
esta máquina aislaba y **en Windows no**, así que todas esas pruebas compartían la carpeta de verdad del
runner. No se vio porque el trabajo de Windows corre **un subconjunto** —`-run
'Manifiest|Chromium|Perfil|Windows|Registro'`— en el que solo una prueba creaba bóveda; el día que
`TestElRegistroDistingueLaValvula` entró en ese filtro por la palabra «Registro» y corrió antes, la de
proyectos se encontró una bóveda que no había creado. **Tiró una publicación.** El arreglo no fue añadir la
variable que faltaba sino que **haya un solo sitio** —`enConfiguracionDePruebas`—, porque eran siete listas
copiadas y a seis les faltaba Windows; es el mismo patrón que lo de `vaciarLoSensible`. Y como aquí no se
puede ejecutar Windows, lo vigila una prueba que **lee los propios ficheros de prueba** y se pone roja si
alguien vuelve a escribir la lista a mano.

**Y había una cuarta copia, fuera de las pruebas** (2026-10-20): `cmd/dev/main.go` ponía tres de las cinco,
con un comentario diciendo que las apuntaba todas. No hacía daño —ahí solo corre Linux— y es exactamente la
forma en que esto vuelve. La lista vive ahora en `rutas.ApartarLaCarpeta`, que **no importa `testing`** y
recibe el `t.Setenv` que la aplica, y el vigilante **recorre el repositorio entero** en vez de un paquete: un
paquete nuevo queda cubierto sin que nadie se acuerde de nada. La encontró el vigilante el día que se amplió.

**Y `zip` no existe en la máquina Windows de GitHub.** El guion que armaba el `.mcpb` murió ahí con `zip:
command not found` y se llevó el trabajo entero con todo lo demás en verde. Lo arma ahora un programa de Go
(`herramientas/armar-mcpb`), que es lo único garantizado en los tres trabajos porque los tres compilan
Esfinge. Las dos alternativas eran peores: `Compress-Archive` de PowerShell sería **un camino aparte solo
para Windows** —o sea el que nadie prueba desde aquí— y además **no guarda el permiso de ejecución**. Ése,
de paso, era un segundo fallo esperando: lo que se instala en macOS y en Linux sale de ese zip y un servidor
sin el bit puesto **se instala y no arranca**; lo ponía `zip` por costumbre y no lo comprobaba nadie. Ahora
se pone a mano y hay aserción. **La regla general, que ya es la tercera vez: una herramienta de línea de
órdenes que está en los tres sistemas de desarrollo puede no estar en los tres de la publicación.**

**Y un tampón para no escribir tanto puede ser un agujero de auditoría.** Se apuntó como deuda que cada
apunte del registro guarda la bóveda entera, y **se descartó midiendo, no escribiéndolo**: veinte apuntes con
dos mil entradas son 838 ms **en total** —42 ms cada uno— y `EsperaTrasGuardar` ya agrupa las subidas tres
segundos. Pero lo que de verdad lo cierra es otra cosa: **`Boveda.Cerrar()` descarta el cuerpo sin guardar**,
así que con tampón **matar el proceso borraría el rastro** — que es justo lo que podría hacer un agente con
terminal después de pedir cosas. Agrupar en memoria lo que sirve para dejar constancia le quita lo que lo
hace constancia.


**Y la otra mitad, que ya ha costado tres veces: un método del puente que no llama nadie.** Escribir el
método en Go y su envoltorio en `puente.ts` **parece terminar el trabajo**, y no lo es: falta la pantalla.
`VolverALaBovedaPersonal` existía y la única forma de salir de un proyecto era bloquear la bóveda y
desbloquear; `AceptarAcceso` existía y un acceso que llegaba al buzón se enseñaba como un sobre roto que solo
se podía descartar —y el mensaje, «Este envío no es para esta bóveda», **era mentira**—; `ordenar` era un
envoltorio muerto cuyo comentario decía que servía para las pruebas, y las pruebas van directas a
`/api/Ordenar`. Las tres las encontró **alguien recorriendo el camino a mano**, nunca una prueba. Ahora lo
vigila `TestLoQueEstaEnElPuenteLoLlamaLaVentana`, que lee `puente.ts` y exige que cada método se nombre en
algún sitio de `frontend/src/`. Dos cosas de ese vigilante: **busca el nombre, no la llamada**, porque un
método se puede pasar como referencia —`hacer(esfinge.vaciarPapeleraDeBoveda)`— y señalar lo que está bien
es como se consigue que un aviso deje de leerse; y **no dice que la pantalla haga algo útil**, solo que
exista quien lo llame.

**Y el patrón del día, que apareció tres veces seguidas: una prueba escrita por quien acaba de arreglar
algo tiende a ordenar los pasos como funcionan.** El 2026-10-15, las tres: la primera probaba mover **en la
dirección que no necesita conmutar**; la segunda **reintentaba en el acto**, cuando en la realidad pasan
diez segundos; y la tercera **aprobaba después del respiro**, que es el único orden en que el sí sobrevive.
Las tres pasaban en verde con el fallo dentro. Lo que las separa de una prueba buena no es más aserciones:
es **escribir la secuencia con los tiempos de quien la vive** —que tarda en leer, en decidir y en pulsar—
en vez de con los de quien la programa. Y lo único que lo dice es mutar.

**Una operación que conmuta de bóveda se invalidaba a sí misma.** `cambiarBoveda` cierra la válvula y con
ella **el sí concedido**, que es lo correcto cuando el suelo cambia por cualquier otro motivo: un permiso
dado mirando una bóveda no puede valer en otra. Pero `mover` **tiene que conmutar al origen**, así que el
reintento borraba su propia aprobación antes de mirarla y la ventana volvía a pedirla. Para siempre. Lo vio
el cliente moviendo una credencial **de vuelta** a su bóveda, seis veces seguidas, y su agente dejó dicho
dónde mirar: «con qué datos se guarda la aprobación».

Y la asimetría que lo escondía: de la personal a un proyecto **con la personal abierta no hay conmutación de
origen**, así que ahí el sí sobrevivía. Solo fallaba al revés.

**Dos cosas de pruebas de ahí, y las dos ya han costado antes.** La prueba que había **solo ejercitaba la
dirección que funciona**. Y la prueba nueva **tampoco lo cazaba**: reintentaba en el acto, y en la realidad
entre que Esfinge pide permiso y la persona lo da pasan **más de diez segundos**, o sea que el respiro ya ha
devuelto la ventana y es el reintento el que vuelve a conmutar. Sin reproducir ese tiempo pasaba con el
fallo dentro. Lo dijo mutarla, las dos veces.

**Y el arreglo de arriba era medio arreglo, que es lo que de verdad hay que aprenderse de esto.** Guardaba el
sí a mano antes de conmutar y lo reponía después, **dentro de `mover`** — así que tapaba la conmutación de ese
verbo y dejaba fuera **la vuelta automática de la ventana**, que pasa por el mismo `cerrarLaValvula` diez
segundos más tarde. Resultado: aprobando en **menos** de diez segundos el bucle seguía igual, y por eso al
cliente le fallaba siempre —aprueba deprisa— mientras aquí parecía bien. Y dejaba fuera a
`copiar_contrasena`, `codigo` y `borrar`, que en una bóveda nombrada tenían el mismo fallo **sin que nadie lo
hubiera probado**. Dos señales de que se estaba parcheando el síntoma: **el arreglo vivía en un verbo** cuando
la causa era del permiso, y **había que acordarse de repetirlo** en cada sitio que conmutara.

Lo que lo cerró (2026-10-16) fue darle la vuelta: en vez de proteger el sí de quien conmuta, **que el sí nombre
entera la acción que se aprobó** —quién, verbo, entrada, bóveda y destino, en `mismaPeticion`—. Entonces no hay
dónde gastarlo mal y conmutar no tiene de qué protegerlo; `cerrarLaValvula` deja de tocarlo y quien se lo lleva
son los dos gestos que lo quieren, el botón de cortar y retirar al agente. Es **más** estricto que antes, no
menos: antes se comparaba solo el identificador de la entrada. Y la señal de que esta vez estaba en el sitio:
**se cayó el andamio de la versión anterior**. Cuando un arreglo obliga a añadir código en cada sitio que
pueda pisarlo, el sitio es otro. El coste —un sí sobrevive a cerrar la bóveda, dos minutos— está en la ADR
0054 y en `docs/seguridad.md`, dicho con todas las letras.

**Y el lunes siguiente el bucle volvió, por debajo y por otra causa: la misma regla escrita en tres
listas.** «¿Este verbo entra en la válvula?» se contestaba en Go con un literal en cada llamada a `pedir`, en
la tarjeta con un `que !== "codigo"` a mano, y en `concederComo` con una suposición —los dos botones anchos
ponían la válvula o el «siempre» **y nada más**, dando por hecho que el verbo pasaría por ahí—. El día que
`mover` entró en el grupo de los que **no** entran, Go se enteró y las otras dos no.

Dos cosas salían de ahí, y la segunda es la grave. **Una**: aprobar un movimiento con cualquiera de los dos
botones anchos **no concedía nada en absoluto**, así que el reintento volvía a preguntar — bucle **determinista**,
no la carrera de la víspera, y el mismo para el código y para editar una contraseña, **sin que nadie lo
hubiera probado**. **Dos**: `AprobarSiempreAlAgente` guardaba el permiso que no caduca **antes** de mirar el
verbo, así que pulsar «No volver a preguntar» en una tarjeta de mover **encendía el permiso permanente de
copiar contraseñas y de borrar** y el movimiento seguía sin hacerse. La persona lee un movimiento y concede
otra cosa: **un permiso se concede sobre lo que la pantalla enseña, nunca sobre lo de al lado.**

Lo que lo cierra: `PasaPorLaValvula(que)` **es el único sitio** —función del verbo, no argumento—, el valor
**viaja en la petición** y los botones de la tarjeta cuelgan de él. Más dos cinturones, por si vuelve a
desalinearse algo: un alcance amplio sobre un verbo que no entra **concede solo esta vez**, y no se enciende
ningún permiso global que esa petición no pueda usar.

**Y es la tercera vez que esto cuesta un fallo**, después de `vaciarLoSensible` y de
`enConfiguracionDePruebas` —que tiró una publicación—. El patrón no es el descuido: es que **añadir un caso
obliga a acordarse de N sitios**, y acordarse no es defensa. Un sitio, y los demás preguntan; y cuando no se
pueden juntar —Go y TypeScript—, **una prueba que compare las dos listas entre sí**, que es lo que de verdad
cierra la familia. Aquí son dos: una **lee `App.tsx`** y exige que los botones cuelguen del campo de Go, y
otra recorre las herramientas declaradas y las compara con la lista por su nombre.

Y la señal que estaba escrita y no se leyó: el comentario de esa misma tarjeta decía que «ofrecer aquí un
botón que no va a hacer lo que dice sería peor que no ofrecerlo». La regla era correcta; lo que falló es que
**su condición era una copia y no la fuente**.

**Y de ahí la tercera prueba mala seguida con la misma forma.** La primera ejercitaba la dirección que no
conmuta; la segunda reintentaba en el acto; la tercera **forzaba el respiro antes de aprobar**, que es el único
orden en el que el sí sobrevive. Las tres en verde con el fallo dentro. Escribir la secuencia con los tiempos
de quien la vive —la tarjeta sale, se lee, se decide, se pulsa, y eso son **segundos**, no diez— la caza a la
primera.

**Añadir una herramienta deja mintiendo a las de al lado, y en este canal los textos no son
documentación: son lo único que el modelo ve.** En tres días —`boveda`, `bovedas` y `mover`— se quedaron
falsas cuatro descripciones: `estado` decía «todas las demás actúan sobre esa bóveda **y ninguna otra**»
cuando ya se podía nombrar otra, `buscar` decía «busca en la bóveda abierta», `borrar` decía ser «lo único
que quita algo de su vista» cuando `mover` también quita, y `editar` prometía «cambiar el código» sin
tener parámetro para ello. Ninguna la cazó una prueba: las cazó **el agente del cliente leyendo los
esquemas**, y dos de ellas le habrían hecho pedirle a una persona que conmutara de bóveda a mano, que es
justo lo que se acababa de quitar.

**Y lo mismo vale para lo que no es texto.** Al hacer que `mover` pidiera aprobación (2.58.0) se quedó su
`PideAprobacion: false`, de cuando no pedía nada. Nadie lee esa bandera en producción —es documentación con
forma de campo—, y por eso no rompió nada: lo único que la lee es **el paseo a mano**, que imprime «pide un
sí» al lado de cada herramienta. Así que el único sitio donde se leen todas juntas las estaba leyendo mal. Lo
mismo le pasó al registro: `apuntarMovimiento` no ponía `Como`, de la misma época, y los apuntes salían
`"Brevo" · hecho ()` al lado de los `(preguntado)` y `(directo)` de los demás. **Las dos las vio el paseo y
ninguna prueba**, que es la tercera vez que ese paseo encuentra algo. **Al añadir un verbo o un campo, releer
las descripciones de los
vecinos** — y la regla de arriba, la de que lo que el núcleo aplica tiene que estar en el esquema, tiene
gemela: **lo que el esquema promete tiene que existir**.

**Una palabra con dos sentidos le quitó al agente una funcionalidad entera.** `esfinge_bovedas` devolvía
`abierta: false` para las bóvedas que no son la activa, y en Esfinge **«cerrada» quiere decir bloqueada** —la
que pide la contraseña maestra—. El modelo juntó los dos sentidos y contestó «están cerradas, Esfinge no me
deja abrirlas desde aquí». **Y sí podía**: al nombrar la bóveda funcionó a la primera, y lo dijo él mismo —«la
búsqueda funcionó aunque la bóveda figuraba como cerrada»—. Justo lo que la ADR 0055 vino a quitar, devuelto
por el vocabulario. El campo se llama ahora `activa`, y la descripción **desmiente la lectura equivocada en
vez de callarla**: que las demás se nombran igual, que se conmuta **sin pedir contraseña** y qué significa
«cerrada» aquí. Es la trampa de `huella` —la criptográfica y la dactilar— en un sitio donde cuesta más:
**aquí los textos son el producto**, así que una palabra ambigua no es un detalle de estilo.

**Y el permiso de los agentes es del equipo, no de un programa, pero la pantalla decía lo contrario.** El
testigo que guarda el cliente MCP vive en **un fichero por usuario** (`agente-testigo`), así que lo crea el
primero que llega y lo reutilizan todos: el cliente instaló Claude Code, entró con el permiso que le había
dado a Claude Desktop **y no salía en la lista**, que seguía enseñando un solo agente con el nombre del otro
y su botón de «Retirar». Y `permitir` hacía `append`, así que un emparejamiento nuevo habría dejado filas
**válidas y sin uso** —el cliente solo tiene un fichero—, sin forma de saber cuál estaba viva.

Tres cosas lo cierran: `permitir` **sustituye** en vez de acumular, se apunta **quién ha hablado de verdad**
—el nombre viaja en cada petición, y por eso la tarjeta sí decía «Claude Code»— y la pantalla dice «Permiso
de este equipo · Autorizado desde… · Han hablado: …» con un solo Retirar. **Y lo grave de por qué nadie lo
vio en tres semanas: esa lista no tenía ninguna prueba de interfaz.** Lo que no se mira no se mira.

**Una petición nueva pisaba la que alguien estaba leyendo, y el botón aprobaba «lo que hubiera».** Dos
descuidos que por separado no se ven y juntos dejan aprobar a ciegas: `pedir` ponía lo pendiente encima de lo
que hubiera, y `AprobarLoQuePideElAgente` concedía sin que nadie dijera **qué**. Con dos agentes a la vez
—Claude Desktop y Claude Code— eso pasa de verdad. Ahora cada petición lleva **referencia** y la ventana la
devuelve al contestar, así que un sí sobre una tarjeta que ya cambió no concede nada; la segunda **espera
turno**; y lo pendiente **caduca a los dos minutos al mirarlo**, que es lo que hace que la tarjeta se caiga
sola sin un reloj. La referencia se lee **en el render** y no dentro del manejador del clic — leída ahí
volvería a aprobar lo de después.

Y la frontera que costó pensar: **la espera de turno es entre agentes, no dentro de uno**. El mismo agente
pidiendo otra cosa es el mismo interlocutor cambiando de tema, y bloquearlo contra sí mismo lo deja dos
minutos sin trabajar porque nadie contestó lo anterior; lo destapó una prueba que hacía exactamente eso.

**Y dos cosas de ahí.** `mover` contestaba `no-entiendo` cuando pedía aprobación —su traductor de errores se
quedó corto al añadir el verbo, así que el texto llegaba bien y **el motivo era falso**—. Y **cuatro pruebas
preparaban lo pendiente a mano** (`p.pendiente = &q`): mientras eso no tuvo consecuencias pasó
desapercibido, y al darle plazo caducaban al instante y se cayeron todas. Es la trampa del testigo puesto a
mano otra vez: **el ayudante hacía el paso que hace producción**, así que ese paso no lo probaba nadie.

**Un verbo que genera sin decir cuánto largo da invita al modelo a prometer que lo puede cambiar.** El
agente del cliente creó una credencial de Apple, eligió `alnum` por su cuenta —bien razonado: el hex no lleva
mayúsculas y Apple exige una— y añadió «si la rechaza por larga, te la regenero más corta». **No podía**: el
largo iba a fuego, y la única forma de acortarla era generarla con `esfinge_generar` y pegarla en `secreto`,
o sea **hacerla pasar por la conversación de un modelo**, que es lo que `generar` existe para evitar. Dos
cosas salieron de ahí: el esquema **dice ahora lo que da** —48 con hex, 33 con alnum, 30 con símbolos— y hay
`caracteres` en las tres herramientas. **La regla: un parámetro que no existe se deduce igual si el verbo
hace algo que admite grados.**

Y el mínimo de ese parámetro **sale de los bits, no de un número redondo**: el suelo son los 64 bits de
`Generar`, que en hexadecimal son 16 caracteres y con símbolos 10. Un mínimo igual para los tres sería dejar
pasar basura en hex o prohibir largos que con símbolos están bien.

**Mover una entrada no cambia nada de dentro, así que no puede decir que la contraseña es de hoy.** `Poner`
pone la fecha de cambio a ahora —correcto para una edición— y un movimiento pasaba por ahí, así que una
contraseña de septiembre salía «cambiada hoy» **en la columna que se mira para decidir si rotarla**. La repone
`PonerNueva`, que es el único sitio donde se sabe que el contenido no se ha tocado; la de creación ya se
conservaba. Y la prueba **espera a que cambie el segundo**: el formato va a segundos, así que un movimiento
dentro del mismo segundo pasaría por bueno pisando la fecha — es lo mismo que costó la de la lápida.

**Y de una auditoría suya, lo que vale es lo que vio.** De nueve hallazgos, ocho eran ciertos y uno era una
decisión escrita que él no podía conocer: que la semilla del segundo factor **no se ofrezca a propósito**,
porque cualquier cadena de letras es base32 válida y una mal puesta deja la cuenta sin segundo factor sin
que nada avise. Aceptar los nueve habría abierto esa puerta con un argumento ajeno.

**Un plazo de gracia convierte «lo de siempre» en «lo de la llamada anterior».** El respiro de la ADR 0055
—la ventana no vuelve tras cada acción del agente sino a los diez segundos, para que tres acciones en la
misma bóveda no sean seis conmutaciones— dejaba la bóveda **nombrada** activa durante ese rato. Así que la
llamada siguiente **sin** nombrar bóveda trabajaba sobre la de antes: con una lectura es confuso —«esa
entrada no está» de una que sí está, en otra bóveda— y **con una escritura es justo lo que todo esto viene
a impedir**, guardar en la bóveda de un cliente sin que nadie lo haya pedido. Ahora cualquier llamada que no
nombre bóveda **deshace el respiro antes de hacer nada**, así que «sin nombrar» vuelve a significar siempre
lo mismo.

**Y van tres: el agente da el síntoma bien y la causa mal.** La tercera fue el 2026-10-19 — vio que mover
cambiaba la fecha, que es cierto, y dedujo que eso engañaría a `esfinge_higiene` haciendo pasar una
contraseña vieja por nueva. **Falso**: la higiene no mide antigüedad de contraseñas y su propio comentario
dice que «no hay caducidad de contraseñas en este formato, y no se inventa una». El hallazgo valía, pero por
otra razón que la que daba, y aceptar la razón habría llevado a tocar la higiene en vez de `PonerNueva`.
**Comprobar la deducción en el código cuesta dos minutos y es lo que separa un hallazgo de un desvío.**

**Y la mitad de método, que ya ha pasado dos veces en dos días: el agente da el síntoma bien y la causa
mal.** Éste dijo «llamadas en paralelo», y el servidor MCP atiende **en serie** —lee una línea, contesta,
lee la siguiente—, así que por ahí no podía ser. Creerle habría puesto un cerrojo donde no hacía falta
dejando el fallo dentro. **Lo que vale de un informe suyo es lo que vio, no lo que deduce**, igual que de
una persona.

**Avisar de que alguien ha escrito no es avisar de que ha cambiado la bóveda, y la lista solo oía lo
primero.** Son dos eventos: `boveda-cambiada` lo manda quien **escribe** desde fuera, y conmutar manda
`EventoEstadoBoveda`. La lista de la bóveda solo escuchaba el primero, y hasta la ADR 0055 eso bastaba
**por una casualidad**: conmutar con el ratón **rehace esa sección entera** (`selloBoveda`), así que la
lista se rehacía con ella. Desde que **un agente conmuta por debajo** esa casualidad desaparece: el agente
va a la bóveda de un proyecto, **escribe** —y la lista se repinta ahí dentro, con lo suyo, correctamente—,
vuelve a la personal, y ese aviso la lista no lo oye. **Se queda enseñando las entradas del proyecto para
siempre**: filas que al pulsarlas dicen «Esa entrada ya no está en la bóveda», con el contador diciendo «80
entradas en total» porque ése sí se refresca. Lo vio el cliente **a los diez minutos** de publicar la 2.56.0.

Dos cosas de método que costaron encontrarlo. **La primera explicación era otra** —«dos listas en vuelo y
gana la última»— y era plausible, encajaba con el síntoma y **estaba equivocada**; lo único que lo dijo fue
que **la prueba escrita para ella pasaba en verde al mutar el arreglo**. Y la prueba de esto **tiene que
volver por la API y no por el botón de salir**: por el botón, `selloBoveda` rehace la sección y el fallo no
se puede ver, así que la prueba pasaría con él dentro.

Y el corolario para cualquier contador de éstos: **se cuentan peticiones, no cambios**. Contando cambios,
un aviso que llegue entre que se pide la buena y que contesta la tira también, y la lista se queda con lo de
antes. La pregunta no es «¿ha cambiado algo?», es «¿hay una más nueva que ésta?».

**Y la tercera cara de lo mismo: la pantalla existe, y está donde no se mira.** La tarjeta con la que se
aprueba lo que pide un agente —el centro de la ADR 0054, «cada secreto se aprueba en la ventana»— vivía
**dentro de Ajustes**, encima de sus pestañas. Cumplía la letra de la regla que ese mismo fichero tiene
escrita —lo que espera respuesta no vive en ninguna pestaña— y no lo que hacía falta: con la bóveda abierta
en la lista, que es como se trabaja, el agente decía «apruébalo en la ventana de Esfinge» y **en la ventana
no había nada que aprobar**. La petición caducaba a los dos minutos sin que nadie se enterara. Igual el
contador de la válvula con su botón de cortar, y el permiso que no caduca: dar el permiso en un sitio y
esconder cómo se retira es peor que no darlo.

**Lo grave es por qué estaba en verde.** Había **cuatro** pruebas de esa tarjeta, y las cuatro hacían
`seccion(page, "Ajustes").click()` **antes** de buscarla. Una prueba que navega hasta donde está la cosa y
entonces comprueba que está no prueba nada sobre si está donde hace falta. La regla general: **cuando algo
es estado y no configuración, la prueba tiene que buscarlo desde donde esté la persona**, y si puede estar
en varias pantallas, recorrerlas. Lo vigila ahora «la tarjeta de un agente sale en cualquier pantalla, no
solo en Ajustes», que las recorre y además **contesta desde una de ellas**.

Y el detalle que lo tapó al probarlo en el Mac: funcionó porque el cliente **venía de Ajustes → Agentes**
de instalar el paquete, así que la ventana estaba aparcada en la única pantalla donde se veía. Lo mismo que
«no preguntó tras hacerlo una vez no es no pregunta»: funcionar una vez no dice en qué condiciones.

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

**Un código QR que se dibuja bien puede no leerlo ningún móvil, y el que lo prueba no puede ser el mismo
que lo escribe.** El generador de `internal/qr` salía perfecto —tres buscadores, alineación, temporización—,
el descodificador de sus pruebas lo leía sin problema y **ninguna cámara lo reconocía**: los quince bits de
la información de formato se colocaban al revés, el más significativo primero, y un lector busca ahí antes
de mirar los datos. El descodificador no lo cazaba porque leía con la misma idea equivocada. Dos cosas que
deja: **lo que acota un fallo así es sacar los datos probando las ocho máscaras sin mirar el formato** —si
el texto sale entero, el problema está encerrado en esos quince bits— y **lo que lo cierra es un móvil**,
porque aquí no hay `qrencode`, ni `zbarimg`, ni `BarcodeDetector` en el Chromium de las pruebas. Por eso el
vector del código que sí se escaneó está congelado como los del formato `ESF1`: si se pone rojo **no se
regenera**.

**Y una tabla escrita a mano se comprueba contra su propia definición, no contra sí misma.** En `internal/qr`
las secuencias de formato y de versión se recalculan con su BCH y se comparan con las constantes, las dos
tablas de tamaños se comprueban una contra otra, y la corrección de errores se verifica porque datos y
corrección juntos son divisibles por el polinomio generador. Eso caza una cifra mal copiada, que era el
fallo más probable del fichero y el que ninguna prueba de ida y vuelta vería.

**Un acento grave dentro de un literal de plantilla lo cierra**, y los comentarios también cuentan. Los
guiones que las pruebas inyectan en la página van en plantillas, así que un comentario en español bien
escrito —con `código` entre acentos— **rompe el fichero entero** con un error de sintaxis que señala a otra
línea. Dentro de esas plantillas, las comillas angulares.

**`addInitScript` envuelve el código en una función**, así que el `var` de un paquete `iife` inyectado así
**no llega a `window`**. Con `addScriptTag` sí, y por eso el resto de las pruebas de la extensión no se
topan con esto. Hace falta `addInitScript` cuando lo que se prueba es que algo llegue **antes que la
página** —el guion del mundo principal—, y entonces el global se asigna a mano (`codigo + "\nwindow.X = X;"`).
El síntoma es `undefined` y una prueba que dice que no se instaló nada por el motivo equivocado.

**Y un doble no engaña a `Function.prototype.toString.call`.** Poner un `toString` propio en una función no
sirve: `call` va al del prototipo y devuelve el fuente de verdad. Para que un doble parezca nativo hay que
sustituir `Function.prototype.toString`, que además es lo que haría otro gestor de verdad.

**`postMessage` no está en `Window.prototype`: es propiedad propia del objeto global, y se puede
sustituir.** Capturarla del prototipo para tener «la de verdad» —que es lo que parecía obvio— devuelve
`undefined` y revienta con «Cannot read properties of undefined». Lo dijo preguntárselo a un Chromium, no
razonarlo. Es `writable` y `configurable`, así que la página **puede cambiarla**, y por eso el puente de las
llaves de acceso la captura **al cargar el módulo**, que en el mundo principal es `document_start`: antes
del primer `<script>` de la página. Esa es la mitad del porqué de `document_start`; la otra es que el saludo
del puente se perdería si el mundo aislado arrancara tarde.

**Una privada P-256 no se puede importar sabiendo solo el escalar.** WebCrypto exige `x` e `y` y no expone
ninguna forma de multiplicar por el generador: `importKey("jwk", {kty:"EC", crv:"P-256", d})` contesta
`DataError`. La ADR 0048 daba por hecho lo contrario —«la parte pública se recalcula»— y solo se vio al
implementarlo. Las llaves de acceso guardan **PKCS#8**, que lleva las dos partes dentro y lo entienden los
dos lados sin escribir una línea.

**Y WebCrypto firma y verifica en P1363, nunca en DER.** WebAuthn exige DER, así que la extensión convierte
`r ‖ s` a ASN.1 a mano. Dos reglas que ese formato no perdona: un `INTEGER` va **sin ceros por delante**, y
si el primer bit está a uno hay que **añadir un cero** o se lee negativo. La prueba de punta a punta —firmar
allí y verificar aquí— **no cubre el recorte de ceros**, porque una firma al azar empieza por cero una vez
de cada 256: eso lleva vectores fijos aparte, y se comprobó mutándolo. La asimetría tiene un corolario útil:
«firmar aquí y verificar allí» no se puede probar sin escribir un descodificador de DER que no hace falta en
ningún sitio, así que **no se prueba y se dice por qué**.

**Y el origen que va dentro del `clientDataJSON` no es pegar esquema y anfitrión.** Con el puerto por
defecto escrito a mano —`https://github.com:443/`— eso da `https://github.com:443` y el navegador y el
sitio dicen `https://github.com`: cuatro caracteres de diferencia y la firma se rechaza **sin decir por
qué**. En la extensión lo escribe `new URL(x).origin`; en Go, `OrigenDe`, que además tiene que **devolver
los corchetes de una IPv6** porque `Hostname()` se los come. Las dos cosas las cazó la prueba cruzada de
tabla, no la lectura.

**`Encaja` no vale para decidir con qué `rpId` se firma una llave de acceso** (ADR 0048), por mucho que se
le parezca. `Encaja` compara **dominio registrable contra dominio registrable**, y por eso
`accounts.google.com` y `mail.google.com` son «el mismo sitio»: es lo que se quiere para ofrecer una
contraseña. WebAuthn pide otra cosa y más estrecha —**el anfitrión o un sufijo suyo separado por punto, y
nunca un sufijo público**— y además **el hash se calcula sobre la cadena exacta**, así que dar por buenos dos
nombres distintos no es ser tolerante: es firmar para quien no es. Eso vive aparte, en `RPIDPermitido` /
`rpIdPermitido`, con tabla propia en los dos lados y prueba cruzada.

Y el caso que hay que tener en la tabla, que **lo encontró una mutación y no la lectura**: sin exigir el
punto que separa, `malaejemplo.com` **termina en** `ejemplo.com` y firmaría por él. La tabla tenía
`ejemplo.com.malo.com` —que no ataca nada, porque ahí el nombre va en medio— y con ella la comprobación se
podía quitar entera sin que nada se pusiera rojo.

**Un saludo en un solo sentido entre dos guiones de contenido no se tiende, y el fallo es mudo.** Los dos
—el del mundo aislado y el del principal— entran en `document_start` y **su orden no está garantizado**;
peor aún, el aislado **no puede escuchar hasta haber leído el consentimiento**, y eso es un `await` a
`storage`. Así que el saludo del principal se dispara contra un `window` sin oyentes y se pierde: el shim no
se instala, la página funciona como si Esfinge no estuviera y **no hay error en ningún sitio**. Dio la cara
como una prueba real que pasaba tres veces y a la cuarta no. Ahora cada lado anuncia y cada lado escucha, y
el que saluda **deja de tender en cuanto tiene acuse** —un puerto transferido por `window.postMessage` lo
puede recoger cualquier oyente, así que solo se manda mientras no existe el primer `<script>` del sitio—.

Y la mitad de método, que vale para cualquier protocolo entre dos piezas: **las cuatro pruebas que había
atendían antes de saludar**, o sea el caso que no ocurre. La que faltaba **fuerza el orden** y es la única
que lo caza siempre; la real solo lo caza cuando pierde la carrera. Cuando algo se sincroniza entre dos
piezas, hay que probar **las dos órdenes**, y probar la mala primero.

**Cuando un lado no puede comprobar su propio resultado, la comprobación se pasa al lado que sí puede.**
La prueba de crear una llave con la extensión de verdad comprobaba que firma, pero **no que el sitio pueda
verificar esa firma** — y en TypeScript no se puede, porque WebCrypto solo verifica en P1363 y WebAuthn
manda DER. Con eso, guardar en la bóveda una privada distinta de la que se le dice al sitio **pasaba en
verde**, y eso es una cuenta con una credencial registrada y sin forma de entrar. Se cerró con una cruzada
donde el núcleo crea, guarda y firma, y **Go verifica con la pública que fue al sitio**. Omitir la
comprobación porque «aquí no se puede» es dejar el agujero con una excusa.

**Y un procedimiento se ejecuta antes de escribirlo, también a escala de entrega.** La P2 —«usar una llave
que ya existe»— se planificó, se escribió y **se publicó** sin que nadie se preguntara de dónde iba a salir
esa llave: a mano no se pueden crear, el navegador no podía todavía y ningún gestor las exporta en su CSV.
Recorrer «probar la P2 en el Mac» paso a paso se para en el primero. Lo mismo que costó `ExportarLlaves`
—preguntar antes de preguntar—, una escala más arriba.

**Esperar a que la sincronización diga «al día» no es esperar a que llegue lo del otro equipo.** `alDia`
se conforma con una pasada anterior, y en una prueba entre dos equipos la del que recibe suele haber
terminado **antes** de que el otro subiera: entonces dice que no le ha llegado nada cuando lo que pasa es que
todavía no ha vuelto a mirar. Se espera **al contenido** —pidiendo pasadas hasta que el título aparece—, no
al estado. Costó creer que la bóveda compartida no se sincronizaba cuando se sincronizaba perfectamente.

**Y «pasó tres veces» no es «funciona».** Vale para el puente y volvió el mismo día con una prueba de frenos
del servidor: lo que hay que hacer con un intermitente es **apuntarlo con lo que se vio**, no volver a
correrlo hasta que salga verde y seguir. Un flake que se da por bueno es un fallo con una coartada.

Y la otra mitad, del 2026-10-01: **un intermitente conocido es la mejor tapadera que tiene un fallo nuevo.**
La prueba de las llaves con la extensión de verdad falló dos tandas completas seguidas con dos síntomas
distintos, y pasaba corriendo cada fichero por separado; la tentación era el flake de la víspera —el saludo
del puente, de la misma familia— y dentro había **dos fallos de producto**. Lo que los separó no fue volver a
correrlo, que pasó estando roto, sino **hacer que la prueba dijera en qué tramo se quedó**: al vencer el plazo
cuenta si el `shim` está instalado y qué dominios hay apuntados, y con eso cantó «instalado: true · dominios:
[…]», o sea que lo que faltaba era la bandera. Y un detalle logístico que costó la primera caída: el
`error-context.md` **se lo lleva la tanda siguiente**, así que lo que no se copie del registro se pierde.

**Una pregunta sin contestar no es una respuesta que diga «no».** `pedir` no lanza cuando el trabajador de
fondo no contesta: devuelve `ok: false` con un `error` y **sin `motivo`** —el motivo lo pone el núcleo, y ahí
no ha hablado nadie—. La bandera de las llaves empujaba eso como «aquí no hay nada» y **no lo corregía nunca**:
el `shim` cedía el resto de la vida de la pestaña y el banner no salía aunque hubiera llaves, sin error en
ninguna parte. Y pasa de verdad, porque el trabajador de MV3 se muere cada pocos minutos y la primera pregunta
de una página puede llegarle dormido. Está en `bandera.ts`, que **insiste solo cuando no se ha podido
preguntar**: una respuesta que dice que no hay llaves no se repite, que insistir sobre eso gastaría una
pregunta del freno en cada carga de cada sitio del mundo.

**Y un plazo es una decisión sobre el caso malo, así que la prueba tiene que pasarse de ese plazo.** El puente
se rendía a los dos segundos y lo que hay al otro lado es un `await` a `storage`: en una máquina cargada pasa
de eso, y entonces el anuncio del aislado llega cuando el otro lado ya ha quitado el oyente. La prueba que lo
vigilaba retrasaba al que atiende **150 ms** —dentro del plazo, o sea el caso bueno otra vez—. Ahora son quince
segundos, la prueba espera **más que el plazo viejo** y usa el de serie, así que bajarlo la pone en rojo.

Pero **ese plazo no era la causa del fallo que se estaba arreglando, y averiguarlo costó una pasada**: con los
quince segundos puestos, «el shim no se ha instalado» volvió igual. Lo que había era una carrera **en la
prueba**, que lo comprobaba justo después del `goto` cuando el `shim` no puede estar hasta que el puente tenga
acuse. De ahí las dos reglas: **con un intermitente, la pasada que importa es la de después de arreglar** —y
tres síntomas no son tres causas ni una sola—, y **esperar a que algo aparezca no es tapar nada** mientras
siga en rojo el caso de que no aparezca nunca.

**Y una prueba que llama en bucle a algo con freno se envenena sola, con un síntoma que no se parece a la
causa.** El otro síntoma de aquel intermitente —«el banner no salió»— era el **freno de sesenta preguntas por
minuto**: la pregunta que enciende la bandera pasaba y, ochenta milisegundos después, la de firmar volvía con
`motivo: "demasiado"`, se cedía y no salía nada. No lo gastaba esa prueba: lo habían gastado **las anteriores
de la tanda**, y las de las llaves van al final. Dos reglas: **en la compilación de pruebas los frenos van
holgados** —igual que los del servidor de cuentas en local, y el freno de verdad lo sigue midiendo la prueba
unitaria, que no pasa por Vite—, y **una prueba espera a que la otra pieza diga que está lista en vez de
insistir**, que además es lo que de verdad se quiere comprobar.

**Y cuando lo que falla está dentro de una pieza que no se puede mirar, se le pone bitácora a esa pieza.** El
diagnóstico desde la página llegaba a «el shim está instalado y los dominios están apuntados» y ahí se
acababa; lo que faltaba era **qué contestó el trabajador de fondo a cada pregunta**, y eso solo lo sabe él.
Veinte líneas en `storage.session`, **solo en la compilación de pruebas** (`__RAIZ_CUENTAS__` vacío en la que
se publica), sin nada de la bóveda dentro: el verbo, el sitio y lo que se contestó. El «demasiado» apareció a
la primera, y con la misma bitácora se cazó después un fallo que leyendo el código no salía. Esto **no** es la
consola de la extensión, que el cliente no quiere: es para que una prueba que falla pueda decir por qué.

**Un doble que apunta el argumento no dice si le han llamado.** `sistemaFalso.ElegirDondeGuardar`
guardaba el `desde` que recibía, y ese `desde` **viene vacío** hasta que alguien recuerda una carpeta: una
prueba que comprobara `desdeGuardar == ""` no distingue «no me han llamado» de «me han llamado sin
carpeta», y pasa en verde con el fallo dentro. Pasó al arreglar la exportación de llaves de acceso. Ahora el
doble **cuenta las veces**, y eso es lo que se mira.

**Y el fallo que lo destapó, que es de orden y no de código: preguntar antes de preguntar.** `ExportarLlaves`
abría el diálogo del sistema y descubría **al ir a escribir** que no había ninguna llave —que es como sale
la bóveda de fábrica—, así que pedía una clave y un sitio para un fichero que no iba a existir. Lo vio el
cliente en la 2.32.0 recién instalada. La regla: **lo que hace falta para decidir se mira antes de pedirle a
alguien que decida**, y la ventana lo dice de entrada en vez de dejar que se descubra al final.

**Y los booleanos los escribía todos encima de `papelera`**, que es el mismo fallo con el otro tipo: el
`case "si"` tenía el nombre del campo a fuego, de cuando la papelera era el único booleano del formato.
`oculta`, de la red wifi, iba a ser el segundo — y una red oculta habría aparecido **borrada**. Se arregló
**antes** de añadir el campo, que es la única diferencia con la vez anterior.

**Y lo que de verdad cierra esa familia de fallos: las dos listas de campos se comparan entre sí**
(`TestCruzadaLosCamposSonLosMismos`). Hasta la red wifi no existía, y por eso quitar un campo del espejo
**no ponía roja ninguna cruzada**: la forma canónica ordena las claves, el campo vuelve por `extra` con los
mismos bytes, y `TestCruzadaLoQueSeVacia` solo se entera **si ese campo era un secreto**. Con uno que no lo
sea, el espejo se quedaba corto en silencio hasta el día en que alguien añadiera uno sensible. Go saca su
lista por reflexión de las etiquetas JSON, así que ahora un campo nuevo que falte en `CAMPOS` salta sin que
nadie tenga que acordarse.

**El espejo de TypeScript leía todos los números como si fueran `revision`.** El caso `"numero"` de
`entradaDesde` tenía el nombre del campo escrito a fuego, de cuando la revisión era el único número del
formato. `algoritmo`, de la llave de acceso, es **el segundo número de la historia del formato**, y se
guardaba encima de la revisión: la misma bóveda fundía distinto en los dos lados. Llevaba ahí desde el
principio esperando a que hubiera un segundo número, y lo cazó `TestCruzadaFusionAlAzar` —no la forma
canónica, que solo compara lo que cada lado escribe—. La regla que deja: **en el espejo, un campo se escribe
por su nombre, nunca a mano**.

**Un campo que se quita de `Entrada` no da error: cae en `Extra` y no se ve nunca más.** `Extra` existe
para conservar lo que escribe una versión **más nueva**, y al quitar un campo hace justo lo contrario:
guarda callado lo que esta versión sí entendería. Pasó al partir la dirección del dato personal en sus
nueve trozos (ADR 0047): lo que escribió la 2.30.0 se trae a `calle` al leerlo y se deja de escribir. **Y
eso hay que hacerlo en los dos lados**, porque si uno lo trae y el otro lo deja en `extra`, la misma
entrada da bytes distintos y las dos bóvedas se la pasan sin fin.

**Y una sección nueva del cuerpo no se funde sola: `fundirContenido` arma el contenido campo a campo.**
Añadir un campo a `contenido` y nada más deja una sección que se guarda bien y **desaparece en la primera
sincronización**, sin error en ninguna parte. No es la trampa de `Extra` —que conserva lo que no se
entiende—: aquí se entiende y se tira, porque el `out` de la fusión se construye desde cero y lo que nadie
copie no existe. Pasó con `proyectos` (ADR 0050) y lo cazó una prueba de dos equipos, no las cruzadas. **Y
hay que hacerlo en los dos lados**: en el espejo, una sección que `Contenido` no declara cae en `extra` y
**se funde como un bloque**, o sea que gana la del servidor entera —así se perdería un proyecto creado en la
ventana, y así es como dos bóvedas se pasan la una a la otra sin fin—. Lo que hay que escribir, entonces, son
tres cosas: la fusión en Go, la fusión en TypeScript, y **datos de esa sección en el generador de
`TestCruzadaFusionAlAzar`**, que sin ellos los 400 casos no la tocan y la cruzada pasa sin comprobar nada.

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

**Una etiqueta `//go:build` que excluye un sistema es una promesa, y aquí no se ve si no se cumple.**
`internal/llavero` se partió por sistemas y al de Linux se le puso `!darwin && !windows` dando por hecho
que los otros dos llegarían enseguida: **los dos objetivos de Windows se quedaron sin `delSistema`** y
nada lo dijo, porque `go vet ./...` mira solo esta máquina y **la línea de comandos se cruza en `make
publicar`**, o sea en medio de una publicación. Con ello va una segunda trampa que no es la misma:
**un fichero de cgo no entra cuando `CGO_ENABLED=0`**, que es como se cruza esa línea de comandos, así
que un `_darwin.go` con cgo **no cubre darwin** y hace falta su pareja `darwin && !cgo`. Ahora
`make comprobar` compila los seis objetivos sin cgo, que tarda segundos; lo que eso **no** comprueba son
precisamente los ficheros de cgo, y ésos siguen siendo cosa del trabajo de macOS.

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

**La columna de contenido está topada en 560 px y no crece con la ventana**, así que una regla que
decida qué cabe mirando `@media (min-width: …)` está preguntando por lo que no manda. Los rótulos de las
pestañas de clases volvían así a partir de 740 px de ventana y funcionaba de milagro: cinco rótulos
medían 527 y cabían por doce píxeles. La sexta clase pide 160 más y **no cabe a ningún tamaño de
ventana**. Ahora el rótulo se queda solo en la pestaña activa, y hay una prueba que compara lo que la
barra necesita con lo que mide **con cada pestaña activa**, porque la activa es la única con rótulo y
mirando solo la primera se comprueba el mejor caso. Se vio en una captura; la causa la dijo **medir la
barra a siete anchos**, no razonar sobre el CSS.

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

**Y son seis ficheros, no cuatro: el quinto es `personalinfo.csv`** (ADR 0047), y trae tres cosas que
ningún otro. **Se declara fila a fila** —una columna `type` que dice si esa línea es un nombre, un correo
o una dirección—, así que es el único donde la clase de cada entrada no se puede sacar solo de sus
campos: **un nombre a secas no tiene ninguno que lo distinga**, y por eso `tipoDe` recibe la forma del
fichero. **`title` viene vacío en todas las filas** y el rótulo está en `item_name`, que además hay que
mandar `title` a las notas para que llegue: `Adivinar` se queda con la primera columna que reclama un
campo, y `title` va antes. Y **la dirección viene en nueve columnas en el orden de Dashlane**, que no es
el del sobre: por orden de columna sale «Calle Mayor 1, España, Madrid, Madrid, 28001».

**Y el sexto es `wifi.csv`** (ADR 0049), que es el más pequeño y **el que más miente**: su columna de
seguridad dice `unsecured` en redes que tienen contraseña, así que lo que traiga ahí **no se copia**, se
deduce de si hay clave —`internal/wifi.Normalizar`—. Creyéndole, el código QR sale marcado como red abierta
y **el móvil no se conecta**: el fallo parece del código y está en el dato. Dos cosas más de ese fichero:
la columna va mal escrita en origen —`encription_type`, sin la «y»— y `name` viene vacío en todas las filas,
así que el título sale del SSID.

**Y una huella que choca no impide que la entrada entre: la marca.** `Importar` cuenta `Metidas`,
`Repetidas` y `Conflictos`, y una huella común a todo cae en la tercera. Una prueba que compruebe
«entraron las seis y ninguna repetida» **pasa en verde con la huella rota** —comprobado mutándola—: lo
que hay que mirar es `Conflictos`, que es lo que el cliente vería como «6 duplicadas» al importar seis
cosas que no tienen nada que ver.

**Los avisos y las notas no pueden ser contenedores flexibles.** Lo fueron desde el principio, para
colocar el glifo delante, y con `display: flex` cada trozo del párrafo se convierte en un elemento
por su cuenta: **un `<strong>` en medio de una frase se sale a una columna aparte** y la frase se lee
en vertical, partida en pedazos. No se notó mientras todos los avisos fueron texto pelado, y se vio
mirando una captura, no en una prueba en verde. Lo que se quería es una sangría francesa
—`padding-left` más `text-indent` negativo—, y hay una prueba de interfaz que lo vigila.

**Todos los botones miden 28 px de alto, y dentro de uno alto lo que hay se encoge a cero.** La regla
general de `button` fija `height: 28px`, que es lo correcto para un control del sistema; la huella de
desbloquear es un botón en columna con un dibujo de 72 px dentro, y sin `height: auto` el dibujo —que es un
elemento flexible más, y `flex-shrink` vale 1 de fábrica— **salía midiendo 72 de ancho y 0 de alto**. Se
veía como un botón con su rótulo y sin dibujo, y **no lo dijo ninguna prueba**: se vio sacando la captura y
mirándola, y la causa la dijo preguntarle a la página por la caja del elemento, no razonar sobre el CSS.

**Y un `<svg>` sin dimensiones se estira hasta llenar la fila.** `Icono` devuelve un `svg` con `viewBox` y
sin ancho ni alto, así que lo pone el CSS — y cada sitio que lo usaba lo fijaba **con su propio selector**:
`.lateral .icono`, `.segmentado .icono`, `.lista-boveda .icono`… El día que se usó en una pantalla nueva sin
acordarse, el icono salió midiendo **doscientos píxeles** y el nombre que tenía al lado se encogió a cero, o
sea una fila con una carpeta gigante y sin texto. **No lo vio ninguna de las 110 pruebas de la interfaz**:
se vio sacando la captura, y la causa la dijo preguntarle a la página por la caja del elemento. Ahora
`.icono` mide 16 px mientras nadie diga otra cosa, así que lo peor que puede pasar es que salga pequeño.

Y el corolario de método, que costó una vuelta entera: **la captura que se mira tiene que ser la que se
acaba de sacar**. El fichero viejo seguía ahí y el recorrido que se lanzó no llegaba a esa pantalla —las
capturas estaban en otro `test` del mismo fichero—, así que el arreglo parecía no funcionar. Se borra la
carpeta antes, o se mira la hora del fichero.

Y con ello, el que costó buscar la causa en el sitio equivocado: **antes de estrenar una clase de CSS hay
que mirar si ya existe**. `.huella` ya era la huella de identidad que se enseña al compartir —un `<code>`
en monoespaciada— y la clase nueva del dibujo le estaba poniendo 72×72 **en todas las pantallas de
compartir**. Aquí conviven dos cosas que se llaman igual en español y no lo son: la huella criptográfica
(`huella`) y la dactilar (`dactilar`).

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

**Un mensaje impreciso no es impreciso: señala a otro sitio.** Las cuatro pantallas que comprueban la
contraseña maestra lo hacen abriendo la bóveda, y `Abrir` falla por más cosas que por la llave —un
fichero cortado, una bóveda de una versión más nueva, un JSON que esta versión no sabe leer—. Las cuatro
contestaban «esa no es la contraseña», y con la 2.30.0 eso mandó a buscar una tarde entera una contraseña
equivocada que era la correcta. Ahora pasan por `noAbre`: **la ventana sigue viendo siempre lo mismo**
—decirle a quien prueba contraseñas en qué ha fallado es ayudarle a acertar— y **lo que no es la llave se
registra**, que es donde puede mirarlo quien arregla. Las contraseñas fallidas de verdad **no** se
registran: un gestor de contraseñas con el registro lleno de intentos es justo lo que no hay que guardar.

**«Failed to load resource: 400» no deja ir a ninguna parte, y es todo lo que dice el navegador.** Dos
pruebas de la interfaz caían por eso —sin relación aparente entre ellas y solo corriendo los dos temas
seguidos—, y se persiguió una hipótesis equivocada antes de hacer lo que había que hacer: **que el vigilante
de consola apunte el método y el camino de cada respuesta de error**. Con eso cantó a la primera
`POST /api/Proyectos`, que era la ventana pidiendo una lista solo para traducir una referencia en un nombre,
a veces con la bóveda ya cerrada. Dos cosas que deja: **lo que la ventana necesita para dibujar va en el
estado que ya pide**, no en una llamada aparte que puede fallar; y el vigilante **no cuenta el 400 de
`AbrirBovedaConElSistema`**, que es una pregunta que puede decir que no y no un fallo.

**Y avisar a la ventana es que la ventana va a preguntar.** `cambiarBoveda` manda el evento del estado de la
bóveda, y la interfaz contesta pidiendo `EstadoBoveda`. Así que apuntar **después** lo que ese estado tiene
que decir —cuál es la bóveda activa y cómo se llama— deja a la ventana con lo de antes hasta el siguiente
aviso, que puede no llegar nunca. El estado se deja completo **y luego** se avisa.

**Dos clics seguidos sobre un botón de dos pasos caen en el mismo render, y el segundo no
confirma: vuelve a armar.** El idioma de la casa —«Borrar» → «Sí, a la papelera»— guarda el armado en
un estado de React, así que el manejador del segundo clic sigue siendo el de antes si no ha habido
pintado en medio. Una prueba que haga `click()` dos veces seguidas **no sale del proyecto** y sigue
como si nada. Entre los dos clics se espera **al rótulo nuevo**, que es la única señal de que el
botón ya está armado.

**Y un ayudante de pruebas que pregunta «¿hay algo?» justo al entrar en una sección pregunta
demasiado pronto.** `volverALaPersonal` miraba `.proyectos li[data-activo="si"]` nada más pulsar
«Proyectos» —y esa lista **se vuelve a pedir al entrar**—: salía cero, daba por hecho que ya estaba
en la personal y **se iba dejando el proyecto abierto**, que es justo lo que ese ayudante existe para
impedir. Lo que no tiene esa carrera es lo que ya está en la ventana: el botón de salir de la barra,
que sale del estado que la interfaz tenía desde que se entró.

Con ello, lo que de verdad cerró esa familia: **desde que el `title` de la barra lateral dice *cuál*
es la bóveda abierta, una fuga entre pruebas se cae sola** en «la barra lateral dice si la bóveda
está abierta o cerrada», y además **señala a la culpable por su nombre** —«La bóveda de «Beta …» está
abierta»—. La fuga llevaba ahí desde antes y no la veía nadie: lo único que pasaba era que las
pruebas del segundo tema corrían dentro de otra bóveda sin que ninguna aserción lo mirara. Si esa
línea se pone roja, **lo que hay que arreglar no es la aserción**.

**Y una prueba que conmuta de bóveda tiene que dejar abierta la de siempre** (ADR 0050). El Go es uno y la
bóveda sigue abierta entre pruebas **y entre temas**: una prueba que acabe dentro de una bóveda de proyecto
deja a las siguientes mirando otra, y lo que se ve entonces son **cuatrocientos en la consola y dos pruebas
de la lista cayendo sin relación aparente** —y solo en el segundo tema, porque el primero las deja así—.
Pasa aislada, pasa el fichero entero en un solo tema, y solo cae con los dos. Es la misma familia que lo de
abajo.

**Y un `test.skip` de un `describe` no cubre lo que se escriba debajo del cierre.** Una prueba de capturas
escrita fuera del bloque corría siempre, y además **la primera de toda la tanda** —`capturas.spec.ts` va
antes que `esfinge.spec.ts` por orden alfabético—: creaba la bóveda del 5173 con *su* contraseña maestra.
Como Go deja la bóveda abierta entre pruebas, cuarenta y cinco no se enteraban y **solo caían las dos que
vuelven a teclear la maestra**. Tiró una publicación. Al añadir una prueba a un fichero, mirar dónde cierra
el bloque, no dónde acaba el fichero.

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

**Un correo no es una página, y el botón de la invitación lo enseñó.** Se escribió como un `<a>` con
`background:` abreviado —lo que funciona en cualquier navegador— y **Gmail no lo pintó**: se quedó en un
enlace suelto. Lo vio el cliente en su buzón; aquí no lo decía nada, porque el único sitio donde se puede
ver un correo es un cliente de correo. Dos reglas que salen de ahí y valen para cualquier correo que se
añada: **el color de un botón vive en una celda con `bgcolor`**, que es un atributo de HTML y no hay cliente
que lo tire —y además Outlook de escritorio compone con Word e ignora el relleno de un enlace—; y **nada de
forma abreviada en CSS**, que Gmail tira `background:` y respeta `background-color:`. Lo vigila
`TestElCorreoDeInvitacionSeLee`, que lee el propio `correo.ts`.

Y con ello, lo que el correo **no** puede dar por hecho: **las imágenes**. Los correos llevan el icono
porque lo pidió el cliente sabiendo lo que cuesta —pedirla le dice a quien la sirve que el correo se ha
abierto, con hora e IP—, pero **el nombre va escrito al lado**: con las imágenes bloqueadas, que es como
llegan de entrada a casi todo el mundo, la cabecera se sigue leyendo.

**Un diálogo del sistema que pide una contraseña sin avisar es lo que un gestor de contraseñas enseña a
desconfiar.** Al guardar la llave de Touch ID, macOS pide **la contraseña del Mac** para autorizar a Esfinge
a usar el llavero —es lo normal en una aplicación sin firmar— y con «Permitir siempre» no vuelve a
preguntarlo. Se avisa **antes** de que salga y se dice qué botón pulsar, en los tres sitios donde se activa,
y **solo en macOS**: en Windows la credencial de Hello no pide nada y en Linux no hay nada que pedir. Sin
ese aviso, lo sensato por parte de quien lo ve es cancelar, y cancelar deja la activación a medias.

Y **una vez por actualización vuelve a preguntar**, ahora al **abrir** con la huella: el permiso va atado al
binario y sin firmar cada versión es otro (ADR 0044, decisión 3). Se avisa en la pantalla de desbloquear, y
**solo entre una actualización y el primer desbloqueo que funcione**: para eso las preferencias guardan qué
versión consiguió abrir ya, que no es lo mismo que qué versión corre. Decirlo siempre sería mentir casi
siempre.

Y con ello, una del método y no de macOS: **«no preguntó» tras hacerlo una vez no es «no pregunta»**. Se dio
por comprobado que activar no pedía nada, y era la primera activación: crear un elemento en el llavero no
pide permiso, **acceder a uno que ya está, sí**. Lo mismo vale para cualquier cosa que solo se haya visto
funcionar una vez.

**Un `catch` vacío en algo que ha pedido una persona es un fallo, no una precaución.** La casilla «Abrir
con Touch ID a partir de ahora» activaba entre abrir la bóveda y entrar, y se tragaba el error con el
argumento de que esa pantalla desaparece de todos modos. El cliente marcó la casilla, entró, y **la tarjeta
de dentro le ofreció otra vez lo que acababa de pedir**: ni él sabía por qué, ni aquí se pudo reproducir
—el llavero de mentira no falla— ni leyendo el código salió la causa. Eso es lo que cuesta callar un error.
La regla: **lo que alguien pide se hace donde haya pantalla para contar que no se ha podido**, y si el sitio
natural va a desaparecer, se mueve la acción, no se esconde el fallo.

**Y desde que los correos son HTML, lo que se interpola en ellos es una inyección.** El nombre de un equipo
son ochenta caracteres libres que manda el cliente, y en texto pelado eso daba igual; en un correo con
formato, `<img src=x onerror=…>` es una etiqueta. Todo lo que entra en un correo pasa por `escapar`, y hay
una prueba que lo mete de verdad y comprueba que sale escrito y no ejecutado. **El asunto no se escapa**, y
tampoco debe: ahí no hay HTML.

Y dos reglas de los correos de código, que no son de estilo: **el código va en texto**, grande y
seleccionable, nunca en una imagen —si no, no se puede copiar ni leer con un lector de pantalla—; y
**ninguno lleva un enlace que entre por ti**. Un correo de un gestor de contraseñas que diga «pulsa aquí
para entrar» es entrenar a su gente para el phishing. Lo vigila una prueba del servidor.

**Y una prueba no puede dar por hecho el valor de fábrica de una preferencia, porque otra las guarda a
medias a propósito.** La prueba del interruptor de actualizaciones manda `{"buscarActualizaciones":true}`
para ejercitar la regla del cero, y eso llega con **todos los demás booleanos en `false`**: apaga las llaves
de acceso en el navegador. Luego, al hacer `check()`, React vuelve a mandar las preferencias enteras **con
lo que tenga en memoria**, así que si las había leído antes del guardado a medias las restaura sin querer y
si las lee después las manda apagadas. **Qué lectura gana la carrera lo decide lo rápida que sea la
máquina**: la prueba de las llaves pasaba en verde aquí y caía en los dos temas en GitHub, y **tiró una
publicación**. La regla: si una prueba mira el valor de una preferencia, **se lo pone ella primero**, leyendo
y guardando el objeto entero, que es lo que hace la ventana.

Y el síntoma no se parece a la causa: Playwright dice `Received: unchecked` enseñando un
`<input checked type="checkbox"/>`. No es una contradicción — React deja el atributo del primer render, con
las preferencias todavía sin llegar, y al llegar cambia solo la propiedad.

**Una casilla cuyo estado viene de Go no se marca con `check`.** Las de Ajustes se desactivan, le piden
el cambio a Go y se vuelven a dibujar con lo que Go conteste —que es lo correcto: marcarla antes de saber si
ha funcionado es prometer—. Playwright's `check` exige que el estado cambie **en el mismo clic** y falla con
«clicking the checkbox did not change its state», que no se parece en nada a la causa. Se mueven con
`click()` y se espera con `toBeChecked()`.

Y con ello, la trampa de las preferencias otra vez, que vuelve en cuanto algo se lee más de una vez: **una
lectura pedida antes de un cambio puede llegar después**. Con un interruptor se ve peor que con un
desplegable, porque el siguiente clic parte del estado viejo y deshace lo que se acababa de hacer. El
remedio es el mismo, `cambiosHechos`, y hay que ponerlo desde el principio.

**Una fecha del formato se escribe con `rfc3339(ahora())`, nunca con `toISOString()`.** El formato tiene
resolución de un segundo y el núcleo de TypeScript tiene su `rfc3339()` y su reloj parable desde el principio;
aun así `confirmada` se escribió con `new Date().toISOString()` y entró **con milisegundos**, o sea algo que Go
no escribiría nunca para el mismo instante. **Las cruzadas no lo cazan**, porque comparan entradas generadas
con el mismo valor en los dos lados: lo caza una aserción del formato en la prueba de comportamiento. Y ojo con
el nombre: en `fuente.ts`, `ahora` ya es el instante en milisegundos de los frenos, así que el reloj del núcleo
entra con alias.

**Y lo que no se nota tiene que no notarse por los dos lados: ni en la respuesta ni en lo que tarda.** El
servidor contesta lo mismo tenga cuenta o no la dirección a la que mandas —es media protección contra la
enumeración de correos—, y la B3 le añadió un camino entero por debajo: mandar la invitación. Ese camino
**no puede lanzar nada** —ni pasarse del tope de invitaciones, ni un fallo del correo, ni su cupo agotado—
porque cualquiera de esas cosas sería un `429` o un `503` que solo sale cuando esa dirección **no** tiene
cuenta. Y el correo va en `ctx.waitUntil`, porque esperar a que salga son cientos de milisegundos —más
desde la ADR 0045, que le añade un salto— y eso se
mide desde fuera igual de bien que un código de estado. **La regla:** cuando dos caminos tienen que parecer
uno, el que sobra se traga sus errores y no alarga la respuesta.

**Y el correo no lo entrega el Worker, por mucho que pueda** (ADR 0045). `connect()` de
`cloudflare:sockets` abre TCP saliente y el 465 funciona —la ADR 0041 decía lo contrario y era falso—,
pero **Google rechaza la autenticación SMTP cuando la conexión sale de Cloudflare**: contesta
`535 5.7.8 … p=BadCredentials` a las mismas credenciales que acepta desde un Mac. Se descartó todo lo
demás midiendo, y eso es lo que no hay que repetir: **la misma huella SHA-256 de la clave en los dos
sitios**, `AUTH LOGIN` fallando igual tras aceptar el usuario, `smtp.gmail.com` fallando igual y **ninguna
alerta de seguridad** —que es lo que acompañaría a un bloqueo por ubicación—. Así que el Worker compone
(`servidor/src/mensaje.ts`, con la maqueta y sus pruebas) y entrega `cartero/`, en el VPS, cuya IPv4 sí
está autorizada en el relé. Dos cosas que arrastra: **el VPS entra en el camino crítico del alta**, y el
cartero **revisa lo que le llega aunque venga firmado** —el remitente tiene que ser la cabecera nuestra
entera, no solo la dirección, o pasaría `Banco Santander <esfinge@webcafeina.com>`—.

**Y `wrangler deploy` borra las variables de texto puestas a mano en el panel de Cloudflare.** Los
secretos sobreviven; las variables no. Poner una en el panel y desplegar después para que el código la
lea es borrarla justo antes de usarla, y el síntoma es que no pasa nada. Van en `wrangler.jsonc`, o se
ponen **como Secret** si tienen que sobrevivir a un despliegue aunque no sean sensibles.

**Y todo lo que llega pegado se recorta, no solo lo que parece un secreto.** Una contraseña con un
espacio de más da un error; **un interruptor con un espacio de más da la apariencia de funcionar**, que
es peor. `CORREO=" enviar"` dejaba el Worker contestando `202` con el correo cayendo en el buzón de
pruebas, y se descubrió mirando el registro de Caddy de **otra máquina** para ver que la petición al
cartero no estaba. De ahí las dos reglas: `.trim()` en todo lo que venga de una variable de entorno, y
**que el propio servicio diga en qué modo está** —`/v1/salud` del Worker de pruebas dice por dónde
manda el correo— en vez de dejar que haya que deducirlo.

**Y la pimienta del servidor se puede rotar, pero una cuenta no migra entera al entrar** (ADR 0046). Los
verificadores llevan su versión dentro (`2:abc…`) porque **son dos y no migran a la vez**: el de acceso
se reescribe al entrar, y el de posesión **solo se comprueba al cambiar la maestra y al recuperar**, así
que hace falta una llamada de cortesía al arrancar la sincronización. Dos cosas que costó encontrar, las
dos al ir a rotar de verdad y no al escribirlo: que «migra en cada pasada de la sincronización» era falso
—y con ello el contador de D1 no habría llegado a cero nunca—, y que **mandarla al entrar tampoco vale**,
porque la posesión sale de la clave de la bóveda y al entrar la bóveda no está abierta.

**Para poder recibir hay que haber publicado, y quien nunca manda no publicaba nunca.** Las llaves de la
identidad se publicaban al entrar en «Compartir», que es lo que parece natural: se publican cuando se van a
usar. Pero para **recibir** una copia hacen falta las llaves de quien la recibe, y quien no ha mandado nada
en su vida no había entrado ahí jamás. Entonces el servidor le da a quien manda **unas llaves inventadas
pero fijas** —que es a propósito, así es como no dice quién tiene cuenta— y el sobre llega cifrado hacia
nadie: los dos lados hacen lo suyo bien y el buzón enseña «No se puede abrir» sin que ninguno de los dos
pueda entender por qué. Ahora las publica la ventana al arrancar la sincronización y la extensión en cada
pasada, de cortesía las dos. **La regla general:** cuando algo hay que publicarlo para que *otro* pueda
usarlo, el momento de publicarlo no lo marca lo que hace su dueño.

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

**Dos entradas de la misma clase pueden ser idénticas en pantalla, y en las llaves de acceso eso duele.**
El título de una llave es el nombre del sitio y la segunda línea de la lista es la cuenta, así que **cuatro
llaves de GitHub de la misma persona se ven como cuatro filas iguales**. En una contraseña es molesto; en una
llave no se puede recrear ni corregir, así que borrar la que no es cuesta perder la forma de entrar. Lo único
que las separa es **cuándo se crearon**, y va en la lista y en la ficha —no solo en el editor: el dato que
distingue tiene que estar donde se mira primero—. **Con segundos**: las del cliente se crearon en pocos
minutos y varias en el mismo, así que hasta el minuto seguían saliendo idénticas.

Y el porqué de que hubiera cuatro, que es la otra mitad: **crear guarda antes de entregarle la credencial al
sitio**, a propósito —lo contrario deja al sitio con una llave que aquí no existe—, así que un registro que
falla después deja una huérfana **y el navegador no puede enterarse**. Está en `deuda.md`.

**Un objeto que se devuelve a la página tiene que imitar al de verdad en mutabilidad, no solo en forma.**
Los métodos de la credencial —`toJSON`, `getClientExtensionResults`— se pusieron con `defineProperty` sin
`writable`, por reflejo defensivo. En un `PublicKeyCredential` de verdad viven en el **prototipo**, así que
`cred.toJSON = …` crea una propiedad propia y funciona; en el nuestro son de la instancia y **la asignación
lanza**. GitHub usa `@github/webauthn-json`, cuyo ponyfill hace exactamente eso, y registrar una llave moría
con «Cannot assign to read only property 'toJSON'» **después** de haberla creado y guardado: la llave
quedaba en la bóveda y el sitio no la registraba. Ahora los **métodos** van `writable` y `configurable`, y
los **datos** solo `configurable`, que es lo que hace el real —allí son captadores sin asignador—. Blindar
ahí no protegía nada: el objeto es nuestro y se lo estamos dando.

Y estaba en **las dos** credenciales, la de firmar y la de crear. La de firmar no lo había enseñado solo
porque no había ninguna llave con la que entrar, así que **el fallo llevaba una entrega entera esperando**.

Con ello, una de método: **una prueba que compruebe que algo no se puede asignar no vale dentro de
`page.evaluate`**, que no corre en modo estricto: asignar a una propiedad no escribible **falla en silencio**
y el `catch` no se ejecuta nunca. La primera versión pasaba diciendo lo contrario de lo que quería decir. Se
mira el **descriptor**.

**`extensions` viene vacío al entrar y lleno al crear, y eso costó la primera prueba en su Mac.** La regla
de la P2 —«cualquier clave dentro de `extensions`, se cede»— se escribió con el diagnóstico de
`navigator.credentials.get`, donde GitHub lo manda **presente y vacío**. Al **crear** manda
`appidExclude` y `credProps`, así que el `shim` cedía siempre: el cliente pulsó «Add passkey» y salió el
diálogo del navegador con Dashlane, el llavero de Apple y Chrome, **y sin Esfinge**. Ahora hay lista blanca
—`credProps` se contesta `rk: true`, que es verdad porque todas las llaves de Esfinge son residentes;
`appidExclude` se ignora porque una llave nuestra nunca es una credencial U2F heredada— y **ante cualquier
otra se sigue cediendo**, con su prueba para que arreglar esto no se convierta en abrir la puerta.

La lección no es de WebAuthn: **un diagnóstico de un verbo no vale para el verbo de al lado.** La forma de
`get` se pidió por la consola y se usó para decidir también lo de `create`, que nadie había mirado. Y lo que
lo destapó fue otra vez la consola del cliente, no una prueba ni la lectura del código.

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

**La ficha de Chrome se pega a mano, y describir una consola que no se ve sale mal tres veces de tres.**
Firefox recibe su ficha con cada etiqueta (`amo-metadata.json`); **la API de la Chrome Web Store sube el
paquete y no edita la ficha**, así que cambiar `docs/tiendas/ficha.md` no cambia lo que la tienda enseña y
**nada avisa**: lo que la entrega de compartir cambió el 2026-09-24 se quedó seis días sin pegar, y lo vio
el cliente buscando una línea que se daba por puesta. Ahora los bloques que se pegan van marcados en esa
ficha y `ficha-de-chrome.mjs` guarda su huella, así que olvidarse **para** `pnpm run comprobar`. Se hashea
solo lo marcado: un aviso que salta al arreglar una coma de una nota interna se deja de leer.

Y una tercera, del 2026-10-01: **los campos de esa consola tienen topes de caracteres, y desde aquí no se
ven**. La justificación de los permisos de host admite **mil** y el texto de la 2.34.0 llegó a 1197: se supo
**al ir a pegarlo**, que es el peor sitio donde enterarse. Ahora el tope va **en la marca**
—`<!-- consola de Chrome: empieza (máximo 1000) -->`— y `ficha-de-chrome.mjs` lo mide, así que pasarse
**para** la comprobación. Un bloque sin tope no se mide: no sabemos cuál es, y adivinarlo sería peor que no
comprobar nada.

Y dos reglas de esa ficha que costaron el mismo día, las dos por escribir para una pantalla que desde aquí
no se puede abrir: **dentro de las marcas no va markdown** —la consola no lo interpreta, y si aquí hay
negritas y allí prosa la huella vigila un texto que no es el de la tienda—, y **solo se marca lo que tiene
un campo donde pegarlo**. Esto último apareció tres veces en una tarde: las casillas de uso de datos no
tienen descripción, «Código remoto» pierde la explicación al contestar «no», y la viñeta que faltaba. Qué
campos existen **lo dice quien mira la consola**, no lo que parezca razonable desde aquí.

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

**`go test` sin `-count=1` da verdes de pruebas que no ha ejecutado, y así `make comprobar` tapó una que se
había puesto roja sola.** La primera orden de la puerta era `go test ./...` a secas, así que Go reutilizaba
resultados en caché: `ok internal/boveda (cached)`. Las otras dos tandas de ese objetivo filtran —una con
`-run Cruzada` y otra con `-race` sobre tres paquetes—, de modo que **esa prueba no corría en ninguna**. Quien
la cazó fue la puerta de la publicación, con caché frío, **después de empujar la etiqueta**: los seis trabajos
en «skipped» y ninguna *release*. La puerta hizo su trabajo; la de aquí mentía. Ahora lleva `-count=1`, que
tarda unos minutos más y es lo que hace que un verde signifique algo.

**Y lo que tapaba: un dato congelado leído con el reloj de verdad caduca solo.** `boveda-2.22.2.esfinge` es
un vector fijo —como los del formato `ESF1`, no se regenera— y lleva dentro **una entrada en la papelera
borrada el 2026-09-18**. Abrir una bóveda **purga lo que pasó de `PlazoPapelera`**, treinta días, así que el
**2026-10-18** —un domingo, sin que nadie tocara nada— la entrada dejó de estar ahí y la prueba se puso roja.
Lo que cede es el reloj, no el fichero: ahora se lee con `conReloj` fijado al día siguiente de lo último que
se escribió dentro. **La regla, que vale para cualquier vector fijo con fechas**: si una purga por antigüedad
corre contra él con el reloj del sistema, la prueba tiene fecha de caducidad y no está escrita en ninguna
parte. Las de `cripto` y `qr` no la tienen —son bytes y matrices, sin plazos—, y se comprobó antes de darlo
por cerrado.

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
- **Está desplegado desde el 2026-09-18**: producción en `https://esfinge-cuentas.webcafeina.com`, y **el
  registro está abierto desde el 2026-09-23** (entrega A4), una vez publicadas la política y las condiciones
  y hecho lo que prometen. La tabla `admision` sigue ahí sin usarse, por si hay que volver a cerrarlo; el
  modo está en `REGISTRO` de `wrangler.jsonc` y lo dice `/v1/salud`. Desplegar es lanzar a mano
  `servidor.yml`; los secretos los pone el cliente en el panel y **la `PIMIENTA` de producción la guarda
  él**.
- **pnpm frena las versiones recién publicadas** y, si se le deja, las mete solo en
  `minimumReleaseAgeExclude`. No se acepta: se fijan versiones con una semana, que es lo que ese freno
  pide.
- **Y el Worker de pruebas desplegado tiene el buzón detrás de Cloudflare Access.** `/v1/*` contesta, pero
  `/_pruebas/buzon` devuelve un 302 a la pantalla de Access y el cuerpo es HTML: el síntoma es un
  «invalid character '<'» al descodificar el JSON. El código de un alta contra ese Worker se saca de la
  tabla `buzon_pruebas` de su D1 (`esfinge-cuentas-pruebas`), o lo abre el cliente en el navegador, que es
  quien pasa la puerta. Solo pasa con el desplegado: el que levanta `con-servidor.sh` en local no tiene
  Access delante.
- **Y ahí hay dos códigos de seis cifras que no son el mismo, y confundirlos costó tres rondas.** El que le
  **llega al correo** al cliente es el de **Cloudflare Access** —el portero— y se teclea **en la página del
  navegador**; el de Esfinge **no se manda a ninguna parte**, porque el Worker de pruebas escribe en un
  buzón de mentira que solo se ve ya pasada esa puerta. El servidor contesta a los dos lo mismo —«El código
  no es correcto o ha caducado»—, así que **el error no dice que sean dos**: hay que decirlo al pedirlos.
  Dos detalles más de ese buzón: **acumula mensajes y solo vale el más reciente** —lo más limpio es estrenar
  dirección cada vez— y hay **tres altas por IP y día**.
- **Y la del método, que es la que de verdad costó**: ante ese error me inventé dos causas seguidas —cifras
  dentro de la dirección de correo, mensajes acumulados—, las dos plausibles, las dos bien razonadas y las
  dos falsas. Lo resolvió **preguntarle a quien lo estaba haciendo qué veía**. Cuando algo falla en un
  tramo que no se puede mirar desde aquí, la primera pregunta no es «qué puede estar mal» sino **«qué estás
  viendo»**.

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

**La forma canónica no caza que un campo se caiga del espejo de TypeScript**, y eso costó creer que
estaba cubierto. Ordena las claves, así que un campo que no esté en `CAMPOS` vuelve por `extra` y **los
bytes salen idénticos**. Lo que se rompe es otra cosa: `sinSecretos` lo borra por su nombre, ya no está
ahí, y **el secreto cruza al panel con todo en verde**. Lo vigila `TestCruzadaLoQueSeVacia`, que compara
lo vaciado y no lo escrito, y que era un hueco abierto para las cuatro clases anteriores.

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

Y **la clase de datos personales** (ADR 0047), vista con la 2.31.0: las pestañas de clases sin rótulo
salvo la activa, el glifo de la silueta al lado del carné de las identidades y el formulario de catorce
campos. Las tres se le señalaron antes de que las viera, que es la diferencia entre que las mire y que
las descubra.

Y **el canal con los agentes, usado de verdad** (2026-10-13), que es lo que lo ha pulido: con la 2.52.0,
que el agente **pregunte antes de escribir** cuando lo abierto es la bóveda de un proyecto y nadie la ha
nombrado; que **«No volver a preguntar» no cubra el código de un solo uso**; y que `esfinge_resumen`
conteste de un tiro. Y con la 2.53.0, lo que salió de ahí: que **la tarjeta de aprobación se vea en una
pantalla de trabajo** —«ha salido el aviso en la bóveda»—, que es lo que hasta ese día no pasaba.

Y el 2026-10-14 las dos que quedaban. **La tarjeta de emparejar un navegador** (2.54.0), retirando antes el
permiso en Ajustes → Navegador y pidiéndolo con Esfinge en la bóveda: sale arriba y se contesta ahí. Y **la
versión por el canal** (2.55.0), que además confirmó el relevo por otro camino: el agente enumeró **las once
herramientas**, que son exactamente las once del fuente — con el relevo fallando habría salido la lista
corta del paquete que se instaló en su día.

Y el 2026-10-19, **la última ficha que le quedaba a la ADR 0055**: con la bóveda de un proyecto abierta, le
pidió crear una credencial **sin nombrar bóveda** y el agente **preguntó antes de escribir**, ofreciéndole las
tres que tiene en el equipo. Es la mitad de esa ADR que no vive en el código sino en las descripciones, y la
pregunta salió **en los controles nativos de Claude Desktop** — o sea que una frase del esquema se convirtió
en una pregunta de verdad y no en un párrafo que nadie lee. De paso preguntó el alfabeto sin que se lo
pidieran, que es lo que se gana al decir en el esquema lo que hay.

Y el 2026-10-20, **el canal en Claude Code** —hasta entonces solo se había usado Claude Desktop—: la ruta que
calcula la aplicación, la versión, las trece herramientas, el nombre «Claude Code (claude-code)» en la tarjeta
y el camino de aprobación entero. De ahí salieron los dos fallos de la 2.58.7, y **el arreglo del vocabulario
quedó comprobado por los dos lados el mismo día**: con la bóveda bloqueada el agente dice «cerrada, o sea
bloqueada y a la espera de la contraseña maestra» —el sentido verdadero, y lo explica— y con ella abierta dice
«Personal es **la activa**; para usar las otras basta con que me la nombres, Esfinge cambia sola **sin pedir
contraseña**». Las dos mitades, que es lo que no se podía dar por hecho: que la palabra siguiera sirviendo
para el caso en que sí está bloqueada.

Sin verificar todavía, y es lo único que queda en todo el proyecto: **cómo quedan las estructuras de
Windows y de GNOME** en máquinas de verdad, y ahí mismo el icono de los `.esf`, que en Windows lo
pone el instalador y en Linux el `.deb`.
