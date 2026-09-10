# ADR 0028 — Rellenar en la página

**Fecha:** 2026-09-10 · **Estado:** aceptada · **Continúa la [0027](0027-el-canal-con-el-navegador.md)** ·
**Revisar cuando** se haya usado unos días en sitios de verdad, y al enviarlo a las tiendas

## Contexto

La [ADR 0027](0027-el-canal-con-el-navegador.md) montó el canal y se paró justo
antes de lo que se nota: la extensión reconoce el sitio, enseña las cuentas y
copia, **sin tocar ninguna página**. Eso ya sirve —es Dashlane sin el relleno— y
está comprobado en un Mac en Firefox y en Chrome.

Lo que queda es el gesto que se repite veinte veces al día y que decide si se deja
Dashlane: que la contraseña aparezca en el formulario. Y es donde entra el riesgo
que la entrega 1 no tiene, dicho desde el primer plan con estas palabras: **código
nuestro en todas las páginas**.

Dos propiedades que la entrega 1 tenía y que esta entrega **rompe a conciencia**.
Hay que decirlas antes de nada, porque son lo que cambia:

1. **«Por el canal no sale ningún secreto».** Era cierta y era fuerte: copiaba
   Esfinge y por el socket solo volvía cuánto tardaría en borrarse el portapapeles.
   Para escribir una contraseña en un formulario hay que tenerla, así que deja de
   ser cierta.
2. **«Esfinge no toca ninguna página».** Desde aquí hay un guion nuestro en cada
   página https que se abra.

## Decisión

**Se rellena; y en la página no se dibuja nada.**

### Un verbo nuevo, y solo uno

`rellenar(id, origen)` devuelve el usuario y la contraseña de **una** entrada. Es
el único sitio de todo el protocolo por el que sale un secreto, y lleva las mismas
cinco llaves que lo demás —testigo, origen que da el navegador, dominio
registrable, bóveda abierta, entrada de ese sitio— más una suya:

- **Su propio freno, más estrecho** (doce por minuto frente a sesenta). Preguntar
  de más enseña una lista de sitios; rellenar de más entrega contraseñas.

Y una consecuencia que no se puede evitar y por eso se escribe: **lo que sale por
ahí no hereda el borrado del portapapeles**, porque no pasa por el portapapeles.
Que se escriba en el campo y se olvide es una promesa de la extensión, y Esfinge
no tiene forma de comprobarla.

### En la página no se dibuja nada

Ni desplegable sobre el campo, ni icono dentro, ni marco flotante. El guion **lee
el formulario y escribe en él**, y nada más.

Con **una sola cuenta** guardada del sitio —el caso corriente— se rellena sola al
cargar la página, que es exactamente lo que se pidió. Con **varias** no se toca
nada: se elige en el panel de la extensión, que ya existe, ya está probado y está
fuera de la página por construcción.

Es la decisión de forma de esta entrega y la que más reduce el riesgo que la
entrega trae. Un desplegable propio dentro de la página de otro obliga a un marco
de nuestro origen, a pelearse con el `z-index` y el `position` de cada sitio, y a
demostrar que la página no lo puede leer ni pulsar por su cuenta. Todo eso es
trabajo, y **todo eso es superficie**. Con una cuenta no hace falta elegir; con
varias, un clic más en un sitio seguro es un precio razonable hasta saber cuántas
veces pasa de verdad.

### El origen lo pone el navegador, y ahora es una línea

En la entrega 1 la dirección la mandaba el panel, sacada de `tabs.query`. Ahora
también habla el guion de la página, y una página con una vulnerabilidad podría
querer decir de qué sitio es. Así que el trabajador de fondo **tira lo que llegue**
por el puerto de una página y pone `sender.tab.url`, que lo rellena el navegador.

### Nunca en un marco de otro origen

El guion se pone en todos los marcos, y lo primero que hace es comprobar que
comparte origen con el de arriba. Si no —o si preguntarlo lanza, que es lo que
pasa cuando no lo comparte—, no hace nada más. Un `iframe` ajeno puede ser
cualquiera y la dirección que se ve en la barra no es la suya.

### Y las reglas de qué campo se toca, en negativo

Equivocarse de campo es escribir una contraseña donde la va a leer alguien. Así
que ante la duda no se rellena: no rellenar es una molestia, rellenar mal es el
fallo que hace daño. No se toca un formulario con dos contraseñas visibles —eso es
registrarse—, ni un campo declarado `new-password` o `one-time-code`, ni uno
invisible, de un píxel, desactivado o de solo lectura. El usuario se busca **hacia
atrás** desde la contraseña, nunca hacia delante, porque hacia delante está el
buscador de la cabecera siguiente.

## Alternativas descartadas

**El desplegable dentro del campo, como Dashlane.** Es lo que se espera de un
gestor y llegará si hace falta. Hoy no se hace por lo dicho arriba: con una cuenta
no aporta nada, y trae toda la superficie de dibujar en la página de otro. Queda
apuntado en `docs/deuda.md` para decidirlo con el uso, no con la intuición.

**Que la contraseña la escriba el panel.** El panel podría pedir el secreto y
mandárselo al guion. Se descartó por un motivo pequeño y bueno: un sitio menos por
el que pasa una contraseña. El panel manda un identificador y quien pide el
secreto es el guion, que es el que tiene el campo. Además el panel se cierra solo
al perder el foco, que es la peor clase de sitio donde dejar algo.

