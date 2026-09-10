# La extensión de Esfinge

Rellena las contraseñas de tu bóveda de Esfinge. Va por la **entrega 2** de la
fase 2: con una cuenta guardada del sitio, rellena sola al cargar la página; con
varias, se elige en el panel.

## Lo que hay que saber antes de mirar el código

**En la página no se dibuja nada.** Ni desplegable sobre el campo, ni icono
dentro, ni marco flotante. El guion de página lee el formulario, escribe en él y
calla. Es la decisión de forma de la entrega ([ADR
0028](../docs/adr/0028-rellenar-en-la-pagina.md)) y lo que más reduce el riesgo
que la entrega trae: dibujar en la página de otro obliga a un marco de nuestro
origen y a pelearse con el `z-index` de cada sitio, y todo eso es superficie.

**La contraseña vive dentro de una función y en ningún otro sitio.** Llega por el
canal, se escribe en el campo y se va con la llamada: ni una variable de módulo,
ni `storage`, ni un registro. Esfinge no puede comprobarlo desde fuera, así que es
una promesa de este código —está dicho así en `docs/seguridad.md`—. Y lo que se
copia al portapapeles sí lo copia Esfinge, no la extensión, con lo que hereda el
borrado que Esfinge hace desde la 2.12.0.

**La dirección de la pestaña la da el navegador, no la página.** Desde la entrega 2
eso es una línea y no una frase: lo que llegue en el campo `origen` por el puerto
de una página **se tira** y se pone `sender.tab.url`. Una página no puede decir de
qué sitio es.

**Y nunca en un marco de otro origen, ni sobre `http://`.**

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

## Cómo se comprueba

```sh
pnpm run comprobar   # tipos, permisos del manifiesto y las pruebas de campos
```

Las pruebas ejercitan **qué campo se rellena**, en un Chromium de verdad y contra
el fuente compilado en memoria. Media tabla son casos donde lo correcto es **no
rellenar nada**: el buscador de la cabecera, el formulario de registrarse, el
escondido, el de un píxel. Equivocarse de campo es escribir una contraseña donde
la lea alguien, y es lo único de esta entrega que puede hacer daño.

## Lo que falta

- Guardar y actualizar contraseñas desde la página (entrega 3).
- El código de un solo uso, rellenado también (entrega 4).
- El desplegable dentro del campo, **si el uso dice que hace falta**.
- Las tiendas, y con ellas el identificador fijo de Chrome (entrega 5).
- Windows: el manifiesto va al registro y todavía no se escribe.
