# ADR 0031 — El icono con estados, y la marca en el campo rellenado

**Fecha:** 2026-09-14 · **Estado:** aceptada · **Matiza la [0028](0028-rellenar-en-la-pagina.md)** ·
**Revisar cuando** se use unos días, y al enviar la extensión a las tiendas

## Contexto

Con la 2.19.1 funcionando en su Mac, en Firefox y en Chrome, el cliente pidió tres mejoras visuales
antes de guardar contraseñas desde la página. Dos de ellas tocan decisiones escritas:

1. **El icono de la barra se veía pequeño y regular**, y quería que **dijera el estado de la
   extensión sin tener que abrir el panel**. Hasta la 2.19.1 el icono era el de la aplicación
   reducido —una placa con margen— y no cambiaba nunca.
2. **Con el relleno automático, quien mira puede no saber por qué** ha aparecido la contraseña.
   Quería algo en el campo que dijera que ha sido Esfinge. Y la [ADR 0028](0028-rellenar-en-la-pagina.md)
   decía, con sus razones, **«en la página no se dibuja nada»**.

La tercera —el panel menos básico— amplía la [ADR 0029](0029-el-panel-de-la-extension-es-esfinge.md)
y está escrita allí.

Todo se decidió con él por preguntas con opciones, y lo elegido está abajo con sus palabras.

## Decisión

### El icono de la barra es la silueta, sin placa

`build/icono-barra.svg` reutiliza los trazados de `icono.svg` —tocado de oro con sus bandas, rostro,
ojos y ojal— **sin la placa y con el `viewBox` ajustado al dibujo**, así que ocupa el cuadro entero.
**Con un contorno de piedra**: sin placa detrás, el oro solo sobre una barra clara no se ve, y la
piedra sola no se ve sobre una oscura. Tres variantes: activa (oro), apagada (grises) y cerrada
(grises con un candado dibujado en el icono, porque la insignia del navegador solo admite texto).

El icono de las tiendas y de la página de extensiones sigue siendo el de la aplicación, con placa.

### Refleja el estado de cada pestaña

| Situación | Icono | Insignia |
|---|---|---|
| Abierta, nada de este sitio | activo | — |
| Abierta, N cuentas | activo | `N` (hasta `9+`) |
| Ha rellenado esta página | activo | `✓` verde |
| Bóveda cerrada | cerrado | — |
| Sin Esfinge, sin permiso, demasiadas preguntas | apagado | `!` ámbar |
| Página sin https | apagado | — |

Y al pasar el ratón, la frase que lo explica: «Esfinge · 2 cuentas de login.brevo.com».

**Los problemas ganan al ✓**: una página rellenada con la bóveda ya cerrada enseña el candado.

Todo lo que decide va en una **función pura** (`navegador/src/insignia.ts`), probada entera; al
trabajador de fondo solo le queda preguntar y pintar.

### Se pone al día solo, y también cada minuto

Al cambiar de pestaña, al cargar una página, al cambiar de ventana, al abrir el panel —con lo que el
panel acaba de saber, sin volver a preguntar— y **con una alarma cada minuto para la pestaña activa**.
El cliente eligió la alarma sabiendo el precio: **un permiso más (`alarms`)** y arrancar el puente
una vez por minuto. Es la única forma de que el candado aparezca si la bóveda se cierra sola: Esfinge
no puede avisar a la extensión.

Tres cautelas en `fondo.ts`:

- **El refresco no pide permiso cuando falta.** El `pedir` del panel se empareja solo, y eso hace
  aparecer en la ventana de Esfinge «un navegador pide permiso»; desde el refresco, eso sería un aviso
  cada minuto sin que nadie hubiera tocado nada. Se pregunta con `consultar`, que no se empareja.
- **El freno de sesenta preguntas por minuto es del canal entero**, y lo gastan también el panel y las
  páginas. Así que se agrupan las ráfagas y no se repite la misma pestaña y dirección antes de cinco
  segundos, salvo la alarma y los cambios de dirección.
- **Nada se guarda en `storage`**: solo en la memoria del trabajador, que muere solo.

