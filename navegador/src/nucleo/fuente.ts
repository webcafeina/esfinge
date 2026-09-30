/**
 * Lo que la extensión contesta **con su propia bóveda**, cuando hay cuenta
 * (ADR 0040): los mismos verbos del canal con la aplicación, con las mismas reglas
 * y las mismas frases que `Servidor.Atender` (`internal/navegador/servidor.go`) y
 * `fuenteDelNavegador` (`internal/app/navegador.go`).
 *
 * Así el panel, el guion de las páginas y la tarjeta de guardar no cambian: piden
 * lo mismo y reciben lo mismo, venga la respuesta de la aplicación o de aquí.
 *
 * Dos diferencias con la aplicación, dichas porque se notan:
 *
 * - **No hay emparejamiento**: la bóveda está dentro de la propia extensión, así
 *   que no hay otro programa al que pedir permiso.
 * - **Copiar no lo hace quien tiene la bóveda.** La aplicación copia por Go y
 *   borra el portapapeles pasado el plazo; el trabajador de fondo no puede tocar
 *   el portapapeles, así que devuelve lo que hay que copiar y lo copia el panel,
 *   que es donde se ha hecho el clic. **Ese portapapeles no se borra solo**, y está
 *   dicho en `docs/seguridad.md`.
 */

import type { Boveda } from "./boveda";
import { codigoEn, leerSemilla, quedan } from "./codigos";
import { dominioDeOrigen, encaja, hostDe } from "./dominios";
import { cambiarSecreto, type Entrada } from "./entrada";
import type { Afirmacion, Cuenta, Oferta, Peticion, Respuesta } from "../protocolo";
import { llavesDe, rpIdPermitido, firmarConLlave } from "./llaves";
import {
  aBase64Url,
  BANDERAS_AL_FIRMAR,
  datosDelAutenticador,
  datosDelCliente,
  deBase64Url,
  loQueSeFirma,
} from "./afirmacion";

const PREGUNTAS_POR_MINUTO = 60;
const RELLENOS_POR_MINUTO = 12;
const ESCRITURAS_POR_MINUTO = 6;

/** Un freno de ventana fija de un minuto, como `contador` en Go. */
class Freno {
  private desde = 0;
  private n = 0;
  constructor(private readonly tope: number) {}
  cabe(ahora: number): boolean {
    if (ahora - this.desde >= 60_000) {
      this.desde = ahora;
      this.n = 0;
    }
    if (this.n >= this.tope) return false;
    this.n++;
    return true;
  }
}

/**
 * **Los frenos viven lo que vive el trabajador de fondo**, que en MV3 son unos
 * minutos. No es el freno del canal —ahí protege de otros programas de la
 * máquina—: aquí quien pregunta es la propia extensión, y el freno solo acota lo
 * que una página rara pueda pedir por el puerto de las páginas.
 */
const frenos = {
  preguntas: new Freno(PREGUNTAS_POR_MINUTO),
  rellenos: new Freno(RELLENOS_POR_MINUTO),
  escrituras: new Freno(ESCRITURAS_POR_MINUTO),
};

const mal = (motivo: Respuesta["motivo"], error: string): Respuesta => ({ ok: false, motivo, error });

export type EstadoDeLaFuente = { existe: boolean; boveda: Boveda | null };

/**
 * Contesta una petición. Lo que se escribe en la bóveda lo guarda ella, y su
 * `alGuardar` es lo que avisa a quien la use para que suba el cambio.
 */
