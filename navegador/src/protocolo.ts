/**
 * Lo que se puede pedirle a Esfinge, y nada más.
 *
 * **Es una copia de `internal/navegador/protocolo.go` y no una biblioteca
 * compartida**, a propósito: son dos programas que se publican por caminos
 * distintos —éste por una tienda con revisión de días, aquél empujando una
 * etiqueta— y van a estar descompasados casi siempre. Un tipo compartido daría la
 * impresión contraria. Lo que los mantiene juntos es el número de versión, que va
 * en cada petición y que Esfinge comprueba.
 */
export const VERSION_DEL_PROTOCOLO = 1;

export type Peticion = {
  version: number;
  que:
    | "estado"
    | "emparejar"
    | "cuentas"
    | "copiar-secreto"
    | "copiar-codigo"
    | "rellenar"
    | "rellenar-codigo"
    | "ofrecer"
    | "guardar-cuenta"
    | "actualizar-cuenta"
    | "nunca-aqui"
    // Las llaves de acceso (ADR 0048).
    | "llaves"
    | "firmar-llave"
    | "crear-llave";
  origen?: string;
  id?: string;
  testigo?: string;
  quien?: string;
  /** Lo que se acaba de enviar en un formulario (entrega 3). */
  usuario?: string;
  secreto?: string;
  titulo?: string;
  forma?: Forma;

  /**
   * Las llaves de acceso (ADR 0048).
   *
   * `rpId` es **el que ha pedido el sitio**, y se comprueba contra el origen que
   * pone el navegador: es la única vía por la que esto podría firmar para quien no
   * es. `permitidas` son los identificadores de credencial que el sitio dice
   * aceptar —no son secretos: los emitió él y acaba de mandarlos—, y `reto` es su
   * desafío, también suyo.
   */
  rpId?: string;
  permitidas?: string[];
  reto?: string;
  /**
   * Lo que hace falta para **crear** una llave (P3), y lo dice el sitio.
   *
   * `usuario` y `titulo` ya están arriba y se reutilizan. `idUsuario` es el `user.id`
   * del sitio: **bytes opacos** que hay que devolver tal cual al firmar y que no
   * significan nada aquí —no es el correo—. `excluidas` son las llaves que el sitio
   * dice tener ya para esa cuenta, y `algoritmos` los que acepta: **sin `-7` no se
   * crea nada**, porque es el único que Esfinge sabe firmar.
   */
  idUsuario?: string;
  excluidas?: string[];
  algoritmos?: number[];
};

/** Una llave de acceso, **tal como se enseña en el banner**: sin nada de dentro. */
export type LlaveParaElBanner = {
  /** El identificador de la entrada en la bóveda, no el de la credencial. */
  id: string;
  /** Lo que se lee: la cuenta a la que corresponde. */
  nombre: string;
};

/**
 * Lo que sale de firmar, que es **lo único que sale**: la clave privada no cruza
 * ningún puente, ni éste ni el de la ventana.
 *
 * Todo en base64url, que es como WebAuthn escribe lo binario, y **son los bytes
 * exactos que se han hasheado**: el sitio los compara byte a byte, así que volver a
 * serializarlos al otro lado daría otra cosa.
 */
export type Afirmacion = {
  idCredencial: string;
  idUsuario?: string;
  datosDelCliente: string;
  datosDelAutenticador: string;
  firma: string;
};

/**
 * Lo que sale de crear una llave, y **lo que el sitio se queda para siempre** (P3).
 *
 * No lleva firma: con `fmt: "none"` no hay nada que firmar, y la clave pública viaja
 * **dentro** del objeto de atestación, en su `authData`. Sacarla aparte sería tener el
 * mismo dato en dos sitios.
 */
export type Atestacion = {
  idCredencial: string;
  /** Los bytes exactos que se hashearon, en base64url. */
  datosDelCliente: string;
  /** El `attestationObject` en CBOR, en base64url. */
  objeto: string;
};

/**
 * Qué es un formulario que se envía. **Si no está claro, no se manda nada**: aquí
 * solo hay tres.
 */
export type Forma = "entrar" | "registro" | "cambio";

/**
 * Lo que la tarjeta de la página tiene que ofrecer. **Sin secretos**: la
 * contraseña se queda en el trabajador de fondo, que es quien la guarda.
 */
export type Oferta = {
  accion: "guardar" | "actualizar" | "nada";
  /** El anfitrión para el que se guardaría. */
  sitio: string;
  /** El título que se sugiere para una cuenta nueva. */
  titulo?: string;
  /** Las candidatas a actualizar; con varias, se elige en la tarjeta. */
  cuentas?: Cuenta[];
};

