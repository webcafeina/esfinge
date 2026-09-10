# Seguridad

Última actualización: **2026-09-09**

Qué protege Esfinge y qué no. En una herramienta que cifra, lo segundo importa tanto como lo
primero: una expectativa equivocada sobre lo que protege es exactamente lo que hace daño.

## Lo que hace

Coge un secreto y una clave y produce un contenedor que **solo se abre con esa clave**. Quien tenga
el contenedor y no la clave no tiene nada: ni el contenido, ni una versión parcial, ni pistas sobre
su longitud más allá de la evidente.

Y **detecta si el contenedor ha cambiado**. Alterar un byte, cortarlo por la mitad o reordenar sus
partes hace que la apertura falle en vez de devolver algo distinto de lo que se guardó.

## De qué protege

| Amenaza | Mitigación | Riesgo residual |
|---|---|---|
| Alguien intercepta el texto cifrado por correo o chat | XChaCha20-Poly1305; sin la clave no hay contenido | Se ve **que** hay algo cifrado y su tamaño aproximado |
| Alguien roba el fichero `.esf` de un disco | Lo mismo | Igual |
| Alguien modifica el contenedor por el camino | Etiqueta de autenticación por segmento, con la cabecera autenticada | Ninguno conocido: cualquier cambio rompe la apertura |
| Alguien corta el fichero y lo pasa por entero | Marca en el último segmento ([ADR 0003](adr/0003-marca-de-final.md)) | Cortar dentro del **primer** segmento no se distingue de una clave equivocada |
| Fuerza bruta sobre la clave | Argon2id con 64 MiB y 3 pasadas: cada intento cuesta medio segundo y mucha memoria | Una clave corta sigue siendo una clave corta. El medidor avisa |
| Dos mensajes iguales delatan que lo son | Sal y nonce nuevos en cada operación | Ninguno |

## De qué NO protege

Esto es lo importante de este documento.

- **De quien ya está dentro de la máquina.** Mientras la ventana está abierta, la clave vive en
  memoria. Quien pueda leer la memoria del proceso, poner un registrador de teclas o hacerse pasar
  por el usuario, no necesita romper nada.
- **De perder la clave de un `.esf`.** No hay recuperación, ni puerta trasera, ni copia en ninguna
  parte. Si se pierde la clave, el contenido se ha perdido. Esto no es un fallo: es lo que significa
  cifrar.

  **La bóveda es la excepción, y es deliberada.** Ahí sí hay una segunda llave —la clave de
  recuperación— porque perder la contraseña maestra no puede significar perder *todas* las
  contraseñas de golpe. Eso trae su propia contrapartida, y va abajo.
- **De que se sepa qué has cifrado.** El historial guarda nombres de fichero y fechas
  ([ADR 0010](adr/0010-que-guarda-el-historial.md)). No guarda contenidos ni claves, pero saber que
  el martes cifraste `credenciales-banco.env` ya dice algo. Se puede vaciar desde la propia ventana.
- **De un canal inseguro para la clave.** Si el contenedor y la clave viajan por el mismo correo,
  quien lea ese correo lo tiene todo. Esto no lo puede resolver el programa.
- **De un fichero manipulado antes de cifrarlo.** Esfinge sella lo que le den; no sabe si lo que le
  dieron era lo que debía.
- **Del portapapeles.** Al cifrar, el resultado se copia solo ([ADR 0011](adr/0011-copiado-automatico.md)).
  Cualquier programa que vigile el portapapeles lo verá. Sin la clave no le sirve, pero conviene
  saberlo. Al descifrar no se copia nada por su cuenta, justamente por esto.

  **Y desde la 2.8.0, «Usar como clave» copia también la contraseña generada**, que sí es un secreto
  aprovechable por sí mismo. Va en la misma línea que lo anterior —se copia para poder pegarla en un
  gestor de contraseñas sin dar un rodeo— pero la diferencia importa: lo que queda en el portapapeles
  es la llave, no el candado.

  **Desde la 2.12.0 el portapapeles se borra solo** pasado el plazo que diga Ajustes, y eso arregla ese
  agujero. Con dos límites que conviene saber: solo se borra **si sigue conteniendo lo que Esfinge
  puso** —nunca se pisa lo que se haya copiado después—, y un gestor de portapapeles del sistema, o
  el Portapapeles Universal de Apple, ya se lo pueden haber llevado a otro sitio. Eso no lo puede
  borrar nadie.

