# 0049 · Las redes wifi, con su código QR

**Fecha:** 2026-10-01 · **Estado:** aceptada

## Contexto

El cliente lo pidió así: *«la funcionalidad nueva de incluir credenciales Wifi, como hace Dashlane»*. Es la
**séptima clase** de la bóveda, y la segunda que no guarda un secreto en el sentido clásico — guarda uno,
pero el valor de tenerla no es guardarlo.

**Lo que hace útil una red en un gestor es el código.** Un invitado apunta el móvil, se conecta y nadie
dicta en voz alta una contraseña de veinte caracteres. Eso es lo que el cliente usa de Dashlane, y es lo que
decidió el coste de esta entrega: **en el proyecto no había nada que dibujara un QR y ninguna dependencia
que pudiera hacerlo**.

El recorrido de «una clase nueva» ya estaba medido dos veces, con los datos personales (ADR 0047) y las
llaves de acceso (ADR 0048). Lo que no estaba medido era el dibujo.

## Decisión

Se decidió con el cliente, por preguntas con opciones, y no se cambia sin preguntar:

| | |
|---|---|
| **Cómo se llama** | **«Wi-Fi»**, con guion y mayúsculas, que es como lo escribe el sistema |
| **El código** | **Sí, y visible en la ficha**, no detrás de un botón |
| **El generador** | **Escrito aquí**, sin dependencia nueva |
| **Dónde se enseña** | La **ventana** y la **línea de comandos**. En el panel de la extensión no |
| **Qué guarda** | Nombre de la red, contraseña, seguridad, si es **oculta**, y notas |
| **La contraseña en texto** | **Oculta con su ojo**, como cualquier otra de la bóveda |

Tres campos nuevos —`ssid`, `seguridad`, `oculta`— y la clave **reutiliza `secreto`**, como
`nombreCompleto` se comparte entre la identidad y el dato personal. Con eso vienen hechos el ojo, el
copiado con borrado, el historial de claves anteriores y lo que se vacía al mandar a la papelera.

**El SSID va aparte del título** porque no son lo mismo: el título es cómo se llama la entrada para quien
la busca —«La oficina»— y el SSID es lo que el móvil tiene que encontrar, con sus mayúsculas exactas.

## Por qué el generador se escribe y no se trae

Es lo que este proyecto ya había decidido dos veces por escrito. `internal/codigos` dice que cuarenta
líneas de biblioteca estándar valen más que un paquete nuevo «en un programa que guarda contraseñas», y
`internal/iconos` que `golang.org/x/image/draw` «sería una dependencia nueva de verdad en el binario que
guarda las contraseñas de una empresa».

Aquí pesa además algo que no pasaba en esos dos casos: **el generador recibe la contraseña del wifi en
claro** para meterla en el dibujo. Una dependencia que la vea es una dependencia que podría llevársela.

El coste es real y hay que decirlo: `internal/qr` son unas 400 líneas, la pieza más grande escrita a mano
en este proyecto —CBOR fueron 142 y los códigos de un solo uso, 242—. Está **acotado a lo que pide una
red**: modo de bytes, nivel de corrección M y versiones 1 a 10. No es un codificador de QR completo y no
pretende serlo.

## Lo que el fichero del cliente enseñó, y no se podía adivinar

El cliente pegó su `wifi.csv` entero. Tres cosas cambiaron el diseño:

- **Es el sexto fichero de Dashlane**, no el quinto. `CLAUDE.md` decía otra cosa.
- **La columna de seguridad miente.** Sus dos redes dicen `unsecured` **y las dos tienen contraseña**.
  Copiando ese valor, el código saldría marcado como red abierta y **el móvil no se conectaría**: el fallo
  parecería del código y estaría en el dato. Por eso `wifi.Normalizar` decide por la clave y no por la
  etiqueta — la misma regla que el importador ya aplicaba al tipo de entrada.
- **La columna va mal escrita en origen**: `encription_type`, sin la «y».

Y dos del contenido: `name` y `note` vienen vacías —así que el título sale del SSID, como pasó con
`personalinfo.csv`— y **sus dos redes comparten la misma clave**, lo que obliga a que la huella de
duplicados sea el SSID y nunca el secreto. Con la huella mirando la clave, su segunda red no habría
entrado: se habría marcado como repetida. Es el fallo de las tarjetas otra vez.

## Alternativas descartadas

- **Una dependencia de QR** (`rsc.io/qr`, `go-qrcode`): días menos de trabajo y código muy leído. Se
  descartó por lo de arriba. Si algún día el generador propio da problemas, esta decisión se revisa con el
  coste delante y no por cansancio.
- **Mezclar las dos señales de seguridad**: creer lo que diga el fichero. Habría dado códigos que no
  funcionan en el caso exacto del cliente.
- **El código detrás de un botón.** Se le ofreció al cliente y eligió verlo. Lo que se pierde está dicho
  abajo.
