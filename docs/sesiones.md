# Sesiones

Bitácora. Una entrada por sesión, la más reciente arriba. Sirve para retomar exactamente donde se
dejó aunque se pierda la conversación.

Plantilla al final.

## 2026-09-08 · Vidrio, carpetas recordadas y tandas en paralelo

- **2.5.0**, tres cosas que llevaban días anotadas en `siguiente.md` y no dependían de nadie.
- **El vidrio del sistema** en la barra y el pie, con la zona de trabajo opaca (ADR 0017). Lo que no
  era evidente: pedirlo hace que el fondo lo tenga que pintar el CSS, y donde no hay vidrio —Linux,
  o un Windows sin Mica— un `body` transparente no enseña el escritorio, enseña un agujero. De ahí
  el atributo `data-vidrio` y una prueba que vigila que sin él nada cambia.
- El tinte se ajustó **mirando**: se simuló la ventana con un degradado saturado detrás y a 0,72 el
  texto apagado del pie se lavaba. Subido a 0,82. La simulación no tiene el desenfoque del sistema,
  así que es un caso peor que el real.
- **Los diálogos recuerdan su carpeta**, una para abrir y otra para guardar. Con una trampa que
  habría dejado el diálogo sin abrir: Wails **falla la llamada entera** si el directorio por defecto
  ya no existe, así que se comprueba antes de proponerlo.
- **Las tandas, en paralelo** con tope de la mitad de los núcleos, máximo cuatro (ADR 0018). Medido
  antes y después en la misma máquina: **veinte ficheros de 4,42 s a 1,29 s**. El tope no es
  prudencia vaga: cada derivación ya usa cuatro hilos y 64 MiB por dentro.
- Verificado: `make comprobar`, `make contraste`, **`go test -race` sobre todo**, que aquí importa
  porque es la primera vez que se cifra desde varias gorrutinas, y 26 pruebas de interfaz en los dos
  temas.
- **2.5.1**, al probarlo: el vidrio **no se notaba**. La cañería estaba bien —Wails mete un
  `NSVisualEffectView` detrás de la ventana—; el error fue calibrar el tinte contra una simulación
  sin desenfoque. macOS no enseña el escritorio, enseña un material ya suavizado, así que dejar pasar
  el 18 % de eso es no tener efecto. Bajado a 0,55.
- Y Ajustes pasa a decir si la ventana usa el vidrio del sistema: la primera vez no había forma de
  distinguir «no llega la señal» de «el tinte tapa demasiado», y eso costó una versión.
- **2.5.2, y aquí estaba el fallo de verdad**: seguía sin verse, «solo un gris». Ese gris plano es la
  pinta exacta de un `NSVisualEffectView` cuya ventana sigue siendo opaca. Wails crea el efecto con
  mezcla «BehindWindow» pero **nunca pone la ventana como no opaca**, y una `NSWindow` opaca compone
  como opaca aunque su color tenga alfa cero. Se remata desde `vidrio_darwin.go` con cgo, junto con
  el `underPageBackgroundColor` del `WKWebView`, que desde macOS 12 tapa igual.
- Lección: bajar el tinte dos veces sin entender el síntoma era el camino equivocado. El «gris plano»
  no era un tinte de más, era el material sin nada que mezclar.

## 2026-09-08 · El icono del disco montado

- **2.7.1**: el volumen del DMG usaba el icono de la aplicación, así que el disco montado y lo que
  hay dentro se veían **exactamente igual** —en la barra lateral del Finder no había forma de
  distinguirlos—. Ahora es un disco con la marca encima.
- El `.icns` se arma en el trabajo de macOS con `iconutil`, que solo existe allí; aquí se rasterizan
  los PNG del `.iconset` con `make icono`, y el guion vuelve al icono de antes si algo falla.
- Los nombres de los ficheros del `.iconset` los impone `iconutil`: si falta uno o se llama distinto,
  se niega a construir el fichero.
- **2.7.2, con dos correcciones del humano mirando su Mac**: el disco estaba **apaisado** y los del
  sistema van **de pie**; y la marca repintada en dorado sobre gris quedaba desvaída, así que va el
  icono de la aplicación **tal cual**, que lleva su propio fondo oscuro y se sostiene contra la cara
  clara del disco. Buscar la forma en la web no sirvió —los resultados hablan de iconos de
  aplicación, no de volúmenes—: lo resolvió quien tenía el sistema delante.
