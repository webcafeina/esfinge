import { expect, test } from "@playwright/test";
import { build } from "vite";
import path from "node:path";

/**
 * El guion del mundo principal (ADR 0048).
 *
 * **Se inyecta con `addInitScript` y no con `addScriptTag`**, y no es un detalle:
 * lo que hay que probar es que llega *antes* que la página, que es lo único que
 * hace que capture la `postMessage` y los métodos de verdad. Con `addScriptTag`
 * llegaría después y la prueba diría que sí a algo que no se ha comprobado.
 *
 * Lo que se mira aquí es **que no rompa nada**, que es el riesgo de toda la fase:
 * un fallo aquí no deja el relleno a medias, deja el sitio sin poder entrar.
 */
let codigo = "";
let puente = "";

test.beforeAll(async () => {
  const compilar = async (entrada: string, nombre: string) => {
    const salida = (await build({
      configFile: false,
      logLevel: "silent",
      build: {
        write: false,
        lib: { entry: path.resolve(entrada), formats: ["iife"], name: nombre, fileName: () => "x.js" },
      },
    })) as unknown as { output: { code: string }[] }[];
    return salida[0].output[0].code;
  };
  codigo = await compilar("src/mundo.ts", "M");
  puente = await compilar("src/puente.ts", "P");
});

/**
 * Una página con el shim ya puesto, y con **un doble de `CredentialsContainer`**
 * que apunta lo que le llega: aquí no hay WebAuthn de verdad, y lo que importa no
 * es lo que devuelve sino **con qué se le llama**.
 */
async function conElShim(
  page: import("@playwright/test").Page,
  { atender, hay = false, afirmacion = null }: { atender: boolean; hay?: boolean; afirmacion?: unknown },
) {
  await page.addInitScript(`
    window.__llamadas = [];
    class CredentialsContainer {}
    CredentialsContainer.prototype.get = function (o) {
      window.__llamadas.push({ que: "get", o, esteEsCC: this instanceof CredentialsContainer });
      return Promise.resolve("original");
    };
    CredentialsContainer.prototype.create = function (o) {
      window.__llamadas.push({ que: "create", o });
      return Promise.resolve("original");
    };
    // **Que digan «[native code]» donde se les pregunta.** El shim mira con
    // \`Function.prototype.toString.call\`, que **ignora un \`toString\` propio** y
    // devuelve el fuente de verdad: poner uno encima no engaña a nadie, y con eso la
    // prueba decía que el shim no se instalaba por el motivo equivocado. Se miente
    // donde se pregunta, que además es lo que haría otro gestor de verdad.
    const nativas = new WeakSet([CredentialsContainer.prototype.get, CredentialsContainer.prototype.create]);
    const toStringDeVerdad = Function.prototype.toString;
    Function.prototype.toString = function () {
      return nativas.has(this) ? "function () { [native code] }" : toStringDeVerdad.call(this);
    };
    window.CredentialsContainer = CredentialsContainer;
    window.navigator.__cc = new CredentialsContainer();

    // **Y los dos tipos que hay que devolver**, que en about:blank no existen.
    // Con los accesores que lanzan, que es la trampa de verdad: un objeto con el
    // prototipo puesto y **sin propiedades propias** da «Illegal invocation» al leer
    // «id», y el sitio revienta al mirar lo que le hemos devuelto.
    // (Sin acentos graves: esto va dentro de un literal de plantilla y lo cerrarían.)
    class PublicKeyCredential {}
    class AuthenticatorAssertionResponse {}
    for (const [tipo, nombres] of [
      [PublicKeyCredential, ["id", "rawId", "type", "response", "authenticatorAttachment"]],
      [AuthenticatorAssertionResponse, ["clientDataJSON", "authenticatorData", "signature", "userHandle"]],
    ]) {
      for (const n of nombres) {
        Object.defineProperty(tipo.prototype, n, {
          get() { throw new TypeError("Illegal invocation"); },
          configurable: true,
        });
      }
    }
    window.PublicKeyCredential = PublicKeyCredential;
    window.AuthenticatorAssertionResponse = AuthenticatorAssertionResponse;
  `);
  // **`addInitScript` envuelve el código en una función**, así que el `var` del
  // paquete no llega a `window` y `window.P` salía `undefined`: el saludo no tenía
  // quien lo atendiera y el shim se rendía sin instalar nada. Se asigna a mano.
  await page.addInitScript(puente + "\nwindow.P = P;");
  if (atender) {
    await page.addInitScript(`
      window.__pedidos = [];
      window.P.atenderElPuente((p) => {
        window.__puerto = p;
        p.postMessage({ hay: ${hay} });
        // El otro lado, de mentira: apunta lo que le piden y contesta lo que se le
        // haya dicho. Sin afirmación, ceder — que es el caso normal.
        p.onmessage = (e) => {
          window.__pedidos.push(e.data);
          p.postMessage({ n: e.data.n, afirmacion: ${JSON.stringify(afirmacion)} ?? undefined });
        };
      });
    `);
  }
  await page.addInitScript(codigo);
  await page.goto("about:blank");
  await page.waitForFunction(() => (window as any).__listo !== undefined || true);
  await page.waitForTimeout(150);
}

