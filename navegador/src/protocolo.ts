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
    | "firmar-llave";
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
