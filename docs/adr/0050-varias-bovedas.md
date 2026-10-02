# 0050 · Varias bóvedas: una por proyecto, y una abierta a la vez

**Fecha:** 2026-10-02 · **Estado:** aceptada · **las seis entregas hechas** · **sin desplegar** y sin ver en un Mac

## Contexto

El cliente lo pidió así: *«La última gran funcionalidad será poder crear bóvedas para proyectos (con un
nuevo apartado "Proyectos", por ejemplo). Donde cada proyecto/cliente tendrá su bóveda y yo podré apuntar
ahí los mismos elementos que tengo en mi propia bóveda.»* Y añadió lo que convierte esto en una decisión y
no en una tarea: *«Habría que ver cómo casa esto con la extensión y demás entorno de todo Esfinge.»*

Resuelve cuatro cosas, y marcó las cuatro: **organizar** lo de cada cliente, **entregárselo** al acabar,
**que una filtración no se lo lleve todo** y **trabajar con otra persona**.

Es lo más grande que se ha planteado en este proyecto porque toca las cinco piezas a la vez —el núcleo, la
ventana, la línea de comandos, la extensión y el servidor— y porque **casi todo en Esfinge da por hecho que
hay una bóveda**: el reloj del bloqueo, el candado de la barra lateral, el goteo de iconos, el portapapeles,
la ranura del llavero del sistema, la ceremonia de recuperación, el socket del navegador y el Durable Object
del servidor, que lleva `ajustes["idBoveda"]` y **rechaza con 409 cualquier otra bóveda**.

## Decisión

Lo eligió el cliente por preguntas con opciones, en tres rondas, y no se cambia sin preguntar:

| | |
|---|---|
| **Qué es una bóveda de proyecto** | **Fichero propio**, y se abre con **su misma contraseña maestra** |
| **Cuántas abiertas** | **Una a la vez.** Abrir un proyecto cierra el anterior |
| **Su bóveda** | Sigue siendo **la principal**; los proyectos son un apartado aparte |
| **Dónde se elige** | En el apartado **«Proyectos»**, una fila más en la barra lateral |
| **Sincronizar** | **Sí, como la suya** |
| **La extensión** | Trabaja con **la activa**, y la del navegador se elige en su panel |
| **Touch ID** | **En las que quiera** |
| **Mover entradas entre bóvedas** | **Sí, y es importante** |
| **Clave de recuperación propia** | **No**: basta la de la cuenta |
| **Al acabar un proyecto** | **Entregar**, **archivar** y **borrar**: las tres |
| **Compartir** | **Una copia**, como ya hace Esfinge con una entrada |

Y una consecuencia que aceptó con el coste delante: **la misma maestra las abre todas**, así que esto **no
protege de que le roben la contraseña**. Lo que separa es el contenido —una bóveda se entrega entera— y que
lo que está cerrado no está en memoria.

### Lo técnico: la ranura `boveda-principal`

Un proyecto es una bóveda igual que cualquier otra y lo único que la distingue es **con qué se abre**: una
ranura más, de las que [la 0023](0023-la-boveda.md) dejó como «lista abierta a propósito», envuelta con **la
clave de bóveda de la personal** y no con la contraseña maestra.

Esa distinción es la decisión entera, porque la opción evidente pierde datos. Está abajo.

### Y una sola abierta a la vez

Todo el acceso a la bóveda en Go pasa por un accesor, `App.boveda()`, con 51 llamadas. Conmutar es cambiar
qué devuelve, así que **su firma no cambia** y no se tocan ni esas 51, ni los 40 métodos de `*App`, ni las
~30 funciones del puente. Con eso, las ocho piezas que se habrían multiplicado se quedan como están, cada
una con un porqué nuevo escrito al lado.

## Alternativas descartadas

**Envolver la clave del proyecto con la contraseña maestra.** Es lo que la decisión del cliente parece pedir
literalmente, y es la que hay que no volver a proponer:

- el día que la maestra cambie, hay que reenvolver N ficheros en una operación que puede fallar a medias y
  que exige tenerlos todos presentes;
- y el día que la maestra **se recupere** —que es para lo que existe la clave de recuperación— los proyectos
  siguen envueltos con la maestra vieja, **que ya nadie sabe**. Se perderían todos, y el cliente se
  enteraría justo el peor día.

Con la clave de bóveda no pasa ninguna de las dos, porque esa clave **no cambia nunca**: cambiar la maestra
reenvuelve su sobre y la clave sigue siendo la misma. Hay prueba de las dos cosas.

