# Estado

Última actualización: **2026-09-14**

## Dónde estamos

Esfinge es una **aplicación de escritorio** con ventana propia, más una línea de comandos que
comparte núcleo y formato. Va por la **2.21.2**. Funciona de punta a punta: cifra y descifra textos y
ficheros, genera contraseñas, **guarda contraseñas en una bóveda cifrada**, lleva un historial de qué
y cuándo, y se compila sola para macOS, Windows y Linux en GitHub Actions.

Desde la 2.12.0 Esfinge **deja de ser un cifrador sin estado**. La bóveda es la fase 1 de sustituir a
Dashlane, decidida con el cliente y con las fases 2 a 4 —autorrelleno, cuentas y compartir— sin
empezar a propósito: se hace la primera, se usa a diario, y solo entonces se decide si las otras
valen su coste.

La 1.x fue una herramienta de terminal con menús. Esos menús se retiraron: su público —el cliente—
tiene ahora una ventana. La línea de comandos se quedó, que es la que se mete en tuberías y scripts.

## Completado

- **Núcleo criptográfico** (`internal/cripto`): XChaCha20-Poly1305 con Argon2id, formato `ESF1`
  versionado, ficheros por segmentos con marca de final. Con tests que cubren la ida y vuelta, cada
  byte alterado, el truncado y la reordenación de segmentos.
- **Aplicación con ventana** (Wails, React 19 + TypeScript): cifrar y descifrar textos o tandas de
  ficheros, arrastrar y soltar, diálogos del sistema, historial e generador de contraseñas.
- **Actualizaciones dentro de la aplicación**: aviso, descarga comprobada con SHA256 y, en macOS y
  Windows, **reemplazo y reinicio sin que nadie arrastre nada** (ADR 0016). En Linux se le pasa al
  gestor de paquetes, que instala como root. Fue la única conexión que hacía el programa hasta la
  2.14.0; ahora son dos, con la de los iconos (ADR 0024). Las dos están dichas y se apagan en Ajustes,
  y `ESFINGE_SIN_RED` las apaga todas.
- **Menús del sistema en español** en los tres sistemas, construidos a mano porque los roles de Wails
  traen los rótulos en inglés escritos a fuego (ADR 0015). Con atajos ⌘1…⌘6 a las seis pantallas.
- **Apertura de un `.esf`** por doble clic o «Abrir con», que abre la pantalla que le toca según lo
  que lleve dentro: ficheros o texto.
- **Línea de comandos**: intacta desde la 1.5.0, con sus códigos de salida distintos por caso.
- **Color generado desde Go** (`internal/tema`), con el contraste de los dos temas medido en cada
  compilación.
- **Compilación automática** de los tres sistemas, más los seis binarios de la línea de comandos.
- **El vidrio del sistema en la barra lateral** —macOS siempre, Windows 11 con Mica—, con la zona de
  trabajo opaca (ADR 0017). Es donde el sistema pone la vibrancia. En Linux no lo hay, y ahí la
  ventana queda como estaba.
- **Los diálogos recuerdan su carpeta**, una para abrir y otra para guardar.
- **Las tandas se cifran en paralelo**, con tope: veinte ficheros pasaron de 4,42 s a 1,29 s en una
  máquina de cuatro núcleos (ADR 0018).
- **Estructura nativa en los tres sistemas**: barra lateral y contenido en todos, y luego lo de cada
  casa —macOS sin barra de título (ADR 0019), el panel de Fluent en Windows y la cabecera de GNOME en
  Linux (ADR 0020)—.
- **La marca, dentro de la ventana** (ADR 0021): el lockup de Esfinge en la barra lateral, la firma
  `▍ webcafeína` al fondo, la esfinge tenue en el historial vacío y la ficha de producto en Ajustes
  —que es lo que la ADR 0007 prometía y nunca se había construido—. Y el acento de la interfaz pasa a
  ser el oro del tocado en vez del azul del sistema.
