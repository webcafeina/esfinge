# ADR 0011 — Cifrar copia al portapapeles solo; descifrar no

**Fecha:** 2026-09-07 · **Estado:** aceptada

## Contexto

Lo siguiente que se hace con un texto cifrado es pegarlo en algún sitio. Siempre. Obligar a pulsar
un botón para eso es un paso de más en el noventa y nueve por ciento de las veces.

## Decisión

Al cifrar un texto, el resultado va al portapapeles sin que nadie lo pida, y la pantalla lo dice con
un `✓` bien visible. Al descifrar, no.

## Alternativas descartadas

- **Copiar también al descifrar**, por simetría. Lo que sale ahí es el secreto en claro, y dejarlo
  en el portapapeles sin que nadie lo pida es meterlo donde puede leerlo cualquier cosa que vigile
  el portapapeles.
- **No copiar nunca**, y dejar solo el botón. Es lo que había, y era un paso de más siempre.

## Consecuencias

- Un texto cifrado pasa por el portapapeles sin que se haya pedido. Es aceptable: sin la clave no
  vale para nada.
- La asimetría hay que explicarla, y está en el README y en los comentarios del código.

## Verificación

`TestCifrarPideElCopiadoSolo` comprueba que cifrar lo pide y que descifrar no.
