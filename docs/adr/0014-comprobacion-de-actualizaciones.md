# ADR 0014 — Esfinge comprueba si hay versión nueva, y se la descarga

**Fecha:** 2026-09-07 · **Estado:** aceptada, con una parte sustituida por la
[0016](0016-actualizarse-sola.md) · **Revisar si** algún día se firma con Apple

> **Lo que aquí se decidió sobre no reemplazarse a sí misma dejó de valer el mismo día.** Daba por
> hecho que hacía falta firmar con Apple, y no es cierto: la firma evita el aviso de Gatekeeper, no
> habilita el reemplazo. Lo corrige la [0016](0016-actualizarse-sola.md). El resto de esta ficha
> —qué se envía, cada cuánto, qué protege el SHA256— sigue en pie.

## Contexto

Esfinge se publica sola desde la [0013](0013-repositorio-publico.md): una etiqueta `v*` compila los
tres sistemas y cuelga el DMG, el instalador de Windows y el `.deb`. Pero quien la tiene instalada no
se entera de nada. Para pasar de una versión a la siguiente hay que saber que existe, ir al
repositorio, encontrar el fichero del sistema propio y bajarlo. Nadie hace eso, así que en la
práctica el cliente se queda en la versión con la que se fue, incluidos los arreglos.

Contra eso está lo que promete la portada: que nada sale de tu ordenador. Hasta hoy era literal —el
binario del cliente no tenía ni `net/http`, y el único que había estaba tras la etiqueta `dev`
([0009](0009-puente-de-dos-caminos.md))—.

## Decisión

La aplicación **comprueba una vez al día** si hay versión nueva, **se descarga el fichero** que le
toca a ese sistema comprobando su SHA256, y **abre el instalador**. No se sustituye a sí misma.

- **Automática y desactivable**, no manual: un botón «Buscar actualizaciones» que nadie pulsa no
  resuelve el problema. La contrapartida es que **se cuenta en Ajustes**, con lo que se envía escrito
  a la vista: una petición GET a `api.github.com` que lleva la versión instalada en el `User-Agent`,
  y donde GitHub ve la IP, como cualquier página. Nada de lo que se cifra, ni identificadores, ni
  cuántas veces se usa.
- **Se enseña el número y un enlace**, no las notas de la publicación.
- **La línea de comandos también avisa**, por la salida de error, nunca por la estándar, y **solo si
  esa salida es un terminal**: en una tubería o un `cron` no escribe una línea. `ESFINGE_SIN_RED=1`
  la apaga entera.
- **Una compilación de trabajo no recibe avisos**: si la versión instalada no es de tres números
  —`dev`, o el `2.0.3-3-gabc1234` de `git describe`— no se dice nada.

## Alternativas descartadas

- **Reemplazo automático del binario**, que es lo que hacen las aplicaciones grandes. En macOS exige
  firmar con Apple —99 $ al año, descartado en la [0012](0012-sin-firmar.md)— o el sistema bloquea el
  resultado; en Windows y Linux choca con los permisos del sitio donde está instalada. Se llega hasta
  la puerta y llama: el instalador de cada sistema pone el resto, y ninguno de los tres exige
  desinstalar antes.
- **Solo avisar y abrir el navegador.** Es lo más sencillo y lo más limpio de contar, pero deja el
  paso de encontrar el fichero correcto entre once adjuntos, que es justo donde se pierde la gente.
- **Comprobar solo cuando se pide.** Honesto, pero inútil: el que no pulsa el botón —o sea, casi
  todos— no se entera nunca.
- **Un servidor propio de actualizaciones.** Otra cosa que mantener y otro sitio donde confiar, para
  no ganar nada: las publicaciones ya están en GitHub.

## Consecuencias

- **El binario del cliente lleva red.** Es un cambio de naturaleza, no de detalle, y obliga a matizar
  lo que dice la portada. Va en `docs/seguridad.md`, no solo aquí.
- **Lo que protege el SHA256 es una descarga rota, no una publicación manipulada**: el resumen sale
  del mismo sitio que el fichero. Lo que sostiene la confianza es el TLS contra GitHub. Decirlo al
  revés sería vender una garantía que no existe.
- El `SHA256SUMS` que se publica tuvo que **rehacerse sobre todos los adjuntos**: hasta la 2.0.3 solo
  cubría los seis binarios de la línea de comandos, así que el DMG, el instalador y el `.deb` se
  publicaban sin resumen contra el que comparar.
- Aparece un fichero de preferencias (`preferencias.json`, junto al historial, con los mismos
  permisos) y una quinta pestaña, Ajustes. Antes no había ninguno de los dos.
- El límite de la API de GitHub sin credenciales son 60 peticiones por hora y por IP. Con una diaria
  sobra; si falla, no se dice nada y se reintenta al día siguiente.

## Verificación

- Tests contra un `httptest.Server`, sin tocar internet: comparación de versiones incluidos `dev` y
  lo que sale de `git describe`, elección del fichero de cada sistema, **una descarga cuyo resumen no
  cuadra se rechaza y se borra**, el plazo de un día se respeta, y con el interruptor apagado **no se
  hace ni una petición** —eso se cuenta, no se supone—.
- Pruebas de interfaz contra una API de mentira (`frontend/e2e/api-falsa.mjs`), en los dos temas: sale
  la banda, «Ahora no» la cierra, y el interruptor sobrevive a recargar.
- En la línea de comandos se comprueba que, con la salida de error redirigida —que es el caso de un
  script—, **no se arranca ninguna consulta**.

**Lo que no se ha comprobado:** la instalación de verdad desde la propia aplicación, en ninguno de
los tres sistemas. Queda por ver si al descargar el DMG desde Go —y no desde un navegador— la copia
instalada se libra del aviso de Gatekeeper, porque la cuarentena la pone quien descarga.
