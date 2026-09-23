# Revisión de los textos legales, septiembre de 2026

**Qué es esto.** La política de privacidad de Esfinge contrastada **frase por frase con el código**, más
lo que le falta para ser una información de tratamiento completa, más unas condiciones de uso que no
existían. Lo decidió el cliente el 2026-09-23 ([ADR 0041](adr/0041-los-papeles-de-la-cuenta.md)).

**Y sus límites, dichos aquí y no en letra pequeña.** Esto no es asesoramiento jurídico y no responde nadie
de ello. Lo que sí puede hacerse bien, y es la mitad que más valor tiene, es **comprobar que lo que dicen
los textos es lo que hace el programa**: eso se verifica contra el código, y es lo que ocupa la primera
parte. La segunda —los puntos que el RGPD pide que aparezcan— es una lista contrastada con el reglamento,
no el criterio de un despacho.

Los textos son tres y tienen que decir lo mismo (ADR 0033): `web/privacidad.html`, el aviso del panel de
la extensión (`navegador/src/panel.html`) y lo declarado en las fichas de las dos tiendas.

## 1 · Lo que la política dice y el código confirma

Comprobado leyendo el servidor, la aplicación y la extensión. **Todo esto es verdad**, y conviene que
conste porque es lo que sostiene la promesa del producto:

| Lo que dice la política | Dónde se comprueba |
|---|---|
| La bóveda sale cifrada y el servidor no puede abrirla | El servidor solo ve `sobres`, `sello` y `cuerpo` ya sellados; la clave sale de la maestra con Argon2id en el cliente |
| «Tu dirección IP no la guardamos»: se cuenta una huella y se tira a los dos días | `claveDeIP` es un HMAC de la IP con `SECRETO_PRELOGIN`; `contar` borra lo anterior a dos días |
| Diez versiones anteriores y una por día del último mes | `VERSIONES_RECIENTES = 10`, `DIAS_CON_VERSION = 30` |
| Los doscientos últimos avisos | `EVENTOS_GUARDADOS = 200`, recortado en cada escritura |
| Los códigos caducan a los diez minutos | `RETO = 10 * MINUTO` |
| «Una huella de la clave de acceso, no tu contraseña» | El servidor guarda `HMAC(PIMIENTA, claveDeAcceso)` |
| Al borrar la cuenta se borra todo del servidor | `deleteAll()` del Durable Object y `DELETE FROM cuentas` en D1, en la misma petición |
| Sin cuenta, la extensión no se conecta a internet | Sin cuenta solo habla por el canal nativo; el aviso de datos frena los tres caminos |
| No hay analítica ni terceros | No hay ninguna salida más; los registros del Worker están apagados (`observability.enabled: false`, con su prueba) |

## 2 · Lo que no cuadraba

### 2.1 · El correo de quien empieza un alta y no la termina se quedaba para siempre — **arreglado**

`servidor/src/indice.ts`. La fila de `altas` —correo, huella del código, caducidad— **solo se borraba al
completar el registro**. Quien pedía un código, se lo pensaba y no seguía, dejaba su dirección en la base
sin cuenta, sin servicio y sin nada que la sostuviera. La política no lo decía porque nadie lo había
mirado.

Se arregla en el código y no en el texto, que es lo que toca: cada vez que alguien pide un código se barren
las caducadas. El código no vale pasados diez minutos; la fila tampoco. Con su prueba, comprobada al revés.

### 2.2 · La lista de admisión no se menciona

Mientras el registro es por invitación, D1 tiene una tabla `admision` con los patrones de quién puede
entrar —hoy, `@webcafeina.com`—. Si algún día lleva direcciones concretas de personas, eso es un dato
personal que la política no cuenta. **Propuesta**: decirlo en una frase, y que desaparezca sola el día que
el registro se abra y la tabla deje de usarse.

### 2.3 · «Prueba otra vez en un momento» cuando el correo se agota

