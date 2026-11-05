# 0058 · Que el servidor escriba, y que el permiso llegue

**Fecha:** 2026-11-05 · **Estado:** aceptada · **Continúa la [0057](0057-la-boveda-en-un-servidor.md) · Toca la [0043](0043-la-identidad-para-compartir.md), la [0052](0052-dar-acceso-a-un-proyecto.md) y la [0054](0054-los-agentes-por-mcp.md)**

## Contexto

La [0057](0057-la-boveda-en-un-servidor.md) está montada en el VPS desde el 2026-10-21 y funciona: la
bóveda se abre allí sin contraseña, las compartidas se sincronizan solas y `esfinge boveda ejecutar` le
pone los secretos en el entorno a un programa sin que aparezcan en ninguna pantalla.

Lo que falta lo preguntó el cliente nada más verlo funcionar: **un agente suyo o de Nacho que corra ahí va
a querer crear o cambiar una credencial.** Hoy no puede, por dos razones a la vez: la línea de comandos
**solo leía**, y el permiso que se le da al servidor es de **solo ver**.

Y no poder tiene un coste que es el que justifica todo esto: si el agente rota un testigo en Cloudflare y
no puede guardarlo, **se lo tiene que dar a una persona por el chat** — o sea que el secreto acaba en la
conversación de un modelo, que es exactamente lo que la 0054 y la 0057 existen para evitar. El agujero no
se tapa prohibiendo escribir: se abre.

## Decisión

### Escribir no se aprueba, y eso no contradice a la 0054

La [0054](0054-los-agentes-por-mcp.md) dice que la válvula **no cubre lo que no se deshace — cambiar una
contraseña o una semilla**. Parece que esto la contradice, y mirado de cerca cae del otro lado:

- **Dentro de la bóveda sí se deshace.** `CambiarSecreto` empuja el anterior al historial de esa entrada
  antes de escribir el nuevo, que es lo que deja volver atrás si el sitio no acepta el nuevo.
- **Lo irreversible pasó fuera.** Si el agente ya rotó el testigo en el proveedor, el viejo está muerto se
  guarde o no. Guardarlo no destruye nada: lo que destruye es no guardarlo.
- Así que **en una máquina sin pantalla, no guardarlo es más peligroso que guardarlo.**

**En la ventana no cambia nada**: allí hay alguien delante y el sí no cuesta nada. Lo que esta ficha dice es
que el mismo razonamiento, en una máquina sin pantalla, se lee al revés.

### Lo que la línea de comandos gana, y lo que no

`esfinge boveda crear` y `esfinge boveda editar`. Y **no** `borrar`: en la ventana pide un sí y aquí no hay
quién lo dé, y un servidor no tiene ninguna razón para borrar la credencial de un cliente. Es una frontera
barata de sostener, que es lo que la hace frontera.

**El secreto no es nunca un argumento**, que es la misma regla que `OrigenClave`: lo que se escribe en la
línea queda en el historial del shell y lo lee cualquiera con un `ps`. Cuatro orígenes y ninguno es la
línea de órdenes: `--secreto-stdin`, `--secreto-env`, `--secreto-fichero` y **`--generar`**, que es el mejor
caso de todos porque la contraseña **no la ve nadie**, ni quien la pide. De una generada se dice **cómo
es** —largo y alfabeto—, nunca cuál: sin eso, quien la pide no puede saber si cumple lo que exige el sitio
y lo único que le queda es pedir que se la enseñen.

`editar` **cambia solo lo que se nombre**, y eso se decide por si la bandera se usó y no por su valor:
`--notas ""` es «bórrame las notas» y no decir `--notas` es «déjalas». Con el valor vacío las dos cosas se
ven igual, y entonces cambiarle el usuario a una credencial le borraría de paso las notas, el sitio y la
carpeta.

**Queda apuntado**, con `--quien` —que **no autentica nada** y la ayuda lo dice: es lo que dice de sí mismo
quien escribe el guion, igual que el nombre de un cliente MCP— y con `Como` = `llave-del-equipo`, que aquí
significa lo que dice su comentario: **no había a quién preguntar**.

