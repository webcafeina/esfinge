# ADR 0048 — Las llaves de acceso, la sexta clase de entrada

**Fecha:** 2026-09-29, ampliada el 2026-09-30 con la P2 y la P3 · **Estado:** aceptada; **la P1 vista en
el Mac (2.32.0); la P2 y la P3 escritas, sin ver** · Tres de las cuatro entregas de `docs/passkeys.md` ·
**Revisar** al empezar la P4, que es Firefox

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

## La P3: crear llaves, y por qué salió antes de lo previsto (2026-09-30)

**La P3 se adelantó porque sin ella la P2 no se puede usar.** Al ir a guiar al cliente en la primera
prueba apareció que **no hay ninguna forma de que exista una llave de acceso**: a mano no —el selector de
clases excluye `llave` a propósito—, desde el navegador no —crear era esta entrega— e importando tampoco,
porque ningún gestor exporta passkeys en su CSV. La P2 quedó publicada protegiendo a quien no usa Esfinge
y sin poder hacer lo que se pidió.

Es el fallo de «preguntar antes de preguntar» a escala de entrega, y la regla que lo habría cazado ya
estaba escrita: **un procedimiento se ejecuta antes de escribirlo**. Recorrer «probar la P2 en el Mac»
paso a paso se para en el primero.

### Se le dice al sitio que aquí hay un autenticador de plataforma

`isUserVerifyingPlatformAuthenticatorAvailable` contesta `true`. **Eso es decir que el equipo tiene un
autenticador de plataforma cuando puede no tener ninguno**, y se hace a sabiendas: es lo que los sitios
preguntan para decidir si ofrecen crear una llave, y sin eso Esfinge no serviría justo en el ordenador sin
Touch ID ni Hello, que es donde más falta hace.

Lo que acota la afirmación, y no es poco: **solo se dice que sí cuando Esfinge de verdad puede crear**. Con
el interruptor apagado se devuelve lo que conteste el navegador, que es la verdad de ese equipo.

Y una diferencia técnica que importa: **ese método sí puede esperar** a que llegue la bandera del mundo
aislado, medio segundo, porque **no consume la activación de usuario**. Esperar en `create()` o en `get()`
es lo que puede agotarla y romper el inicio de sesión de quien no usa Esfinge; aquí no hay nada que agotar.

### Lo que no se crea, y cada cosa por su motivo

- **Sin `-7` en `pubKeyCredParams` no se crea nada.** Es el único algoritmo que Esfinge sabe firmar, y
  crear la llave la registraría en el sitio dejando la cuenta con **una credencial muerta**.
- **Si el sitio dice en `excludeCredentials` que ya tiene una llave nuestra, tampoco.** El motivo es suyo y
  no nuestro: dos llaves de Esfinge para la misma cuenta son dos credenciales que él guarda sin que nadie
  las haya pedido. Lo que hace el `shim` entonces es **ceder**, así que sale el diálogo del navegador.
- **Y el `rpId` se comprueba contra el origen igual que al firmar.** Aquí es peor que al firmar: crearía en
  la bóveda una llave atada a un sitio que no la pidió.

### El AAGUID va a ceros

Identifica el modelo de autenticador, y los gestores suelen poner el suyo para que el sitio enseñe su
nombre. Aquí va a cero porque **con `fmt: "none"` es lo que dice la especificación** —sin atestación no hay
nada que identificar— y porque inventarse un identificador de modelo es afirmar algo que nadie ha
certificado. **El coste es que el sitio dirá «una llave de acceso» y no «Esfinge»**, y se acepta.

### Y el aviso de datos sube otra vez, de 4 a 5

Sube aunque la 4 no haya llegado a nadie —la versión que la lleva está en revisión— y aunque **no haya
ninguna categoría de datos nueva**. Lo que hay es una frase del aviso que dejaría de ser cierta: decía que
lo que sale hacia el sitio es la firma, y ahora también sale una clave pública recién hecha. La regla de la
ADR 0033 no dice «si cambian los datos», dice **si cambia lo que dice el aviso**.

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

