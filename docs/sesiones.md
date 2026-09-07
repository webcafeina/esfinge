# Sesiones

Bitácora. Una entrada por sesión, la más reciente arriba. Sirve para retomar exactamente donde se
dejó aunque se pierda la conversación.

Plantilla al final.

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
- Queda abierto: montar el DMG en un Mac y ver cómo queda la ventana. Desde aquí solo se puede leer
  su contenido, no verlo.

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
