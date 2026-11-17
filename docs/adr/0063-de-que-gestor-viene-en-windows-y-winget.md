# ADR 0063 · De qué gestor viene una copia de Windows, y winget

Fecha: 2026-11-17 · Estado: aceptada · Amplía la [ADR 0061](0061-las-arquitecturas-y-los-canales.md) y la [0062](0062-actualizarse-sola-en-linux.md)

## Contexto

Preguntando por los caminos de instalación en Windows —*«aparte del exe y el choco, ¿está
contemplado winget?»*— salieron dos cosas, y la segunda no estaba apuntada en ninguna parte.

**Una.** En Windows no había ningún camino más: el instalador `.exe` y Chocolatey. **winget** es
hoy el gestor que de verdad usa la gente en ese sistema, es gratis, y Apolo ya tiene un PR abierto
en `microsoft/winget-pkgs` parado únicamente en **firmar la CLA de Microsoft** — o sea, un paso del
cliente y no técnico.

**Dos, y es un agujero que ya teníamos.** En Linux, cada paquete deja escrito con qué gestor se
instaló (`/usr/share/esfinge/gestor`), y de ahí salen dos comportamientos: en Arch la banda dice «la
instala tu gestor» y se calla, y desde la 0062 en Debian y Fedora se instala sola con `pkexec`.
**En Windows no había nada equivalente**, así que a quien instalaba con `choco` se le ofrecía
igualmente actualizarse por su cuenta: Esfinge quedaba al día y **el gestor viendo una versión
vieja para siempre**. Con winget el problema sería el mismo, multiplicado por su número de usuarios.
Estaba apuntado como deuda en Apolo y aquí no estaba ni apuntado.

## Decisión

**Una copia de Windows dice de qué gestor viene, y winget se publica como un canal más.**

- **El instalador deja la marca cuando se lo piden.** El NSIS acepta `/GESTOR=<nombre>` y escribe
  ese nombre **junto al programa**; un doble clic no pasa nada y no deja fichero, que es
  exactamente lo que distingue una instalación a mano de una de un gestor. Chocolatey lo pasa en
  sus `silentArgs` y winget en `InstallerSwitches.Custom`.
- **Al lado del programa y no en un sitio compartido**, porque en Windows no lo hay: el instalador
  escribe en su propia carpeta y ahí es donde Esfinge sabe mirar sin inventarse rutas.
- **A quien viene de un gestor no se le ofrece nada**: ni descarga ni instalación, igual que en
  Arch. Su gestor es quien lo pone al día, y adelantarnos sería dejarlo descuadrado.
- **Y esa regla vive en Go, no en la pantalla.** Estaba escrita en TypeScript como «no hay fichero
  y el gestor es pacman», que es la misma regla en dos sitios — el patrón que en este proyecto ya
  ha costado seis fallos. Ahora Go contesta `loActualizaSuGestor` y la ventana pregunta.
- **winget se manda desde la publicación**, con los manifiestos en `empaquetado/winget/` rellenados
  por el mismo `herramientas/rellenar` que las otras recetas, y enviados con `wingetcreate`
  **fijado y con su suma**, como NSIS. Hacen falta dos cosas que son del cliente: **firmar la CLA
  de Microsoft** y un **fork de `microsoft/winget-pkgs`** con un testigo en `WINGET_TOKEN`.
- **Sin ese testigo el trabajo avisa y no falla**, y deja los manifiestos colgados como artefacto.
  Es lo mismo que se decidió con Chocolatey: lo que no está en nuestra mano no deja en rojo todas
  las publicaciones.

## Alternativas descartadas

**Mirar el registro de Windows** para saber si lo instaló un gestor. Ni Chocolatey ni winget dejan
una marca propia que se pueda leer sin adivinar; lo que sí controlamos es qué le pasa el paquete a
nuestro instalador.

**Dejar que Esfinge se actualice igual y que el gestor se arregle solo.** No se arregla: `choco
upgrade` seguiría creyendo que tiene la vieja, y en el mejor caso reinstala encima. El resultado es
una máquina con dos versiones de la verdad.

**Publicar winget con la acción que genera los manifiestos sola** (`winget-releaser`). Los genera a
partir de la publicación y **no sabe nada de `/GESTOR=winget`**, que es justo lo que hace falta
aquí. Con nuestros manifiestos en el repositorio, lo que se manda es lo que se lee.

**La Microsoft Store.** Exige empaquetar como MSIX y una cuenta de desarrollador; es otra
conversación y no se cierra con esto.

**Scoop**, que sería el equivalente barato del tap de Homebrew —un repositorio nuestro, sin CLA ni
revisión—. No se descarta por malo: queda fuera de esta ficha y apuntado en `deuda.md`.

## Consecuencias

- Quien instale con `choco` o con `winget` **verá que hay versión nueva y que la instala su
  gestor**, sin un botón que descuadre su equipo.
- **Esa frase depende de que el paquete pase el parámetro**: si un día se publica un paquete que no
  lo pasa, su gente volverá a ver la oferta de actualización. Lo vigila la prueba de la tabla, que
  cubre los cinco gestores.
- **winget tiene revisión, como Chocolatey**: una versión puede tardar en aparecer, y eso no es un
  fallo nuestro.
- Y hay un efecto que conviene decir: **ahora hay dos sitios donde se escribe el gestor** —el
  `postinstall` en Linux y el instalador en Windows—, pero **un solo sitio donde se lee y decide**.

## Verificación

**Comprobado aquí**: la tabla entera de `gestorSegun` —los cinco gestores, las dos ubicaciones y
los cuatro casos de «no viene de ninguno»— con el mundo por parámetro, de modo que lo de Windows se
comprueba desde este VPS. Y `LoActualizaSuGestor` para los siete valores posibles. Dos mutaciones
las ponen rojas: no recortar el salto de línea que escribe el `postinstall`, y partir la ruta de
Windows con el separador de esta máquina. **La primera destapó un falso verde que ya estaba ahí**:
el caso del `.deb` pasaba por el camino de «vivo en /usr/bin» y no por el fichero.

Los manifiestos de winget se rellenan con el catálogo y con las sumas de la publicación de verdad,
y la prueba de recetas exige que cada `{{NOMBRE:…}}` nombre una clave publicada.

**No comprobado, y hace falta una máquina**: que el instalador escriba el fichero al recibir
`/GESTOR=`, que `choco install` lo pase, y que con eso la banda diga lo que debe. Eso se ve en la VM
de Windows 11 ARM del cliente. **Y nada de winget se ha probado todavía**: el PR no se puede abrir
hasta que la CLA esté firmada y exista el fork.
