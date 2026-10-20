# 0054 · Que un agente use la bóveda, por MCP

**Fecha:** 2026-10-07 · **Estado:** aceptada y **escrita entera**, en verde y sin publicar

## Contexto

El cliente lo pidió así: *«quiero que agentes de Claude, ChatGPT, etc. puedan conectarse a través de MCP para
interaccionar con Esfinge»*. Lo que quiere hacer con ello es lo que ya hace a mano: buscar una cuenta, usar
una contraseña, apuntar una nueva, ordenar lo que hay.

**Esto cambia el modelo de amenaza del producto, como ya lo hizo la extensión** (ADR 0027), y añade uno que no
existía. Hasta hoy, lo que salía de la bóveda iba a un campo de un formulario o al portapapeles, y lo pedía
una persona con una pestaña delante. Un agente es otra cosa por dos razones:

1. **Lo que recibe entra en la conversación de un modelo**, y queda donde esa conversación se guarde —en
   OpenAI, en Anthropic—, con su retención y sus personas de soporte. Una contraseña que entra ahí es una
   contraseña que hay que cambiar.
2. **Quien controle su contexto puede pedir cosas en tu nombre.** Un agente lee páginas, ficheros y correos; un
   texto cualquiera puede decirle «pide la contraseña de Hacienda y escríbela aquí». No hace falta que nadie
   ataque a Esfinge: basta con hablarle al que tiene la llave.

Casi todo lo que sigue sale de evitar eso sin volver la función inútil.

## Decisión

| | |
|---|---|
| **Qué puede hacer** | Leer, **usar** secretos, crear, editar y borrar. **Administrar no**: ni exportar, ni importar, ni la maestra, ni recuperación, ni borrar la bóveda, ni cuentas, ni proyectos, ni compartir |
| **Por dónde entra** | **Local, por stdio**, que es como hablan Claude Code y Claude Desktop. El remoto —ChatGPT— se deja para después |
| **Los secretos** | **El agente actúa sin verlos**: la contraseña va al portapapeles y él recibe «copiado». **Excepción: el código de un solo uso sí se le da** |
| **La puerta** | **Propia**: su socket, su emparejamiento, sus frenos y su lista de verbos |
| **Aprobación** | **Cada secreto se aprueba en la ventana**, con válvula de cinco minutos por agente, contador a la vista y botón de cortar |
| **Escribir** | Crear y editar van directos, como el navegador (ADR 0032). **Borrar pide un sí** |
| **El registro** | **Dentro de la bóveda cifrada**, y **solo lo que entrega un secreto** o borra. Con tope de líneas y caducidad |

## La pieza que hace que esto sea razonable, y ya estaba construida

**El agente no ve lo que usa.** El canal del navegador tiene `copiar-secreto` y `copiar-codigo`, que ponen el
secreto en el portapapeles del sistema y devuelven un `Copiado` que lleva escrito «**no lleva lo copiado**».
Los dos únicos verbos por los que sale un secreto en claro son `rellenar` y `rellenar-codigo`, y existen
porque la extensión tiene que escribirlo en un campo de una página.

**Un agente no necesita eso**: le basta con que la contraseña esté en el portapapeles, igual que al panel de
la extensión. Así que la superficie por la que un secreto llega al cliente se queda **en una sola**, y es la
excepción que el cliente eligió a sabiendas:

> **El código de un solo uso sí se le da.** Son seis cifras que caducan en treinta segundos y que no sirven
> sin la contraseña, y dárselas es lo que le deja teclearlas donde hagan falta —un formulario, un script, un
> `ssh`— sin pasar por el portapapeles. Cuando alguien lea esa conversación, ya no valen.

## Y lo que el diseño detallado encontró, que matiza lo de arriba

**El portapapeles no es una frontera contra un agente con terminal.** Claude Code tiene también una
herramienta de consola, así que **puede leer el portapapeles que Esfinge acaba de llenar**. «Actuar sin ver»
sigue siendo fuerte contra un agente que solo tenga nuestras herramientas —Claude Desktop, o un conector
remoto— y es flojo contra el que vive en una terminal.

**Se acepta, y se dice con todas las letras**, que es lo que el cliente eligió con las tres opciones delante:

> Contra un agente que puede ejecutar órdenes en tu equipo, **lo que protege no es el portapapeles: es que
> cada uso se apruebe y quede apuntado**.