/**
 * **Sin puente no se instala nada**, que es la salida de emergencia de toda la
 * fase. Está comprobado por la propiedad, no por la línea: quitar el
 * `if (!puerto) return` deja la prueba en verde porque la línea siguiente reventaría
 * igual, pero instalar con un puerto de mentira la pone roja.
 */
test("mundo: sin nadie al otro lado del puente no se instala nada", async ({ page }) => {
  await conElShim(page, { atender: false });
  // **Se espera a que el plazo del saludo venza.** Mirando antes, el shim todavía
  // está esperando y no ha instalado nada *todavía*: la prueba salía en verde
  // aunque se quitara el «si no hay puerto, no se instala» —comprobado mutándolo—.
  await page.waitForTimeout(2300);
  const parcheado = await page.evaluate(() => {
    const cc = (window as any).CredentialsContainer.prototype;
    return Function.prototype.toString.call(cc.get).includes("[native code]");
  });
  // Sigue siendo la original: nada nuestro ha entrado en la página.
  expect(parcheado).toBe(true);
});

test("mundo: con puente, se instala y no se puede quitar", async ({ page }) => {
  await conElShim(page, { atender: true });
  const r = await page.evaluate(() => {
    const cc = (window as any).CredentialsContainer.prototype;
    const d = Object.getOwnPropertyDescriptor(cc, "get")!;
    let borrado = false;
    try {
      borrado = delete cc.get;
    } catch {
      borrado = false;
    }
    let redefine = false;
    try {
      Object.defineProperty(cc, "get", { value: 1 });
      redefine = true;
    } catch {
      redefine = false;
    }
    return { escribible: d.writable, configurable: d.configurable, borrado, redefine };
  });
  expect(r).toEqual({ escribible: false, configurable: false, borrado: false, redefine: false });
});

/**
 * **Lo que de verdad importa: al ceder, la original recibe lo mismo.**
 *
 * `Object.is` y no una comparación profunda, a propósito: es la prueba que impide
 * que dentro de seis meses alguien meta un `structuredClone` «para limpiar». Un
 * clon pierde `options.signal` —el `AbortSignal` deja de abortar la llamada de
 * verdad— y rompe cualquier `getter` que el sitio haya puesto.
 */
test("mundo: al ceder, la original recibe el mismo objeto y el mismo this", async ({ page }) => {
  await conElShim(page, { atender: true });
  const r = await page.evaluate(async () => {
    const cc = (window as any).navigator.__cc;
    const opciones = { publicKey: { challenge: new Uint8Array(32), rpId: "ejemplo.com" } };
    const devuelto = await cc.get(opciones);
    const l = (window as any).__llamadas;
    return {
      devuelto,
      cuantas: l.length,
      mismoObjeto: Object.is(l[0].o, opciones),
      esteEsCC: l[0].esteEsCC,
    };
  });
  expect(r).toEqual({ devuelto: "original", cuantas: 1, mismoObjeto: true, esteEsCC: true });
});

/**
 * Las cuatro formas que se ceden **sin esperar a nadie**, que es lo que evita el
 * peor riesgo de la fase: `create()` exige activación de usuario, dura unos
 * segundos, y esperar a un trabajador dormido la agota.
 */
