# ADR 0065 — La marca sale de la barra lateral; al pie queda la versión

**Fecha:** 2026-11-19 · **Estado:** aceptada · **Deroga la parte de la
[0021](0021-la-marca-en-la-interfaz.md)** que puso el lockup y la firma en la barra lateral ·
**Revisar si** el cliente echa de menos la marca al abrir la ventana

## Contexto

La [ADR 0021](0021-la-marca-en-la-interfaz.md) metió la marca dentro de la ventana en septiembre, y
no fue un capricho: la 0007 había decidido que Esfinge se vistiera de cada sistema, y sin esto el
producto no se reconocía por ninguna parte. Puso cuatro cosas, y **sólo dos están en discusión aquí**:
el **lockup** —la esfinge a trazo con el nombre— arriba de la barra lateral y la firma
**«Webcafeína ▍ versión»** al pie.

Lo que lo reabre es haberlo mirado. El 2026-11-19, viendo la galería de la fase de Windows, el
cliente lo dijo de una pieza con lo demás: *«También me parece feo (y a cambiar para todas los
sistemas), la cabecera (donde aparece Esfinge y el logo en trazo) y el pié ("Webcafeína | 2.0.3")»*.

Y hay una razón para resolverlo **ahora y no al final**: en Windows viene una barra de título propia
(ADR 0066), que ocupa exactamente esa esquina de arriba. Dibujarla sin saber qué va debajo es
dibujarla dos veces.

## Decisión

**Arriba no va nada; al pie va el número de versión y nada más.** En los cuatro sistemas.

Se le enseñaron cuatro tratamientos en capturas de la aplicación funcionando, en los dos temas, y
eligió la **D**:

| | Tratamiento | |
|---|---|---|
| A | Lo de hoy: lockup arriba, firma abajo | Es lo que llamó feo |
| B | Fuera los dos. **Es lo que hace Apolo** en sus tres sistemas | Se lleva la versión de la vista |
| C | Sólo el símbolo, pequeño y sin nombre | Queda suelto: un dibujo arriba sin nada que hacer |
| **D** | **Sin nada arriba; al pie, sólo la versión** | **Elegida** |

**La marca no desaparece del producto**, y eso es lo que hace que esto no contradiga a la 0021 más
que en el sitio: sigue en el icono de la ventana y del documento, en las portadas —bienvenida y
cuenta—, en **la esfinge grande y tenue del historial vacío** y en Ajustes. Lo que se quita es el
cromo: la marca deja de estar en la pantalla **mientras se trabaja**, que es justo lo que la propia
0021 decidió para las pantallas de trabajo —*«nunca en las pantallas de trabajo: ahí se trabaja»*— y
que la barra lateral se saltaba por estar siempre a la vista.

**Y la versión se queda** porque es lo único de aquel pie que servía para algo: es lo primero que se
pregunta cuando algo va raro, y en Ajustes hay que ir a buscarla. Va en `--apagado` y al tamaño
chico, que es la pareja que ya llevaba la firma: no entra ninguna medida de contraste nueva.

## Alternativas descartadas

**Dejarlo como estaba** (A). La descarta quien paga, que es quien mira esta ventana todos los días.

**Quitarlo todo** (B), que es lo que hace Apolo y lo que más limpio queda. Se descartó por la
versión: hoy está a la vista y mandarla a Ajustes es quitarle el sitio al dato que se pide cuando
algo falla. La D es la B conservando eso.

**Sólo el símbolo** (C). Es la solución de manual —marca sin ocupar el sitio de un título— y en la
captura se ve lo que tiene de malo: sin nombre al lado y sin nada que hacer, la esfinge se lee como
un adorno. Un símbolo solo funciona cuando ya se reconoce; el nuestro tiene dos meses.

**Decidirlo después de la estructura de Windows.** Era lo cómodo y habría costado dibujar dos veces
la esquina de arriba.

## Consecuencias

- **Queda derogada la parte de la 0021** que puso el lockup y la firma en la barra lateral. Lo demás
  de aquella ficha sigue entero: el oro de la marca como acento en vez del azul del sistema, la
  piedra sobre el oro, la esfinge del historial vacío y la regla de las pantallas de trabajo.
- **El componente `Firma` no se borra**: lo siguen usando la bienvenida, la pantalla de cuenta y la
  ficha de Ajustes, que son portadas y no cromo.
- **La barra lateral empieza por la navegación**, detrás del hueco de los semáforos en macOS. En
  Windows y en Linux ese hueco es de altura mínima, y cuando llegue la barra de título propia de
  Windows será ella quien ocupe la esquina.
- **Dos pruebas de interfaz pasan a comprobar lo contrario**: que arriba no hay marca y que al pie
  está la versión. Se afirma **que no han vuelto**, porque lo que se quita de una pantalla vuelve
  solo en cuanto alguien echa de menos la marca.
- Y una que se conserva tal cual, porque sigue siendo la que protege media suite: **la barra lateral
  tiene ocho botones**. Todo el fichero localiza las secciones con `.lateral + getByRole("button")`;
  el día que la versión sea pulsable, entra en ese localizador.

## Verificación

**Comprobado aquí**, con capturas sacadas después del cambio —no las de la galería— en los dos temas
y con el aspecto de macOS:

- Arriba empieza la navegación y al pie se lee el número, en `--apagado`.
- `la barra lateral lleva la versión al pie y nada más`: no hay `.marca` ni `.firma` dentro de
  `.lateral`, la palabra «Webcafeína» no aparece, la versión casa con `^\d+\.\d+\.\d+$` y siguen
  siendo ocho botones.
- `la versión va al fondo, debajo de Ajustes`, comparando posiciones pintadas: el orden en el DOM no
  dice dónde acaba una cosa que cuelga de un `margin-top: auto`.

**No comprobado, y hace falta su máquina**: cómo se lee el número a 11 px en una pantalla de verdad
—aquí no hay la tipografía de ninguno de los tres sistemas— y si la ventana, sin nada arriba, se
sigue reconociendo como Esfinge al abrirla. Eso último es lo que habría que mirar antes de dar por
buena la decisión: es el riesgo de la D, y es reversible.
