# ADR 0037 — Las claves de la cuenta: acceso derivado aparte y posesión de la bóveda

**Fecha:** 2026-09-18 · **Estado:** aceptada, construida entera en la A3 (2.24.0) y matizada en la 2.24.1 · **Continúa la [0035](0035-las-cuentas.md)**
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

### Construido en la A3 (2026-09-21)

- **Cambiar la contraseña con cuenta** (`cambiarMaestraEnLaCuenta`): se para la sincronización de fondo,
  se sincroniza una vez, se prepara la bóveda con la ranura nueva **sin tocar el fichero de aquí**
  (`SubidaConMaestra`), se sube con la clave de acceso nueva —bóveda, verificador y sesiones revocadas a
  la vez en el servidor— y **solo entonces** se pone la ranura aquí (`PonerMaestra`). Si el servidor dice
  que no, aquí no ha cambiado nada. Con cuenta, la nueva tiene que llegar a «Buena».
- **El equipo que se quedó con la contraseña de antes**: la nueva no abre su fichero, pero la clave de
  bóveda es la misma. Al entrar con la nueva se abre el fichero de aquí con la clave de la bóveda bajada
  (`AbrirConLaLlaveDe`) y **se funde** con lo del servidor: trae la ranura nueva y no se pierde lo que
  tuviera sin subir. Solo si no se puede fundir se aparta, como antes.
- **Recuperar sin ningún equipo** (`TerminarRecuperacion`): el código del correo da el sobre de
  recuperación, la clave de recuperación lo abre y da la posesión, con ella el servidor da una sesión
  restringida, y con esa sesión se cambia la contraseña; después es como entrar.
- **Los equipos, exportar y borrar la cuenta**, en Ajustes. Borrar pide la contraseña y un código, y **la
  bóveda de cada equipo se queda** en local.

### Matizado en la 2.24.1: el otro equipo se abre con la contraseña nueva

Probando la 2.24.0, el cliente preguntó lo evidente: «si la cambio en un ordenador, ¿por qué tengo que
indicarlo en el otro?». Cambiar la contraseña **sigue cerrando las sesiones de los demás equipos** —se le
preguntó y eligió «pedir la nueva una vez»—, pero ya no hace falta ir a «¿Cambiaste la contraseña en otro
equipo?»: **basta escribir la nueva en «Abrir la bóveda»**.

- Si la contraseña no abre el fichero de aquí y el equipo está en una cuenta, `AbrirBoveda` prueba a
  entrar en la cuenta con ella (`abrirConLaCuenta`). Si el servidor la acepta, es la nueva: se baja la
  bóveda y se funde con la de aquí, igual que al entrar. Si no, el error es el de siempre.
- Para que eso no pida un código por correo, **el testigo de confianza del equipo va en claro** en
  `cuenta.json`, como la cookie de «recordar este equipo» de cualquier web. Sellado con la clave de la
  bóveda no se podía leer justo cuando hace falta: con la bóveda cerrada y una contraseña que ya no la
  abre. La sesión sigue sellada. Los de antes se pasan a claro al abrir la bóveda.
- Si el equipo no tiene testigo —caducó a los 90 días, o venía sellado de la 2.24.0—, **la contraseña
  nueva no se toma por mala**: el servidor la reconoce, manda el código al correo y «Abrir la bóveda» lo
  pide en la misma pantalla (2.24.2). En la 2.24.1 daba «Esa llave no abre esta bóveda», y le pasó al
  cliente en su segundo Mac.

**Lo que cuesta**: quien copie `cuenta.json` se lleva el testigo, que le ahorra el código por correo **si
además sabe la contraseña**. Pero quien tiene ese disco tiene también `boveda.esfinge`, y con la contraseña
la abre sin hablar con nadie: el segundo factor protege de una contraseña filtrada sin el equipo, y eso
sigue igual. Y **una contraseña mal escrita en un equipo con cuenta llega al servidor** —un Argon2 más y
un intento fallido—; los equipos de confianza no quedan fuera por los fallos, así que no se bloquea uno
mismo tecleando mal.

### Matizado en la 2.24.5: un equipo que pierde la sesión cierra la bóveda