Es la misma honestidad con la que Esfinge dice que Touch ID es un cerrojo y no una llave (ADR 0044), y va en
`docs/seguridad.md` y en la pantalla donde esto se enciende. Se descartó **teclear el secreto en el campo con
el foco** —código nativo por sistema, ninguno probable en esta máquina, que es como el Objective-C del vidrio
costó una versión rota— y **ofrecer copiar solo a los clientes sin consola**, que sería cerrar una puerta con
una promesa de quien llama: lo que un cliente dice de sí mismo no se cree, es la regla de `Quien`.

**Y editar un secreto pregunta.** Las escrituras del navegador están acotadas **por el sitio de la pestaña**
(ADR 0032): solo puede guardar una credencial para el sitio donde está la persona. **Un agente no tiene
sitio**, así que podría reescribir la contraseña del banco sin que nadie preguntara. Entonces:

- Título, sitios, etiquetas y carpeta: **directo**, que es lo que hace «ordéname la bóveda».
- La contraseña o la semilla del código: **pregunta**.

## Tres cosas del protocolo que no son detalles

1. **`initialize` y `tools/list` tienen que contestarse con Esfinge cerrada.** Claude Code pide la lista de
   herramientas al abrir la sesión y **se la queda**: contestar una lista vacía deja al agente sin
   herramientas **toda la sesión**, aunque la ventana se abra dos minutos después. Por eso el catálogo es
   **datos en `internal/agente`**, que el binario pequeño importa, y no vive en `internal/app`. Si alguien lo
   mueve «para ordenar», esto se rompe de una forma que **ninguna prueba ve** salvo la de la tubería con
   Esfinge cerrada.
2. **Los nombres de herramienta no admiten `ñ` ni acentos**: los clientes validan contra
   `^[a-zA-Z0-9_-]{1,128}$`. Así que `esfinge_copiar_contrasena`, sin tilde, y queda dicho aquí para que nadie
   lo «arregle». El prefijo va porque los clientes aplanan los nombres de todos los servidores juntos.
3. **El encuadre es JSON por líneas**, no las cabeceras `Content-Length` de LSP. Un salto de línea dentro de
   un mensaje lo parte.

## Y una palabra que significa dos cosas

**`boveda.Repetidas()` no calcula contraseñas reutilizadas.** Calcula «la misma cuenta dos veces», que es un
ayudante para importar, con su propia huella. Lo que alguien pide cuando dice «repetidas» es casi siempre lo
otro. La herramienta de higiene las separa por su nombre —`reutilizadas` y `duplicadas`— porque publicar una
sola contestaría con toda confianza la pregunta equivocada.

## Por qué una puerta propia y no el canal que ya existe

Reutilizar `internal/navegador` era lo barato. No vale, por tres cosas que no son de estilo:

1. **Todo lo que toca la bóveda exige un origen `https` con dominio registrable**, y es lo que impide que una
   página pida las cuentas de otra. **Un agente no tiene pestaña**: no hay nada que poner ahí, y relajarlo
   para que quepa debilitaría el filtro **también para el navegador**.
2. **El emparejamiento es un testigo global sin ámbito**: quien lo tiene, tiene los catorce verbos. Emparejar
   un agente le daría de paso escribir en páginas y firmar llaves de acceso.
3. **Los frenos cuelgan del `Servidor`, no del cliente.** Un agente compartiría los 60/12/6 por minuto con la
   extensión y con el refresco del icono, que ya gasta de ahí. El día que uno de los dos se pasara, el que se
   quedaría sin cupo sería el otro.

Lo que **sí** se reutiliza, y literalmente: escuchar y limpiar el socket —el tope de 104 caracteres de
`sun_path`, el `0600`, distinguir el socket vivo del huérfano con `net.Dial`—, la forma de `Respuesta` con sus
**motivos estables**, y el orden de `Atender`: versión, frenos, testigo y estado **antes** de tocar la bóveda.

## Alternativas descartadas

- **Devolverle los secretos al agente.** Es lo que haría cualquiera y es lo que convierte la bóveda en algo
  que se vacía solo: lo que el agente recibe queda en la conversación del modelo, y lo que el agente lee puede
  decirle qué pedir. Se descartó con el cliente delante, eligiendo «que actúe sin ver».
- **Un servidor HTTP local** en vez del socket. Ya se descartó en la ADR 0027 y vale igual: un puerto lo abre
  cualquiera de la máquina y no tiene permisos de fichero.
- **Añadir un SDK de MCP.** Lo que hace falta son tres métodos de JSON-RPC 2.0 —`initialize`, `tools/list`,
  `tools/call`— y es menos código que el CBOR/COSE de las llaves de acceso. Aquí pesa además que **una
  dependencia nueva en el binario que habla con la bóveda** es superficie que auditar en cada actualización.
