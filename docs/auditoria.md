# La auditoría externa

La [ADR 0035](adr/0035-las-cuentas.md) puso una puerta: **el registro libre, solo después de una auditoría
de seguridad externa**. Hasta entonces las cuentas son por invitación. Este documento es lo que se le
entrega a quien la haga, y sirve igual para pedir presupuesto.

Está escrito para alguien de fuera: no da por sabido nada del proyecto.

## Qué es Esfinge, en un párrafo

Un gestor de contraseñas de escritorio (Go + Wails, macOS, Windows y Linux) con una extensión de navegador
(Chrome y Firefox) y, desde la 2.23.0, **cuentas**: la bóveda cifrada se guarda en un servidor propio en
Cloudflare (Worker + Durable Objects + D1, jurisdicción `eu`) para tenerla en varios equipos. El servidor
**no puede leer nada**: todo va cifrado con una clave que se deriva de la contraseña maestra, y de ella
solo sale un verificador. Lo usa una empresa pequeña —Webcafeína— para sustituir a Dashlane.

## Lo que hay que mirar, por orden de riesgo

Este orden es nuestro y es una petición, no una limitación: si algo de más abajo resulta más grave, manda
lo que se encuentre.

1. **El servidor y el protocolo de cuenta** (`servidor/`, [ADR 0036](adr/0036-el-servidor-de-cuentas.md) y
   [0037](adr/0037-claves-de-la-cuenta.md)). La promesa es conocimiento cero: clave de acceso derivada
   aparte con Argon2id, verificador con HMAC y pimienta, segundo factor por correo, testigos de confianza,
   frenos por IP y por cuenta, recuperación por posesión de la bóveda. **Preguntas concretas**: ¿puede el
   servidor —o quien lo controle— leer, sustituir o degradar una bóveda sin que el cliente lo note?
   ¿Sirven los frenos para lo que dicen servir? ¿Se puede enumerar correos con `prelogin` o con el alta?
2. **La extensión con cuenta** (`navegador/`, [ADR 0040](adr/0040-la-extension-cliente-de-la-cuenta.md)).
   Es lo más expuesto: la bóveda **abierta** vive en el navegador, con la clave en `storage.session`, y hay
   código nuestro en cada página `https`. **Preguntas**: ¿puede una página llegar a la clave, a una
   contraseña o a los verbos de la cuenta? ¿Aguanta el reparto de origen —`sender.tab.url`, tramas,
   `frameId`— lo que promete? ¿Es sensata la decisión de guardar la bóveda cifrada en `storage.local`?
3. **El formato y la criptografía** (`internal/cripto`, `internal/boveda`, `navegador/src/nucleo`,
   [ADR 0022](adr/0022-vectores-fijos.md), [0023](adr/0023-la-boveda.md) y `docs/formato-boveda.md`).
   `ESF1` es XChaCha20-Poly1305 con Argon2id y una jerarquía de claves: la maestra abre un sobre que
   guarda la clave de bóveda. **Hay dos implementaciones** —Go y TypeScript— que se vigilan con pruebas
   cruzadas. **Preguntas**: ¿los parámetros, el uso del nonce y lo autenticado son correctos? ¿Vale el
   sello para lo que se usa —detectar un cuerpo revertido o una ranura quitada—?
4. **La sincronización y la fusión** ([ADR 0038](adr/0038-sincronizar-la-boveda.md)). Es donde un fallo se
   propaga a todos los equipos. **Preguntas**: ¿se puede perder una contraseña sin que nadie la borre?
   ¿Puede un servidor malicioso provocar una fusión que borre, resucite o reordene a su gusto?
5. **El canal con el navegador sin cuenta** ([ADR 0027](adr/0027-el-canal-con-el-navegador.md) y
   [0028](adr/0028-rellenar-en-la-pagina.md)): un socket de dominio unix, apagado de fábrica, con
   emparejamiento y frenos. **Pregunta**: ¿qué puede hacer otro programa de la misma máquina?
6. **Las actualizaciones** ([ADR 0014](adr/0014-comprobacion-de-actualizaciones.md) y
   [0016](adr/0016-actualizarse-sola.md)): se descargan de GitHub y se comprueba el SHA256, y la
   aplicación **se reemplaza a sí misma** en macOS y Windows. Sin firmar ni notarizar, que es una decisión
   del cliente. **Pregunta**: ¿qué haría falta para colar un binario?
7. **Los textos**: `web/privacidad.html`, el aviso de datos de la extensión y `docs/seguridad.md`.
   Auditar aquí es comprobar que **lo que se promete es lo que se hace**. Una promesa de más es un fallo.

## Lo que se promete y lo que no

Está entero en [`docs/seguridad.md`](seguridad.md) y es lo primero que conviene leer. En corto, lo que
**no** se promete y ya está dicho ahí:

- Con la bóveda abierta, la memoria del proceso tiene las contraseñas; no hay defensa contra quien pueda
  leerla.
- Una bóveda abierta entrega la contraseña **y el segundo factor**, porque Esfinge calcula los códigos.
- Quien robe el fichero —o el servidor— puede **probar contraseñas sin conexión**: la defensa es una
  maestra fuerte, y con cuenta se exige.
- La papelera guarda treinta días lo borrado, y el historial, las contraseñas anteriores.
- Ir a por el icono de cada sitio **reparte** la lista de sitios en el DNS y en el saludo TLS.
- El testigo de «este equipo es de confianza» va **en claro** en el disco (ADR 0037), y dice por qué.
- La extensión con cuenta copia desde su panel y eso **no se borra solo** del portapapeles.

## Qué se entrega

