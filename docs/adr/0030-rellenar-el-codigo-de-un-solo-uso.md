# ADR 0030 — Rellenar el código de un solo uso

**Fecha:** 2026-09-14 · **Estado:** aceptada · **Continúa la [0028](0028-rellenar-en-la-pagina.md)
y la [0025](0025-los-codigos-de-un-solo-uso.md)** · **Revisar cuando** se use en sitios de verdad

## Contexto

Desde la 2.15.0 la bóveda calcula el código de un solo uso, y desde la entrega 1 el panel lo copia.
Lo que faltaba es lo que el cliente pidió al cerrar la sesión del 10 de septiembre: **que el código
también se escriba solo cuando el sitio lo pida**. Era la entrega 4 del plan y se adelantó.

La mitad estaba hecha: Go sabe calcularlo y `campos.ts` ya reconocía `autocomplete="one-time-code"`,
aunque solo para **excluirlo** de ser confundido con un usuario. Faltaba decidir tres cosas, y ninguna
es fontanería:

- **Por dónde sale.** Sería el segundo verbo que entrega un secreto hacia el navegador.
- **Dónde se escribe.** Los formularios de segundo factor no se parecen entre sí: un campo que lo
  declara, seis casillas de un carácter, o un campo que se llama `totp` y no dice nada más. Y la web
  está llena de campos de «código» que no lo son.
- **Cuándo.** Un código caduca, y uno que muere mientras alguien busca el botón de verificar se ve
  como que Esfinge calcula mal.

## Decisión

### Un verbo, `rellenar-codigo`, con las llaves y el freno de `rellenar`

Devuelve el código de **una** entrada y los segundos que le quedan. Pasa por `entradaDe` como todo lo
que sale hacia el navegador —testigo, origen que da el navegador, dominio registrable, bóveda abierta,
entrada de ese sitio— y **no cuenta como actividad**.

**Y gasta del mismo freno que rellenar la contraseña.** No por ahorrar un contador: un código dura
treinta segundos, pero junto a la contraseña es la cuenta entera, y dos frenos separados serían el
doble de margen para quien no debería tener ninguno.

### Dónde se escribe, en tres formas y en negativo

En orden de cuánto hay que creerse a la página:

1. **Lo declara el sitio** con `autocomplete="one-time-code"`, y es un solo campo entero. Si hay más
   de uno, no se sabe cuál y no se toca.
2. **Seis u ocho casillas, juntas.** Es la forma más común de los formularios de segundo factor.
   Cuenta como casilla un campo con `maxlength="1"`, uno con un `pattern` de una sola cifra, o uno
   que declara `one-time-code` dentro de un grupo. Muchas van cada una en su caja, así que se agrupa
   subiendo un nivel cuando la caja solo contiene esa casilla. **Cuatro no**: eso es un PIN.

   **Y las casillas se miran antes que el campo declarado**, que es la corrección de la 2.19.1: el
   formulario de Cloudflare son seis casillas que declaran **todas** `one-time-code` y **ninguna**
   tiene `maxlength="1"` —la primera admite seis cifras, para el autorrelleno del sistema—. Con el
   campo declarado primero, la regla veía seis y se callaba por no saber cuál; y sin `maxlength="1"`
   no eran casillas. Lo que sí llevan todas es `pattern="\d{1}"`.
3. **Por el nombre** —`otp`, `totp`, `mfa`, `2fa`, `two factor`, `one time`, `authenticat…`,
   `verification code`—, **solo si en la página no hay ninguna contraseña visible y el campo tiene
   cara de numérico**. Es la regla más débil y por eso la que más condiciones lleva.

**«code» a secas no cuenta, y es la ausencia importante**: es el código postal, el promocional, el de
la tarjeta regalo y el CVC, y todos tienen seis caracteres a menudo. Lo que el sitio declara como
tarjeta (`cc-*`) no se toca nunca. Y un código que no cabe —seis casillas, ocho cifras— **no se
escribe a medias**: no se escribe.

### Cuándo

- **Solo con una cuenta del sitio, y solo si Esfinge dice que tiene código.** Una Esfinge anterior a
  la 2.19.0 no lo dice, y ahí no se pide uno a ciegas. Con varias cuentas se elige en el panel, como
  la contraseña.
- **Si al código le quedan menos de tres segundos, se espera al siguiente.** Escribir uno que caduca
  antes de que alguien pulse «Verificar» produce un «código incorrecto» con el código de Esfinge
  puesto, que es la peor forma de fallar.
