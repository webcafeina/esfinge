# ADR 0026 — La papelera guarda lo borrado, treinta días

**Fecha:** 2026-09-10 · **Estado:** aceptada · **Matiza la [0023](0023-la-boveda.md)** ·
**Revisar si** aparece la sincronización, que fija su propio plazo mínimo

## Contexto

El borrado de la bóveda es suave desde el primer día, y por una razón que no
tiene nada que ver con deshacer: **sin rastro, «borrada aquí» y «nunca existió
allí» son indistinguibles al sincronizar**. Lo que quedaba era una entrada
marcada, para siempre y sin forma de quitarla.

Esa misma mañana se arregló una mitad de esto —el borrado limpiaba la contraseña
pero **no** el texto de una nota segura ni el número de una tarjeta, que son el
secreto entero de esas clases— vaciando todo lo sensible al borrar. Quedaba el
resto: rastros que no se podían quitar, y un borrado del que **no se podía
volver**. Dos clics y la contraseña se había ido para siempre.

Y ahí está la pregunta de verdad, que no es técnica: **¿qué debe significar
«Papelera» en un gestor de contraseñas?** Hay dos respuestas coherentes y la
diferencia entre ellas es de producto, así que la tomó el cliente con las dos
consecuencias delante.

## Decisión

**Papelera de verdad, como la de Dashlane: lo borrado se guarda entero y se puede
restaurar durante treinta días.**

- Al borrar, la entrada se marca y **conserva su contenido**, contraseña incluida.
- La papelera se ve, y desde ahí se **restaura** o se **borra del todo**, entrada
  a entrada.
- Se **vacía a mano** con un botón, en dos pulsaciones.
- Y **se vacía sola a los treinta días** de haber borrado cada entrada.

Lo elegido revierte a propósito la limpieza de esa misma mañana, y el motivo del
cambio es el que había cambiado: **mientras no existía forma de vaciar la
papelera, guardar el secreto era dejarlo dentro del fichero para siempre**. Con
una papelera que se vacía, lo que se compra a cambio es que un clic mal dado deje
de perder una contraseña.

### El coste, dicho en voz alta

Durante esos treinta días **la contraseña borrada sigue dentro del fichero**. Es
el mismo trato que ya se hace con el historial de contraseñas anteriores, y está
en `docs/seguridad.md` junto a él. Contra quien no tiene la maestra no cambia
nada —va cifrado con todo lo demás—; contra quien la tiene, es una entrada más
que ya estaba.

### Por qué se vacía al abrir y no con un reloj

Una bóveda cerrada no ejecuta nada. Un reloj solo contaría mientras la aplicación
estuviera puesta, así que «treinta días» pasaría a significar «treinta días de
uso» y dependería de cuánto abra cada uno el programa. Al abrir el fichero se
sabe qué día es y se decide de una vez.

### Vaciar no pide la contraseña maestra

Borrar la bóveda entera sí la pide. Esto no, y la diferencia es la que hay entre
las dos cosas: aquí se tira lo que ya se tiró una vez, en dos pulsaciones y con
la cuenta delante. Pedir la maestra para cada limpieza la convertiría en un
trámite, que es la forma de que deje de proteger nada donde sí hace falta.

## Alternativas descartadas

- **Borrar es borrar, y solo se limpian los rastros** (lo de esa mañana,
  completado con un botón de vaciar). Es la opción más segura y la más pequeña de
  hacer, y se descartó por lo que le pasa a quien se equivoca: en un gestor de
  contraseñas, **perder una contraseña por un clic es peor que conservar treinta
  días una que se quiso tirar**. Se le ofreció al cliente con ese argumento
  delante y eligió la papelera.
- **Papelera sin plazo**, vaciada solo a mano. Se descartó porque quien no la
  mire nunca acaba con un almacén paralelo de contraseñas borradas creciendo sin
  que nadie lo sepa. Un plazo que se cumple solo es lo que hace que la papelera no
  se convierta en otra bóveda.
