# La extensión de Esfinge

Consulta la bóveda desde el navegador. Es la **entrega 1** de la fase 2: enseña
las cuentas que tienes del sitio que estás mirando y copia lo que le pidas.
**Todavía no rellena formularios**, eso es la entrega siguiente.

## Lo que hay que saber antes de mirar el código

**Por aquí no pasa ningún secreto.** La contraseña la copia Esfinge al
portapapeles del sistema, no la extensión, y por el canal solo vuelve cuánto
tardará en borrarse. Eso hace dos cosas a la vez: que un secreto no viva nunca en
el navegador, y que el borrado del portapapeles que Esfinge ya hacía desde la
2.12.0 valga también aquí.

**La dirección de la pestaña la da el navegador, no la página** (`chrome.tabs`).
Es la diferencia entre preguntar por un sitio y preguntar por lo que un documento
dice que es.

**Y no se guarda nada de lo que se pregunta**: ni el sitio, ni las cuentas, ni
cuándo. Un caché aquí sería la lista de sitios de tu bóveda escrita sin cifrar en
el perfil del navegador, que es justo lo que Esfinge cifra en su disco. Lo único
que se guarda es el testigo del permiso.

## Cómo se construye

```sh
make extension     # desde la raíz: deja dist/chrome y dist/firefox
```

Son dos porque los manifiestos no son el mismo: Chrome quiere un
`service_worker` y Firefox una lista de `scripts`.

## Cómo se prueba a mano, hoy

Hace falta **Esfinge instalada** y el canal encendido en sus Ajustes.

**Firefox** funciona ya, porque su identificador lo elegimos nosotros:

1. `make extension`
2. En `about:debugging` → «Este Firefox» → «Cargar complemento temporal», elige
   `navegador/dist/firefox/manifest.json`.
3. Enciende el canal en los Ajustes de Esfinge.
4. Abre una página de la que tengas una cuenta guardada y pulsa el botón.
5. La primera vez, Esfinge preguntará en su ventana si permite ese navegador.

**Chrome, Edge, Brave, Vivaldi y Opera** funcionan igual, y sin pasos extra:

1. `make extension`
2. En `chrome://extensions`, activa «Modo de desarrollador» y pulsa «Cargar
   descomprimida»; elige la carpeta `navegador/dist/chrome`.
3. Lo demás, igual que en Firefox.

El identificador **está fijado**: el manifiesto lleva una clave pública (`key`) y
Chrome deriva el identificador de ella, así que es el mismo en cualquier máquina
y en cualquier instalación. Es `jkkadfdagaojlgffkcboniepfgjkeenk`.

Esa clave **no firma nada y no es un secreto**: solo decide el identificador. La
privada ni se guarda ni hace falta.

Lo que queda por saber es el identificador que asigne la **tienda** al subirla. Si
no coincide con éste, se añade a `extensionesDeChrome` en
`internal/app/manifiestos.go` y ya. Para probar cualquier otra extensión sin
publicar sigue estando `ESFINGE_EXTENSIONES`, con los identificadores separados
por comas.

## Lo que falta

- Rellenar formularios (entrega 2).
- Guardar y actualizar contraseñas desde la página (entrega 3).
- Las tiendas, y con ellas el identificador fijo de Chrome (entrega 5).
- Windows: el manifiesto va al registro y todavía no se escribe.