- **El formato `ESF1`, congelado con vectores fijos** (ADR 0022): once contenedores grabados una vez
  y no vueltos a generar, más los de la 1.5.0 sellados con el código de entonces. Antes lo único que
  decía congelarlo era un test que se miraba al espejo, y un cambio coherente en los dos sentidos
  habría pasado en verde dejando de abrir lo ya emitido.
- **La bóveda** (ADR 0023): local, cifrada, con clave de recuperación, cuatro clases de entrada,
  historial de contraseñas anteriores, importación desde Dashlane, Bitwarden, 1Password, LastPass y
  Chrome —cada gestor exporta **varios ficheros** y cada uno se reconoce por su forma—, exportación
  en claro para poder salir, bloqueo por inactividad, borrado del portapapeles y borrado de la bóveda
  entera pidiendo la maestra. Con su sección en la ventana, sus dos plazos en Ajustes y
  `esfinge boveda listar|ver|codigo|exportar` en la línea de comandos. **No toca `internal/cripto`**,
  así que los `.esf` y las claves ya emitidos siguen valiendo.
- **Los códigos de un solo uso** (ADR 0025): la bóveda calcula el código de seis cifras a partir de la
  semilla que ya guardaba, con su cuenta atrás en la ventana y con `esfinge boveda codigo` para los
  scripts. `internal/codigos`, sin dependencias, con los vectores de RFC 4226 y RFC 6238 enteros.
  Con eso se va la última cosa que obligaba a tener Dashlane abierto —y entra la consecuencia
  incómoda: **el segundo factor pasa a vivir al lado de la contraseña**, dicho tal cual en
  `docs/seguridad.md`.
- **La papelera** (ADR 0026): borrar deja de ser irreversible. Lo borrado se guarda entero, se
  restaura o se tira del todo una a una, se vacía a mano y **se va solo a los treinta días**. El
  coste está dicho en `docs/seguridad.md`: durante esos días la contraseña borrada sigue dentro del
  fichero, igual que las del historial de contraseñas anteriores.
- **La fase 2, entrega 1** (ADR 0027): la extensión del navegador consulta la
  bóveda por un **canal local** —un socket en tu carpeta, no un puerto—, apagado de fábrica y
  encendido en Ajustes. Enseña las cuentas del sitio de la pestaña y copia lo que se le pida.
  Comprobado en un Mac de verdad, en Firefox y en Chrome.
- **La fase 2, entrega 2** (ADR 0028): **rellena los formularios**. Con una cuenta guardada del
  sitio, sola al cargar la página; con varias, desde el panel. Y **en la página no se dibuja nada**:
  el guion lee el formulario, escribe en él y calla. Dos propiedades de la entrega 1 se rompen a
  conciencia y hay que decirlas: **por el canal ya sale una contraseña de verdad** —para escribirla
  en un campo hay que tenerla, y eso no hereda el borrado del portapapeles— y **hay código nuestro en
  cada página `https`**. A cambio, el verbo nuevo lleva su propio freno, más estrecho que el de
  preguntar, y la detección de campos está escrita en negativo: ante la duda, no se rellena.
  **Comprobado en un Mac, en Chrome y en Firefox** (2.18.1): rellena solo al cargar y acierta
  el formulario en Brevo y en Cloudflare, y el botón del panel escribe tras borrar los campos.
- **La fase 2, entrega 3** (ADR 0032): **ofrece guardar y actualizar** lo que se envía en un
  formulario, con una tarjeta en la página. Publicada en la 2.21.0 y corregida hasta la **2.21.2** con lo que salió de usarla —Google con dos pasos y varias cuentas, y el cambio de contraseña de Brevo—; **comprobada en el Mac**.
- **Pruebas de la interfaz** con Playwright contra el Go de verdad, en tema claro y oscuro, en una
  máquina sin entorno gráfico. Son **84**.

## En curso

