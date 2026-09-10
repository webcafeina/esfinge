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
    | "rellenar";
  origen?: string;
  id?: string;
  testigo?: string;
  quien?: string;
};

/** Una cuenta de la bóveda. **Sin secretos**: aquí no hay dónde ponerlos. */
export type Cuenta = {
  id: string;
  titulo: string;
  usuario: string;
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

export type Respuesta = {
  ok: boolean;
  error?: string;
  /** Etiqueta estable para decidir sin mirar el texto. Los textos cambian. */
  motivo?: Motivo;
  estado?: Estado;
  cuentas?: Cuenta[];
  copiado?: Copiado;
  relleno?: Relleno;
  testigo?: string;
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
  | "sin-esfinge";
