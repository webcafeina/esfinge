# 0051 · Entregar una bóveda de proyecto

**Fecha:** 2026-10-02 · **Estado:** aceptada y escrita · **sin ver en un Mac**

## Contexto

Las bóvedas por proyecto ([ADR 0050](0050-varias-bovedas.md)) se pensaron para cuatro cosas, y la segunda que
el cliente marcó fue **entregarle al cliente lo suyo cuando el proyecto acaba**. Hasta ahora eso no existía en
Esfinge: lo más parecido era mandar una copia de **una entrada** a otra cuenta ([ADR 0043](0043-la-identidad-para-compartir.md)).

Y hay una segunda razón, que no es comercial: la ADR 0050 acepta un coste grande —**perder la bóveda personal
y su clave de recuperación es perder todos los proyectos**, porque se abren con ella—. Entregar es lo que
saca una bóveda de esa dependencia: desde que se entrega, tiene su propia contraseña y su propia clave de
recuperación y no depende de nada de aquí.

## Decisión

**`Desprender` hace una copia independiente**, sobre una copia en memoria, y cada paso quita algo que no
puede salir de aquí:

| | |
|---|---|
| **Identificador nuevo** | Es una copia, no la misma bóveda en dos sitios: con el mismo, la sincronización de quien la reciba la confundiría con la nuestra. El argumento ya estaba escrito en `envio.go` para una entrada suelta |
| **Fuera la ranura `boveda-principal`** | O la bóveda personal de quien entrega **seguiría abriendo la del cliente** |
| **Maestra nueva y recuperación nueva** | Las únicas dos ranuras que quedan. La clave de recuperación se enseña **una vez**, con la ceremonia de siempre |
| **Fuera la identidad** | Es la semilla con la que se firman los envíos compartidos: **regalarla es regalar la firma**. Es el paso que más fácil se olvida y más daño hace |
| **Fuera lo que era de nuestra cuenta** | Las copias que esperaban, las lápidas, lo que la sincronización recordaba y la lista de proyectos — que es la lista de clientes de quien entrega |

Lo que **sí** se lleva es lo único que importa: las entradas con sus secretos.

**La contraseña la elige quien entrega**, y la pantalla dice con todas las letras que **se le diga por otro
camino, no en el mismo correo que el fichero**. Es lo que separa entregar un fichero de entregarlo bien.

Y con ello, las otras dos cosas que se hacen al acabar un proyecto:

- **Archivar** lo saca de la lista del día a día **y borra su fichero de este equipo**, dejando el del
  servidor. Eso es lo que lo hace útil: «lo cerrado no está en memoria» pasa a ser también «no está aquí».
  Desarchivar lo devuelve a la lista y el fichero se baja aparte.
- **Borrar** se lo lleva de este equipo, del servidor y de la lista, y **pide la contraseña maestra**, como
  borrar la bóveda personal y por la misma razón: es lo único irreversible que hay aquí, y lo que se lleva
  son las contraseñas de un cliente entero.

**Las tres van detrás de un botón —«Al acabar…»— y no en la fila**, porque se usan una vez en la vida de un
proyecto y las tres son decisiones.

## Alternativas descartadas

**Mandar la bóveda por el sobre de compartir** (ADR 0043), que habría reutilizado todo el cifrado y la huella.
No cabe: `TAMANO_DE_ENVIO` son 64 KiB y una bóveda son cientos, y subir ese tope es tocar algo puesto contra
el abuso — cualquiera que sepa tu correo te llena el buzón. Lo que **sí** cabe por ahí es la contraseña de lo
entregado, y eso queda escrito como lo que se haría si algún día se quiere: una entrada con la maestra nueva
dentro, sin una línea de cripto nueva.

**Entregar sin contraseña propia, con la ranura de la personal dentro.** Sería más cómodo —el cliente no
tiene que teclear nada— y deja a quien entrega abriendo la bóveda del cliente **para siempre**. Es justo lo
que esta ADR viene a cortar.

**Dejar la identidad dentro.** Nadie la echaría de menos y no se notaría nunca: por eso es peligrosa. Quien
reciba la bóveda podría firmar envíos que el servidor y los destinatarios atribuirían a quien se la entregó.

**Que archivar solo marcara una casilla**, sin borrar el fichero. Era lo primero que se escribió. Pero
entonces archivar no hace nada que importe: lo que se quiere al acabar un proyecto es que **deje de estar en
el disco**, no que se vea menos.

**Que borrar no pidiera nada**, o que pidiera una segunda pulsación como la papelera. La segunda pulsación es
lo correcto para algo que va a la papelera y se puede sacar; esto no vuelve.

## Consecuencias

**Lo entregado no se sincroniza con nada.** Es una bóveda suelta: si aquí se cambia algo después, hay que
volver a entregarla. Es la misma regla que compartir una entrada (ADR 0043), y por eso la pantalla lo dice.

**Y quien entrega se queda con su copia.** Entregar no es desprenderse: el proyecto sigue en la lista y en el
servidor hasta que se archive o se borre, que son las otras dos acciones.

**La clave de recuperación de lo entregado se enseña una vez y va para quien la recibe.** Quien entrega no
debería guardarla: es la segunda puerta de una bóveda que ya no es suya.

**Y borrar un proyecto no se puede deshacer**, al contrario que borrar una entrada (ADR 0026). La papelera
protege de un clic despistado dentro de una bóveda; aquí lo que se borra es la bóveda entera, y guardarla
treinta días «por si acaso» sería guardar las contraseñas de un cliente que pidió que se borraran.

## Verificación

**Lo comprobado, mutando cada prueba:**

- **Lo que se lleva y lo que no**, pieza a pieza: las entradas con su secreto llegan; la identidad, las
  copias pendientes y el identificador **no**. *Mutado el paso de la identidad: la prueba dice que la copia se
  lleva la firma de quien la entrega.*
- **Abre con su contraseña nueva y con su clave de recuperación**, y **no** con la de quien la entregó ni con
  su bóveda personal.
- **La original no se toca**: sigue con su identidad y con sus copias esperando, que es lo que comprueba que
  la copia es honda y no plana — la misma trampa que ya costó algo al juntar dos bóvedas (ADR 0039).
- **No se abre el diálogo del sistema para un fichero que no se va a escribir**, con la bóveda cerrada o sin
  contraseña. Es la regla que costó `ExportarLlaves`, y el doble **cuenta las llamadas**.
- **Archivar se lleva el fichero de este equipo** y lo deja en la lista, marcado. *Mutado para que no lo
  borre: la prueba lo caza.* Y no se archiva la que se está usando.
- **Borrar pide la maestra.** *Mutada la comprobación: la prueba dice que ha borrado con una contraseña que no
  era.* Y con la mala no se lleva el fichero ni lo quita de la lista.
- En la ventana, el camino entero en los dos temas: entregar con su ceremonia, archivar y ver que pasa a
  «Dormido en este equipo», y borrar rechazando primero una contraseña que no es.

**Lo que no se ha comprobado:**

- **Nada de esto se ha visto en un Mac**, ni el diálogo de «Guardar como» de verdad, que aquí lo contesta un
  doble.
- **Nadie ha abierto una bóveda entregada en otro ordenador**, que es lo que de verdad cierra esto: aquí se
  abre con `boveda.Abrir` en la misma máquina.
- **Lo de mandar la contraseña por el sobre** está razonado y no está hecho.
