# ADR 0069 · Esfinge también en inglés

Fecha: 2026-11-21 · Estado: **aceptada, sin empezar** · Lo pidió el cliente el 2026-11-21, junto con
el selector de aspecto

## Contexto

Esfinge está **en español y sólo en español**, y es una decisión tomada: la cabecera de
`CLAUDE.md` la lleva desde el principio —«Español, y todas las frases empiezan en mayúscula, aunque
sean de una palabra»— y la vigila `internal/cripto/textos_test.go`.

El cliente lo pidió con estas palabras: *«también debe poder traducirse a inglés (esto también para
todos los sistemas)»*, en la misma frase en que pidió el selector de tema.

**No es traducir una pantalla.** El texto que ve alguien está repartido en cinco sitios, y se contó
antes de proponer nada:

| Dónde | Cuánto | Qué es |
|---|---|---|
| La ventana (`frontend/src`) | ~136 textos visibles | Lo que se lee en las pantallas |
| Go (`internal/`) | ~324 errores | **Lo que la ventana enseña cuando algo falla** |
| La extensión (`navegador/src`) | ~239 | El panel, la tarjeta de guardar y el aviso de la página |
| La línea de comandos (`internal/cli`) | ~270 | Lo que se ve en el VPS y en los scripts |
| Los esquemas de MCP (`internal/agente`) | ~77 | **Lo que lee el modelo**, no una persona |

Y el recuento de la ventana es **el suelo**: cuenta `>Texto<` y deja fuera los rótulos de atributo
—`aria-label`, `title`, los textos de ayuda— que aquí son muchos, porque media interfaz explica lo
que hace antes de hacerlo.

## Decisión

**Entran los cinco**, el idioma **sale del sistema y se puede cambiar**, y se hace **después de
GNOME y de KDE**. Lo eligió el cliente con el recuento delante.

De lo de los cinco sale la primera consecuencia que conviene no volver a discutir: **la ventana y
los errores de Go son una sola entrega y no dos**. Una ventana en inglés que falla en español está a
medias de la forma más visible que hay, porque el error es justo el momento en que alguien lee con
atención.

**El idioma del sistema manda, con un selector al lado del de aspecto.** Es lo que hace cualquier
aplicación del sistema, y el selector es lo que permite volver al español a quien tenga el equipo en
inglés — que es el caso real de este cliente.

**Y va después de nativizar**, no antes: GNOME y KDE están planificados, son más cortos y tocan los
mismos ficheros que acaba de tocar macOS. Metiendo el inglés en medio, las dos hojas que quedan se
escribirían sobre un marcado que está cambiando de forma.

## Alternativas descartadas

**Sólo la ventana.** Es la mitad barata y deja los errores en español, la extensión en español y el
VPS en español. Se ofreció y el cliente quiso los cinco.

**Sólo lo que diga el sistema, sin selector.** Menos pantalla y menos que guardar, y deja sin salida
a quien tenga el equipo en inglés y quiera leer en español.

**Hacerlo ahora, antes de GNOME.** Para en seco la nativización con dos escritorios viéndose los dos
como GNOME, que es justo lo que la ADR 0020 dejó dicho que había que arreglar.

## Consecuencias

**Lo que cuesta, dicho antes de empezar:**

- **La regla de la mayúscula deja de ser una regla del proyecto y pasa a ser una regla del
  español.** `textos_test.go` la vigila hoy sobre los errores de Go; con dos idiomas tiene que
  vigilarla **por idioma**, o se pondrá roja con el inglés dentro.
- **Los esquemas de MCP son el caso delicado**, y no por el tamaño: ahí **los textos son el
  producto**. Una palabra ambigua le quita capacidades al modelo, que es exactamente lo que pasó con
  «cerrada» —que en Esfinge quiere decir «bloqueada»— y le costó a un agente dar por imposible una
  bóveda que sí podía abrir. Traducir eso no es traducir: es volver a escribirlo en inglés y
  **comprobarlo con un agente de verdad**, como se comprobó aquello.
- **La línea de comandos tiene lectores que no son personas.** Un script que mire lo que imprime
  `esfinge` se rompe al cambiarle el idioma. Hay que mirar qué se promete ahí antes de tocarlo.
- **Los correos del servidor no entran** y conviene decirlo: los manda el Worker, no la aplicación,
  y el idioma de quien recibe no lo sabe nadie en ese momento.
- **Y una ficha de tienda por idioma.** Firefox recibe la suya con cada etiqueta; la de Chrome **se
  pega a mano** y ya ha costado tres veces.

## Verificación

**Nada de esto se ha hecho todavía.** Lo que sí se hizo el 2026-11-21 es el recuento de la tabla de
arriba, con `grep` sobre los cinco sitios, y es el suelo y no el total.

Lo que habrá que comprobar cuando se haga, escrito ahora para que no se decida al final:

- Que **la ventana y los errores van en la misma versión**.
- Que **la regla de la mayúscula se mide por idioma** y que su prueba se pone roja al mezclarlos.
- Que los esquemas de MCP se prueban **con un agente de verdad**, no leyéndolos.
- Y que **el selector recuerda lo elegido**, con la misma regla del valor vacío que el tema: una
  cadena en blanco es «no lo he dicho», nunca «el del sistema».
