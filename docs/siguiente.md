# Lo siguiente

Última actualización: **2026-09-07**

Por prioridad. Lo cerrado se tacha y se queda, con la fecha: saber qué se descartó vale tanto como
saber qué se hizo.

## Alta

- **Que el humano vea si salta el aviso de Gatekeeper** al actualizarse desde dentro. Si no salta,
  hay que quitarlo del LÉEME del DMG para el caso de la actualización.
- **Que el Finder enseñe el icono de los `.esf`.** La asociación está declarada y la aplicación
  abre con el fichero, pero falta ver si el icono del documento sale.
- **Ver la estructura en máquinas Windows y GNOME de verdad.** La de macOS está juzgada en un Mac;
  las otras dos se escribieron a partir de las convenciones de cada sistema y no las ha visto nadie
  corriendo (ADR 0020).

## Media

- **Una entrada en `compilar.yml` para pedir compilación con inspector.** Wails admite
  `wails build -devtools`, que añade la etiqueta `devtools` y deja abrir el Web Inspector en la
  aplicación empaquetada; en el binario que se publica no va, y así debe seguir.

  **Por qué merece existir:** el vidrio de macOS estuvo roto tres versiones —2.5.1, 2.9.0 y 2.9.1— y
  las tres salieron con un cambio que aquí no se podía probar. Dos no hicieron nada visible y la
  tercera **cerró la aplicación al arrancar**, con el cliente sin herramienta hasta publicar la
  2.9.2. Cada corazonada costaba una versión publicada. Con el inspector, cada corazonada cuesta una
  línea en una consola.

  Al final el vidrio se resolvió leyendo el código de Wails y no hizo falta, así que esto **no es
  urgente**: es la red para el siguiente problema de macOS que no se reproduzca aquí. Sería una
  entrada `workflow_dispatch` de tipo booleano que se pase al `wails build` del trabajo de macOS, y
  conviene que el artefacto salga marcado para que no se confunda nunca con uno de publicación.

## Baja

- **Traducción al inglés.** Hoy está todo en español, a propósito. Solo hace falta si el programa
  sale de la casa.
- **Empaquetar para Homebrew** (`brew install --cask esfinge`), que exige una URL estable y firma.

## Cerrado

- ~~El vidrio de macOS, que no se veía~~ → resuelto en la 2.10.0, y no calibrando sino leyendo el
  código de Wails: nunca le pone material a su `NSVisualEffectView`, así que se quedaba con uno
  obsoleto que se dibuja plano. Comprobado en el Mac, y comparado con el Finder al lado se ve igual
  (2026-09-07).
- ~~Reemplazo automático del binario, sin pasar por el instalador~~ → **estaba mal descartado.** Se
  dio por hecho que exigía firmar con Apple, y no tiene nada que ver: firmar evita el aviso de
  Gatekeeper, no habilita el reemplazo. Hecho en macOS y Windows con un guion que espera a que el
  proceso muera; en Linux no, porque el `.deb` instala como root (ADR 0016, que corrige la 0014)
  (2026-09-07).
- ~~Barra de menús propia~~ → hecha entera y en español en los tres sistemas, porque los roles de
  Wails traen los rótulos en inglés escritos a fuego (ADR 0015). Con atajos ⌘1…⌘5 (2026-09-07).
- ~~Montar el DMG en un Mac~~ → comprobado: se monta y se arrastra sin más. De ahí salió además lo
  del nombre del icono pisando el fondo, que ahora se dibuja aquí con `make ventana-dmg`
  (2026-09-07).
- ~~Windows y Linux, a su estructura~~ → hecho en la 2.7.0: panel de Fluent y cabecera de GNOME, con
  el marco del sistema en los dos (2026-09-08).

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