- **2.7.3, y aquí estaba lo de fondo**: con capturas de discos de verdad —una unidad del sistema y
  los volúmenes de FUSE y de Affinity— se vio que **un disco no se dibuja de frente**. Va en
  perspectiva, mirado desde arriba: carcasa que se estrecha hacia el fondo, placa en color con el
  logo grande arriba, banda con su piloto abajo. Y casi cuadrado, no alargado.
- Tres intentos: apaisado, vertical y plano, y por fin en perspectiva. Los dos primeros salieron de
  imaginar cómo era; el tercero, de mirar tres discos de verdad. La diferencia entre uno y otro
  método está en el resultado.

## 2026-09-08 · Windows y GNOME, cada uno a lo suyo

- **2.7.0**: los tres sistemas tienen ya su estructura. La de fondo es la misma —navegación a un
  lado, contenido al otro, que es como se organizan los tres escritorios modernos— y lo que cambia
  son las formas, las densidades y quién dibuja el marco (ADR 0020).
- **El marco lo dibuja el sistema** en Windows y Linux; solo macOS se queda sin barra de título.
  Hacerlo a mano en los tres se parecería más, pero cerrar, maximizar y redimensionar pasarían a ser
  código nuestro y **aquí no hay dónde probarlo**.
- Windows lleva el panel de Fluent, con **la barra de acento a la izquierda de la fila activa**, que
  es lo que distingue un `NavigationView` de una barra lateral cualquiera. Linux va a lo GNOME:
  título centrado en negrita con su línea debajo.
- Aparece `App.Plataforma()` y un `data-sistema` en la raíz, igual que el `data-vidrio`. Con eso, el
  CSS de cada sistema cuelga de un sitio y no se mezcla.
- **Corregido un error que había escrito en la 0017**: dije que Wails no ofrecía translucidez en
  Linux, y sí la ofrece. No se usa por otra razón —sin desenfoque del compositor, la transparencia de
  GTK enseña el escritorio a pelo— pero la ficha decía algo falso.
- Verificado: 28 pruebas en los dos temas, dos tandas seguidas —y ahora ejercitan la variante de
  GNOME, porque el servidor de desarrollo corre en Linux—, más las tres variantes miradas una a una.
  **Sin comprobar**: Windows y GNOME de verdad.

## 2026-09-08 · La estructura de una aplicación de macOS

- **2.6.0**: la ventana deja de ser una página web dentro de un marco. Barra lateral con la
  navegación, título en la barra de herramientas, formularios en tarjetas agrupadas y sin barra de
  título propia —los semáforos caen sobre la barra lateral, como en Finder— (ADR 0019).
- **Las medidas no se inventaron.** El humano dejó cuatro capturas de macOS 26 en `referencias/`, y
  de ahí salen: los semáforos miden 12 pt clavados, así que sirven de regla para sacar la escala de
  cualquier captura. Medido: barra lateral **224,6 pt**, paso entre filas **38,5 pt**, semáforos a
  **21 pt** del borde. Esa técnica vale para cualquier captura futura.
- Dos cosas se vieron al primer render y no antes: las filas de la barra lateral salían con una
  **línea separadora** que era la sombra del botón de acción heredada, y los **emoji de color**
  desentonaban —el sistema usa trazo monocromo—. Los iconos se dibujan ahora en SVG, que además
  esquiva lo de los SF Symbols, que no se pueden detectar desde CSS.
- En las pruebas apareció una colisión nueva: **«Cifrar» nombra a la vez la sección y el botón que
  cifra**, así que los selectores se acotan a `.lateral` o a `.contenido`.
- Verificado: 26 pruebas de interfaz actualizadas y en verde en los dos temas, dos tandas seguidas;
  `make comprobar` y `make contraste`. Capturas de la portada rehechas.
- **2.6.1**, con lo primero que dijo el Mac: el título se veía con fondo y el resaltado del ratón
  pisaba la fila activa.
  - Lo del fondo no era CSS sino macOS: `TitleBarHiddenInset` activa `UseToolbar` y el sistema dibuja
    **su propia banda** justo donde va nuestro título. Con `TitleBarHidden` desaparece.
  - Lo del hover era un **empate de especificidad**: `.lateral button` pesa lo mismo que la regla
    general de `button:hover`, que va más abajo en el fichero y por eso ganaba. Los selectores llevan
    ahora `nav` para desempatar, y tiene prueba: esos empates vuelven solos en cuanto alguien añade
    una regla al final.
  - De paso, la prueba del interruptor de Ajustes **se prepara su propio punto de partida**. Daba por
    hecho que arrancaba encendido, y el fichero de preferencias del servidor de desarrollo es de
    verdad: una tanda interrumpida lo dejaba apagado y a partir de ahí fallaba siempre.
