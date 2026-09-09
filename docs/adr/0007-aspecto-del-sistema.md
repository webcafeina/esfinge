# ADR 0007 — La ventana sigue la apariencia del sistema, no una identidad propia

**Fecha:** 2026-09-07 · **Estado:** aceptada · **Sustituye a** la identidad de ClickHouse de la 1.x ·
**Matizada por la [0021](0021-la-marca-en-la-interfaz.md)**

> **Corrección (2026-09-09).** Lo de fondo de esta ficha sigue en pie: tipografía, controles,
> estructura y apariencia del sistema, sin identidad ajena. Lo que decae es la frase «la marca queda
> en el icono y en *Acerca de*», por dos motivos. Uno, que **ese «Acerca de» nunca se construyó**: el
> menú navega a Ajustes y allí solo había una línea con el número de versión, así que media promesa
> llevaba sin cumplirse desde la 2.0.0. Y dos, que desde la 2.11.0 **el acento de la interfaz es el
> oro de la esfinge y no el azul del sistema**. La 0021 lo cuenta entero, con los números.

## Contexto

La 1.x vestía a Esfinge con la identidad completa de ClickHouse tomada del catálogo de la casa:
negro, amarillo eléctrico, filetes de un píxel. Funcionaba en un terminal, donde no hay convenciones
que seguir y una identidad fuerte ayuda a que la herramienta se reconozca.

En una ventana la convención sí existe, y saltársela hace que la aplicación se sienta ajena.

## Decisión

Apariencia del sistema: tipografía y controles de macOS y Windows, barra de herramientas
translúcida, radios generosos, controles de 28 píxeles, foco como anillo separado del control. Modo
claro y oscuro siguiendo la preferencia del escritorio, no una propia. La marca queda en el icono y
en «Acerca de».

## Alternativas descartadas

- **Mantener la identidad de ClickHouse.** Es la decisión correcta para un terminal y la equivocada
  para una ventana.
- **Inventar una identidad propia para la aplicación.** Más trabajo y el mismo problema: una ventana
  que no se parece a las demás del sistema.

## Consecuencias

- El mecanismo de color se conserva entero —el tipo `Tema`, `AcentoLegible`, los tests de
  contraste—; lo que cambian son los valores. Ver [ADR 0008](0008-color-generado-desde-go.md).
- **El azul de botón del sistema no cumple AA con texto blanco encima**: el `#007aff` de macOS da
  3,6:1. Ni Apple ni Microsoft lo cumplen. Aquí se oscurece hasta que se lee, con `RellenoLegible`,
  y sigue leyéndose como el azul del sistema.
- La primera versión de este aspecto se quedó corta —«muy básico, no parece nativo»— y hubo que
  rehacerla con barra translúcida, radios mayores y bloques agrupados.

## Verificación

`make contraste` mide las parejas que la interfaz usa de verdad en los dos temas, incluidas las
superficies con nombre propio del sistema —barra, campo, botón—, y falla si una no llega a AA.