- **Enseñarlo también en el panel de la extensión.** Obligaría a espejar el generador en TypeScript —otras
  400 líneas y su prueba cruzada— para algo que no hace falta: quien está en el navegador ya está conectado
  a una red.
- **Redes de empresa** (WPA2-Enterprise): no se guardan usuario ni certificado. **El formato del QR no las
  cubre**, así que guardarlos daría un código que no sirve.
- **Conectarse a la red desde Esfinge**: es tocar el wifi del sistema en tres plataformas y nadie lo pidió.

## Consecuencias

**El código está a la vista, y eso es una contraseña en pantalla.** Es la primera vez que Esfinge enseña un
secreto sin que nadie lo pida: en todo lo demás hay que pulsar un ojo. Lo eligió el cliente con el coste
delante, y por eso **la ficha lo dice con esas palabras** —«este dibujo es la contraseña: quien lo
fotografíe entra en la red igual que tú»— y no solo esta ADR. La contraseña en texto **sigue oculta con su
ojo**: el código lo lee una cámara apuntada a propósito y el texto lo lee de un vistazo quien pase por
detrás.

**Los colores del código no salen de la paleta.** Negro sobre blanco, escritos a mano, en la ventana y en
la terminal. Es lo único de la interfaz que se salta los tokens: con los colores del tema oscuro quedaría
gris sobre gris y no lo leería ninguna cámara. No se mide en `contraste_test.go` porque no es texto que
nadie vaya a leer.

**La barra de clases hubo que apretarla.** Con ocho pestañas y la ventana a 700 px pedía 451 px y caben
427 — medido, no estimado. Cuatro píxeles menos de relleno por lado en modo compacto son 64 px y vuelve a
caber con sitio. Si algún día no basta, **lo siguiente no es seguir apretando**: es dejar que la fila se
parta en dos.

**Y la red sí sale en la exportación en claro**, al contrario que la llave de acceso: su clave es una
contraseña como las demás, y poder llevársela a otro gestor es la mitad de lo que significa poder salir.

## Verificación

**Lo que cerró el asunto fue un móvil.** En esta máquina no hay `qrencode`, ni `zbarimg`, ni `qrcode` de
Python, y **el Chromium de las pruebas no trae `BarcodeDetector`** — se comprobó. Así que el primer
entregable no fue la clase sino un QR en pantalla: el cliente lo escaneó y su móvil le ofreció unirse a la
red. Hasta entonces no se escribió nada más, que es la regla de la casa aplicada a la pieza cara.

**Y la primera vez no lo leyó.** El código se dibujaba perfecto, el descodificador de las pruebas lo leía
sin problema y ningún móvil lo reconocía. Lo acotó un diagnóstico que saca los datos **probando las ocho
máscaras sin mirar la información de formato**: el texto salía entero, así que el zigzag, el entrelazado y
la corrección estaban bien, y el fallo quedó encerrado en quince bits — que se colocaban al revés, el más
significativo primero. **Un test que se mira al espejo pasa en verde con el fallo dentro**, y el de las
pruebas lo era: leía con la misma idea equivocada que el escritor.

De ahí las pruebas que quedan, en dos clases que conviene no confundir:

- **Las que no se miran al espejo**: las secuencias de formato y de versión se **recalculan con su BCH** y
  se comparan con las tablas escritas a mano; las dos tablas de tamaños se comprueban una contra otra; y la
  corrección de errores se verifica por la propiedad que la define — datos y corrección juntos son
  divisibles por el polinomio generador.
- **Las del espejo**: el descodificador del fichero de pruebas, el vector del código que leyó el móvil
  —congelado como los del formato `ESF1`, y si se pone rojo **no se regenera**— y la convención del orden
  de bits.

Lo demás, lo de siempre: las tres cruzadas con el booleano en sus dos valores, la papelera con una entrada
de cada clase, los duplicados mirando `Conflictos`, la vuelta entera de la exportación y las pruebas de la
interfaz con la barra medida a tres anchos. Todo mutado.

**Y una prueba cruzada nueva que no es de esta clase sino del formato entero**: al añadir `oculta` se
comprobó que **quitarlo del espejo no ponía roja ninguna cruzada**. La forma canónica ordena las claves, así
que un campo caído vuelve por `extra` con los mismos bytes, y la prueba de lo que se vacía solo se entera si
ese campo era un secreto. Ahora las dos listas de campos se comparan entre sí —Go saca la suya por
reflexión—, así que un campo nuevo que no esté en el espejo salta solo.

**Lo que no se ha comprobado:**

- Que el código funcione con **su red de verdad**. Lo escaneado fue una red inventada: el móvil la reconoció
  y ofreció unirse, que es lo que prueba el formato. Conectarse del todo solo se puede con una red que
  exista.
- **La importación de su `wifi.csv` real.** Las pruebas usan un fichero con la misma forma y datos
  inventados.
- Nada de esto se ha visto todavía **en un Mac**.