/** Una cuenta de la bóveda. **Sin secretos**: aquí no hay dónde ponerlos. */
export type Cuenta = {
  id: string;
  titulo: string;
  usuario: string;
  /**
   * Si la entrada guarda semilla de un solo uso. Solo el hecho, no la semilla.
   * Llega desde la 2.19.0: una Esfinge anterior no lo manda, y entonces vale
   * `undefined`, que el panel trata como «no se sabe» y enseña el botón igual.
   */
  tieneCodigo?: boolean;
};

export type Estado = {
  existe: boolean;
  abierta: boolean;
};

/**
 * Lo que se sabe después de copiar algo. **No lleva lo copiado.**
 *
 * Copia Esfinge, no la extensión, y por eso en esta entrega **ningún secreto
 * llega al navegador**. De regalo viene el borrado del portapapeles que Esfinge
 * ya hacía desde la 2.12.0.
 */
export type Copiado = {
  /** Segundos hasta que Esfinge lo borre, o cero si el borrado está apagado. */
  portapapeles: number;
  /** Vida que le queda al código de un solo uso. Solo en «copiar-codigo». */
  quedan?: number;
};

/**
 * Con qué rellenar un formulario. **Es lo único de este protocolo que lleva un
 * secreto dentro**, y por eso tiene su propio tipo: para que buscar quién toca
 * una contraseña en la extensión sea buscar un nombre.
 *
 * Quien lo recibe tiene una obligación que ningún tipo puede imponer: escribirlo
 * en el campo y olvidarlo. Nada de guardarlo, nada de registrarlo, nada de pasarlo
 * a otra parte de la extensión. Lo que sale por aquí **no hereda el borrado del
 * portapapeles**, porque no pasa por el portapapeles.
 */
export type Relleno = {
  usuario: string;
  secreto: string;
};

/**
 * El código de un solo uso para escribirlo en un formulario. Como `Relleno`,
 * lleva un secreto y va en su propio tipo; y la misma obligación: se escribe y se
 * olvida.
 */
export type CodigoParaRellenar = {
  codigo: string;
  /** Segundos de vida que le quedan. */
  quedan: number;
};

export type Respuesta = {
  ok: boolean;
  error?: string;
  /** Etiqueta estable para decidir sin mirar el texto. Los textos cambian. */
  motivo?: Motivo;
  estado?: Estado;
  cuentas?: Cuenta[];
  copiado?: Copiado;
  relleno?: Relleno;
  codigo?: CodigoParaRellenar;
  oferta?: Oferta;
  /** Lo que se ha guardado o actualizado desde la página. */
  guardada?: Cuenta;
  /**
   * **Solo con cuenta** (ADR 0040): lo que el panel tiene que copiar. La aplicación
   * copia ella misma y no lo manda nunca; con cuenta, la bóveda vive en el
   * trabajador de fondo, que no puede tocar el portapapeles, así que copia el
   * panel, donde se ha hecho el clic. Quien lo recibe lo copia y lo olvida.
   */
  paraCopiar?: string;
  testigo?: string;
  /** Las llaves de acceso que hay para ese sitio (ADR 0048). */
  llaves?: LlaveParaElBanner[];
  /** Lo que sale de firmar con una de ellas. */
  afirmacion?: Afirmacion;
  /** Lo que sale de crear una (P3). */
  atestacion?: Atestacion;
  /**
   * Los dominios que tienen llave, **solo al preguntar sin decir de qué sitio**.
   *
   * Es lo único de la bóveda que el navegador guarda —en `storage.session`, que
   * muere al cerrarlo—, y es lo que permite que el banner salga con la bóveda
   * cerrada, cuando no hay a quién preguntar (ADR 0048).
   */
  dominios?: string[];
  /**
   * «Aquí hay llave, pero la bóveda está cerrada.»
   *
   * Lo pone **el trabajador de fondo**, no la bóveda: viene de la lista de arriba y
   * es lo que hace que el shim no ceda y salga el banner que ofrece abrirla. Va con
   * `ok: false` y `motivo: "cerrada"`.
   */
  quizas?: boolean;
};

export type Motivo =
  | "cerrada"
  | "sin-boveda"
  | "sin-emparejar"
  | "origen"
  | "no-encaja"
  | "demasiado"
  | "no-entiendo"
  /** Lo pone el propio puente cuando Esfinge no está abierta. */
  | "sin-esfinge"
  /**
   * Lo pone el trabajador de fondo mientras no se ha aceptado el aviso de datos de
   * la extensión (ADR 0033). Esfinge no lo manda nunca: la pregunta no llega.
   */
  | "sin-consentimiento";
