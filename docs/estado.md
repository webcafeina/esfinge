# Estado

Última actualización: **2026-09-07**

## Dónde estamos

Esfinge es una **aplicación de escritorio** con ventana propia, más una línea de comandos que
comparte núcleo y formato. Va por la **2.4.0**. Funciona de punta a punta: cifra y descifra textos y
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
- **Pruebas de la interfaz** con Playwright contra el Go de verdad, en tema claro y oscuro, en una
  máquina sin entorno gráfico.

## En curso

Nada a medias. El código de las 2.1.0 a 2.4.0 está escrito, probado hasta donde se puede desde una
máquina sin Mac ni Windows, y publicado.

## Comprobado en un Mac de verdad

- Que la comprobación de versiones funciona: con la última instalada no sale la banda, y «Buscar
  ahora» dice que ya se está al día.
- La descarga de una actualización, con su barra.
- El doble clic en un `.esf`, en los dos momentos —con Esfinge cerrada y con Esfinge abierta— y con
  las dos clases de contenedor: el que lleva un fichero abre la pantalla de ficheros y el que lleva
  un texto abre la de texto, con la línea puesta.

## Siguiente acción concreta

**Esperar a que el humano pruebe dos cosas en su Mac**, que son el código nuevo con más riesgo y no
se pueden ejercitar desde aquí. Están abajo, en las preguntas abiertas. Hasta entonces no hay nada
que empezar: lo que venga después depende de lo que salga de ahí.

## Bloqueantes

Ninguno técnico. Lo único pendiente son dos comprobaciones que solo puede hacer el humano.

## Preguntas abiertas para el humano

**Las dos primeras son las que hay que resolver al volver.** Son de código escrito y publicado que
nunca se ha ejecutado aquí: en esta máquina no hay ni Mac ni Windows.

- **¿Copiar y pegar siguen bien dentro de los campos?** Al construir los menús a mano (ADR 0015) se
  perdieron los selectores nativos, así que ⌘C, ⌘X, ⌘V y ⌘A pasan ahora por código propio: el menú
  manda una orden y la interfaz la ejecuta sobre el campo con el foco. El pegar es el más delicado,
  porque el portapapeles lo lee Go y el texto lo coloca la interfaz en el cursor. **Si algo falla
  ahí, falla pegar una contraseña**, que es lo que más se hace con Esfinge.
- **¿Salta el aviso de Gatekeeper al actualizarse desde dentro?** El guion le quita la cuarentena al
  paquete antes de ponerlo, y como el DMG lo descarga Go y no un navegador, es posible que no salte.
  Si no salta, hay que quitarlo del LÉEME del DMG para las actualizaciones y dejarlo solo para la
  primera instalación.
- **¿La ventana ya pasa por nativa?** La 2.0.1 rehízo el aspecto siguiendo macOS —barra translúcida,
  radios generosos, controles de 28 px— pero eso solo se juzga con la aplicación abierta en un Mac.
  Las capturas salen de un navegador y ahí la transparencia no se ve.
- **¿Merece la pena que el Finder enseñe el icono propio de los `.esf`?** La asociación funciona y el
  doble clic abre lo que toca —comprobado en el Mac—, pero el icono del documento no se ha mirado. Es
  lo único que queda de ese frente, y es cosmético.
