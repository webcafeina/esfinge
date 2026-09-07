# ADR 0006 — De herramienta de terminal a aplicación de escritorio, con Wails

**Fecha:** 2026-09-07 · **Estado:** aceptada · **Sustituye a la interfaz de menús de la 1.x**

## Contexto

La 1.x tenía dos caras: menús de terminal para el cliente y comandos para quien la encarga. Los
menús estaban cuidados —ratón, botones, temas medidos— pero seguían viviendo en el Terminal, y eso
es pedirle a alguien que no vive ahí que entre en un sitio donde no está cómodo.

## Decisión

Aplicación con ventana propia hecha con **Wails v2**: Go por dentro, React 19 y TypeScript por
fuera, que es el stack de los demás proyectos de la casa. Los menús de terminal se retiran. La línea
de comandos se queda.

## Alternativas descartadas

- **Fyne, en Go puro.** Un solo lenguaje y sin Node en la construcción, pero dibuja sus propios
  controles: no parece una aplicación del sistema, y sus diálogos de fichero son suyos y no los de
  macOS. Chocaba con las dos cosas que se habían pedido.
- **Electron.** Empaqueta bien y se puede generar desde Linux, pero pesa unos 150 MB frente a 18.
- **Servir la interfaz en el navegador.** Habría conservado la compilación cruzada, pero es una
  pestaña, no una aplicación.

## Consecuencias

- **La aplicación ya no se compila en la máquina de desarrollo**: necesita el webview de cada
  sistema, y aquí faltan `webkit2gtk` y `pkg-config`, sin `sudo` sin contraseña. Lo hace GitHub
  Actions. La línea de comandos sigue cruzando de plataforma sin problema.
- **El `main.go` de la aplicación vive en la raíz del proyecto**, no en `cmd/`, que no es idiomático
  en Go. Lo impone Wails: genera los enlaces buscando el paquete main junto a `wails.json`, y en
  `cmd/` falla con «no Go files». Costó un intento de compilación descubrirlo.
- Al retirar la interfaz de terminal se fue con ella Bubble Tea, y con Bubble Tea **el arranque de
  cinco segundos** en terminales que no contestan al OSC 11: de 5,1 s a 0,11 s.
- Se retiraron 1.830 líneas y sus 1.492 de test.

## Verificación

Los tres sistemas compilan en GitHub Actions. El paquete de macOS es universal —Intel y Apple
Silicon— y ambos con firma ad-hoc. Lo que **no** está verificado es la aplicación abierta en un
escritorio: ver [deuda.md](../deuda.md).
