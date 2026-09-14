import { expect, test } from "@playwright/test";
import { build } from "vite";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const aqui = dirname(fileURLToPath(import.meta.url));

/**
 * El código de un solo uso escrito en **el componente de verdad** de Cloudflare.
 *
 * # Por qué existe
 *
 * La 2.19.0 no rellenaba el código en Cloudflare, y el cliente lo vio a la
 * primera. La causa era la detección —seis casillas que declaran `one-time-code` y
 * ninguna con `maxlength="1"`—, y eso lo prueba `campos.spec.ts` con el formulario
 * copiado de la consola. Pero encontrar las casillas no dice que escribirlas
 * funcione: el componente es de React y **lo que importa es lo que él cree que
 * vale**, no lo que se ve en las casillas.
 *
 * Así que se prueba contra el componente mismo —`OTPField` de Base UI, el paquete,
 * en React—, mirando su estado y si el botón de verificar se activa.
 *
 * **Y esta prueba ya corrigió una suposición**: leyendo el código de Base UI
 * parecía que escribir las seis casillas seguidas no le valdría, y se escribió un
 * rodeo. Aquí se vio que sí le valía, y el rodeo se quitó. Leer el código de una
 * biblioteca dice qué hace; lo que no dice es cómo lo hace junto al resto, y eso
 * solo lo dice ejecutarlo.
 */

async function compilar(entrada: string, nombre: string): Promise<string> {
  const salida = (await build({
    configFile: false,
    logLevel: "silent",
    define: { "process.env.NODE_ENV": JSON.stringify("production") },
    build: {
      write: false,
      minify: false,
      lib: { entry: entrada, formats: ["iife"], name: nombre, fileName: () => `${nombre}.js` },
    },
  })) as any;
  return salida[0].output[0].code;
}

let campos = "";
let formulario = "";

test.beforeAll(async () => {
  campos = await compilar(resolve(aqui, "../src/campos.ts"), "Campos");
  formulario = await compilar(resolve(aqui, "paginas/otp-base-ui.ts"), "Formulario");
});

test("el código entra en el campo de Base UI y el formulario se entera", async ({ page }) => {
  const errores: string[] = [];
  page.on("pageerror", (e) => errores.push(String(e)));

  await page.setContent(`<!doctype html><meta charset="utf-8"><div id="app"></div>`);
  await page.addScriptTag({ content: formulario });
  await expect(page.locator("#verificar")).toBeDisabled();
  await page.addScriptTag({ content: campos });

  const hecho = await page.evaluate(async () => {
    // @ts-expect-error el módulo se inyecta como global en la página
    const destino = Campos.buscarCodigo(document);
    if (!destino) return null;
    // @ts-expect-error el módulo se inyecta como global en la página
    const puesto = await Campos.escribirCodigo(destino, "482913");
    return {
      tipo: destino.tipo,
      casillas: destino.tipo === "casillas" ? destino.campos.length : 1,
      puesto,
    };
  });

  expect(hecho).toEqual({ tipo: "casillas", casillas: 6, puesto: true });
  await expect(page.locator("#valor")).toHaveText("482913");
  await expect(page.locator("#verificar")).toBeEnabled();
  expect(errores).toEqual([]);
});