### En el campo rellenado: un filete, y un aviso de tres segundos

Esto **matiza la ADR 0028**. En la página ahora se dibuja algo, con estos límites:

- **El filete** va en el propio campo de la web, sin añadir ningún elemento: un anillo de oro por
  dentro y uno de piedra por fuera, para que se vea en webs claras y oscuras. Se quita con lo primero
  que escribe una persona —un evento `isTrusted`—; los eventos que dispara Esfinge al rellenar no lo
  quitan. Devuelve el campo como estaba, estilo de la web incluido.
- **El aviso «Rellenado por Esfinge»** —o «Código rellenado por Esfinge»— sí es un elemento nuestro:
  **no se puede pulsar** (`pointer-events: none`), va en un `shadowRoot` **cerrado** que la web no
  puede leer ni restilar, **no lleva ningún texto que venga de la web ni de la bóveda**, hay uno a la
  vez y **se va solo a los tres segundos**. Solo en el relleno automático: desde el panel, la persona
  acaba de pulsar «Rellenar» y ya sabe por qué.

### Corregido tras probarla en Firefox (2.20.1)

El cliente probó la 2.20.0 en su Mac y pidió cuatro cambios, que se hicieron tal cual:

- **El filete, sin el anillo de piedra**: se leía como un borde negro alrededor del campo. Queda un
  anillo de oro con un halo del mismo oro, más suave. **El precio, dicho**: el oro como línea sobre
  una web clara da 1,37-1,68:1, por debajo de los 3:1 de un indicador. El halo lo ensancha —en la
  captura de una web blanca se distingue— y quien dice que ha sido Esfinge es el aviso, no el filete.
- **El candado, como una insignia más**: en la 2.20.0 era un círculo pequeño y se veía mal al lado
  del número y del ✓. Ahora es una placa de piedra del tamaño de una insignia, abajo a la derecha, con
  el candado en blanco. Sobre una barra oscura la placa se funde y queda el candado, igual que le pasa
  a la insignia del número.
- **La esfinge del aviso, rellena**: era la marca a trazo, que tiene la cara hueca, y sobre la piedra
  del aviso el hueco se veía como una cara negra. Ahora es la silueta de la barra, con la cara en
  crema, medida a 10,36:1 sobre la piedra.
- **Y el marco de la fila del panel, solo con el teclado** (ADR 0029).

### Y otra vez el candado, y el aviso más grande (2.20.2)

Con la 2.20.1 el cliente siguió viendo el candado demasiado pequeño, y pidió que fuera **como el ✓ y
el número, pero naranja oscuro con el candado en blanco**; y que el aviso fuera más grande.

- **La insignia del navegador solo admite texto**, así que el candado blanco no se le puede pasar. Se
  le ofrecieron las dos salidas: la insignia de verdad con el emoji 🔒, que el sistema pinta en color y
  no en blanco, o el candado dibujado en el icono imitando la insignia. **Eligió el dibujado.**
- **La placa ocupa ahora dos tercios del icono**, hasta los bordes, con el candado blanco más grueso.
  Es lo más grande que cabe: la insignia del navegador puede salirse del cuadro del icono, y un dibujo
  dentro del icono no.
- **Naranja `#c2410c`**, y no la piedra de la 2.20.1 —que se fundía con las barras oscuras— ni el ámbar
  del «!», que significa otra cosa. Blanco sobre ese naranja, **5,18:1**, medido.
- **El aviso, de 12 a 14 px**, con más relleno y la esfinge de 16 a 20 px.

## Alternativas descartadas

**El icono de la aplicación ampliado, o la cabeza sola sobre un círculo.** Se ofrecieron las dos; el
cliente eligió la silueta sin fondo. La placa a sangre se veía más grande y la cabeza sola se leía mejor
en pequeño, pero las dos se alejaban de la forma que ya se reconoce.

**Sin número de cuentas, o solo los problemas.** Más discretos: quien mira la pantalla no sabe si
tienes cuentas guardadas de esa web. El cliente prefirió saber de un vistazo si hay algo que rellenar.

