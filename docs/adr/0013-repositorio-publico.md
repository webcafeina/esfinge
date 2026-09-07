# ADR 0013 — El repositorio es público, con licencia propietaria

**Fecha:** 2026-09-07 · **Estado:** aceptada

## Contexto

Las compilaciones se entregaban mandando un fichero a mano. Un repositorio privado no puede tener
descargas públicas: cualquiera que quiera bajarse la aplicación necesita cuenta e invitación.

## Decisión

Repositorio público, con las compilaciones publicadas en cada versión, y un fichero de licencia que
dice que el código se puede leer y auditar pero no reutilizar ni redistribuir sin permiso.

## Alternativas descartadas

- **Privado, invitando al cliente.** Le obliga a tener cuenta de GitHub e iniciar sesión. Para una
  persona es llevadero; para diez, un incordio.
- **Privado, siguiendo con el envío a mano.** Cero fricción para el cliente y un paso manual para
  siempre.
- **Licencia MIT.** Haría del repositorio una carta de presentación, a cambio de renunciar a
  controlar quién lo usa.
- **Sin fichero de licencia.** Legalmente equivale a todos los derechos reservados, pero mucha gente
  asume lo contrario al ver código público.

## Consecuencias

- **Que el código de una herramienta de cifrado se pueda auditar es una ventaja**, no una
  concesión: quien la use puede comprobar qué hace en vez de creérselo.
- Los minutos de compilación pasan a ser gratis e ilimitados, que en repositorio privado se contaban
  diez veces más caros en macOS.
- El correo de los commits, `info@webcafeina.com`, queda público. Es el de la empresa.
- **Esto no se deshace del todo**: aunque se volviera a privado, lo indexado o clonado ya está fuera.

## Verificación

`gitleaks` sobre el historial completo antes de cambiar la visibilidad: 11 commits, sin
filtraciones. Y queda en el flujo de integración continua para que siga corriendo después.
