# ADR 0023 — La bóveda: Esfinge pasa de cifrar secretos a custodiarlos

**Fecha:** 2026-09-09 · **Estado:** aceptada · **Matiza la [0010](0010-que-guarda-el-historial.md)** ·
**Revisar si** aparece sincronización o compartir en equipo

## Contexto

El cliente quiere dejar de depender de Dashlane y sustituirlo poco a poco por
Esfinge. Se valoró entero —bóveda, autorrelleno, cuentas, servidor y compartir—
y la conclusión fue que **eso no es ampliar Esfinge, es construir Dashlane**: un
proyecto por fases, con puertas para parar. Esta ficha cubre solo la primera.

Y cambia lo que el producto es. Hasta ahora Esfinge era **un cifrador sin
estado**: cifra lo que le den y no guarda nada. Eso está escrito en la portada
—«sin cuentas y sin servidores»— y la [ADR 0010](0010-que-guarda-el-historial.md)
llegó a descartar explícitamente guardar el contenedor cifrado en el historial,
con este argumento: *convierte un fichero de conveniencia en un objetivo que
merece la pena robar*.

**La bóveda invierte ese argumento a conciencia.** Hoy un fallo pierde un
fichero; a partir de ahora puede perder todas las contraseñas de una empresa.
Eso sube el listón de las pruebas, de las publicaciones y de la prisa, y por eso
la fase 0 —congelar el formato con vectores fijos, [ADR 0022](0022-vectores-fijos.md)—
fue antes que una sola línea de esto.

## Decisión

**Una bóveda local, cifrada, con clave de recuperación.** Sin servidor, sin
cuentas, sin navegador y sin móvil.

### El formato: un JSON en claro con líneas `ESF1.` dentro

La bóveda es un documento JSON legible cuyos campos criptográficos son líneas
`ESF1.` corrientes, indistinguibles de las que salen de `esfinge cifrar`.

**La idea evidente no funciona, y conviene decirlo porque es lo primero que uno
intenta**: un contenedor `ESF1` cuyo contenido sea la bóveda no vale, porque un
contenedor tiene una sal y una clave, y la clave de recuperación exige dos
entradas independientes al mismo secreto.

Lo que se gana con el JSON por fuera:

- **No se toca `internal/cripto`.** Justo después de congelarlo con vectores
  fijos, la opción que no lo toca vale mucho más de lo que parece. Y es lo que
  garantiza que **los `.esf` y las claves ya emitidos siguen valiendo**.
- **Las ranuras son una lista abierta.** Añadir `llavero-del-sistema` para Touch
  ID, o `servidor` cuando haya cuentas, no cambia el formato ni migra nada. Es la
  puerta de las fases siguientes, dejada abierta sin construirla.
- **Se abre con las piezas de siempre.** Hay un test que descifra el sobre y el
  cuerpo con llamadas peladas a `internal/cripto`, sin pasar por el código de la
  bóveda: si este paquete tuviera un fallo dentro de cinco años, los secretos se
  sacan igual con `esfinge descifrar` y dos pasos.

### La jerarquía de claves

La contraseña maestra **no cifra la bóveda**: cifra la clave que la cifra. Esa
clave —32 bytes de azar— va envuelta dos veces, con la maestra y con la de
recuperación. Cualquiera de las dos abre; ninguna revela la otra.

**Y no hizo falta escribir criptografía nueva**: `cripto.Sellar` ya *es* una
envoltura de clave con derivación, con su sal y sus parámetros dentro de la
cabecera. Cada sobre es una llamada a `SellarTexto`.

Dos decisiones que parecen detalles y no lo son:

- **La clave de bóveda viaja como texto base64**, nunca como bytes crudos. La
  línea de comandos lee claves con `--clave-fichero`, y ahí se recortan los
  saltos de línea del final: una clave binaria que acabara en `0x0A` —una de cada
  cien— se truncaría en silencio y la vía de escape fallaría de forma
  aparentemente aleatoria.
- **El cuerpo se sella con `PerfilLlave`, no con `PerfilInteractivo`.** Estirar
  una clave que ya es aleatoria no añade nada, y costaría medio segundo **en cada
  guardado**. Los parámetros viajan en la cabecera, así que el fichero se
  describe a sí mismo.

### La clave de recuperación

Base32 de Crockford —sin I, L, O ni U—, en grupos de cuatro, con prefijo `ESF` y
**una suma de control**. La suma es la diferencia entre «te has equivocado al
copiarla» y «has perdido la bóveda», y se comprueba antes de gastar medio segundo
derivando. Al teclearla se acepta lo que la gente escribe de verdad: minúsculas,
sin guiones, con «O» donde va un cero.

