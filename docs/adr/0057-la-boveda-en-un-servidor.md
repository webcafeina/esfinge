# 0057 · La bóveda en un servidor, sin nadie delante

**Fecha:** 2026-10-20 · **Estado:** aceptada · **Toca la [0044](0044-desbloquear-con-el-sistema.md), la [0050](0050-varias-bovedas.md), la [0052](0052-dar-acceso-a-un-proyecto.md) y la [0054](0054-los-agentes-por-mcp.md)**

## Contexto

En el VPS viven dieciséis proyectos, y sus secretos están hoy en ficheros `.env` **en claro**: solo
`webcafeina-local/.env` tiene **111 variables**, `Tempero` otras 38 más 24 de producción, `coach` 9, y
Cronos cuatro ficheros. Cualquiera que entre en el servidor los lee, no hay rastro de qué se usó, están
repetidos entre `.env` y `.env.produccion`, y cambiar uno es editar un fichero a mano por SSH.

Y los agentes que trabajan ahí **ya los leen**: corren como `webcafeina` y los ficheros están al lado. Así
que cuando un agente necesita un testigo para desplegar, lo saca de un `.env` — y al leerlo **entra en la
conversación de un modelo**, donde se queda. Es exactamente lo que la [ADR 0054](0054-los-agentes-por-mcp.md)
se construyó para evitar en el Mac.

Lo pidió el cliente el 2026-10-20:

> *«Habría que pensar alguna forma de resolver o "facilitar" la gestión de credenciales que puedan usar
> agentes en proyectos en el VPS, ya que este caso se acabará dando.»*

**Lo que el VPS no puede tener, y manda sobre todo lo demás:** la 0054 se apoya en una cosa, que **cada
secreto se aprueba en una ventana donde hay alguien mirando**. En un servidor eso no existe, y aquí no se
puede trasladar. Hay que decir qué sustituye a esa ventana: **nada la sustituye**. Lo que queda es acotar el
alcance, dejar rastro, y que el secreto no pase por la conversación.

**Y la alternativa realista no es tenerlos a salvo: es tenerlos en `.env` en claro.** Es el mismo argumento
de la [0047](0047-los-datos-personales.md) con los datos personales y de la
[0025](0025-los-codigos-de-un-solo-uso.md) con el segundo factor, y es lo que hace que esto valga la pena
aunque la llave acabe en el servidor.

## Decisión

Lo decidido con el cliente, que no se cambia sin preguntar:

| | |
|---|---|
| **Cómo se abre** | Una **ranura propia del servidor**: una llave al azar en un fichero del equipo, fuera de la carpeta de la bóveda |
| **Alcance** | Las bóvedas de proyecto que ya existen, cada una con un selector: «trabajar con esta bóveda en el servidor» |
| **Identidad** | **El VPS es otra persona**: su cuenta de Esfinge, y se le da acceso como a Nacho (ADR 0052) |
| **Qué puede un agente** | **Ejecutar con el secreto dentro**, nunca verlo: el secreto no pasa por la conversación |
| **Cómo corre** | Se abre y se cierra en cada orden. **Sin servicio permanente**: ninguna bóveda abierta entre órdenes |
| **La sesión** | La clave de acceso de la cuenta del VPS queda sellada en su equipo, para que **vuelva a entrar solo** cuando la sesión caduque |

### Lo que cuesta, dicho en voz alta

> **Quien entre en el VPS puede abrir las bóvedas marcadas.** La llave está ahí, porque sin ella el
> servidor no puede trabajar solo. No protege de eso, igual que Touch ID no protege de un programa que
> corra como tú.

Está dicho con esas palabras **en la pantalla donde se activa** y no solo aquí, que es la regla de la
[0044](0044-desbloquear-con-el-sistema.md). Y lo que sí cambia respecto a hoy, que es por lo que vale la
pena:

| Hoy | Con esto |
|---|---|
| 111 variables en claro en un fichero | cifradas en reposo |
| repetidas entre `.env` y `.env.produccion` | una copia, sincronizada |
| cambiar una es editar en el servidor | se cambian en el Mac y llegan solas |
| sin rastro de qué se usó | registro dentro del cuerpo cifrado |
| quitarle el acceso = entrar y borrar | revocar en el Mac **rota la clave** (ADR 0052) |
| el agente las lee y entran en su conversación | el camino cómodo no se las enseña |

