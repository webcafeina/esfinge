# ADR 0062 · Actualizarse sola en Linux, con `pkexec`

Fecha: 2026-11-16 · Estado: aceptada · Matiza la [ADR 0016](0016-reemplazarse-sola.md)

## Contexto

La 0016 dijo que en macOS y en Windows Esfinge se reemplaza a sí misma y se reinicia, y que
**en Linux no**, «porque el `.deb` instala como root». Lo que hacía la banda era bajar el
paquete, comprobar su SHA256 y entregárselo al sistema con `xdg-open`: se abre el Centro de
aplicaciones con el paquete cargado y la persona pulsa «Instalar» y teclea su contraseña.

Lo preguntó el cliente el 2026-11-16, con su Ubuntu 26.10 delante y después de actualizar a
mano con `apt`: *«Apolo, por ejemplo, si lo instalo con el deb se actualiza solo desde la
ventana, ¿esfinge también?»*. La respuesta era que no, y al mirar por qué, **la razón escrita
no se sostenía**: que el paquete instale como root no impide nada, porque para eso está
`pkexec`, que le pide la autorización al escritorio con su propio diálogo y ejecuta la orden
como root. No es una hipótesis: Apolo lo hace desde octubre y su `deuda.md` lo da por
comprobado **en esa misma máquina virtual** —«el `.deb` de la v0.9.1 pasó a la v0.10.0 desde
la banda, con `pkexec`, en Ubuntu 26.10 ARM64»—.

Con los repositorios APT y RPM ya montados (ADR 0061) había además un segundo camino posible
—decir «la actualiza tu gestor» y callarse, como ya se hace en Arch—, y el cliente eligió con
las dos opciones delante: *«Quiero que haga lo mismo que Apolo, que se actualice solo desde la
ventana, con la banda. Tanto si lo he instalado con deb (o el paquete que corresponda de
linux) como desde la terminal.»*

## Decisión

**En Linux, una copia instalada con un paquete se actualiza sola desde la banda**, igual que
en macOS y en Windows: el botón dice «Instalar y reiniciar», sale el diálogo de contraseña
del escritorio, el gestor de paquetes instala y Esfinge se vuelve a abrir.

- **Da igual cómo se instalara**: el `.deb` o el `.rpm` descargado a mano y el `apt install` /
  `dnf install` desde el repositorio acaban en lo mismo —una copia que el gestor conoce—, y el
  paquete descargado se instala encima en los dos casos sin dejar nada descuadrado.
- **La orden sale por tabla, del gestor y de lo que haya instalado**, prefiriendo el gestor de
  paquetes al instalador crudo: `apt-get install` antes que `dpkg -i`, y `dnf` (o `zypper` en
  openSUSE) antes que `rpm -U`. El motivo no es estético: una versión nueva puede traer una
  dependencia nueva, y `dpkg -i` la dejaría a medias con un «dependency problems» donde
  `apt-get` la resuelve.
- **Se instala con la ventana todavía abierta**, y sólo después se cierra y se vuelve a abrir.
  Al revés, el diálogo de contraseña saldría cuando Esfinge ya no está en pantalla, y un
  diálogo de contraseña huérfano es justo lo que un gestor de contraseñas enseña a no creerse.
- **Cancelar no es un fallo del programa y se dice con palabras**: `pkexec` devuelve 126 cuando
  nadie autoriza y 127 cuando no ha podido ni preguntar, y «exit status 126» no le dice nada a
  nadie.
- **Sin `pkexec`, o sin herramienta que reconozcamos, se cae a lo de siempre** —bajar el
  paquete y dárselo al sistema—, que es lo que había hasta hoy y sigue funcionando.

## Alternativas descartadas

**Decir «la actualiza tu gestor» y no ofrecer nada**, como en Arch. Es lo correcto allí porque
no publicamos un paquete suelto para pacman, pero aquí sí lo publicamos: obligaría a abrir una
terminal a quien ya tiene la aplicación abierta y mirando el aviso. El cliente lo descartó
explícitamente.

**Llamar a `apt`/`dnf` contra el repositorio** (`apt-get install --only-upgrade esfinge`) en
vez de instalar el fichero descargado. Suena más limpio y tiene dos fallos: no funciona para
quien se bajó el paquete suelto sin añadir el repositorio, y haría que la banda dependiera de
que el repositorio esté al día en ese momento, cuando lo que la banda ya tiene en el disco es
exactamente la versión que anuncia, con su suma comprobada.

**Hacer el `pkexec` desde el guion que reabre**, como el cambiazo de macOS. Se descartó por lo
de arriba: el diálogo saldría sin ventana detrás.

**Que la AppImage también se reemplace sola.** Queda fuera a propósito y sigue en `deuda.md`:
es otro camino —escribir junto a `$APPIMAGE` y renombrar encima— y la fase 4 se hizo aislada
justamente para no mezclar las dos cosas.

## Consecuencias

- La frase «Instalar y reiniciar» pasa a ser verdad en Linux, que es lo que la banda ya decía
  en los otros dos sistemas.
- **Esfinge ejecuta algo como root**, que es nuevo en Linux. Lo que se ejecuta es el gestor de
  paquetes del propio sistema sobre un fichero que Esfinge acaba de descargar **y cuya suma ha
  comprobado**; el permiso lo concede la persona en un diálogo del escritorio, no Esfinge.
- Quien no tenga `pkexec` —un escritorio muy pelado, o un contenedor— no nota ningún cambio.
- En Arch no cambia nada, y desde una AppImage tampoco.

## Verificación

**Comprobado aquí**: la tabla de la orden para los cuatro gestores y las seis combinaciones de
herramientas, los dos caminos de `ComoSeInstala` —paquete con `pkexec` → se instala sola;
AppImage, Arch, sin `pkexec` o sin paquete detrás → como antes—, que `Instalar` llama al gestor
**con el fichero recién descargado** y pide el relevo después, y que una autorización denegada
da un error con palabras y **no reabre nada**. Las dos mutaciones que importan —no exigir
`pkexec`, e instalar sin reabrir— ponen rojas esas pruebas.

**No comprobado todavía, y es lo que hay que mirar en una máquina**: que el diálogo de `pkexec`
salga de verdad sobre la ventana, que el gestor instale sin pedir nada más, y que Esfinge se
vuelva a abrir ya actualizada. Eso se ve en las máquinas virtuales de Ubuntu y de Fedora del
cliente, y hasta entonces esto es una pieza que pasa sus pruebas y no una funcionalidad vista.
Tampoco se ha comprobado el camino de openSUSE con `zypper`, que no hay máquina donde hacerlo.
