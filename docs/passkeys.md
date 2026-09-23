# Passkeys: que Esfinge sea la llave

**Qué pidió el cliente (2026-09-23)**, con sus palabras: «la opción que tiene Dashlane del passkey, como
por ejemplo al acceder a GitHub, que simplemente sale un banner de passkey que reconoce Dashlane y es
darle a Aceptar».

Es decir: que al entrar en un sitio que admite passkeys, **la llave la ponga Esfinge** —guardada en la
bóveda, sincronizada entre equipos— en vez del llavero del sistema o una llave física. Sin contraseña y
sin código.

Esto **no es una tarea, es una fase**, del tamaño de las cuentas. Este documento dice por qué, qué habría
que construir y qué hay que decidir antes de empezar. **Nada de esto está hecho.**

## Lo que ya se ha comprobado

Investigado el 2026-09-23 en la documentación de los navegadores, para no volver a empezar de cero:

- **En Chrome no hay una API de extensión para ser proveedor de passkeys.** Hay una que lo parece,
  [`chrome.webAuthenticationProxy`](https://developer.chrome.com/docs/extensions/reference/api/webAuthenticationProxy),
  pero está hecha para **escritorio remoto**: mientras una extensión está «attached», **se suspende todo el
  procesamiento normal de WebAuthn** en ese navegador y solo puede haber una extensión a la vez. Usarla
  significaría dejar sin servicio al Touch ID del propio usuario y a sus llaves físicas mientras Esfinge
  esté conectada. No es el camino.
- **Lo que hacen los gestores de verdad —1Password, Bitwarden, Dashlane— es reemplazar
  `navigator.credentials`** en el **mundo principal** de la página, con un accesor no configurable para
  que la propia página no pueda volver a cambiarlo. Es un apaño, y es la práctica de la industria.
- **Los dos navegadores lo permiten.** Chrome con `world: "MAIN"` en los guiones de contenido, y
  **Firefox desde la 128** ([MV3 en Firefox 128](https://blog.mozilla.org/addons/2024/07/10/manifest-v3-updates-landed-in-firefox-128/));
  la extensión ya pide `strict_min_version: 140.0`, así que no deja a nadie fuera.
- **En Chromium, solo una extensión a la vez** puede atender la interfaz condicional de passkeys: si hay
  otro gestor instalado, el usuario tiene que elegir cuál manda.

## Por qué es grande

Tres cosas, y ninguna es la parte criptográfica:

**1 · Hay que ejecutar código nuestro en el mundo principal de cada página `https`.** Hoy el guion de
Esfinge vive en el mundo aislado: la página no lo ve ni lo puede tocar. Un shim de `navigator.credentials`
vive **dentro** de la página, donde cualquier script puede leerlo e intentar engañarlo, y donde un fallo
nuestro no rompe el relleno: **rompe el inicio de sesión del sitio**, incluido el de quien no use Esfinge
para esa cuenta. Es un salto de riesgo mayor que el de la ADR 0028, y esa ADR se escribió precisamente
para no dibujar de más en la página de otro.

**2 · Una passkey es una clase de secreto nueva en la bóveda**, y la bóveda existe **dos veces** (ADR
0040). Cada passkey guarda el identificador de la credencial, el dominio del sitio, el identificador del
usuario, **la clave privada** y un contador de firmas. Eso significa: formato nuevo en Go y en
TypeScript, reglas de fusión, `vaciarLoSensible`, `SinSecretos`, la lista blanca del puente, el
importador, la exportación, la papelera, la interfaz de la bóveda y las pruebas cruzadas. El contador de
firmas además **no se puede fusionar bien** entre equipos —dos pueden incrementarlo a la vez—, y la salida
habitual es no usarlo nunca, lo que hay que decidir a conciencia.

**3 · El «banner» es la primera vez que Esfinge sustituye un diálogo del navegador.** Hoy dibuja un filete,
un aviso de tres segundos y una tarjeta que se pulsa (ADR 0028 y 0032). Esto es otra cosa: es la pantalla
donde alguien decide identificarse, y tiene que ser **imposible de confundir** con una de la página.

## Lo que no es problema

- **La criptografía es la parte fácil y ya está medio hecha**: P-256 con `ECDSA` de WebCrypto, que el
  navegador trae. Hace falta CBOR y COSE para el objeto de atestación, y con atestación `none` —que es lo
  que usan los gestores— eso son unas pocas decenas de líneas.
- **El almacén ya existe**: la bóveda cifrada, sincronizada y con clave que no sale del equipo es
  exactamente lo que una passkey necesita.

## Lo que hay que decidir antes de empezar

1. **¿En los dos navegadores o solo en uno?** El shim es el mismo, pero doblar la superficie de prueba en
   algo que puede romper inicios de sesión ajenos no es gratis.
2. **¿Qué pasa con el Touch ID del propio usuario?** Si Esfinge se ofrece para todo, compite con el
   llavero del sistema. Lo razonable es ofrecerse **solo cuando la bóveda tenga una passkey de ese sitio**
   o cuando el usuario lo pida, y no siempre.
3. **¿Se pide la contraseña maestra al usar una passkey?** Una passkey sin desbloquear es una llave sin
   dueño: quien se siente delante de un navegador con la bóveda abierta entra en todo. Dashlane resuelve
   esto con su propio desbloqueo.
4. **El contador de firmas**: dejarlo siempre a cero, que es lo que hace todo el mundo, o intentar
   mantenerlo y aceptar que la sincronización lo estropee.
5. **Las tiendas.** Un shim sobre `navigator.credentials` con acceso a todas las páginas es de lo más
   revisado que hay. Conviene contarlo en las notas de revisión antes de que lo pregunten.

## Cómo lo partiría

- **P1 · La passkey en la bóveda, sin navegador.** Clase nueva en las dos implementaciones, con su fusión,
  sus pruebas cruzadas y su sitio en la ventana. Invisible para el usuario y sin riesgo en páginas ajenas.
- **P2 · Usar una passkey que ya existe** (`navigator.credentials.get`), en un solo navegador, con el
  banner y con el desbloqueo que se decida. Es lo que pidió el cliente: entrar en GitHub y aceptar.
- **P3 · Crear passkeys** (`navigator.credentials.create`), que es lo que ata al usuario a Esfinge y lo
  que más cuidado pide: una passkey creada aquí y perdida es una cuenta perdida.
- **P4 · El segundo navegador**, y lo que salga del uso.

**Antes de la P2 hay que probar con las páginas de verdad** —GitHub, Google, Cloudflare— igual que se hizo
con el relleno: leyendo lo que la página pide, no adivinándolo.
