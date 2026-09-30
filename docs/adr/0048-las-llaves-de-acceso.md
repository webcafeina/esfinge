# ADR 0048 — Las llaves de acceso, la sexta clase de entrada

**Fecha:** 2026-09-29, ampliada el 2026-09-30 con la P2 · **Estado:** aceptada; **la P1 vista en el Mac
(2.32.0); la P2 escrita, sin ver** · Primeras dos de las cuatro entregas de `docs/passkeys.md` ·
**Revisar** al empezar la P3, que es cuando se crean llaves

## Contexto

El cliente pidió las passkeys el 2026-09-23: *«la opción que tiene Dashlane del passkey, como por ejemplo al
acceder a GitHub, que simplemente sale un banner de passkey que reconoce Dashlane y es darle a Aceptar»*.
Se investigó ese día en `docs/passkeys.md`, que concluye que **no es una tarea, es una fase**, y se partió
en cuatro entregas. **Ésta es la primera**, y es la única que no se ve: la llave de acceso dentro de la
bóveda, sin navegador.

Se publica sola **a propósito**, y ésa es la decisión menos obvia de la ficha: mete el formato nuevo en
todos los equipos y en la extensión **antes** de que existan llaves de verdad. Cuando empiecen a aparecer,
ninguna copia de Esfinge se encontrará una clase que no entiende. Es barato ahora y caro al revés.

## Decisión

**Una clase nueva, `llave`, con los campos de una credencial WebAuthn**: `rpId`, `idCredencial`,
`idUsuario`, `nombreVisible`, `algoritmo` y `clavePrivada` —ésta **en PKCS#8**, ver abajo—. Los nombres del protocolo se dejan sin traducir,
como `TOTP`: tienen que poder compararse con la especificación sin nada por el medio.

### Sin contador de firmas

WebAuthn define un contador por credencial para que un sitio detecte una llave clonada. **No se guarda**, y
se firmará siempre con cero, que es lo que hacen Dashlane, 1Password y Bitwarden.

La razón no es la pereza: **una llave sincronizada entre equipos no puede llevar un contador coherente**
—dos equipos firman a la vez y lo descuadran—, así que la detección que se perdería no funcionaría de todos
modos. Y tiene una consecuencia técnica que vale la entrada entera: **esta clase no necesita ninguna regla
de fusión propia**. Un contador monótono habría sido el primer campo de la bóveda con reglas a medida,
escritas dos veces, y con ellas la prueba de fusión al azar de tres equipos habría dejado de salir gratis.

### La clave privada no se enseña, no se copia y no cruza

Es la única clase cuyo secreto **no tiene ninguna pantalla**: de una llave no hay nada que leer ni que
teclear en ningún sitio. `vaciarLoSensible` la vacía, así que no llega ni a la ficha; y cuando llegue la P2,
lo que salga del trabajador será **la firma ya hecha**, nunca la clave — igual que `CodigoDeBoveda` cruza el
código de seis cifras y jamás la semilla.

Del identificador de credencial y del de usuario se vacían también. No son secretos —el sitio los emitió él—
pero no hacen falta en ninguna lista, y lo que no hace falta no viaja. **El sitio y el nombre visible sí se
quedan**: son lo único por lo que alguien puede reconocer una llave en una lista donde no se enseña nada
más.

### No se puede crear a mano

`llave` no sale en el selector de «Nueva». Una llave no se inventa: la emite el sitio. Una escrita a mano no
abriría nada, así que ofrecerla sería ofrecer una entrada que no puede funcionar. El sitio y la cuenta se
enseñan en el editor **desactivados**: que estén dice qué llave es ésta; que no se toquen dice que
cambiarlos aquí no cambiaría nada allí.

### Exportar: aparte y cifrada

La exportación en claro **no las lleva**, y lo dice en su aviso. Una fila de CSV con una clave privada
dentro sería lo más peligroso que Esfinge escribiera nunca en el disco —lo que abre la cuenta, en texto— y
además **no le serviría a ningún gestor**, porque ninguno sabe leerla.

Salen por `ExportarLlaves`, en un contenedor **ESF1** con su propia clave, reutilizando `internal/cripto`
tal cual. Así se sigue cumpliendo «una bóveda de la que no se puede salir es una trampa» sin pagar el precio
de cumplirla en claro. **La clave del fichero no es la maestra** y se pide aparte: quien guarda esa copia la
guarda en otro sitio, y reutilizar la maestra haría que perder el fichero fuera perder la bóveda.

### Compartir sí, con su propio aviso