- **Escribir la configuración en el fichero de cada cliente**, como hace el native messaging. Allí no hay
  alternativa; aquí sí, y los clientes MCP son muchos y cambian. Ajustes **enseña el bloque y lo copia**.
- **Permisos por agente.** Todos los emparejados pueden lo mismo; lo que se puede es echar a uno. Hacerlo por
  agente es una funcionalidad nueva y no se ha pedido.
- **El remoto, ahora.** Tiene una decisión gorda dentro que no se puede tomar de paso: o pasa por nuestro
  Worker —y entonces **el servidor ve los secretos**, que rompe la promesa de la ADR 0036— o hace falta un
  túnel al equipo. Se decide con el local funcionando delante.

## Consecuencias

- **Hay una puerta más hacia dentro**, y hay que decirlo en `docs/seguridad.md` como se dijo la del navegador:
  no es la red, pero baja el listón dentro de «quien ya está en la máquina».
- **El agente no mantiene la bóveda abierta.** La regla de que lo que se repite solo no cuenta como actividad
  es absoluta y vale aquí entera. Lo que **sí** cuenta es aprobar en la ventana, que es un clic de una persona.
- **La bóveda es la que esté abierta**, y el agente no puede conmutar: cambiar de proyecto es administrar. Si
  trabajas dentro de un proyecto, el agente ve ese proyecto; y con una compartida de solo ver, no escribe.
- **Aparece la pregunta de la auditoría**, que hasta hoy no existía: nadie podía contestar «¿qué entradas se
  han abierto?». Con un programa pidiendo cosas, esa pregunta se hace sola, y por eso hay registro —**dentro
  de la bóveda**, nunca en el historial, que es texto en claro—.
- **Y una que no se puede tapar**: esto no protege de un agente al que alguien le haya dicho qué pedir. Lo
  único que hay entre eso y tu bóveda es la aprobación de la ventana, y por eso la válvula de cinco minutos
  **no cubre lo que sí se le enseña**.

## Lo que cambió al escribirlo

**La petición no bloquea.** El plan decía esperar el clic con treinta segundos de plazo, y no se sostiene: si
la ventana está detrás nadie llega, y alargarlo choca con el tope que impone el propio cliente MCP. La forma
buena ya estaba en la casa —es la de `Emparejar`, que **no espera a nadie** porque al otro lado hay un proceso
que pueden cortar en cualquier momento—. El agente pide, Esfinge dice «apruébalo y vuelve a pedirlo», y la
persona aprueba cuando llega.

Con eso se resuelve de paso la tensión que el diseño veía con la ADR 0027: **no hace falta traer la ventana al
frente**, así que esa prohibición se queda intacta.

**El sí va atado a la entrada y vale una vez.** Aprobar «la de GitHub» no sirve para que el intento siguiente
se lleve la del banco, y pedirla dos veces pregunta dos veces — como el testigo de emparejamiento, que se
entrega una sola vez.

**Y la válvula no cubre tres cosas, no una**: el código de un solo uso, cambiar una contraseña y cambiar la
semilla. Todas son lo mismo visto de cerca: **lo que no se deshace**. Copiar se deshace —el portapapeles se
borra solo—, borrar se deshace —treinta días—, y dejar una cuenta sin forma de entrar o soltar seis cifras al
contexto de un modelo, no.

## Y cómo se instala, que no era una línea de JSON

El plan decía: **la configuración se enseña, no se escribe**, con el bloque de `mcpServers` y un botón de
copiar. Es correcto y sigue en pie — pero el cliente miró otro proyecto de la casa, **Cronos**, y dijo que
allí instalarlo «era natural desde Extensiones». Lo es, y por una razón que no se puede copiar: Cronos es un
servidor **remoto**, `{"type":"http","url":"https://cronos.webcafeina.com/mcp"}`, así que se instala pegando
una URL. **Esfinge no puede serlo**: la bóveda está en este equipo y lo que sale de ella no sale de él.

Lo que sí se puede es lo otro que Claude Desktop acepta: un **paquete `.mcpb`** —antes `.dxt`—, que es un zip
con un manifiesto, el binario dentro y un icono, y que se instala **arrastrándolo a su ventana**. Vive en
`internal/mcpb`, lo arma la publicación **y también la ventana**, y cuelga de la *release* al lado
del DMG, uno por sistema. Tres cosas de ese formato que no avisan si se hacen mal:

