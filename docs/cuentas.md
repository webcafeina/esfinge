# Cuentas en Esfinge: tu bóveda en todos tus equipos, y compartir copias

## Contexto

Hasta hoy Esfinge es solo local: cada ordenador tiene su bóveda y no hay forma de que dos se hablen. El
cliente quiere **un nivel de cuenta** para dos cosas:

- tener **la misma bóveda en todos sus equipos**;
- poder **mandar una credencial a la cuenta de otra persona** (clientes de Webcafeína) y recibirlas.

Esto cambia lo que el producto es: hoy la portada, la política de privacidad y `docs/seguridad.md`
prometen «sin cuentas y sin servidores» y «Webcafeína no recibe nada». Con cuenta, un servidor nuestro
guarda la bóveda **cifrada de extremo a extremo**. Sigue sin poder leerla, pero ya sabe quién eres y
cuándo te conectas.

### Decidido con el cliente (2026-09-18)

| Pregunta | Elegido |
|---|---|
| Para quién | Webcafeína y sus clientes |
| Registro | **Libre**, pero **solo después de una auditoría de seguridad externa**; hasta entonces, por lista de admisión |
| Servidor | Propio, en **Cloudflare**, con los datos en la **UE** (D1 y Durable Objects con jurisdicción `eu`, comprobado en su documentación) |
| Dirección | `esfinge-cuentas.webcafeina.com` |
| Entrar | **Correo + contraseña maestra**, y **código por correo** en cada equipo nuevo; Touch ID o PIN más adelante |
| Correo | **Resend**, ya verificado en `webcafeina.com` según el inventario de Cronos (§8) |
| Al empezar | **Bienvenida visual al arrancar** por primera vez: «En este ordenador» o «Con cuenta», con ventajas e inconvenientes de cada una. **Reversible en los dos sentidos desde Ajustes** |
| Equipos u organizaciones | **Ninguno**. Cada persona tiene su bóveda |
| Compartir | **Una copia, sin permisos**: al recibirla es de quien la recibe. Si cambia, se vuelve a mandar |
| Recuperar | **Solo con la clave de recuperación personal**. Webcafeína no puede recuperar nada |
| Contraseña con cuenta | **Exigir «fuerte»** con el medidor que ya existe (`cripto.Evaluar`). En local, se avisa como ahora |
| Dos bóvedas en un equipo | **Preguntar cada vez**: «Juntar» o «Quedarme con la de la cuenta», y la local se aparta en una copia, no se borra |
| La extensión con cuenta | **Un cliente más de la cuenta**: entra, se desbloquea con la maestra y **lo hace todo sin la aplicación** —rellenar, códigos, guardar y actualizar—. **Con cuenta va siempre por la cuenta**, aunque la aplicación esté abierta, así que se desbloquea por separado en cada sitio. Sin cuenta, sigue como hoy, por el canal nativo |
| Orden | **Por fases**: primero la bóveda propia sincronizada, luego **la extensión autónoma, antes de la auditoría**, luego compartir, luego Touch ID o PIN |

Una idea que conviene tener presente al leer todo lo demás: **el servidor no puede leer nada, y aun así
hay cosas que no podemos prometer**. Si alguien roba el servidor, puede atacar la contraseña maestra
probando sin conexión, igual que hoy quien robe el fichero. De ahí que se exija «fuerte». Y un servidor
malicioso podría enseñar a un equipo una versión vieja y no la última. Las dos cosas quedan dichas tal
cual en `docs/seguridad.md` y en las ADR.

## Criptografía (fase A, y lo que la B necesita ya)

- **La contraseña de la cuenta es la maestra de la bóveda.** La ranura `maestra`, que ya existe, abre el
  documento descargado. **La ranura `"servidor"` que se dejó prevista se descarta**: una ranura envuelta
  con una clave que guarde el servidor rompería el conocimiento cero.
- **La clave de acceso se deriva aparte de la ranura**:
  - `raiz = Argon2id(maestra, sal_cuenta)`, con la sal elegida por el cliente al registrarse y guardada en el servidor;
  - `claveDeAcceso = HKDF(raiz, "esfinge/cuenta/acceso/v1")`;
  - el servidor guarda `HMAC(PIMIENTA, claveDeAcceso)`.
  
  El cliente **rechaza parámetros de Argon2id por debajo de `cripto.PerfilInteractivo`**, para que un
  servidor malicioso no le haga rebajar el coste.
