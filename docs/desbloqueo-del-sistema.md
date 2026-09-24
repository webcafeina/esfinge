# Desbloquear con el sistema: Touch ID, Windows Hello y un PIN

**Estado: decidido, en construcción.** Es la fase **C** del plan de cuentas
([`cuentas.md`](cuentas.md)), y el plan ya avisaba de lo que había que mirar antes de tocar código:
**puede exigir firmar la aplicación**, en contra de una decisión tomada con el cliente. Esto es lo que se
ha encontrado al mirarlo, el 2026-09-24.

La conclusión, por delante, porque cambia de qué va la fase:

> **En los dos sistemas, y sin firmar, «desbloquear con el sistema» es un cerrojo, no una llave.** Pide la
> huella o la cara y, si el sistema dice que sí, Esfinge abre la bóveda con una clave que estaba ahí de
> todos modos. Eso protege de quien se sienta delante del ordenador desbloqueado; **no** protege de un
> programa que corra como tú, que es de lo que protege hoy la contraseña maestra.

Eso no lo hace inútil —Dashlane, 1Password y el resto hacen exactamente esto en escritorio—, pero **hay que
decirlo en la ventana y en `seguridad.md`**, y hay que decidirlo sabiéndolo.

## Lo que ya está hecho

El formato lo previó desde la ADR 0023: **los tipos de ranura son una lista abierta**, y
`llavero-del-sistema` y `pin` ya están reservados y **excluidos de la sincronización** en las dos
implementaciones (`ranurasLocales` en Go, `RANURAS_LOCALES` en TypeScript), con su prueba. Una ranura así
**no se sube nunca**: es de un equipo, y subirla sería darle al servidor una segunda puerta.

Así que la parte de la bóveda no hay que inventarla. Lo que falta es de dónde sale la clave que envuelve
esa ranura, y ahí es donde está todo el problema.

## macOS

### El camino fuerte exige firmar, y está cerrado

Lo que hace un gestor de contraseñas en un Mac es guardar la clave en el llavero con un **control de
acceso biométrico** (`SecAccessControlCreateWithFlags` con `.biometryCurrentSet`), o crear una llave en el
**Secure Enclave** con ese mismo control. Las dos cosas viven en el **llavero de protección de datos**
(`kSecUseDataProtectionKeychain`), y ése:

> **exige la entitlement `keychain-access-groups`**, que solo lleva una compilación firmada con un perfil
> de aprovisionamiento. Sin ella, `SecItemAdd` devuelve **-34018**, «a required entitlement isn't present».

Un perfil de aprovisionamiento sale de una cuenta de desarrollador de pago. Es **justo lo que la ADR 0014
descartó**: «sin firmar ni notarizar para macOS, los 99 $/año de Apple no compensan para un cliente». Así
que este camino no está cerrado por lo técnico, está cerrado por una decisión que hay que volver a abrir
si se quiere.

### El camino que sí funciona sin firmar

`LAContext.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics)`: pedirle al sistema que identifique a
quien está delante, y quedarse con un sí o un no. **No requiere entitlements**; lo que sí requiere es que
el programa sea un **bundle `.app` con identificador**, porque el diálogo dice «*«Esfinge» está intentando
…*» y lo saca de ahí. Esfinge ya es un bundle, así que eso está.

Lo que hay que entender de este camino, y es lo que decide la fase:

- **Devuelve un booleano.** La clave de la bóveda no la protege la huella: la protege lo que sea que la
  guarde mientras tanto. Si se guarda en un fichero, quien lea ese fichero abre la bóveda sin poner el
  dedo. Si se guarda en el llavero de siempre, la protege la sesión del usuario.
- **Y un booleano se puede cambiar.** Un programa que corra como tú puede parchear Esfinge en el disco,
  o sencillamente leer la clave de donde esté. Frente a eso, hoy, la contraseña maestra sí protege: no
  está en ninguna parte.

### El llavero de siempre, y por qué tampoco resuelve

El llavero de inicio de sesión —el de toda la vida, el que no necesita entitlements— ata cada elemento a
la **firma de código** de quien lo guardó. Sin firma, esa atadura no se puede hacer bien: macOS pregunta
cada vez que el binario cambia, y **cada actualización de Esfinge lo cambia**. Una firma ad hoc no vale,
porque cambia en cada compilación.

### Y aun así es donde va el secreto (decidido el 2026-09-24, al escribir la C2)

La alternativa era **un fichero de 0600 al lado de la bóveda**, que es predecible, sobrevive a las
actualizaciones y no pregunta nada. Se descartó, y conviene que quede dicho por qué:

> Con el secreto en un fichero, **cualquiera que pueda leer tu carpeta abre la bóveda sin poner el dedo**.
> Eso no es un cerrojo peor: es quitar la puerta. Activar Touch ID pasaría de «no tengo que teclear la
> maestra» a «mi bóveda la abre quien copie mi carpeta de usuario», y eso es **bajar** la protección de la
> bóveda a cambio de una comodidad.

En el llavero de inicio de sesión el secreto está cifrado con la contraseña de macOS. Sigue siendo un
cerrojo —un programa que corra como tú puede pedirle al llavero lo mismo que le pide Esfinge—, pero **un
disco copiado no lo lleva dentro**. Lo que se paga por eso es la pregunta al actualizar de la que habla el
apartado anterior, y ése es el orden de prioridades correcto en un gestor de contraseñas.

## Windows

### Se puede sin empaquetar, pero la credencial no es tuya, es de la sesión

`KeyCredentialManager` —las llaves respaldadas por Hello y el TPM— **funciona en una aplicación de
escritorio sin empaquetar**, pero con una letra pequeña que lo cambia todo. Lo dice Microsoft:

> «For Unpackaged Apps (Classic Win32 Desktop): **credentials are scoped only to the Windows User
> Account**», y por eso «Application B runs as an unpackaged Win32 process under the same user, it
> successfully **loads the credential**».

Es decir: **cualquier otro programa que corra como tú y sepa el nombre de la credencial puede usarla.**
No hay frontera de aplicación sin AppContainer, y AppContainer no es donde vive una aplicación de
escritorio normal. Un programa que quiera la bóveda de Esfinge solo tiene que pedirle a Hello lo mismo que
le pide Esfinge.

Y la recomendación de Microsoft para este caso es exactamente lo contrario de sustituir la contraseña:

> «Windows Hello would not be the only protection mechanism but rather one component of a **defense-in-depth**
> design, where **both** user knowledge (the application password) **and** Windows Hello are required».

`UserConsentVerifier` —solo el consentimiento, sin llave— es más sencillo y más débil: es el booleano de
macOS otra vez, y además en una aplicación de escritorio hay que llamarlo por la interfaz de interoperación
`IUserConsentVerifierInterop`, pasándole la ventana.

## El PIN

Un PIN de cuatro o seis cifras **no puede proteger una bóveda por sí solo**: el espacio es tan pequeño que
probarlos todos contra el fichero es instantáneo, y Argon2id no arregla eso —subir el coste lo bastante
para que un millón de intentos duelan haría que abrir la bóveda tardara minutos—. Un PIN solo vale si algo
**limita los intentos**, y eso solo lo puede hacer el hardware: el Secure Enclave o el TPM. Que es volver
exactamente a los dos problemas de arriba.

La salida honesta, si se quiere un PIN, es la que usan los demás: **el PIN no protege la bóveda, protege un
desbloqueo rápido dentro de una sesión ya abierta**, y al reiniciar o al cerrar sesión hay que poner la
maestra. Eso se puede hacer sin hardware, y hay que llamarlo por su nombre.

## Lo que hay que decidir antes de escribir nada

> **Decidido con el cliente el 2026-09-24**, después de leer esto:
>
> - **No se firma en macOS** (pregunta 2). Así que esto es **el camino del cerrojo**, y hay que escribirlo
>   donde se active y en `seguridad.md`.
> - **Se hace en los tres sistemas** —Touch ID, Windows Hello y, en Linux, nada—, sabiendo que el código
>   de Windows se escribe a ciegas: nadie ha ejecutado nunca Esfinge en un Windows.
> - **Y no hay PIN.** «Siempre será la contraseña maestra», dicho así. La única alternativa a teclearla es
>   la biometría del sistema; donde no la haya —Linux, un Mac sin Touch ID, un Windows sin Hello— se
>   teclea la maestra, y la pantalla lo dice en vez de ofrecer algo que no está.
>
> Se descartó a la vez la propuesta de **un desbloqueo rápido con PIN en memoria**, que daba la misma
> protección real sin código nativo y se podía probar aquí entero. El cliente prefiere que lo único que
> sustituya a la maestra sea la huella, y no un número corto.

1. **¿Se acepta que esto sea un cerrojo y no una llave?** Con la decisión de no firmar, en macOS es lo
   único que hay; en Windows, lo único que hay sin aceptar que otra aplicación tuya pueda pedir lo mismo.
   Si la respuesta es sí, **hay que escribirlo en la ventana**, donde se activa, y en `seguridad.md`.