**La fase 2 va por la entrega 2.** La 1 —el canal y la extensión que consulta y copia— se comprobó
en un Mac en las dos familias de navegador. La 2 —rellenar— está escrita y probada aquí hasta donde
se puede: **falta verla en sitios de verdad**, que es lo único que ninguna prueba puede decir.

De escribirla salió, sin buscarla, una de la entrega 1: **el freno de preguntas no frenaba nada**. El
contador vivía en la conexión y la extensión abre una conexión por petición, así que el tope de
sesenta por minuto que impedía reconstruir la lista de sitios de la bóveda con un diccionario de
dominios no se alcanzaba jamás. La prueba estaba en verde porque le pasaba un contador hecho a mano a
sesenta llamadas seguidas, que es el caso que no ocurre. Es la misma familia que los iconos.

La 2.15.0 cerró los códigos de un solo uso —comprobados contra Dashlane el mismo
día— y la 2.16.0, la papelera. En medio salió, sin buscarla, una que llevaba desde la 2.12.0:
**borrar una nota segura o una tarjeta dejaba su contenido dentro del fichero**, porque la lista de
campos sensibles estaba escrita en dos sitios y solo uno estaba completo.

**Con la bóveda ya no queda nada de Dashlane por traer al escritorio.** Lo siguiente es la fase 2, y
su puerta sigue siendo de uso y no técnica.

La 2.12.0 salió con la bóveda y de usarla salieron cinco versiones seguidas de
correcciones —2.12.1 a 2.12.5—, todas de cosas que **solo aparecen usando la aplicación en un Mac**:
pegar con ⌘V, copiar de un campo de contraseña, el filtro del diálogo de abrir, el importador que
solo entendía una forma de fichero y una negrita que partía los avisos en columnas. Ninguna se
habría encontrado desde esta máquina. Los tres sistemas tienen ya su estructura. La de macOS está probada en un Mac; la de
GNOME se puede mirar aquí, porque el servidor de desarrollo corre en Linux; la de Windows no la ha
visto nadie.

## Comprobado en un Mac de verdad

- Que la comprobación de versiones funciona: con la última instalada no sale la banda, y «Buscar
  ahora» dice que ya se está al día.
- La descarga de una actualización, con su barra.
- El doble clic en un `.esf`, en los dos momentos —con Esfinge cerrada y con Esfinge abierta— y con
  las dos clases de contenedor: el que lleva un fichero abre la pantalla de ficheros y el que lleva
  un texto abre la de texto, con la línea puesta.
- **Copiar y pegar con los atajos de ⌘**, que era lo que más riesgo tenía: al construir los menús a
  mano se perdieron los selectores nativos y esas acciones pasan por código propio.
- **La estructura de macOS** de la 2.6.x: barra lateral, título sin banda y el resaltado del ratón
  respetando la fila activa.
- **El disco del DMG**, después de tres intentos y de que llegaran capturas de discos de verdad.
- **Usar la contraseña generada como clave**, y que **cambiar de sección ya no borra lo escrito**
  (2.9.0), que era la deuda que dejó abierta la versión anterior.
- **El vidrio, por fin, en la 2.10.0.** «Ahora sí se ve». Y comparado con la barra lateral del Finder
  al lado, **se ve igual**: el desenfoque que parecía excesivo es el que macOS 26 pone en todas las
  barras laterales del sistema. No hay nada que calibrar.
- **La actualización desde dentro, y que Gatekeeper no aparece.** Se ha usado de verdad, varias
  versiones seguidas: se descarga, se reemplaza y se reinicia sola sin preguntar nada. Confirma lo
  que sostenía la ADR 0016: la cuarentena la pone quien descarga, y aquí descarga Go. El aviso de
  programa no identificado es cosa **solo de la primera instalación**.
