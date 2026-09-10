# ADR 0025 — Los códigos de un solo uso, en la misma bóveda

**Fecha:** 2026-09-10 · **Estado:** aceptada · **Matiza la [0023](0023-la-boveda.md)** ·
**Revisar si** aparece el autorrelleno, o si alguna vez hay bóvedas separadas

## Contexto

La bóveda de la [ADR 0023](0023-la-boveda.md) guardaba desde el primer día un
campo `TOTP` con la semilla del segundo factor —la importación de Dashlane la
trae, y también la de Bitwarden y la de 1Password—, pero **nadie calculaba el
código de seis cifras**. Lo que se veía en pantalla era la semilla en base32, que
no sirve para entrar en ningún sitio.

Eso convertía la fase 1 en media mudanza: las contraseñas se podían llevar y los
segundos factores no. Quien quisiera entrar en su banco seguía necesitando
Dashlane abierto al lado, y con él **media razón para no dejarlo**.

Es, además, la pieza barata de todo el plan de sustitución: un algoritmo
estándar, congelado desde 2011, sin dependencias y sin red.

## Decisión

**Esfinge calcula los códigos, en Go, a partir de la semilla que ya guarda.**

Con cuatro decisiones alrededor que es lo que esta ficha viene a dejar escrito.

### El segundo factor vive al lado de la contraseña, y hay que decirlo

Es la consecuencia incómoda, y no se arregla con una nota al pie: **una bóveda
abierta entrega la contraseña y el segundo factor a la vez**. Quien tenga la
contraseña maestra los tiene los dos, así que contra ese atacante concreto el
segundo factor no está haciendo el trabajo que su nombre promete.

Se hace igualmente, por dos razones que no se anulan entre sí:

- **Contra quien no tiene la bóveda —que es contra quien se inventó el segundo
  factor— sigue valiendo entero.** Una contraseña filtrada en la brecha de un
  servicio no abre nada sin el código, y esa es la amenaza de la que protege TOTP
  el 99 % de las veces.
- **Y la alternativa realista no es «dos sitios distintos», es «Dashlane».** El
  cliente no va a repartir sus segundos factores entre dos programas: los tenía
  todos en el mismo sitio que las contraseñas antes de esto, y los tendrá todos
  en el mismo sitio después. Lo que cambia es **cuál** es ese sitio.

Está escrito en `docs/seguridad.md`, en el apartado de lo que la bóveda no
protege, y en los mismos términos que aquí.

### Se escribe, no se trae

Son cuarenta líneas de biblioteca estándar: un HMAC, un contador de ocho bytes y
una división. En un programa que guarda contraseñas, **cada dependencia nueva es
código de otro dentro del binario que custodia los secretos**, y el paquete de
TOTP más usado de Go arrastra más de lo que resuelve.

Vive en `internal/codigos`, que no toca disco, no sale a la red y no sabe qué es
una bóveda: recibe texto y una hora, y devuelve seis cifras.

### El cálculo va en Go, no en la ventana

Podría hacerlo el JavaScript —el algoritmo cabe igual y la semilla ya cruza el
puente al abrir una entrada—. Se hace en Go por dos motivos:

- **La semilla es el segundo factor entero.** Cuanto menos viva en el montón del
  webview, mejor, y si algún día `VerDeBoveda` deja de mandarla —que sería lo
  suyo— el método ya está en su sitio.
- **La línea de comandos también lo necesita**, y ahí no hay ventana. Escrito una
  vez, las dos caras dan el mismo código; escrito en la interfaz, la segunda copia
  se escribiría aparte y divergiría.

Lo que cruza el puente son **las seis cifras ya calculadas**, que caducan en
treinta segundos, más cuánto les queda de vida.

### Pedir el código no cuenta como actividad

La ventana lo vuelve a pedir sola cada vez que caduca, para que en pantalla esté
siempre el que vale. Si eso tocara el reloj del bloqueo, **una entrada abierta
encima de la mesa mantendría la bóveda abierta para siempre** y el «se cierra a
los quince minutos» dejaría de ser verdad.

Es la misma regla que el goteo de iconos de la [ADR 0024](0024-iconos-de-los-sitios.md),
y la segunda vez que aparece: conviene tratarla ya como una regla del proyecto y
no como un detalle de cada caso.

## Alternativas descartadas