Lo eligió el cliente y no se discute. Pero **no es lo mismo que compartir una contraseña** y la pantalla lo
dice antes de pulsar: una contraseña compartida se puede cambiar y el sitio avisa del cambio; una llave
compartida es **la identidad en ese sitio**, no se puede revocar desde Esfinge y **el sitio no se entera
nunca de que ahora hay dos**. Para dejar de compartirla hay que borrar la llave en el sitio.

Técnicamente ya funcionaba —`envio.go` manda la entrada entera— y eso era justo el peligro: **funcionaba sin
que nadie lo hubiera decidido**.

### La clave privada se guarda entera, y eso corrige lo que decía esta ficha

Esta ADR decía, escrita antes de implementarla, que `clavePrivada` sería «el escalar de 32 bytes, no el JWK
entero: la parte pública se recalcula». **Es falso**, y solo se ve al hacerlo: WebCrypto **no puede importar
una privada P-256 sin `x` e `y`** —lo rechaza con `DataError`— y no expone ninguna forma de multiplicar un
escalar por el generador. Se comprobó intentándolo.

Se guarda en **PKCS#8**, que lleva las dos partes dentro, lo entienden los dos lados sin escribir una línea
—`x509.ParsePKCS8PrivateKey` y `crypto.subtle.importKey("pkcs8", …)`— y son 138 bytes. Queda escrito aquí
porque es exactamente la clase de detalle que parece un ahorro sobre el papel y no existe.

## La P2: usar una llave que ya existe (2026-09-30)

Lo que pidió el cliente con esas palabras: entrar en GitHub y darle a Aceptar. Cuatro decisiones que no
estaban en la P1 y que no se cambian sin preguntar.

### Hay código de Esfinge dentro de cada página `https`, y es lo más caro de toda la fase

`navigator.credentials` solo existe en el mundo de la página, así que para enterarse de que un sitio pide
una llave hay que estar ahí. Hasta ahora todo lo de Esfinge iba en el mundo aislado: la página no lo veía
ni lo podía tocar, y **un fallo nuestro rompía el relleno**. Ahora un fallo **rompe el inicio de sesión del
sitio**, también para quien no use Esfinge en esa cuenta.

De ahí que `navegador/src/mundo.ts` esté escrito entero en negativo, y que el orden de sus comprobaciones
sea parte de la decisión: **en un marco ajeno no se instala nada**, **sin acuse del puente no se instala
nada**, **si otro gestor ya ha parcheado no se instala nada**, y **ante cualquier duda se cede**, donde
ceder es llamar al método original con los mismos argumentos y el mismo `this`. Un `catch` que envuelve
todo cede también: de ahí no sale nunca una excepción hacia la página.

Y una consecuencia que hay que decir en voz alta porque nadie la va a deducir: **el shim cede en la misma
vuelta del bucle de eventos** cuando en este sitio no hay nada que ofrecer. No es una optimización. Es lo
que impide que esperar a un trabajador MV3 dormido se coma la activación de usuario que `create()` exige,
que es literalmente el escenario «rompemos a quien no usa Esfinge».

### El navegador se queda la lista de dominios con llave, y eso relaja una regla escrita

La cabecera de `navegador/src/fondo.ts` dice, desde que existe, que **no se guarda nada de lo que se
pregunta**: «un caché aquí sería la lista de sitios de la bóveda escrita en el perfil del navegador, sin
cifrar, que es exactamente lo que Esfinge cifra en su disco».

**Esta entrega abre una excepción a esa regla, y se dice con esas palabras**, no en un comentario que se
pueda leer como un detalle: el navegador guarda **los dominios que tienen llave de acceso**. Van en
`storage.session` —que muere al cerrar el navegador y nunca llega al disco— y son solo dominios: ni
cuentas, ni usuarios, ni identificadores de credencial, ni nada de la llave.

Hace falta porque el cliente decidió que **con la bóveda cerrada el banner sale y ofrece abrirla**, y con la
bóveda cerrada no hay a quién preguntar. Sin la lista, esa decisión quedaba escrita y no hecha: el shim
cedía antes de preguntar y salía el diálogo del navegador.

Dos correcciones de lo que decía el plan, las dos por implementarlo:

- **Va el `rpId`, no su dominio registrable.** Con el registrable, una llave de `accounts.google.com`
  sacaría el banner en `mail.google.com`, donde WebAuthn no la deja usar: ofrecer abrir la bóveda para algo
  que luego no se puede dar es un banner que estorba. Sigue siendo un dominio y nada más.
