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

**Chrome** necesita un paso más, y no es un descuido: su identificador **lo
asigna la tienda** al subir la extensión por primera vez, así que hasta entonces
no hay nada que escribir en el manifiesto que autoriza a lanzar el puente. Para
probar antes:

1. Cárgala descomprimida desde `chrome://extensions`.
2. Copia el identificador que Chrome le haya dado.
3. Arranca Esfinge con `ESFINGE_EXTENSIONES=<ese identificador>` y vuelve a
   encender el canal en Ajustes, para que reescriba el manifiesto.

## Lo que falta

- Rellenar formularios (entrega 2).
- Guardar y actualizar contraseñas desde la página (entrega 3).
- Las tiendas, y con ellas el identificador fijo de Chrome (entrega 5).
- Windows: el manifiesto va al registro y todavía no se escribe.