Y de la P3:

- **Dejar crear una llave a mano en la ventana**, para poder probar la P2 sin la P3. Se descartó en cuanto
  se dijo en voz alta: una llave creada aquí **no sirve para entrar en ningún sitio**, porque el sitio tiene
  que conocer su parte pública y solo la conoce si él la pidió. Habría sido una pantalla que no lleva a
  ninguna parte.
- **Poner un AAGUID propio** para que los sitios enseñen «Esfinge». Ver arriba.
- **Rechazar con `InvalidStateError`** cuando el sitio dice que ya tiene una llave nuestra, que es lo que la
  especificación pide para que el sitio diga «ya tienes una». Descartado porque **de esta pieza no sale
  nunca una excepción hacia la página**, que es la regla que protege el inicio de sesión de todo el mundo.
  Se cede, y el coste es que el navegador ofrecerá crear una del sistema.
- **Sacar la pública y el `authData` del objeto de atestación** en vez de mandarlos aparte. Exigiría
  **descodificar CBOR en el mundo principal**, dentro de la página de otro, y aquí no hay descodificador de
  CBOR a propósito. Duplicar una clave pública es más barato y más seguro; ninguno de los dos es secreto.

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

Y de la P3:

- **Una llave creada en Esfinge y perdida es una cuenta perdida.** Es la consecuencia que ordena todo lo
  demás, y por eso la exportación cifrada se hizo en la P1: cuando se pudiera crear la primera, ya tenía que
  haber por dónde sacarla.
- **Los sitios sin Touch ID empezarán a ofrecer llaves de acceso**, porque se les dice que este navegador
  puede guardarlas. Es lo que se quería, y a la vez significa que la oferta aparecerá en equipos donde antes
  no aparecía.
- **Y hay un CBOR escrito a mano en el proyecto**, solo codificador. Un descodificador sería superficie de
  ataque sobre bytes de fuera y no hace falta en ningún sitio.

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

**Lo que se comprobó de la P3, y cómo**

- **El ciclo entero en Go**: crear y luego **firmar con lo guardado**, verificando que la pública que fue al
  sitio es la de esa llave. Sin esa segunda mitad, verificar la firma solo demostraría que ECDSA funciona.
  Cinco mutaciones, las cinco rojas — incluida guardar una privada y mandar la pública de otra.
- **Las cuatro puertas de antes de escribir**, cada una con su caso: el algoritmo, el `rpId`, el reto y las
  excluidas. La del algoritmo casi se queda sin probar: quitarla del código **no hacía caer nada**, y la
  mutación solo se ponía roja porque dejaba un `import` sin usar. Eso no es una prueba, es una casualidad
  del compilador.
- **Los bytes, cruzados**: el COSE, el `authenticatorData` con la credencial dentro y el objeto de
  atestación, con casos elegidos por lo que puede divergir —una coordenada que empieza por cero, una corta,
  un identificador de más de 255 bytes y un `rpId` con acentos—.
- **Y el orden canónico de los mapas CBOR, que no estaba comprobado en ninguna parte.** Lo dijo una
  mutación que pasó en verde: ninguno de los dos mapas reales distingue «ordenar por largo y luego por
  bytes» de «solo por bytes», porque en el COSE todas las claves miden uno. Ahora hay una cruzada con
  claves donde sí discrepa.
- **Un vector fijo con las dos coordenadas cortas.** Diez llaves al azar **no cazan el relleno**: una
  coordenada de P-256 empieza por cero una vez de cada 256, así que con veinte la rama se toca el 7 % de las
  veces. Dejarlo al azar sería peor que no probarlo — una prueba que falla una de cada trece veces es un
  intermitente. Es lo mismo que se hizo con el recorte de ceros del DER.
- **Y el relleno estaba en dos sitios**, así que quitando uno el otro lo tapaba y ninguna de las dos líneas
  estaba probada. Ahora rellenar es del formato y vive donde se escribe el formato.