- **2.6.2**: el fondo del título seguía ahí, y esta vez sí era CSS mío. Bajo el vidrio, **lo que no
  declara fondo se vuelve transparente y enseña el material**; la barra de herramientas se había
  quedado sin declararlo al reestructurar, así que se veía una banda encima del contenido. El fondo
  pasa a la columna entera.
- De paso, el reparto del vidrio se rehace por columnas: **el material va en la barra lateral**, que
  es donde lo pone macOS, y la zona de trabajo se queda opaca. Puede que además resuelva lo de que el
  vidrio no se apreciaba: la barra lateral es una superficie grande, y hasta ahora el material solo
  tenía dos franjas finas donde asomar.
- **Sin comprobar, y es lo que decide lo siguiente**: cómo queda en un Mac. De eso depende si Windows
  y Linux se hacen igual.

## 2026-09-08 · Lo que dijo el Mac

- **Copiar y pegar funcionan** con los atajos de ⌘. Era el trozo con más riesgo de todo lo escrito
  el día anterior: al construir los menús a mano (ADR 0015) se perdieron los selectores nativos y
  esas acciones pasan por código propio, con el pegar repartido entre Go —que lee el portapapeles— y
  la interfaz —que coloca el texto en el cursor—.
- Queda abierto lo del aviso de Gatekeeper al actualizarse desde dentro.

## 2026-09-07 · Cierre de la jornada

De la 2.0.3 a la **2.4.0** en una sesión, todas publicadas y todas con su ficha cuando la decisión lo
merecía. El orden en que salieron cuenta bien lo que pasó: primero la distribución, luego enterarse
de que hay versión nueva, luego traérsela, luego instalarla sin que nadie arrastre nada, y por el
camino dos cosas que se daban por cerradas y no lo estaban.

**Lo que se corrigió de decisiones anteriores**, que es lo que más vale conservar:

- La **0014** daba por hecho que reemplazarse a sí misma exige firmar con Apple. No es cierto: la
  firma evita el aviso de Gatekeeper, no habilita el reemplazo. Lo arregla la **0016**.
- La **2.0.1** dio por cerrado el doble clic en un `.esf` porque enganchó `Mac.OnFileOpen`. Estaba a
  medias: nadie escuchaba el evento. Cerrado en la **2.3.1**.
- El `SHA256SUMS` publicado tuvo **dos fallos seguidos**: primero solo cubría los binarios de la
  línea de comandos, y luego se incluía a sí mismo con un resumen calculado a medio escribir.

**Al volver, lo primero es leer las preguntas abiertas de `estado.md`**: hay dos comprobaciones que
solo puede hacer el humano en su Mac —copiar y pegar con los menús nuevos, y si el aviso de
Gatekeeper desaparece al actualizarse desde dentro— y lo que venga después depende de lo que salga
de ahí.

## 2026-09-07 · Un .esf se abre en la pantalla que le toca

- **2.4.0**: abrir un `.esf` llevaba siempre a la pantalla de ficheros, y eso es un lío cuando lo que
  lleva dentro es un texto: ahí lo que se quiere ver es el secreto, no otro fichero al lado.
- Un `.esf` puede ser dos cosas y la extensión no lo dice: un fichero cifrado, o la línea `ESF1.…`
  que sale de cifrar un texto y que alguien guardó. **Los dos empiezan por la misma magia**; lo que
  los separa es el byte siguiente —la versión en el binario, el punto en el de texto—. De ahí sale
  `cripto.FormaDe`, que es donde tiene que vivir porque es el formato quien lo sabe.
- `AperturaDe` mira dentro y decide la pantalla. Con varios ficheros van todos como ficheros: en la
  pantalla de texto no cabe más que uno y elegir cuál sería adivinar.
- Verificado: seis casos de la detección de formato, cinco de la apertura, y dos pruebas de interfaz
  —una fabrica un `.esf` de texto de verdad, cifrando y guardando, y comprueba que al abrirlo sale
  en la pantalla de texto y se descifra desde ahí—. 24 pruebas de interfaz en los dos temas, dos
  tandas seguidas.

## 2026-09-07 · El doble clic en un .esf, arreglado de verdad

