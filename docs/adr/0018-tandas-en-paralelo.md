# ADR 0018 — Las tandas de ficheros se cifran en paralelo, con tope

**Fecha:** 2026-09-07 · **Estado:** aceptada · **Revisar si** cambia el perfil de derivación

## Contexto

Cada fichero de una tanda deriva su propia clave con Argon2id —64 MiB, 3 pasadas, 4 hilos— porque
cada contenedor lleva su propia sal, y eso es a propósito: es lo que hace que reventar uno no sirva
para el siguiente. El precio es medio segundo por fichero, y en fila.

Medido aquí, en cuatro núcleos: **veinte ficheros, 4,42 s**.

## Decisión

Se reparten, con un tope: **la mitad de los núcleos, máximo cuatro, mínimo uno**.

Medido después, en la misma máquina: **1,29 s**. Tres veces y media más rápido.

Dos cosas que el reparto no puede romper, y que están escritas en el código porque no se ven solas:

- **Los resultados conservan el orden de entrada**, aunque terminen desordenados. La lista que se ve
  tiene que corresponderse con la que se soltó; que el nombre de un fichero salga junto al resultado
  de otro sería la peor forma de equivocarse aquí.
- **El progreso se cuenta al terminar cada fichero, no al empezarlo.** Con varios a la vez,
  «empezando el 3 de 50» no significa nada.

## Alternativas descartadas

- **Tantos como núcleos.** Es lo obvio y es peor: cada derivación ya usa cuatro hilos por dentro, así
  que se reparten los mismos núcleos, y ocho a la vez son medio giga de memoria. En una máquina justa
  puede acabar siendo más lento que en fila.
- **Un control en Ajustes.** Una perilla que casi nadie tocaría y que habría que explicar.
- **Derivar la clave una sola vez para toda la tanda.** Sería mucho más rápido, y **no**: cada
  contenedor lleva su sal justamente para que no exista un atajo así.

## Consecuencias

- El pico de memoria pasa a ser el tope por 64 MiB: unos 256 MiB en una máquina de ocho núcleos.
- Es la primera vez que Esfinge cifra desde varias gorrutinas a la vez, así que `go test -race` deja
  de ser una cortesía. El historial ya tenía su mutex y la clave derivada solo se lee.

## Verificación

- Tests: que el tope cae dentro de sus límites en cualquier máquina; que una tanda de veinte
  **devuelve los resultados en el orden de entrada**; y que el progreso pasa por todas las cuentas
  sin saltarse ninguna, aunque lleguen desordenadas.
- `go test -race` sobre todo el paquete.
- La medida de arriba, con `MEDIR=1 go test -run TestMedirUnaTandaDeVeinte ./internal/app/`, que se
  ejecuta a mano y por eso no alarga la batería de siempre.
