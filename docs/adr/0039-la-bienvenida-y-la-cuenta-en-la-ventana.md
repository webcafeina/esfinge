# ADR 0039 — La bienvenida y la cuenta en la ventana

**Fecha:** 2026-09-18 · **Estado:** aceptada, publicada en la 2.23.0 y matizada en la 2.23.1 · **Continúa la [0035](0035-las-cuentas.md)** ·
**Revisar cuando** se use en el Mac con dos equipos de verdad

## Contexto

La ADR 0035 decidió con el cliente **una bienvenida visual al arrancar** para elegir entre trabajar en
el ordenador o con cuenta, «con las ventajas y desventajas de cada decisión», y que se pudiera cambiar
de idea en los dos sentidos desde Ajustes. La A1 dejó hecho todo lo de por debajo —la fusión, el
cliente del servidor, la sincronización— sin nada en la ventana. Esto es la A2: la parte que se ve.

## Decisión

### La bienvenida

- **Sale solo a quien no tiene nada**: sin bóveda y sin haber elegido. Quien ya tenía bóveda antes de
  que hubiera cuentas trabaja en local sin que se le pregunte; la novedad está en Ajustes.
- **A ventana entera, sin barra lateral**: se elige antes de entrar. Es la puerta y no una pantalla de
  trabajo, así que la esfinge grande tiene sitio (ADR 0021).
- **Dos tarjetas iguales**, sin destacar ninguna: «En este ordenador» y «Con cuenta», cada una con lo
  bueno (✓) y lo malo (–) dicho al lado, y con la cuenta **diciendo que su bóveda cifrada se guarda en un
  servidor de Webcafeína en la UE** y que pide contraseña fuerte y código por correo. Debajo: «Por ahora,
  las cuentas son por invitación» y «Puedes cambiar de idea cuando quieras, en Ajustes».
- Elegir «En este ordenador» no crea nada: apunta el modo y abre la ventana de siempre. Crear la bóveda
  en local también cuenta como elegirlo.

### El asistente

**Matizado el 2026-09-21, probándola el cliente con la 2.23.0** (2.23.1): con la bóveda de siempre,
crear la cuenta **la abre con la contraseña que se acaba de escribir** —pedía abrirla antes y había que
salir y volver a escribirlo todo—, y **si esa contraseña no llega a «Buena», pide una nueva en el mismo
paso**. La nueva se pone en la bóveda **después** de que el servidor acepte el alta: un código mal
escrito no deja la bóveda con otra contraseña y sin cuenta.

Crear la cuenta y entrar en ella van en el mismo marco que la bienvenida, a ventana entera:

- **Crear**: correo → código → contraseña maestra, dos veces, que **tiene que ser al menos «Buena»**
  —lo dice Go, con su frase, no la ventana—. Si en el equipo no había bóveda, se crea **en memoria** y
  solo se guarda si el alta sale, y después se enseña su clave de recuperación con la ceremonia de
  siempre. Si ya la había, la cuenta se crea con ella y con su contraseña.
- **Entrar**: correo y contraseña → código. Si en el equipo ya había **otra** bóveda, se pregunta:
  **juntarlas** —pide la contraseña de la de aquí si es otra— o **quedarse solo con la de la cuenta**. En
  los dos casos la de aquí **se aparta con otro nombre y no se borra nunca**, y se dice dónde ha quedado.
- **Volver a entrar**, cuando la sesión caduca: el mismo asistente, con el correo puesto.

### La sincronización, a la vista

- Una línea bajo la barra de la bóveda: «Sincronizada hace un momento», «Sin conexión: se sube al
  volver», «Hay que volver a entrar en la cuenta», con su botón.
- **Un grupo en Ajustes**, «Cuenta y sincronización»: en local, crear una cuenta o entrar en una; con
  cuenta, el correo, el nombre del equipo, el estado, «Sincronizar ahora» y **«Dejar la cuenta en este
  equipo»**, que pide la contraseña, cierra la sesión en el servidor y deja la bóveda aquí tal como está.

**Dejar la cuenta entraba en la A3 y se ha adelantado**: la bienvenida promete que se puede cambiar de
idea en Ajustes, y sin esto la frase sería mentira durante una versión. Lo que sigue en la A3 es
**borrar la cuenta del servidor**, exportar y la lista de equipos.

### Y lo que cambia hacia fuera

Antes de publicar, **la política de privacidad** deja de decir «Webcafeína no recibe nada» sin matiz y
gana una sección «Si creas una cuenta» —qué se guarda, dónde, quién más interviene, hasta cuándo y qué se
puede pedir—; **la portada y el README** dicen las dos formas de usarla; y **`docs/seguridad.md`** cuenta
la tercera salida a la red con lo que no protege. **La extensión no cambia**: sigue hablando solo con la
aplicación de este ordenador, y su aviso y lo declarado en las tiendas siguen siendo ciertos.

## Alternativas descartadas

- **Enseñar la bienvenida a todo el mundo en la versión que la trae**, también a quien ya tiene bóveda.
  Sería preguntar algo que ya está contestado: esa persona eligió local al crearla.
- **Poner la bienvenida dentro de la sección de la bóveda**, como la pantalla de crearla. Pero la
  pregunta es de toda la aplicación, no de una sección, y hecha ahí la vería solo quien fuera a la bóveda.
- **Radios para elegir entre juntar o apartar.** La hoja de estilos no los dibuja, y el navegador los
  pinta con el azul del sistema; el control segmentado ya existe y ya está medido.
- **Una raya de acento arriba de cada tarjeta.** Se probó: salía negra en claro y blanca en oscuro, y
  una elección neutra no puede destacar una de las dos.

## Consecuencias

- **Las pruebas de interfaz de siempre eligen local antes de empezar** (`beforeEach` con
  `ElegirModoLocal`): el servidor de desarrollo arranca sin nada, y la bienvenida taparía la ventana.
- `make e2e` levanta ahora **el servidor de cuentas de verdad y dos ventanas más**, cada una con su Go
  (`herramientas/servidor-para-e2e.sh`, `ESFINGE_GO` en Vite).
- La política de privacidad dice que exportar y borrar la cuenta entera **se piden por correo** mientras
  la aplicación no tenga esos botones (A3).

## Verificación

**Comprobado aquí**:

- **La tubería entera en la ventana, con dos equipos a la vez** (`e2e/cuentas.spec.ts`): bienvenida en uno,
  crear la cuenta con su código —y el rechazo de una contraseña débil, que es el único fallo que admite
  la prueba—, la clave de recuperación, guardar una entrada y **cerrar en el acto**; en el otro, entrar
  con su código, **ver la entrada del primero** y la sincronización al día; Ajustes con la cuenta, y
  dejarla.
- En Go, contra el servidor de verdad: crear desde cero y con la bóveda que ya había, entrar desde otro
  equipo, **juntar con otra bóveda y que la de aquí quede apartada y se abra con su contraseña**, que
  sincronizar no cuente como actividad, que borrar la bóveda se lleve la base y la cuenta, que la sesión
  no quede en claro en el disco, y dejar la cuenta.
- Capturas de cada pantalla nueva **en los dos temas, miradas**: de ahí salieron la raya de las tarjetas,
  el campo del correo y del código sin estilo —la hoja no vestía `type="email"` ni los campos sin tipo—
  y un falso fallo del tema oscuro que era la transición de los botones a mitad en la captura.

**Sin comprobar**: todo en el Mac —la bienvenida bajo el vidrio, arrastrar la ventana desde ella, dos
equipos de verdad con la red que va y viene— y que el correo del código llegue a otros buzones además
del de Webcafeína.