- **El icono del documento `.esf` en el Finder** (2.10.1): sale, y sin tener que forzar la caché de
  LaunchServices. De mirarlo puesto salió la corrección de la 2.10.2: la placa estaba centrada en 636
  y la hoja tiene su centro en 512, así que caía 124 px baja. Se había bajado a propósito, creyendo
  que macOS pone ahí el emblema de los documentos; no era cierto.
- **La bóveda con datos de verdad, en el Mac** (2026-09-09): una exportación de Dashlane entra
  entera —credenciales, notas seguras, tarjetas y documentos, que son cuatro ficheros— y **la clave
  de recuperación abre la bóveda**, que era lo único que ninguna prueba podía decir. Costó cuatro
  versiones: el filtro del diálogo de abrir en macOS, el importador que solo entendía una forma de
  fichero, las tarjetas que salían todas duplicadas y lo exportado que no volvía a entrar entero.
- **El código de un solo uso es el mismo que el de Dashlane** (2026-09-10, 2.15.0), con los dos
  programas abiertos uno al lado del otro. Es la comprobación que ninguna prueba de aquí podía hacer,
  y cierra las dos mitades a la vez: que **la semilla se importó bien** —que era lo que de verdad
  estaba en duda, porque los vectores del RFC solo dicen que el algoritmo está bien— y que el
  cálculo concuerda con el de un gestor que lleva años en producción.
- **El canal con el navegador, de punta a punta** (2026-09-10, 2.17.6): con Firefox y un sitio de
  verdad, la extensión reconoce el sitio, enseña la cuenta guardada y **copia la contraseña al
  portapapeles**. Comprobado además lo que hace que eso sea aceptable: **el portapapeles se borra
  solo** pasado el plazo de Ajustes, igual que copiando desde la ventana. Y el **código de un solo
  uso** también, contra un sitio con segundo factor de verdad. **Y en Chrome igual** (2.17.7), con el
  identificador fijado por la clave pública del manifiesto: las dos familias de navegador funcionan.
  Era lo único que no se podía ejercitar aquí —no hay navegador con el que probar
  `connectNative`— y por eso costó **seis versiones**: el puente, el socket, el emparejamiento de
  dominios y los manifiestos estaban bien desde el principio, y lo que faltaba era **una palabra en
  una lista**, el permiso `storage` del manifiesto de la extensión. Sin él esa API no da error: es
  `undefined`, y el fallo se veía como si el puente no contestara.
- **La banda de versión nueva sale sola** (2026-09-09), con la ventana abierta y sin tocar nada. Es
  la prueba buena de la 2.10.4: antes la comprobación ocurría **solo al arrancar** y esa banda no
  había aparecido nunca, aunque la portada llevara desde la 2.1.0 prometiendo «una vez al día».
- Y de preguntar si ese icono valía para los otros dos sistemas salió la 2.10.3: en **Windows** sí,
  del mismo PNG y sin tocar nada, pero en **Linux** no llegaba y seguía saliendo el de la aplicación.
  El `.deb` instala ahora `application-x-esfinge.png` en `hicolor/<tamaño>/mimetypes/`, verificado
  aquí con `dpkg -c`. Verlo puesto en un escritorio de verdad sigue pendiente.

## Siguiente acción concreta

**Lo decidió el cliente al cerrar la sesión del 10 de septiembre, y va en este orden:**

### ~~1. Las mejoras visuales de la extensión~~ — hechas el 2026-09-14, usadas en el Mac

Resuelto en la [ADR 0029](adr/0029-el-panel-de-la-extension-es-esfinge.md): **el panel es la misma
Esfinge en otro sitio**. Importa los tokens de la ventana tal cual —ni una escala ni una pareja de
color nueva, cada combinación es una de las que ya mide `contraste_test.go`—, la marca va solo en la
cabecera como el lockup de la barra lateral, y el oro solo en «Rellenar». Una forma de fila para todas
las cuentas, con el correo entero siempre que quepa; copiar pasa a iconos con nombre; los estados
tienen título, instrucción y, aparte, lo que dijo el navegador. La extensión estrena **iconos**, que no
tenía, y el protocolo dice **si una cuenta tiene código** para no enseñar el botón donde no lo hay.

