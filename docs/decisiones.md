# Decisiones

Última actualización: **2026-09-07**

Una ficha por decisión no trivial, en [adr/](adr/). Las que se superan **no se borran**: se marcan y
se quedan, porque saber qué se pensaba antes explica por qué el código es como es.

Cada ficha lleva Contexto, Decisión, Alternativas descartadas, Consecuencias y Verificación. La
última sección es la que más se agradece meses después: dice qué se comprobó de verdad y qué no.

| # | Decisión | Fecha | Estado |
|---|---|---|---|
| [0001](adr/0001-go-y-binario-unico.md) | Go, y un binario por plataforma | 2026-09-04 | aceptada |
| [0002](adr/0002-formato-esf1.md) | XChaCha20-Poly1305 con Argon2id, contenedor versionado | 2026-09-04 | aceptada |
| [0003](adr/0003-marca-de-final.md) | Ficheros por segmentos, con marca en el último | 2026-09-04 | aceptada |
| [0004](adr/0004-contrasenas-en-hexadecimal.md) | Contraseñas en hexadecimal por defecto | 2026-09-04 | aceptada |
| [0005](adr/0005-espanol-y-mayuscula-inicial.md) | Todo en español, y con mayúscula inicial | 2026-09-04 | aceptada |
| [0006](adr/0006-de-terminal-a-ventana.md) | De terminal a ventana, con Wails | 2026-09-07 | aceptada |
| [0007](adr/0007-aspecto-del-sistema.md) | Apariencia del sistema, no identidad propia | 2026-09-07 | aceptada · sustituye la identidad de la 1.x |
| [0008](adr/0008-color-generado-desde-go.md) | El color se genera desde Go | 2026-09-07 | aceptada |
| [0009](adr/0009-puente-de-dos-caminos.md) | El puente con la interfaz tiene dos caminos | 2026-09-07 | aceptada |
| [0010](adr/0010-que-guarda-el-historial.md) | El historial guarda qué y cuándo, nada más | 2026-09-07 | aceptada |
| [0011](adr/0011-copiado-automatico.md) | Cifrar copia solo; descifrar no | 2026-09-07 | aceptada |
| [0012](adr/0012-sin-firmar.md) | No se firma con Apple | 2026-09-04 | aceptada · revisar si crece el número de clientes |
| [0013](adr/0013-repositorio-publico.md) | Repositorio público, licencia propietaria | 2026-09-07 | aceptada |
| [0014](adr/0014-comprobacion-de-actualizaciones.md) | Comprueba y descarga actualizaciones | 2026-09-07 | aceptada · revisar si se firma con Apple |
| [0015](adr/0015-menus-en-espanol.md) | La barra de menús se construye entera, en español | 2026-09-07 | aceptada · revisar si Wails localiza sus roles |
| [0016](adr/0016-actualizarse-sola.md) | Se reemplaza a sí misma y se reinicia | 2026-09-07 | aceptada · sustituye parte de la 0014 |
| [0017](adr/0017-vidrio-solo-en-el-marco.md) | El vidrio va en el marco, no en la zona de trabajo | 2026-09-08 | aceptada · revisar si Wails lo ofrece en Linux |
| [0018](adr/0018-tandas-en-paralelo.md) | Las tandas se cifran en paralelo, con tope | 2026-09-08 | aceptada · revisar si cambia el perfil |
| [0019](adr/0019-estructura-de-macos.md) | La ventana se organiza como una aplicación de macOS | 2026-09-08 | aceptada · Windows y Linux van después |

## Cuándo escribir una

Cuando la decisión costaría volver a discutir, cuando el código quedaría raro sin explicación, o
cuando se descartó algo que parecía la opción evidente. Si no se cumple ninguna de las tres,
probablemente sea un comentario en el código y no una ficha.
