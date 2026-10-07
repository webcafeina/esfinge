# 0053 · Quitar el acceso borra la copia del otro equipo

**Fecha:** 2026-10-07 · **Estado:** aceptada, hecha, desplegada y **publicada en la 2.42.0** · **comprobada en los dos Macs del cliente** el mismo día

## Contexto

La [ADR 0052](0052-dar-acceso-a-un-proyecto.md) dejó funcionando quitar el acceso a una bóveda de proyecto
—comprobado el 2026-10-07 en los dos Macs del cliente— y dejó escrito, en «Lo que no se hace», esto:

> **Borrado remoto** de lo que ya se bajó: **no se puede prometer**, y media promesa aquí es peor que
> ninguna.

El cliente lo probó y preguntó lo evidente: *«¿en Dashlane u otros servicios se pueden quitar de otra cuenta
una credencial/bóveda que se le haya compartido? ¿O también se le tiene que quedar sí o sí?»*. Se
investigó, y la respuesta cambia la decisión:

- **Dashlane borra.** Con los permisos «Can view», «Can edit» y «Can autofill», al revocar *«no verán el
  elemento la próxima vez que inicien sesión»*, y **no se avisa** a quien lo pierde.
- **Bitwarden borra, y dice la letra pequeña**, que es exactamente la que importa aquí: el elemento
  desaparece de la colección, pero *«los dispositivos sin conexión guardan una copia de solo lectura…
  algunos clientes pueden retener el acceso a esos datos durante un tiempo»*, y añade: *«si prevés un uso
  malicioso, **cambia las credenciales** a las que ese miembro tenía acceso»*.
- **1Password, cuando lo que se comparte es un enlace, no borra**: *«se crea una copia del elemento»*, y su
  documentación dice que *«para asegurarte de que el destinatario no tiene los datos después de dejar de
  compartir, cámbialos»*.

O sea que **el borrado de todos ellos es cooperativo, no criptográfico**: el servidor deja de servir y el
cliente obedece borrando su caché. Y ahí está el error de la frase de la 0052: juntó dos cosas que son
distintas. **No se puede prometer** es verdad. **No se puede hacer** no lo era, y por eso la 0052 no hizo la
mitad que sí se puede hacer: Esfinge cierra la puerta en el servidor —el 403— y **rota la clave**, que los
demás no hacen, pero nadie le dice al Esfinge del otro que borre el fichero que ya tiene en el disco.

Entonces lo que pasa hoy es lo que el cliente vio: la bóveda de un cliente suyo **se queda abierta y
legible** en el equipo de quien ya no trabaja con él, para siempre, a menos que esa persona la borre a mano.

## Decisión

**Quitar el acceso borra la bóveda en el equipo de quien lo pierde, y su pantalla se lo dice.** Lo eligió el
cliente el 2026-10-07 con las tres opciones delante, y eligió además la variante que **Dashlane no hace**:
avisar. Con bóvedas de clientes dentro, una que desaparece en silencio asusta más que una que explica por
qué se ha ido.

| | |
|---|---|
| **Qué se borra** | El fichero de esa bóveda y su memoria de sincronización. **Su fila en la lista no: se tacha** |
| **Cuándo** | En la primera pasada que reciba la respuesta del servidor diciendo que ya no hay acceso |
| **Lo que estuviera escrito y sin subir** | **Se va con ella**, como Dashlane. Lo decidió el cliente con el coste delante |
| **Si estaba abierta en ese momento** | Se cierra y se vuelve a la lista, con el aviso |
| **El aviso** | Se queda hasta que se cierra, y dice **quién** la compartía y que ha sido el dueño quien ha quitado el acceso |
| **Dónde pasa** | **En la ventana y en la extensión.** Son dos copias distintas —un fichero y `storage.local`—, y borrar solo una deja la bóveda entera en la otra. La extensión **no tacha la fila**: eso es de la ventana |

**Y se dice lo que esto no es, en la pantalla de quien revoca y con estas palabras**, igual que lo de Touch
ID y lo del permiso que impone el servidor:

> Desaparecerá de su equipo la próxima vez que abra Esfinge. **Si no lo abre, no desaparece**, y esto no
> protege de quien haya querido guardarse una copia: si la contraseña importa, cámbiala.