Probando la A3, el cliente olvidó un Mac desde el otro, y el olvidado se quedó con la bóveda abierta
diciendo «vuelve a entrar». Pidió que se cerrara, y es lo que se espera de «olvidar»: sirve sobre todo
para un equipo perdido o robado, y abierto encima de una mesa no protege nada. Ahora, cuando el servidor
contesta que la sesión no vale (401), la bóveda se cierra, se olvida la sesión guardada y «Abrir la
bóveda» dice por qué.

- **Se comprueba antes con la sesión de ahora** (`cerrarPorSesionPerdida`): una pasada que salió con la de
  antes puede volver con un 401 justo después de volver a entrar, y cerraría una bóveda que ya está bien.
- **La sesión guardada se olvida**: si no, al reabrir arrancaría con ella, volvería el 401 y se cerraría a
  cada minuto.
- Vale igual para las otras dos causas del 401 —**la contraseña cambiada en otro equipo** y **la sesión
  caducada**—: en las dos hace falta volver a entrar, y en la primera cerrar es además lo prudente.
- **Lo que no hace**: borrar nada, ni cerrar un equipo apagado o sin conexión. Dicho en `docs/seguridad.md`.

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

**Comprobado en la A3**, contra el servidor de verdad: cambiar la contraseña en un equipo, que el otro
—con la de antes y **algo guardado sin subir**— entre con la nueva sin apartar su bóveda ni perder nada, y
que uno nuevo entre con la nueva y no con la vieja; recuperar en un equipo vacío con la clave de
recuperación, y no con otra; ver y olvidar equipos; exportar sin nada en claro, y borrar la cuenta
dejando la bóveda de aquí. Se rompió a propósito la fusión del equipo de la contraseña vieja y el orden
servidor-antes-que-aquí, y las pruebas lo cazaron.

**Comprobado en la 2.24.1**, contra el servidor de verdad: con la contraseña cambiada en A, B cerrado y con
algo guardado sin poder subir **se abre escribiendo la nueva** en «Abrir la bóveda», sin código, y se queda
con todo —lo de A y lo suyo, que llega a A—; una contraseña que no es ninguna de las dos sigue dando el
error de siempre, y la de antes deja de abrir. El testigo queda en claro en el disco. Se quitó a propósito
el paso por la cuenta y la prueba lo cazó.

**Comprobado en la 2.24.2**, contra el servidor de verdad y en la ventana: sin testigo, la contraseña nueva
pide el código en «Abrir la bóveda» y con él abre; una mala sigue siendo mala y no deja código pendiente.
En la ventana, con un equipo olvidado desde el otro, que es lo que le quita el testigo.

**Comprobado en los dos Macs del cliente con la 2.24.1** (2026-09-21): cambiar la contraseña en uno; en el
otro, la de antes todavía abre la copia de aquí y avisa de que hay que volver a entrar, y con la nueva entra
en la cuenta.

**Comprobado en su Mac con la 2.24.3**: exportar los datos de la cuenta —el fichero sale bien, y con la
bóveda cerrada el botón avisa—.

**Comprobado en sus Macs con la 2.24.4** (2026-09-21, tarde): **recuperar con la clave de recuperación**
—abre con la nueva, y el otro Mac entra escribiéndola— y **olvidar un equipo**, que pedía volver a entrar
pero dejaba la bóveda abierta: de ahí la 2.24.5.

**Comprobado en la 2.24.5**, contra el servidor de verdad y en la ventana: olvidado con la bóveda abierta,
se cierra solo y lo dice; al reabrir no se vuelve a cerrar; al volver a entrar sincroniza; y un 401 tardío
con una sesión que ya vale no cierra nada (rompiendo esa comprobación, la prueba lo caza).

**Sin comprobar en un Mac**: el cierre por olvido de la 2.24.5, y borrar la cuenta —el cliente no lo va a
probar con la suya; está probado aquí contra el servidor—. Y que `NormalizarCorreo` de Go y `normalizarCorreo` del servidor coinciden en todos los casos raros
de Unicode: las dos pasan a minúsculas con reglas de lenguajes distintos. Manda la del servidor.
