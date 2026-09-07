# ADR 0002 — XChaCha20-Poly1305 con Argon2id, en un contenedor versionado

**Fecha:** 2026-09-04 · **Estado:** aceptada

## Contexto

Hacía falta cifrar con una clave que teclea una persona. Eso son dos problemas distintos: convertir
una clave humana en material criptográfico, y cifrar con ella de forma que se note si alguien toca
el resultado.

## Decisión

- **Derivación:** Argon2id, 64 MiB, 3 pasadas, paralelismo 4, sal de 16 bytes. Ronda el medio
  segundo, que castiga la fuerza bruta sin que la interfaz parezca colgada.
- **Cifrado:** XChaCha20-Poly1305, con nonce de 24 bytes generado al azar.
- **Contenedor `ESF1`:** magia, versión, modo, los parámetros de derivación, sal y nonce, todo en
  claro pero **autenticado como datos asociados**.

En modo texto se emite como `ESF1.<base64url>` en una línea: sin `/`, `+` ni `=`, así que sobrevive
dentro de una URL, de un `.env` o de un JSON sin escapar nada.

## Alternativas descartadas

- **AES-GCM.** Su nonce de 96 bits es demasiado corto para generarlo al azar sin llevar cuenta. Los
  192 bits de XChaCha20 se pueden sortear sin miedo a colisiones, que es exactamente lo que hace
  falta cuando cada mensaje es independiente.
- **Guardar los parámetros de Argon2id fuera del contenedor.** Meterlos dentro permite subir el
  coste en el futuro sin romper lo ya cifrado.

## Consecuencias

- Un contenedor de la 1.0 se abre con la 2.x y al revés. El formato no ha cambiado.
- Se puede endurecer la derivación cuando las máquinas den para más, sin migraciones.
- La cabecera va autenticada: alterar un parámetro invalida la etiqueta.

## Verificación

`internal/cripto/contenedor_test.go` comprueba la ida y vuelta, que **cada byte** del contenedor
alterado uno a uno rompe la apertura, que un contenedor cortado a cualquier longitud falla, y congela
el tamaño y el orden de los campos de la cabecera para que un cambio accidental salte.
