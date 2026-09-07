# Estado

Última actualización: **2026-09-07**

## Dónde estamos

Esfinge es una **aplicación de escritorio** con ventana propia, más una línea de comandos que
comparte núcleo y formato. Va por la **2.0.3**. Funciona de punta a punta: cifra y descifra textos y
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
- **Línea de comandos**: intacta desde la 1.5.0, con sus códigos de salida distintos por caso.
- **Color generado desde Go** (`internal/tema`), con el contraste de los dos temas medido en cada
  compilación.
- **Compilación automática** de los tres sistemas, más los seis binarios de la línea de comandos.
- **Pruebas de la interfaz** con Playwright contra el Go de verdad, en tema claro y oscuro, en una
  máquina sin entorno gráfico.

## En curso

Nada a medias. La infraestructura —documentación, portada, licencia, instaladores de los tres
sistemas y el flujo que publica— está escrita y comprobada hasta donde se puede desde esta máquina.
El repositorio ya es público y la publicación de GitHub lleva sus adjuntos.

## Siguiente acción concreta

Abrir el DMG en un Mac: montarlo, ver si la ventana sale como se diseñó, arrastrar la aplicación a
Aplicaciones y comprobar que arranca y que el Finder enseña el icono de los `.esf`. Eso no se puede
hacer desde aquí.

## Bloqueantes

Ninguno.

## Preguntas abiertas para el humano

- **¿La ventana ya pasa por nativa?** La 2.0.1 rehízo el aspecto siguiendo macOS —barra translúcida,
  radios generosos, controles de 28 px— pero eso solo se juzga con la aplicación abierta en un Mac.
  Las capturas salen de un navegador y ahí la transparencia no se ve.
- **¿Merece la pena cerrar el `.esf` en macOS del todo?** Ya abre la aplicación con el fichero
  puesto. Falta comprobar que el Finder enseña el icono correcto.
