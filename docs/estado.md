# Estado

Última actualización: **2026-09-07**

## Dónde estamos

Esfinge es una **aplicación de escritorio** con ventana propia, más una línea de comandos que
comparte núcleo y formato. Va por la **2.2.0**. Funciona de punta a punta: cifra y descifra textos y
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
- **Actualizaciones dentro de la aplicación**: aviso, descarga comprobada con SHA256 y entrega al
  instalador de cada sistema. La única conexión que hace el programa, dicha y apagable en Ajustes.
- **Línea de comandos**: intacta desde la 1.5.0, con sus códigos de salida distintos por caso.
- **Color generado desde Go** (`internal/tema`), con el contraste de los dos temas medido en cada
  compilación.
- **Compilación automática** de los tres sistemas, más los seis binarios de la línea de comandos.
- **Pruebas de la interfaz** con Playwright contra el Go de verdad, en tema claro y oscuro, en una
  máquina sin entorno gráfico.

## En curso

Nada a medias. La 2.1.0 cierra el círculo de la distribución: la aplicación comprueba si hay versión
nueva, se descarga el instalador de su sistema y lo abre. Con eso, el que la tiene instalada deja de
depender de que alguien le avise.

## Siguiente acción concreta

**Actualizar de verdad, desde dentro de la aplicación.** Instalar la 2.0.3 en el Mac, publicar la
2.1.0 y ver el camino entero: que salga la banda, que la descarga llegue, que el DMG se monte solo y
—esto es lo interesante— si la copia instalada así se libra del aviso de Gatekeeper, porque la
cuarentena la pone quien descarga y aquí descarga Go. No se puede comprobar desde esta máquina.

## Bloqueantes

Ninguno.

## Preguntas abiertas para el humano

- **¿La ventana ya pasa por nativa?** La 2.0.1 rehízo el aspecto siguiendo macOS —barra translúcida,
  radios generosos, controles de 28 px— pero eso solo se juzga con la aplicación abierta en un Mac.
  Las capturas salen de un navegador y ahí la transparencia no se ve.
- **¿Merece la pena cerrar el `.esf` en macOS del todo?** Ya abre la aplicación con el fichero
  puesto. Falta comprobar que el Finder enseña el icono correcto.
