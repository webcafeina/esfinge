# ADR 0032 — Guardar y actualizar contraseñas desde la página

**Fecha:** 2026-09-14 · **Estado:** aceptada · **Matiza la [0028](0028-rellenar-en-la-pagina.md) y la
[0031](0031-el-icono-con-estados-y-la-marca-en-el-campo.md)** · **Revisar cuando** se use en sitios de
verdad, y al enviar la extensión a las tiendas

## Contexto

Con la parte visual de la extensión cerrada en la 2.20.3, lo que faltaba para dejar Dashlane en el
navegador era lo contrario de rellenar: **que al entrar, registrarse o cambiar la contraseña en un
sitio, Esfinge ofrezca guardarla o actualizarla**.

Es un cambio de naturaleza en dos cosas, y se dice así:

- **Es la primera vez que el navegador escribe en la bóveda.** Hasta la 2.20.3 solo leía.
- **Es el primer elemento de Esfinge que se puede pulsar dentro de la web de otro.** La 0028 decía «en
  la página no se dibuja nada» y la 0031 lo matizó con un filete y un aviso que no se pulsan.

Todo se decidió con el cliente por preguntas con opciones, en dos rondas.

## Decisión

| Pregunta | Elegido |
|---|---|
| Dónde se pregunta | **Una tarjeta en la página**, arriba a la derecha, como Dashlane |
| Quién lo aprueba | **Basta con pulsar en el navegador**; no se confirma en la ventana de Esfinge |
| Cuándo se ofrece | **Los cuatro casos**: entrar con una cuenta nueva, entrar con otra contraseña, registrarse y cambiar la contraseña |
| Con la bóveda cerrada | **La tarjeta dice que la abras**, y «Ya la he abierto» vuelve a mirar |
| Título de la cuenta nueva | **El del sitio, editable en la tarjeta** («brevo.com» → «Brevo») |
| «Nunca en este sitio» | **Sí**, con la lista **dentro de la bóveda, cifrada**, y deshacible en Ajustes |
| Cuánto dura la tarjeta | **Hasta que se decida** o se cambie de página |
| Varias cuentas al actualizar | **Se elige la cuenta en la tarjeta** |

### De punta a punta

```
página A (formulario)        trabajador de fondo            Esfinge (Go)
  envío detectado ──────────▶ «pendiente» en memoria,
  (usuario, secreto, forma)   por pestaña, dos minutos
página B (tras navegar)
  «¿hay algo?» ─────────────▶ si es del mismo sitio ───────▶ ofrecer
  tarjeta ◀─────────────────── acción, cuentas, título ◀──── guardar / actualizar / nada
  clic «Guardar» ───────────▶ guardar-cuenta con el secreto ─▶ Poner, y aviso a la ventana
                              que tiene él, no la página
```

### La contraseña no vuelve a la página

Al pulsar «Entrar» la página cambia y su guion muere, así que lo enviado tiene que esperar en algún
sitio. **Espera en la memoria del trabajador de fondo, y en ningún otro**: nunca en `storage`, nunca en
la página siguiente. La tarjeta de la página B recibe el sitio, el usuario, las cuentas y el título, y al
pulsar manda **solo la decisión**. Quien tiene la contraseña —el trabajador— es quien llama a Esfinge.

El pendiente se va al decidir, **a los dos minutos**, al cerrar la pestaña y si la pestaña pasa a otro
sitio. Si MV3 mata el trabajador antes, la oferta se pierde: se acepta.

### Qué es cada formulario, en negativo

`queSeEnvia` (`navegador/src/campos.ts`), como toda la detección, **ante la duda no ofrece nada**:

- **Una contraseña**: entrar. O registrarse, si el campo se declara `new-password`.
- **Dos iguales**: registrarse.
- **Dos distintas** si el sitio dice cuál es cuál —con `autocomplete` o, desde la 2.21.2, con el nombre
  del campo— y lo dice de una sola: cambiar, y la nueva es la segunda.
- **Tres**, con las dos últimas iguales y distintas de la primera: cambiar.
- **Cualquier otra cosa**, nada. Dos distintas sin decir cuál es cuál, o una nueva y una repetida que
  no coinciden, no se ofrecen.

El envío se detecta por tres caminos, porque ninguno solo basta: el evento `submit`, un clic en un
botón del formulario o del contenedor que tiene la contraseña, e Intro en el campo. **Solo cuentan los
de una persona** (`isTrusted`).