- **Cinco mutaciones del `shim`**, las cinco rojas: cambiar la bandera de crear por la de usar, atender sin
  poder crear, quitar `getPublicKey`, decir siempre que hay autenticador y perder las excluidas.
- **Con la extensión de verdad** (`pruebas-reales/con-cuenta.spec.ts`): crear, que esté en la bóveda, firmar
  con ella, verla subir a la cuenta y que el banner de crear salga en una página.

**Y la mutación que destapó el agujero de esa última**, que es la lección de método de la P3: guardando en
la bóveda **una privada distinta de la que se le dice al sitio**, la prueba con la extensión de verdad
seguía en verde. Comprueba que firmar funciona, no que el sitio pueda verificar esa firma — y en TypeScript
no se puede verificar, porque WebCrypto solo hace P1363 y WebAuthn manda DER.

Eso es una cuenta con una credencial registrada y sin ninguna forma de entrar. Se cerró por el único sitio
donde se podía: una cruzada en la que **el núcleo de la extensión crea, guarda y firma, y Go verifica con la
pública que fue al sitio**. La regla que deja: cuando un lado no puede comprobar su propio resultado, la
comprobación no se omite — **se pasa al lado que sí puede**.

**Lo que no se ha comprobado de la P3, y hay que verlo en su Mac**

- **Crear una llave en GitHub de verdad** y volver a entrar con ella. Es lo que cierra la fase.
- Que el banner de crear **se distingue** del diálogo del navegador y que su texto se entiende: dice que esa
  llave será la forma de entrar en esa cuenta, y eso hay que leerlo en pantalla, no en un fichero.
- Que los sitios **empiezan a ofrecer** llaves de acceso donde antes no lo hacían, que es la consecuencia
  de decir que hay autenticador de plataforma.
- Y **la exportación cifrada con llaves de verdad dentro**, que hasta ahora se ha probado con entradas
  escritas por las pruebas.

## La P4 resultó estar hecha (2026-10-01)

Se planificó como la cuarta entrega —«el segundo navegador»— y al ir a empezarla no quedaba nada que
escribir. Lo que decía el plan que había que hacer eran dos cosas, y las dos estaban:

- **Lo mecánico**, ya hecho en la P2 sin pensarlo: el bloque `world: "MAIN"` se añadió **a los dos
  manifiestos**, `mundo.js` está en `COMPILAR.md` y el único fichero de fuera de `navegador/` que el banner
  importa —`build/icono-barra.svg`— ya iba en `herramientas/fuente-de-la-extension.sh`. La prueba de que
  bastaba es que **Mozilla aprobó la 2.34.0**, y su revisión compila el código fuente y lo compara byte a
  byte con el paquete.
- **Y la incógnita, que era toda la entrega**: «`world: "MAIN"` existe en Firefox desde la 128, pero su
  comportamiento en `document_start` y con `MessageChannel` entre mundos no es necesariamente el de Chrome»,
  y el plan decía que eso **hay que comprobarlo en un Firefox de verdad, no deducirlo**. El cliente lo probó
  el 2026-10-01 con la 2.34.0: entró en GitHub **con el banner de Esfinge**, sin que apareciera ningún
  diálogo del navegador. Funciona igual que en Chrome.

**Y hay que decir cómo se supo, porque no fue probándolo a propósito.** Con la 2.33.0 —la que llevaba el
`toJSON` de solo lectura— Firefox le dijo «Authentication failed» al entrar, y eso **ya demostraba que el
shim se instalaba allí**. Apareció de rebote, preguntando por los números de versión de las dos tiendas, y
mientras tanto en `deuda.md` había una deducción mía —«lo más probable es que allí no haga nada»— que era
falsa por los dos lados: se instala, y funciona.

De ahí la lección de planificación, que es el espejo de la de la P2: **una entrega cuyo contenido es una
incógnita no es una entrega, es una comprobación**, y conviene hacerla antes de reservarle un hueco en el
plan. La P2 se publicó sin poder usarse porque nadie recorrió su procedimiento; la P4 se planificó durante
días y se resolvió con un inicio de sesión.

