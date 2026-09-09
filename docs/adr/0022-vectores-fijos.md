# ADR 0022 — El formato se congela con vectores fijos, no con un test que se mira al espejo

**Fecha:** 2026-09-09 · **Estado:** aceptada · **Revisar si** se sube la versión del contenedor

## Contexto

La [ADR 0002](0002-formato-esf1.md) afirma desde el principio que «un contenedor de la 1.0 se abre
con la 2.x y al revés». **Nada lo verificaba.**

El único test que decía congelar el formato era `TestCabeceraCongelada`: sella un contenedor en el
momento y comprueba que la cabecera mide 55 bytes, que la magia es la que es y que los parámetros
sobreviven la ida y vuelta. Es útil contra un descuido —cambiar el orden de dos campos, olvidar un
byte— y **es ciego a toda una clase de fallos**: los que cambian la semántica de forma **coherente en
los dos sentidos**.

El orden de bytes de los parámetros de Argon2id. El offset donde arranca el contador de segmento
dentro del nonce. Qué entra exactamente en el AAD. Cambiar cualquiera de esas cosas al escribir *y*
al leer deja todos los tests en verde y **deja de abrir los ficheros ya emitidos, en silencio**.

Nadie se enteraría hasta que un cliente intentara abrir algo suyo de hace un año.

Y esto salió a la luz justo antes de empezar la bóveda de contraseñas, que se va a apoyar en estas
primitivas exactas. Tocar el formato sin red era construir sobre arena.

## Decisión

**Contenedores reales grabados una vez, en `internal/cripto/testdata/`, que no se regeneran nunca.**

Once vectores buenos —los dos modos, los dos formatos, los bordes de segmento, un claro y una clave
con acentos, y uno con el perfil de verdad— más tres rotos que congelan **qué error** da cada clase
de daño, porque de esa clasificación salen los códigos de salida 3 y 4 de la línea de comandos.

Se comprueban **dos caminos, no uno**:

- **Que se abren.** Basta con tenerlos grabados.
- **Que se sellan igual.** Con la misma sal y el mismo nonce, sellar el mismo claro tiene que dar los
  mismos bytes. **Éste es el que atrapa el cambio coherente**, y sin él la mitad del formato seguiría
  sin congelar.

Para lo segundo hace falta poder fijar la sal y el nonce, así que **`azar` pasa de función a
variable** en `clave.go`. Es el único cambio en código de producción, son tres caracteres, y solo lo
sustituyen los tests.

## Alternativas descartadas

- **Dejar `TestCabeceraCongelada` como estaba.** Es lo que había, y la demostración de que no basta
  está en la verificación de abajo.
- **Un test de propiedades o de fuzzing.** Encuentran pánicos y casos raros; no congelan un formato.
  El fuzzing sigue siendo buena idea, pero para otra cosa.
- **Grabar los vectores con una bandera `-actualizar`.** Es el patrón habitual con ficheros dorados y
  aquí es justo el error: invita a regenerar cuando un test se pone rojo, que es exactamente el
  momento en que no hay que hacerlo. **El programa que los generó se borró en el mismo commit**: no
  hay botón que apretar, y volver a grabarlos exige escribirlo a conciencia.
- **Guardar también el claro de los vectores grandes.** Duplicaría 130 KiB en el repositorio para
  nada: se reconstruye con una fórmula fija de tres líneas.

## Consecuencias

- **El formato queda cerrado de verdad.** Evolucionarlo pasa a ser un acto deliberado: subir la
  versión del contenedor y grabar vectores nuevos **al lado**, sin tocar los viejos, que siguen
  demostrando que lo antiguo se abre.
- **`azar` es una variable.** Cualquiera que la vea tiene que entender por qué; el comentario lo
  explica en el propio fichero.
- Un test clava `PerfilInteractivo` en `{64 MiB, 3, 4}`. Subir el coste **no rompe nada ya cifrado**
  —los parámetros viajan dentro del contenedor— y por eso es fácil hacerlo sin querer: ahora hay un
  test en rojo delante de quien lo haga.
- Otro test vigila que **el byte de versión nunca valga 46**. Hoy es trivialmente cierto, y por eso
  hay que escribirlo ahora: `FormaDe` distingue el binario de la forma de texto mirando el quinto
  byte, y `ESF1.` es la forma de texto. Quien suba la versión dentro de años pasará por la 45 y la 47
  sin sospechar que la 46 rompe la detección.
- 324 KiB más en el repositorio, y unas décimas de segundo en cada `make comprobar`.
- **El riesgo que queda no es técnico.** Si alguien regenera los vectores para que un test vuelva a
  verde, la red desaparece sin que nadie lo note. Contra eso solo hay una frase escrita donde se va a
  leer: `testdata/LÉEME.md`, y esta ficha.

## Verificación

**Se comprobó que fallan cuando tienen que fallar, que es lo único que demuestra que sirven.**
Cambiando el orden de bytes de los parámetros a la vez al escribir y al leer:

| | resultado |
|---|---|
| `TestIdaYVuelta` | pasa |
| `TestCabeceraCongelada` | **pasa** |
| `TestLosVectoresFijosSeAbren` | **falla** |

Ése es exactamente el fallo que hasta hoy se habría publicado en verde.

Y lo de siempre: los 11 vectores se abren y se vuelven a sellar idénticos, los 3 rotos dan el error
que les toca, y `make comprobar` y `go test -race` en verde.

### Y la retrocompatibilidad, demostrada en vez de afirmada

Los once vectores de arriba se grabaron con la 2.11.2, así que congelan de aquí en adelante. Eso
dejaba sin comprobar justo lo que la ADR 0002 lleva prometiendo desde el principio: **que un
contenedor de la 1.x se abre con la versión de ahora**.

Se comprobó de verdad. El historial dice que el contenedor nació en el commit de la **1.5.0** y que
desde entonces `contenedor.go` solo se tocó una vez, para añadir `FormaDe`, que solo lee. Pero eso es
creerse un diff, así que se sacó ese commit con `git archive`, se compiló **el código de la 1.5.0 tal
cual** y se sellaron con él tres contenedores —binario, línea de texto y fichero por segmentos— que
viven ahora en `testdata/v15-*`.

`TestLosContenedoresDeLa150SeSiguenAbriendo` los abre con el código de hoy, **con la clave de
entonces tecleada tal cual**. Si algún día alguien tocara la codificación de la clave —normalizar
acentos, recortar espacios—, ese test se pondría rojo: las claves ya emitidas tienen que seguir
valiendo tanto como los ficheros.

De esos tres se comprueba **solo que se abren**, y es deliberado: se sellaron con sal y nonce de
verdad, así que no se pueden reproducir. Son vectores de lectura, que es lo único que significa la
retrocompatibilidad.

**Lo que sigue sin cubrirse:** contenedores de la 1.0 a la 1.4, si es que llegaron a existir con este
formato. El commit de la 1.5.0 es el primero que lo tiene.