Y el alcance está acotado: la llave del servidor abre **su** bóveda, y de ahí solo las que se hayan marcado.
**Nunca la personal de nadie.**

### Una sola llave en disco

Es lo que hace que casi todo el mecanismo estuviera ya escrito. El VPS tiene su cuenta, con su bóveda
personal **vacía**, y a partir de ahí todo va en cascada con lo que ya existe:

- la **ranura del servidor** abre solo esa bóveda personal;
- con ella abierta, **los proyectos se abren con su clave de bóveda** (ADR 0050);
- y **las compartidas, con su identidad**, que vive dentro de su personal (ADR 0052).

O sea: **un solo secreto sin sellar en el disco**, y no uno por bóveda. Eso es lo que hay que proteger y lo
único que hay que poder rotar. La sesión de la cuenta y su clave de acceso van **selladas con la clave de la
bóveda**, así que cuelgan de esa misma llave y no añaden una segunda.

La ranura es una más, como la maestra, la de recuperación, la del sistema y la de cada titular de una
compartida. Comparte máquina con la de Touch ID (`internal/boveda/ranura_con_secreto.go`): de las dos, **lo
único que cambia es quién guarda el secreto**. Y está en `ranurasLocales`, igual que la del sistema: **no se
sube nunca**, porque subirla sería darle al servidor una segunda puerta.

Tres cosas de esa ranura que no son detalles:

- **El fichero va fuera de la carpeta de las bóvedas** (`~/.esfinge-servidor`, 0600, o lo que diga
  `ESFINGE_LLAVE_DEL_SERVIDOR`). Lo dice ya la 0044: con la llave al lado, quien copie la carpeta abre la
  bóveda, y eso no es un cerrojo peor, es quitar la puerta.
- **No se activa sola en ningún sitio.** En Linux sigue sin haber llavero del sistema, con su porqué escrito
  en `llavero_otros.go` —el Secret Service lo abre la sesión y no pide nada al leerlo: sería un cerrojo sin
  cerradura—, y esto no lo cambia: hay que pedirlo, a mano, una vez, **con la bóveda abierta**. Lo que en un
  portátil sería un cerrojo sin cerradura, en un servidor es justo lo que hace falta: allí no hay a quién
  proteger de quien se sienta delante, porque el servidor *es* el que trabaja.
- **Volver a activarla la rota**, y la de antes deja de servir.

### `esfinge boveda ejecutar`

Es el equivalente del portapapeles de la 0054 en una máquina sin pantalla: allí el agente recibe «copiado» y
la contraseña va al portapapeles; aquí recibe la salida del programa y la contraseña va **al entorno del
proceso hijo**.

```sh
esfinge boveda ejecutar --var CF_TOKEN="Cloudflare API" --var DB_URL="Neon Tempero" -- wrangler deploy
```

- **Nunca escribe un fichero.** Un `.env` generado por Esfinge es el problema de partida otra vez con un
  paso más.
- **La bóveda se cierra antes de lanzar el programa**, que puede tardar media hora. Lo que hacía falta ya se
  ha sacado.
- **O todo o nada**: si una de las tres búsquedas no encaja, no se lanza nada. Desplegar con dos
  credenciales de tres es un fallo que aparece lejos de aquí.
- **Queda apuntado en el registro** —qué entrada, cuándo y a qué programa—, con `Como:
  llave-del-equipo`, que dice lo que importa: **eso no lo aprobó nadie**. Y si no se ha podido apuntar
  —una compartida de solo ver—, **se dice**, que es lo que pide `registro.go`.
- **El código del programa se devuelve tal cual**, o un guion no se entera de que el despliegue falló.

**Y esto no es una frontera contra quien tenga terminal en el servidor:** con la llave de ese equipo,
`esfinge boveda ver` también saca la contraseña. Se acepta a sabiendas, con las palabras de Touch ID y del
portapapeles de la 0054. Lo que esto hace es que **el camino cómodo no se la enseñe** y deje rastro.

### La cuenta del servidor, desde la línea de comandos