## Y el aviso no es un sitio nuevo: es la fila tachada

Lo primero que se pensó para el aviso fue lo evidente —apuntarlo al lado de las preferencias, que son
locales como la ranura de Touch ID— y **no se puede**, por dos razones que aparecieron al ir a escribirlo:

1. **Ahí acabaría el nombre de un proyecto de un cliente, en claro y fuera de la bóveda.** Es exactamente lo
   que la regla del historial prohíbe: un fichero sin cifrar que diga «Beta Industrial» es una señal de
   tráfico apuntando a lo que alguien tenía. Y el aviso **tiene que decir cuál**, porque sin el nombre no
   dice nada.
2. **Y las preferencias no aguantan una lista.** `GuardarPreferencias` recibe el objeto entero, así que un
   guardado a medias —que es lo que hace una prueba a propósito— la borraría en silencio. Eso ya está
   escrito en `CLAUDE.md` para los números; para una lista es peor.

Así que **la fila de la compartida no se quita: se tacha**, con la fecha, dentro del cuerpo cifrado de la
bóveda personal donde ya vive. Es el protocolo de la casa aplicado a los datos —*«lo que se cierra no se
borra: se tacha y se queda, con la fecha»*— y sale gratis lo que hacía falta: el nombre sigue cifrado, el
aviso **viaja a los dos equipos del titular** —que es lo correcto, porque los dos van a borrar su fichero— y
«Quitarla de la lista» ya existe, es `DejarDeVerCompartida`.

## La pieza que hace que esto no sea peligroso

**No se borra por un código de estado: se borra porque el servidor lo dice.**

Hoy `cuenta.SinAcceso(err)` es «el servidor ha contestado 403», y **con eso no se puede borrar nada**, por
dos razones que no son teóricas:

1. **Ese 403 tiene dos causas.** El Worker contesta 403 con «Ya no tienes acceso a esa bóveda.» cuando no
   eres titular, y con «En esta bóveda solo puedes ver.» cuando lo eres y has intentado subir. Borrar por lo
   segundo **destruiría la bóveda de un cliente porque alguien con permiso de solo ver intentó guardar**.
   Distinguirlas por el texto del mensaje sería atar un borrado irreversible a una cadena en español.
2. **Un 403 puede no venir del Worker.** El Worker de pruebas está detrás de Cloudflare Access, que contesta
   a `/_pruebas/*` con HTML — eso ya costó una vez el «invalid character '<'»—, y cualquier portero, proxy o
   WAF que alguien ponga delante puede devolver 403 por su cuenta. Con la regla «403 y borro», **un portero
   mal configurado borra bóvedas en todos los equipos a la vez**, y eso no tiene vuelta.

Así que el Worker manda un **código estable en el cuerpo** —`{"codigo":"revocado"}`— y el cliente borra
**solo** con ese código, dentro de un JSON que ha podido leer. Un 403 sin ese código es lo de siempre: se
deja de sincronizar y se dice, sin tocar el disco.

**Y solo cuenta el de bajar.** Un `GET` es lo único que todo titular puede hacer, con cualquiera de los dos
permisos; si bajar da «revocado», no hay interpretación posible. El 403 de subir no borra nunca, porque ahí
«no puedes» puede significar «solo puedes ver».

## Alternativas descartadas

- **Borrar con el 403 a secas**, que es lo que parece que esto es. Descartado arriba: son dos causas y el
  403 puede no ser del Worker.
- **Preguntar antes de borrar** —«te han quitado el acceso, ¿la quito de aquí?»—. Es lo prudente y el
  cliente lo descartó al elegir: deja la bóveda del cliente en el equipo de quien ya no trabaja con él
  justamente en el caso en que a esa persona no le apetezca pulsar. Y con el borrado protegido por el código
  del servidor, el accidente que la pregunta evitaba ya no existe.
- **Guardarle lo que no subió** —ni como aviso para que lo copie, ni como contenedor `ESF1` aparte—. Las dos
  se le ofrecieron y eligió lo de Dashlane. Queda dicho que **eso destruye trabajo que la otra persona hizo
  cuando sí tenía permiso**.