**Guardar la clave de cada proyecto en la bóveda personal**, «por redundancia». Estaba en el plan y se quitó
al implementarlo: **no sirve para nada**. Lo que abre un proyecto es la ranura de su propio fichero, y esa
ranura se sube, así que un equipo nuevo se baja el fichero y lo abre sin consultar ninguna lista. Guardarla
sería amontonar las llaves de todos los proyectos en un sitio más sin ganar un caso de uso.

**Una clave de proyectos intermedia** —32 bytes dentro de la personal que envuelven los proyectos—. Daba lo
mismo que usar la clave de bóveda directamente, con una indirección más y un secreto más que mantener.

**Varias bóvedas abiertas a la vez.** Es lo que multiplica las ocho piezas de arriba, y con ellas sus
razonamientos: el espaciado del goteo de iconos existe para no dibujar «un pico reconocible» en la red, y N
goteos lo anulan. Lo que cuesta es cambiar de proyecto para mirar algo de otro cliente, que con Touch ID es
un gesto.

**Una contraseña maestra por proyecto.** Lo descartó el cliente: con muchos proyectos son muchas
contraseñas, y la que de verdad hay que recordar es una.

**Un índice de títulos fuera de la bóveda** para poder buscar en todos los proyectos a la vez. Es
exactamente lo que [la 0024](0024-iconos-de-los-sitios.md) rechazó para los iconos: una lista de
títulos dice casi tanto como la bóveda. Buscar en todos cabe más adelante como una acción explícita que los
abre por turnos, no como una caja de búsqueda viva.

**Tratar la sección `proyectos` como opaca en el espejo de TypeScript**, dejándola en `extra`. Habría
funcionado mientras la extensión no la escribiera, y el día que divergiera serían dos bóvedas pasándose la
una a la otra sin fin, que es el fallo que [la 0038](0038-sincronizar-la-boveda.md) vino a cerrar. La fusión
está espejada y la cruzada al azar la vigila.

## Consecuencias

**Una entrada en el llavero del sistema, no una por bóveda.** La ranura de Touch ID va solo en la personal y
un proyecto se abre *a través* de ella, así que tras una actualización sale **un** diálogo del sistema. El
cliente había aceptado pagar N sabiendo lo que costaba; este diseño no se lo cobra.

**Una sola ceremonia de clave de recuperación.** Un proyecto no tiene la suya, así que la pantalla que lo
crea **tiene que decirlo**: quien ha creado una bóveda antes espera esa ceremonia, y su ausencia sin
explicación parece un olvido.

**Perder la bóveda personal y su clave de recuperación es perder todos los proyectos**, aunque sus ficheros
sobrevivan. Va en `docs/seguridad.md` con esas palabras. Su mitigación real es entregar
[la 0051](0051-entregar-una-boveda.md), que le pone a la copia maestra y recuperación propias y la deja
sin depender de nada.

**Cambiar la maestra no toca ningún proyecto**, que es lo que compra la decisión de arriba y conviene no
perder al refactorizar.

**La lista de proyectos vive dentro del cuerpo cifrado de la personal**, por lo mismo que los sitios
excluidos: la lista de proyectos de Webcafeína es la lista de sus clientes. Dos efectos que se notan: se
sincroniza gratis, sin una ruta nueva en el servidor, y **con la personal cerrada la pantalla no enseña
nombres de clientes**.

**Y los ficheros se llaman por su referencia, no por el nombre del proyecto.** Un listado de la carpeta deja
de ser una lista de clientes en claro; renombrar un proyecto no puede renombrar un fichero, porque los
satélites (`.anterior`, `.iconos`, `.base`, `.sincro`) cuelgan de su nombre; y un nombre libre lleva `/` y
acentos a un sistema de ficheros ajeno. El coste es que la carpeta es ilegible para una persona, y lo
compensa la línea de comandos.

**La extensión solo ve la bóveda activa**, así que con el proyecto de un cliente abierto no rellena las
cuentas personales. Está aceptado, y la válvula de escape —caer a la personal cuando la activa no tiene nada
para ese dominio— es un `if` que **no se hace ahora**: contradice «solo la activa» y va en `deuda.md` con su
línea exacta, para que se pida si molesta.

**Un proyecto dormido no se sincroniza hasta que se abre.** Con proyectos que se tocan cada meses eso es más
ventaja que coste, pero hay que decirlo: un cambio hecho en un Mac no llega al otro hasta que el otro abre
ese proyecto.

**Y la que hay que vigilar: la clave de la personal se queda en memoria** mientras hay un proyecto abierto,
para que conmutar entre proyectos no pida la maestra cada vez. Es una relajación deliberada —esa clave
estaba en memoria hace un segundo y el reloj de inactividad sigue valiendo— y se borra con `cripto.Borrar`
cuando el vigilante cierra, con su prueba.