- **Restaurar sin el secreto** —papelera que devuelve el título y la cuenta pero
  no la contraseña—. Es lo que había de hecho, y no sirve para nada: quien
  restaura lo hace porque quiere la contraseña.
- **Un plazo configurable en Ajustes.** Un ajuste más para una pregunta que casi
  nadie quiere contestar. Treinta días es lo que usa todo el mundo, y es lo que
  tarda alguien en darse cuenta —normalmente al ir a entrar en el sitio—.
- **Confirmar con un diálogo del sistema.** Pararía la ventana entera para una
  pregunta que se contesta ahí mismo; se usa la segunda pulsación, como en el
  resto de la pantalla.

## Consecuencias

- **Borrar deja de ser irreversible**, y hay que decirlo donde se borra: la
  segunda pulsación dice «Sí, a la papelera» y después se explica que la entrada
  está ahí treinta días. Antes decía «Sí, borrar», que hacía pensar lo contrario.
- **La lista de la papelera viaja sin secretos**, como cualquier otra lista de la
  bóveda: estar borrada no hace a una entrada menos secreta, y quien quiera ver lo
  que había la restaura primero.
- **Lo borrado deja de bloquear su propia reimportación**, y esto es una trampa
  que trajo el cambio: con la entrada entera dentro de la papelera, el índice de
  duplicados del importador la reconocía y volver a pasar el CSV la daba por
  repetida. Se vería como «la borré, la reimporté y no ha vuelto» —con la única
  copia escondida en la papelera y a punto de caducar—. El índice ignora ahora lo
  que está en la papelera.
- **Abrir la bóveda puede escribirla**, cosa que antes no pasaba: si al abrir hay
  algo que caducó, se purga y se guarda. Solo entonces.
- **La línea de comandos no toca la papelera.** Sigue siendo de solo lectura por
  decisión de la [ADR 0023](0023-la-boveda.md), y vaciar es escribir.
- **Y cuando llegue la sincronización, el plazo de la papelera es también el plazo
  de las lápidas.** Treinta días es un mínimo razonable —es lo que usan los que
  sincronizan— pero conviene mirarlo entonces y no darlo por bueno: purgar una
  lápida antes de que el otro lado se entere resucita la entrada.

## Verificación

- Que **lo borrado vuelve entero**, con **una entrada de cada clase**: el secreto
  de una credencial es su contraseña, el de una nota su texto y el de una tarjeta
  su número, así que una papelera que solo devolviera bien las credenciales
  estaría rota para tres de las cuatro. Y se restaura **después de cerrar y volver
  a abrir**, que es el caso de verdad y de paso comprueba que se guardó entero.
- Que **vaciar se lo lleva del fichero**, comprobado volviendo a abrirlo: mirar la
  lista en memoria diría que está limpia aunque no se hubiera reescrito nada.
- Que **se vacía sola a los treinta días**, con el reloj parado cuarenta días
  atrás, y que en la misma pasada **no se lleva ni lo borrado hace un rato ni lo
  que no estaba borrado**.
- Que **una entrada en la papelera sin fecha no se toca**: solo puede venir de una
  versión que no la escribía, y tirar datos de alguien por no saber cuándo los
  borró es exactamente lo que no hay que hacer.
- Que **reimportar lo borrado lo devuelve**, que es la trampa de arriba.
- Y en la ventana, en los dos temas: que borrar dice adónde va, que la entrada
  desaparece de la lista y aparece en la papelera, que **restaurar devuelve la
  contraseña** —destapándola, no solo el título— y que vaciar pide una segunda
  pulsación y deja la papelera vacía y su botón fuera de la barra.

**Lo que no se ha comprobado:** que el plazo de treinta días se cumpla en una
bóveda de verdad, que exige dejar pasar treinta días. Lo que se ha comprobado es
el cálculo, con el reloj movido a mano.
