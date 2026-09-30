import { chromium, expect, test, type BrowserContext, type Page } from "@playwright/test";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

/**
 * **La prueba que protege a quien no usa Esfinge para esa cuenta** (ADR 0048).
 *
 * Es el riesgo número uno de toda la fase: hay código nuestro dentro de cada página
 * `https`, sustituyendo el método con el que un sitio identifica a la gente. Un
 * fallo aquí no deja el relleno a medias — **deja el sitio sin poder entrar**, y a
 * quien tiene su llave en el Touch ID del sistema y no en Esfinge.
 *
 * Así que se registra una llave **que no está en la bóveda**, en un autenticador
 * virtual del propio Chrome (el que usa DevTools), y se comprueba que con la
 * extensión puesta **crear y entrar siguen funcionando exactamente igual**. Sin
 * esta prueba, la P2 no se publica.
 *
 * Y se comprueba en los dos estados que importan: con el aviso de datos aceptado
 * —el shim instalado— y sin aceptarlo, donde el guion ni arranca.
 */

const EXTENSION = fileURLToPath(new URL("../dist/pruebas", import.meta.url));
const SITIO = "https://sitio.prueba/";

let contexto: BrowserContext;
let id: string;

/** Una página con dos botones: crear una llave y entrar con ella. */
const PAGINA =
  '<!doctype html><meta charset="utf-8"><title>Entrar</title>' +
  '<h1>Un sitio con llaves de acceso</h1>' +
  "<script>\n" +
  "const b64 = (b) => btoa(String.fromCharCode(...new Uint8Array(b))).replace(/\\+/g,'-').replace(/\\//g,'_').replace(/=+$/,'');\n" +
  "window.crear = async () => {\n" +
  "  const c = await navigator.credentials.create({ publicKey: {\n" +
  "    challenge: new Uint8Array(32).fill(1),\n" +
  "    rp: { id: 'sitio.prueba', name: 'Sitio' },\n" +
  "    user: { id: new Uint8Array(16).fill(2), name: 'yo@sitio.prueba', displayName: 'Yo' },\n" +
  "    pubKeyCredParams: [{ type: 'public-key', alg: -7 }],\n" +
  "    authenticatorSelection: { authenticatorAttachment: 'platform', residentKey: 'required' },\n" +
  "  }});\n" +
  "  return { id: c.id, tipo: c.type, tieneAtestacion: b64(c.response.attestationObject).length > 0 };\n" +
  "};\n" +
  "window.entrar = async (permitida) => {\n" +
  "  const c = await navigator.credentials.get({ publicKey: {\n" +
  "    challenge: new Uint8Array(32).fill(3),\n" +
  "    rpId: 'sitio.prueba',\n" +
  "    userVerification: 'discouraged',\n" +
  "    allowCredentials: permitida ? [{ type: 'public-key', id: Uint8Array.from(atob(permitida.replace(/-/g,'+').replace(/_/g,'/')), (x) => x.charCodeAt(0)) }] : [],\n" +
  "    extensions: {},\n" +
  "  }});\n" +
  "  return { id: c.id, tipo: c.type, firma: b64(c.response.signature).length > 0, cliente: new TextDecoder().decode(c.response.clientDataJSON) };\n" +
  "};\n" +
  "</script>";

test.beforeAll(async () => {
  contexto = await chromium.launchPersistentContext(mkdtempSync(join(tmpdir(), "esfinge-llaves-")), {
    channel: "chromium",
    headless: true,
    args: [`--disable-extensions-except=${EXTENSION}`, `--load-extension=${EXTENSION}`],
  });
  await contexto.route("https://*.prueba/**", (ruta) => ruta.fulfill({ contentType: "text/html", body: PAGINA }));
  const trabajador = contexto.serviceWorkers()[0] ?? (await contexto.waitForEvent("serviceworker"));
  id = new URL(trabajador.url()).host;
});

test.afterAll(async () => {
  await contexto?.close();
});

/** Una pestaña con un autenticador virtual del navegador, como el de DevTools. */
async function conAutenticador(): Promise<Page> {
  const p = await contexto.newPage();
  const cdp = await contexto.newCDPSession(p);
  await cdp.send("WebAuthn.enable", { enableUI: false });
  await cdp.send("WebAuthn.addVirtualAuthenticator", {
    options: {
      protocol: "ctap2",
      transport: "internal",
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true,
    },
  });
  await p.goto(SITIO);
  return p;
}

async function aceptarElAviso() {
  const p = await contexto.newPage();
  await p.goto(`chrome-extension://${id}/panel.html`);
  await p.click("#aceptar");
  await p.close();
}

test.describe.serial("las llaves de acceso con la extensión puesta", () => {
  /**
   * **Sin aceptar el aviso de datos**, el guion de la página ni arranca, así que el
   * mundo principal no recibe acuse y no instala nada. Es el caso más parecido a «no
   * tener Esfinge», y tiene que funcionar igual.
   */
  test("sin el aviso aceptado, crear y entrar funcionan como siempre", async () => {
    const p = await conAutenticador();
    const creada = await p.evaluate(() => (window as any).crear());
    expect(creada.tipo).toBe("public-key");
    expect(creada.tieneAtestacion).toBe(true);

    const entrada = await p.evaluate((x) => (window as any).entrar(x), creada.id as string);
    expect(entrada.id).toBe(creada.id);
    expect(entrada.firma).toBe(true);
    await p.close();
  });

  /**
   * **Y con el aviso aceptado y el shim instalado, igual.** Es la prueba que da
   * nombre a todo esto: la bóveda está cerrada y no hay ninguna llave de Esfinge
   * para ese sitio, así que hay que ceder — y ceder significa que el autenticador
   * del navegador contesta como si no estuviéramos.
   */
  test("con el shim instalado y sin llaves nuestras, el navegador sigue entrando", async () => {
    await aceptarElAviso();
    const p = await conAutenticador();

    // Que el shim está de verdad ahí, o esta prueba no diría nada.
    const parcheado = await p.evaluate(
      () => !/\[native code\]/.test(Function.prototype.toString.call(CredentialsContainer.prototype.get)),
    );
    expect(parcheado, "el shim no se ha instalado: esta prueba no está comprobando nada").toBe(true);

    const creada = await p.evaluate(() => (window as any).crear());
    expect(creada.tipo).toBe("public-key");

    const entrada = await p.evaluate((x) => (window as any).entrar(x), creada.id as string);
    expect(entrada.id).toBe(creada.id);
    expect(entrada.firma).toBe(true);
    // **Y lo que firma es el autenticador del navegador**, no Esfinge: el origen del
    // `clientDataJSON` lo pone él y el tipo es el de siempre.
    expect(entrada.cliente).toContain('"origin":"https://sitio.prueba"');
    expect(entrada.cliente).toContain('"type":"webauthn.get"');
    await p.close();
  });

  /**
   * Y sin `allowCredentials`, que es el camino de credencial descubrible: es otra
   * rama del shim —ahí sí se le preguntaría a Esfinge— y también tiene que ceder.
   */
  test("y sin allowCredentials tampoco se estorba", async () => {
    const p = await conAutenticador();
    const creada = await p.evaluate(() => (window as any).crear());
    const entrada = await p.evaluate(() => (window as any).entrar(null));
    expect(entrada.id).toBe(creada.id);
    expect(entrada.firma).toBe(true);
    await p.close();
  });
});