Se enseña **una sola vez** al crearla y no se guarda en ninguna parte.

## Alternativas descartadas

- **Versión 2 del contenedor `ESF1`.** El byte de versión pasaría a significar
  dos cosas ortogonales —qué criptografía y qué clase de objeto— y `leerCabecera`
  se convertiría en un despachador. Además una bóveda necesita dos ranuras con
  dos sales, y en la cabecera de 55 bytes solo cabe una.
- **Una magia nueva** (`ESFB`). Obligaría a un segundo camino en `FormaDe`, en la
  apertura por doble clic, en el `.mime` de Linux, en el icono del documento y en
  el instalador de Windows. Todo ese trabajo de empaquetado —que en este proyecto
  ya costó tres versiones— para un fichero que casi nunca se toca a mano.
- **Guardar la clave de recuperación en algún sitio** para poder volver a
  enseñarla. Entonces deja de ser una llave guardada fuera y pasa a ser otra copia
  dentro, que es exactamente lo que no puede ser.
- **No tener clave de recuperación**, como los `.esf`. Es lo coherente con lo que
  Esfinge era, y es inaceptable para lo que pasa a ser: perder la maestra no puede
  significar perder todas las contraseñas de golpe. Lo decidió el cliente con las
  consecuencias delante.
- **`mlock` y memoria protegida.** No es portable y, sobre un montón que Go
  gestiona a su manera, tampoco serviría de mucho. Lo que sí se hace es mandar a
  la ventana **una contraseña cada vez**, que hace más que todo el borrado de
  búferes junto.

## Consecuencias

- **Esfinge pasa a ser el único punto de fallo.** Está dicho arriba y va aquí
  otra vez porque es lo que hay que recordar antes de cada publicación.
- **La clave de recuperación es una segunda puerta a todo**, y no caduca hasta
  que se rota. Guardarla es tan importante como guardar la maestra, y en otro
  sitio.
- **El historial de contraseñas conserva las anteriores**, así que un secreto
  sustituido sigue dentro hasta que se borre a mano. Lo pidió el cliente, con el
  caso real de «cambié la contraseña y el servicio no se enteró».
- **El JSON exterior no va autenticado en su conjunto.** Quien pueda escribir el
  fichero no puede leer nada ni fabricar una bóveda que abra, pero sí estropearla
  o revertirla. Se **detecta** con un sello por dentro; no se impide.
- **La ADR 0010 se matiza**: el historial sigue sin guardar secretos, y la bóveda
  **nunca escribe en él**. `credenciales-dashlane.csv` en el historial sería una
  señal de tráfico apuntando a lo que alguien acaba de exportar en claro.
- `docs/seguridad.md` deja de prometer que no hay recuperación, y suma tres
  advertencias nuevas: la segunda llave, las contraseñas anteriores y que la
  bóveda pasa horas abierta en memoria.

## Verificación

- **35 pruebas del paquete de la bóveda** con `-race`, entre ellas: que la de
  recuperación abre; que una errata se distingue de una llave que no abre; que
  **cambiar la maestra deja el cuerpo byte a byte idéntico**; que quitar una
  ranura o revertir el cuerpo se detectan; y que dos Esfinges abiertas no se
  pisan.
- Del importador: **Dashlane, Bitwarden, 1Password, LastPass y Chrome**, más la
  marca de orden de bytes, el punto y coma del Excel europeo, los saltos de línea
  dentro de una nota, el CRLF y los ficheros en Windows-1252. Con ida y vuelta:
  lo exportado se vuelve a leer.
- Del bloqueo: que **un salto de ocho horas bloquea** —la lección de la ADR
  0014— y que el portapapeles se borra solo **sin pisar lo que se haya copiado
  después**.
- **`TestLoQueCruzaElPuenteEstaEnLaLista`**: una lista blanca de los métodos
  exportados de `*App`. `CLAUDE.md` avisaba de ese riesgo desde hacía versiones y
  la única defensa era acordarse; con contraseñas dentro, un método de más puede
  ser la clave maestra saliendo por ahí.
- Medido, no estimado: **20.000 entradas se guardan en 92 ms** y se abren en
  276 ms contando el Argon2id. El límite no es la CPU.

**Lo que no se ha comprobado:** nada de esto se ha usado con datos de verdad ni
en un Mac. Y sigue sin haber **auditoría externa**, que para un cifrador era una
nota al pie y para un gestor de contraseñas es la primera pregunta que hará
cualquiera.