**Y crear también funciona en Firefox, el mismo día.** Se dejó dicho aquí como lo único que quedaba —son
los dos caminos del `shim` y el de crear tiene más piezas: el banner de crear, el objeto de atestación, los
métodos que el ponyfill del sitio llama—, así que no se dio por bueno porque el otro funcionara. El cliente
borró la llave de GitHub y la volvió a crear **desde Firefox**: salió el banner de Esfinge, bastó con
«Aceptar», y **no apareció ningún diálogo del navegador ni de ningún otro gestor**. Y la usó **desde el otro
Mac**, que era lo último que nadie había tocado: la llave se sincroniza y sirve donde no se creó.

## Una llave la confirma el sitio, no la firma (2026-10-01)

La P3 dejó una deuda conocida y escrita: **crear guarda en la bóveda antes de entregarle la credencial al
sitio** —a propósito, porque lo contrario deja al sitio con una llave que aquí no existe—, así que un registro
que falla **después** deja una huérfana y **el navegador no puede enterarse**: `create()` devuelve la
credencial y el sitio la registra por su cuenta, sin decir nada de vuelta. No es teórico: los dos fallos del
2026-09-30 fallaban justo ahí y el cliente se encontró **cuatro llaves de GitHub** de las que solo una
servía.

Lo urgente se hizo entonces —se distinguen por **cuándo se crearon**, con segundos—, y lo que cierra el
asunto es esto: un campo `confirmada` con la fecha en que **se supo que el sitio la conoce**. Una huérfana no
se confirma nunca, así que se ve sola.

**Y lo que confirma no es firmar, es que el sitio la nombre.** Parecía más natural apuntarlo al firmar —«ha
servido para entrar»— y es peor por dos motivos. El sitio manda en `allowCredentials` los identificadores de
las llaves que tiene registradas para esa cuenta, así que **nombrarla ya es la prueba**, y llega antes:
cuando el banner sale, sin esperar a que nadie acepte. Y si hay varias llaves nuestras para el mismo sitio,
el usuario firma con una y **las demás quedarían sin marcar aunque el sitio las conozca todas**, que es
exactamente el falso negativo que haría borrar la buena. Se marcan **todas las que el sitio nombre**, y por
eso el campo se llama «confirmada» y la ficha dice *«Reconocida por el sitio»* y no «Usada».

Tres consecuencias que hay que decir:

- **Lo escribe el navegador**, que es la primera vez que una lectura de la bóveda provoca una escritura. Se
  salta **si la bóveda está en solo lectura** —lo que la extensión sin cuenta hace con la de la aplicación—
  y un fallo al guardar **no interrumpe la firma**: confirmar es información, entrar es el trabajo.
- **No cuenta como actividad.** Pasa por el camino de `Llaves`, que ya era una lectura, y la regla de la
  casa manda: lo que se repite solo no toca el reloj del autobloqueo.
- **Una llave sin confirmar no se da por mala.** La ficha dice que *el sitio todavía no la ha pedido* y deja
  claro que puede ser simplemente que no se haya entrado aún. Marcarla de rojo sería afirmar algo que no se
  sabe, y lo que se pierde al borrar una llave buena no se recupera.

**Lo que el campo no hace:** no borra nada, no avisa y no ordena la lista. Es un dato más de la ficha, y qué
hacer con una llave que lleva meses sin que el sitio la pida se decide **con el uso**, no ahora.

### Qué se comprobó, y qué dejó claro una mutación

En los dos núcleos, por mutación: que se marca cuando el sitio nombra la llave, que **no se vuelve a
escribir** si ya estaba marcada —si no, cada inicio de sesión sería un guardado y una subida—, y que no se
marca cuando el sitio no nombra ninguna. En Go, `TestUnaLlaveSeConfirmaCuandoElSitioLaNombra`; en TypeScript,
una prueba de comportamiento en `nucleo-fuente.spec.ts`.