### Si vuelve a salir el formulario, la contraseña era mala

**Guardar una contraseña equivocada es peor que no ofrecer.** En la página B, al entrar, cualquier
campo de contraseña visible descarta el pendiente; al registrarse o cambiar, solo si sigue relleno,
porque muchos sitios dejan el formulario puesto y vacío tras un cambio bien hecho.

Para los sitios que entran sin cambiar de página se mira también **a los tres segundos de enviar**,
pero ahí **no se descarta**: puede ser la página de antes esperando a que el sitio conteste.

### Todo lo que decide qué ofrecer vive en Go

El verbo `ofrecer` decide si la cuenta existe, si la contraseña es la misma —entonces nada—, si el
sitio está excluido, qué cuentas son candidatas y qué título se sugiere. La extensión solo pinta.
**Publicar un arreglo en Go es empujar una etiqueta; en una tienda son días.**

### Solo se guarda para el sitio del que salió

El origen lo pone el navegador al enviar (`sender.tab.url`), y **la decisión viaja con ese origen, no
con el de la página donde está la tarjeta**. La cuenta nueva lleva ese sitio y ningún otro; actualizar
pasa por `entradaDe`, que exige que la entrada encaje con el origen aunque se mande su identificador.

### Los límites de la tarjeta

- **Sombra cerrada.** La web no lee lo que hay dentro ni alcanza sus botones.
- **Solo cuentan los clics de verdad** (`isTrusted`). Lo peor que consigue una web tramposa engañando
  para pulsar es guardar lo que se escribió **en su propio dominio**.
- **Solo en la trama de arriba**, para no poner una por marco. Los envíos sí se leen en las tramas
  del mismo origen, y **nunca en un marco de otro origen**, igual que el relleno.
- **No lleva nada de la web** salvo el anfitrión y el usuario, escritos con `textContent`.
- Colores fijos —piedra, blanco, oro y el gris `#d8d8de`— **medidos** en
  `internal/tema/extension_test.go`, que lee `tarjeta.ts` y falla si cambian sin medirse.

### Escribir tiene su propio freno, y no es actividad

`guardar-cuenta`, `actualizar-cuenta` y `nunca-aqui` gastan de un freno de **seis por minuto**, más
estrecho que el de preguntar y el de rellenar. `ofrecer` gasta del de preguntar. Nada de esto llama a
`Actividad()`: una bóveda abierta sobre la mesa se sigue cerrando sola.

Con la bóveda en solo lectura, `ofrecer` no ofrece nada y escribir falla con su frase.

### La ventana se entera

Tras escribir, Go emite `boveda-cambiada`. La lista de la bóveda se vuelve a pedir y Ajustes vuelve a
leer los sitios excluidos. Antes no existía ese aviso: nada de fuera escribía en la bóveda.

### Los sitios excluidos, cifrados

`SitiosExcluidos` va dentro del cuerpo de la bóveda, que se sella entero con su llave. La lista de
sitios en los que usas cuentas es justo lo que la [ADR 0024](0024-iconos-de-los-sitios.md) decidió no
dejar en claro. **Por eso Ajustes solo la enseña con la bóveda abierta**. La prueba mira el fichero y
comprueba que el dominio no aparece.

### Corregido tras probarla en Google (2.21.1)

El cliente probó la 2.21.0 hasta cambiar la contraseña y encontró un fallo **de las páginas que piden el
usuario y la contraseña por separado**. En Google, con `info@` y `alvaro@` guardadas, borró `alvaro@`
para que saliera como nueva. Al entrar:

1. En la página del correo, Esfinge **rellenó `info@`**, que era la única cuenta. Él lo cambió por
   `alvaro@`.
2. En la página de la contraseña **ya no hay ningún campo con el usuario**, así que Esfinge rellenó la
   contraseña de `info@` encima de lo que él iba a escribir.
3. Al enviar la de `alvaro@`, la tarjeta llegó **sin usuario**, y sin usuario Go ofrece actualizar las
   cuentas del sitio: **«Actualizar» la de `info@` con la contraseña de `alvaro@`**.

El arreglo es **recordar quién entra**:

- El guion de la página avisa del **usuario que hay** en el campo de una página de solo usuario —el
  mismo criterio con el que se rellena: `autocomplete="username"` y ninguna contraseña a la vista—.
  **Lo ponga quien lo ponga**: tecleado, puesto por el navegador, recordado por el sitio o escrito por
  Esfinge, porque es el que se va a enviar. Se lee cuando el campo cambia y otra vez **con un clic o un
  Intro de una persona** en cualquier sitio de la página, así que no hace falta saber dónde está el
  botón «Siguiente» y vale también para un sitio que pone el valor sin avisar.
- El trabajador de fondo lo guarda por pestaña, **en memoria, cinco minutos y solo para el mismo
  sitio** (`usuariosEscritos`). No es una contraseña, y tampoco sale de ahí hacia otro sitio.
- **Quién entra**, en la página siguiente: el usuario escrito en el propio formulario; si no hay, el
  **escondido que declara el sitio** (`autocomplete="username"`, que muchos ponen para los gestores); si
  no, el recordado.
- **Rellenar sola**: si se sabe quién entra, **la cuenta con ese usuario, si hay exactamente una**; si no
  se sabe, como antes, solo con una única cuenta. Lo decide `cuentaParaRellenarSola` (`identidad.ts`),
  una función pura. Vale también para el código de un solo uso.
- **Al enviar sin usuario en el formulario**, el trabajador pone el recordado. Con eso la tarjeta ofrece
  **guardar `alvaro@` como cuenta nueva**.

**Y con varias cuentas, la que coincida** (2.21.1). Al verlo arreglado, el cliente pidió que con varias
cuentas del sitio se rellene sola la del correo que va a entrar. **Matiza la ADR 0028**, que decía que con
varias se elige siempre en el panel: ahora solo cuando no se sabe quién entra, o no coincide ninguna, o
coinciden dos.

**Y el selector de cuentas de Google**, donde se pulsa una ficha y no hay campo de correo que leer. Se
resolvió **mirando, no adivinando**: el cliente sacó por la consola los campos de la página de la
contraseña tras elegir `alvaro@`, y Google no declara ningún `username` —por eso no se veía— pero lleva
el correo en un **`<input type="email" name="identifier" autocomplete="off">` escondido**. Así que el
usuario escondido vale también así: **un `type="email"` con valor, solo con una contraseña a la vista y
si todos esos campos dicen el mismo correo**. Esa página, copiada tal cual, está en las pruebas. Con eso
da igual cómo se llegue a la página de la contraseña.

### Brevo: la actual y la nueva, por el nombre (2.21.2)

Con la 2.21.1, cambiar la contraseña en Brevo **no ofrecía nada**. El cliente sacó por la consola el
formulario de `app.brevo.com/profile/password`: **dos contraseñas distintas sin `autocomplete`**, llamadas
`currentPassword` y `newPassword`, y un botón `type="button"` dentro del `<form>`. Sin la declaración
estándar, la regla no sabía cuál era la nueva y callaba, a propósito.

- **El nombre del campo cuenta como declaración**: `current`, `old`, `actual`, `antigua`, `anterior` o
  `vieja` para la actual; `new` o `nueva` para la nueva. **Solo si lo dice de uno y no del otro**:
  `newPassword` y `confirmNewPassword` con valores distintos es un error al teclear y sigue sin ofrecerse.
- **Se mira varias veces tras enviar** —a los tres, ocho y quince segundos— y no una: Brevo cambia la
  contraseña sin cambiar de página y puede tardar en contestar y en vaciar el formulario. Sin repintar una
  tarjeta que ya esté a la vista.

**Lo que no se sabe**: qué hace Brevo con los campos después de guardar. La segunda salida de la consola
no se pudo sacar —la cuenta de pruebas se quedó sin la contraseña original—. **Si los deja rellenos, la
tarjeta seguirá sin salir**, porque eso es lo que dice «la contraseña era mala».

## Alternativas descartadas

**El panel de la extensión, sin tocar la página.** Mantenía la 0028 al pie de la letra, pero nadie abre
el panel después de entrar en un sitio: la oferta no se vería nunca.

**Confirmar en la ventana de Esfinge.** Más seguro contra una web que engañe para pulsar, pero dos
aprobaciones por cada contraseña. Con los límites de arriba, lo que se gana es poco.

**Guardar la contraseña en la tarjeta de la página B.** Era lo más sencillo: se la pasa el trabajador y
se manda de vuelta al pulsar. Pero entonces la contraseña estaría en una página que no es donde se
escribió, y un fallo de la sombra la expondría allí.

