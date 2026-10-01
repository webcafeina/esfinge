# Lo siguiente

Última actualización: **2026-10-01**

Por prioridad. Lo cerrado se tacha y se queda, con la fecha: saber qué se descartó vale tanto como
saber qué se hizo.

## Alta

- **Publicar la 2.36.0 cuando Chrome apruebe la 2.35.0.** Lleva dos cosas escritas y en verde: el campo
  `usada` de las llaves de acceso y **las redes wifi con su código QR** ([ADR 0049](adr/0049-las-redes-wifi.md)).
  Se espera a la aprobación por una razón medida, no por costumbre: publicar mientras Chrome revisa deja ese
  paquete **fuera de la tienda** hasta la publicación siguiente, que es lo que le pasó a la 2.34.0. Lo decidió
  el cliente el 2026-10-01 con las dos opciones delante.
- **Abrir la aplicación en máquinas Windows y GNOME de verdad**, y mirar ahí dos cosas: la estructura
  de cada sistema y el icono de los `.esf` en el explorador de ficheros. En Windows el icono lo pone
  el instalador, en Linux el `.deb`; los dos están comprobados por dentro pero no puestos.
  La estructura de macOS está juzgada en un Mac; las otras dos se escribieron a partir de las
  convenciones de cada sistema y no las ha visto nadie corriendo (ADR 0020).

~~**Probar la extensión en Firefox**~~ → **hecha el 2026-09-28**: «pruebas de Firefox perfectas», con la
aplicación cerrada y la cuenta desde el panel. Con eso se cerró la E3 y el plan de cuentas entero.

~~**Desbloquear con el sistema**~~ → **hecho de la 2.27.0 a la 2.27.6** ([ADR 0044](adr/0044-desbloquear-con-el-sistema.md)),
Touch ID en macOS y Windows Hello escrito a ciegas, con lo que se decidió: es un cerrojo y no una llave, y se
dice en la pantalla donde se activa.

~~**Passkeys**~~ → **hechas enteras**, las cuatro entregas, de la 2.32.0 a la 2.34.0
([ADR 0048](adr/0048-las-llaves-de-acceso.md)). El cliente crea y usa llaves de GitHub en Chrome y en
Firefox, y la llave sirve desde su otro Mac.

~~**Poder rotar la pimienta del servidor**~~ → **hecha y rotada de verdad en pruebas el 2026-09-29**
([ADR 0046](adr/0046-rotar-la-pimienta.md)). Ya no hay nada vencido en la deuda.

~~**Vivir con la bóveda unos días**~~ → **superada por los hechos**: era la puerta de decisión para las fases
2 a 4, y esas fases están hechas y publicadas. El uso diario que pedía ha ocurrido por el camino.
- **Abrir la aplicación en máquinas Windows y GNOME de verdad**, y mirar ahí dos cosas: la estructura
  de cada sistema y el icono de los `.esf` en el explorador de ficheros. En Windows el icono lo pone
  el instalador, en Linux el `.deb`; los dos están comprobados por dentro pero no puestos.
  La estructura de macOS está juzgada en un Mac; las otras dos se escribieron a partir de las
  convenciones de cada sistema y no las ha visto nadie corriendo (ADR 0020).

## Media

- **Nada por ahora.**

## Baja

- **Traducción al inglés.** Hoy está todo en español, a propósito. Solo hace falta si el programa
  sale de la casa.
- **Empaquetar para Homebrew** (`brew install --cask esfinge`), que exige una URL estable y firma.
- **Compilar la aplicación para ARM64 en Windows y en Linux.** Hoy la ventana sale solo `amd64` en esos dos
  —`darwin/universal` en macOS, así que el Mac ya va nativo—, mientras que **la línea de comandos sí cruza a
  los seis objetivos**. Lo pidió el cliente el 2026-09-30 pensando en máquinas virtuales: en un M1 las VM
  nativas son ARM64, y ahí **el `.deb` no instala** —declara `arch: amd64` y `dpkg` lo rechaza— aunque el
  instalador de Windows x64 debería funcionar emulado, que Windows 11 on ARM lo hace. **No es urgente**: lo
  eligió así, y para lo que queda por ver —el marco de Fluent, la cabecera de GNOME, el icono de los `.esf`—
  la emulación no falsea el aspecto. Lo que costaría: **Linux ARM64** necesita un runner ARM, porque la app
  usa cgo por WebKit y no cruza desde x86, más un `.deb` con `arch: arm64`; **Windows ARM64** hay que
  **probarlo en el flujo de «Compilar» primero**, porque si Wails cruza a arm64 desde x64 no lo sé y es
  justo la clase de cosa que aquí solo se descubre compilando de verdad.

## Cerrado

- ~~La papelera no se puede vaciar~~ → hecha en la 2.16.0, y de paso borrar deja de ser
  irreversible: lo borrado se guarda entero, se puede restaurar, se vacía a mano y se va solo a los
  treinta días ([ADR 0026](adr/0026-la-papelera.md)). Se eligió entre dos opciones con las dos
  consecuencias delante; la descartada era «borrar es borrar», más segura y más pequeña, y perdió
  porque en un gestor de contraseñas perder una por un clic es peor que conservar treinta días una
  que se quiso tirar (2026-09-10).
- ~~Los códigos de un solo uso (TOTP)~~ → hechos en la 2.15.0, en la ventana y en la línea de
  comandos, con `internal/codigos` y sin dependencias nuevas. Con ellos se va la última cosa que
  obligaba a tener Dashlane abierto, y con ellos entra también la consecuencia incómoda: el segundo
  factor pasa a vivir al lado de la contraseña, dicho tal cual en `docs/seguridad.md` y en la
  [ADR 0025](adr/0025-los-codigos-de-un-solo-uso.md) (2026-09-10).
- ~~Probar un código contra Dashlane~~ → **da el mismo**, con los dos programas abiertos uno al lado
  del otro. Era lo único que ninguna prueba de aquí podía contestar, porque los vectores del RFC
  dicen que el algoritmo está bien y no que la semilla importada sea la buena. Con eso los segundos
  factores están mudados (2026-09-10).
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