- **La lista viaja solo en la pregunta de antes de que el sitio hable.** En la de firmar el sitio ya está
  dicho, y repetirla sería mandar la lista entera de la bóveda en cada firma.

Y la consecuencia que hay que decir en la pantalla y no solo aquí: **recién abierto el navegador, y hasta
abrir la bóveda una vez, el banner no sale.**

### Se publica encendida, con un interruptor en Ajustes

«Usar tus llaves de acceso en el navegador», en los Ajustes de la ventana, **encendido de fábrica**. Es el
freno de emergencia de la fase: si un sitio grande cambia y deja de entrar, esto se apaga y se sigue
trabajando **sin esperar a una versión**, que en una tienda son días.

Apaga **en las tres puertas** y no en una: `Llaves` contesta que no hay ninguna —sin error, para que el
banner no diga nada y Esfinge no se note—, `DominiosConLlave` deja de apuntar nada, y `FirmarLlave` se
niega. Las tres se pueden olvidar por separado.

Va en Go y no en la extensión a propósito, que es la regla de la ADR 0032: **lo que decide qué se ofrece
vive en el núcleo**, porque publicar un arreglo en Go es empujar una etiqueta. Con ello, **una laguna que
se apunta y no se esconde**: la extensión con cuenta no le pregunta nada a la ventana (ADR 0040), así que
para quien use solo la extensión ese interruptor no existe y su único freno es desactivarla entera. Está
en `docs/deuda.md` y se decide en la P3.

### El aviso de datos sube de 3 a 4

Lo pide la ADR 0033 y aquí no hay margen de interpretación: **cambia dónde corre el código**. Que lo que
salga hacia el sitio sea una firma y nunca la clave no quita que sea una práctica de datos distinta de lo
declarado, y darlo por sabido sería decidirlo por quien instaló la extensión antes. Con él cambian el aviso
del panel, `web/privacidad.html`, el texto generado de Firefox y las dos fichas de tienda.

## Alternativas descartadas

- **Guardar la llave dentro de la credencial del sitio**, como el código de un solo uso. Menos entradas y
  todo lo de un sitio junto. Se descartó porque un sitio puede tener llave **sin** contraseña —y entonces
  habría que inventar una credencial vacía— y porque compartir o borrar dejarían de poder separarse.
- **Mantener el contador de firmas.** Ver arriba.
- **Exportar las llaves en el CSV de siempre.** Coherente con «se puede salir de todo», y descartado: un
  fichero de texto con claves privadas dentro.
- **No exportarlas en absoluto**, que es lo que hacen los demás gestores. Se descartó con el cliente: deja
  la bóveda sin salida para esa clase.

Y de la P2:

- **Un identificador secreto en el primer mensaje, en vez de un `MessagePort`.** No vale, y por una razón que
  se ve sola en cuanto se escribe: el identificador **viaja en ese primer mensaje**, que cualquier oyente de
  la página puede leer, y a partir de ahí lo tiene. El puerto sí, porque no se puede escribir en él sin
  tenerlo. La marca sirve para distinguir nuestro tráfico del de Stripe o Intercom, **no para autenticar**.
- **Pedirle la bandera al mundo aislado en vez de que él la empuje.** Preguntar es esperar, y esperar es lo
  que puede agotar la activación de usuario. Se empuja.
- **Un plazo corto de espera cuando la bandera todavía no ha llegado.** Descartado por lo mismo, y con un
  coste que se acepta y se apunta: quien pulse «Entrar con llave» en el primer segundo de cargar la página
  verá el diálogo del navegador. Está en `docs/deuda.md`, y lo que haría falta para reconsiderarlo es
  **medir** cuánto tarda de verdad un trabajador frío, que hoy es una estimación.
- **Usar `encaja()` de `dominios.ts` para decidir con qué `rpId` se firma.** Es la alternativa que más se
  parecía a la buena y la más peligrosa: compara dominios registrables, así que `accounts.google.com` y
  `mail.google.com` serían el mismo sitio. WebAuthn pide el anfitrión o un sufijo suyo separado por punto, y
  **el hash se calcula sobre la cadena exacta**: dar por buenos dos nombres distintos no es ser tolerante,
  es firmar para quien no es.
- **Guardar la lista de dominios en `storage.local`.** Sobreviviría al cierre del navegador y el banner
  saldría siempre, que es mejor producto. Descartado: eso es la lista de sitios de la bóveda escrita en el
  perfil, sin cifrar, que es exactamente lo que la regla de `fondo.ts` prohíbe.

