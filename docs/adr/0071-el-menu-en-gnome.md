# ADR 0071 · En GNOME el menú va en la cabecera, no en una barra

**Fecha**: 2026-11-23
**Estado**: aceptada
**Deroga** la parte de la [ADR 0015](0015-la-barra-de-menus.md) que dice que la barra de menús se
monta en los tres sistemas.

## Contexto

La ADR 0015 decidió **construir la barra de menús entera**, en español, porque los roles de Wails
traen los rótulos en inglés escritos a fuego en su Objective-C y no hay forma de traducirlos. Esa
razón sigue siendo cierta — y **sólo explica macOS**, que es donde el sistema exige que haya un menú.
De ahí se pasó a montarla también en Windows y en Linux sin que nadie lo decidiera.

El cliente lo vio al probar la **2.77.0 en su Ubuntu**: «veo que tiene menú superior». Tiene razón en
que chirría. En GNOME **ninguna aplicación lleva barra de menús**: lo que hay es un botón en la
cabecera, y una banda de «Archivo · Edición · Ver» se reconoce como una aplicación de GTK2.

Y hay un antecedente en casa: en **Apolo el menú es sólo de macOS**
(`#[cfg(target_os = "macos")] mod menu_macos`), con el mismo porqué escrito en su fuente —sin él,
Tauri pone el suyo en inglés—. Aquí estaba al revés.

## Decisión

**En GNOME no hay barra de menús. Lo que había se reparte en tres sitios**, y se eligió sobre
capturas entre tres opciones —dejarla, quitarla sin más como Apolo, o quitarla y poner el botón—:

1. **Un botón de menú en la cabecera**, con lo que la barra tenía **y no está en ninguna otra
   parte**: los dos enlaces de ayuda. Y, porque un menú principal de GNOME los lleva siempre,
   «Buscar actualizaciones…» y «Acerca de Esfinge», que además siguen en Ajustes.
2. **Los atajos de Ctrl+1 a Ctrl+6**, que colgaban del menú «Ver», los repone la ventana con el
   teclado. **Las seis entradas no vuelven a ningún menú**: son la barra lateral, y repetirlas sería
   decir dos veces lo mismo.
3. **Nada más.** «Edición» la resuelve el propio webview en Linux, y «Minimizar» y «Pantalla
   completa» las tiene el gestor de ventanas de GNOME.

### Lo que no se cambia sin preguntar

- **Quién lleva barra de menús se decide en un solo sitio**: `app.MenuDelSistema`. Lo lee `main.go`
  para pasarle —o no— el menú a Wails, que es una opción de arranque, y lo lee la interfaz por
  `Plataforma`. Escrito dos veces, una ventana acaba **con dos menús o sin ninguno**.
- **El botón va en el marcado de los cuatro sistemas y lo esconde el CSS**, por lo mismo que la barra
  de título (ADR 0066): montarlo sólo donde se usa rompe que el marcado sea el mismo (ADR 0064) y la
  simulación por URL, que cambia de aspecto sin recargar. Escondido con `display: none` **no está en
  el árbol de accesibilidad ni en el orden de tabulación**: donde no se usa, no existe.
- **El menú del sistema y el de la ventana llaman a la misma función** (`obedecer` en `App.tsx`). Son
  dos puertas a lo mismo y escritas por separado se desalinean solas.

### Windows queda abierto, y a propósito

Allí Wails pone un menú **nativo de Win32**, y desde la ADR 0066 esa ventana es `Frameless`: **no se
sabe si la barra llega a dibujarse**. Eso sólo se ve en un Windows y ahora mismo no abre (ver
`deuda.md`), así que se deja como está y se decide cuando se pueda mirar. Lo que sí se puede decir
desde aquí: una barra de menús clásica tampoco es de WinUI, y **la hamburguesa allí ya está cogida**
—es la que pliega la barra lateral—, así que si hiciera falta un botón habría que buscarle otra
forma.

## Alternativas descartadas

**Dejarla.** Funciona y no se pierde nada; lo único que pasa es que en GNOME no la lleva ninguna
aplicación. Se descartó al verla en capturas al lado de las otras dos.

**Quitarla sin más, como Apolo.** Es lo más barato y deja **la documentación sin forma de llegar**:
los dos enlaces de ayuda sólo estaban en el menú. Habría habido que meterlos en Ajustes, que es
donde no los busca nadie.

**Reponer también «Ver» en el botón.** Repetir en un menú las seis secciones que ya son la barra
lateral. Lo que de verdad faltaba de ahí eran los atajos, no las entradas.

**Decidir Windows a la vez.** Habría sido decidir sobre lo que no se puede mirar.

## Consecuencias

- En GNOME la ventana empieza en su cabecera, como cualquier aplicación de Ubuntu.
- **Los atajos ya no los sirve el sistema sino la ventana** donde no hay menú. Van contra el mismo
  `obedecer`, así que hacen exactamente lo que hacían.
- La ADR 0015 se queda con lo que sigue siendo cierto: **dónde hay menú, se construye entero**, en
  español, porque los roles de Wails no se pueden traducir.

## Verificación

**Lo que se comprobó.** `make comprobar`, `make e2e` y cinco pruebas nuevas: que en GNOME el botón
está y lleva los dos enlaces de ayuda con su `target="_blank"`, que se cierra con Escape, que **donde
el sistema pone su barra el botón no existe** —no que esté apagado—, que los atajos de Ctrl+1 a
Ctrl+6 siguen cambiando de sección, y que **la tabla de `plataforma.ts` dice lo mismo que
`app.MenuDelSistema`** (`TestElMenuEsElMismoEnLosDosLados`, hermana de la del marco).

**Cada una mutada para verla roja**: el `data-menu` del escondite cambiado por uno que no existe, el
oyente del teclado desconectado, la entrada de `linux` puesta a `true` y una entrada borrada de la
tabla.

**Lo que no se ha comprobado**: que en una Ubuntu de verdad la barra haya desaparecido. Aquí no se
puede: esa barra **la dibuja GTK por fuera del webview**, así que no sale en ninguna captura y lo que
se mira desde aquí es sólo el botón que la sustituye. Y en Windows, si había barra o no, sigue sin
saberse.