**Refrescar solo al cambiar de pestaña.** Sin permisos nuevos y sin nada en segundo plano, pero con el
candado apareciendo tarde si la bóveda se cierra sola mientras sigues en la misma pestaña.

**Solo el filete, sin aviso.** Mantenía la ADR 0028 al pie de la letra, pero no dice que ha sido
Esfinge: se ve que el campo es distinto, no por qué, que era justo lo que se pedía.

**Una esfinge fija dentro del campo**, como 1Password. Siempre a la vista, pero hay que moverla con la
página cuando se desplaza o cambia de tamaño, y es lo que más se rompe según la web y más superficie
pone en la página.

## Consecuencias

- **En la página se dibuja**, aunque no se pueda pulsar y dure tres segundos. Un fallo del aviso sería
  un fallo en la página de otro; lo acotan la sombra cerrada, que no lleve datos y que se vaya solo.
- **Una web puede saber que usas Esfinge**, por el elemento del aviso o por el estilo del filete.
  Podía intuirlo ya por el relleno instantáneo; ahora es seguro. Y puede esconder o tapar el aviso,
  que es inofensivo.
- **El número de cuentas se ve en la barra**, para quien mire la pantalla.
- **Un permiso más, `alarms`**, que las tiendas revisan, y **`favicon` en Chrome** para el icono de la
  web en el panel (ADR 0029). Y el puente arranca una vez por minuto mientras el navegador esté abierto.
- **Los colores de la barra, las insignias y el aviso no son tokens**, porque no los pinta nuestro CSS.
  Se miden aparte en `internal/tema/extension_test.go`, que además **lee los ficheros** y falla si un
  color cambia en el SVG o en el TypeScript sin cambiar en la medición.

## Verificación

**Comprobado aquí:**

- **Contraste**, en `extension_test.go`: contorno de piedra contra las barras claras de Chrome y
  Firefox (13-14:1), oro y gris apagado contra las oscuras (4,3-8,5:1), el texto de las tres insignias
  (6,2-14:1) y el aviso (14:1 el texto, 8,4:1 la esfinge).
- **Qué enseña el icono en cada situación**, entero, en `navegador/pruebas/insignia.spec.ts`:
  singular, plural y `9+`, páginas sin https, cada motivo con su frase, que los problemas ganen al ✓ y
  que todas las frases empiecen en mayúscula.
- **El filete y el aviso en Chromium** (`navegador/pruebas/marcas.spec.ts`): el filete **no lo quita**
  un evento de Esfinge y **sí** una tecla de verdad, y devuelve el estilo que tenía el campo; el aviso
  tiene la sombra cerrada, no se puede pulsar, sustituye al anterior y se va a los tres segundos.
- **Capturas** (`pnpm run capturas`): la silueta con cada insignia sobre las cuatro barras, y el campo
  con filete y aviso en una web clara y una oscura. La barra es **una composición**: la de verdad no
  se puede capturar aquí.

**Visto por el cliente en Firefox con la 2.20.0**: el número y el ✓ del icono se ven bien; el
candado se veía pequeño, el filete tenía un borde negro y la esfinge del aviso una cara negra. Los tres
se corrigieron en la 2.20.1 y se miraron en capturas antes de publicar.

**Y con la 2.20.2 y la 2.20.3, en el Mac, en Firefox y en Chrome (2026-09-14): el cliente dice que
funciona y se ve todo bien**, candado incluido. No ha comentado expresamente si el candado aparece solo
al minuto de cerrarse la bóveda, así que eso sigue abajo.

**Sin comprobar, y hay que mirar en el Mac:**

- **Si la silueta se lee en la barra de verdad**, a su tamaño y con Retina.
- **Si la insignia cambia sola** al cambiar de pestaña, y **si el candado aparece como mucho un minuto
  después** de que la bóveda se cierre.
- **El filete y el aviso en Brevo y en Cloudflare**, que tienen sus propios estilos de campo.
- **Nada del trabajador de fondo está probado con la extensión cargada**: ni los eventos de pestaña
  ni la alarma. Sigue en `docs/deuda.md`.