Un servidor tiene que **recibir** las bóvedas que se le marquen, y eso es sincronizar, y sincronizar
necesita una cuenta. Así que la línea de comandos gana lo justo: `servidor alta` —que crea la cuenta **y su
bóveda**, porque sin bóveda personal no hay dónde viva su identidad—, `servidor entrar`, `servidor recibir`
y `servidor sincronizar`. **Administrar sigue fuera**: ni cambiar la maestra, ni recuperar, ni compartir, ni
borrar la cuenta. Para eso está la aplicación, igual que dice la 0054 de los agentes.

Y **se pone al día antes de dejar leer nada**, con `--sin-sincronizar` para cuando corra prisa: una
credencial vieja en un servidor es un fallo mudo — el despliegue falla con un testigo caducado y nada dice
por qué.

**La sesión caduca y la clave de acceso se guarda**, sellada con la clave de la bóveda. Es lo que permite
volver a entrar sin que nadie venga, y la confianza del equipo se renueva en cada entrada: mientras trabaje
al menos una vez cada noventa días, se mantiene solo.

**No hace falta `internal/app`**: `internal/sincro` ya es autónomo —`Sincronizador{Servidor, Bóveda,
Memoria, Token}`— así que la línea de comandos arma el suyo sin arrastrar el paquete de la ventana, que
lleva Wails dentro y no cruza de plataforma. El plan decía que haría falta montar un `Sistema` sin pantalla;
leyendo el paquete, no.

### El selector, en la ventana

En cada bóveda de proyecto, un interruptor: **«Trabajar con esta bóveda en el servidor»**. Por debajo es
**dar acceso de solo ver a la cuenta del VPS** — el mecanismo entero de la [0052](0052-dar-acceso-a-un-proyecto.md),
sin inventar nada: permiso que **impone el servidor y no el cifrado**, ranura sellada hacia su identidad
pública, **clave rotada al quitarlo** con su lápida, y la fila en la lista de quién tiene acceso. No añade
ni un método al puente: llama a `darAcceso` y `quitarAcceso`, que ya estaban.

Lo que añade respecto a escribir el correo a mano en «Dar acceso» son dos cosas, y la segunda es la que
importa: no hay que acordarse del correo, y **la pantalla dice lo que cuesta** — un servidor tiene su llave
en un fichero, y eso no se parece a dar acceso a una persona.

**Cuál es tu servidor se dice una vez, en Ajustes → Cuenta**, y es una preferencia local, como las carpetas
de los diálogos: cuál es tu servidor es de este equipo, no de la bóveda. **Sin ella puesta, la ventana no
menciona nada** — un interruptor para un servidor que no existe es ruido sobre algo que no se puede
arreglar, que es la regla de la biometría. Y el servidor **sigue saliendo en la lista de quién tiene
acceso**: esconderlo ahí porque tiene su propio interruptor dejaría un acceso invisible el día que alguien
borrara el correo de Ajustes.

## Alternativas descartadas

**Que el VPS sea otro equipo de la cuenta de Álvaro.** Es lo más corto —entrar desde un equipo nuevo ya
existe— y se cae por lo mismo que lo hace corto: entrar en una cuenta **se baja la bóveda personal**, así
que el servidor tendría todos los secretos de la empresa y no solo los del proyecto que se le marque.

**Entregar una copia y volver a entregarla cuando cambie** (ADR 0051). No hace falta nada nuevo, y es lo que
el cliente ya descartó para las personas el 2026-10-05: *«necesito que estas bóvedas de proyecto sí que se
actualicen en tiempo real»*. Una credencial vieja en un servidor es un fallo mudo — el despliegue falla con
un testigo caducado y nada dice por qué.

**Un servicio permanente con la bóveda abierta**, tipo `systemd`. Es lo que pediría un agente que quisiera ir
rápido, y deja la clave en memoria para siempre en una máquina expuesta: en un portátil eso lo acota el reloj
del bloqueo, y aquí no hay nadie que vuelva a teclear nada. Cada orden abre y cierra. De ahí sale gratis que
**el canal MCP no exista en el servidor**: sin proceso permanente no hay socket que apagar, y una puerta de
aprobación sin ventana donde aprobar es un cerrojo pintado.

**Escribir los `.env` desde Esfinge.** Resolver el problema generando el problema.

