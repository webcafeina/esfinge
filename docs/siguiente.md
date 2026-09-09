# Lo siguiente

Última actualización: **2026-09-09**

Por prioridad. Lo cerrado se tacha y se queda, con la fecha: saber qué se descartó vale tanto como
saber qué se hizo.

## Alta

- **Vivir con la bóveda una semana.** Los datos ya están dentro —la exportación de Dashlane entra
  entera— y la clave de recuperación ya se ha usado de verdad, así que lo que queda no es una
  comprobación sino uso. Es la puerta de decisión del plan: si no se abre Esfinge para buscar una
  contraseña, las fases 2 a 4 —autorrelleno, cuentas, compartir— no se empiezan, porque su coste solo
  se justifica si la primera funcionó (ADR 0023).
- **Abrir la aplicación en máquinas Windows y GNOME de verdad**, y mirar ahí dos cosas: la estructura
  de cada sistema y el icono de los `.esf` en el explorador de ficheros. En Windows el icono lo pone
  el instalador, en Linux el `.deb`; los dos están comprobados por dentro pero no puestos.
  La estructura de macOS está juzgada en un Mac; las otras dos se escribieron a partir de las
  convenciones de cada sistema y no las ha visto nadie corriendo (ADR 0020).

## Media

- **Códigos de un solo uso (TOTP) de verdad.** La bóveda ya guarda la semilla y la trae al importar,
  pero **nadie calcula el código de seis cifras todavía**: se enseña la semilla y ya. Mientras eso no
  esté, los segundos factores se quedan en Dashlane, y con ellos la mitad de la razón para no
  dejarlo. Es un algoritmo estándar y sin dependencias raras. **Es lo primero que haría** si al vivir
  con la bóveda resulta que eso es lo que frena.
- **La papelera no se puede vaciar desde la ventana.** Borrar es borrado suave —hace falta para
  sincronizar después— así que una entrada borrada sigue en el fichero con su contraseña dentro. Hoy
  la única forma de quitarla de verdad es no tenerla.

## Baja

- **Traducción al inglés.** Hoy está todo en español, a propósito. Solo hace falta si el programa
  sale de la casa.
- **Empaquetar para Homebrew** (`brew install --cask esfinge`), que exige una URL estable y firma.

## Cerrado

- ~~Ver que la banda de versión nueva sale sola~~ → **sale**, con la ventana abierta y sin tocar nada
  (2026-09-09). Es la prueba buena del reloj de la 2.10.4: hasta entonces solo se comprobaba al
  arrancar y esa banda no había aparecido nunca, con la portada prometiendo «una vez al día» desde la
  2.1.0.
- ~~Que el humano vea si salta el aviso de Gatekeeper al actualizarse desde dentro~~ → **no salta**.
  La cuarentena la pone quien descarga, y ahí descarga Go y no un navegador; el guion además la quita
  explícitamente. El aviso es cosa solo de la primera instalación, y el LÉEME del DMG lo dice ahora
  (2026-09-08).
- ~~Que el Finder enseñe el icono de los `.esf`~~ → el icono estaba, pero **era el de la aplicación**:
  `build/esf.png` era una copia byte a byte de `appicon.png`. Ahora hay un documento de verdad,
  `build/documento.svg`, y el Finder lo enseña sin forzar la caché. De verlo puesto salió la 2.10.2,
  que lo centró en la hoja (2026-09-08).
- ~~Una entrada en `compilar.yml` para pedir compilación con inspector~~ → hecha: el disparador
  manual acepta `inspector`, que compila con `wails build -devtools`
  (`gh workflow run compilar.yml -f inspector=true`). El artefacto sale como `-CON-INSPECTOR`, dura
  siete días y la versión lleva sufijo, que además calla la comprobación de actualizaciones. Probada
  disparándola de verdad, no solo leyendo el YAML (2026-09-08).
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
  el marco del sistema en los dos (2026-09-07).

- ~~Ventana con vibrancy de verdad~~ → hecho en la 2.5.0, en la barra y el pie (2026-09-07).
- ~~Recordar la última carpeta de los diálogos~~ → hecho en la 2.5.0, una para abrir y otra para
  guardar (2026-09-07).
- ~~Un modo para tandas grandes~~ → en paralelo con tope desde la 2.5.0: veinte ficheros de 4,42 s a
  1,29 s (2026-09-07).

- ~~Copiar y pegar con los menús construidos a mano~~ → comprobado en el Mac: los atajos de ⌘ van
  bien (2026-09-07).
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
