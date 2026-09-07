# ADR 0016 — Esfinge se reemplaza a sí misma y se reinicia

**Fecha:** 2026-09-07 · **Estado:** aceptada · **Sustituye** la parte de la
[0014](0014-comprobacion-de-actualizaciones.md) que descartaba el reemplazo automático

## Contexto

La [0014](0014-comprobacion-de-actualizaciones.md) dejó a Esfinge descargando su propia
actualización y entregándosela al instalador del sistema. En macOS eso significa que, después de
esperar la descarga, se monta la imagen y aparece la ventana de arrastrar a Aplicaciones: **el
trabajo se lo acaba haciendo el usuario**, que es justo lo que se quería quitar. Probado en un Mac,
el veredicto fue inmediato: «este proceso no lo debería hacer el usuario».

Aquella decisión daba por hecho que reemplazarse exige firmar con Apple. **Era demasiado
conservadora, y estaba mal.** La firma de Apple hace falta para que no salte Gatekeeper en algo que
descarga un navegador; no para que un proceso sustituya su propio paquete. Lo que de verdad hace
falta es permiso de escritura donde vive la aplicación —que en `/Applications` lo tiene cualquier
administrador— y hacer el cambiazo **desde fuera del proceso**, porque un programa no puede
reemplazarse mientras corre.

## Decisión

Donde se puede, Esfinge se actualiza sola: descarga, sustituye y vuelve a abrirse. El botón dice
**«Instalar y reiniciar»** y no hay nada que arrastrar.

Lo hace un guion que Esfinge escribe, lanza suelto —fuera de su grupo de procesos, para que cerrar la
ventana no se lo lleve por delante— y que entonces espera a que este proceso muera antes de tocar
nada.

- **macOS**: monta el DMG, saca el `.app` con `ditto` —que conserva atributos y firma, cosa que `cp`
  no—, le quita la cuarentena, cambia el paquete y vuelve a abrirlo.
- **Windows**: lanza el instalador NSIS en silencio (`/S`) y reabre la aplicación. Seguirá pidiendo
  permiso de administrador, que es del sistema y no hay forma de saltárselo mientras la instalación
  sea para toda la máquina.
- **Linux**: no. El `.deb` instala como root, y pedir la contraseña desde la aplicación para hacer
  lo que el gestor de paquetes ya sabe hacer sería peor. Ahí el botón sigue diciendo «Abrir el
  instalador».

**Quién decide no es el sistema, es si se puede escribir donde vive la aplicación**, y eso se
comprueba escribiendo: los permisos de `Stat` no valen —en macOS hay listas de control de acceso— y
una copia en una carpeta ajena tiene que caer en el camino del instalador aunque esté en un Mac.

## Alternativas descartadas

- **Dejarlo como estaba.** Es lo que motivó esta ficha.
- **Sparkle**, el marco de actualización de siempre en macOS. Resuelve esto y más, pero es una
  dependencia grande de Objective-C para una aplicación de dos pantallas, y su modo seguro se apoya
  en firmas EdDSA que habría que gestionar aparte.
- **Pedir la contraseña de administrador en Linux** para escribir en `/usr/bin`. Una herramienta que
  cifra no debería acostumbrar a nadie a darle la contraseña de root.

## Consecuencias

- **El momento del cambiazo es el único punto del programa donde se puede perder algo**: si fallara
  a medias, quedaría una máquina sin Esfinge. Por eso el guion no borra lo viejo hasta que lo nuevo
  está en su sitio, comprueba que lo descargado es una aplicación de verdad antes de tocar la que
  funciona, y si la copia falla devuelve lo viejo a donde estaba.
- La interfaz tiene que decir la verdad en cada máquina, así que la novedad viaja con **cómo se va a
  instalar** y el botón cambia según eso.
- `Sistema` gana un método `Cerrar()`: la aplicación necesita poder cerrarse a sí misma, que antes
  no hacía falta.

## Verificación

- Tests de que se puede escribir se decide escribiendo, incluida una carpeta de solo lectura; de que
  fuera de un paquete `.app` se dice que no hay nada que reemplazar; y de que en Linux se va al
  instalador del sistema.
- Compila para los tres sistemas, que aquí es lo único que se puede afirmar del código específico de
  cada uno.

**Lo que no se ha comprobado, y es la parte que importa:** el cambiazo de verdad. No hay Mac ni
Windows en esta máquina. Falta ver que el paquete se reemplaza, que la aplicación vuelve a abrirse
sola, y si al haber descargado el DMG desde Go la copia nueva se libra del aviso de Gatekeeper.