- **La pre-entrada no revela qué correos existen**: si el correo no existe, la sal es
  `HMAC(SECRETO, correo)`. El registro contesta siempre 202, y el correo que llega dice «ya tienes cuenta»
  si ya la tenías.
- **Segundo factor.** El código por correo se manda **solo después de comprobar la contraseña**: 6 cifras,
  dura 10 minutos y admite 5 intentos. Cada equipo recibe un testigo de confianza de 90 días, que se
  revoca desde Ajustes.
- **En el disco, la sesión va sellada con la clave de bóveda** (el testigo, desde la 2.24.1, en claro: ADR 0037), en `cuenta.json`. Sin la
  maestra, un disco robado no habla con el servidor.
- **Recuperación por posesión.**
  - `posesion = HKDF(clave_de_bóveda, cuentaId, "esfinge/cuenta/posesion/v1")`.
  - La clave de bóveda no cambia nunca, así que este verificador no hay que tocarlo al cambiar la maestra.
  - Sin fichero local, recuperar va así:
    1. Código por correo.
    2. El servidor entrega **solo el sobre de la ranura `recuperacion`**.
    3. La clave de recuperación lo abre.
    4. Con la posesión se demuestra que se tiene la bóveda, y se pone contraseña nueva.
- **Cambiar la contraseña es atómico y va primero al servidor**:
  1. Preparar en memoria.
  2. `PUT /cuenta/clave` con `If-Match`: blob y verificador en una sola transacción, y se revocan las demás sesiones.
  3. Solo entonces se guarda en local.
  
  Sin conexión no se cambia. **Hoy `CambiarMaestraDeBoveda` guarda primero**, y en modo cuenta eso
  desincronizaría las dos.
- **La identidad para compartir se crea al registrarse**, en la fase A: una semilla de 32 bytes dentro
  del cuerpo cifrado (sección nueva `identidad`, que las versiones viejas conservan por `Extra`). De ella
  salen la clave de cifrado (HPKE, que ya viene en la biblioteca estándar de Go 1.27) y la de firma
  (Ed25519).
  - **El conjunto de HPKE lo decide el auditor**: X‑Wing (`MLKEM768X25519`, resistente a cuántica pero borrador) o `DHKEM(X25519)`, RFC 9180. Viaja como campo, así que se puede cambiar.
  - La autenticidad de la clave pública es **TOFU con huella comparable**. Lo que eso no protege —el primer envío si no se comparan huellas— se dice tal cual.

## Sincronización

- **Un blob por cuenta**: el mismo JSON de la bóveda, sin las ranuras locales.
- **Concurrencia optimista**: `GET` con `If-None-Match` y `PUT` con `If-Match`; el servidor asigna `versión + 1`.
- **Si hay conflicto, fusión a tres bandas en el cliente.** La base es la última versión del servidor que
  vio este equipo, en `boveda.esfinge.base`.

### Cambios en el modelo (`internal/boveda`), compatibles con lo que hay

Se mantiene `Formato = 1`, **ESF1 no se toca** y todo lo nuevo son campos opcionales.

- **`Entrada.Revision`**: un contador por entrada que suben `Poner`, `Borrar`, `Restaurar` y
  `BorrarDelTodo`. `cambiada` no se toca: pasarla a nanosegundos rompería la comparación de textos de
  `purgarPapelera`.
- **`Lapidas`** (identificador → fecha), que duran **180 días**, más que la papelera. Las escriben
  `BorrarDelTodo`, `VaciarPapelera` y `purgarPapelera`. Esto matiza la ADR 0026.
- **`sello.Sincro`**: la versión del servidor, autenticada con la clave de bóveda. Sirve para detectar
  retrocesos: el cliente recuerda la última versión vista y rechaza una menor.
- **Apertura en memoria** (`abrirEnMemoria`): **`AbrirBytes` escribe en disco** cuando purga la papelera,
  y con un documento remoto pisaría el fichero local o fallaría con `ErrCambiada`.
