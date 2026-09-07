# ADR 0004 — Las contraseñas se generan en hexadecimal por defecto

**Fecha:** 2026-09-04 · **Estado:** aceptada

## Contexto

Una contraseña generada acaba, casi siempre, dentro de una cadena de conexión.

## Decisión

Hexadecimal por defecto. Los alfabetos con símbolos existen, pero avisan cada vez que se eligen.

## Alternativas descartadas

- **Base64 o alfabetos con símbolos por defecto.** Más entropía por carácter, y una trampa: una
  contraseña con `/` parte la cadena de conexión —`postgres://usuario:pa/ss@host` deja de ser una
  URL—. El error que sale por el otro lado no menciona la contraseña por ninguna parte, así que se
  buscan horas donde no es. Costó rehacer un despliegue de Tempero entero, con el worker en bucle de
  reinicios.

## Consecuencias

- Una contraseña hexadecimal necesita más caracteres para la misma fuerza. Es un intercambio
  deliberado: caracteres sobran, horas de depuración no.
- El aviso de los alfabetos con símbolos aparece en la interfaz **y en la línea de comandos**,
  incluso cuando la salida va a un fichero: se manda por la salida de error, que es donde sigue
  viéndose.

## Verificación

`internal/cripto/flujo_test.go` comprueba que los alfabetos declarados seguros en URL no producen
nunca `/ + @ : # ? & = %`, y que el que no lo es trae aviso.
