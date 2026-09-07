# ADR 0020 — Windows y Linux, con el marco del sistema y su propia forma

**Fecha:** 2026-09-08 · **Estado:** aceptada · **Revisar si** aparece un cliente en Windows o Linux

## Contexto

La [0019](0019-estructura-de-macos.md) llevó la ventana a la estructura de macOS y dejó a Windows y
Linux con esa misma forma, que ahí no es la de nadie. Tocaba darles la suya.

Los tres escritorios modernos se organizan igual de fondo —navegación a un lado, contenido al otro—,
así que la estructura no cambia. Lo que cambia son las formas, las densidades y **quién dibuja el
marco de la ventana**.

## Decisión

**El marco lo dibuja el sistema** en Windows y en Linux. Solo macOS se queda sin barra de título.

Dentro, cada uno a lo suyo, con un atributo `data-sistema` en la raíz del que cuelga todo el CSS:

- **Windows**, panel de navegación de Fluent: filas de 40 px, radios de 4 en los controles y 8 en las
  tarjetas, y sobre todo **la barra de acento a la izquierda de la fila activa**, que es lo que
  distingue un `NavigationView` de una barra lateral cualquiera. En Fluent el relleno de la selección
  es tenue y quien dice «estás aquí» es esa marca.
- **Linux, a lo GNOME**: cabecera con el **título centrado y en negrita**, con su línea debajo, que es
  lo primero que se reconoce de una aplicación de GNOME; filas de 36 px, tarjetas a 12 de radio como
  libadwaita, y líneas de un píxel entero, porque el escritorio no dibuja a media resolución como
  macOS.
- El **hueco de los semáforos** desaparece en los dos: sin barra de título propia no hay nada que
  esquivar. Y las **zonas de arrastre** también, porque la ventana ya tiene de dónde agarrarse.

## Alternativas descartadas

- **Ventana sin marco en los tres**, dibujando nosotros los botones de minimizar, maximizar y cerrar.
  Es lo que hacen las aplicaciones modernas de Windows 11 y de GNOME, y se parecería más. Pero
  entonces cerrar, maximizar, arrastrar y redimensionar pasan a ser código nuestro, **y aquí no hay
  ni Windows ni GNOME donde probarlo**. Demasiado que romper a ciegas.
- **Que Linux se pareciera a KDE**, o quedarse en algo neutro entre los dos. GNOME es el escritorio
  por defecto de Ubuntu, Fedora y Debian; en KDE se verá correcta aunque no de la casa.
- **Dejar Linux con la estructura de macOS.** Habría sido lo barato, pero es el único de los tres que
  sí se puede mirar desde esta máquina, así que era el que menos excusa tenía.

## Consecuencias

- Aparece `App.Plataforma()`, que devuelve lo que devuelve Go. La interfaz lo pone en
  `data-sistema` y **todo el CSS de cada sistema cuelga de ahí**, igual que el del vidrio cuelga de
  `data-vidrio`.
- **El servidor de desarrollo corre en Linux**, así que lo que se ve en el navegador es ya la variante
  de GNOME. Es la primera vez que una de las tres se puede mirar de verdad mientras se escribe.
- Windows sigue con Mica; Linux se queda opaco, y **no porque Wails no lo ofrezca**: sí lo hace, con
  `linux.Options.WindowIsTranslucent`. Es que sin desenfoque del compositor la transparencia de GTK
  enseña el escritorio a pelo, y eso no es vibrancia, es un agujero.

## Verificación

- Las 28 pruebas de interfaz, en los dos temas y dos tandas seguidas. Ahora ejercitan la variante de
  GNOME, que es la del servidor de desarrollo.
- Las tres variantes miradas una a una forzando el atributo, que es lo más cerca que se puede estar
  de Windows desde aquí.

**Lo que no se ha comprobado:** ni Windows ni GNOME de verdad. No hay ninguno de los dos en esta
máquina, y a diferencia de macOS tampoco hay quien los abra al otro lado.