- **Una biblioteca de TOTP.** Arriba: código de otro dentro del binario que
  guarda las contraseñas, para resolver cuarenta líneas.
- **Calcularlo en la interfaz.** Arriba: la semilla se quedaría viviendo en el
  webview y la línea de comandos necesitaría su propia copia.
- **Aceptar también HOTP** (`otpauth://hotp/…`). Va por un contador que hay que
  guardar y **subir a cada uso**: no es una semilla que se lee, es un estado que
  se escribe, con su sincronización y su ventana de reintentos. Casi nadie lo usa.
  Prometerlo a medias sería peor que rechazarlo, así que se rechaza con un error
  que dice exactamente por qué.
- **Corregir el desfase del reloj de la máquina**, probando también el intervalo
  anterior y el siguiente. Eso lo hace **quien valida**, no quien genera: por este
  lado no hay forma de saber cuál de los tres códigos aceptaría el servicio, y
  enseñar tres no es enseñar uno. Si el reloj va desviado más de medio minuto, el
  código no vale y se dice en la ayuda.
- **Guardar el código calculado.** No tendría sentido: caduca en treinta segundos
  y la semilla es lo que permite volver a calcularlo sin preguntarle nada a nadie.
- **Enseñar la semilla junto al código**, como hacía antes. La semilla es lo único
  que hay que proteger de verdad aquí, y ahora solo aparece en el editor, que es
  donde se pega al crear la entrada.

## Consecuencias

- **La bóveda pasa a ser también el autenticador.** Con eso se va la última cosa
  que obligaba a tener Dashlane abierto, y también la separación entre los dos
  factores. Las dos cosas son ciertas a la vez.
- **El código se enseña destapado**, al revés que todo lo demás de esa pantalla.
  Es deliberado: caduca en treinta segundos y hay que poder teclearlo mirando.
- **La línea de comandos gana `esfinge boveda codigo`**, que es lo que le faltaba
  a un script para no tener que coger el teléfono. Y es otra costura de la que
  puede colgar el autorrelleno cuando llegue.
- **Una semilla mal copiada no se detecta aquí**, y no hay forma de que se
  detecte: cualquier cadena de letras es base32 válida. Lo que sí se detecta es
  que le falte o le sobre un carácter, y que lleve un `0`, un `1`, un `8` o un
  `9`, que no están en ese alfabeto.

## Verificación

- **Los vectores de RFC 4226 y RFC 6238**, los dos apéndices enteros: los diez
  contadores del primero y los ocho instantes del segundo, con SHA-1, SHA-256 y
  SHA-512 —cada uno con su semilla, que la tabla del RFC alarga sin decirlo—.
  Son números que no ha calculado este programa, que es lo que los hace valer.
- Que la semilla entra **como la copia un humano**: en minúsculas, con espacios,
  con guiones, con relleno y sin él; y que una a la que le falta un carácter
  **no se adivina**.
- Que la URI `otpauth://` con sus parámetros da el mismo código que la semilla
  pelada, y que lo que no venga toma el valor por defecto del RFC.
- **El recorrido entero desde la bóveda**, con el reloj parado: guardar una
  entrada con semilla y pedir el código por el mismo método que usa la ventana,
  comparado contra el vector del RFC. No prueba el algoritmo —eso es cosa del
  paquete— sino que **están conectados**.
- Y la que gana el sueldo de esta ficha: **pedir el código treinta veces seguidas
  con la bóveda quieta y comprobar que se cierra igual**. Si alguien añade un
  `Actividad()` ahí, esa prueba se pone roja.
- En la ventana, con Playwright y en los dos temas: que salen seis cifras, que en
  el texto van **seguidas** aunque en pantalla se vean en dos grupos —el hueco lo
  pone el CSS, para que al seleccionarlas se copien como las espera el servicio—,
  que **la semilla no aparece en la pantalla**, y que la cuenta atrás corre.
- Y a mano, contra una implementación de fuera: el mismo instante y la misma
  semilla dando el mismo código en Esfinge y en tres líneas de Python.

**Lo que no se ha comprobado:** que un código de Esfinge abra de verdad una cuenta
de un servicio real. Eso solo puede hacerlo el cliente, y es la comprobación que
importa; los vectores del RFC dicen que el algoritmo está bien, no que la semilla
que Dashlane exportó sea la que el servicio espera.