- **2.3.1**: la 2.0.1 dio por cerrado el doble clic en un `.esf` porque enganchó `Mac.OnFileOpen`.
  Estaba a medias: el evento se emitía y **la interfaz no lo escuchaba**. Con la ventana ya abierta
  el doble clic no hacía nada, y al abrir la aplicación con un fichero había una carrera —si la
  ventana preguntaba antes de que macOS entregara el fichero, se perdía—.
- Son dos momentos distintos y hay que tratarlos distinto: lo que llega **antes** de que haya alguien
  escuchando se guarda, y lo que llega **después** se avisa. Y la interfaz **se suscribe antes de
  preguntar**, que al revés deja el hueco por el que se colaba el fallo.
- De paso: varios ficheros a la vez —macOS manda un evento por cada uno— y la pantalla de descifrar
  se rehace si llega otro con ella ya abierta.
- Verificado: cuatro pruebas de Go de los dos momentos, con `-race`, y una de interfaz que recorre el
  camino entero. Una prueba anterior dependía del orden —el servidor de desarrollo es uno solo y se
  acuerda de la novedad—, y se ha quitado esa dependencia; tres tandas seguidas en verde.
- **Sin comprobar**: si el Finder enseña el icono propio del documento. Anotado en la deuda.

## 2026-09-07 · Actualizarse de verdad, sin arrastrar nada

- Probada la 2.2.0 en el Mac: la descarga va, pero al instalar salía la ventana de arrastrar a
  Aplicaciones. Es decir, el trabajo se lo acababa haciendo el usuario, que es lo que se quería
  quitar.
- **La 0014 estaba mal en un punto y hay que decirlo**: daba por hecho que reemplazarse exige firmar
  con Apple. No es cierto. La firma evita el aviso de Gatekeeper; el reemplazo solo necesita permiso
  de escritura donde vive la aplicación y hacer el cambiazo desde fuera del proceso. Corregido en la
  0016.
- **2.3.0**: Esfinge se sustituye y se reinicia sola. Lo hace un guion que ella escribe, lanza fuera
  de su grupo de procesos —para que cerrar la ventana no se lo lleve por delante— y que espera a que
  el proceso muera antes de tocar nada. En macOS monta el DMG, saca el `.app` con `ditto` y lo
  cambia; en Windows lanza el NSIS en silencio; en Linux no, porque el `.deb` instala como root.
- Quién puede hacerlo no lo decide el sistema sino **si se puede escribir donde vive la aplicación**,
  y eso se comprueba escribiendo: los permisos de `Stat` no valen en macOS, que tiene listas de
  control de acceso.
- El botón dice ahora «Instalar y reiniciar» o «Abrir el instalador» según lo que se pueda hacer en
  esa máquina, con el modo viajando dentro de la novedad.
- Verificado: tests del reparto, compilación para los tres sistemas, y las 22 pruebas de interfaz.
  Una de ellas era inestable —la orden del menú se emitía antes de que el navegador enganchara el
  flujo de eventos— y se hizo robusta reintentando; se comprobó con tres tandas seguidas.
- **Sin verificar, y es lo que importa**: el cambiazo de verdad. Aquí no hay Mac ni Windows.

## 2026-09-07 · La barra de menús, en español

- Probada la 2.1.0 en el Mac: la comprobación funciona —con la última instalada no sale la banda y
  «Buscar ahora» dice que ya se está al día—, que es exactamente lo que tenía que pasar.
- **2.2.0**: la barra de menús del sistema salía en inglés. No se podía traducir: los roles de Wails
  llevan los rótulos escritos a fuego en su Objective-C y no son los que localiza macOS. Se construye
  entera (ADR 0015), igual en los tres sistemas.
- Eso obliga a hacer las acciones de edición por nuestra cuenta, porque sin roles no hay selectores
  nativos: el menú manda una orden y la ventana la ejecuta sobre el campo que tiene el foco. El pegar
  necesita las dos mitades —el portapapeles lo lee Go, el texto lo coloca la interfaz—, que es el
  camino de más riesgo de todo esto: si falla, falla pegar una contraseña.
- De paso, atajos a las cinco pantallas (⌘1 a ⌘5), que antes no había forma de alcanzar sin ratón.
- Verificado: prueba de interfaz en los dos temas que recorre el camino entero desde que Go manda la
  orden, y prueba de Go de que salen por el mismo canal de eventos que el progreso. **Sin verificar**:
  los menús dibujados de verdad, que no hay Mac ni Windows aquí.

## 2026-09-07 · Que la aplicación se entere de sus propias versiones