## Lo que cambia con la bóveda

La bóveda guarda contraseñas, así que **cambia el modelo de amenazas del propio programa**. Antes,
un fallo perdía un fichero; ahora puede perderlas todas. Lo que hay que tener claro:

- **La clave de recuperación es una segunda puerta a todo.** Quien la consiga tiene la bóveda
  entera, y no caduca: hasta que no se rote —lo que genera una clave nueva y deja la anterior
  inservible— sigue abriendo. Se enseña **una sola vez** al crear la bóveda, no se guarda en ninguna
  parte, y no se puede volver a ver. Guardarla es tan importante como guardar la maestra, y en otro
  sitio distinto.
- **El historial de contraseñas conserva las anteriores.** Cambiar una contraseña no borra la vieja:
  se guarda para el caso de «cambié la contraseña y el servicio no se enteró». Eso significa que un
  secreto sustituido **sigue dentro de la bóveda** hasta que se borre a mano.
- **La bóveda está abierta durante horas**, y mientras lo está, todas las contraseñas viven
  descifradas en memoria. Un volcado de memoria o el fichero de intercambio pueden contenerlas. Es
  la misma limitación de siempre —Go no permite borrar una cadena— pero aquí la ventana es mucho más
  larga. Se acorta con el bloqueo por inactividad, que va a quince minutos por defecto, y
  mandando a la ventana **una contraseña cada vez**, solo cuando se pide, en vez de la lista entera.
- **El segundo factor está al lado de la contraseña, y eso le quita parte de su gracia.** Desde la
  2.15.0 la bóveda calcula los códigos de un solo uso, así que **una bóveda abierta entrega la
  contraseña y el código a la vez** y quien tenga la contraseña maestra los tiene los dos. Contra
  ese atacante concreto el segundo factor deja de ser un segundo factor. Contra el que se inventó
  —una contraseña filtrada en la brecha de un servicio, que es lo que pasa el 99 % de las veces—
  sigue valiendo entero. Se hace así porque la alternativa realista no era tenerlos separados: era
  tenerlos juntos en Dashlane ([ADR 0025](adr/0025-los-codigos-de-un-solo-uso.md)). Quien quiera de
  verdad dos factores separados tiene que dejar la semilla fuera de aquí, en un teléfono o en una
  llave física, y eso Esfinge no lo puede decidir por nadie.
- **Los iconos van cifrados aunque un icono sea público.** Lo que hay que ocultar no es el dibujo: es
  **la lista de sitios**. Una carpeta con `banco.es.png` y `hacienda.es.png` diría qué hay dentro de
  la bóveda, y hashear los nombres no salvaría nada.
- **El fichero de la bóveda es JSON en claro por fuera.** Lo cifrado son los campos. Quien pueda
  escribirlo no puede leer nada ni fabricar una bóveda que abra, pero sí puede estropearla o
  revertirla a una copia vieja. **Se detecta al abrir** —hay un sello por dentro que no cuadraría—
  pero no se impide.

## Dónde queda algo en disco

| Qué | Dónde | Permisos |
|---|---|---|
| Historial | Carpeta de configuración del usuario, `Esfinge/historial.json` | `600` |
| **La bóveda** | La misma carpeta, `Esfinge/boveda.esfinge`, más `.anterior` con la copia previa | `600` |
| Los iconos de los sitios | `Esfinge/boveda.esfinge.iconos`, **cifrado con la clave de la bóveda** | `600` |
| Preferencias | La misma carpeta, `Esfinge/preferencias.json` | `600` |
| La actualización descargada | Carpeta de caché del usuario, `Esfinge/descargas/` | `600` |
| Ficheros cifrados | Junto al original, con `.esf` al final | `600` |
| Lo que se guarda desde la ventana | Donde diga el diálogo del sistema | `600` |

