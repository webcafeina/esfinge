# ADR 0067 — La marca lleva letra propia, y el trazo se retira

**Fecha:** 2026-11-20 · **Estado:** aceptada · **Matiza la [0021](0021-la-marca-en-la-interfaz.md)
y la [0007](0007-aspecto-del-sistema.md)** · **Revisar si** hace falta la marca por debajo de 30 px

## Contexto

Se le enseñaron al cliente cuatro tratamientos para el dibujo del instalador de Windows y los
descartó los cuatro, por una razón que no era del instalador: **los cuatro usaban la esfinge a
trazo** (`build/marca.svg`), y esa versión **se va a ir retirando de donde esté**. Lo que pidió fue
«un diseño con el logo completo, y con texto».

Eso obligaba a decidir algo que no existía: **un lockup**. Hasta hoy el nombre se ponía con la letra
de la interfaz al lado del trazo, así que no había ni tipografía de marca ni composición fijada.
Hacerlo directamente para el instalador habría fijado por accidente cómo se ve la marca en los otros
cinco sitios donde va, **y en el menos visible de los seis**.

## Decisión

**La marca es el logo completo con el nombre en Space Grotesk**, y la esfinge a trazo se retira.

Se eligió sobre cuatro direcciones tipográficas enseñadas en una galería, con el logo de verdad en
todas y cada una aplicada a la banda del instalador a tamaño real:

| | Dirección | |
|---|---|---|
| A | La letra del sistema, sin tipografía nueva | Coherente, pero una marca que no se reconoce fuera de su ventana |
| B | Grotesca sólida (Archivo) | Firme y poco memorable |
| **C** | **Geométrica (Space Grotesk)** | **Elegida** |
| D | Romana (Fraunces) | La más memorable; se lee «marca» y no «herramienta» |

Lo que no se cambia sin preguntar:

- **La tipografía de la interfaz no se toca.** Sigue siendo la del sistema (ADR 0007). Space Grotesk
  es **sólo para la marca**: la banda del instalador, la cabecera, y lo que venga de ese orden.
  **No viaja dentro del programa.**
- **Entra como fichero fijado**, no como descarga: `build/tipografias/space-grotesk/` con su `OFL.txt`
  y el commit de `google/fonts` del que salió. Es la misma regla que los iconos de Fluent.
- **El generador comprueba que la letra ha cargado** antes de escribir nada. Un `@font-face` que
  falla no da error: da la letra de respaldo, y el dibujo saldría con otra sin que nadie se entere.
- **El dibujo del instalador lleva texto**, con su precio dicho: NSIS estira esa imagen hasta llenar
  su hueco y el hueco crece con el DPI, así que a 200 % el nombre se dibuja al doble. Lo pidió el
  cliente sabiéndolo, y por eso el nombre va grande: lo que se estira mal es lo fino.

## Alternativas descartadas

**Seguir con el trazo**, que es lo que había. Lo descarta quien paga, y con un argumento que no es de
gusto: el trazo nació para poder teñirse dentro de la ventana, y la marca acabó teniendo dos caras
sin que ninguna fuera la de verdad.

**Hacer el instalador y decidir la marca después.** Era lo cómodo y fijaba la tipografía en la
pantalla que menos se ve, tres veces en la vida de una instalación.

**No usar tipografía de marca** (dirección A). Es la más barata y la más coherente con la 0007, y se
descartó por lo que cuesta: una marca sin letra propia cambia de forma según dónde se vea.

**Licenciar una tipografía de pago.** No se miró siquiera: las cuatro direcciones se eligieron ya con
licencia libre, que es lo que dice el catálogo de la casa —«usar la tipografía de una de estas marcas
exige licenciarla»—.

## Consecuencias

- **El trazo sale de cinco sitios**, y se comprobó antes de decidir que el cambio es viable: la marca
  **nunca se usa por debajo de 30 px** y el logo completo aguanta a 32.

  | Dónde | Tamaño | Con el logo completo |
  |---|---|---|
  | La ficha de Ajustes | 44 px | Entra |
  | La bienvenida y la cuenta | 44 y 88 px | Entra |
  | El banner de la extensión | 30 px | Entra, justo |
  | El correo | — | Ya usaba el logo completo |
  | **El historial vacío** | 112 px, muy apagado | **No sale solo**: ahí la esfinge es una filigrana teñida, y el logo con su placa no se apaga |

- **Queda pendiente el historial vacío**, que no es un logo sino una marca de agua. Se decide aparte
  y no bloquea nada; está en `deuda.md`.
- **Entra la primera tipografía del repositorio.** Con ella, la regla de que nada se descarga al
  compilar: el `.ttf` está comprometido.
- Y lo que no cambia: **la 0021 sigue en pie** en lo suyo —el oro como acento, la piedra encima, y
  que en las pantallas de trabajo no hay marca—.

## Verificación

- **El instalador montado en un Windows de verdad** antes de publicarlo, con `compilar.yml`:
  `Building 'arm64' installer: Done.` y **sin un solo aviso de NSIS**.
- **El dibujo, mirado**, que es lo que encontró los dos fallos que ninguna prueba vio: el marcador
  del logo aparecía también en el comentario que lo explicaba —y `replace` sustituyó ése, así que la
  banda salió con el marcador escrito dentro, la misma trampa que ya costó una publicación en
  `herramientas/rellenar`—; y quitarle el tamaño al SVG con una expresión a lo bruto se llevó el del
  rectángulo del fondo, así que el logo salía sin placa.
- **Dos guardas en el generador**, mutada la primera para verla roja: que el dibujo tenga dónde poner
  el logo, y que no quede ningún marcador sin sustituir.

**No comprobado, y hace falta su máquina**: cómo se ve el nombre a 150 % y a 200 %, que es donde NSIS
estira. Es el único riesgo que queda de haber metido texto en la imagen.