Se miró en capturas de cada estado en los dos temas (`pnpm run capturas` en `navegador/`) y hay seis
pruebas del panel compilado. **Lo que falta es verlo en un navegador de verdad en el Mac**: allí la
letra es San Francisco y los anchos cambian, así que lo primero es si el correo sigue cabiendo; y si el
icono de 16 px se lee en la barra.

### ~~2. Rellenar el código de un solo uso~~ — hecho y comprobado en el Mac el 2026-09-14

Resuelto en la [ADR 0030](adr/0030-rellenar-el-codigo-de-un-solo-uso.md). Un verbo nuevo,
`rellenar-codigo`, con las mismas llaves que `rellenar` y **gastando de su mismo freno**. Se escribe en
el campo que el sitio declara, en **seis u ocho casillas de un carácter**, o en un campo con nombre de
segundo factor si no hay contraseña en la página; **nunca en uno que solo se llame «code»**. Solo con
una cuenta del sitio que tenga código, esperando al siguiente si al actual le quedan menos de tres
segundos. El botón «Rellenar» del panel hace formulario y código a la vez.

**Probado en Firefox en el Mac con la 2.19.0: todo funciona salvo el código en Cloudflare**, que no se
detectaba —seis casillas que declaran `one-time-code` y ninguna con `maxlength="1"`—. Arreglado en la
2.19.1: su formulario copiado de la consola está en las pruebas, y la escritura se ha comprobado contra
**su mismo componente**, `OTPField` de Base UI, en React. **Y con la 2.19.1 funciona en Cloudflare, en
Firefox y en Chrome**, comprobado por el cliente el mismo día.

### ~~3. La segunda pasada visual de la extensión~~ — hecha y comprobada en el Mac el 2026-09-14 (2.20.3)

El cliente la pidió después de usar la 2.19.1, **antes** de guardar desde la página, y la decidió por
preguntas con opciones ([ADR 0031](adr/0031-el-icono-con-estados-y-la-marca-en-el-campo.md) y la
ampliación de la [0029](adr/0029-el-panel-de-la-extension-es-esfinge.md)):

- **El icono de la barra es la silueta de Esfinge sin placa**, con contorno de piedra para verse en
  barras claras y oscuras, y **dice el estado de cada pestaña**: número de cuentas, ✓ si ha rellenado,
  candado si la bóveda está cerrada, «!» si algo falla, y la frase al pasar el ratón. Se pone al día
  al cambiar de pestaña y **cada minuto** (permiso `alarms`).
- **El campo rellenado lleva un filete** de oro y piedra hasta que se escribe en él, y en los rellenos
  automáticos **un aviso de tres segundos**, «Rellenado por Esfinge». Esto **matiza la ADR 0028**: en
  la página ya se dibuja algo, aunque no se pueda pulsar ni dure.
- **El panel**: cuadro con la inicial en cada cuenta —la misma función que la ventana—, icono de la web
  sin salir a internet, «● Abierta», «✓ Hecho» en la fila, esfinge tenue en los avisos, cuenta atrás
  del código, firma al pie y atajos de teclado.

Lo que falta es **verlo en el Mac**: si la silueta se lee en la barra, si el candado aparece solo, el
filete y el aviso en Brevo y Cloudflare, y si el correo cabe entero en el panel.

**Y probada en Firefox con la 2.20.0** (2026-09-14): el número y el ✓ del icono se ven bien. El cliente
pidió cuatro correcciones —el filete sin borde negro, el marco de la fila solo con teclado, el candado
del tamaño de una insignia y la cara de la esfinge del aviso en crema—, hechas en la 2.20.1. Falta verlas
en el Mac, y lo que quedaba de antes: si el candado aparece solo al cerrarse la bóveda y si el correo
cabe entero.

