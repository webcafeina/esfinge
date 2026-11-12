# 0060 · Un nombre por adjunto, y un solo sitio donde se dice

**Fecha**: 2026-11-12 · **Estado**: en marcha, la fase 0 publicada

## Contexto

Esfinge publica 28 ficheros y va a publicar unos 37: la aplicación con ventana para los seis objetivos,
`.deb` y `.rpm` por arquitectura, Arch, AppImage, los `tar.gz`, los dieciocho de la línea de comandos y los
paquetes de agentes. Los nombres de todos ellos vivían en **siete sitios**:

- el `Makefile` (`publicar-cli`),
- `.github/workflows/publicar.yml`, que arma el DMG, el instalador y el `.deb`,
- `herramientas/actualizar-descargas.sh`, con la tabla del README escrita a mano,
- el cuerpo de la publicación, con **otra tabla igual** escrita a mano,
- `internal/actualizacion`, que elige qué descargarle a cada equipo,
- `empaquetado/linux/instalar.sh`,
- y cinco ficheros de prueba, más la API de mentira del e2e.

En esta casa **una regla escrita en N sitios ya ha costado cinco fallos** —`vaciarLoSensible`,
`enConfiguracionDePruebas` (que tiró una publicación), `PasaPorLaValvula`, el sello armado en dos mitades y
el gancho de la cadena que dejó sordo al VPS—. El patrón no es el descuido: es que añadir un caso obliga a
acordarse de N sitios, y acordarse no es defensa.

Y había un fallo esperando, que es lo que forzó la fecha. `instaladoConPaquete()` contestaba «¿vengo de un
paquete?» mirando si el binario vive en `/usr/bin`. Con un solo paquete en el mundo eso valía; **el día que
exista un `.rpm`, una Fedora contesta que sí y recibe la oferta de un `.deb`**, que `dnf` no sabe instalar.
La pregunta que hay que saber contestar no es si vino de un paquete, sino **de cuál**.

## Decisión

**Un catálogo, en Go, en `internal/adjuntos`.** Cada adjunto es una ficha con su clave, su sistema, su
arquitectura, su clase, **el gestor con el que se instala** y la plantilla de su nombre. Los demás preguntan:
el que elige la descarga importa el paquete, y quien no es Go llama a `herramientas/catalogo`.

**En Go y no en un YAML ni en Python**, por la lección de `zip`: una herramienta de línea de órdenes que
está en los tres sistemas de desarrollo puede no estar en los tres de la publicación, y lo único garantizado
en las cinco máquinas que compilarán —incluidas las dos imágenes ARM que nadie de esta casa ha visto por
dentro— es Go, porque las cinco compilan Esfinge. Es el mismo argumento que decidió `herramientas/armar-mcpb`.

**Cada ficha dice si está publicada.** Es lo que hace que las fases se puedan publicar por separado: el
catálogo nombra un adjunto **antes** de que exista, para que el que elige la descarga lo acepte el día que
aparezca, mientras la puerta de la publicación solo exige los que ya tienen quien los fabrique. Cada fase es
un `true` más el trabajo que lo produce, y nunca al revés.

**El que elige recibe el equipo por parámetro** —sistema, arquitectura y gestor— en vez de leer `runtime`.
Y **sigue buscando por sufijo**, que es la garantía de la versión anterior (una copia instalada encuentra su
fichero aunque el nombre cambie), pero el sufijo sale de la plantilla y **ya no hay ninguno genérico**.

**Y el gestor lo escribe el paquete al instalarse**, en `/usr/share/esfinge/gestor`: es el único que lo sabe
de verdad, porque el binario de un `.deb` y el de un `.rpm` viven los dos en `/usr/bin`. Cada empaquetador
lleva su propio guion de postinstalación, porque ese fichero se ejecuta en el equipo de quien instala y una
variable de entorno de la máquina que arma el paquete no llegaría allí.

## Alternativas descartadas

**Un YAML o un JSON con la lista, leído por todos.** Es lo primero que se piensa y falla por donde falla
siempre: hace falta un programa que lo lea, y ese programa sería `python3` o `yq` — que es exactamente la
apuesta que ya costó una publicación cuando `zip` no estaba en la máquina Windows de GitHub.

**Dejarlo en el `Makefile`.** Ya tiene los seis objetivos, así que parecía su sitio. Pero el `Makefile` no lo
puede leer el código que elige la descarga, que es el consumidor que de verdad importa: el que decide en el
equipo del cliente.

