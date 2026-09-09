# Vectores fijos del formato ESF1

**Estos ficheros no se regeneran.** Se grabaron una vez, con la 2.11.2, y su valor entero consiste en
no volver a tocarlos.

## Qué son

Contenedores `ESF1` reales, con su sal y su nonce anotados en `vectores.json` para poder reproducir
cada uno byte a byte. Los vigila `../vectores_test.go`, que comprueba dos cosas distintas:

- que **se siguen abriendo** —el camino de lectura—;
- que **sellar el mismo claro con la misma sal y el mismo nonce da los mismos bytes** —el camino de
  escritura—.

La segunda es la que importa, y es la razón de que `azar` sea una variable en `clave.go`.

## Por qué existen

Hasta la 2.11.2, el único test que decía congelar el formato era `TestCabeceraCongelada`, que **sella
y abre en el momento**: comprueba el código contra sí mismo. Eso deja pasar toda una clase de fallos
—los que cambian la semántica de forma coherente en los dos sentidos— y son precisamente los peores,
porque no rompen ninguna prueba y dejan de abrir en silencio los ficheros ya emitidos.

Se comprobó de verdad antes de dar esto por bueno: cambiando el orden de bytes de los parámetros a
la vez al escribir y al leer, `TestCabeceraCongelada` y `TestIdaYVuelta` **siguen pasando** y estos
vectores **fallan**. Ése es todo el motivo de la carpeta.

Y la ADR 0002 afirmaba desde el principio que un contenedor de la 1.0 se abre con la 2.x. Hasta
ahora, nada lo verificaba.

## Si un cambio los pone en rojo

Solo hay dos posibilidades, y ninguna es regenerarlos:

1. **El cambio está mal.** Es el caso en la inmensa mayoría de las veces. Se arregla el cambio.
2. **El cambio es intencionado y el formato tiene que evolucionar.** Entonces se **sube la versión
   del contenedor** y se graban vectores nuevos **al lado**, sin tocar éstos, que siguen
   demostrando que la versión vieja se abre.

Regenerarlos para que el test vuelva a verde borra la única red que hay, y nadie se enteraría hasta
que un cliente no pudiera abrir algo suyo de hace un año. Por eso el programa que los generó **se
borró** en el mismo commit: no hay un botón que apretar.

## Qué congela cada uno

| Fichero | Qué fija |
|---|---|
| `unico-vacio.esf` | Carga de cero bytes: 55 de cabecera + 0 + 16 de etiqueta |
| `unico-corto.esf` | El caso normal: offsets, AAD y orden de bytes |
| `unico-utf8.esf` | Que **la clave son sus bytes UTF-8 crudos, sin normalizar**. Nadie lo había escrito, y rompería a quien teclee acentos en otro sistema |
| `unico-parametros.esf` | Que los parámetros de Argon2id viajan dentro y se honran al abrir |
| `unico-perfil.esf` | El único con `PerfilInteractivo`: clava 64 MiB / 3 pasadas / 4 hilos en un fichero de verdad |
| `texto.esf1` | La forma de texto: prefijo, base64url y ausencia de relleno |
| `flujo-vacio.esf` | Flujo de cero bytes: cuántos segmentos hay y dónde va la marca de final |
| `flujo-un-segmento.esf` | Por debajo del tamaño de segmento |
| `flujo-limite.esf` | **Exactamente 64 KiB**: si hay o no un segmento final vacío |
| `flujo-dos-segmentos.esf` | 64 KiB + 1: el primer cruce del contador |
| `flujo-tres-segmentos.esf` | El contador más allá de 1, que es donde vive el error de índice |
| `roto-truncado.esf`, `roto-bit.esf`, `roto-version99.esf` | La **clasificación** del error, no solo que falle. De ahí salen los códigos de salida 3 y 4 de la línea de comandos |

El claro de los vectores de flujo no se guarda: se reconstruye con una fórmula fija en el test, para
no duplicar 130 KiB en el repositorio.