**Y esa prueba de TypeScript hubo que escribirla porque las cruzadas no lo cazaban**, lo que es la trampa ya
escrita vista una vez más: quitar `confirmada` de la lista `CAMPOS` del espejo **no pone roja ninguna
cruzada**, porque la forma canónica ordena las claves y el campo vuelve por `extra` con los mismos bytes. Lo
que se rompe no es el formato: es que el campo deja de poder escribirse por su nombre. Las cruzadas comparan
lo que cada lado **escribe**; para esto hace falta comprobar lo que cada lado **hace**.

En la ventana, una prueba de punta a punta guarda una llave sin confirmar y otra confirmada y comprueba lo
que la ficha dice de cada una. Dejó su propia lección, pequeña y repetible: **`conLaBovedaAbierta` no
funciona con una ficha abierta**, porque entonces no hay ni buscador ni botón que esperar, y el síntoma es un
plazo vencido buscando un elemento que no podía estar.

## Lo que el intermitente tenía dentro (2026-10-01)

La prueba de la extensión de verdad falló **dos tandas completas seguidas** de `make comprobar`, con dos
síntomas distintos, mientras **cada fichero por separado pasaba**. El 2026-09-30 un intermitente de la misma
familia se cerró con el saludo bidireccional del puente, así que la tentación era darlo por el mismo flake y
seguir. Dentro había **dos fallos de producto**, y ninguno lo dijo la lectura del código.

**Lo primero fue ponerle voz a la prueba**, porque lo único que dejó la primera caída fue «contó 0»: nada. Al
vencer el plazo, ahora dice si el `shim` está instalado y qué dominios hay apuntados —las dos causas posibles,
distinguibles desde fuera—. La segunda caída cantó **«shim instalado: true · dominios apuntados:
["sitio.prueba"]»**, lo que descartó el puente y señaló la bandera. Y dejó un corolario pequeño y repetible:
el `error-context.md` de la caída se lo llevó la tanda siguiente, así que **de la primera solo quedó el
registro**.

**Uno: una pregunta sin contestar no es una respuesta.** `pedir` no lanza cuando el trabajador de fondo no
contesta —devuelve `ok: false` con un `error` y **sin `motivo`**, porque el motivo lo pone el núcleo y ahí no
ha hablado nadie—, y eso se empujaba como un «aquí no hay nada». La bandera se quedaba en falso **el resto de
la vida de la pestaña**: el `shim` cedía en todas las llamadas y el banner no salía aunque hubiera llaves, sin
un error en ninguna parte. Y no es un caso raro: el trabajador de MV3 **se muere cada pocos minutos**, así que
la primera pregunta de una página puede llegarle dormido. Ahora vive en `bandera.ts` y **insiste solo cuando
no se ha podido preguntar**: una respuesta que dice que ahí no hay llaves no se repite, porque insistir sobre
eso gastaría una pregunta del freno en cada carga de cada sitio, que es casi siempre el caso.

**Dos: la prueba mientras tanto comprobaba el `shim` sin esperarlo.** El otro síntoma —«el shim no se ha
instalado»— **no era el puente**, y eso hay que escribirlo así porque el primer arreglo fue una suposición: se
subió el plazo del saludo y el fallo **volvió igual** en la pasada siguiente. Lo que había era una carrera de
la prueba: miraba si `CredentialsContainer.prototype.get` estaba parcheada **justo después del `goto`**, y el
`shim` se instala cuando el puente tiene acuse, que no puede llegar antes de que el otro lado lea el
consentimiento. Ahora espera. Esperar ahí no tapa nada: lo que esa línea existe para impedir es seguir
adelante **si no se instala nunca**, y eso sigue poniéndola en rojo. Que durante el primer tramo de la página
el `shim` no esté es el diseño —es lo que protege la activación de usuario—.