- **El binario va dentro del paquete**, no apuntado desde fuera: `${__dirname}` se expande a donde se haya
  instalado, y es obligatorio —una ruta relativa no encuentra nada, por lo mismo de siempre—.
- **Claude Desktop le añade `.exe` en Windows** por su cuenta, así que el manifiesto nombra `esfinge-mcp` a
  secas en los tres.
- **No va firmado**, como el resto de lo que sale de aquí (y por lo mismo: los 99 $/año no compensan).

Y el guion **no usa la herramienta de Anthropic** para armarlo: es un zip, y meter una dependencia más en el
camino de publicar es exactamente lo que ya costó cuatro publicaciones caídas.

Con ello, Ajustes enseña **tres caminos en orden de coste** y no uno: el paquete que se arrastra, la orden de
una línea para Claude Code —`claude mcp add esfinge <ruta>`, que ahí no hay fichero que tocar—, y el bloque de
JSON **plegado**, para lo demás. Lo vigila `TestLasTresFormasDeInstalarloNombranLoMismo`, porque **cambiar el
nombre del binario en uno de los tres sitios no da ningún error**: el paquete se instala igual y no arranca.

## Lo que encontró el paseo a mano, y ninguna prueba

Antes de publicar se recorrió el camino entero como lo haría un cliente MCP de verdad —arrancar el binario,
saludar, listar y llamar— y **salieron tres fallos que tenían prueba verde encima**:

1. **El binario no se emparejaba nunca.** Solo `estado` contestaba; todo lo demás decía `sin-emparejar`.
2. **El nombre del cliente no llegaba a la ventana**, que enseñaba «Un agente» en vez de «Claude Code».
3. **Un campo se llamaba `repetidas` y llevaba dentro las reutilizadas.**

Lo que las tapaba es lo mismo en los tres: **las pruebas ponían el testigo a mano**, incluida la que se llama
«la tubería entera», que hablaba con `Atender` directamente. Es la trampa de siempre —cada pieza en verde no
prueba que estén conectadas— con una vuelta de tuerca: el ayudante de la prueba **hacía el trabajo que faltaba
en producción**. De ahí la regla, que ya vale para cualquier cosa con emparejamiento: **lo que el binario tiene
que hacer solo, la prueba no se lo puede dar hecho.**

Y una cuarta la cazó la puerta de publicación y no el paseo: `RegistroDelAgente` era **una llamada aparte para
dibujar**, y con la bóveda cerrada contestaba 400 en cuatro pruebas de Ajustes. Lo que la ventana necesita para
dibujar va en el estado que ya pide — está escrito en el LÉEME del proyecto desde la vez anterior.

## El tampón del registro: descartado midiendo

Quedó escrito como deuda que apuntar en el registro guarda la bóveda entera en cada uso, y que hacía falta un
tampón. **Se midió y no**: veinte apuntes con dos mil entradas son 838 ms **en total**, 42 ms cada uno, y
`EsperaTrasGuardar` ya agrupa las subidas tres segundos. Y hay una razón mejor para no hacerlo: `Cerrar()`
descarta el cuerpo sin guardar, así que un tampón convertiría **matar el proceso en borrar el rastro**. La fila
de la deuda se cerró con la medición, no con código, y lo que había escrito en ella era incorrecto.

## Qué autoriza el emparejamiento, y qué palabra frenaba al agente (2026-10-20)

Probarlo en **Claude Code** —hasta entonces solo se había usado Claude Desktop— destapó dos cosas, y las dos
son de lo mismo: **lo que la pantalla y los textos dicen no era lo que pasa**.

### El permiso es del equipo

El testigo con el que un cliente MCP se identifica lo guarda **él**, en un fichero de la carpeta de
configuración del usuario. O sea **uno por persona**: lo crea el primero que llega y lo reutilizan todos. Así
que al instalar Claude Code entró con el permiso que se le había dado a Claude Desktop, sin preguntar nada
—correcto— y **sin salir en la lista de Ajustes**, que seguía enseñando un solo agente con el nombre del otro
y su botón de «Retirar».

Eso es una pantalla con la forma de un permiso por programa sobre un permiso por equipo. Y había una segunda
mitad peor: `permitir` hacía `append`, así que un emparejamiento nuevo habría dejado **filas válidas y sin
uso** —el cliente solo tiene un fichero, así que solo el último testigo se usa—, sin forma de saber desde la
pantalla cuál estaba viva.