**Guardar el pendiente en `storage.session`.** Sobreviviría a la muerte del trabajador, pero es una
contraseña en un almacén que otras partes de la extensión pueden leer. Perder alguna oferta es mejor.

**Decidir en la extensión qué ofrecer.** Ahorraba un viaje por el canal, pero los arreglos tardarían días
en llegar por las tiendas.

**Ofrecer con cualquier formulario con contraseña.** Se ofrecería guardar la contraseña de un formulario
de cambio con la vieja, o una que no coincide con su repetición.

## Consecuencias

- **El navegador escribe en la bóveda.** Quien pueda hablar por el canal —con el permiso dado— puede
  crear cuentas, cambiar contraseñas y apuntar sitios excluidos, a un ritmo de seis por minuto. La
  contraseña anterior **va al historial**, así que un cambio no deseado se deshace desde la ventana.
- **Un elemento pulsable nuestro en la página de otro.** Un fallo de la tarjeta sería un fallo en la
  página de otro.
- **Una web puede saber que usas Esfinge** al ver la tarjeta, además de lo dicho en la 0031.
- **Tres segundos después de cada envío** el guion de la página vuelve a preguntar. Sin nada pendiente,
  contesta el trabajador y no se llega a Esfinge.
- **Un registro en un sitio que no responde bien** puede dejar el formulario relleno y no ofrecer nada.
  Es el lado seguro del error.

## Verificación

**Comprobado aquí:**

- **Go**: los cuatro verbos en la lista blanca; `ofrecer` devuelve guardar, actualizar o nada en cada
  caso —misma contraseña, excluido, cerrada, solo lectura, sin usuario—; **guardar pone solo el sitio del
  origen**; **actualizar una entrada de otro sitio falla** con el identificador en la mano; la anterior
  pasa al historial; el freno de escrituras corta a la séptima; **no cuenta como actividad**; la lista
  de excluidos **sobrevive a cerrar y abrir** y **no aparece en claro** en el fichero. Y **la tubería
  entera**, de los bytes del navegador al fichero, con `ofrecer` y `guardar-cuenta`.
- **Chromium** (`navegador/pruebas/`): ocho formularios para `queSeEnvia`, la mitad donde lo correcto es
  nada; `pendientes.ts` entero —caducidad, mismo sitio o no, http—; y la tarjeta con **un clic fabricado
  que no hace nada**, la sombra cerrada, el título editable, el selector con varias cuentas, la bóveda
  cerrada, «Nunca en este sitio», «Ahora no», el error y la confirmación que se va sola.
- **Capturas** de la tarjeta en web clara y oscura, en sus cinco estados, **miradas**. De mirarlas salió
  que «Guardar» no cabía en la fila a 320 px: «Nunca en este sitio» pasó a su línea, subrayado.
- `make comprobar` y `make e2e` en verde.

**Publicada en la 2.21.0 (2026-09-14)** y probada por el cliente hasta cambiar la contraseña, en su
Mac. De ahí salió el fallo de Google de arriba, corregido en la 2.21.1 con pruebas del teclado y el ratón de verdad —lo tecleado se
recuerda, lo puesto por Esfinge o el navegador también, lo puesto por el sitio sin avisar se lee al pulsar
«Siguiente», un clic fabricado no lo lee y con una contraseña a la vista nada—, del campo de solo usuario,
del usuario escondido y de `cuentaParaRellenarSola` con una y con varias cuentas. **Sin comprobar contra el Google de verdad**: el criterio del campo es el mismo
que ya rellenaba esa página, y eso es lo único que se sabe de su HTML.

Sin comprobar todavía, y hay que mirar en el Mac, en Firefox y en Chrome:

- **Todo lo del trabajador de fondo y del guion de la página con la extensión cargada**: el pendiente
  cruzando la navegación, la detección del envío en sitios de verdad y la heurística del formulario que
  vuelve. Sigue siendo la deuda alta de `docs/deuda.md`.
- Entrar con una cuenta nueva, entrar con otra contraseña y ver la anterior en el historial,
  registrarse, cambiar la contraseña, **que un inicio de sesión fallido no ofrezca nada**, «Nunca en este
  sitio» y deshacerlo en Ajustes, y que la ventana enseñe la cuenta nueva sin tocar nada.
- **La lista de sitios excluidos en Ajustes** no tiene prueba de interfaz.
