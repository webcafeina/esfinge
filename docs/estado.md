# Estado

Última actualización: **2026-09-08**

## Dónde estamos

Esfinge es una **aplicación de escritorio** con ventana propia, más una línea de comandos que
comparte núcleo y formato. Va por la **2.7.0**. Funciona de punta a punta: cifra y descifra textos y
ficheros, genera contraseñas, guarda un historial de qué y cuándo, y se compila sola para macOS,
Windows y Linux en GitHub Actions.

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
  gestor de paquetes, que instala como root. Es la única conexión que hace el programa, dicha y
  apagable en Ajustes.
- **Menús del sistema en español** en los tres sistemas, construidos a mano porque los roles de Wails
  traen los rótulos en inglés escritos a fuego (ADR 0015). Con atajos ⌘1…⌘5 a las cinco pantallas.
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
- **Pruebas de la interfaz** con Playwright contra el Go de verdad, en tema claro y oscuro, en una
  máquina sin entorno gráfico.

## En curso

Nada a medias. Los tres sistemas tienen ya su estructura. La de macOS está probada en un Mac; la de
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

## Siguiente acción concreta

**Abrir la 2.6.0 en el Mac y juzgar la estructura.** Si los semáforos caen donde deben sobre la barra
lateral, si la ventana se arrastra por donde se espera, y si la barra lateral pasa por una del
sistema. De eso depende lo siguiente, que es llevar Windows y Linux a lo suyo.

## Bloqueantes

Ninguno técnico. Lo pendiente son comprobaciones que solo puede hacer el humano.

## Preguntas abiertas para el humano

**Las dos primeras son de código publicado que nunca se ha ejecutado aquí**: en esta máquina no hay
ni Mac ni Windows.

- **El vidrio sigue sin verse**, y queda aparcado a propósito. La 2.5.2 arregló lo que sí estaba mal
  —Wails deja la ventana opaca y el material no tenía nada que mezclar— y aun así no se apreciaba. La
  estructura nueva toca la misma zona, así que conviene volver a mirarlo con la 2.6.0 puesta antes de
  seguir tirando del hilo.
- **¿Salta el aviso de Gatekeeper al actualizarse desde dentro?** El guion le quita la cuarentena al
  paquete antes de ponerlo, y como el DMG lo descarga Go y no un navegador, es posible que no salte.
  Si no salta, hay que quitarlo del LÉEME del DMG para las actualizaciones y dejarlo solo para la
  primera instalación.
- **¿La estructura pasa por nativa?** Es la pregunta de la 2.6.0, y de la respuesta depende si
  Windows y Linux se hacen igual o distinto.
- **¿Merece la pena que el Finder enseñe el icono propio de los `.esf`?** La asociación funciona y el
  doble clic abre lo que toca —comprobado en el Mac—, pero el icono del documento no se ha mirado. Es
  lo único que queda de ese frente, y es cosmético.