Tres cosas lo cierran. `permitir` **sustituye** en vez de acumular, porque una fila es decir lo que hay. Se
apunta **quién ha hablado de verdad** —el nombre viaja en cada petición, que es por lo que la tarjeta sí
decía «Claude Code»— con `visto`, que **no cuenta como actividad** y solo guarda si el nombre es nuevo o si
su última vez tiene más de un minuto, para no escribir un fichero por llamada. Y la pantalla dice **«Permiso
de este equipo · Autorizado desde… · Han hablado: …»**.

**Y lo grave de por qué nadie lo vio en tres semanas: esa lista no tenía ninguna prueba de interfaz.** Es la
misma familia que la tarjeta que vivía dentro de Ajustes: lo que no se mira no se mira.

### Y «cerrada» significaba dos cosas

`esfinge_bovedas` devolvía `abierta: false` para las bóvedas que no son la activa. En Esfinge **«cerrada»
quiere decir bloqueada** —la que pide la contraseña maestra—, así que el modelo juntó los dos sentidos y
contestó «están cerradas, Esfinge no me deja abrirlas desde aquí».

**Y sí podía.** Al nombrar la bóveda funcionó a la primera, y lo dijo él mismo: «la búsqueda funcionó aunque
la bóveda figuraba como cerrada, porque Esfinge cambia a ella sola». O sea que **una palabra con dos sentidos
le quitaba la funcionalidad entera de la ADR 0055**, que existe precisamente para que el agente no diga que
no puede.

El campo se llama ahora `activa`, y la descripción **desmiente la lectura equivocada en vez de callarla**:
que las que no son la activa se nombran igual, que Esfinge conmuta **sin pedir ninguna contraseña**, y qué
quiere decir «cerrada» aquí. La guarda mira las dos mitades —el nombre en el JSON y las frases— porque el
fallo era de texto y **aquí los textos son el producto**.

Es la trampa de `huella` —la criptográfica y la dactilar— en un sitio donde cuesta más: una palabra ambigua
en una pantalla la lee una persona que ya sabe de qué va; en un esquema la lee un modelo que solo tiene eso.

## Una pregunta a la vez, y se aprueba lo que se leyó (2026-10-19)

Hay **una persona y una ventana**, y desde que puede haber dos agentes a la vez —Claude Desktop y Claude
Code— eso deja de ser una obviedad. Lo que había: `pedir` ponía lo pendiente encima de lo que hubiera, y
`AprobarLoQuePideElAgente` concedía **lo que hubiera, sin decir qué**. Entre las dos cosas, una petición que
llegara entre leer la tarjeta y pulsarla **se aprobaba en lugar de la que se leyó**. Lo vio el cliente —«el
aviso nuevo se ha puesto encima»— y eligió cerrarlo por las dos mitades:

**Una · la ventana dice qué aprueba.** Cada petición lleva una referencia al azar; la tarjeta la devuelve al
contestar, y si ya no es la que espera, **no se concede nada** y se dice por qué. Es la misma regla que
obligó a poner la bóveda en la tarjeta, aplicada al tiempo en vez de al sitio: aprobar sin saber **qué** es
aprobar a ciegas. La referencia se lee **en el render** y no dentro del manejador del clic, que si no
volvería a aprobar lo de después.

**Dos · la segunda espera turno**, con `otra-esperando` como motivo propio, y el agente sabe qué hacer
porque lo dicen las descripciones de los cuatro verbos que piden aprobación.

**Y la frontera fina: entre agentes, no dentro de uno.** Un mismo agente que pide otra cosa es el mismo
interlocutor cambiando de tema, y bloquearlo contra sí mismo lo dejaría dos minutos sin poder trabajar
porque nadie contestó lo de antes — lo destapó una prueba que hacía justo eso. Dentro de un agente la
tarjeta sí puede cambiar, y ahí lo que protege a quien lee no es la puerta, es la referencia.

**Y lo pendiente caduca**, a los dos minutos y **al mirarlo**: la ventana pregunta por el estado a menudo,
así que la tarjeta se cae sola sin un reloj que la vigile. Sin eso, un agente que pide algo y se va —su
cliente lo mató, la conversación se cerró— dejaría a los demás fuera para siempre.

### Dos cosas que salieron de hacerlo

**`mover` contestaba `no-entiendo` cuando pedía aprobación.** Su traductor de errores se quedó corto al
añadir el verbo: el texto llegaba bien y **el motivo era falso**, así que un agente no podía distinguir
«apruébalo» de «no sé qué me pides» sin leer español. Es la regla de releer a los vecinos, aquí en el
traductor de errores y no en las descripciones.

