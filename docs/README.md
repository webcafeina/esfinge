# Documentación de Esfinge

Sistema de contexto vivo: lo que hace falta para volver al proyecto después de semanas y entender
dónde está y por qué está así. Todo en español, fechas en formato `AAAA-MM-DD`.

| Documento | Qué es | Cuándo se actualiza |
|---|---|---|
| [estado.md](estado.md) | Dónde estamos hoy y cuál es la siguiente acción concreta | Al cerrar cada sesión |
| [siguiente.md](siguiente.md) | Lo que viene, por prioridad | Al decidir qué se hace después, y al cerrar algo |
| [decisiones.md](decisiones.md) | Índice de decisiones, con su estado | Al añadir o superar una decisión |
| [adr/](adr/) | Una ficha por decisión no trivial | Al tomar una decisión que costaría volver a discutir |
| [deuda.md](deuda.md) | Lo que sabemos que está a medias o mal | Al descubrir deuda, y al saldarla |
| [sesiones.md](sesiones.md) | Bitácora: qué se hizo en cada sesión | Al cerrar cada sesión |
| [seguridad.md](seguridad.md) | Qué protege Esfinge y qué no | Al tocar el cifrado, el historial o lo que se guarda |
| [revision-2026-09.md](revision-2026-09.md) | La revisión de seguridad hecha aquí: hallazgos, lo arreglado y lo pendiente | Al arreglar algo de la lista o al hacer otra revisión |
| [revision-textos-2026-09.md](revision-textos-2026-09.md) | La política y las condiciones contrastadas con el código, y lo que falta | Al tocar lo que se guarda o lo que dicen los textos |
| [cuentas.md](cuentas.md) | El plan de las cuentas, por entregas, con lo hecho marcado | Al cerrar una entrega o cambiar el plan |
| [passkeys.md](passkeys.md) | Lo que haría falta para que Esfinge guarde y use passkeys, y qué hay que decidir antes | Al decidir algo de esa fase o al empezarla |
| [../CLAUDE.md](../CLAUDE.md) | Cómo se trabaja aquí: protocolo, convenciones, trampas | Al cambiar una convención o encontrar una trampa nueva |
| [../README.md](../README.md) | La portada, para quien llega de fuera | Al cambiar lo que se ve o cómo se descarga |

## Por dónde empezar

Si vuelves después de tiempo: **[estado.md](estado.md)** primero, y de ahí a
**[siguiente.md](siguiente.md)**. Si te encuentras algo raro en el código y quieres saber por qué
está así, busca en **[decisiones.md](decisiones.md)**; si no está, probablemente sea deuda y esté en
**[deuda.md](deuda.md)**.

## Cómo se mantiene

Con disciplina, no con un control automático. Es lo mismo que hacen Tempero y webcafeína-local, y
por el mismo motivo: un validador de documentación se convierte en algo que se sortea, y lo que hay
que sostener aquí es el hábito de escribir lo que se decide cuando se decide.

Lo que se cierra **no se borra**: se tacha y se queda, con la fecha. Saber qué se descartó y cuándo
vale tanto como saber qué se hizo.
