import { expect, test } from "@playwright/test";
import { build } from "vite";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

// El paquete es de módulos («type: module»), así que aquí no hay `__dirname`.
const aqui = dirname(fileURLToPath(import.meta.url));

/**
 * Qué campo se rellena, contra páginas de verdad en un navegador de verdad.
 *
 * # Por qué esta prueba existe y por qué está escrita así
 *
 * La entrega 2 tiene una pieza que puede hacer daño y solo una: **elegir el campo
 * equivocado**. El canal ya está probado de punta a punta y no entrega la entrada
 * de un sitio a otro; nada de eso sirve si aquí se escribe la contraseña en el
 * buscador de la cabecera. Así que lo que se comprueba no es tanto que acierte
 * como que **no meta la pata**: media tabla de abajo son casos donde lo correcto
 * es no rellenar nada.
 *
 * Y se ejecuta en Chromium y no contra un DOM simulado, a propósito. La detección
 * se apoya en `getComputedStyle`, `getBoundingClientRect` y
 * `compareDocumentPosition`: en un DOM de mentira los tres devuelven lo que se les
 * haya enseñado a devolver, y lo que se estaría probando es el simulador. Es la
 * lección de los iconos con otro traje.
 */

/**
 * El módulo, compilado en memoria y sin dejar nada en el disco.
 *
 * Se compila el fuente de verdad —no una copia— para que esto no pueda quedarse
 * atrás respecto a lo que se publica.
 */
let modulo = "";

test.beforeAll(async () => {
  const salida = (await build({
    logLevel: "silent",
    // **Sin coger el `vite.config.ts` de al lado**, que es el del panel: con él,
    // esto compila el panel entero y en la página no aparece ningún `Campos`. El
    // síntoma es «Campos is not defined», que no apunta a ningún sitio.
    configFile: false,
    build: {
      write: false,
      lib: {
        entry: resolve(aqui, "../src/campos.ts"),
        formats: ["iife"],
        name: "Campos",
        fileName: () => "campos.js",
      },
    },
  })) as any;
  modulo = salida[0].output[0].code;
});

/** enUnaPagina monta el HTML, mete el módulo y devuelve lo que se detecta. */
async function loQueSeDetecta(page: import("@playwright/test").Page, html: string) {
  await page.setContent(`<!doctype html><meta charset="utf-8">${html}`);
  await page.addScriptTag({ content: modulo });
  return page.evaluate(() =>
    // @ts-expect-error el módulo se inyecta como global en la página
    Campos.buscarFormularios(document).map((f: Record<string, HTMLInputElement | null>) => ({
      usuario: f.usuario?.id ?? null,
      secreto: f.secreto?.id ?? null,
    })),
  );
}