**Y tres: lo que de verdad tumbaba el banner era el freno, agotado por las pruebas de antes.** Esto se supo
al tercer intento y con la bitácora delante, que es lo único que lo dijo. La secuencia era: la pregunta que
enciende la bandera pasa y contesta `quizas: true`; **ochenta milisegundos después**, la pregunta de firmar
vuelve con `motivo: "demasiado"` —el tope de sesenta preguntas por minuto—, se cede, y el banner no sale. No
lo gastaba esta prueba: lo habían gastado **las anteriores de la tanda**, que rellenan, guardan y mandan
copias; las de las llaves van al final. Y la prueba se envenenaba un poco más, porque pedía la llave en bucle
veinte veces en treinta segundos.

Se arregla por los dos lados. En la compilación de pruebas **los frenos van holgados**, por la misma razón y
con el mismo precedente que los del servidor de cuentas en local: una tanda hace en un minuto lo que una
persona no hace en una hora. **Lo que no se deja de comprobar** es el freno mismo: lo mide
`pruebas/nucleo-fuente.spec.ts`, que importa el núcleo sin pasar por Vite y ve los topes de producción. Y la
prueba deja de llamar en bucle: **espera a que el trabajador diga que ahí hay llave** —por la bitácora, que no
cuesta ninguna pregunta— y después pide la llave una vez.

**Y el plazo se subió igualmente, por lo que se razonó y no por lo que se midió.** Eran **dos segundos**, y lo
que hay al otro lado es un `await` a `storage` que en una máquina cargada pasa de eso: entonces el anuncio del
aislado llega cuando el mundo principal ya ha quitado su oyente y el `shim` no se instala en toda la carga. Es
la misma carrera que el saludo bidireccional vino a arreglar, con el plazo como límite nuevo, y la prueba que
la vigilaba retrasaba al que atiende **150 ms**, o sea dentro del plazo: el caso bueno otra vez. Ahora son
quince segundos, y la prueba nueva espera **más que el plazo viejo** usando el de serie, así que volver a
bajarlo la pone en rojo. **Lo que no hay es una medida de que eso ocurriera aquí**, y así queda dicho.

Esperar ahí no cuesta lo que cuesta esperar en otros sitios: pasa en `document_start`, antes de que nadie
pueda pulsar nada, así que no hay activación de usuario que agotar. **Lo que relaja, y va dicho en el fichero
y aquí**: antes el puerto solo se transfería mientras no existía el primer `<script>` del sitio, y con quince
segundos puede transferirse con la página corriendo. Un anuncio con la marca lo puede forjar la página, así
que **la página puede hacerse con un puerto**. Lo que gana con él es nada que no tuviera: por el puente no
pasa nada secreto, **el origen lo pone el trabajador** con `sender.tab.url`, y firmar exige un clic
`isTrusted` en la sombra cerrada. Como mucho consigue que salga el banner de su propio sitio, que es lo que
consigue llamando a `navigator.credentials.get`.

**Y tres, que salió de paso y es de las que más duelen: `mundo.ts` documentaba que la bandera llega «cada vez
que cambie —al abrirse o cerrarse la bóveda—», y nadie reavisaba nunca.** El comentario está corregido y la
laguna, en `deuda.md`: si la página se cargó con la bóveda cerrada y sin dominios apuntados, abrirla no
enciende la bandera de esa pestaña hasta recargar. No se arregla aquí porque **es una decisión, no un olvido**
—reavisar al volver a la pestaña gasta una pregunta del freno cada vez, casi siempre para oír que ahí no hay
llaves—, y el caso se estrecha solo: con una visita previa con la bóveda abierta, la lista de dominios ya hace
salir el banner.

**La lección de método, que no es de WebAuthn:** un intermitente conocido es la mejor tapadera que tiene un
fallo de verdad. Lo que separó una cosa de la otra no fue volver a correrlo —pasó, y seguía roto— sino
**hacer que la prueba dijera en qué tramo se había quedado**.

Y la segunda mitad, que es la que costó tres pasadas: **tres síntomas no son tres causas, ni una sola.** Se
atribuyeron a dos arreglos y los dos eran suposiciones que sonaban bien —el plazo del saludo y la bandera sin
reintento— mientras la causa de verdad era el freno, y otra la carrera de la prueba. Lo dijo **volver a correr
la tanda después de arreglar**, que es lo que no hay que ahorrarse: con un intermitente, la pasada que importa
es la de después.