**Y cuatro pruebas preparaban lo pendiente a mano** (`p.pendiente = &q`). Mientras eso no tuvo consecuencias
pasó desapercibido; al darle plazo, caducaban al instante y se cayeron todas. Es la trampa del testigo puesto
a mano: **el ayudante hacía el paso que hace producción**, así que ese paso no lo probaba nadie. Ahora dejan
la petición con `pedir`, como el programa.

## Qué vida tiene un sí, y por qué nadie tiene que vigilarlo (2026-10-16)

Un permiso concedido se guarda hasta que se gasta o caduca, y la pregunta que costó **dos versiones** es qué
otras cosas pueden llevárselo por delante. La respuesta que parecía evidente era «todo lo que cambie el suelo»
—cerrar la bóveda, el bloqueo, conmutar de bóveda— y era la que había: `cerrarLaValvula` borraba la válvula
**y el sí concreto**, con el argumento de que un permiso dado mirando una bóveda no puede valer en otra.

**El argumento era bueno y la implementación lo volvía falso.** Desde la ADR 0055, la ventana vuelve sola a su
bóveda a los diez segundos, y esa vuelta pasa por `cambiarBoveda` y por `alCerrarLaBoveda`. Así que aprobar en
**menos** de diez segundos —o sea, aprobar deprisa, que es lo que hace cualquiera— borraba el sí antes de que
el agente reintentara, y Esfinge volvía a preguntar. Sin fin. El cliente lo vio seis veces seguidas moviendo
una credencial de vuelta a su bóveda; tardando más de diez segundos funcionaba, lo que explica que aquí
pareciera bien.

**La 2.58.1 lo parcheó en el verbo y no en la causa**: guardaba el sí antes de conmutar y lo reponía después,
dentro de `mover`. Eso tapaba la conmutación de `mover` y dejaba fuera la vuelta automática, que pasa por el
mismo sitio — y dejaba fuera a `copiar_contrasena`, `codigo` y `borrar`, que tienen el mismo problema en una
bóveda nombrada y que **nadie había probado**.

### Lo que se hace, que es lo contrario

En vez de proteger el sí de quien conmuta, **el sí nombra entera la acción que se aprobó**: quién la pide, qué
verbo, qué entrada, en qué bóveda y —si mueve— hacia dónde (`mismaPeticion`). Antes se comparaba solo el
identificador de la entrada, y por eso había que tirarlo cada vez que algo se movía: era genérico de más.
Nombrándolo entero **no hay dónde gastarlo mal**, así que conmutar no tiene de qué protegerlo.

Eso es **más** estricto que lo que había, no menos: un sí para «mover esto de Acme a Personal» ya no se puede
gastar en copiar su contraseña, ni en moverlo hacia otra parte, ni en una entrada de otra bóveda con el mismo
identificador.

Quien se lleva el sí por delante son **los dos gestos de una persona que quiere retirarlo**: el botón de
cortar y retirar al agente de la lista (`retirarLoConcedido`). Y la válvula sigue cerrándose con todo lo
demás, porque **ella sí es genérica**: vale para lo que el agente pida después, y lo que pida después puede
estar en otro sitio.

### Lo que esto cuesta, dicho en voz alta

**Un sí puede sobrevivir a cerrar la bóveda**, hasta los dos minutos de `PlazoDelPermiso`. Antes no.

No abre ninguna puerta, y conviene ver por qué antes de aceptarlo: con la bóveda cerrada el agente no puede
hacer nada —todo contesta `cerrada`—, y para que vuelva a haber bóveda hace falta la contraseña maestra o el
dedo, o sea una persona delante. Lo que podría gastarse en esa ventana es **exactamente la acción que esa
misma persona aprobó hace menos de dos minutos**. La alternativa era distinguir «cerrar» de «conmutar» dentro
de `alCerrarLaBoveda`, por donde pasan los dos, y eso es la línea repetida en siete sitios que ya costó una
publicación con `enConfiguracionDePruebas`.

### Y la mitad de método

**La prueba que había pasaba en verde con el bucle dentro, dos versiones seguidas, por el orden de sus pasos.**
Forzaba el respiro **antes** de aprobar, que es el único orden en que el sí sobrevive. Escrita con los tiempos
de quien la vive —la tarjeta sale, se lee, se pulsa, y eso son segundos— cae a la primera. Es la tercera vez en
dos días que una prueba ordena los pasos como funcionan en vez de como pasan, y lo único que lo dijo fue
mutar.