Y con la 2.20.1 el cliente pidió **el candado otra vez más grande y el aviso más grande**: en la 2.20.2
el candado es una placa naranja de dos tercios del icono con el candado blanco, y el aviso pasa a 14 px.

Con la 2.20.2 **todo se ve bien en Firefox y en Chrome**. Quedaban dos detalles, hechos en la 2.20.3: la
esfinge tenue de los avisos, entera y con su trazo de serie, y «Rellenar» con la misma letra en Chrome
que en Firefox. **Y con la 2.20.3 el cliente dice «Ahora está perfecto»**: la parte visual de la
extensión queda cerrada.

### ~~4. Guardar y actualizar desde la página~~ — comprobado en el Mac el 2026-09-14 (2.21.2)

Resuelto en la [ADR 0032](adr/0032-guardar-desde-la-pagina.md), decidido con el cliente por preguntas
con opciones. **Al enviar un formulario, una tarjeta arriba a la derecha ofrece guardar la cuenta** —o
actualizar la contraseña, eligiendo la cuenta si hay varias—, con el título del sitio editable, «Ahora no»
y «Nunca en este sitio», que se deshace en Ajustes. Con la bóveda cerrada, la tarjeta dice que la abras.

Dos cosas que cambian y hay que decir: **el navegador escribe en la bóveda** por primera vez —con su
propio freno de seis por minuto y la contraseña anterior al historial— y **la tarjeta es lo primero de
Esfinge que se pulsa en la página de otro**. La contraseña enviada espera **en la memoria del trabajador
de fondo** y no llega a la página siguiente; qué ofrecer lo decide Go; y si el formulario vuelve a salir,
se da por mala y no se ofrece.

Probado aquí por piezas y la tubería de Go entera. **Lo que falta es usarlo en el Mac, en Firefox y en
Chrome**: entrar con una cuenta nueva, entrar con otra contraseña, registrarse, cambiar la contraseña,
un inicio fallido, «Nunca en este sitio» y que la ventana se ponga al día sola.

**Publicada la 2.21.0 el 2026-09-14** (todas las compilaciones en verde). **La siguiente acción concreta
es recoger lo que diga el cliente**, que la prueba la tarde del 14 en su Mac, en Firefox y en Chrome,
con la lista de ocho pasos: cuenta nueva, otra contraseña con la anterior en el historial, registro,
cambio de contraseña, **inicio fallido sin tarjeta**, «Nunca en este sitio» y quitarlo en Ajustes, la
ventana al día sola, y la bóveda cerrada con «Ya la he abierto». Hay que **reinstalar la extensión** con
los zip de la 2.21.0, o la tarjeta no sale.

**Con la 2.21.0 probó hasta cambiar la contraseña** y salió un fallo en Google, que pide usuario y
contraseña en páginas separadas: rellenaba y ofrecía actualizar la cuenta de otro usuario. **Corregido
en la 2.21.1** recordando el usuario de la página anterior, y con varias cuentas se rellena la que
coincida con él, a petición del cliente. **Publicada la 2.21.1 el mismo día**, con todas las
compilaciones en verde, y **Google funciona perfecto** con ella. Después, **cambiar la contraseña en Brevo
no ofrecía nada** —actual y nueva sin `autocomplete`, solo distinguidas por el nombre—: corregido en la
**2.21.2, publicada el mismo día**. **Y Brevo también funciona**, comprobado por el cliente. **Y con la 2.21.2 los pasos 5 a 8 también funcionan**, dicho por el cliente: el inicio fallido no ofrece nada, «Nunca en este sitio» se deshace en Ajustes, la ventana se pone al día sola y con la bóveda cerrada «Ya la he abierto» deja guardar. Y **la extensión no escribe nada en la consola** por
defecto, decidido por el cliente: los diagnósticos se le piden con un fragmento para pegar (ADR 0032, «Corregido tras probarla en
Google»). **Quedan por probar los pasos 5 a 8.**