- **Ranuras locales** (`llavero-del-sistema`, `pin`, para la fase C): nunca se suben. `PrepararSubida` las
  quita y **vuelve a sellar**; si solo las quitara, el otro equipo daría `ErrManipulada`.

### La fusión (`internal/boveda/fundir.go`)

- **Por entrada, a tres bandas.** Si solo cambió un lado, gana ese lado.
- **Si cambiaron los dos:**
  - Se fusiona campo a campo.
  - Cuando chocan en un mismo campo, gana la mayor `(revision, cambiada, sha256)`. Es un desempate **simétrico**, para que los dos equipos lleguen al mismo resultado.
  - **La contraseña que pierde entra en el historial de la entrada**, así que ningún conflicto pierde una contraseña. Lo que sí se puede perder es una nota editada a la vez en dos sitios, y se dice.
- **Borrado contra edición**: se queda la editada.
- **`SitiosExcluidos`**: se fusiona como conjunto.
- **Ranuras**: gana la más reciente de cada tipo, verificada por el sello.
- **Freno**: una fusión que se llevaría más de la mitad de las entradas no se aplica sola, se pregunta.
  Cualquier fusión que borre algo deja antes `boveda.esfinge.antes-de-fundir`.

### Cuándo se sincroniza (`internal/sincro`)

- Al abrir la bóveda.
- **3 s después de cada guardado**, con un gancho `boveda.AlGuardar` que se dispara fuera del cerrojo.
  Cubre de una vez la ventana, el navegador y el importador.
- Cada 5 minutos con la bóveda abierta (casi siempre un 304).
- Al volver el foco a la ventana, como mucho una vez cada 30 s.
- Al cerrar, vaciando lo pendiente durante 3 s como mucho.

**Nunca llama a `Actividad()`.** El turno se **reserva** con `CompareAndSwap`, no solo se pregunta por
él. `red.SinRed()` la apaga entera.

**Sin conexión, todo sigue en local**, con el estado «Sin conexión, N cambios por subir» y reintentos con
espera creciente.

**Nunca se sincronizan:** el historial (regla absoluta), las preferencias, `navegadores.json` ni los
iconos. Cada equipo baja los suyos.

## El servidor (`servidor/`, TypeScript en Workers)

- **D1 `esfinge-cuentas`, jurisdicción `eu`**: el índice correo → cuenta, la lista de admisión y los
  contadores por IP y día.
- **Un Durable Object `Cuenta` por cuenta, jurisdicción `eu`**: verificadores, sal, versión, sesiones,
  equipos, retos, fallos y registro de eventos. Serializa todo lo de una cuenta, de modo que la
  comparación con intercambio y «blob + verificador + revocar sesiones» van en una sola transacción. Y
  **los frenos por cuenta viven lo que vive la cuenta**, que es la lección del contador por conexión.
- ~~**R2 `esfinge-bovedas`**~~ — **cambiado al escribirlo (ADR 0036)**: la bóveda va **en el propio
  Durable Object, en trozos de 1 MB**, para que comprobar la versión y escribir sean una sola
  transacción. Se conservan las diez últimas versiones y la última de cada uno de los treinta días
  anteriores.
- **Extremos `/v1`:**
  - `salud`, `prelogin`, `registro/{inicio,fin}`, `sesion`, `sesion/codigo`;
  - `boveda` (`GET`/`PUT`), `boveda/versiones`;
  - `cuenta/clave`, `recuperacion/{inicio,codigo,fin}`, `dispositivos`;
  - `cuenta/exportacion` (RGPD) y `DELETE /cuenta`.
  
  En la fase B: `llaves`, `envios`, `buzon` e `invitaciones`.
- **Frenos:**
  - por IP, con el enlace `ratelimit` de Workers;
  - 3 altas por IP y día, más un tope global de altas diarias;
  - por cuenta, en el Durable Object: tras 10 fallos, solo entran los equipos de confianza durante 15 minutos. **Nunca se bloquea la cuenta sin más.**
  - cuotas: 8 MB por blob y 60 subidas por hora;
  - **ningún contador en variables globales del Worker**.
- **`REGISTRO`**: `cerrado`, `lista` o `abierto`, más la tabla `admision`. Abrirlo tras la auditoría es
  un cambio de configuración, sin publicar versión de la aplicación.