- **Avisar al que revoca de que allí se ha borrado.** Haría falta un acuse del equipo del otro, o sea
  decirle al dueño cuándo abrió Esfinge esa persona. Es telemetría sobre alguien, y por una confirmación que
  no cambia nada de lo que puede hacer.
- **No hacer nada, que es lo que dice hoy la 0052.** Es más honesto con la promesa y peor con el encargo:
  entre «no insinuar una garantía» y «no dejar la bóveda de un cliente en un equipo ajeno», se puede tener
  las dos cosas si se dice con claridad lo que el borrado es. Eso es lo que hacen Bitwarden y 1Password.

## Consecuencias

- **La frase de la 0052 se corrige**, no se deroga: lo que no se puede es **prometer** el borrado.
- **Un equipo que no vuelve a abrir Esfinge conserva la bóveda**, y es la mitad que no se puede cerrar. Va
  dicho en la pantalla.
- **Quitar una compartida ya es representable en la fusión** —`fundirCompartidas` va a tres bandas contra la
  base—, así que no hace falta inventar una lápida como con las ranuras. Hay que comprobarlo con dos
  equipos del mismo titular: sin base, la fusión conserva, y entonces el segundo equipo podría resucitar la
  fila.
- **Y rotar la clave sigue siendo lo que de verdad aguanta.** El borrado es cooperativo; la rotación es lo
  que hace que una copia vieja del fichero no sirva para leer nada de lo que venga después.

## Verificación

Lo que hay que poder decir al acabar, y cada prueba mutada:

| Qué protege | Prueba |
|---|---|
| Se borra al revocar | Dos cuentas con el Worker de verdad en local: dar acceso, bajar, revocar, y que el fichero **no esté** en el disco del otro y su fila no esté en la lista |
| **Un «solo puedes ver» no borra nada** | Titular con permiso de ver que intenta subir: 403, y el fichero **sigue ahí**. **Mutando el código por el estado**, esta prueba tiene que ponerse roja |
| **Un 403 que no es del Worker no borra nada** | Un 403 con cuerpo que no es JSON —el HTML de un portero— y otro con JSON sin `codigo`: se deja de sincronizar y **no se toca el disco** |
| El aviso se queda | Tras el borrado, la pantalla dice de quién era y por qué se ha ido, y sigue diciéndolo al volver a entrar |
| La bóveda abierta se cierra | Revocar con esa bóveda abierta deja la ventana en la lista, con la personal **abierta** |
| La extensión también borra | Con la extensión de verdad: revocar y que su copia de `storage.local` y su entrada del selector se vayan |
| La fila no vuelve | Dos equipos del mismo titular: el segundo sincroniza después y **no resucita** la compartida borrada |

**Y una del método, que la escribió el propio paseo al ponerse rojo:** el tramo que comprobaba la
revocación esperaba a **leer** `sin-acceso`, y con esto **ese estado pasa a ser transitorio** —Beto se entera,
sale a su bóveda personal y la de ésa va perfectamente, así que medio segundo después vuelve a decir
«al-dia»—. La prueba se cayó con el borrado funcionando. Ahora espera **al hecho** —la fila tachada y el
fichero fuera— y apunta por el camino los estados que llegó a ver, que es lo que separa «no se enteró» de «se
enteró y no borró»: con el borrado puesto dice `[al-dia]` y quitándolo dice `[sin-acceso]`. **Un estado que
deja de ser estable rompe a quien lo esperaba**, y lo que no deja de ser estable es lo que pasó.

**Y comprobado en el equipo de otra persona de verdad** (2026-10-07, Mac 2 del cliente, con la 2.42.0 y los dos Workers desplegados): actualizó, entró, y **al sincronizar la bóveda desapareció** y la fila quedó tachada con su aviso. No esperó ni al minuto: le llegó en la pasada que dispara abrirla.

Lo único que salió de verlo, y es de aspecto: **el aviso queda embutido en la fila**, apretado contra el botón de quitarla. Aquí no se vio porque la captura de Chromium en Linux sale holgada. Está en `deuda.md`, y el cliente lo dejó para más adelante con esas palabras.