**No rellenar solo, y exigir un clic siempre.** Es más seguro y es lo que hacen
1Password y Bitwarden por defecto. Se descartó porque **es lo que se decidió con el
cliente**, con estas palabras: «Solo, como Dashlane». El precio está en las
consecuencias, dicho entero.

**Enviar el formulario después de rellenar.** No. Se rellena; darle a «entrar» lo
da la persona.

## Consecuencias

**Lo que se gana:** el gesto que se repite veinte veces al día deja de necesitar
Dashlane.

**Lo que se paga, y hay que decirlo sin suavizarlo:**

- **Código nuestro en cada página https que se abra.** Un fallo ahí es un fallo en
  las páginas de otros. Lo que lo acota es que ese código no dibuja nada, no
  guarda nada y solo habla por un puerto con nombre.
- **Una contraseña escrita en el DOM sin que nadie la haya pedido.** Con relleno
  automático, cualquier guion que ya esté corriendo en esa página puede leerla.
  Eso exige que el atacante ya ejecute código en el dominio —donde también podría
  falsificar el formulario entero—, pero es un escalón menos que antes.
- **Un secreto cruzando el canal**, sin el borrado del portapapeles detrás.
- **El permiso más ancho que dan las dos tiendas**: `https://*/*` y un guion en
  todas las páginas. Es la revisión más estricta y la que más tarda.
- Y **queda fuera lo que no es https**: la página de administración de un router
  no se rellena. Es una carencia conocida, no un descuido.

**Lo que no cambia:** la bóveda sigue siendo la única que tiene los secretos
descifrados, el puente sigue sin saber abrirla, el emparejamiento sigue siendo por
persona y en la ventana, y **rellenar no cuenta como actividad** para el bloqueo
por inactividad. Esta última es la tercera vez que la regla aparece —el goteo de
iconos, el código de un solo uso y ahora esto— y aquí es donde más se notaría,
porque navegar por sitios guardados pregunta todo el rato.

## Verificación

**Comprobado aquí, sin navegador ni entorno gráfico:**

- **Qué campo se rellena, en Chromium de verdad** (`navegador/pruebas/campos.spec.ts`,
  doce casos). Se ejercita el fuente compilado, no una copia, y contra un DOM
  auténtico: la detección se apoya en `getComputedStyle`,
  `getBoundingClientRect` y `compareDocumentPosition`, y en un DOM simulado los
  tres devuelven lo que se les haya enseñado a devolver. **Media tabla son casos
  donde lo correcto es no rellenar nada**: el buscador de la cabecera, el
  formulario de registrarse, el escondido, el de un píxel, el campo de texto
  suelto.
- **Que escribir vale donde el formulario es de React**, que es media web. Se imita
  el accesor que React pone sobre el elemento y se comprueba lo que de verdad
  importa: que su registro interno **todavía no** se ha puesto al día cuando llega
  el evento. Es la misma trampa que en la 2.12.x dejó el botón de cifrar sin
  activarse al pegar, ahora en casa ajena.
- **Que `rellenar` no entrega la entrada de un sitio a otro**, con el
  identificador correcto en la mano, sin testigo y sobre `http://`.
- **La tubería entera**, de los bytes que manda el navegador al fichero de la
  bóveda y vuelta, incluida la contraseña.
- **Que el freno frena.** Y aquí hay una corrección de la entrega 1 que conviene
  leer: el contador vivía **en la conexión**, y la extensión abre **una conexión
  por pregunta**, así que el tope de sesenta por minuto no se alcanzaba jamás. La
  prueba tampoco lo veía porque le pasaba un contador hecho a mano. Ahora los
  frenos cuelgan del servidor y la prueba abre una conexión por pregunta.

**Comprobado en un Mac de verdad (2026-09-10, en la 2.18.1):**

- **Rellena solo al cargar la página, y acierta el formulario**, en `login.brevo.com` y en
  `dash.cloudflare.com`. **En Chrome y en Firefox.**
- **Y el botón «Rellenar» del panel escribe** tras borrar los campos a mano, que es lo que falló en la
  2.18.0 y costó la 2.18.1: eran dos fallos encadenados, los dos fuera de la detección de campos —el
  oyente del panel contestando también desde las tramas de otro origen, y `yaRellenados` bloqueando un
  relleno pedido por una persona—. Están contados en `CLAUDE.md` y en `docs/sesiones.md`.

**Lo que no se puede comprobar aquí, y hay que mirar en el Mac:**

- **Si los campos de un banco de verdad se detectan.** Ninguna prueba puede
  decirlo: los doce casos son formularios que he escrito yo, y lo que hay ahí
  fuera lo han escrito otros. Dos sitios corrientes ya aciertan; falta uno difícil.
- **Si rellenar solo es demasiado.** Si aparecen rellenos que nadie quería, la
  respuesta no es afinar la detección a ciegas: es un interruptor en Ajustes.
- **Si un clic más en el panel molesta** cuando hay varias cuentas. De eso depende
  si el desplegable en el campo se hace.
- **Y si en Firefox hace falta conceder el permiso de sitio a mano**, que en MV3 no
  se da al instalar. Si hace falta, se ve enseguida: la extensión lo dice con una
  frase que explica dónde darlo, en vez de contestar que ahí no se rellena.