- **El correo pasa por una interfaz `Cartero`**, con Resend en producción y un buzón en D1 para las
  pruebas. El código va en el cuerpo y no en el asunto, que se ve con la pantalla bloqueada. **Para abrir
  el registro hace falta el plan de pago de Resend**: el gratuito da 100 correos al día.
- **La ruta de pruebas `/_pruebas/buzon`** existe solo con `ENTORNO=pruebas`, y **una prueba exige 404
  en cualquier otro entorno**.
- **Pruebas**: vitest con `@cloudflare/vitest-pool-workers`. Los frenos se prueban **abriendo una petición
  nueva por intento**.
- **Despliegue**: `.github/workflows/servidor.yml`.
  - Trabajo de comprobación, y otro de despliegue con aprobación en el entorno `produccion`.
  - Migraciones de D1 y `wrangler deploy`, envueltos en `herramientas/reintentar.sh`.
  - En GitHub: `CLOUDFLARE_API_TOKEN` y `CLOUDFLARE_ACCOUNT_ID`.
  - **`PIMIENTA`, `SECRETO_PRELOGIN` y `RESEND_API_KEY` los pone el cliente una sola vez con
    `wrangler secret put`.** Nunca por el chat ni desde el flujo.
  - Habrá un Worker de pruebas aparte.
- **Registros del Worker**: nunca cuerpos, cabeceras `Authorization`, correos ni IP completas.

## La aplicación

- **Refactor previo:** la carpeta de configuración pasa a ser un campo de `App` (`NuevaEn`). Hoy sale de
  `os.UserConfigDir()` en todo el proceso, y sin este cambio no caben dos «equipos» en la misma prueba.
- **`internal/cuenta`**, sin Wails:
  - el cliente HTTP, con `red.SinRed()`, plazos propios y `TLSHandshakeTimeout`;
  - `DerivarAcceso`, que valida los parámetros mínimos, y `Posesion`;
  - los tipos del protocolo.
  
  La raíz del servidor se inyecta con `ApuntarCuentasA`, **como función y no método**, igual que
  `ApuntarAAPI`.
- **`internal/sincro`**: el `Sincronizador`, probado contra un servidor falso.
- **`internal/app/cuenta.go`**. Los métodos nuevos van a `loQuePuedeCruzarElPuente`:
  - `EstadoDeCuenta`, `ElegirModoLocal`, `SincronizarAhora`;
  - `EmpezarRegistro`, `TerminarRegistro`, `EntrarEnCuenta`, `ConfirmarEntrada`, `SalirDeCuenta`;
  - `DispositivosDeCuenta`, `OlvidarDispositivo`;
  - `EmpezarRecuperacion`, `TerminarRecuperacion`;
  - `PedirCodigoParaBorrarCuenta`, `BorrarCuenta`, `ExportarDatosDeCuenta`.
  
  **El reto del segundo factor se queda en Go.** Evento nuevo `sincro`, por el `escuchar()` único de
  `puente.ts`.
- **`cuenta.json` aparte de `preferencias.json`**: las preferencias se guardan como objeto entero, y la
  trampa del cero cambiaría el modo o apagaría la sincronización en silencio. **`BorrarBoveda` borra
  también `.base` y `.antes-de-fundir`.**
- **Métodos que ya existen y cambian en modo cuenta:** `CambiarMaestraDeBoveda` y
  `RotarRecuperacionDeBoveda` (el servidor primero), y `BorrarBoveda`, que pregunta si es «solo de este
  ordenador» o si se borra la cuenta.
- **Interfaz** (`frontend/src/cuenta.tsx`):
  - **`Bienvenida`**, a pantalla completa en el primer arranque sin bóveda. Dos tarjetas, «En este
    ordenador» y «Con cuenta», con sus ventajas e inconvenientes. La segunda lleva «Por invitación»
    mientras el servidor diga `lista`. **Es la primera pantalla con peso visual de marca**, así que se
    diseña con la ADR 0021: nada fuera de los tokens medidos, y se mira en capturas en los dos temas.
  - **`Registro`**: correo, código, maestra con la exigencia de «fuerte», y la `Ceremonia` de la clave
    de recuperación, reutilizada de `boveda.tsx`.
  - `Entrar`, `Recuperar`, y el indicador de sincronización en la barra de la Bóveda.
  - Si en ese equipo ya había otra bóveda, la pregunta «Juntar» o «Quedarme con la de la cuenta».
  - **Ajustes**, grupo «Cuenta y sincronización»: modo, correo, última sincronización, «Sincronizar
    ahora», equipos, «Pasar a solo este ordenador» y, dentro del grupo peligroso, «Borrar cuenta».
  - Quien ya tiene bóveda sigue en local y ve un aviso de la novedad, una sola vez.