Y la tercera, que es la que de verdad cerró el asunto: **cuando lo que falla está dentro de una pieza que no
se puede mirar, se le pone bitácora a esa pieza.** El diagnóstico desde la página llegaba a «el shim está
instalado y los dominios están apuntados», y ahí se acababa; lo que faltaba era **qué contestó el trabajador a
cada pregunta**, y eso solo lo sabe el trabajador. Veinte líneas en `storage.session`, solo en la compilación
de pruebas, y el «demasiado» apareció a la primera. Con ella se cazó después un fallo mío en el aviso nuevo
—la página lo ignoraba porque contaba `sePuedeCrear` como «ya estoy ofreciendo algo», y con la bóveda cerrada
eso vale `true`—, que leyendo el código no salía.

## Y la bandera se refresca sin recargar la página (2026-10-01)

Lo que quedaba abierto del apartado anterior: **la bandera se calculaba una vez, al cargar la página**, así que
quien abriera la bóveda con la pestaña ya abierta se quedaba sin banner hasta recargar. El cliente lo leyó en
`deuda.md` y dijo que le preocupaba, con razón, porque el caso que lo dispara es **el primer inicio de sesión
tras abrir el navegador**: ahí la lista de dominios está vacía —nace vacía y se llena cuando una página
pregunta con la bóveda abierta—, así que el `shim` cede y luego nada vuelve a mirar.

**Lo que se descartó, y por qué.** La primera idea fue que la página volviera a preguntar **al hacerse visible
la pestaña**. Habría funcionado y cuesta una pregunta del freno en cada vuelta a la pestaña, en todos los
sitios y casi siempre para oír que ahí no hay llaves. Se descartó por eso.

**Lo que se hizo.** El refresco del icono **ya pregunta cada minuto si la bóveda está abierta** para poner el
candado, así que el trabajo se cuelga de ahí:

- con la bóveda abierta, **si no hay lista apuntada** se pide una vez y se apunta — la respuesta trae todos los
  dominios de golpe, así que es una pregunta y no una por sitio;
- después, abierta o cerrada, **si en ese sitio hay llave se le dice a la pestaña que vuelva a mirar**. Leer la
  lista no pasa por ningún freno, así que el aviso es gratis;
- y la página **solo vuelve a preguntar si no tenía nada que ofrecer**, con lo que ocurre una vez por pestaña y
  no en cada aviso.

Tres detalles que no son de adorno. **La condición es que falte la lista, no una bandera en memoria**: con una
bandera —«ya la he pedido»— hay un estado que puede no cuadrar con lo guardado, y entonces no se pide lo que
falta; lo cazó la prueba, que borra la lista a mano. **El aviso va al marco principal y a ninguno más**
(`frameId: 0`), que es la trampa de `tabs.connect` otra vez: en un marco de otro origen no se instala nada, así
que no hay a quién avisar. Y **es un mensaje suelto y no un puerto**, al contrario que todo lo demás en esta
extensión, porque aquí no hay respuesta que prometer — que es lo que no se promete igual en los dos
navegadores.

**Lo que no cubre:** si el aviso llega y la bóveda se cierra antes de que la persona pulse, se cede como
siempre. Y el caso de la pestaña abierta **antes** de aceptar el aviso de datos sigue necesitando recargar, que
es lo correcto: aceptar el aviso no debe instalar un `shim` en una página que ya está corriendo.

Lo comprueba una prueba con la extensión de verdad que **empieza por el caso malo** —con la bóveda cerrada y la
lista borrada, el banner no sale—, abre la bóveda, vuelve a la pestaña, hace sonar el reloj del icono y
comprueba que el banner sale **sin recargar**. Mutada por los dos lados: sin el aviso del trabajador y con la
página ignorándolo, se pone roja.

## Dos señales y no una (2026-10-01, corregida el 2026-10-02)

