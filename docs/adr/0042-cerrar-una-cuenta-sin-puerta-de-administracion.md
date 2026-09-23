# ADR 0042 — Cerrar una cuenta sin abrir una puerta de administración

**Fecha:** 2026-09-23 · **Estado:** aceptada, hecha en el servidor · **Continúa la [0036](0036-el-servidor-de-cuentas.md)**
y la [0041](0041-los-papeles-de-la-cuenta.md) · **Revisar cuando** haya que cerrar una cuenta de verdad

## Contexto

Las condiciones de uso prometen dos cosas que hasta ahora el servidor no sabía hacer:

- **cerrar la cuenta de quien las incumpla**, avisando con antelación y dándole un plazo para llevarse
  sus datos;
- **borrar la cuenta de un menor** de la edad mínima, si nos consta.

Lo único que sabía hacer era borrar **a petición de su dueño**, con su contraseña maestra y un código de
su correo (`indice.ts`, `DELETE /v1/cuenta`). Nadie más puede hacer nada, y eso es una virtud del diseño:
el servidor no tiene administradores porque no hay nada que administrar dentro de una bóveda que no se
puede leer.

Escribir una promesa que el programa no puede cumplir es peor que no prometerla, así que había que elegir:
rebajar el texto o darle al servidor la capacidad justa.

## Decisión

**Una marca en D1, puesta a mano, y ninguna ruta nueva.**

- La tabla `cuentas` gana una columna `suspendida` (migración `0002_suspension.sql`). Se pone y se quita
  con `wrangler d1 execute`, **igual que la lista de admisión**, que es un camino que ya existe, que ya se
  usa y que no está expuesto a internet.
- **Suspendida, la cuenta no entra ni sube**: `POST /v1/sesion`, `POST /v1/sesion/codigo` y
  `PUT /v1/boveda` contestan **403** con «Esta cuenta está suspendida. Escríbenos a info@webcafeina.com».
- **Pero sigue dejando leer y exportar**: `GET /v1/boveda` y `GET /v1/cuenta/exportacion` funcionan. Sin
  eso, «te damos un plazo para llevarte tus datos» sería una frase vacía, porque el plazo se cumple
  bajando los datos.
- **Cerrar del todo es borrar la fila y el objeto**, y eso también se hace a mano, pasado el plazo.

El procedimiento entero, con las órdenes, está en `servidor/LÉEME.md`.

## Alternativas descartadas

- **Una ruta de administración** (`POST /v1/admin/…`) con un secreto. Es lo evidente y lo cómodo, y es
  exactamente lo que no queremos: una puerta que existe se puede forzar, y en un servidor que presume de
  no poder leer nada, la única puerta interesante para alguien de fuera sería justo ésa. Con `wrangler`
  el acceso ya está limitado a quien tenga el token de Cloudflare, sin nada nuevo expuesto.
- **Suspender dentro del Durable Object.** Sería más fino —la marca viviría con el resto del estado de la
  cuenta— pero desde `wrangler` no se puede tocar un objeto sin escribir una ruta que lo haga. Vuelve al
  problema anterior.
- **Rebajar el texto** y no prometer el cierre. Se podía, pero entonces tampoco se puede prometer borrar
  la cuenta de un menor, que es lo que la política dice sobre la edad mínima.
- **Bloquearlo todo, incluida la lectura.** Más simple de explicar y peor: convierte el plazo para
  llevarse los datos en algo que hay que pedir por correo y atender a mano.

## Consecuencias

- **Hay que acordarse de las dos partes**: suspender no borra. Quien suspenda una cuenta tiene que anotar
  cuándo termina el plazo y volver a borrarla, porque nada lo hace solo.
- **La marca está en D1 y la bóveda en el Durable Object**, así que borrar solo la fila deja el objeto
  huérfano con la bóveda cifrada dentro. El procedimiento del `LÉEME` borra primero el objeto —con la
  ruta de siempre— y después la fila.
- **Una cuenta suspendida no se distingue de una que no existe** en la pre-entrada, que sigue contestando
  lo mismo para todo el mundo. Eso es deliberado: la pre-entrada no puede decir quién tiene cuenta.
- **Un 403 nuevo que el cliente tiene que saber leer.** Hoy la aplicación lo enseñará como un error más;
  el mensaje es claro y dice a quién escribir, que es lo que hace falta mientras esto no ocurra nunca.

## Verificación

**Comprobado** con tres pruebas (`servidor/test/suspension.test.ts`): que suspendida no se puede entrar y
que el mensaje dice a quién escribir; que **no se puede subir pero sí bajar y exportar**, que es la parte
que sostiene el plazo; y que sin suspender no cambia nada.

**Sin comprobar**: no se ha suspendido ninguna cuenta de verdad contra el servidor desplegado, ni se ha
cronometrado el procedimiento completo de aviso, plazo y borrado. Y **nadie ha tenido que usarlo**, que es
la mejor de las noticias posibles sobre esta ficha.
