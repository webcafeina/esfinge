# ADR 0019 — La ventana se organiza como una aplicación de macOS

**Fecha:** 2026-09-07 · **Estado:** aceptada · **Revisar si** se hace lo mismo en Windows y Linux

## Contexto

Desde la [0007](0007-aspecto-del-sistema.md) Esfinge no tiene identidad propia: usa la tipografía y
los controles del sistema. Pero se quedó en la superficie. **La estructura seguía siendo la de una
página web dentro de una ventana**: cinco pestañas centradas arriba en un control segmentado, un
panel de 560 px en medio y un pie con la versión. Ninguna aplicación de macOS se organiza así.

Para no diseñar de memoria, el humano dejó cuatro capturas de macOS 26 en `referencias/` —Ajustes
del sistema, App Store, y dos aplicaciones con barra lateral—. **De ahí salen las medidas, no de la
intuición**: los semáforos miden 12 pt clavados, así que sirven de regla para sacar la escala de
cualquier captura y medir el resto.

Medido así: **barra lateral de 224,6 pt**, **paso entre filas de 38,5 pt** y **semáforos a 21 pt** del
borde de arriba.

## Decisión

Dos columnas: barra lateral con la navegación a la izquierda, y a la derecha el título de la sección
y su contenido.

- **La ventana pierde su barra de título** (`mac.TitleBarHidden`): el contenido llega hasta arriba y
  los semáforos quedan encima de la barra lateral, como en Finder o Correo.

  Con `TitleBarHiddenInset`, que fue el primer intento, macOS dibuja **su propia banda de barra de
  herramientas** —el preajuste activa `UseToolbar`— justo donde va nuestro título, y se ve un fondo
  que no cuadra con el resto de la ventana. Aquí la barra de herramientas la dibujamos nosotros.
- **La barra lateral sustituye a las pestañas.** Ajustes va separado abajo, que es donde el sistema
  pone lo que configura la aplicación en vez de lo que se hace con ella.
- **El pie desaparece.** La versión ya estaba en Ajustes y «Modo desarrollo» se va a la esquina de la
  barra de herramientas.
- **Los formularios van en tarjetas agrupadas**, que es como el sistema junta lo que va junto.
- El ancho mínimo sube de 560 a **760 px**: 225 se los lleva la navegación.

**El control segmentado se queda** donde tiene sentido —Texto/Ficheros, los alfabetos del
generador—, que es justo para lo que lo usa el sistema: modos excluyentes dentro de una pantalla, no
navegación.

## Alternativas descartadas

- **Tres estructuras nativas a la vez**, una por sistema. Es a donde se va, pero de golpe habría
  sido publicar mucho código que aquí no se puede ejecutar. Va macOS primero, se prueba en un Mac, y
  Windows y Linux después.
- **Una barra lateral que se pliega al estrechar.** Es lo que hace el sistema, pero añade un estado
  más que recordar y probar. Se prefirió subir el ancho mínimo.
- **Iconos con SF Symbols.** Ya se sabe lo que pasa: no hay forma de comprobar desde CSS si la
  fuente está, y cuando no está salen cuadrados vacíos. Y con emoji, que fue el primer intento, la
  barra lateral parece cualquier cosa menos una aplicación del sistema: son de color y el sistema usa
  trazo monocromo. Se dibujan en SVG.

## Consecuencias

- **Sin barra de título, arrastrar la ventana deja de ser gratis.** Hay que declarar las zonas con
  `--wails-draggable`, y están en la barra lateral y la de herramientas; los botones se desmarcan uno
  a uno. Si esto se olvida, la ventana se queda clavada en la pantalla.
- **Las pruebas de interfaz cambiaron de selector.** Las secciones ya no son `role="tab"` sino filas
  de la barra lateral con `aria-current="page"`. Y aparece una colisión que no había: «Cifrar» nombra
  a la vez la sección y el botón que cifra, así que los selectores se acotan a `.lateral` o a
  `.contenido`.
- Las capturas de la portada se rehicieron: enseñaban la ventana vieja.

## Verificación

- Las 26 pruebas de interfaz, actualizadas y en verde en los dos temas, dos tandas seguidas.
- `make comprobar` y `make contraste`.
- Comparación con las capturas de referencia, que es lo que se puede hacer aquí.

**Lo que no se ha comprobado, y es lo que importa:** cómo queda en un Mac. Si los semáforos caen
donde deben, si la ventana se arrastra por donde se espera, y si la barra lateral pasa por una del
sistema. Un navegador no puede contestar a eso.
