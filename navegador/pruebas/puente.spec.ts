import { expect, test } from "@playwright/test";
import { build } from "vite";
import path from "node:path";

/**
 * El saludo entre los dos mundos (ADR 0048).
 *
 * Se ejercita **en una página de verdad**, con el `postMessage` y los
 * `MessagePort` del navegador, porque es justo lo que un DOM simulado devolvería
 * a medida: los puertos se transfieren, y transferir es lo único que hace que el
 * canal no se pueda falsificar.
 *
 * Los dos lados corren aquí en el mismo mundo. Eso **no** prueba el salto entre
 * el mundo principal y el aislado —eso solo lo dice la extensión de verdad— pero
 * sí prueba el protocolo: quién contesta, qué se descarta y qué pasa si nadie
 * responde.
 */
let codigo = "";

test.beforeAll(async () => {
  const salida = (await build({
    configFile: false,
    logLevel: "silent",
    build: {
      write: false,
      lib: { entry: path.resolve("src/puente.ts"), formats: ["iife"], name: "P", fileName: () => "p.js" },
    },
  })) as unknown as { output: { code: string }[] }[];
  codigo = salida[0].output[0].code;
});

const conElPuente = async (page: import("@playwright/test").Page) => {
  await page.goto("about:blank");
  await page.addScriptTag({ content: codigo });
};

test("puente: quien atiende contesta, y el que saluda se queda con el puerto", async ({ page }) => {
  await conElPuente(page);
  const ida = await page.evaluate(async () => {
    const P = (window as any).P;
    let recibido: MessagePort | null = null;
    P.atenderElPuente((p: MessagePort) => (recibido = p));
    const puerto = await P.abrirPuente(window, 1000);
    // Y que el canal sirva de verdad en los dos sentidos, no solo que exista.
    const vuelta = await new Promise<string>((listo) => {
      recibido!.onmessage = (e: MessageEvent) => {
        recibido!.postMessage("eco:" + e.data);
      };
      puerto!.onmessage = (e: MessageEvent) => listo(String(e.data));
      puerto!.postMessage("hola");
    });
    return { hayPuerto: puerto !== null, hayRecibido: recibido !== null, vuelta };
  });
  expect(ida).toEqual({ hayPuerto: true, hayRecibido: true, vuelta: "eco:hola" });
});

/**
 * **Sin acuse no hay puente**, y eso es la salida de emergencia de toda la fase:
 * si el mundo aislado no está —porque el aviso de datos no se ha aceptado, porque
 * es un marco ajeno o porque algo ha fallado— la página tiene que funcionar
 * exactamente como si Esfinge no estuviera.
 */
test("puente: si nadie atiende, se rinde y no devuelve puerto", async ({ page }) => {
  await conElPuente(page);
  const p = await page.evaluate(async () => (await (window as any).P.abrirPuente(window, 150)) !== null);
  expect(p).toBe(false);
});

/**
 * Lo que la página puede fabricar: un mensaje con la marca correcta. **No cuenta**,
 * porque lo que se atiende no es la marca sino el puerto que viene con ella, y un
 * puerto no se puede inventar desde fuera.
 */
test("puente: un saludo sin puerto no engancha a nadie, y no deja sordo al puente", async ({ page }) => {
  await conElPuente(page);
  const r = await page.evaluate(async () => {
    const P = (window as any).P;
    let recibido = false;
    P.atenderElPuente(() => (recibido = true));
    window.postMessage({ esfinge: P.MARCA }, "/");
    window.postMessage({ esfinge: "otra cosa" }, "/");
    await new Promise((x) => setTimeout(x, 100));
    const tras = recibido;
    // **Y esto es lo que de verdad hay que mirar.** Comprobar solo que no ha
    // enganchado pasa en verde sin la comprobación de puertos: `e.ports[0]` sería
    // `undefined`, `start()` lanzaría y nadie se enteraría. Lo que distingue una
    // defensa de una excepción es que **después siga funcionando**.
    const puerto = await P.abrirPuente(window, 500);
    return { tras, sigueVivo: puerto !== null, engancho: recibido };
  });
  expect(r).toEqual({ tras: false, sigueVivo: true, engancho: true });
});

/**
 * Y el oyente **se quita en cuanto engancha**: un segundo saludo, forjado por la
 * página con un puerto suyo, no puede sustituir al que ya está puesto.
 */
test("puente: el segundo saludo no sustituye al primero", async ({ page }) => {
  await conElPuente(page);
  const cuantos = await page.evaluate(async () => {
    const P = (window as any).P;
    let veces = 0;
    P.atenderElPuente(() => veces++);
    await P.abrirPuente(window, 500);
    const suyo = new MessageChannel();
    window.postMessage({ esfinge: P.MARCA }, "/", [suyo.port2]);
    await new Promise((r) => setTimeout(r, 100));
    return veces;
  });
  expect(cuantos).toBe(1);
});