No es un asunto de privacidad, pero sale aquí porque es lo mismo: **decir lo que pasa**. Con el plan
gratuito de Resend, agotado el cupo del día, el servidor contesta esa frase y no es verdad: es hasta
mañana. Está en [`deuda.md`](deuda.md) y en la [ADR 0041](adr/0041-los-papeles-de-la-cuenta.md), para
arreglarlo antes de abrir el registro.

## 3 · Lo que falta para ser una información de tratamiento completa

La política cuenta muy bien **qué** se guarda y **dónde**. Lo que le falta es la parte formal, y son seis
cosas:

1. **Quién responde, con nombre y dirección.** Hoy dice «la hace Webcafeína» y un correo. Hace falta la
   denominación social, el NIF y el domicilio, tanto para el RGPD como para la [Ley 34/2002 (LSSI)](https://www.boe.es/buscar/act.php?id=BOE-A-2002-13758),
   que pide esos datos a quien presta un servicio por internet. **Esto solo lo puede poner el cliente.**
2. **Con qué base se tratan los datos.** Para la cuenta, la ejecución del contrato (art. 6.1.b del RGPD):
   sin correo no hay forma de darte el servicio que pides. Para los frenos y los avisos de seguridad, el
   interés legítimo (art. 6.1.f). Sin cuenta no hay tratamiento ninguno, y eso también conviene decirlo
   con esas palabras.
3. **Los derechos, enumerados, y adónde reclamar.** Acceso, rectificación, supresión, limitación,
   oposición y portabilidad; y que se puede reclamar ante la **Agencia Española de Protección de Datos**.
   La política ya explica cómo ejercerlos desde Ajustes, que es mejor que la mayoría; lo que falta es
   nombrarlos.
4. **Las transferencias fuera de la UE.** Los datos se guardan en la UE, y eso está comprobado. Pero
   Cloudflare y Resend son empresas estadounidenses, así que el acceso desde fuera es posible y hay que
   decir con qué garantía —cláusulas contractuales tipo o el marco de adecuación—, que es lo que cada una
   declara en su acuerdo de encargado.
5. **Menores.** Un gestor de contraseñas no es para niños, pero si no se dice nada no hay edad mínima.
   Lo habitual es pedir catorce años, que es la edad del consentimiento en España.
6. **Cuánto duran las cosas, dicho una a una.** La política dice «mientras tengas la cuenta». Los plazos
   de verdad están en el código y son buenos: dos días las huellas de IP, diez minutos los códigos, treinta
   días las versiones, doscientos los avisos, una hora los envíos. Decirlos da más confianza que la frase
   genérica, y ya están comprobados.

## 4 · Las condiciones de uso

No existían. Se escriben en `web/condiciones.html` y tienen que decir, como mínimo, lo que este servicio
tiene de particular:

- **Qué es y qué cuesta**: hoy, gratis y por invitación; que se puede dejar de ofrecer, avisando.
- **Que perder la contraseña maestra y la clave de recuperación es perderlo todo**, y que Webcafeína no
  puede devolver nada. Es la cláusula más importante del documento y no puede estar escondida.
- **Que el programa se da tal cual**, sin garantía, en la línea de su licencia, y con la responsabilidad
  limitada a lo que la ley permite limitar.
- **Qué no se puede hacer** con el servicio: usarlo para lo que la ley prohíbe, o para reventarlo.
- **Baja y cierre**: el cliente se da de baja desde Ajustes; Webcafeína puede cerrar una cuenta que rompa
  estas condiciones, avisando y dando tiempo a llevarse los datos.
- **Ley aplicable y juzgados**, que para un consumidor son los de su domicilio y conviene decirlo bien.

## 5 · La segunda lectura, y lo que salió de ella

Los dos borradores se pasaron por una lectura independiente (modelo Fable), con el servidor delante.
**Cada hallazgo se comprobó a mano antes de aceptarlo.** Lo que cambió:

- **«Al borrarla, se borra todo, en el momento» prometía de más.** Nuestras bases sí, y es atómico. Pero
  **Cloudflare conserva copias de recuperación hasta treinta días** —está en su documentación, para los
  Durable Objects con SQLite y para D1— y Resend conserva los correos que ya mandó. Ahora el texto lo
  dice.
- **«No queda nada» del alta a medias tampoco era cierto**, y la causa estaba en el código: la limpieza
  que se había escrito **solo corría cuando otra persona pedía un código**, y con el registro por
  invitación eso puede tardar semanas. Se ha añadido un **reloj en el Worker** (`scheduled`, cada hora)
  que barre las tres tablas con datos de alguien: `altas`, `envios_alta` y los contadores de IP. Con su
  prueba.
- **«Tu dirección IP no la guardamos» es más de lo que se hace**: una huella con HMAC es
  seudonimización, no anonimato, y **Cloudflare ve la IP de cada petición**. Dicho tal cual.
- **Resend ve más que la dirección y el código**: por el asunto y el cuerpo ve el nombre del equipo y de
  qué aviso se trata. Dicho.
- **El servidor guarda el sobre de recuperación** y lo entrega a quien presente un código del correo, sin
  contraseña ni sesión. Es el diseño (ADR 0037), pero el texto decía «no guardamos la clave en ninguna
  parte» y se leía como que ahí no hay nada que ayude a recuperar. Ahora se dice, y con su consecuencia:
  **el buzón importa tanto como las claves**.
- **Las condiciones tenían dos cláusulas que no aguantarían** frente a un consumidor: el «tal cual» de
  toda la vida —nulo por el art. 86 del texto refundido— y un cierre de cuenta sin plazo ni criterio.
  Reescritas: responsabilidad por dolo y negligencia con las exclusiones que la ley permite, y treinta
  días de preaviso salvo urgencia justificada.
- **Faltaban el desistimiento, los pasos del alta y el canal de reclamaciones**, que la ley de consumo y
  la LSSI piden expresamente. Añadidos.
- **La licencia del repositorio no concede el derecho a usar el programa**: enumera leer, auditar,
  compilar y poco más. Las condiciones ahora dan una licencia personal de uso y remiten a la del código
  solo para el código.
- **«Ni aunque viniera con una orden»** se leía como que Webcafeína desobedecería a un juez. Lo que no
  podemos entregar es el contenido; la copia cifrada y los datos de la cuenta, sí. Corregido.

## 6 · Lo que solo puede hacer el cliente

1. **Los datos de identificación**: denominación social, NIF y domicilio para la política, las condiciones
   y un aviso legal en la web.
2. **Los encargos de tratamiento**: aceptar el de **Cloudflare** y el de **Resend** en sus paneles. Son
   dos casillas, y sin ellas el tratamiento no está cubierto por escrito.
3. **Decidir la edad mínima** y confirmar la ley y los juzgados que quiere nombrar.
4. **El registro de actividades de tratamiento** (art. 30 del RGPD), que es un documento interno y corto:
   con lo que hay en esta revisión se escribe en una tarde.
5. **De Resend**: en qué región guarda los datos, cuánto conserva los correos y su acuerdo de encargado.
   De Cloudflare la región está garantizada por el código; de Resend **no hay nada que lo diga**.
6. **El plazo de preaviso** —los borradores proponen treinta días— y si Webcafeína está adherida a alguna
   entidad de resolución alternativa de conflictos.

## 7 · Lo que queda por hacer en el código

Salió de la segunda lectura y **no está hecho**:

- **La ventana no enlaza las condiciones ni la política**, y la extensión enlaza solo la política. Antes
  de crear una cuenta hay que poder leerlas: una línea con los dos enlaces bajo el botón de crear.
- **No hay correo de confirmación del alta**, que el art. 28 de la LSSI pide. Es una carta más en
  `correo.ts` y un envío en `terminarAlta`. Gasta un correo más de los cien al día.
- **No hay forma de cerrar ni suspender una cuenta** salvo que lo haga su dueño con su contraseña y su
  código. Mientras no la haya, la política no puede prometer que se borrará la cuenta de un menor ni las
  condiciones un cierre por incumplimiento con su plazo. **O se escribe, o se rebajan las dos frases.**