### La huella del servidor, obligatoria y una vez

**Esto era un agujero y no una mejora.** El selector de la 0057 llamaba a «dar acceso» con el correo del
servidor **sin enseñar ninguna huella**, así que sellaba la clave de la bóveda hacia las llaves públicas que
el servidor de cuentas diera para esa dirección — y eso es justo lo que la [0043](0043-la-identidad-para-compartir.md)
dice que es lo único que protege en el primer envío. Con una persona la pantalla te obliga a mirarla; el
interruptor se lo saltaba. La primera vez se comparó a mano porque el camino manual lo exigía.

Se confirma **una vez por servidor** y en Ajustes → Cuenta, no una vez por proyecto: la identidad es la
misma para todas sus bóvedas, y preguntarla en cada una es la forma segura de que se pulse sin mirar.

**Pero se compara en cada proyecto.** Confirmar una vez es de la persona; comparar es de cada vez. Si la
comparación no se repitiera al sellar, el servidor de cuentas podría publicar otras llaves después y **solo
el primer proyecto estaría protegido**.

**Y lo que se guarda es cuál es la huella, no un sí/no.** Es exactamente la lección de la ranura de Touch ID
(ADR 0044, donde «una vez» tuvo que pasar a ser «una vez por bóveda»): con un booleano, cambiar el correo a
otro servidor dejaría la marca puesta y el interruptor sellaría hacia unas llaves que nadie ha visto en su
vida **sin que nadie se lo mencionara**. Go la borra en cuanto el correo cambia, y la ventana no la manda.

**Y hace falta poder leerla en el servidor**, que es la mitad que se escribió sin pensar: `esfinge boveda
servidor huella`. Sin ella la pantalla manda a comparar con algo que allí no se puede ver — la primera vez
se sacó con un programa de un solo uso. Es la trampa de `ExportarLlaves` otra vez: **lo que hace falta para
decidir se mira antes de pedirle a alguien que decida**.

### El permiso se cambia en el sitio, y el servidor lo dice al bajar

El segundo interruptor —«y puede escribir en ella», debajo del primero y solo si está puesto— llama a
`CambiarPermiso`, que mueve a esa cuenta entre «ver» y «editar» **sin tocar nada más**: la tabla del
servidor es un `upsert` por `(ref, cuenta)` y `PonerTitular` actualiza por identificador. **Ni rota la
clave, ni añade ranura, ni obliga a nadie a volver a bajar nada.**

Los dos caminos que parecen equivalentes y no lo son, escritos para que nadie los reintente:

- **Volver a llamar a `DarAcceso`** acuña un titular nuevo, así que deja una **ranura huérfana** y el
  titular guardado deja de ser el que conoce el servidor — y entonces «quitar» apunta al que no es.
- **Quitar y volver a dar** es correcto y caro: quitar hace que esa persona **borre su copia**
  ([0053](0053-quitar-el-acceso-borra-la-copia.md)), así que subirle el permiso le costaría volver a
  bajarse la bóveda entera. En un servidor, en medio de un despliegue, eso es el despliegue caído.

**Y aquí está lo que esta ficha de verdad añade, porque el plan lo daba por hecho y era falso.** El permiso
que guarda quien recibe vive **en su propia bóveda**, y solo se escribía al aceptar el sobre del buzón. El
servidor no lo decía al bajar. Así que cambiárselo a alguien **no le llegaba nunca**:

- **Al subirlo**, el servidor empezaba a aceptarle subidas y su Esfinge seguía creyendo que solo podía ver,
  así que **se negaba a escribir ella misma**. El interruptor habría dicho una cosa y la máquina otra, que
  es justo lo que el comentario de esa tarjeta tiene escrito que no se hace.
- **Al bajarlo** es peor de leer: cree que puede editar, escribe, y se encuentra un 403 al subir — con lo
  escrito quedándose en su equipo y en ninguna otra parte.

