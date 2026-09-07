# ADR 0005 — Todo en español, y toda frase empieza en mayúscula

**Fecha:** 2026-09-04 · **Estado:** aceptada

## Contexto

La herramienta la usan quien la encarga y un cliente, los dos hispanohablantes. Y hubo una petición
explícita sobre las mayúsculas que choca con la costumbre del lenguaje.

## Decisión

Todo lo que lee una persona, en español: menús, errores, ayuda de los comandos, descripciones de las
opciones. Y **toda frase empieza con la primera letra en mayúscula**, aunque sea de una sola palabra.

Se traduce incluso la ayuda que Cobra trae de fábrica, porque una ayuda medio traducida se lee peor
que ninguna.

## Alternativas descartadas

- **Errores en minúscula, como manda la costumbre de Go**, para poder encadenarlos con `%w`. Se
  descartó: estos errores se le enseñan tal cual a quien usa el programa, y ahí manda lo que se ve
  en pantalla.
- **Inglés**, por si el programa sale de la casa. Se deja para cuando haga falta de verdad.

## Consecuencias

- Los errores de los paquetes de Go que se envuelven siguen en minúscula y en inglés. Es inevitable
  y solo asoma en casos raros.
- Los identificadores del código también están en español, lo que hace el código más coherente con
  su documentación y más ajeno a quien venga de fuera.

## Verificación

`internal/cripto/textos_test.go` recorre los errores exportados, las valoraciones del medidor de
claves y los avisos de los alfabetos, y falla si alguno empieza en minúscula.