Se le explicó cómo decide «la contraseña era mala» y se le pidió que apunte **los sitios donde falle en
cualquiera de los dos sentidos** —no sale tras entrar bien, o sale tras un error—: esa heurística se
afina con esos casos (`docs/deuda.md`). Si algo no hace nada, lo primero es la consola de la extensión.

**La siguiente acción concreta son las tiendas (entrega 5)**, que hay que plantear con el cliente antes de escribir nada.

### 5. Las tiendas (entrega 5) — en curso desde el 2026-09-14

**Decidida y planificada el 2026-09-14** (plan aprobado por el cliente), por preguntas con opciones y tras
investigar las dos tiendas en su documentación oficial: **públicas en Chrome y en Firefox**, subida
automática al publicar, cuentas de Webcafeína con info@webcafeina.com, **aviso de consentimiento en el
panel la primera vez**, datos de comerciante públicos en la ficha de Chrome, **web del proyecto con GitHub
Pages** para la privacidad y el soporte, y **Windows arreglado antes de publicar**.

Cuatro bloques, en orden:

1. **~~Windows~~ — hecho el 2026-09-14, sin publicar** (ADR 0034): manifiestos apuntados desde el registro
   y el puente en el instalador. Probado con un registro de mentira aquí; el de verdad y el instalador se
   comprueban en la máquina Windows de GitHub al publicar. **Ningún navegador lo ha lanzado en Windows.**
2. **~~La extensión, lista para las tiendas~~ — hecho el 2026-09-14, sin publicar** (ADR 0033): el aviso
   de datos en el panel la primera vez, que hacen cumplir también el trabajador de fondo y el guion de la
   página; sin `activeTab`; Firefox 140 con `data_collection_permissions`; la versión pasada de verdad;
   **sin `innerHTML`**; y el código fuente para Mozilla, **reproducible byte a byte** en una carpeta limpia.
   Queda para el bloque 4 quitar `key` del paquete de la tienda de Chrome.
3. **La web del proyecto** en `web/` con GitHub Pages: portada, privacidad y soporte.
4. **Las fichas** (`docs/tiendas/`, imágenes con Playwright), **el trabajo `tiendas`** en `publicar.yml`
   (Firefox con `web-ext`, Chrome con la API v2 y cuenta de servicio) y **los pasos del cliente** para dar
   de alta las cuentas, hacer la primera subida a Chrome a mano y guardar los secretos.

Lo que se preguntó entonces, ya decidido:

- **Pública u oculta** en cada tienda: la de Chrome admite «no listada» y la de Firefox (AMO) permite
  firmar sin listar.
- **A nombre de quién** van las cuentas de desarrollador —Webcafeína— y quién las administra. La de
  Chrome cuesta una cuota única.
- **El identificador de la extensión en Chrome cambia al subirla**: hay que añadirlo al manifiesto de
  native messaging al lado del actual (`docs/deuda.md`, el canal del navegador), o fijar la clave como
  ahora.
- **Qué contar en las fichas**: los permisos —`alarms`, anfitriones, `favicon`, native messaging— tienen
  que justificarse, y hace falta una política de privacidad que diga lo que ya dice `docs/seguridad.md`.
- **Cómo se actualiza**: hoy la extensión viaja en cada publicación de GitHub en dos zip; con las tiendas,
  si se sigue publicando ahí y quién sube cada versión.
- Y lo que queda de Windows para que la extensión funcione allí: **el manifiesto en el registro** y el
  instalador sin `esfinge-puente` (`docs/deuda.md`).

### Lo que yo recomendaría meter en medio, y no es lo que se decidió