## Consecuencias

- **Séptima pestaña en la bóveda.** Medido: los siete glifos con el rótulo de la activa ocupan ~475 px de
  los 560 de la columna. Lo vigila la prueba que ya existe, que mide **con cada pestaña activa**.
- **Un glifo más que convive con el de credencial**, y los dos son una llave: lo que los separa es la
  orientación —la de la contraseña va tumbada, la de acceso de frente—. Hay que mirarlo en un Mac.
- **Todo cambio del formato se hace dos veces**, como manda la 0040.
- **La exportación gana una puerta**, y con ella un método más en la lista blanca del puente. Cruza una
  clave: la del fichero. No cruza ninguna clave privada.

Y de la P2:

- **Hay código nuestro en cada página `https` que se abra**, y un fallo ahí no deja el relleno a medias: deja
  el sitio sin poder entrar. Es el riesgo número uno de la fase y lo que justifica el interruptor.
- **`pagina.ts` se ha movido de `document_idle` a `document_start`**, porque el puerto se transfiere antes de
  que exista el primer `<script>` de la página. Ahí no hay `<body>`, así que el arranque está partido en dos:
  el puente y el consentimiento al principio, y todo lo demás tras `DOMContentLoaded`.
- **El navegador guarda algo de la bóveda**, por primera vez sin que esté cifrado: la lista de dominios. Ver
  arriba.
- **Tres reglas nuevas en `herramientas/permisos.mjs`**, porque los tres fallos serían mudos: un guion del
  mundo principal **no puede usar `api.`/`chrome.`/`browser.`** —no existen ahí—, si hay uno **todos** los
  guiones de contenido van a `document_start`, y todo `js` del manifiesto tiene que estar en `COMPILAR.md`,
  que es lo que Mozilla compara byte a byte.

## Verificación

**Lo que se comprobó, y cómo**

- **Cinco mutaciones, las cinco rojas**: sin vaciar la clave privada de la papelera, sin sacar las llaves de
  la exportación en claro, escribiendo el claro al lado de lo cifrado, sin la huella propia de la llave en
  el importador, y sin el identificador en `claveDeCuenta`.
- **La prueba de duplicados mira `Conflictos`**, no solo `Metidas`: una huella que choca no impide que la
  entrada entre, la marca y la mete igual. Es la lección de los datos personales, aplicada de entrada.
- **La lista blanca del puente cazó el método nuevo** antes de que nadie se acordara de él, que es para lo
  que está.
- **Las tres pruebas cruzadas**: forma canónica, lo que se vacía y la fusión al azar de tres equipos.
- **Y la firma, de punta a punta**: la extensión firma con una llave hecha allí y con otra hecha en Go, y
  **Go las verifica las dos**. Es lo único que dice que la conversión de P1363 a DER está bien sin un sitio
  de verdad, porque los bytes de una firma ECDSA cambian en cada llamada y compararlos no vale para nada.
  **La dirección contraria no está a propósito**: WebCrypto solo verifica en P1363, así que probarla exigiría
  escribir un descodificador de DER que producción no usa, y el DER de Go lo escribe la biblioteca estándar.
- **El DER, con vectores fijos aparte** (`navegador/pruebas/llaves.spec.ts`), y ésa es la parte que la
  prueba de punta a punta **no puede cubrir**: una firma al azar empieza por cero una vez de cada
  doscientas cincuenta y seis, así que el recorte de ceros no se ejercita nunca. Comprobado mutándolo — la
  cruzada seguía en verde.

**Y una cosa que encontró la fusión al azar y no era de esta clase:** el espejo de TypeScript leía **todos**
los campos numéricos como si fueran `revision` —el nombre estaba escrito a fuego, de cuando la revisión era
el único número del formato—. `algoritmo` es el segundo número de la historia del formato, y se guardaba
encima de la revisión: la misma bóveda fundía distinto en los dos lados. Estaba ahí desde el principio,
esperando a que hubiera un segundo número.

**Y un fallo que se vio en el Mac con la 2.32.0 ya publicada:** exportar abría el diálogo del sistema y
descubría **al ir a escribir** que no había ninguna llave, así que pedía una clave y un sitio para un
fichero que no iba a existir — con la bóveda recién creada, que es el caso más probable de todos. Arreglado
en la 2.32.1: se mira antes de preguntar, y la ventana lo dice de entrada. La prueba que lo cubre **pasó en
verde al escribirla**, porque el doble del sistema apuntaba el argumento del diálogo y no si se le había
llamado; ahora cuenta las veces.

