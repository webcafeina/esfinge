# ADR 0037 — Las claves de la cuenta: acceso derivado aparte y posesión de la bóveda

**Fecha:** 2026-09-18 · **Estado:** aceptada, sin interfaz todavía · **Continúa la [0035](0035-las-cuentas.md)**
y la [0036](0036-el-servidor-de-cuentas.md) · **Revisar cuando** llegue la auditoría

## Contexto

Con cuenta, el servidor tiene que saber si quien entra es el dueño **sin ver nunca la contraseña
maestra ni la clave de la bóveda** (ADR 0035). Y tiene que poder dejar cambiar la contraseña a quien la
ha olvidado pero tiene su clave de recuperación, sin que Webcafeína pueda recuperar nada por su cuenta.

## Decisión

**La contraseña de la cuenta es la maestra de la bóveda.** Una sola contraseña, y la ranura `maestra`
de siempre abre la bóveda que baja del servidor. La ranura `servidor` que la ADR 0023 dejó prevista se
descarta: una ranura envuelta con una clave que guarde el servidor rompería el conocimiento cero.

**La clave de acceso se deriva aparte** (`cuenta.DerivarAcceso`):

    raíz  = Argon2id(maestra, sal de la cuenta, coste)
    clave = HKDF-SHA256(raíz, «esfinge/cuenta/acceso/v1»)

- **La sal es de la cuenta**, 16 bytes que elige el equipo al darse de alta, y no la de la ranura de la
  bóveda: aunque la contraseña es la misma, las dos derivaciones no tienen nada que ver.
- El servidor guarda `HMAC(pimienta, clave)`. Robar solo su base no permite probar contraseñas.
- **El coste lo dice el servidor, y el cliente no acepta menos que el de siempre**
  (`cripto.PerfilInteractivo`: 64 MiB, 3 pasadas) ni más que el tope de un contenedor. Un servidor que
  pidiera una pasada y 8 MiB haría la clave de acceso mucho más fácil de atacar que la ranura.

**La posesión demuestra que se tiene la bóveda abierta** (`Boveda.Posesion`):

    posesión = HKDF-SHA256(clave de bóveda, «esfinge/cuenta/posesion/v1»)

La clave de bóveda **no cambia nunca**, ni al cambiar la maestra ni al rotar la de recuperación. Por eso
sirve para cambiar la contraseña de la cuenta tanto a quien la sabe como a quien entra con su clave de
recuperación, y el servidor no tiene que tocar su verificador en ninguno de los dos casos.

**El correo se normaliza igual en los dos lados**: sin espacios alrededor, NFC y minúsculas. Nada de
quitar puntos ni lo que va tras un «+», que es una regla de Gmail y uniría cuentas de personas distintas.

## Alternativas descartadas

- **Posesión atada a la cuenta**, con el identificador de la cuenta como sal del HKDF, que era el plan.
  Al escribirlo salió que **al darse de alta el equipo todavía no conoce ese identificador**: lo asigna el
  servidor en el mismo paso. Y no hace falta: la clave de bóveda ya es única de cada bóveda.
- **Mandar la contraseña y que el servidor la derive**, como hace casi cualquier web. Es justo lo que el
  conocimiento cero descarta.
- **OPAQUE u otro protocolo sin verificador**, que protegería también de un servidor que mirara las
  claves de acceso al llegar. Es mejor, y es más código criptográfico propio sin auditar; con el HMAC y
  una derivación de coste alto, lo que llega al servidor ya es inútil para abrir la bóveda. Lo decide el
  auditor.

## Consecuencias

- Entrar cuesta **dos Argon2id**: el de la clave de acceso y el de la ranura. Cerca de un segundo.
- Quien controle el servidor puede **probar contraseñas contra la bóveda cifrada**, igual que quien robe
  hoy el fichero del disco: la defensa es una maestra fuerte, y por eso con cuenta se exige.
- Cambiar la contraseña es **todo o nada en el servidor** —bóveda con la ranura nueva, verificador nuevo
  y las demás sesiones fuera— y **solo después** se guarda en el equipo (la A3).

## Verificación

**Comprobado aquí**: que no se deriva con un coste por debajo del de siempre ni con uno imposible; que la
clave de acceso es estable y depende de la contraseña y de la sal; que la posesión **no cambia al cambiar
la maestra ni al rotar la de recuperación** y es distinta en cada bóveda; y el correo normalizado. Y
**contra el servidor de verdad**, levantado en local: alta con la clave derivada, entrar desde otro equipo
con su código y con un equipo de confianza, recuperar con la posesión y cambiar la contraseña, y que
`ESFINGE_SIN_RED` no deje salir ni una petición —en un proceso aparte, y rompiendo el freno a propósito
para ver que la prueba lo caza—.

**Sin comprobar**: nada de esto lo ha usado todavía la aplicación, que no tiene interfaz de cuenta hasta
la A2. Y que `NormalizarCorreo` de Go y `normalizarCorreo` del servidor coinciden en todos los casos raros
de Unicode: las dos pasan a minúsculas con reglas de lenguajes distintos. Manda la del servidor.