## La extensión, cliente de la cuenta (fase E)

Con cuenta, la extensión deja de pedirle nada a la aplicación: tiene su propia sesión, baja el blob, lo
abre y lo sube. **Es una segunda implementación del formato de la bóveda y de la fusión**, y de ahí
salen casi todas las reglas de esta sección.

### Una sola especificación y dos implementaciones que se vigilan entre sí

- **`docs/formato-boveda.md`**: el JSON exterior, `sobres`, `sello` (cómo se calcula), `cuerpo`, las
  secciones del contenido, `Revision`, `Lapidas`, `sello.Sincro` y las reglas de la fusión. Hasta ahora
  el formato vivía solo en el código Go. Con dos implementaciones, el documento es el contrato.
- **ESF1 en TypeScript** (`navegador/src/cripto/`): Argon2id con **`hash-wasm`** y XChaCha20‑Poly1305 con
  **`@noble/ciphers`**, porque WebCrypto no trae ninguno de los dos. **Se prueba contra los vectores
  fijos que ya existen** en `internal/cripto/testdata/` (abrir `texto.esf1` y sellar con la misma sal y
  el mismo nonce da los mismos bytes), sin copiarlos ni regenerarlos. Es la misma promesa de la ADR 0022,
  ahora en dos lenguajes.
- **La fusión, con casos compartidos**: `internal/boveda/testdata/fusion/*.json` (base, local, remoto →
  esperado), escritos a mano y leídos por las pruebas de Go y por las de TypeScript.
- **La prueba que cierra las dos**, en `make comprobar`: Go genera escenarios al azar, un guion de Node
  los fusiona con el código de la extensión, y se exige **el mismo resultado byte a byte** tras
  canonicalizar. Y un viaje de ida y vuelta: una bóveda que escribe TypeScript la abre Go, y al revés.
  Sin esto, dos fusiones «probadas» por separado pueden no estar de acuerdo, y el resultado sería el
  bucle de ida y vuelta entre equipos del riesgo número uno.
- **El código de un solo uso en TypeScript** (HMAC‑SHA1 de WebCrypto), probado con los vectores del RFC
  6238 **y contra el de Go** con la misma semilla.

### Cómo vive la bóveda dentro del navegador

- **El panel** gana «Entrar con tu cuenta de Esfinge» (correo, contraseña, código por correo) y
  «Desbloquear». Sin cuenta, sigue el camino de hoy por la aplicación.
- **La clave de bóveda abierta va solo en `storage.session`**: en memoria, nunca en disco, y los guiones
  de las páginas no pueden leerla (nivel de acceso por defecto, solo contextos de confianza). Se pierde
  al cerrar el navegador. **El blob cifrado, la base de la fusión y el testigo sellado** van en
  `storage.local`, igual que en disco hace la aplicación, para poder desbloquear sin conexión.
- **Bloqueo propio por inactividad**, con `alarms`, y **solo lo que hace una persona cuenta como
  actividad**: rellenar al pulsar, guardar o abrir el panel. El refresco del icono, la sincronización
  periódica y el relleno automático no cuentan. Es la misma regla que en la aplicación.
- **Sincronización**: al desbloquear, tras cada guardado y cada 5 minutos con `alarms`, con el turno
  reservado y sin llamar a la actividad.
- **Argon2id de 64 MiB, dos veces**, al entrar: cerca de un segundo en WebAssembly. Va en el trabajador
  de fondo, que sigue compilado en una sola pieza. **El WebAssembly exige `'wasm-unsafe-eval'` en la
  CSP del manifiesto**, en los dos navegadores, y `permisos.mjs` lo vigilará como vigila `storage`.
- **La extensión cuenta como un equipo más** («Chrome en MacBook»), con su testigo de confianza, y se ve
  y se revoca desde Ajustes de la aplicación.