export async function atender(p: Peticion, e: EstadoDeLaFuente, ahora = Date.now()): Promise<Respuesta> {
  if (!frenos.preguntas.cabe(ahora)) return mal("demasiado", "Demasiadas preguntas seguidas");
  if ((p.que === "rellenar" || p.que === "rellenar-codigo") && !frenos.rellenos.cabe(ahora)) {
    return mal("demasiado", "Demasiados rellenos seguidos");
  }
  if (esEscritura(p.que) && !frenos.escrituras.cabe(ahora)) {
    return mal("demasiado", "Demasiados cambios seguidos en la bóveda");
  }

  if (p.que === "estado") return { ok: true, estado: { existe: e.existe, abierta: e.boveda !== null } };
  if (p.que === "emparejar") return { ok: true, testigo: "cuenta" };

  // El origen se comprueba **antes** de mirar si hay bóveda, como en Go: una
  // dirección que no vale contesta siempre lo mismo, esté abierta o cerrada.
  let dominio: string;
  try {
    dominio = dominioDeOrigen(p.origen ?? "");
  } catch (err) {
    return mal("origen", (err as Error).message);
  }
  const b = e.boveda;
  if (!b) {
    if (!e.existe) return mal("sin-boveda", "Aquí no hay ninguna bóveda todavía");
    return mal("cerrada", "La bóveda está cerrada");
  }

  try {
    switch (p.que) {
      case "cuentas":
        return { ok: true, cuentas: cuentasDe(b, dominio) };
      case "copiar-secreto": {
        const x = entradaDe(b, p.id ?? "", dominio);
        if (!x.secreto) throw new Error("Esa entrada no tiene contraseña");
        return { ok: true, copiado: { portapapeles: 0 }, paraCopiar: x.secreto };
      }
      case "copiar-codigo": {
        const c = await codigoDe(entradaDe(b, p.id ?? "", dominio));
        return { ok: true, copiado: { portapapeles: 0, quedan: c.quedan }, paraCopiar: c.codigo };
      }
      case "rellenar": {
        const x = entradaDe(b, p.id ?? "", dominio);
        if (!x.secreto) throw new Error("Esa entrada no tiene contraseña");
        return { ok: true, relleno: { usuario: x.usuario ?? "", secreto: x.secreto } };
      }
      case "rellenar-codigo":
        return { ok: true, codigo: await codigoDe(entradaDe(b, p.id ?? "", dominio)) };
      case "ofrecer":
        return { ok: true, oferta: ofrecer(b, p.origen ?? "", dominio, p) };
      case "guardar-cuenta":
        return { ok: true, guardada: await guardarCuenta(b, p.origen ?? "", dominio, p) };
      case "actualizar-cuenta":
        return { ok: true, guardada: await actualizarCuenta(b, p.id ?? "", dominio, p) };
      // **Las llaves de acceso** (ADR 0048). Las dos empiezan por lo mismo: el
      // `rpId` que pide el sitio se comprueba contra el origen que pone el
      // navegador. `encaja` no sirve aquí y el porqué está en `llaves.ts`.
      case "llaves": {
        // **Sin `rpId` se contesta «lo que este origen podría usar»**, que es otra
        // pregunta y hace falta: para saber si hay que ofrecerse hay que mirar antes
        // de que el sitio diga nada, y una llave guardada como `ejemplo.com` sirve
        // en `login.ejemplo.com`. Preguntando por el anfitrión a secas esa llave no
        // saldría nunca.
        if (p.rpId === undefined) {
          const usables = b
            .buscar("")
            .filter((x) => x.tipo === "llave" && rpIdPermitido(x.rpId, p.origen ?? "") === x.rpId)
            .map((x) => b.ver(x.id))
            .filter((x): x is Entrada => Boolean(x))
            .filter((x) => Boolean(x.clavePrivada));
          return { ok: true, llaves: usables.map((x) => ({ id: x.id, nombre: x.nombreVisible || x.titulo })) };
        }
        const rp = rpIdPermitido(p.rpId, p.origen ?? "");
        if (!rp) return mal("no-encaja", "Ese sitio no puede pedir esa llave de acceso");
        // **`buscar` devuelve las entradas pasadas por `sinSecretos`**, y eso vacía
        // la clave privada y el identificador de credencial — que es justo por lo
        // que hay que filtrar. Se cogen las candidatas por lo que sí viaja —clase y
        // sitio— y se miran enteras con `ver`, que es la regla que este proyecto ya
        // tiene escrita y en la que se cayó igual.
        const enteras = b
          .buscar("")
          .filter((x) => x.tipo === "llave" && x.rpId === rp)
          .map((x) => b.ver(x.id))
          .filter((x): x is Entrada => Boolean(x));
        return {
          ok: true,
          llaves: llavesDe(enteras, rp, p.permitidas).map((x) => ({
            id: x.id,
            // Lo que se lee en el banner, y nada más. Ni el identificador de
            // credencial ni el de usuario: no hacen falta para elegir.
            nombre: x.nombreVisible || x.titulo,
          })),
        };
      }
      case "firmar-llave": {
        const rp = rpIdPermitido(p.rpId, p.origen ?? "");
        if (!rp) return mal("no-encaja", "Ese sitio no puede pedir esa llave de acceso");
        const x = b.ver(p.id ?? "");
        if (!x || x.tipo !== "llave" || x.papelera || x.rpId !== rp || !x.clavePrivada) {
          return mal("no-encaja", "Esa llave no es de este sitio");
        }
        return { ok: true, afirmacion: await afirmar(x, rp, p.origen ?? "", p.reto ?? "") };
      }
      case "nunca-aqui":
        if (b.soloLectura) throw new Error(SOLO_LECTURA);
        await b.excluir(dominio);
        return { ok: true };
    }
  } catch (err) {
    return mal("no-encaja", (err as Error).message);
  }
  return mal("no-entiendo", "Esfinge no sabe hacer eso");
}