Así que **el servidor dice el permiso al bajar**, en la cabecera `Esfinge-Permiso`, y la sincronización lo
apunta. Va en esa llamada y no en un endpoint propio porque **ya se hace cada minuto**, y un camino más para
saber lo mismo es un camino más que puede fallar solo.

**Y va también en el 304**, que no es simetría: el 304 es el caso normal —se baja cada minuto y casi nunca
ha cambiado nada—, y además es el que importa en la extensión, que se acuerda del permiso en
`storage.session` y lo pierde al cerrar el navegador.

**Quién lo escribe, y quién no.** La ventana lo guarda en su lista de compartidas y pone al día lo que tiene
en la mano, para que a quien esté dentro de la bóveda le cambie sin salir y volver a entrar. La línea de
comandos lo guarda igual, y **vuelve a leer la fila después de la pasada**: devolviendo la de antes, la
primera orden tras un cambio de permiso seguiría negándose y la segunda funcionaría, que es un «a la segunda
va» imposible de entender desde fuera. **La extensión no lo escribe**: esa lista la gestiona la ventana —es
la línea que `Boveda.compartidas()` tiene escrita, y por eso `anotarUso` tampoco está allí—, así que se
acuerda de lo que el servidor acaba de decir y lo prefiere a lo de su lista.

## Alternativas descartadas

**Mandar un sobre nuevo al buzón al cambiar el permiso.** Es lo más barato y no toca el servidor. Se
descartó porque quien recibe **tiene que aceptarlo**: en el VPS eso es `esfinge boveda servidor recibir`, un
paso a mano que nadie va a correr, así que el interruptor parecería no hacer nada hasta que alguien entrara
allí por SSH.

**Y que la pasada de sincronización vaciara el buzón** para que ese sobre entrara solo. Habría funcionado
sin tocar el servidor, y el coste era de alcance: aceptar accesos pasaría a ser automático en cada pasada y
no solo con `recibir`, lo que relaja la regla de que **lo del buzón no entra solo** para cualquiera que use
la línea de comandos, no solo para un servidor.

**Inventar una aprobación a distancia** —el agente deja una propuesta y el cliente la aprueba desde el Mac—.
Es mecanismo nuevo: dónde espera, cuánto vive, qué pasa si nadie mira. Y deja al agente parado. Lo descartó
el cliente.

**Un freno de escrituras en la línea de comandos.** El canal MCP limita a seis por minuto y ese contador
vive en el `Servidor`, que es un proceso de larga vida; aquí **cada invocación es un proceso nuevo**, así que
un contador en memoria no frenaría nada —es la trampa del contador que vivía en la conexión—. En disco no
compra nada real: quien está en esa máquina ya tiene la llave y puede leerlo todo.

**Exigir `--compartida`.** Se valoró que los dos verbos solo escribieran en bóvedas compartidas, para que la
personal del equipo quedara fuera. Lo eligió el cliente al revés, con el coste delante: escriben en
**cualquier bóveda que la línea de comandos pueda abrir**.

## Consecuencias

- **La línea de comandos deja de ser solo de lectura**, que era una propiedad que se podía decir en una
  frase. Ahora lo que se dice en una frase es que **no borra**.
- **Con una ventana de Esfinge abierta en el mismo equipo, su lista se queda con lo de antes** hasta que algo
  la refresque: la línea de comandos no puede mandar `EventoBovedaCambiada`. Lo guardado está guardado y sube
  igual; lo que no se entera es la pantalla. Está en la ayuda de los dos verbos.
- **Quien tenga el permiso de escribir en una bóveda de proyecto puede dar acceso a otras cuentas**, que con
  solo ver no puede. Lo dice la pantalla del interruptor.
- **El Worker cambia**, así que esto no funciona entero hasta desplegarlo. Mientras no se despliegue, el
  permiso sigue sin propagarse y el segundo interruptor cambia el permiso en el servidor sin que el equipo de
  quien lo recibe se entere.
- **Y una cosa que no hace y parece que sí**: el registro sigue sin enseñarse en la línea de comandos. Los
  apuntes de una compartida llegan a la ventana del dueño por la sincronización, que es donde se leen.

## Verificación