test("mundo: lo que no es suyo se cede sin preguntar a nadie", async ({ page }) => {
  // **Con la bandera puesta**, que si no la primera línea de `podemosAtender` corta
  // y no se llega a mirar ninguna de las otras condiciones.
  await conElShim(page, { atender: true, hay: true });
  const r = await page.evaluate(async () => {
    const cc = (window as any).navigator.__cc;
    const casos: Record<string, unknown> = {
      "sin publicKey": { password: {} },
      "mediación condicional": { publicKey: { challenge: new Uint8Array(1) }, mediation: "conditional" },
      "extensión que no entendemos": { publicKey: { challenge: new Uint8Array(1), extensions: { prf: {} } } },
      // **Presente y vacío sí es nuestro**: es lo que manda GitHub, y contarlo como
      // extensión desconocida haría que Esfinge no funcionara nunca.
      "extensions vacío": { publicKey: { challenge: new Uint8Array(1), extensions: {} } },
    };
    const out: Record<string, string> = {};
    const preguntados: string[] = [];
    for (const [nombre, o] of Object.entries(casos)) {
      const antes = (window as any).__pedidos.length;
      out[nombre] = String(await cc.get(o));
      if ((window as any).__pedidos.length > antes) preguntados.push(nombre);
    }
    return { out, cuantas: (window as any).__llamadas.length, preguntados };
  });
  // **Lo que distingue es a quién se le ha preguntado.** Los cuatro acaban cediendo
  // —el otro lado de mentira contesta sin afirmación—, así que contar cesiones no
  // dice nada. Lo que dice algo es que los tres primeros ni preguntan, y que el
  // cuarto sí: `extensions` presente y vacío es lo que manda GitHub, y contarlo como
  // extensión desconocida haría que Esfinge no funcionara nunca.
  expect(r.cuantas).toBe(4);
  expect(Object.values(r.out)).toEqual(["original", "original", "original", "original"]);
  expect(r.preguntados, "se ha preguntado por algo que no es nuestro, o no se ha preguntado por lo que sí").toEqual([
    "extensions vacío",
  ]);
});

test("mundo: si algo revienta dentro, se cede igual", async ({ page }) => {
  // Con la bandera puesta, por lo mismo: sin ella no se llega a leer `publicKey` y
  // el objeto que lanza no lanza nunca.
  await conElShim(page, { atender: true, hay: true });
  const r = await page.evaluate(async () => {
    const cc = (window as any).navigator.__cc;
    // Un objeto cuyo `publicKey` lanza al leerlo: lo más parecido a un sitio raro.
    const malo = {};
    Object.defineProperty(malo, "publicKey", {
      get() {
        throw new Error("no me leas");
      },
    });
    return String(await cc.get(malo));
  });
  expect(r).toBe("original");
});

/**
 * **El camino entero, con una firma de vuelta**: lo que se le devuelve al sitio
 * tiene que parecerse a lo que le devolvería el navegador, o el sitio lo rechaza
 * —o peor, revienta— y quien no usa Esfinge se queda sin entrar.
 */
test("mundo: con una afirmación, devuelve una credencial con la forma buena", async ({ page }) => {
  await conElShim(page, {
    atender: true,
    hay: true,
    afirmacion: {
      idCredencial: "Y3JlZC0x",
      idUsuario: "dXN1LTE",
      datosDelCliente: "Y2xpZW50ZQ",
      datosDelAutenticador: "YXV0ZW50aWNhZG9y",
      firma: "ZmlybWE",
    },
  });
  const r = await page.evaluate(async () => {
    const cc = (window as any).navigator.__cc;
    const cred: any = await cc.get({ publicKey: { challenge: new Uint8Array(32), rpId: "github.com" } });
    const texto = (b: ArrayBuffer) => new TextDecoder().decode(new Uint8Array(b));
    return {
      // No se ha cedido: el sitio recibe lo nuestro.
      cedio: (window as any).__llamadas.length > 0,
      id: cred.id,
      tipo: cred.type,
      // **Leer `id` y `response` no puede lanzar**: sin propiedades propias, los
      // accesores del prototipo dan «Illegal invocation».
      rawId: texto(cred.rawId),
      cliente: texto(cred.response.clientDataJSON),
      autenticador: texto(cred.response.authenticatorData),
      firma: texto(cred.response.signature),
      usuario: texto(cred.response.userHandle),
      extensiones: JSON.stringify(cred.getClientExtensionResults()),
      json: cred.toJSON().response.signature,
      esCredencial: cred instanceof PublicKeyCredential,
    };
  });
  expect(r).toEqual({
    cedio: false,
    id: "Y3JlZC0x",
    tipo: "public-key",
    rawId: "cred-1",
    cliente: "cliente",
    autenticador: "autenticador",
    firma: "firma",
    usuario: "usu-1",
    extensiones: "{}",
    json: "ZmlybWE",
    esCredencial: true,
  });
});