const SOLO_LECTURA = "Esta bóveda es de una versión más nueva de Esfinge y aquí no se puede escribir en ella";

function esEscritura(que: Peticion["que"]): boolean {
  return que === "guardar-cuenta" || que === "actualizar-cuenta" || que === "nunca-aqui";
}

const leEncaja = (x: Entrada, dominio: string) => (x.sitios ?? []).some((s) => encaja(s, dominio));

/**
 * Firma una aserción con una llave de acceso.
 *
 * **El origen lo construye este lado**, del que puso el navegador, y jamás el que
 * mande la página: es lo que va dentro de `clientDataJSON` y lo que el sitio
 * compara. Y lo que sale son **los bytes exactos** que se han hasheado.
 */
async function afirmar(x: Entrada, rpId: string, origen: string, reto: string): Promise<Afirmacion> {
  const cliente = datosDelCliente("webauthn.get", deBase64Url(reto), new URL(origen).origin);
  const autenticador = await datosDelAutenticador(rpId, BANDERAS_AL_FIRMAR);
  const firma = await firmarConLlave(deBase64Url(x.clavePrivada ?? ""), await loQueSeFirma(autenticador, cliente));
  return {
    idCredencial: x.idCredencial ?? "",
    idUsuario: x.idUsuario,
    datosDelCliente: aBase64Url(cliente),
    datosDelAutenticador: aBase64Url(autenticador),
    firma: aBase64Url(firma),
  };
}

const cuentaDe = (x: Entrada): Cuenta => ({
  id: x.id,
  titulo: x.titulo,
  usuario: x.usuario ?? "",
  tieneCodigo: !!x.totp,
});

/** Solo credenciales y solo de ese sitio. Si tiene código se mira en la entrada entera. */
export function cuentasDe(b: Boveda, dominio: string): Cuenta[] {
  const out: Cuenta[] = [];
  for (const x of b.buscar("")) {
    if (x.tipo !== "credencial" || !leEncaja(x, dominio)) continue;
    out.push(cuentaDe(b.ver(x.id)!));
  }
  return out;
}

/** Una entrada **que sea de ese sitio**: el único sitio por el que sale un secreto. */
function entradaDe(b: Boveda, id: string, dominio: string): Entrada {
  const x = b.ver(id);
  if (!x || x.papelera) throw new Error("Esa entrada ya no está en la bóveda");
  if (!leEncaja(x, dominio)) throw new Error("Esa entrada no es de ese sitio");
  return x;
}

