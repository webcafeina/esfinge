# Lo siguiente

Última actualización: **2026-09-07**

Por prioridad. Lo cerrado se tacha y se queda, con la fecha: saber qué se descartó vale tanto como
saber qué se hizo.

## Alta

- **Probar la aplicación en un Mac de verdad.** Nunca se ha abierto en un escritorio, solo
  compilado. Sin esto, todo lo visual es una suposición. Depende del humano.
- **Montar el DMG en un Mac.** Está armado y el flujo lo publica, pero cómo queda la ventana al
  montarla solo se ve allí: `create-dmg` coloca los iconos hablando con el Finder y eso puede salir
  distinto en un runner sin sesión gráfica.
- **Que el Finder enseñe el icono de los `.esf`.** La asociación está declarada y la aplicación
  abre con el fichero, pero falta ver si el icono del documento sale.

## Media

- **Ventana con vibrancy de verdad.** Wails permite pedir ventana translúcida al sistema
  (`WindowIsTranslucent`, `WebviewIsTransparent`). Ahora la barra imita el efecto con CSS, que no es
  lo mismo: el desenfoque de macOS toma lo que hay detrás de la ventana, no dentro.
- **Barra de menús propia.** Ahora solo está el menú de fábrica. Faltan las órdenes de siempre:
  cifrar, descifrar, y sus atajos de teclado.
- **Recordar la última carpeta usada** en los diálogos de abrir y guardar.
- **Un modo para tandas grandes**: cifrar cincuenta ficheros son cincuenta derivaciones de clave,
  medio segundo cada una. Se puede hacer en paralelo.

## Baja

- **Traducción al inglés.** Hoy está todo en español, a propósito. Solo hace falta si el programa
  sale de la casa.
- **Actualización automática.** Requiere firmar, que está descartado.
- **Empaquetar para Homebrew** (`brew install --cask esfinge`), que exige una URL estable y firma.

## Cerrado

- ~~Convertir la herramienta de terminal en aplicación de escritorio~~ → hecho en la 2.0.0 (2026-09-07).
- ~~Arreglar el arrastrar y soltar~~ → hecho en la 2.0.1 (2026-09-07).
- ~~Doble clic en un `.esf` en macOS~~ → hecho en la 2.0.1 (2026-09-07).
- ~~Instaladores de los tres sistemas~~ → DMG, NSIS y `.deb`, publicados por `publicar.yml` al
  empujar una etiqueta (2026-09-07).
- ~~El arranque de cinco segundos en terminales que no contestan~~ → desapareció al retirar la
  interfaz de terminal, que era quien lo causaba (2026-09-07).