const casos: { nombre: string; html: string; espera: { usuario: string | null; secreto: string | null }[] }[] = [
  {
    nombre: "el formulario de entrar de toda la vida",
    html: `<form>
      <input id="u" type="text" name="usuario">
      <input id="p" type="password" name="clave">
      <button>Entrar</button>
    </form>`,
    espera: [{ usuario: "u", secreto: "p" }],
  },
  {
    // El caso que más veces se ve mal hecho: el buscador de la cabecera está
    // **antes** que el formulario, así que «el último campo de texto de la
    // página» sería el usuario y «el primero» sería el buscador.
    nombre: "el buscador de la cabecera no es el usuario",
    html: `<input id="buscar" type="search" placeholder="Buscar">
    <input id="tambien-buscar" type="text" name="q">
    <form>
      <input id="u" type="email" name="correo">
      <input id="p" type="password">
    </form>`,
    espera: [{ usuario: "u", secreto: "p" }],
  },
  {
    // Media web moderna no usa `<form>`. Entonces el ámbito es la página entera y
    // la regla que salva es «el último que está antes», no «el único que hay».
    nombre: "sin form alrededor, vale el que está justo antes",
    html: `<input id="buscar" type="text" name="q">
    <div>
      <input id="u" type="text" name="login">
      <input id="p" type="password">
    </div>`,
    espera: [{ usuario: "u", secreto: "p" }],
  },
  {
    nombre: "registrarse no se rellena: dos contraseñas visibles",
    html: `<form>
      <input id="u" type="email">
      <input id="p1" type="password">
      <input id="p2" type="password">
    </form>`,
    espera: [],
  },
  {
    nombre: "una contraseña nueva no se rellena con la vieja",
    html: `<form>
      <input id="u" type="email">
      <input id="p" type="password" autocomplete="new-password">
    </form>`,
    espera: [],
  },
  {
    // Muchos sitios llevan el formulario de entrar escondido en **todas** sus
    // páginas. Rellenarlo pondría la contraseña en el DOM de cada una.
    nombre: "un formulario escondido no se rellena",
    html: `<div style="display:none">
      <input id="u" type="text">
      <input id="p" type="password">
    </div>`,
    espera: [],
  },
  {
    nombre: "ni uno de un píxel, que es la trampa clásica contra los rellenadores",
    html: `<form>
      <input id="u" type="text" style="width:1px;height:1px">
      <input id="p" type="password" style="width:1px;height:1px">
    </form>`,
    espera: [],
  },
  {
    nombre: "un campo desactivado o de solo lectura tampoco",
    html: `<form>
      <input id="u" type="text" readonly>
      <input id="p" type="password" disabled>
    </form>`,
    espera: [],
  },
  {
    // Google, Microsoft y unos cuantos más: primero el usuario, la contraseña en
    // la pantalla siguiente.
    nombre: "las que van en dos pantallas, si el sitio lo declara",
    html: `<form>
      <input id="u" type="email" autocomplete="username">
      <button>Siguiente</button>
    </form>`,
    espera: [{ usuario: "u", secreto: null }],
  },
  {
    // Y el reverso: sin contraseña delante y sin que el sitio declare nada, no hay
    // forma de saber que ese campo es un usuario. Aquí lo correcto es callarse.
    nombre: "un campo de texto suelto no es un usuario",
    html: `<input id="q" type="text" name="q" placeholder="Buscar">`,
    espera: [],
  },
  {
    nombre: "un código de un solo uso no es el usuario de la contraseña",
    html: `<form>
      <input id="otp" type="text" autocomplete="one-time-code">
      <input id="p" type="password">
    </form>`,
    espera: [{ usuario: null, secreto: "p" }],
  },
];

for (const caso of casos) {
  test(caso.nombre, async ({ page }) => {
    expect(await loQueSeDetecta(page, caso.html)).toEqual(caso.espera);
  });
}

/**
 * Escribir tiene que valer también donde el formulario es de React, que es media
 * web. **Es la lección de `CLAUDE.md` en casa ajena**: React sustituye la
 * propiedad `value` del elemento por un accesor con registro propio, así que
 * asignando directamente el evento `input` que se dispara después no le parece un
 * cambio y no llama a `onChange`. En pantalla el valor está y para la aplicación
 * el campo sigue vacío.
 *
 * Aquí se imita ese accesor a mano —no hace falta React entero para reproducirlo—
 * y se comprueba lo que de verdad importa: que el registro **no** se haya puesto
 * al día antes de tiempo cuando llega el evento.
 */
test("escribir vale aunque el campo tenga un accesor propio, como en React", async ({ page }) => {
  await page.setContent(`<!doctype html><meta charset="utf-8"><input id="p" type="password">`);
  await page.addScriptTag({ content: modulo });

  const resultado = await page.evaluate(() => {
    const campo = document.getElementById("p") as HTMLInputElement;

    // El accesor que pone React sobre el elemento concreto, con su registro.
    let registro = "";
    const delPrototipo = Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value",
    )!;
    Object.defineProperty(campo, "value", {
      get: () => delPrototipo.get!.call(campo),
      set: (v: string) => {
        registro = v; // esto es lo que React pone al día antes de tiempo
        delPrototipo.set!.call(campo, v);
      },
      configurable: true,
    });

    let cambios = 0;
    let registroAlLlegarElEvento = "";
    campo.addEventListener("input", () => {
      cambios++;
      registroAlLlegarElEvento = registro;
    });

    // @ts-expect-error el módulo se inyecta como global en la página
    Campos.escribir(campo, "s3cr3t0");
    return { valor: delPrototipo.get!.call(campo), cambios, registroAlLlegarElEvento };
  });

  expect(resultado.valor).toBe("s3cr3t0");
  expect(resultado.cambios).toBe(1);
  // Lo que hace que React se entere: cuando llega el evento, su registro todavía
  // dice lo de antes, así que el cambio le parece un cambio.
  expect(resultado.registroAlLlegarElEvento).toBe("");
});