2. **¿Se reabre lo de firmar en macOS?** 99 $ al año compran el camino fuerte —Secure Enclave y llavero de
   protección de datos—, y de paso quitan el aviso de Gatekeeper de la primera instalación. Es una decisión
   de producto, no técnica.
3. **¿Qué pasa al actualizar?** Cada versión nueva de Esfinge es un binario nuevo. En Windows la credencial
   de Hello sobrevive; en macOS, con el llavero de siempre, no necesariamente. Hay que probarlo, y hay que
   decidir qué se le dice a alguien cuya huella «deja de funcionar» tras actualizar.

   **Escrita la C2, esto es lo único que queda por ver de macOS**, y solo se ve actualizando de verdad:
   si al abrir tras una actualización macOS pregunta «¿Permitir que Esfinge use esta información?», se
   contesta **«Permitir siempre»** y a partir de ahí no vuelve a preguntar hasta la siguiente. Si
   preguntara en cada apertura, o si en vez de preguntar fallara, hay que volver aquí: el arreglo sería
   **borrar la ranura y pedir que se active otra vez**, que es lo que ya hace el código cuando el secreto
   deja de abrir, y decirlo en la pantalla.
4. **¿Y si se pierde la huella?** La ranura del sistema **nunca** puede ser la única: la maestra y la clave
   de recuperación siguen abriendo. Eso ya está resuelto por el formato, pero hay que decirlo en la
   pantalla, porque quien activa Touch ID tiende a olvidar la contraseña que ya no escribe.

## Cómo se partiría

| Entrega | Qué lleva | Cómo se comprueba |
|---|---|---|
| **C1** · **hecha el 2026-09-24** | La ranura local, el interruptor de Ajustes y el botón de la pantalla de desbloquear, con un llavero de mentira detrás para poder moverlo aquí. **Sin PIN**: donde no hay biometría se teclea la maestra y la pantalla lo dice | Pruebas de Go —abre, la maestra sigue abriendo, no viaja en `PrepararSubida`, cancelar no deja nada a medias, un secreto que ya no abre se olvida— y e2e de las dos pantallas en los dos temas |
| **C2** · **escrita el 2026-09-24** | macOS: `LAContext` por cgo más el llavero de inicio de sesión, en un fichero que **no se puede compilar aquí**, como `vidrio_darwin.go` | El trabajo de macOS de la publicación dice que compila; **solo un Mac de verdad dice si arranca, si pregunta al actualizar y si el diálogo se lee bien** |
| **C3** | Windows: `UserConsentVerifier` o `KeyCredentialManager` | La máquina de Windows de la publicación compila; **nadie lo ha ejecutado nunca en un Windows** |
| ~~**C4**~~ | ~~El PIN como desbloqueo rápido de sesión~~ → **descartado por el cliente el 2026-09-24**: «siempre será la contraseña maestra». La ranura `pin` se queda reservada en el formato y sin usar | — |

## Lo que cuesta, dicho de una vez

Esta fase tiene **la peor relación entre lo que se puede probar aquí y lo que hay que escribir**: dos
ficheros de cgo que esta máquina no compila —la lección de `vidrio_darwin.go` es que eso ya costó una
versión que no arrancaba—, en dos sistemas de los que uno no se ha ejecutado nunca. Antes de empezarla
conviene mirar si lo que se gana —no teclear la maestra— vale eso, o si se gana más en otro sitio.

## Fuentes

- [macOS Keychain SecItemAdd complains about missing entitlement when using biometric access control](https://developer.apple.com/forums/thread/684453)
- [-34018 A required entitlement isn't present](https://developer.apple.com/forums/thread/130264)
- [LAContext · Apple Developer Documentation](https://developer.apple.com/documentation/localauthentication/lacontext)
- [localauthentication package · go-macos](https://pkg.go.dev/github.com/go-macos/localauthentication)
- [What is the security boundary of Windows Hello KeyCredentialManager credentials for desktop apps?](https://learn.microsoft.com/en-us/answers/questions/5912130/what-is-the-security-boundary-of-windows-hello-key)
- [How to use Windows Hello / UserConsentVerifier from a Win32 application](https://github.com/microsoft/WindowsAppSDK/discussions/1265)
- [IUserConsentVerifierInterop](https://learn.microsoft.com/en-us/windows/win32/api/userconsentverifierinterop/nn-userconsentverifierinterop-iuserconsentverifierinterop)