**Lo que no se ha comprobado y hay que decir**

- **No se ha visto en un Mac**: ni el glifo al lado del de credencial, ni la ficha de una llave, ni la
  exportación cifrada.
- **No hay ninguna llave de verdad todavía.** Todo lo probado son entradas escritas por las pruebas; la
  primera llave de verdad la creará el navegador en la P2, y hasta entonces no se sabe si los campos que se
  guardan son exactamente los que hacen falta para firmar.

**Lo que se comprobó de la P2, y cómo**

- **El autenticador virtual de CDP, con una llave que no está en Esfinge** (`pruebas-reales/llaves.spec.ts`).
  Es la prueba de «no rompemos a quien no usa Esfinge», y el plan decía que sin ella la P2 no se publica:
  crear y entrar siguen funcionando con la extensión puesta, en los dos estados que importan —el aviso sin
  aceptar, donde el guion ni arranca, y con el shim instalado—. Dos mutaciones: **devolver una credencial
  inventada en vez de ceder** la pone roja, y **quitar el bloque `world: "MAIN"` del manifiesto** también,
  porque la prueba comprueba primero que el shim está de verdad ahí.
- **El banner con la bóveda cerrada, de punta a punta** (`pruebas-reales/con-cuenta.spec.ts`), con la
  extensión de verdad: la llave llega por la sincronización, se visita el sitio con la bóveda abierta para
  que la lista se apunte, se bloquea y el banner sale. Es la única prueba que dice que **las tres piezas
  están conectadas**, y cae al quitar la rama de `quizas`.
- **El `rpId` permitido, con tabla cruzada en los dos lados.** El caso que hay que tener y que lo encontró
  una mutación y no la lectura: sin exigir el punto que separa, `malaejemplo.com` **termina en**
  `ejemplo.com` y firmaría por él. La tabla tenía `ejemplo.com.malo.com` —que no ataca nada, porque ahí el
  nombre va en medio— y con ella la comprobación se podía quitar entera sin que nada se pusiera rojo.
- **El freno de Ajustes, en las tres puertas**, y esa prueba enseñó de paso que **un doble no basta**: con la
  clave privada inventada, `FirmarLlave` falla igual por no poder leerla, así que la comprobación del freno
  se podía quitar entera y la prueba seguía verde. Ahora la llave es de verdad y se firma con ella **antes**
  de apagar.

**Y el fallo de la P2 que más costó, que es el que nadie habría buscado:** el saludo del puente iba en un
solo sentido. Los dos guiones entran en `document_start` y **su orden no está garantizado** —el plan lo dejó
escrito como lo que había que comprobar—, pero lo que lo rompía de verdad es otra cosa: el lado aislado **no
puede escuchar hasta haber leído el consentimiento**, y eso es un `await` a `storage`. Así que el saludo del
mundo principal se disparaba contra un `window` sin oyentes y se perdía; el shim no se instalaba, la página
funcionaba como si Esfinge no estuviera, y **no había error en ninguna parte**. Dio la cara como la prueba
real pasando tres veces y a la cuarta no.

Ahora cada lado anuncia y cada lado escucha, idempotente por los dos: el que atiende **se anuncia** en
cuanto puede, el que saluda **vuelve a tender** si le llega ese anuncio sin tener acuse, y **deja de tender
en cuanto lo tiene** —eso último no es cosmético: un puerto transferido por `window.postMessage` lo puede
recoger cualquier oyente, así que solo se manda mientras no existe el primer `<script>` del sitio—.

Y la lección de método, que vale para cualquier protocolo entre dos piezas: **las cuatro pruebas que había
atendían antes de saludar**, o sea el caso que no ocurre. La que faltaba fuerza el orden de verdad, y es la
única que lo caza siempre; quitando el anuncio, la prueba real vuelve a ser **intermitente**, que es el
síntoma original y no un fallo limpio.

**Lo que no se ha comprobado de la P2, y hay que verlo en su Mac**

- Que el banner **se ve y se distingue** del diálogo del navegador. Aquí solo hay capturas.
- **Entrar en GitHub de verdad**, que es lo que pidió el cliente con esas palabras.
- Que **sin llave guardada el diálogo del navegador sale igual**, en un sitio de verdad y no en uno servido
  por Playwright.
- Que **los antibot** de los sitios que usa no marcan el navegador. No se puede saber sin probarlo, y por eso
  el disfraz de `Function.prototype.toString` **no se ha puesto**.