- **2.1.0**: Esfinge comprueba una vez al día si hay versión nueva, se descarga el instalador de su
  sistema comprobando el SHA256 mientras baja, y lo abre. No se reemplaza a sí misma: eso exige
  firmar con Apple, que está descartado desde la 0012. Ninguno de los tres sistemas pide desinstalar
  antes.
- Paquete nuevo `internal/actualizacion`, sin dependencias fuera de la biblioteca estándar y sin
  saber nada de Wails, para poder usarlo también desde la línea de comandos y probarlo entero contra
  un servidor de mentira.
- **Lo primero que hubo que arreglar no estaba en el plan**: el `SHA256SUMS` que se publicaba solo
  cubría los seis binarios de la línea de comandos. El DMG, el instalador y el `.deb` salían sin
  resumen, así que no había contra qué comparar una descarga. Ahora se rehace en el trabajo de
  publicar, sobre todos los adjuntos.
- Aparecen dos cosas que no existían: un fichero de preferencias —junto al historial, con sus mismos
  permisos— y una quinta pestaña, **Ajustes**, que es donde se cuenta con todas las letras qué se
  envía y dónde se apaga. Eso es la contrapartida de haber elegido comprobación automática: la
  portada decía que nada salía del ordenador y ha habido que matizarlo, en el README y en
  `seguridad.md`.
- En la línea de comandos el aviso va por la salida de error, nunca por la estándar, **solo si esa
  salida es un terminal** —en una tubería o un cron no se escribe ni se pregunta— y con un plazo de
  cortesía: si la red no contesta, el comando no espera.
- Una trampa del diseño de Wails que conviene recordar: **todo método exportado de `*App` queda
  expuesto a la interfaz**, así que `ApuntarAAPI` tuvo que ser función y no método, para que la
  ventana no pueda apuntar la comprobación a donde quiera.
- Verificado: 21 tests de Go nuevos contra un `httptest.Server` —incluido que con el interruptor
  apagado no se hace **ni una** petición, que se cuenta en vez de suponerse—, `make comprobar` en
  verde con y sin `-tags dev`, `-race` limpio, y 10 pruebas de interfaz en los dos temas contra una
  API de mentira.
- Publicada la 2.1.0, se vio otro fallo del `SHA256SUMS`: se incluía **a sí mismo**, con el resumen
  del fichero a medio escribir —la redirección lo crea antes de que corra `find`—. Un
  `sha256sum -c` daba FAILED justo en el fichero que sirve para confiar. Corregido en el flujo, que
  además ahora lo verifica, y reemplazado en la publicación ya hecha: las demás líneas eran buenas,
  comprobado descargando el `.deb`.
- Queda abierto lo que no se puede ver desde aquí: una actualización de verdad, y si al descargar el
  DMG desde Go la copia instalada se libra del aviso de Gatekeeper.

## 2026-09-07 · Infraestructura: documentos, repositorio e instaladores

- Se montó este sistema de documentos vivos siguiendo la convención de Tempero, con trece decisiones
  escritas hacia atrás a partir de lo que ya estaba en `CLAUDE.md` y en los mensajes de los commits.
- `gitleaks` sobre el historial completo: 11 commits, sin filtraciones. Era el requisito para hacer
  público el repositorio.
- Portada del repositorio con capturas, tabla de descargas y distintivos; licencia propietaria que
  permite leer y auditar el código pero no reutilizarlo. Lo técnico que estaba en el README se mudó
  a `docs/`.
- Instaladores de los tres sistemas: DMG con fondo de marca y LÉEME dentro
  (`empaquetado/macos/armar-dmg.sh`, la misma receta para `make dmg` y para el flujo), instalador
  NSIS para Windows, y `.deb` con `nfpm` que lleva las dos caras —ventana y línea de comandos—, su
  lanzador y la asociación de los `.esf`. Los instaladores de doble clic de la época del ZIP se
  retiraron: los sustituye el DMG.
- Flujo `publicar.yml`: se dispara con una etiqueta `v*`, compila en los tres sistemas y cuelga todo
  de la publicación de GitHub.
- Verificado: `make comprobar` en verde, `gitleaks` sin filtraciones, los tres YAML válidos.
- **2.0.1** publicada: el flujo entero de punta a punta, con sus adjuntos. Salió en verde con un
  fallo escondido: Chocolatey instala NSIS pero no lo deja en el `PATH`, y `wails -nsis` avisa de que
  no encuentra `makensis` y **sale con código cero**. El trabajo pasó y la publicación se quedó sin
  instalador de Windows. **2.0.2** lo arregla: se añade NSIS al `PATH` y se comprueba que el fichero
  existe, que es la misma lección que ya estaba escrita para `create-dmg`.
