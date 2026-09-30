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
  { atender, hay = false }: { atender: boolean; hay?: boolean },
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
  `);
  // **`addInitScript` envuelve el código en una función**, así que el `var` del
  // paquete no llega a `window` y `window.P` salía `undefined`: el saludo no tenía
  // quien lo atendiera y el shim se rendía sin instalar nada. Se asigna a mano.
  await page.addInitScript(puente + "\nwindow.P = P;");
  if (atender) {
    await page.addInitScript(`
      window.P.atenderElPuente((p) => { window.__puerto = p; p.postMessage({ hay: ${hay} }); });
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
test("mundo: lo que no es suyo se cede, y en la misma vuelta", async ({ page }) => {
  // **Con la bandera puesta**, que si no la primera línea de `podemosAtender` corta
  // y no se llega a mirar ninguna de las otras condiciones: la prueba pasaría sin
  // ejercitar nada de lo que dice comprobar.
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
    for (const [nombre, o] of Object.entries(casos)) out[nombre] = String(await cc.get(o));
    return { out, cuantas: (window as any).__llamadas.length };
  });
  // **Y aquí hay que decir lo que esta prueba todavía NO comprueba.** Hoy ceden los
  // cuatro por el mismo sitio —no hay llaves que ofrecer—, así que quitar la
  // comprobación de `mediation` o cambiar la de `extensions` la deja en verde:
  // comprobado mutándolas. Es una fijación para cuando el banner exista, y entonces
  // el «extensions vacío» tendrá que dejar de ceder mientras los otros tres siguen
  // cediendo. Se escribe ahora para que el caso de GitHub no se pierda por el
  // camino, no porque esté cubierto.
  expect(r.cuantas).toBe(4);
  expect(Object.values(r.out)).toEqual(["original", "original", "original", "original"]);
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
