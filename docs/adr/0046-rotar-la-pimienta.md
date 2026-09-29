# ADR 0046 — Rotar la pimienta sin dejar a nadie fuera

**Fecha:** 2026-09-29 · **Estado:** aceptada y escrita; **sin rotar todavía** —la pimienta de producción
sigue siendo la 1— · **Salda una deuda vencida de la [0036](0036-el-servidor-de-cuentas.md)** ·
**Revisar cuando** haya que rotar de verdad, que es la primera vez que esto se ejercita fuera de las
pruebas

## Contexto

El servidor no guarda la clave de acceso: guarda `HMAC(PIMIENTA, "acceso|<clave>")`. La pimienta vive
solo en el entorno del Worker, nunca en la base, y es lo que hace que **robar la base entera no sirva
de nada**: sin ella no se pueden probar candidatas contra los verificadores.

Hasta hoy, **cambiarla dejaba fuera a todas las cuentas**. La [ADR 0036](0036-el-servidor-de-cuentas.md)
lo dejó escrito y dejó puesto el gancho —cada cuenta guardaba `pimienta: 1`— pero **nadie leía ese
campo**: el código de rotación no existía.

Una pimienta que no se puede rotar es **un secreto al que no se puede responder**. Si se filtra, la
respuesta correcta es cambiarla ya, y la respuesta real era «no puedo, eso cierra el servicio». La
condición escrita en [`../deuda.md`](../deuda.md) era hacerlo **antes de abrir el registro**, y el
registro se abrió el 2026-09-23 sin ello: la deuda llevaba seis días vencida.

## Decisión

**La versión de la pimienta va dentro de cada verificador** —`2:abc123…`—, y lo que no lleva prefijo se
lee como la versión 1.

No al lado, en un campo de la cuenta, y la razón es la que decidió el diseño: **una cuenta tiene dos
verificadores, el de acceso y el de posesión, y no migran a la vez**. Al entrar se tiene la clave de
acceso pero no la de posesión; al sincronizar, al revés. Con una sola marca por cuenta, la primera
migración mentiría sobre la otra, y el contador que decide cuándo se puede borrar la pimienta vieja
estaría mal justo en el sentido peligroso.

**Cada cuenta migra sola en cuanto se usa**: al entrar reescribe su verificador de acceso, y en
cualquier pasada de la sincronización —cada cinco minutos con la aplicación abierta— el de posesión.
**El servidor no puede migrarlas él solo**, y eso no es pereza: tendría que poder calcular el
verificador nuevo, y no puede, porque no conoce la clave. Es exactamente lo que hace útil a la pimienta.

**Conviven dos generaciones y no más.** Quien se quede dos por detrás no se puede verificar, y eso
**no se contesta como «contraseña incorrecta»**: es un `409` con otras palabras. Decirle a alguien que
su contraseña falla cuando no falla es el peor mensaje posible — se pone a probar contraseñas y a dudar
de su gestor de contraseñas.

**Y se puede contar.** La tabla `cuentas` de D1 guarda la versión **menor** de cada cuenta, así que
`SELECT pimienta, COUNT(*) FROM cuentas GROUP BY pimienta` dice cuántas quedan atrás. Borrar la pimienta
anterior deja de ser una fecha a ciegas y pasa a ser una decisión con el número delante.

## Alternativas descartadas

- **Las dos pimientas para siempre.** Lo más simple y no deja fuera a nadie jamás. Se descartó por lo
  que cuesta: **rotar no llegaría a cerrar el agujero**. Si la vieja se filtró, sigue sirviendo contra
  cualquier cuenta que no haya vuelto, y no habría forma de saber cuántas son ni cuándo deja de ser
  verdad. Rotar sería un gesto y no un remedio.
- **Una fecha fija: la vieja muere a los noventa días.** Cierra el agujero y es poco código, pero la
  fecha se elegiría sin saber a cuántos afecta, y quien no conectara antes quedaría fuera del servidor.
  Se descartó **porque la información se puede tener**, y tenerla cuesta una columna.
- **Migrar todas las cuentas de golpe al rotar.** No se puede, por diseño. Queda escrito para que nadie
  lo proponga otra vez creyendo que se olvidó.
- **Una marca de versión por cuenta** en vez de por verificador. Es lo que la 0036 dejó preparado
  (`pimienta: 1`), y no vale: los dos verificadores migran en momentos distintos.

## Consecuencias

- **Rotar pasa a ser tres variables**: `PIMIENTA_ANTERIOR` ← la de ahora, `PIMIENTA` ← una nueva,
  `PIMIENTA_VERSION` ← el número siguiente. Sin ninguna de las dos últimas, todo es la versión 1 y no
  cambia nada — **lo desplegado hoy no se entera**.
- **No se rota dos veces seguidas.** Mientras el contador no diga cero, rotar otra vez deja colgadas a
  las que faltaban. Está dicho en `src/pimienta.ts`, en el LÉEME y aquí, y el código **contesta nulo**
  en vez de aceptar a la tercera generación, que es lo que convertiría el descuido en silencio.
- **Una escritura más en D1** al migrar una cuenta. Va **después** de guardar en el objeto, y si falla
  solo se registra: lo que queda apuntado es una versión más vieja de la real, que es el lado seguro —se
  espera de más antes de borrar la pimienta anterior, en vez de borrarla creyendo que no queda nadie—.
- **Los códigos de seis cifras no migran**: se emiten con la de ahora y se aceptan las dos mientras dure
  el solape. Caducan en diez minutos.
- **Y lo que esto no arregla**: si la pimienta se filtra, rotar cierra la puerta **hacia adelante**. Lo
  que un atacante ya se hubiera llevado —la base y la pimienta vieja a la vez— sigue sirviéndole para
  atacar sin conexión los verificadores que copió. Rotar limita el daño, no lo deshace.

## Verificación

**Comprobado aquí**, con doce pruebas dentro de `workerd`:

- Un verificador **sin prefijo** —todo lo escrito antes de esto— se lee como la versión 1, así que
  desplegar esto **no reescribe nada**.
- Una cuenta creada antes de rotar **entra con su contraseña de siempre después de rotar**. Es la prueba
  que justifica la ficha entera.
- **Y se migra al entrar**: quitada después la pimienta vieja, sigue entrando.
- **Dos generaciones por detrás da `409`** con un mensaje que no habla de contraseñas.
- **Un código emitido justo antes de rotar sigue valiendo** sus diez minutos.
- **D1 apunta la versión menor**: tras migrar solo el de acceso, sigue diciendo 1. Ésa es la prueba de
  que una marca por cuenta no habría valido.
- Y `env` mutado en las pruebas **llega al Durable Object**, que era la duda técnica del montaje. Se
  resolvió ejecutándolo, no leyendo la documentación del motor de pruebas.

**Las cinco mutaciones se ponen rojas**: que solo valga la pimienta de ahora, que no se reescriba al
entrar, que «imposible» se trate como «no», que los códigos usen solo la de ahora, y que D1 apunte la
mayor en vez de la menor.

**Lo que no se ha comprobado, y hay que decirlo:**

- **No se ha rotado nunca de verdad.** La pimienta de producción sigue siendo la 1, y lo primero que se
  ejercita fuera de las pruebas será la primera rotación real.
- **El contador no se ha visto llegar a cero** con cuentas reales, porque no ha habido rotación.
- **Cuánto tarda una cuenta en migrar en la práctica** es una deducción del código —la sincronización
  toca la posesión cada cinco minutos—, no una medida.