async function codigoDe(x: Entrada): Promise<{ codigo: string; quedan: number }> {
  if (!x.totp) throw new Error("Esa entrada no tiene código de un solo uso");
  const s = leerSemilla(x.totp);
  const t = new Date();
  return { codigo: await codigoEn(s, t), quedan: Math.floor(quedan(s, t)) };
}

const mismoUsuario = (a: string, b: string) => a.trim().toLowerCase() === b.trim().toLowerCase();

/** «brevo.com» da «Brevo». Solo una propuesta. */
export function tituloDeSitio(dominio: string): string {
  const nombre = dominio.trim().split(".")[0];
  const r = Array.from(nombre);
  if (r.length === 0) return "Cuenta nueva";
  return r[0].toUpperCase() + r.slice(1).join("");
}

/** Qué proponer tras un envío, sin escribir nada. Las reglas de `Ofrecer` en Go, en el mismo orden. */
export function ofrecer(b: Boveda, origen: string, dominio: string, e: Peticion): Oferta {
  const host = hostDe(origen);
  const nada: Oferta = { accion: "nada", sitio: host };
  const secreto = e.secreto ?? "";
  const usuarioEnvio = e.usuario ?? "";
  if (!secreto || b.soloLectura || b.excluido(dominio)) return nada;
  const delSitio = b
    .buscar("")
    .filter((x) => x.tipo === "credencial" && leEncaja(x, dominio))
    .map((x) => b.ver(x.id)!)
    .filter(Boolean);
  for (const x of delSitio) {
    if (x.secreto === secreto && (usuarioEnvio === "" || mismoUsuario(x.usuario ?? "", usuarioEnvio))) return nada;
  }
  const usuario = usuarioEnvio.trim();
  if (usuario) {
    const suya = delSitio.find((x) => mismoUsuario(x.usuario ?? "", usuario));
    if (suya) return { accion: "actualizar", sitio: host, cuentas: [cuentaDe(suya)] };
    return { accion: "guardar", sitio: host, titulo: tituloDeSitio(dominio) };
  }
  if (delSitio.length > 0) return { accion: "actualizar", sitio: host, cuentas: delSitio.map(cuentaDe) };
  return { accion: "guardar", sitio: host, titulo: tituloDeSitio(dominio) };
}

async function guardarCuenta(b: Boveda, origen: string, dominio: string, e: Peticion): Promise<Cuenta> {
  if (b.soloLectura) throw new Error(SOLO_LECTURA);
  if (!e.secreto) throw new Error("No hay ninguna contraseña que guardar");
  const host = hostDe(origen);
  if (!host) throw new Error("De esa dirección no se puede sacar un dominio");
  let titulo = (e.titulo ?? "").trim() || tituloDeSitio(dominio);
  titulo = Array.from(titulo).slice(0, 120).join("");
  const nueva: Entrada = {
    id: "",
    tipo: "credencial",
    titulo,
    usuario: (e.usuario ?? "").trim() || undefined,
    sitios: ["https://" + host],
    creada: "",
    cambiada: "",
  };
  cambiarSecreto(nueva, e.secreto, new Date());
  const puesta = await b.poner(nueva);
  return { id: "", titulo: puesta.titulo, usuario: puesta.usuario ?? "" };
}

async function actualizarCuenta(b: Boveda, id: string, dominio: string, e: Peticion): Promise<Cuenta> {
  if (b.soloLectura) throw new Error(SOLO_LECTURA);
  if (!e.secreto) throw new Error("No hay ninguna contraseña nueva");
  const x = entradaDe(b, id, dominio);
  if (x.tipo !== "credencial") throw new Error("Esa entrada no es una cuenta");
  cambiarSecreto(x, e.secreto, new Date());
  if (!x.usuario) x.usuario = (e.usuario ?? "").trim() || undefined;
  return cuentaDe(await b.poner(x));
}