- **Con la aplicación abierta a la vez**: cada una va por su lado y la fusión las pone de acuerdo, como
  a dos ordenadores.

### Lo que cambia hacia fuera

- **La extensión se conecta a internet**, a `esfinge-cuentas.webcafeina.com`. `https://*/*` ya lo cubre,
  pero las promesas no:
  - la ficha y la política dicen hoy «habla solo con Esfinge en tu ordenador»;
  - `data_collection_permissions` de Firefox y las «Privacy practices» de Chrome se rehacen;
  - **`VERSION_DEL_AVISO` sube** y el aviso del panel cuenta el modo con cuenta;
  - las dos tiendas vuelven a revisar.
- **Mozilla revisa también las bibliotecas nuevas**: van en `package.json` con su bloqueo y entran en
  la compilación reproducible, que se sigue comprobando byte a byte.
- **`docs/seguridad.md`**: la bóveda abierta dentro del navegador es lo más expuesto de todo el
  producto. Una extensión maliciosa con permisos amplios o un fallo del navegador llegan ahí, y no a la
  aplicación. Lo que se hace y lo que no se puede hacer contra eso se dice tal cual.

## Promesas y documentación que cambian

- **README**, `web/index.html` y `web/privacidad.html`:
  - «Sin cuentas y sin servidores» y «Webcafeína no recibe nada» pasan a «local, o con cuenta cifrada de extremo a extremo».
  - La política cuenta qué se trata, con qué base, cuánto se guarda, quién lo procesa (Cloudflare y Resend, datos en la UE) y los derechos.
  - **Sirve igual para usuarios de fuera de la UE.** Hay que comprobar en la fuente lo que dice Colombia (Ley 1581 de 2012 y la lista de países adecuados de la SIC) antes de abrir a clientes de allí.
- **Condiciones de uso nuevas** (sin recuperación posible, bajas, responsabilidad), acuerdos de encargo de
  tratamiento con Cloudflare y Resend, y el registro de actividades. **Necesitan revisión legal** antes
  de abrir el registro.
- **`docs/seguridad.md`**:
  - «dos salidas» pasa a «tres con cuenta»;
  - qué ve el servidor (correo, IP, horas, tamaños, equipos y, en la fase B, con quién compartes);
  - el ataque sin conexión al blob, la congelación y lo que no cubre TOFU;
  - los ficheros nuevos del disco.
- **Extensión**: desde la fase A2, lo que guarda por la aplicación acaba en el servidor, cifrado. Desde
  la fase E se conecta ella misma (ver su sección). El aviso del panel, las fichas y las declaraciones de
  las tiendas se cambian **una sola vez, en la E**, con `VERSION_DEL_AVISO` en 2. En la A2 basta una
  frase en la política de la web.
- **ADR:**
  - 0035, cuentas y servidor en Cloudflare UE, con la puerta de la auditoría (escrita);
  - 0036, el servidor de cuentas (escrita, con la A0);
  - 0037, derivación, acceso, segundo factor y recuperación por posesión;
  - 0038, sincronización y fusión;
  - 0039, bienvenida y local ↔ cuenta (escrita);
  - 0040, la extensión como cliente de la cuenta (fase E): dos implementaciones del formato, la bóveda
    abierta dentro del navegador y «siempre por la cuenta»;
  - 0041, identidad y copias (fase B);
  - 0042, desbloqueo con el sistema (fase C).
  
  Matizan la 0014 (tercera conexión), la 0023 (la ranura `servidor` descartada), la 0024 (los iconos no
  se sincronizan), la 0026 (el plazo de las lápidas) y la regla del historial.
- `CLAUDE.md` (decisiones y trampas nuevas), `estado.md`, `deuda.md`, `sesiones.md` y la memoria.

## Entregas