- **El código, entero y público**: <https://github.com/webcafeina/esfinge>. No hace falta acuerdo de
  confidencialidad para mirarlo; si el informe lo necesita, se firma.
- **La documentación viva**: las ADR de `docs/adr/` —cuarenta fichas con el porqué de cada decisión, sus
  alternativas descartadas y **qué se comprobó y qué no**—, `docs/seguridad.md`, `docs/formato-boveda.md`,
  `docs/cuentas.md` y `docs/deuda.md`, que es la lista de lo que sabemos que está a medias.
- **Versiones compiladas** de cada publicación, en las publicaciones de GitHub, y la extensión en las dos
  tiendas.
- **Un servidor de pruebas aparte del de producción** (entorno `pruebas` de `servidor/wrangler.jsonc`, en
  `workers.dev`), con su base y sus datos, para que se pueda romper sin tocar nada real.
- **Una cuenta de prueba** en ese servidor, creada para la auditoría. **Nunca la del cliente.**
- Todo corre en local sin pedir nada a nadie: `make comprobar` levanta el Worker de verdad con
  `wrangler dev` y pasa las pruebas; `make e2e` mueve la interfaz; `navegador/pruebas-reales` carga la
  extensión en un Chromium de verdad.

## Lo que ya está comprobado aquí

Para no gastar su tiempo en lo que ya tiene red. Nada de esto sustituye a mirarlo: dice dónde hay menos
probabilidad de encontrar algo por accidente.

- **El formato está congelado con vectores fijos** que no se regeneran (ADR 0022): se comprueba que lo
  grabado se abre y que sellar con la misma sal y el mismo nonce da los mismos bytes.
- **Las dos implementaciones del formato se cruzan entre sí** en cada publicación: la misma bóveda, los
  mismos códigos y **4.000 fusiones al azar** tienen que dar los mismos bytes en Go y en TypeScript.
- **La fusión tiene una prueba de tres equipos al azar** que exige convergencia y que no se pierda ninguna
  contraseña salvo borrado explícito.
- **El cliente y la sincronización hablan con el Worker de verdad** levantado en local, no con un servidor
  de mentira.
- **La extensión se prueba cargada de verdad** en Chromium, contra ese mismo servidor.
- **Qué campo se rellena** está probado en páginas escritas a mano, y la mitad de la tabla son casos donde
  lo correcto es no rellenar nada.
- **`ESFINGE_SIN_RED` apaga todas las salidas a la red**, con una prueba por cada salida.
- **Los contrastes de color** de los dos temas se miden en cada compilación (no es seguridad, pero explica
  por qué hay tanta prueba de interfaz).

## Lo que sabemos flojo, y nos gustaría que mirarais igual

`docs/deuda.md` lo tiene entero, con severidades. Lo que toca a seguridad:

- **La política de privacidad no ha pasado revisión legal**, y con cuenta hay tratamiento de datos de
  terceros (Cloudflare y Resend). Va antes de abrir el registro.
- **La pimienta del servidor no se puede rotar** hoy: cada cuenta guarda con qué versión se firmó, pero no
  hay procedimiento de rotación.
- **El correo se normaliza con dos minúsculas distintas** —Go y el Worker— y podrían discrepar en casos
  raros de Unicode; manda el servidor.
- **Sin saber quién entra, ofrecer actualizar** una cuenta del sitio es una heurística, y en dos pasos
  (Google) se resolvió recordando el usuario escrito cinco minutos en memoria del trabajador de fondo.
- **La lista de sufijos públicos son dos copias con fechas distintas** (Go y `tldts`): un sufijo muy
  reciente podría estar en una y no en la otra.
- **Nadie ha ejecutado la aplicación en Windows ni en Linux** todavía.
- Lo de la fase B —compartir copias con HPKE y confianza al primer uso— **no está escrito**: si podéis
  opinar sobre el diseño de `docs/cuentas.md` antes de construirlo, mejor.

## Qué esperamos del informe

- Hallazgos con **severidad y cómo reproducirlos**, y lo que haríais vosotros en nuestro lugar.
- Una opinión explícita sobre **la promesa de conocimiento cero** y sobre **si abrir el registro es
  razonable** con lo que hay.
- Si algo está bien, decirlo también: sirve para escribir la verdad en la web.

Lo que haremos con él: **corregir lo que salga antes de abrir el registro** (A4) y dejar en
`docs/seguridad.md` lo que no se pueda corregir. Publicar el informe o no es decisión del cliente.

## Lo que hace falta antes de empezar

1. **Elegir quién la hace y cerrar presupuesto** (decisión del cliente). Sirve enseñarles este documento.
2. **Decidir el alcance**: lo mínimo defendible es el servidor, el protocolo de cuenta y la extensión;
   el máximo, los siete puntos de arriba más los textos.
3. **Crear la cuenta de prueba** en el servidor de pruebas, con un correo que no sea el del cliente.
4. **Los secretos no pasan por aquí**: si hace falta darles algo, lo pone el cliente donde toque.

### Cómo pedir presupuesto

Un correo con: qué es el producto (el primer párrafo de este documento), el enlace al repositorio, este
documento, el tamaño aproximado —unas 54.000 líneas entre Go, TypeScript y el Worker, pruebas incluidas—, el alcance que se
quiera y las fechas. Conviene pedir **quién lo va a mirar** y si entregan revisión de las correcciones.

Hay casas que publican informes de gestores de contraseñas y de extensiones de navegador; la elección y el
precio los cierra el cliente. **Antes de escribir a nadie, comprobar en su web lo que hacen hoy**: esto
cambia cada año y no se decide de memoria.