**Levantar Chromium con la extensión cargada de verdad.** Lo dejo escrito porque la razón sigue en
pie aunque el orden sea otro: la entrega 2 salió con la detección de campos bien probada y **con lo
que la envuelve sin probar por nadie**, y ahí aparecieron los dos únicos fallos —el oyente del panel
contestando también desde las tramas de otro origen, y `yaRellenados` bloqueando un relleno pedido a
mano—. Los dos los vio el cliente a la primera. Playwright puede hacerlo (`launchPersistentContext`
con `--load-extension`, más un manifiesto de native messaging escrito en ese perfil apuntando a un
host de mentira). Está en `docs/deuda.md` con severidad alta.

### Y lo que solo dice el uso, que no es una tarea sino una escucha

Si el relleno acierta en un **sitio difícil** —un banco—; si **rellenar solo resulta demasiado**, y
entonces hace falta un interruptor en Ajustes en vez de afinar la detección a ciegas; y si el **clic
de más molesta** cuando hay varias cuentas del mismo sitio, que es de lo que depende el desplegable
dentro del campo.

Y sigue abierta la deuda de Windows, que la entrega 2 no toca: **el manifiesto del navegador va al
registro y no se escribe**, y el instalador no copia `esfinge-puente`.

**Y lo que sigue siendo verdad aunque la fase 2 esté empezada:** la puerta del plan era no empezarla
hasta usar la bóveda a diario, y se empezó con un día. Fue una decisión tomada a sabiendas, no un
descuido, y conviene recordarla si la entrega 2 se hace larga: lo que la justifica es que la bóveda
se use, no que la extensión avance.

De paso sigue pendiente juzgar el oro de la 2.11.0 puesto, y ver **cuántos de los 65 sitios acaban
con icono real**: de ese número depende si hay que leer el HTML de los que faltan.

Después, y sin prisa, sigue pendiente abrir la aplicación en **Windows** y en **GNOME** de verdad.

Para lo que venga: la **compilación con inspector** ya existe.
`gh workflow run compilar.yml -f inspector=true` da un paquete con el Web Inspector abierto, en el
sistema de verdad, para probar hipótesis en vivo sin publicar una versión por cada una. Va marcado
por tres sitios —nombre del artefacto, retención y sufijo en la versión— para que no se confunda con
una compilación normal.

## Bloqueantes

Ninguno técnico. Lo pendiente son comprobaciones que solo puede hacer el humano.

## Preguntas abiertas para el humano

Queda una, y es de código publicado que no se puede ejercitar aquí: en esta máquina no hay Windows ni
GNOME con la aplicación puesta, y lo visual no se afirma desde una compilación en verde.

- **¿Se usa la bóveda?** Es la única pregunta que decide las fases 2 a 4, y no la contesta ninguna
  prueba: la contesta el uso. Con ella hay que mirar tres cosas que aquí no se pueden ver: si el
  CSV de Dashlane se importa entero y bien, si el bloqueo a los quince minutos molesta o tranquiliza,
  y si copiar una contraseña con el borrado a los treinta segundos llega a tiempo de pegarla.
- **¿Cómo queda el oro puesto, en un Mac?** Es la pregunta de la 2.11.0, y sobre todo por **la
  pastilla dorada de la fila activa**: es el cambio más visible de todos, cumple de sobra —8,38:1— y
  es la gramática del icono, pero eso lo dice el cálculo y no el ojo. Si canta, el repliegue está
  pensado: fondo dorado tenue y texto de tinta, como hace Fluent. Mirar también si el ámbar de los
  avisos y el oro se estorban en la pantalla de cifrar, que son vecinos.
- **¿Cómo queda la aplicación en Windows y en GNOME?** La de macOS está juzgada en un Mac; las otras
  dos se escribieron a partir de las convenciones de cada sistema y nadie las ha visto corriendo. Hay
  tres cosas que mirar de una vez: **la estructura** —el panel de Fluent con su barra de acento, la
  cabecera de GNOME—, **el icono de los `.esf`** en el explorador de ficheros, y ahora **el lockup**,
  que ahí sube hasta el borde porque no hay semáforos que esquivar.
