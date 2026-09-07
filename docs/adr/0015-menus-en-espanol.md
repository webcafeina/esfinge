# ADR 0015 — La barra de menús se construye entera, para poder estar en español

**Fecha:** 2026-09-07 · **Estado:** aceptada · **Revisar si** Wails localiza sus roles

## Contexto

Todo lo que se ve dentro de la ventana está en español desde la primera versión
([ADR 0005](0005-espanol-y-mayuscula-inicial.md)). La barra de menús del sistema no: salía en inglés
—«Edit», «Paste and Match Style», «Quit Esfinge»— y quedaba como un remiendo encima de una
aplicación que por dentro habla español.

Wails trae roles (`menu.AppMenu()`, `menu.EditMenu()`, `menu.WindowMenu()`) que producen esos menús,
y son los que salen de fábrica cuando no se declara ninguno. Los rótulos **están escritos a fuego en
su Objective-C** (`pkg/.../darwin/WailsMenu.m`): no son los menús que localiza macOS a partir del
idioma de la aplicación, así que no hay forma de traducirlos desde fuera. O se usan en inglés, o se
construyen.

## Decisión

Se construye la barra entera, en español, igual en los tres sistemas (`menu.go`). El primer menú se
llama «Esfinge» en macOS —donde el sistema lo trata como menú de aplicación lleve el nombre que
lleve— y «Archivo» en Windows y Linux.

Como sin los roles no hay selectores nativos detrás, **las acciones de edición las hace la ventana**:
el menú manda una orden (`EventoOrden`) y la interfaz la ejecuta sobre el campo que tiene el foco,
que es lo que Go no puede saber. El pegar es el único que necesita las dos mitades: **el portapapeles
del sistema lo lee Go** —el navegador tiene prohibido leerlo sin un permiso que en una ventana de
escritorio no hay a quién pedirle— y el texto lo coloca la interfaz en el cursor.

De paso, el menú **Ver** da atajos a las cinco pantallas (⌘1 a ⌘5), que antes no había forma de
alcanzar sin ratón.

## Alternativas descartadas

- **Dejar los roles de Wails.** Es una línea de código y funciona, pero deja la aplicación medio en
  inglés justo en la parte que más se mira en un Mac.
- **Localizar la aplicación con un `es.lproj`.** No sirve: macOS localiza los menús que vienen de sus
  propias plantillas, no cadenas escritas en el código de Wails.
- **Parchear Wails.** Un `replace` en `go.mod` apuntando a una copia propia se convierte en deuda en
  cuanto salga la siguiente versión.

## Consecuencias

- Se pierden los añadidos que trae el menú de aplicación nativo de macOS —Servicios, «Ocultar
  otras»—. A cambio, todo lo que queda está en español y hace lo que dice.
- **Copiar, cortar, pegar y deshacer pasan ahora por nuestro código.** Es el riesgo real de esta
  decisión: si algo falla ahí, falla pegar una contraseña, que es lo que más se hace con Esfinge.
  Por eso las órdenes tienen prueba, y por eso `Ordenar` está expuesta al puente: es la única forma
  de ejercitar desde un navegador lo que en la aplicación dispara un menú que no existe fuera del
  escritorio.
- `document.execCommand` está marcado como obsoleto y se usa igual: para los comandos de edición
  sigue siendo lo único que se comporta igual en WKWebView y en WebView2.

## Verificación

- Prueba de interfaz en los dos temas: una orden cambia de pantalla, y «seleccionar todo» actúa
  sobre el campo que tiene el foco.
- Prueba de Go de que las órdenes salen por el mismo camino de eventos que el progreso, incluida la
  de pegar con su texto.

**Lo que no se ha comprobado:** los menús dibujados de verdad. No hay Mac ni Windows aquí, así que
falta ver que los rótulos salen donde deben, que ⌘C y ⌘V siguen funcionando dentro de los campos, y
que el menú de aplicación de macOS respeta el nombre.