- **El botón «Rellenar» del panel hace las dos cosas**: el formulario de entrar si lo hay, y el código
  si lo hay y la cuenta lo tiene. Si hay formulario y el código falla, se cuenta que el formulario se
  ha rellenado.

## Alternativas descartadas

**Detectar por la palabra «code».** Es la regla que acierta en más formularios de segundo factor y la
que más veces escribe donde no debe. Un falso negativo es un clic en el panel; un falso positivo es un
segundo factor escrito en un formulario que no lo pidió.

**Un botón aparte en el panel para rellenar el código.** Ya hay dos formas de copiarlo; una tercera
fila de acciones para algo que se decide solo por lo que hay en la página no aporta nada.

**Pedir el código siguiente cuando al actual le queda poco**, en vez de esperar. Los servidores
aceptan casi siempre el código del periodo anterior —es para lo que dejan margen— y no siempre el
siguiente. Esperar tres segundos es seguro; adelantarse, no.

**No rellenar el código solo, y dejarlo al panel.** Es lo que pedía el cliente al revés.

## Consecuencias

**Lo que se gana:** entrar en un sitio con segundo factor deja de necesitar Dashlane también en el
navegador. Con esto, **el gesto completo de entrar se hace sin tocar el teclado**.

**Lo que se paga, y va dicho con las mismas palabras que la [ADR 0025](0025-los-codigos-de-un-solo-uso.md):**

- **Por el canal salen ya la contraseña y el segundo factor**, los dos sin el borrado del portapapeles
  detrás. Una bóveda abierta entregaba las dos cosas desde la 2.15.0; ahora las **escribe en la
  página**. El freno compartido limita cuántas, no que salgan.
- **Y en la página quedan escritas las dos.** Cualquier guion que ya corra en ese dominio puede leer
  del formulario contraseña y código. Para eso tiene que ejecutar código allí —donde también podría
  falsificar el formulario entero—, pero es el mismo escalón menos que la ADR 0028 ya dijo, ahora con
  el segundo factor también.

## Verificación

**Comprobado aquí:**

- **Dónde se escribe, en un Chromium de verdad** (`navegador/pruebas/campos.spec.ts`): el campo
  declarado, seis casillas sueltas y seis en cajas, y por el nombre; y **los que no**: el código
  promocional, el CVC, el código postal, cuatro casillas de un PIN, un campo `mfa_token` con una
  contraseña en la página, y uno escondido. Más que en casillas se escribe una cifra en cada una y, si
  el código no cabe, nada.
- **Que el código sale solo en su sitio, solo de una entrada con semilla y con el freno compartido**,
  en el servidor; y en la aplicación, **que es el de ahora** comparado con `internal/codigos`, que no
  cuenta como actividad y que no sale de la papelera.
- **La tubería entera**, de los bytes del navegador a la bóveda: `tieneCodigo` llega bien y el código
  sale para su sitio y no para otro.

**Comprobado después de publicar la 2.19.0 (2026-09-14):**

- **En Firefox, en un Mac, funciona todo lo demás**, y **en Cloudflare el código no se rellenaba**: el
  botón del panel decía que allí no había formulario. Con un diagnóstico pegado en la consola —solo la
  forma de los campos, sin valores— se vio el formulario de verdad, se copió tal cual a
  `campos.spec.ts` y la prueba **se vio fallar antes del arreglo**.
- **Y se probó la escritura contra el componente que usa Cloudflare**, `OTPField` de Base UI, el
  paquete, en React (`pruebas/otp-de-verdad.spec.ts`): el estado del componente queda con el código y
  el botón de verificar se activa. De ahí salió una corrección a una suposición mía: leyendo su código
  parecía que escribir las casillas seguidas no le valdría, se escribió un rodeo, y **la prueba dijo
  que sí le valía**. El rodeo se quitó.

**Y comprobado en un Mac con la 2.19.1 (2026-09-14): en Cloudflare el código se rellena solo, en
Firefox y en Chrome.** Era lo único que ninguna prueba de aquí podía decir: las pruebas usan su
formulario copiado y su componente, pero no su página.

**Sin comprobar:**

- **Otros formularios de segundo factor.** Hay componentes que solo reaccionan a pegar o a pulsaciones
  de tecla; con esos, `escribirCodigo` lee lo que ha quedado y **dice que no ha quedado puesto** en vez
  de contar que ha ido bien, pero no lo arregla.
- **Nada del guion de página está probado con la extensión cargada**: ni la espera de los tres
  segundos, ni el orden formulario-código del panel. Sigue en `docs/deuda.md`.