- Del `.dmg` publicado, leído desde Linux con 7-Zip: `Esfinge.app` con el binario universal —x86_64
  y arm64—, el enlace a `/Applications`, el `LÉEME.txt` idéntico al del repositorio, el fondo en
  `.background/fondo-dmg.tiff` con las dos resoluciones, el icono de volumen y un `.DS_Store` de 10
  KB, que es la señal de que `create-dmg` sí habló con el Finder y guardó la colocación.
- Del `.deb`: los dos binarios con su bit de ejecución, el lanzador, la asociación de los `.esf`, el
  icono, el copyright y las dependencias del webview.
- El instalador de Windows costó tres intentos y cada uno enseñó algo distinto: que Chocolatey no
  deja NSIS en el `PATH`, que `GITHUB_PATH` quiere la ruta como la entiende Windows y no la de Git
  bash —con la de bash el fichero está y la comprobación pasa, pero wails sigue sin encontrarlo—, y
  que Git bash convierte los argumentos que empiezan por «/», así que `makensis /VERSION` hay que
  ejecutarlo en PowerShell. Comprobado del `.exe` publicado: es un NSIS-3 Unicode y lleva dentro la
  aplicación, el `esf.ico`, el instalador de WebView2 por si el sistema no lo trae, y en su cabecera
  las claves que registran el `.esf` —`Software\Classes`, `DefaultIcon`, `shell\open\command`—.
- **2.0.3**: montado el DMG en el Mac, la imagen estaba bien salvo el `LÉEME.txt`, cuyo nombre caía
  encima del aviso de Gatekeeper del fondo. El Finder centra cada icono en la posición que se le da
  y **escribe el nombre debajo**, unos 64 px más abajo con iconos de 96: ese espacio no se puede
  usar para dibujar. La ventana pasa a 660×470 y el LÉEME baja a su propia banda, con el aviso a su
  derecha en vez de debajo.
- De ahí sale `make ventana-dmg`, que dibuja la ventana con los iconos donde los pondrá el Finder
  **leyendo las posiciones del propio `armar-dmg.sh`**, para que no puedan separarse. Es la forma de
  ver esto sin un Mac, que era justo lo que faltaba.

## 2026-09-07 · La aplicación de escritorio, y arreglarla con lo que dijo el Mac

- **2.0.0**: se retiraron los menús de terminal (1.830 líneas y 1.492 de test) y se construyó la
  aplicación con Wails. Repositorio creado en GitHub y compilación de los tres sistemas.
- Cuatro intentos hasta que compiló, y las cuatro cosas solo se ven compilando de verdad: dónde
  tiene que vivir el `main.go`, que las asociaciones de fichero van dentro de `info` en `wails.json`,
  qué versión de webkit busca Wails en Linux, y que los artefactos de GitHub pierden el bit de
  ejecución.
- **2.0.1**, con lo que salió de abrirla en un Mac: el arrastrar y soltar no hacía nada —el modo
  «zona» exige declarar una propiedad CSS que no se declaraba—, el doble clic en un `.esf` abría la
  ventana vacía —Wails sí lo entrega, por `Mac.OnFileOpen`—, y el aspecto «no parecía nativo», que
  obligó a rehacerlo con barra translúcida, radios mayores y bloques agrupados.
- El generador pasó a medir en caracteres además de en bits, ligados entre sí.
- Verificado: `make comprobar`, 16 pruebas de interfaz en los dos temas, y los tres sistemas
  compilando en verde.

## 2026-09-04 · De cero a herramienta de terminal

- Se construyó Esfinge entero: núcleo criptográfico, línea de comandos e interfaz de menús.
- Se instaló Go en `~/.local/go`, sin tocar el sistema.
- De la 1.0 a la 1.5.0 en una sesión, con lo que fue saliendo al probarlo: el truncado que culpaba a
  la clave, el ratón que no iba en la pantalla de cifrar —la vista se salía del alto y las
  coordenadas dejaban de cuadrar—, el guardado que se pisaba a sí mismo, y el arranque de cinco
  segundos por el `init()` de Bubble Tea.
- Instaladores de doble clic para macOS y Linux.

---

## Plantilla

```
## AAAA-MM-DD · <título>

- Qué se hizo o se decidió.
- Qué se verificó, y con qué.
- Qué queda abierto (→ mover a siguiente.md o deuda.md si procede).
```