## Verificación

**Lo comprobado de verdad, y mutando cada prueba para ver que caza su fallo:**

- Un proyecto se abre con la personal y **su maestra no lo abre**, porque ahí no hay ranura maestra; la
  clave de otra bóveda personal tampoco, con un error que no dice «fichero roto».
- **Cambiar la maestra no reescribe ni un byte** del fichero del proyecto, y después sigue abriendo.
- **Recuperar la personal abre los proyectos**, y también tras poner una maestra nueva: el camino entero de
  «he perdido la contraseña».
- **La ranura se sube**, y lo que se sube lo abre el otro equipo de verdad. *Mutada metiéndola en
  `ranurasLocales`, que es media línea y parece lo prudente: la prueba se pone roja.* Sin ella el fallo sería
  mudo y tardío —se descubre en el segundo Mac, con una bóveda que no abre nadie—.
- **La fusión conserva y funde la lista de proyectos.** *Mutada quitando la línea de `fundirContenido`: la
  lista se queda en cero.* Es la familia de fallos que ese fichero ya tenía, porque arma el contenido campo a
  campo y una sección que nadie añada ahí se pierde en silencio.
- **Los dos lenguajes funden igual**, con 400 casos al azar que ahora llevan proyectos con las mismas
  referencias en los dos lados. *Mutada la regla de `usado` en el espejo: la cruzada canta la diferencia.*
- El sobre del proyecto va con el perfil barato, mirando **los parámetros que lleva dentro** y no lo que
  tarda.
- **Bloquear se lleva la clave de la bóveda personal**, no solo la bóveda abierta. *Mutada quitando la
  línea: tras bloquear se sigue pudiendo abrir cualquier proyecto.*
- **Con un proyecto abierto no se ofrece ni se activa Touch ID para él.** *La primera versión de esa prueba
  pasaba con el fallo dentro —`Sugerir` ya era falso por otro motivo— y lo dijo mutar, no leer: se rehízo con
  el escenario que distingue, la personal sin desbloqueo y con «ahora no» contestado.*
- **Mover una entrada no la pierde.** *Mutado el orden —borrar del origen antes de guardar en el destino— y
  la prueba dice «la entrada se ha perdido: quedan 0».* Es la razón de que el orden sea el que es.
- Y en la ventana, el camino entero en los dos temas: crear un proyecto, entrar, que lo de la bóveda personal
  **no se vea desde dentro**, y llevar una entrada de una a otra **con su contraseña**.
- **La migración del Durable Object, sobre una cuenta que ya tenía su bóveda dentro**: los mismos bytes, el
  mismo ETag —quien estaba al día no se baja nada— y las dos versiones intactas. *Mutada para que copie de
  menos: salta con «Migración incompleta: 2/2 → 2/0» y la transacción se deshace entera.* Y correrla dos
  veces no duplica nada.
- **La extensión cambia de bóveda y rellena lo de esa bóveda**, con la extensión cargada de verdad en un
  Chromium: desde la personal no sale la contraseña del proyecto, desde el proyecto no sale la de la personal,
  y al volver está la de siempre. *Un selector que cambia de rótulo y sigue rellenando lo mismo sería peor que
  no tenerlo.* Y bloquear se lleva también la clave de la personal, como en la aplicación.
- **Un proyecto llega al otro equipo de verdad**, contra el servidor levantado en local: se crea aquí, se
  baja allí, **se abre con la bóveda personal de allí sin preguntar nada** —la ranura viaja dentro del
  fichero— y lo que se guarda allí vuelve. *Mutada la ruta para que el proyecto suba a la de la personal: la
  prueba cae.*

**Lo que no se ha comprobado**, y es casi todo lo que se ve:

- **Nada de esto se ha visto en un Mac.** Aquí se ha mirado en capturas del navegador, en los dos temas, que
  es lo que hay. La ventana de verdad —tipografía del sistema, controles nativos, el material translúcido
  detrás— solo se ve ahí.
- **El servidor no está desplegado.** La migración está escrita y probada contra `workerd`, pero **se ejecuta
  una vez y sobre la bóveda del cliente**: antes de desplegarla hay que guardar un
  `GET /v1/cuenta/exportacion` de la cuenta real. Eso no lo puede hacer esta máquina.
- **Cuántos diálogos de Touch ID salen tras una actualización con varios proyectos.** El diseño existe para
  que sea uno; que sea uno lo dice el Mac.
- **Entregar, archivar y borrar están hechas** ([ADR 0051](0051-entregar-una-boveda.md)) y probadas aquí, pero
  **nadie ha abierto una bóveda entregada en otro ordenador**, que es lo que de verdad lo cierra.
