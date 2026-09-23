# ADR 0041 — Los papeles de la cuenta: quién revisa los textos y con qué correo

**Fecha:** 2026-09-23 · **Estado:** aceptada, sin empezar · **Continúa la [0035](0035-las-cuentas.md)** ·
**Revisar cuando** el uso acerque el correo a su tope, o cuando entre una cuenta de fuera de la UE

## Contexto

Abrir el registro (la entrega A4 del plan) no es escribir código: es un cambio de variable en el Worker.
Lo que lo sostiene son papeles y cupos, y ninguna de las dos cosas estaba decidida:

- La **política de privacidad** existe desde la A2 —en `web/privacidad.html` y, en corto, en el aviso del
  panel de la extensión—, está escrita con los hechos del código y **no la ha mirado nadie más**. Las
  **condiciones de uso no existen**.
- El **correo** lo manda Resend. Su plan gratuito da **cien correos al día**, y cada alta, cada equipo
  nuevo, cada recuperación y cada cambio de contraseña gastan uno.

Con el registro por invitación —la casa y poco más— nada de esto aprieta. Al abrirlo, las dos cosas pasan a
ser el límite real del servicio.

## Decisión

### Los textos los revisa un modelo especializado, no un abogado

Lo decidió el cliente el 2026-09-23, con estas palabras: «la revisión legal también será con un modelo
especializado». Alcance de esa revisión, para que se sepa qué se está comprando:

- **Que lo que dicen los textos sea lo que hace el código.** Esto es lo que más vale y es comprobable: cada
  frase de la política se contrasta con el servidor, la aplicación y la extensión, y lo que no cuadre se
  corrige en el texto o en el código.
- **Que estén los puntos que el RGPD exige** en una información de tratamiento —quién responde, qué datos,
  para qué, con qué base, cuánto duran, quién los trata por encargo, dónde están y qué derechos hay— y que
  las condiciones de uso digan lo que este servicio tiene de particular: **que sin la contraseña maestra y
  sin la clave de recuperación no hay forma de recuperar nada**, que Webcafeína no puede leer la bóveda ni
  devolverla, y qué pasa al darse de baja.
- **Lo que no da**: no es asesoramiento jurídico y no responde nadie de él. Un despacho pone su firma y su
  responsabilidad detrás; esto pone el cuidado de leer y la ventaja de conocer el código por dentro. Queda
  dicho aquí para que la diferencia sea una elección y no un olvido.

### Resend se queda en el plan gratuito

«De momento en su plan gratuito, más adelante según los usos que tengan plantearemos alternativas»
(cliente, 2026-09-23). Consecuencia directa, y es la parte que sí es técnica: **los topes del servidor
tienen que caber en el del correo**. Hoy no caben.

- `TOPE_ALTAS_DIA` viene en **200**, el doble de lo que Resend deja mandar, y encima las altas no son los
  únicos correos del día.
- Cuando Resend contesta que no, `mandarOFallar` devuelve un 502 con «No se ha podido mandar el correo.
  **Prueba otra vez en un momento**». Con el cupo del día agotado esa frase **no es verdad**: no es en un
  momento, es mañana.

Las dos cosas —bajar los topes y decir la verdad en ese mensaje— se hacen **antes** de abrir el registro,
no después. Mientras sea por invitación no hace falta tocar nada.

## Alternativas descartadas

- **Un despacho.** Es lo evidente en un servicio que guarda datos de terceros, y da lo único que una
  revisión así no puede dar: alguien que responde. El cliente decidió que no, y lo que se hace en su lugar
  queda escrito arriba con sus límites.
- **Pasar ya al plan de pago de Resend.** Costaría poco y quitaría el cupo de en medio, pero es pagar por
  un uso que todavía no existe. Se revisa cuando el uso se acerque al tope, que es lo que dice la cabecera
  de esta ficha.
- **Buscar otro proveedor de correo** ahora. Mismo argumento, y además el remitente ya está verificado con
  SPF, DKIM y DMARC en `PASS`: cambiarlo es volver a hacer ese trabajo sin necesidad.
- **Mandar el correo desde el propio servidor.** Un Worker no puede, y montar un servidor de correo propio
  para cuatro códigos al día es cambiar un problema pequeño por uno grande —reputación de IP, listas
  negras, entregabilidad—.

## Consecuencias

- **La A4 pasa a tener cuatro cosas por delante**, todas fuera del código salvo la última: los textos
  revisados y publicados, los encargos de tratamiento firmados con Cloudflare y con Resend —papeles que
  firma el cliente en cada panel, y que hoy no constan—, los topes diarios por debajo de cien, y el mensaje
  del 502 arreglado.
- **El cupo del correo es el límite de crecimiento del servicio**, y conviene decirlo en voz alta: cien
  correos al día son, como mucho, del orden de cincuenta o sesenta altas diarias contando lo demás. El día
  que eso se quede corto, la alternativa es el plan de pago y no bajar más los topes.
- **Quien lea la política verá que la revisó Webcafeína y no un tercero.** No se va a decir lo contrario en
  ninguna parte.

## Verificación

**Comprobado**: que el plan gratuito de Resend se agota con un rechazo de su API y que el servidor lo
convierte hoy en un 502 con una frase que no encaja con ese caso (leído en `servidor/src/correo.ts` y
`indice.ts`, 2026-09-23). Y que `TOPE_ALTAS_DIA` vale 200 por defecto y solo lo sube el entorno `local`.

**Sin comprobar**: cuántos correos al día gasta de verdad una cuenta nueva —hay que mirarlo con uso real,
no estimarlo—, y si Resend cuenta el cupo por día natural o por ventana móvil, que cambia cuánto margen
hay que dejar. Tampoco se ha mirado todavía la Ley 1581 de 2012 de Colombia, que la [ADR 0035](0035-las-cuentas.md)
dejó pendiente para el día que entre alguien de allí.