**Usar el Secret Service de Linux** en vez de un fichero. Sería el llavero «de verdad» del sistema, y no
protege de nada más: lo abre la sesión y no pide nada al leerlo. Añadiría media dependencia de escritorio
para el mismo resultado.

**Entrar a mano cada 90 días** en vez de guardar la clave de acceso. La sesión de la cuenta es deslizante de
30 días con un máximo absoluto de 90, y volver a entrar exige la clave de acceso, que se deriva de la
maestra. Sin guardarla, alguien tendría que entrar por SSH cada tres meses con la maestra de la cuenta del
VPS, y el fallo de olvidarse es mudo: el servidor sigue abriendo su copia y las credenciales se quedan
viejas. Lo eligió el cliente con el coste delante. **Lo que se guarda es la clave de acceso, nunca la
maestra**, y no abre el fichero de la bóveda: abre la cuenta, donde el permiso es de solo ver.

**Un testigo de equipo que no caduque, emitido por el servidor de cuentas**, con permiso de solo bajar y
retirable desde el Mac. Es lo más limpio de los tres y deja el servidor sin ninguna credencial que valga para
otra cosa; es tabla, endpoint y protocolo nuevos para un solo VPS. Si algún día hay más de uno, es por aquí
por donde se vuelve.

## Consecuencias

- **La línea de comandos deja de ser solo de lectura**, y es la primera vez. `servidor activar` escribe una
  ranura, y escribirla exige la bóveda abierta —o sea saber la maestra—, que es justo lo que impide activarlo
  sin ella. Está dicho en la cabecera de `internal/cli/boveda.go`.
- **Y gana lo de la cuenta**: entrar, sellar la sesión y sincronizar en una pasada, sin ventana. De paso es
  la salida que `boveda.go` ya prometía «si algún día la ventana no arranca».
- **Quitarle el acceso al servidor borra su copia**, como en el equipo de una persona (ADR 0053), y eso
  solo pasa **la próxima vez que el servidor trabaje**: parado, el fichero sigue ahí. No se puede prometer
  otra cosa, igual que con una persona.
- **Un apunte del registro puede no llegar al Mac.** Con permiso de solo ver, el servidor no sube nada, así
  que lo que `ejecutar` apunte se queda en su copia. Es coherente con el permiso y hay que saberlo al mirar
  el registro desde el Mac: lo que un servidor haya usado no sale ahí.
- **Si el VPS no trabaja en 90 días, hay que volver a entrar a mano.** La confianza del equipo se renueva en
  cada entrada, así que mientras sincronice al menos una vez cada tres meses se mantiene solo; parado más
  tiempo, caduca.
- **Esto no se activa en el Mac del cliente** y no hay nada en la ventana que lo ofrezca. La llave del
  servidor se pone con una orden, en el equipo donde hace falta.

## Verificación

Lo comprobado de verdad, con sus pruebas:

| Qué protege | Cómo |
|---|---|
| Que la ranura no se sube | `TestLaRanuraDelServidorAbreYNoViaja` mira los bytes de la subida. **Mutado** quitándola de `ranurasLocales`: rojo |
| Que las dos listas de ranuras locales no se desalinean | `TestLasRanurasLocalesSonLasMismas` compara Go con el espejo de TypeScript leyendo el fuente. **Mutado por los dos lados**: rojo cada vez |
| Que la llave no está al lado de la bóveda | `TestLaCarpetaDeLaBovedaNoLlevaLaLlaveDelServidor` copia la carpeta y busca la llave **en los bytes** de todo lo que haya |
| Que no se activa sola | `TestElLlaveroDelSistemaNoEsElDelServidor`: con la llave puesta, `llavero.Del()` sigue diciendo que aquí no hay nada |
| Que antes de activarlo no abre nada | `TestEnUnServidorLaBovedaSeAbreSinContrasenaYSoloSiSePide` empieza por ahí: sin activar, no abre |
| Que una clave dicha a mano manda | `TestLaClaveDichaAManoGanaALaLlaveDelServidor`. **Mutado** quitando la precedencia: rojo |
| Que la llave va en la personal y no en un proyecto | `TestActivarEnUnProyectoSeNiegaAntesDePedirNada`, **sin darle ninguna clave**: si el orden fuera el otro, lo que saldría es que falta la clave. **Mutado**: rojo |
| Que se dice lo que cuesta al activarlo | La misma prueba recoge la salida de error y busca la frase |
| Que el secreto llega al programa y no a la salida | `TestEjecutarLeDaElSecretoAlProgramaYNoALaSalida`: el hijo escribe su entorno en un fichero, y se busca el secreto **en los bytes** de lo que dice Esfinge |
| Que la bóveda está cerrada cuando arranca el programa | `TestCuandoArrancaElProgramaLaBovedaYaEstaCerrada`, por la costura de `lanzar`. **Mutado** quitando el cierre: rojo |
| Que queda rastro, y sin el secreto dentro | `TestEjecutarDejaRastroEnElRegistro` vuelve a abrir el fichero y busca el secreto en claro |
| Que si no se puede apuntar, se dice | `TestSiNoSePuedeApuntarSeDice`, con la carpeta sin permiso de escritura. **Mutado**: rojo |
| Que el código del programa se devuelve | `TestEjecutarDevuelveElCodigoDelPrograma`, con un 7 |
| Que o todo o nada | `TestEjecutarNoLanzaNadaSiFaltaUnSecreto`, en tres casos, comprobando que el programa **no se ha lanzado** |
| Que el alta, la sesión y la pasada funcionan | `TestUnServidorSeDaDeAltaAbreSinContrasenaYSeSincroniza`, **contra el Worker de verdad** levantado en local, con el código del correo tecleado por la entrada estándar mientras el comando espera |
| Que la clave de acceso va sellada | La misma prueba mira que los tres secretos empiezan por `ESF1.`. **Mutada** guardándola en claro: rojo |
| Que vuelve a entrar sola | `TestSiLaSesionNoValeElServidorVuelveAEntrarSolo`. **Mutada** quitando la vuelta a entrar: rojo |
| **La tubería entera** | `TestPaseoDeUnProyectoHastaUnServidor`: la ventana de verdad crea cuenta y proyecto, el servidor se da de alta, activa su llave, la ventana le da acceso de solo ver, el servidor lo recibe y se trae la bóveda, le da el secreto a un programa **por el argv entero**, la ventana cambia la contraseña y el servidor usa la nueva, y con «solo ver» el servidor contesta que no al subir |
| Que al quitarle el acceso se va y borra su copia | El paso 9 del paseo: la ventana quita el acceso, y lo siguiente que el servidor intenta abrir **dice que ya no tiene acceso, borra el fichero y tacha la fila**. **Mutado** quitando la retirada: rojo |
| Que el selector sale donde tiene que salir y dice lo que cuesta | Una prueba de interfaz en los dos temas: **sin servidor en Ajustes no se menciona**, con él puesto sale apagado diciendo a quién y qué podrá hacer, y al quitarlo desaparece |
| Que la lista de variables de la carpeta de configuración es una | `TestLaCarpetaDeConfiguracionSeApartaEnUnSoloSitio`, que ahora recorre **el repositorio entero** en vez de un paquete. Al escribirlo encontró una cuarta copia en `cmd/dev/main.go` **a la que le faltaban dos** |

**Lo que no se ha comprobado:**

- ~~Nada de esto se ha usado todavía en el VPS de verdad.~~ **Montado en el VPS el 2026-10-21**, contra el
  servidor de cuentas de producción y guiando al cliente paso a paso: alta con `info+servidor@webcafeina.com`
  —el `+` evitó tener que crear un buzón en Workspace—, activar, acceso de solo ver a «Zeri's Coffee» con la
  huella comparada por los dos lados, `recibir` y `ejecutar` entregando el secreto a un programa sin que
  apareciera en ninguna pantalla. El apunte quedó en el registro. **El selector de la ventana no hizo falta**:
  por debajo es dar acceso de solo ver, y eso la 2.58.7 ya lo hacía.
- **Que sea cómodo en un despliegue real.** Eso lo dice el cliente usándolo, que es lo que ha encontrado
  todo lo demás este mes.
- **Que el selector se entienda al verlo en el Mac.** Lo de aquí son capturas miradas en dos temas, no una
  ventana de verdad.