> **Corrección, y es del motivo y no de la decisión.** Esta sección se escribió diciendo que la señal fuerte
> «casi nunca llega», a partir de una ficha del cliente que seguía en blanco después de haber entrado con la
> llave. **Esa observación tenía otra explicación que no se consideró**: la extensión que tenía entonces era
> la 2.34.0, que **no escribía ninguno de los dos campos**. Con la 2.37.0 en su Firefox, al entrar le
> aparecieron **los dos**.
>
> Lo que sí se sostiene es el mecanismo: el sitio nombra la llave **cuando ya sabe quién eres**. Lo que
> falla es el «casi nunca» — en su uso real llega casi siempre, porque entra con usuario y contraseña y
> GitHub le pide la llave después, o sea **como segundo factor**. En el camino de entrar solo con la llave,
> el sitio sigue sin nombrar ninguna y ahí la única señal es `usada`.
>
> **Los dos campos se quedan**, pero por su razón de verdad y no por ésta: dicen cosas distintas y **ninguno
> cubre al otro**, porque cada flujo rellena uno. Y la lección de método, que es la que vale: una ficha
> vacía no prueba que el sitio no nombre la llave; prueba que **nadie lo apuntó**, que es otra cosa — y la
> diferencia entre las dos era una versión de la extensión.

Lo que decidió el apartado «Una llave la confirma el sitio, no la firma» se probó esa misma tarde en el Firefox
del cliente, con la 2.35.0 ya aprobada, y **falló por donde no se había mirado**: entró con su llave de GitHub,
abrió la ficha y seguía diciendo que el sitio no había pedido esa llave. El razonamiento era correcto y la
consecuencia no se había pensado: **en el «entrar con llave de acceso» el sitio no nombra ninguna**, pide
«cualquiera que tengas» —credencial descubrible—, así que la señal fuerte no llega nunca en el flujo que el
cliente usa todos los días. El campo funcionaba como se diseñó y **no servía para lo que existía**.

Con los tres caminos delante, el cliente eligió **dos datos separados**:

| | Qué prueba | Cuándo llega |
|---|---|---|
| `confirmada` | Que el sitio **la tiene registrada**: solo puede pedir por su identificador una credencial que conozca | Cuando nombra la llave en `allowCredentials` — en la práctica, cuando ya sabe quién eres |
| `usada` | Que **alguien la eligió** para firmar. No prueba que el sitio la acepte: con la lista vacía se puede firmar con una huérfana y será él quien la rechace después | En cada firma |

**Lo que se descartó, y por qué importa que esté escrito:** mezclar las dos en un solo campo —marcar «confirmada»
también al firmar— era lo más barato y **habría hecho pasar la señal débil por la fuerte**, que es justo lo que
este campo existía para no hacer. La ficha enseña las dos, cada una con su rótulo, y el aviso de «todavía no has
entrado con esta llave» sale solo cuando **no hay ninguna de las dos**.

`usada` guarda **la última** vez y no la primera, al contrario que `confirmada`: lo útil de una fecha de uso es
la última. El coste es una escritura en la bóveda por inicio de sesión —y su subida—, y por eso no se reescribe
si la fecha no ha cambiado: las fechas del formato tienen resolución de un segundo.

**Y de escribirlo salió un fallo que llevaba medio día dentro de `confirmada`**: se escribía con
`new Date().toISOString()`, o sea **con milisegundos**, mientras Go escribe RFC3339 a segundos. El formato tiene
resolución de un segundo y el núcleo de TypeScript ya tenía su `rfc3339()` y su reloj parable; no se usaron. Las
cruzadas no lo cazaban porque comparan entradas generadas con el mismo valor en los dos lados. Lo caza ahora una
aserción del formato en la prueba de comportamiento, comprobada mutándola.

La lección, que no es de WebAuthn: **una señal fiable que casi nunca llega no es mejor que una señal débil que
llega siempre; lo que no se puede hacer es confundirlas.** Y la de procedimiento, otra vez la misma de esta
fase: el diseño se probó contra el flujo que el cliente usa **después** de publicarlo, y con eso se vio en diez
minutos lo que un día de razonamiento no había visto.