| Entrega | Qué lleva | Cómo se comprueba |
|---|---|---|
| **A0** (solo servidor) · **hecha el 2026-09-18** | `servidor/` en el Worker de pruebas: D1 y Durable Objects en la UE (ADR 0036); correo real; flujo de despliegue. **El cliente**: token de Cloudflare, las bases D1 en la UE y los secretos (el DNS de `webcafeina.com` ya está en Cloudflare, comprobado) | vitest; `curl` contra el Worker; correo recibido en Gmail con DKIM alineado |
| **A1** · **hecha el 2026-09-18, sin publicar** | Refactor de la carpeta, `Revision`, `Lapidas`, `sello.Sincro`, fusión, apertura en memoria, `internal/cuenta` e `internal/sincro`. **Sin interfaz**, invisible para el usuario | Pruebas de **convergencia con tres equipos simulados** y operaciones al azar; una bóveda de la 2.22.2 en `testdata` se abre y conserva todo; la tubería con dos `App` contra un servidor falso |
| **A2** · **hecha el 2026-09-18, sin publicar** (ADR 0039) | Bienvenida, registro, entrada con código, sincronización, identidad creada. Servidor con `REGISTRO=lista` (solo la casa) | **e2e de dos equipos**: dos `cmd/dev` y dos Vite contra `wrangler dev` local. A crea la cuenta y guarda; B entra y la ve; los dos editan a la vez y la contraseña perdedora sale en el historial; A borra y en B desaparece. **En el Mac: dos máquinas reales** |
| **A3** · **hecha el 2026-09-21** (2.24.0) | Cambio de contraseña, recuperación, equipos, borrado, exportación, cuenta → local, restaurar una versión | e2e de cada flujo; ensayo de recuperación con la clave en papel |
| **E**, 2.26.0 y extensión | La extensión, cliente de la cuenta: ESF1, fusión y TOTP en TypeScript, entrar, desbloquear, bloquear, sincronizar, guardar y actualizar; fichas, aviso y privacidad al día | Vectores de ESF1 en los dos lenguajes; fusión cruzada Go ↔ TypeScript byte a byte; Playwright con la extensión de verdad contra `wrangler dev` (**salda la deuda alta de las pruebas con la extensión cargada**); en el Mac: rellenar y guardar con la aplicación cerrada, y un cambio en el navegador que aparece en la aplicación |
| **Auditoría** | Diseño, Worker, aplicación, **extensión** y textos | Informe y correcciones |
| **A4** (solo configuración) | `REGISTRO=abierto`, con la web, la privacidad y las condiciones publicadas antes y Resend de pago | Revisión legal |
| **B** | Contactos, huellas, envío y recepción de copias, invitaciones | e2e con dos cuentas |
| **C** | Touch ID o Windows Hello y PIN, como ranuras solo locales | Solo en Mac y Windows reales. **Puede exigir firmar la aplicación**, en contra de la decisión de no firmar: hay que investigarlo antes |

**La primera acción concreta es la A0**, porque todo lo demás se prueba contra ella: crear `servidor/`
con sus pruebas, y pedir al cliente el token de Cloudflare y los tres secretos.

## Verificación

**Aquí:**

- `make comprobar` (que gana `vitest` del servidor), `make e2e` (que gana los dos equipos) y `make contraste`.
- **Convergencia por propiedades**: tres equipos, operaciones y órdenes de sincronización al azar. Se
  exige que converjan y que ninguna contraseña se pierda salvo borrado explícito. **Es el riesgo número
  uno**: un error de fusión se propaga a todos los equipos.
- Una bóveda de la 2.22.2 en `testdata`: se abre, se sincroniza, y la abre una versión vieja.
- Pruebas que vigilan las trampas:
  - la sincronización no llama a `Actividad()`;
  - `SinRed` la apaga;
  - `AbrirBytes` no se usa con documentos remotos;
  - las ranuras locales no suben;
  - `BorrarBoveda` no deja restos;
  - la ruta de pruebas del servidor da 404 en producción;
  - los parámetros de Argon2id rebajados se rechazan.
- Capturas de la bienvenida y de Ajustes en los dos temas, **miradas**.

**Solo lo puede comprobar el cliente o el auditor:**

- La sincronización entre dos Mac reales, con reposo, cambios de red y cierre con cambios pendientes.
- Que los correos lleguen a la bandeja y no al spam.
- La bienvenida bajo el vidrio.
- La fase C entera.
- Que las tiendas aprueben la extensión con conexión propia y WebAssembly.
- **El auditor**: la bóveda abierta dentro del navegador, el conjunto de HPKE, la recuperación por posesión, la fusión frente a un servidor
  malicioso, los frenos y una prueba de intrusión de la API.
