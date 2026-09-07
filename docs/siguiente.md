# Lo siguiente

Última actualización: **2026-09-08**

Por prioridad. Lo cerrado se tacha y se queda, con la fecha: saber qué se descartó vale tanto como
saber qué se hizo.

## Alta

- **Windows y Linux, a su estructura.** La 2.6.0 llevó macOS a barra lateral y los otros dos siguen
  con esa misma forma. Toca el panel de navegación de Windows 11 y la cabecera de GNOME. Depende de
  que la de macOS se dé por buena en el Mac.
- **Que el humano vea si salta el aviso de Gatekeeper** al actualizarse desde dentro. Si no salta,
  hay que quitarlo del LÉEME del DMG para el caso de la actualización.
- **Montar el DMG en un Mac.** Está armado y el flujo lo publica, pero cómo queda la ventana al
  montarla solo se ve allí: `create-dmg` coloca los iconos hablando con el Finder y eso puede salir
  distinto en un runner sin sesión gráfica.
- **Que el Finder enseñe el icono de los `.esf`.** La asociación está declarada y la aplicación
  abre con el fichero, pero falta ver si el icono del documento sale.

## Media

- **Barra de menús propia.** Ahora solo está el menú de fábrica. Faltan las órdenes de siempre:
  cifrar, descifrar, y sus atajos de teclado.

## Baja

- **Traducción al inglés.** Hoy está todo en español, a propósito. Solo hace falta si el programa
  sale de la casa.
- **Reemplazo automático del binario**, sin pasar por el instalador. Requiere firmar, que está
  descartado (ADR 0014).
- **Empaquetar para Homebrew** (`brew install --cask esfinge`), que exige una URL estable y firma.

## Cerrado

- ~~Ventana con vibrancy de verdad~~ → hecho en la 2.5.0, en la barra y el pie (2026-09-08).
- ~~Recordar la última carpeta de los diálogos~~ → hecho en la 2.5.0, una para abrir y otra para
  guardar (2026-09-08).
- ~~Un modo para tandas grandes~~ → en paralelo con tope desde la 2.5.0: veinte ficheros de 4,42 s a
  1,29 s (2026-09-08).

- ~~Copiar y pegar con los menús construidos a mano~~ → comprobado en el Mac: los atajos de ⌘ van
  bien (2026-09-08).
- ~~Convertir la herramienta de terminal en aplicación de escritorio~~ → hecho en la 2.0.0 (2026-09-07).
- ~~Arreglar el arrastrar y soltar~~ → hecho en la 2.0.1 (2026-09-07).
- ~~Doble clic en un `.esf` en macOS~~ → la 2.0.1 lo dio por hecho y estaba a medias: nadie escuchaba
  el evento. Cerrado de verdad en la 2.3.1, y en la 2.4.0 cada `.esf` abre además la pantalla que le
  toca según lo que lleve dentro. Comprobado en el Mac (2026-09-07).
- ~~Que la aplicación avise de las versiones nuevas~~ → hecho en la 2.1.0, con descarga comprobada y
  entrega al instalador (2026-09-07).
- ~~Instaladores de los tres sistemas~~ → DMG, NSIS y `.deb`, publicados por `publicar.yml` al
  empujar una etiqueta (2026-09-07).
- ~~El arranque de cinco segundos en terminales que no contestan~~ → desapareció al retirar la
  interfaz de terminal, que era quien lo causaba (2026-09-07).
