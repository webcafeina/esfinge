# ADR 0010 — El historial guarda qué y cuándo, nunca el contenido

**Fecha:** 2026-09-07 · **Estado:** aceptada

## Contexto

La aplicación necesitaba un historial para reencontrar un fichero cifrado hace dos semanas. Un
historial es un fichero en disco, y en una herramienta que cifra eso hay que pensarlo dos veces.

## Decisión

Se guardan la acción, el nombre del fichero, el destino y la fecha. **Nunca** el contenido, la clave
ni el contenedor cifrado. Vive en la carpeta de configuración del usuario con permisos `600`, tiene
tope de 200 entradas y un botón de vaciar bien visible. La propia pantalla enseña la ruta del
fichero, para que no haya que fiarse de la palabra de nadie.

## Alternativas descartadas

- **Guardar también el contenedor cifrado**, para poder recuperarlo si se cerró la ventana sin
  copiarlo. Sin la clave ese texto no se abre, pero convierte un fichero de conveniencia en un
  objetivo que merece la pena robar.
- **No guardar nada en disco.** Lo más seguro y lo menos útil: el historial moriría al cerrar.

## Consecuencias

- El historial es poco: para eso está.
- Sigue revelando **qué ficheros se han cifrado y cuándo**, que en algunos contextos ya es
  información. Está dicho en [seguridad.md](../seguridad.md).
- Un historial ilegible o corrupto no impide arrancar: se empieza uno nuevo.

## Verificación

`TestElHistorialNoGuardaSecretos` cifra un texto y comprueba que ni el secreto, ni la clave, ni el
contenedor aparecen en el fichero del disco, y que sus permisos son `600`.
