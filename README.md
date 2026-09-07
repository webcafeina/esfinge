# Esfinge

Cifra y descifra contraseñas, ficheros de credenciales y cualquier otro secreto con una clave que
solo conocen las dos partes. De **Webcafeína**.

Viene en dos formas, y lo cifrado por una lo abre la otra:

- **La aplicación**, con su ventana y su icono, para usarla con el ratón.
- **La línea de comandos**, para meterla en un script o en una tubería.

## Aviso primero

**Sin la clave no hay forma de recuperar nada.** Esto no es una cuenta con «he olvidado mi
contraseña»: si se pierde la clave, el contenido se ha perdido, y no hay nadie —tampoco Webcafeína—
que pueda abrirlo. Guarda la clave antes de cifrar, no después.

Y manda el resultado y la clave por caminos distintos. Si van en el mismo correo, quien lea ese
correo lo tiene todo.

## La aplicación

Se abre con doble clic. Cuatro pestañas: **Cifrar**, **Descifrar**, **Generar** e **Historial**.

**Cifrar y descifrar** trabajan con un texto o con ficheros, según el botón de arriba. Con ficheros,
se pueden **arrastrar a la ventana** —incluso varios a la vez, que se cifran todos con la misma
clave y con su barra de progreso— o elegirlos con el diálogo de siempre del sistema. El original
nunca se toca: el cifrado aparece al lado con `.esf` al final.

**Al cifrar un texto, el resultado se copia solo al portapapeles** y la pantalla lo confirma. Al
descifrar no se copia nada por su cuenta: lo que sale ahí es el secreto en claro, y dejarlo en el
portapapeles sin que nadie lo pida es meterlo donde puede leerlo cualquier cosa.

**El historial** dice qué se cifró o descifró y cuándo. Nunca el contenido, ni la clave, ni el texto
cifrado. Vive en la carpeta de configuración del usuario, con permisos que solo dejan leerlo a su
dueño, y la propia pantalla enseña la ruta y el botón de vaciarlo.

**Los ficheros `.esf` quedan asociados a Esfinge.** En Windows y en Linux, hacer doble clic en uno
abre la aplicación directamente en descifrar con el fichero puesto. En macOS el Finder los reconoce
y ofrece abrirlos con Esfinge, pero el fichero hay que arrastrarlo o elegirlo desde dentro: macOS
entrega el fichero por una vía que la librería de la ventana todavía no expone.

### Instalar

Los paquetes salen de la compilación automática, en la pestaña Actions del repositorio.

En **macOS**, la aplicación no está firmada con una cuenta de desarrollador de Apple, así que el
sistema la bloquea la primera vez. Se quita con:

```sh
xattr -dr com.apple.quarantine /Applications/Esfinge.app
```

O desde **Ajustes del Sistema → Privacidad y seguridad**, buscando el aviso sobre Esfinge y pulsando
«Abrir de todos modos». Con una aplicación sin firmar el aviso es más aparatoso que con un programa
de terminal, y no hay forma de evitarlo sin pagar los 99 $ al año de Apple.

En **Windows**, si aparece «Windows protegió su PC», hay que pulsar «Más información» → «Ejecutar de
todas formas», por el mismo motivo.

## La línea de comandos

```sh
esfinge cifrar                                        # pregunta el secreto y la clave
echo -n 'secreto' | esfinge cifrar --clave-env CLAVE  # sin preguntar nada
esfinge cifrar    -i credenciales.env -o credenciales.env.esf
esfinge descifrar -i credenciales.env.esf
esfinge generar --bytes 32 -n 5
esfinge --help
```

De cifrar un texto sale una línea así:

```
ESF1.RVNGMQEAAAEAAAAAAAMEW77N6LiWIRQM4fm0l8P6mt8PbFAz4GJC-C0icDTG7h7iJWpBo
```

Se puede pegar en un correo, en un chat, en un `.env` o dentro de una URL sin escapar nada: no lleva
`/`, ni `+`, ni `=`. **Hay que copiarla entera, con el `ESF1.` de delante.**

Con ficheros aguanta cualquier tamaño sin cargarlos enteros en memoria, escribe con permisos `600` y
de forma atómica, y no sobrescribe nada sin `--forzar`.

La clave **nunca** se pasa como argumento: ahí quedaría en el historial del shell y la vería
cualquiera con un `ps`. Se lee de `--clave-env`, de `--clave-fichero` o preguntándola.

Cuando la salida no es un terminal sale el dato pelado, sin colores ni marca. Los códigos de salida
distinguen qué ha pasado:

| Código | Qué ha pasado |
|---|---|
| `0` | bien |
| `1` | error general |
| `2` | el comando está mal escrito |
| `3` | la clave no es correcta |
| `4` | el contenedor está dañado, cortado o no es de Esfinge |

### Generar contraseñas

Por defecto **hexadecimal**, y es a propósito. Una contraseña con `/` parte una cadena de conexión
—`postgres://usuario:pa/ss@host` deja de ser una URL— y el error que sale por el otro lado no
menciona la contraseña por ninguna parte, así que se pierden horas buscando donde no es. Los
alfabetos con símbolos están disponibles y avisan cada vez.

## Cómo está hecho

- **Cifrado:** XChaCha20-Poly1305, con nonce de 24 bytes al azar y la cabecera autenticada.
- **Derivación de la clave:** Argon2id con 64 MiB, 3 pasadas y paralelismo 4. Los parámetros viajan
  dentro del contenedor, de modo que subir el coste más adelante no rompe lo ya cifrado.
- **Ficheros:** troceados en segmentos de 64 KiB, cada uno con su etiqueta, su número de orden y una
  marca en el último. Eso es lo que hace que un fichero cortado por la mitad se detecte en vez de
  descifrarse a medias y en silencio.
- **Formato:** `ESF1`, versionado y compatible con la 1.x.

Nada de esto sale de la máquina. Esfinge no habla con internet.

## Desarrollo

Go 1.27, Node 22 y pnpm 11. En esta máquina Go vive en `~/.local/go`.

```sh
make comprobar   # vet, tests de Go y tipos de la interfaz
make contraste   # mide el contraste de los dos temas
make e2e         # mueve la interfaz de verdad contra el Go de verdad
make esfinge     # la línea de comandos
make ayuda       # el resto
```

La aplicación con ventana la compila GitHub Actions en los tres sistemas: una ventana necesita el
webview de cada sistema operativo y eso no cruza de plataforma. La línea de comandos sí, y sale para
los seis objetivos desde cualquier sitio.

**El color se genera desde Go.** `internal/tema` es la fuente de verdad y escribe
`frontend/src/tokens.css`; hay un test que falla si el fichero se queda atrás, y otro que mide el
contraste de las parejas que la interfaz usa de verdad, en los dos temas.

**La interfaz se prueba sin entorno gráfico.** El puente entre React y Go tiene dos caminos —Wails
en la aplicación, HTTP durante el desarrollo— y la interfaz no distingue cuál usa, así que
Playwright puede recorrerla entera contra el mismo Go que llevará la ventana.

---

▍ **webcafeína**
