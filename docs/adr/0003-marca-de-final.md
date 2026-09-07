# ADR 0003 — Los ficheros van por segmentos, y el último lleva marca

**Fecha:** 2026-09-04 · **Estado:** aceptada

## Contexto

Un fichero de credenciales puede ser de cualquier tamaño, y cargarlo entero en memoria para cifrarlo
no es aceptable. Trocearlo sí, pero trocear abre un agujero que no es evidente.

## Decisión

Segmentos de 64 KiB, cada uno con su etiqueta de autenticación y su número de orden dentro del
nonce, y **una marca en el último**.

## Alternativas descartadas

- **Trocear sin marcar el final.** Es lo que parece suficiente y no lo es: quien corte el fichero
  por el borde de un segmento obtiene un prefijo perfectamente válido. Todos los segmentos que
  quedan descifran bien, el proceso termina sin quejarse, y el destinatario se queda con medio
  secreto creyendo que lo tiene entero.

## Consecuencias

- Cortar, reordenar o repetir segmentos se detecta.
- Un fichero cortado a mitad de un segmento **a partir del segundo** se distingue de una clave
  equivocada: si el primero abrió, la clave es buena y el mensaje lo dice. Dentro del primero no hay
  forma de distinguirlos, y ahí el mensaje nombra las tres posibilidades en vez de mentir.

## Verificación

`internal/cripto/flujo_test.go` corta un contenedor de tres segmentos por el borde exacto de uno
—el caso que pasaría desapercibido sin la marca— y comprueba que salta. También reordena segmentos,
altera cada byte y verifica que un fichero cortado no culpa a la clave.
