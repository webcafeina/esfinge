# Publicar la extensión en las tiendas: los pasos del cliente

Lo que solo puede hacer quien tiene las cuentas de Webcafeína (ADR 0033). El resto —el código, las
fichas, las imágenes y la subida de cada versión— lo hace el repositorio. **Lo marcado como «por
confirmar» no se ha podido comprobar sin la cuenta**: si en la pantalla pone otra cosa, manda la
pantalla, y hay que apuntarlo aquí.

Los textos de las fichas están en [`ficha.md`](ficha.md) y las imágenes en `docs/tiendas/imagenes/`.

## El orden

**La 2.22.0 no trae los zip de las tiendas** —el de Chrome sin la clave y el de código fuente—: se
añadieron a la publicación después. Así que:

1. **Las cuentas de Mozilla y de Google**, y las claves de Mozilla guardadas en GitHub (apartados 1 y 2.1-2.2).
2. **Publicar una versión nueva.** Con las claves puestas crea la ficha de Firefox, y deja en la publicación
   el zip de Chrome sin la clave.
3. **La primera subida a Chrome, a mano**, con ese zip (apartado 2.3), y el identificador que dé la tienda.
4. La cuenta de servicio de Chrome y sus secretos (apartado 2.4), para que las siguientes suban solas.

---

## 1. Firefox (addons.mozilla.org)

1. **Cuenta de Mozilla** con info@webcafeina.com, en https://addons.mozilla.org, con **verificación en
   dos pasos** activada: sin ella no deja entrar en el panel de desarrollador.
2. Aceptar el acuerdo de distribución de complementos, si lo pide.
3. **Crear las claves de la API** en https://addons.mozilla.org/es-ES/developers/addon/api/key/ :
   salen un «emisor JWT» y un «secreto JWT».
4. **Guardarlas en GitHub** como secretos del repositorio (Ajustes → Secrets and variables → Actions):
   - `AMO_API_KEY`: el emisor JWT.
   - `AMO_API_SECRET`: el secreto JWT.

**No hace falta subir nada a mano**: con las claves puestas, la siguiente publicación crea la ficha y
sube la versión con su código fuente. **Hecho con la 2.22.1 (2026-09-15)**: la versión quedó esperando
revisión en https://addons.mozilla.org/es-ES/developers/addon/esfinge/versions/6486838. Después, en el panel de AMO:

- **La política de privacidad, en texto y no como dirección**: Mozilla no tiene un campo de URL, sino la
  casilla «Esta extensión tiene una política de privacidad» con un cuadro donde se pega el texto. Está en
  [`privacidad-amo.txt`](privacidad-amo.txt), sacado de la web. La descripción de la ficha ya enlaza además
  la de la web, que es lo que Mozilla recomienda. La pantalla exacta del panel donde está la casilla, por
  confirmar.
- **Añadir las capturas** de `docs/tiendas/imagenes/` (por confirmar si se pueden subir por la API).
- Revisar que la ficha haya salido en español: el idioma se pide como `es-ES` y está **por confirmar**
  que AMO lo acepte así en la primera subida.

## 2. Chrome (Chrome Web Store)

1. **Cuenta de desarrollador** en https://chrome.google.com/webstore/devconsole con info@webcafeina.com.
   **El correo no se puede cambiar después.** Tiene un pago único de registro (por confirmar el importe:
   se habla de 5 $) y pide **verificación en dos pasos**.
2. **Declararse comerciante** (*Trader*) y rellenar los datos de Webcafeína: **nombre legal, dirección y
   teléfono**, que se verifica por SMS. **Salen públicos al pie de la ficha**, como se decidió.
3. **Primera subida, a mano** —la API no crea fichas—:
   1. «Añadir elemento» y subir el zip de tienda de Chrome de la última publicación
      (`esfinge-extension-<versión>-chrome-tienda.zip`, sin la clave de desarrollo).
   2. **Apuntar el identificador** que da la tienda —32 letras— y **pasármelo**: hay que añadirlo a Esfinge
      para que la aplicación deje hablar a la extensión de la tienda.
   3. Rellenar la ficha con `ficha.md`: descripción, categoría, idioma, capturas y mosaico de
      `docs/tiendas/imagenes/`, web y soporte.
   4. Rellenar **«Privacy practices»** con `ficha.md`, tal cual.
   5. Visibilidad **pública**, y enviar a revisión. Puede tardar días; con `https://*/*` y
      `nativeMessaging`, quizá semanas.
4. **Para que las siguientes versiones suban solas** (por confirmar los nombres exactos de las pantallas):
   1. En Google Cloud (https://console.cloud.google.com), crear un proyecto, activar la **Chrome Web Store
      API** y crear una **cuenta de servicio**, con una clave JSON.
   2. En la consola de la tienda, en la cuenta, **añadir el correo de esa cuenta de servicio**.
   3. Apuntar el **ID de editor** (*Publisher ID*), en los ajustes de la consola.
   4. Guardar en GitHub como secretos: `CWS_CUENTA_DE_SERVICIO` (el JSON entero), `CWS_EDITOR` (el ID de
      editor) y `CWS_EXTENSION` (el identificador de la extensión).

## 3. Lo que hago yo cuando me pases el identificador de Chrome

- Añadirlo a `extensionesDeChrome` (`internal/app/manifiestos.go`) y publicar una versión de Esfinge,
  para que la aplicación deje entrar a la extensión de la tienda.
- Cambiar «Muy pronto en las tiendas» de la web por los enlaces de verdad.
- Y cuando las dos fichas estén en marcha, **dejar de colgar los zip de la extensión** en cada
  publicación de GitHub.