## Lo único que sale de la máquina

Desde la 2.14.0 son **dos**, y conviene saber exactamente cuáles.

**1 · Una petición `GET` a `api.github.com`, una vez al día**, para preguntar cuál es la última
versión publicada ([ADR 0014](adr/0014-comprobacion-de-actualizaciones.md)). En ella viaja el número
de versión instalada, dentro del `User-Agent`, que es lo que se compara; y GitHub ve la dirección IP,
como cualquier página que se visite.

**2 · El icono de cada sitio de la bóveda, pedido al propio sitio**
([ADR 0024](adr/0024-iconos-de-los-sitios.md)). Se piden poco a poco, espaciados y en orden
aleatorio, unos pocos por sesión. **Nunca a un intermediario**: un servicio de iconos recibiría la
lista completa de sitios donde tienes cuenta.

Y aquí hay que decir algo que no es evidente: **ir directo no oculta esa lista, la reparte**. El
nombre del sitio viaja en claro en la consulta de DNS y en el saludo TLS, antes de que empiece el
cifrado, así que **quien pueda mirar tu red ve a qué sitios se pregunta**. Lo que se gana yendo
directo es no meter a una empresa de por medio, que es otra cosa. Al sitio no se le dice quién
pregunta: la petición no lleva el nombre ni la versión de Esfinge.

Viene **encendido**, se avisa la primera vez y se apaga en Ajustes. Sin él, cada entrada sale con un
cuadro de color y su inicial, que no sale de esta máquina.

**No hay telemetría, ni informes de fallos, ni identificadores.** Nada de lo que se cifra, ni los
nombres de los ficheros, ni cuántas veces se usa el programa, ni nada que permita distinguir una
instalación de otra.

Las dos se apagan en **Ajustes**, donde está dicho con estas mismas palabras.

**Y `ESFINGE_SIN_RED=1` las apaga todas, sin excepción.** Hasta la 2.14.0 esa variable solo la miraba
la línea de comandos —la ventana no la consultaba nunca—, así que quien la ponía creyendo que apagaba
la red apagaba la mitad. Ahora vive en un sitio y la consultan las dos salidas, con una prueba por
cada una. En la línea de comandos, además, no se pregunta nunca si la salida de error no es un
terminal, que es el caso de cualquier script.

Si se descarga una actualización, se comprueba su SHA256 contra el publicado. **Eso protege de una
descarga rota, no de una publicación manipulada**: el resumen sale del mismo sitio que el fichero. Lo
que sostiene la confianza es el TLS contra GitHub, y que la aplicación no se instala sola —el
instalador lo abre quien esté delante—.

## Decisiones que afectan a la seguridad

- [ADR 0002](adr/0002-formato-esf1.md) — El cifrado y por qué esos algoritmos.
- [ADR 0003](adr/0003-marca-de-final.md) — Por qué un fichero cortado se detecta.
- [ADR 0004](adr/0004-contrasenas-en-hexadecimal.md) — Por qué las contraseñas salen en hexadecimal.
- [ADR 0010](adr/0010-que-guarda-el-historial.md) — Qué se guarda y qué no.
- [ADR 0012](adr/0012-sin-firmar.md) — Por qué el sistema avisa al instalarla.

## Lo que no se ha auditado

Nadie de fuera ha revisado esto. El núcleo tiene pruebas que cubren la ida y vuelta, la manipulación
de cada byte, el truncado y la reordenación, y usa implementaciones de la biblioteca estándar
extendida de Go —`golang.org/x/crypto`— en vez de nada escrito aquí. Pero **una batería de pruebas
propia no es una auditoría**, y conviene decirlo antes de que alguien confíe más de la cuenta.