Y la señal de que esta vez el arreglo está donde toca: **se cae el andamio de la anterior**. `elSiEnLaMano` y
`devolverElSi` dejan de hacer falta y se van.

## Qué alcance tiene cada botón, y por qué la ventana no decide eso (2026-10-19)

La tarjeta tiene tres botones que dicen que sí —«Solo ésta», «Todo lo suyo, 5 minutos» y «No volver a
preguntar»— y los dos últimos tienen **el mismo alcance que la válvula y ni un verbo más**, que es la frontera
que eligió el cliente el 2026-10-08. De ahí sale una pregunta que hay que contestar en **un** sitio: ¿este
verbo entra en la válvula?

**Estaba contestada en tres, y desalineada.** En Go era un literal en cada llamada a `pedir`; en la tarjeta,
un `que !== "codigo"` escrito a mano; y en `concederComo`, una suposición —con `unRato` o `paraSiempre`
ponía la válvula o el «siempre» **y nada más**, dando por hecho que el verbo pasaría por ahí—. Cuando `mover`
entró en el grupo de los que no entran (2.58.0), Go se enteró y las otras dos no.

Lo que eso producía, visto por el cliente con la 2.58.2:

- **Bucle infinito y determinista.** Aprobar un movimiento con cualquiera de los dos botones anchos **no
  concedía nada**: el reintento volvía a preguntar, siempre, sin carrera de por medio. No es el fallo del día
  16 —ése era la vuelta de la ventana borrando el sí— sino otro debajo. Y le pasaba igual al código de un
  solo uso y a editar una contraseña, **sin que nadie lo hubiera probado**.
- **Y lo grave: concedía sobre otros verbos.** `AprobarSiempreAlAgente` guardaba el permiso que no caduca
  **antes** de mirar qué verbo estaba pendiente, así que pulsar «No volver a preguntar» en una tarjeta de
  mover **encendía el permiso permanente de copiar contraseñas y de borrar** —en las preferencias, sin
  caducidad— y el movimiento seguía sin hacerse. La persona lee un movimiento y concede otra cosa. Eso es lo
  contrario de lo que esta ADR promete, y por eso cuenta como fallo de seguridad y no de usabilidad.

### Lo que se hace

**`PasaPorLaValvula(que)` es el único sitio donde se decide**, función del verbo y no argumento de nadie. La
frontera, dicha entera: no entra **lo que se le enseña al agente** —el código— ni **lo que no se deshace**
—cambiar una contraseña o una semilla, y mover, que no va a la papelera—.

De ahí salen tres cosas que antes estaban sueltas:

1. **`pedir` pregunta**, en vez de que se lo digan sus cinco llamadas.
2. **El valor viaja en la petición** (`porValvula`), y los dos botones anchos de la tarjeta cuelgan de él. La
   ventana deja de tener lista propia, que es lo que de verdad cierra la familia.
3. **Dos cinturones por debajo**, para el día que alguien vuelva a desalinear algo: un alcance amplio sobre un
   verbo que no entra **concede solo esta vez** —que es lo que la persona está diciendo— y **no se enciende
   ningún permiso global** que esta petición no pueda usar.

### Y la mitad de método

Es **la tercera vez** que la misma regla escrita en dos listas cuesta un fallo: `vaciarLoSensible`
—los campos sensibles en dos sitios y solo uno completo—, `enConfiguracionDePruebas` —siete listas y a seis
les faltaba Windows, y tiró una publicación— y ésta. El patrón no es el descuido: es que **añadir un caso
nuevo obliga a acordarse de N sitios**, y acordarse no es defensa. Lo que lo cierra no es revisar mejor, es
que **haya un sitio y los demás pregunten**; y cuando no se puede juntar —Go y TypeScript—, una prueba que
**compare las dos listas entre sí**, no cada una contra lo que parece.

Y una señal que había escrita y no se leyó: el comentario de la tarjeta decía, de su propia mano, que
«ofrecer aquí un botón que no va a hacer lo que dice sería peor que no ofrecerlo». La regla estaba bien; lo
que falló fue que **su condición era una copia** y no la fuente.

## Verificación

Lo que hay que poder decir al acabar, y cada prueba mutada:

| Qué protege | Prueba |
|---|---|
| Lo que un agente **no** puede conseguir | Tabla donde **cada caso falla por su motivo concreto**: sin emparejar, con la bóveda cerrada, sin bóveda, pasado el freno, verbo que no existe y administrar |
| Que el secreto no vuelve por el canal | Serializar la respuesta a JSON y buscar la contraseña **en los bytes**, no en los tipos |
| Que no mantiene la bóveda abierta | 30 vueltas de un minuto pidiendo cosas con el reloj corriendo, y la bóveda cerrada al final |
| Que la válvula no cubre lo que se enseña | Con la válvula abierta, el código **sigue pidiendo** aprobación |
| Que un sí sobrevive a la vuelta de la ventana | Se pide en una bóveda nombrada, **se aprueba y el respiro vence después** —el orden de quien lo vive—, y el reintento funciona. Para `mover` y para un secreto, que es la mitad que faltaba |
| Que un sí no se gasta en otra cosa | Cinco peticiones que cambian **un solo campo** cada una —verbo, entrada, bóveda, destino, quién— y ninguna puede gastarlo; y la misma, exacta, sí |
| Que los tres botones valen para lo que no entra en la válvula | **Tres botones × dos verbos** —mover y el código—, los seis aprobando y reintentando. Antes pasaban dos de seis |
| Que aprobar una cosa no concede sobre otra | Se aprueba un movimiento con «No volver a preguntar» y después **copiar una contraseña tiene que seguir preguntando** |
| Que la ventana no tiene lista propia | Una guarda **lee `App.tsx`** y exige que los botones anchos cuelguen de `porValvula`, y que no haya vuelto a aparecer un `que !== "<verbo>"` |
| Que los verbos que no entran son los que se sabe | Se recorren **las herramientas declaradas** y se comparan con la lista, por su nombre; y lo que se le enseña al agente **y se aprueba** no entra nunca, dicho por las banderas |
| Que lo que una persona retira se va | El botón de cortar y retirar al agente, los dos con el sí ya concedido |
| Que una petición nueva no pisa la que se lee | **Dos agentes**: el segundo recibe `otra-esperando` y la tarjeta sigue enseñando lo del primero |
| Que un agente no se bloquea contra sí mismo | El mismo pide otra cosa y **entra**, porque es el mismo interlocutor cambiando de tema |
| Que pulsar sobre una tarjeta que cambió no concede nada | Se lee la referencia, cambia lo pendiente, y el sí **sobre la vieja** no concede **ninguna de las dos**. Su mutación es la que destapó que esta prueba faltaba |
| Que lo pendiente caduca | Pasado el plazo la tarjeta se cae sola y otro agente puede pedir lo suyo |
| Que el permiso es uno por equipo | Se empareja **dos veces** y la lista tiene **una fila**; el testigo viejo deja de valer |
| Que se sabe quién ha hablado | Dos clientes con el mismo testigo, el último primero, y hablar otra vez **no duplica** |
| Que la pantalla lo dice | Una de interfaz que entra en Ajustes → Agentes: «Permiso de este equipo», los dos nombres, y **que no vuelva a decir «Agentes permitidos»** |
| Que no se le dice «cerrada» a una bóveda que puede usar | El campo se llama `activa` en el JSON, y la descripción dice las tres cosas que lo desmienten. Mutadas las dos mitades |
| **Y comprobado en su Mac el mismo día** | Con la bóveda bloqueada el agente dice «cerrada, o sea bloqueada y a la espera de la contraseña maestra»; con ella abierta, «Personal es la activa; para usar las otras basta con nombrarlas, Esfinge cambia sola sin pedir contraseña». **Las dos mitades**: que la palabra siga sirviendo donde sí es verdad era lo que no se podía dar por hecho |
| Que el freno es del servidor | **Una conexión nueva por petición**, que es como se descubrió que el otro freno no frenaba |
| Que añadir un verbo es una decisión | La lista recorrida, y ninguno contesta «no entiendo» |
| La tubería entera | Los bytes de MCP por donde entran de verdad → socket → bóveda real → respuesta |
| Que el registro sobrevive a una sincronización | Dos equipos, y **mutando el espejo la cruzada se pone roja** |

| Que el paquete de Claude Desktop tiene la forma | Se arma de verdad, se abre el zip y se mira dentro: **un paquete mal formado no da ningún error**, simplemente no instala |
| Que las tres formas de instalarlo nombran lo mismo | El manifiesto, la orden y el JSON comparados entre sí |

**Y lo que no se puede probar aquí, dicho como tal:** que un cliente MCP de verdad lo cargue y lo use. Eso se
ve en el Mac del cliente, como todo lo demás. Lo más cerca que se llegó es **el paseo a mano**, que arranca el
binario y habla con él por su entrada estándar — y encontró tres fallos de producto que ninguna prueba veía.
