# Seguridad

Última actualización: **2026-09-07**

Qué protege Esfinge y qué no. En una herramienta que cifra, lo segundo importa tanto como lo
primero: una expectativa equivocada sobre lo que protege es exactamente lo que hace daño.

## Lo que hace

Coge un secreto y una clave y produce un contenedor que **solo se abre con esa clave**. Quien tenga
el contenedor y no la clave no tiene nada: ni el contenido, ni una versión parcial, ni pistas sobre
su longitud más allá de la evidente.

Y **detecta si el contenedor ha cambiado**. Alterar un byte, cortarlo por la mitad o reordenar sus
partes hace que la apertura falle en vez de devolver algo distinto de lo que se guardó.

## De qué protege

| Amenaza | Mitigación | Riesgo residual |
|---|---|---|
| Alguien intercepta el texto cifrado por correo o chat | XChaCha20-Poly1305; sin la clave no hay contenido | Se ve **que** hay algo cifrado y su tamaño aproximado |
| Alguien roba el fichero `.esf` de un disco | Lo mismo | Igual |
| Alguien modifica el contenedor por el camino | Etiqueta de autenticación por segmento, con la cabecera autenticada | Ninguno conocido: cualquier cambio rompe la apertura |
| Alguien corta el fichero y lo pasa por entero | Marca en el último segmento ([ADR 0003](adr/0003-marca-de-final.md)) | Cortar dentro del **primer** segmento no se distingue de una clave equivocada |
| Fuerza bruta sobre la clave | Argon2id con 64 MiB y 3 pasadas: cada intento cuesta medio segundo y mucha memoria | Una clave corta sigue siendo una clave corta. El medidor avisa |
| Dos mensajes iguales delatan que lo son | Sal y nonce nuevos en cada operación | Ninguno |

## De qué NO protege

Esto es lo importante de este documento.

- **De quien ya está dentro de la máquina.** Mientras la ventana está abierta, la clave vive en
  memoria. Quien pueda leer la memoria del proceso, poner un registrador de teclas o hacerse pasar
  por el usuario, no necesita romper nada.
- **De perder la clave.** No hay recuperación, ni puerta trasera, ni copia en ninguna parte. Si se
  pierde la clave, el contenido se ha perdido. Esto no es un fallo: es lo que significa cifrar.
- **De que se sepa qué has cifrado.** El historial guarda nombres de fichero y fechas
  ([ADR 0010](adr/0010-que-guarda-el-historial.md)). No guarda contenidos ni claves, pero saber que
  el martes cifraste `credenciales-banco.env` ya dice algo. Se puede vaciar desde la propia ventana.
- **De un canal inseguro para la clave.** Si el contenedor y la clave viajan por el mismo correo,
  quien lea ese correo lo tiene todo. Esto no lo puede resolver el programa.
- **De un fichero manipulado antes de cifrarlo.** Esfinge sella lo que le den; no sabe si lo que le
  dieron era lo que debía.
- **Del portapapeles.** Al cifrar, el resultado se copia solo ([ADR 0011](adr/0011-copiado-automatico.md)).
  Cualquier programa que vigile el portapapeles lo verá. Sin la clave no le sirve, pero conviene
  saberlo. Al descifrar no se copia nada por su cuenta, justamente por esto.

## Dónde queda algo en disco

| Qué | Dónde | Permisos |
|---|---|---|
| Historial | Carpeta de configuración del usuario, `Esfinge/historial.json` | `600` |
| Ficheros cifrados | Junto al original, con `.esf` al final | `600` |
| Lo que se guarda desde la ventana | Donde diga el diálogo del sistema | `600` |

**Nada sale de la máquina.** Esfinge no habla con internet: no hay telemetría, ni comprobación de
versiones, ni informes de fallos.

## Decisiones que afectan a la seguridad

- [ADR 0002](adr/0002-formato-esf1.md) — El cifrado y por qué esos algoritmos.
- [ADR 0003](adr/0003-marca-de-final.md) — Por qué un fichero cortado se detecta.
- [ADR 0004](adr/0004-contrasenas-en-hexadecimal.md) — Por qué las contraseñas salen en hexadecimal.
- [ADR 0010](adr/0010-que-guarda-el-historial.md) — Qué se guarda y qué no.
- [ADR 0012](adr/0012-sin-firmar.md) — Por qué el sistema avisa al instalarla.

## Lo que no se ha auditado

Nadie de fuera ha revisado esto. El núcleo tiene pruebas que cubren la ida y vuelta, la manipulación
de cada byte, el truncado y la reordenación, y usa implementaciones de la biblioteca estándar
extendida de Go —`golang.org/x/crypto`— en vez de nada escrito aquí. Pero **una batería de pruebas
propia no es una auditoría**, y conviene decirlo antes de que alguien confíe más de la cuenta.