Lo que se puede decir al acabar, y cada protección mutada:

| Qué protege | Cómo |
|---|---|
| Que el secreto no viaja como argumento | Se le piden a cobra las banderas que **no** tienen que existir. Mutada **añadiendo** un `--secreto` al lado, que es lo que una prueba de comportamiento no vería |
| Que los tres orígenes de fuera guardan lo mismo | Los tres, y recortando el salto que deja cualquier `echo`. Mutada quitando el recorte |
| Que dos orígenes a la vez se niegan, y ninguno también | Y que **no queda nada guardado** en los tres casos |
| Que `--generar` no enseña la contraseña | Se busca **en los bytes** de lo que Esfinge escribe, y se exige que sí diga el largo y el alfabeto. Mutada quitando lo que dice cómo es |
| Que `editar` cambia solo lo que se nombra | Con una entrada con todo puesto, y comprobando que `--notas ""` **sí** borra. Mutada mirando el valor en vez de `Changed` |
| Que el anterior cae al historial | Mutada escribiendo `e.Secreto` directo en vez de `CambiarSecreto` |
| Que queda apuntado quién y que nadie lo aprobó | `Quien`, `Que`, `Sobre`, `Como` y el secreto **buscado en el fichero**. Mutadas las dos |
| Que el tipo se comprueba y la llave de acceso se niega aparte | Tabla con los seis que valen y los tres que no, cada uno con la razón que tiene que dar |
| Que en una compartida de solo ver no se escribe | Los **dos** verbos, montando la compartida **sin servidor**, y comprobando que no entra nada. Y con `editar` sí escribe, que es lo que impide que pase con la escritura rota del todo |
| Que `boveda` acepta `-q` | En el comando de verdad y en sus hijos: lo que falló fue el cableado |
| Que sin huella comparada no se da acceso | Y que se niega **antes de salir a la red**. Mutada quitando la comprobación |
| Que cambiar el correo se lleva la huella confirmada | Mutada quitando el borrado, que es lo que un booleano no habría podido hacer |
| Que la ventana no manda la huella | Ni la borra con un guardado a medias ni la puede escribir. Mutada quitando la conservación |
| Que la huella se apunta contra el correo que había | Mutada quitando la comparación |
| Que el servidor dice el permiso al bajar, **incluido el 304** | Prueba contra el Worker de verdad, con cuerpo y sin él. Mutada quitando el permiso, quitando la cabecera, y **poniéndola solo en el 200** |
| Que el permiso llega hasta el otro equipo y se usa | **El paseo de dos equipos**, paso 9: la ventana lo sube, el servidor **se entera solo** —sin volver a `recibir`— y escribe, y lo escrito llega a la ventana. Mutadas las cuatro piezas de la cadena |

**Lo que no se ha comprobado**, dicho con todas las letras:

- **Nada de esto se ha visto en una pantalla de verdad**, con ojos. La huella en Ajustes y los dos
  interruptores están escritos, con `tsc` en verde y con la prueba de interfaz que sigue.
- **La puerta nueva sí está vigilada**, y de paso quedó claro que el selector **ya tenía** prueba de
  interfaz: la de la 0057 afirmaba «y ahora sí sale» con el correo puesto, que es exactamente lo que la 0058
  deja de ser verdad. Esa prueba se puso del revés —con la huella sin comparar el interruptor **no está**, y
  en su sitio hay una frase que dice qué falta—, así que la aserción que antes guardaba el agujero ahora
  guarda la puerta. Lo que no cubre es confirmar la huella ni pulsar el interruptor: las dos cosas piden una
  cuenta, y eso vive en el paseo.
- **El segundo interruptor no tiene prueba de interfaz**: para que salga hace falta un acceso dado de
  verdad, o sea una cuenta, y eso es `cuentas.spec.ts` con sus dos ventanas. Está en `deuda.md`.
- **El Worker no está desplegado** con este cambio.
- **Que un agente de verdad lo use bien.** Eso lo dirá Serafín usándolo, que es lo que ha encontrado todo lo
  demás de este mes.
