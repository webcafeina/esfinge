# Lo que hay dentro de Esfinge y no es nuestro

Esfinge dibuja sus iconos con los juegos nativos de cada escritorio, para que una
ventana de Esfinge se lea como una ventana de ese escritorio (ADR 0064 y 0070).
Esos dibujos **no son nuestros** y su licencia pide dos cosas: decir de quién son
y no quedárselos.

Este fichero es la primera; la segunda está en cómo se usan: **no se adapta
ninguno**. Las formas, sus atributos y sus transformaciones salen idénticas a las
del origen, y lo único que cambia es el envase —de SVG a datos, para poder
pintarlos sin cargar ficheros—, que no es una adaptación. Los cosechan
`herramientas/iconos/*.py` desde un commit fijado, y hay pruebas que comparan lo
que se pinta con lo que se cosechó.

La misma atribución está **dentro del programa**, en Ajustes → Acerca de: el
repositorio es público, pero quien recibe el programa no lee el repositorio.

## Iconos

### Fluent System Icons — Windows

- **De**: Microsoft Corporation
- **Licencia**: MIT
- **Dónde**: <https://github.com/microsoft/fluentui-system-icons>
- **Qué se usa**: 18 iconos simbólicos, en sus dos tamaños dibujados (16 y 20).
- **Dónde vive aquí**: `frontend/src/iconos-fluent.ts`, con el texto completo de
  la licencia en su cabecera.

### Yaru — GNOME y el resto de Linux

- **De**: Canonical Ltd. y los colaboradores de Yaru
- **Licencia**: CC BY-SA 4.0
- **Dónde**: <https://github.com/ubuntu/yaru>
- **Qué se usa**: 21 iconos simbólicos de `icons/Yaru/scalable`.
- **Dónde vive aquí**: `frontend/src/iconos-yaru.ts`, con la atribución en su
  cabecera.

**Y por qué importa que no se adapten**: CC BY-SA obliga a publicar **las
adaptaciones** con la misma licencia. Esfinge no es código abierto —se publica
para poder leerse y auditarse—, así que una adaptación de estos dibujos sería un
problema y una inclusión sin tocar nada no lo es. De ahí que el cosechador se
niegue a juntar trazados, a recortarlos y a reescalarlos, y que haya una prueba
que lo vigila.

## Lo que esto no cubre

Las bibliotecas de código —Go, React, Wails y las demás— van con sus propias
licencias y se leen en `go.mod` y en `frontend/package.json`. Aquí sólo está lo
que es **obra de alguien y viaja dibujada dentro del binario**, que es lo que no
se puede saber mirando un fichero de dependencias.