**Componer los nombres en vez de buscar por sufijo.** Más preciso, y rompe la promesa escrita desde la 2.0:
una copia ya instalada tiene que encontrar su fichero aunque el nombre cambie. Se mantiene el sufijo y se
quita la ambigüedad, que era el problema real.

**Adivinar el gestor en el equipo del cliente** (mirando si hay `rpm` o `dpkg`). Una Fedora puede tener
`dpkg` instalado y un Debian puede tener `rpm`: es la misma suposición, con otra cara.

## Consecuencias

- La tabla del README y el cuerpo de la publicación **salen del catálogo**, así que no pueden desalinearse
  entre sí ni con lo publicado. Y una fila solo existe si su fichero existe: nunca un enlace a un 404.
- **La puerta de la publicación compara lo que hay con lo que el catálogo dice, en los dos sentidos.** Que no
  falte ninguno —una pata de la matriz caída no deja la publicación a medias— y que no sobre ninguno, que es
  el sentido que nadie pone: un adjunto con el nombre mal escrito no lo ve nadie hasta que alguien pulsa.
- **El renombrado de los instaladores de Windows cuesta una vez.** El de amd64 pasa de
  `-windows-instalador.exe` a `-windows-amd64-instalador.exe`, así que una copia instalada **antes** de esta
  versión no encontrará fichero y la banda la mandará a la página de descargas. Lo decidió el cliente con el
  coste delante: no hay instalaciones de Windows fuera de sus máquinas virtuales, y a cambio los nombres no
  mienten nunca más. Esta versión acepta **los dos** sufijos, que es lo que hace que el corte afecte solo a
  quien no se actualice antes de la fase 2.
- La receta de actualizar el VPS es `herramientas/actualizar-vps.sh`, y el `CLAUDE.md` la nombra en vez de
  llevar cinco órdenes con `amd64` dentro.
- `macos-latest` pasa a `macos-15` y `nfpm@latest` a una versión fija: una dependencia de la compilación no
  puede colgar de un blanco móvil, y a Apolo le retiraron un runner con apagones programados de por medio.

## Verificación

- **Los 28 nombres que el catálogo dice que están publicados, comparados uno a uno con los 28 adjuntos
  reales de la v2.61.3: idénticos.** Es la comprobación que convierte el catálogo en la fuente y no en una
  lista más.
- **Los seis objetivos por los cuatro gestores**, 24 casos, desde este VPS. La prueba de antes decía en su
  propio comentario que «solo se puede comprobar el sistema en el que corren los tests», y eso es lo que
  dejó sin ver el fallo del `.rpm`. **Tres mutaciones, tres rojas**: devolver el sufijo genérico `".deb"`
  —y canta «a un Linux ARM se le ofrece el de amd64, que dpkg rechazaría»—, ofrecerle el tar.gz a quien vino
  de `pacman`, y darle a un Windows ARM el instalador de amd64.
- **`TestNingunSufijoEsSufijoDeOtro`**, que es la forma estructural del mismo fallo: entre los adjuntos que
  se le pueden ofrecer a un equipo, ningún sufijo puede acabar en otro. Mutada poniendo `esfinge-{v}.deb`,
  roja.
- **Las dos tablas, contra un mundo de mentira donde ARM ya está publicado.** Esto hubo que arreglarlo
  durante la implementación: con el catálogo de hoy la mutación «filtra los arm64» **pasaba en verde**,
  porque hoy no hay ni un adjunto arm64 publicado. Es la lección de `soloLectura` otra vez —si una guarda
  tiene un caso que hoy no puede ocurrir, ese caso no lo prueba nadie— y se cerró igual: las tablas reciben
  el catálogo por parámetro y la prueba les da el mundo que hace falta.
- **La tabla del README se compara con la que genera el catálogo**, y de paso cazó al asistente cambiando un
  rótulo que lee el cliente —«Linux · cualquiera» por «Linux · a mano»— sin que nadie lo pidiera. Se devolvió.
- **La API de mentira del e2e**, cuyos nombres están escritos a mano en un `.mjs` y no se pueden juntar con
  los de Go: se comparan las dos listas entre sí, que es lo que manda esta casa cuando no se pueden juntar.
  Mutada con `-linux-x64.tar.gz`, roja.
- **Y el fichero del gestor**: los tres valores buenos, uno que no se reconoce —que **no se cree**— y la
  AppImage anunciándose por el entorno.

**Lo que no se puede comprobar aquí**: que el renombrado de Windows se vea de verdad en un Windows, y que la
banda mande a la página cuando no hay fichero para ese equipo. Eso lo dice el cliente en su máquina virtual.
