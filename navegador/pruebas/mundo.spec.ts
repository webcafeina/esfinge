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
  {
    atender,
    hay = false,
    sePuedeCrear = false,
    afirmacion = null,
    atestacion = null,
  }: {
    atender: boolean;
    hay?: boolean;
    sePuedeCrear?: boolean;
    afirmacion?: unknown;
    atestacion?: unknown;
  },
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
    class AuthenticatorAttestationResponse {}
    // El estático que los sitios preguntan para saber si ofrecen llaves. Contesta que
    // no, que es lo que diría un equipo sin biometría: así se ve si el shim lo tapa.
    PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable = function () {
      window.__llamadas.push({ que: "disponible" });
      return Promise.resolve(false);
    };
    for (const [tipo, nombres] of [
      [PublicKeyCredential, ["id", "rawId", "type", "response", "authenticatorAttachment"]],
      [AuthenticatorAssertionResponse, ["clientDataJSON", "authenticatorData", "signature", "userHandle"]],
      [AuthenticatorAttestationResponse, ["clientDataJSON", "attestationObject"]],
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
    window.AuthenticatorAttestationResponse = AuthenticatorAttestationResponse;
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
        p.postMessage({ hay: ${hay}, sePuedeCrear: ${sePuedeCrear} });
        // El otro lado, de mentira: apunta lo que le piden y contesta lo que se le
        // haya dicho. Sin afirmación, ceder — que es el caso normal.
        p.onmessage = (e) => {
          window.__pedidos.push(e.data);
          p.postMessage({
            n: e.data.n,
            afirmacion: ${JSON.stringify(afirmacion)} ?? undefined,
            atestacion: ${JSON.stringify(atestacion)} ?? undefined,
          });
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

/* -------------------------------------------------- crear una llave (P3) */

/**
 * **Con `create` y sin poder crear, se cede sin preguntar a nadie.**
 *
 * Es la puerta que hace que el 99 % de las páginas no paguen nada, y al crear importa
 * más que al firmar: `create()` exige **activación de usuario**, que dura unos
 * segundos, y esperar a un trabajador dormido puede agotarla. Lo que se cede aquí se
 * cede en la misma vuelta del bucle de eventos.
 *
 * Y se mira que **no se haya preguntado nada**, no solo que devuelva la original: con
 * una pregunta por medio la prueba pasaría igual y el riesgo seguiría ahí.
 */
test("mundo: sin poder crear, create se cede en el momento", async ({ page }) => {
  await conElShim(page, { atender: true, hay: true, sePuedeCrear: false });
  const r = await page.evaluate(async () => {
    const cc = (window as any).navigator.__cc;
    const salida = await cc.create({ publicKey: { challenge: new Uint8Array(8), rp: {}, user: {} } });
    return { salida, pedidos: (window as any).__pedidos.length };
  });
  expect(r.salida).toBe("original");
  expect(r.pedidos, "ha preguntado al otro lado antes de ceder").toBe(0);
});

/**
 * **Y las dos banderas son distintas**, que es lo que dice que no se ha reutilizado
 * una por comodidad: con llaves para este sitio pero sin poder crear, `get` atiende y
 * `create` cede. Cambiar una por la otra en el shim pone esto rojo.
 */
test("mundo: las banderas de usar y de crear no son la misma", async ({ page }) => {
  await conElShim(page, {
    atender: true,
    hay: false,
    sePuedeCrear: true,
    atestacion: {
      idCredencial: "Y3JlZC1udWV2YQ",
      datosDelCliente: "eyJ0IjoxfQ",
      objeto: "o2NmbXQ",
      datosDelAutenticador: "YXV0aA",
      publica: "cHViYQ",
    },
  });
  const r = await page.evaluate(async () => {
    const cc = (window as any).navigator.__cc;
    const creada = await cc.create({ publicKey: { challenge: new Uint8Array(8), rp: {}, user: {} } });
    const usada = await cc.get({ publicKey: { challenge: new Uint8Array(8) } });
    return {
      creadaEsNuestra: creada !== "original" && creada.id === "Y3JlZC1udWV2YQ",
      usadaEsOriginal: usada === "original",
      pedidos: (window as any).__pedidos.map((p: any) => p.crear === true),
    };
  });
  expect(r.creadaEsNuestra, "con sePuedeCrear no ha atendido create").toBe(true);
  expect(r.usadaEsOriginal, "sin llaves tenía que ceder get").toBe(true);
  // Y solo se ha preguntado una vez, la de crear: `get` cedió sin preguntar.
  expect(r.pedidos).toEqual([true]);
});

/**
 * **La credencial de crear, con la forma que el sitio espera.**
 *
 * Lo que se mira no es que exista: es que **se pueda leer**. Un objeto con el
 * prototipo puesto y sin propiedades propias da «Illegal invocation» al leer `id`, y
 * los cuatro métodos que los sitios llaman —`getAuthenticatorData`, `getPublicKey`,
 * `getPublicKeyAlgorithm` y `getTransports`— tienen que estar, o el sitio revienta
 * mirando lo que le hemos devuelto.
 */
test("mundo: con una atestación, la credencial se puede leer entera", async ({ page }) => {
  await conElShim(page, {
    atender: true,
    sePuedeCrear: true,
    atestacion: {
      idCredencial: "Y3JlZC1udWV2YQ",
      datosDelCliente: "eyJ0IjoxfQ",
      objeto: "o2NmbXQ",
      datosDelAutenticador: "YXV0aA",
      publica: "cHViYQ",
    },
  });
  const r = await page.evaluate(async () => {
    const cc = (window as any).navigator.__cc;
    const c: any = await cc.create({
      publicKey: {
        challenge: new Uint8Array([1, 2, 3]),
        rp: { id: "ejemplo.com", name: "Ejemplo" },
        user: { id: new Uint8Array([9, 9]), name: "yo@ejemplo.com", displayName: "Yo" },
        pubKeyCredParams: [{ type: "public-key", alg: -7 }],
        excludeCredentials: [{ type: "public-key", id: new Uint8Array([7]) }],
      },
    });
    const largo = (x: unknown) => (x instanceof ArrayBuffer ? x.byteLength : -1);
    return {
      id: c.id,
      tipo: c.type,
      adjunto: c.authenticatorAttachment,
      rawId: largo(c.rawId),
      cliente: largo(c.response.clientDataJSON),
      objeto: largo(c.response.attestationObject),
      autenticador: largo(c.response.getAuthenticatorData()),
      publica: largo(c.response.getPublicKey()),
      alg: c.response.getPublicKeyAlgorithm(),
      transportes: c.response.getTransports(),
      extensiones: c.getClientExtensionResults(),
      json: c.toJSON().response.attestationObject,
      // Y lo que se le pidió al otro lado: todo lo que dijo el sitio, entero.
      pedido: (window as any).__pedidos[0],
    };
  });
  expect(r.id).toBe("Y3JlZC1udWV2YQ");
  expect(r.tipo).toBe("public-key");
  expect(r.adjunto).toBe("platform");
  expect(r.rawId).toBeGreaterThan(0);
  expect(r.cliente).toBeGreaterThan(0);
  expect(r.objeto).toBeGreaterThan(0);
  expect(r.autenticador).toBeGreaterThan(0);
  expect(r.publica).toBeGreaterThan(0);
  expect(r.alg).toBe(-7);
  expect(r.transportes).toEqual(["internal", "hybrid"]);
  expect(r.extensiones).toEqual({});
  expect(r.json).toBe("o2NmbXQ");
  // **Lo que el sitio dijo llega entero**, que es lo que evita crear una llave sin
  // usuario o sin saber qué algoritmos acepta.
  expect(r.pedido).toMatchObject({
    crear: true,
    rpId: "ejemplo.com",
    titulo: "Ejemplo",
    usuario: "yo@ejemplo.com",
    algoritmos: [-7],
  });
  expect(r.pedido.excluidas).toHaveLength(1);
  expect(r.pedido.idUsuario).toBeTruthy();
});

/**
 * **Y se le dice al sitio que aquí hay autenticador de plataforma** (P3).
 *
 * Es lo que decide si un sitio ofrece crear una llave, y hay que contestar que sí o
 * Esfinge no serviría justo en el equipo que no tiene Touch ID. Se comprueba las dos
 * mitades: que con Esfinge disponible **tapa** al navegador —que aquí dice que no—, y
 * que **sin poder crear no miente**: contesta lo que conteste él.
 */
test("mundo: dice que hay autenticador de plataforma, y solo cuando lo hay", async ({ page }) => {
  await conElShim(page, { atender: true, sePuedeCrear: true });
  expect(
    await page.evaluate(() => (window as any).PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable()),
  ).toBe(true);
});

test("mundo: con las llaves apagadas, no se miente sobre el autenticador", async ({ page }) => {
  await conElShim(page, { atender: true, sePuedeCrear: false });
  const r = await page.evaluate(async () => {
    const si = await (window as any).PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
    return { si, llamoAlOriginal: (window as any).__llamadas.some((l: any) => l.que === "disponible") };
  });
  expect(r.si, "ha dicho que hay autenticador con las llaves apagadas").toBe(false);
  expect(r.llamoAlOriginal, "no ha preguntado al navegador").toBe(true);
});

/**
 * **La forma que GitHub manda de verdad al crear** (ADR 0048, P3).
 *
 * Es el caso que tiró la primera prueba en el Mac del cliente: pulsó «Add passkey» y
 * salió el diálogo del navegador con Dashlane, el llavero de Apple y Chrome — **y sin
 * Esfinge**. El shim estaba instalado y había cedido, porque GitHub manda
 * `extensions: { appidExclude, credProps }` y la regla era «cualquier clave que venga
 * en `extensions`, se cede».
 *
 * Esa regla se escribió en la P2 con el diagnóstico de `get`, donde GitHub manda
 * `extensions` **presente y vacío**. Al crear no viene vacío, y nadie lo comprobó.
 *
 * Los valores de esta prueba son los que dijo la consola del cliente el 2026-09-30,
 * no una invención: dos extensiones, `[-7, -257]` en ese orden, una llave excluida,
 * `userVerification` y `residentKey` en `required`, atestación `none` y reto de 32.
 */
test("mundo: la petición de crear de GitHub se atiende, con sus dos extensiones", async ({ page }) => {
  await conElShim(page, {
    atender: true,
    sePuedeCrear: true,
    atestacion: {
      idCredencial: "Y3JlZC1udWV2YQ",
      datosDelCliente: "eyJ0IjoxfQ",
      objeto: "o2NmbXQ",
      datosDelAutenticador: "YXV0aA",
      publica: "cHViYQ",
    },
  });
  const r = await page.evaluate(async () => {
    const cc = (window as any).navigator.__cc;
    const c: any = await cc.create({
      publicKey: {
        challenge: new Uint8Array(32).fill(3),
        rp: { id: "github.com", name: "GitHub" },
        user: { id: new Uint8Array(16).fill(4), name: "yo@ejemplo.com", displayName: "Yo" },
        pubKeyCredParams: [
          { type: "public-key", alg: -7 },
          { type: "public-key", alg: -257 },
        ],
        excludeCredentials: [{ type: "public-key", id: new Uint8Array([1, 2, 3]) }],
        authenticatorSelection: { userVerification: "required", residentKey: "required" },
        attestation: "none",
        extensions: { appidExclude: "https://github.com/u2f", credProps: true },
      },
    });
    return {
      esNuestra: c !== "original",
      extensiones: c === "original" ? null : c.getClientExtensionResults(),
      json: c === "original" ? null : c.toJSON().clientExtensionResults,
    };
  });
  expect(r.esNuestra, "ha cedido con las extensiones que GitHub manda al crear").toBe(true);
  // **Y `credProps` se contesta**, porque el sitio lo pidió y es verdad: las llaves de
  // Esfinge son todas residentes.
  expect(r.extensiones).toEqual({ credProps: { rk: true } });
  expect(r.json).toEqual({ credProps: { rk: true } });
});

/**
 * **Y con una extensión que no conocemos se sigue cediendo**, que es la mitad que
 * evita convertir la lista blanca en una puerta abierta.
 *
 * Sin esta prueba, «arreglar» lo de GitHub podía haber sido quitar la comprobación
 * entera — y entonces Esfinge contestaría a peticiones cuya forma no entiende,
 * devolviéndole al sitio algo que no cuadra con lo que pidió.
 */
test("mundo: una extensión que no conocemos sigue cediendo", async ({ page }) => {
  await conElShim(page, {
    atender: true,
    sePuedeCrear: true,
    atestacion: {
      idCredencial: "Y3JlZC1udWV2YQ",
      datosDelCliente: "eyJ0IjoxfQ",
      objeto: "o2NmbXQ",
      datosDelAutenticador: "YXV0aA",
      publica: "cHViYQ",
    },
  });
  const r = await page.evaluate(async () => {
    const cc = (window as any).navigator.__cc;
    const conRara = await cc.create({
      publicKey: {
        challenge: new Uint8Array(8),
        rp: {},
        user: {},
        extensions: { credProps: true, algoQueNoConocemos: true },
      },
    });
    return { conRara, pedidos: (window as any).__pedidos.length };
  });
  expect(r.conRara).toBe("original");
  expect(r.pedidos, "ha preguntado antes de ceder con una extensión desconocida").toBe(0);
});

/**
 * Y **sin pedir `credProps` no se contesta**, que es lo que dice la especificación: una
 * extensión que el sitio no pidió no aparece en el resultado. Devolverla igual es
 * decirle algo que no preguntó, y hay sitios que comparan lo que piden con lo que les
 * llega.
 */
test("mundo: credProps no se contesta si no se ha pedido", async ({ page }) => {
  await conElShim(page, {
    atender: true,
    sePuedeCrear: true,
    atestacion: {
      idCredencial: "Y3JlZC1udWV2YQ",
      datosDelCliente: "eyJ0IjoxfQ",
      objeto: "o2NmbXQ",
      datosDelAutenticador: "YXV0aA",
      publica: "cHViYQ",
    },
  });
  const r = await page.evaluate(async () => {
    const cc = (window as any).navigator.__cc;
    const c: any = await cc.create({
      publicKey: { challenge: new Uint8Array(8), rp: {}, user: {}, extensions: { appidExclude: "x" } },
    });
    return c.getClientExtensionResults();
  });
  expect(r).toEqual({});
});
