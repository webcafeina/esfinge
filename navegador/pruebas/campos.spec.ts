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

/* ------------------------------------------------ el código de un solo uso */

/**
 * Dónde se escribe el código de segundo factor, **y sobre todo dónde no**.
 *
 * La mitad de la tabla son casos donde lo correcto es no tocar nada, por lo mismo
 * que con la contraseña: un código escrito en el campo equivocado es un segundo
 * factor regalado, y los formularios de la web están llenos de campos de «código»
 * que no lo son —el postal, el promocional, el CVC—.
 */
async function codigoQueSeDetecta(page: import("@playwright/test").Page, html: string) {
  await page.setContent(`<!doctype html><meta charset="utf-8">${html}`);
  await page.addScriptTag({ content: modulo });
  return page.evaluate(() => {
    // @ts-expect-error el módulo se inyecta como global en la página
    const d = Campos.buscarCodigo(document);
    if (!d) return null;
    return d.tipo === "uno"
      ? { tipo: "uno", ids: [d.campo.id] }
      : { tipo: "casillas", ids: d.campos.map((c: HTMLInputElement) => c.id) };
  });
}

const seis = (envuelta = false) =>
  [1, 2, 3, 4, 5, 6]
    .map((n) => {
      const casilla = `<input id="c${n}" type="text" inputmode="numeric" maxlength="1" style="width:32px">`;
      return envuelta ? `<span class="caja">${casilla}</span>` : casilla;
    })
    .join("");

const casosDeCodigo: {
  nombre: string;
  html: string;
  espera: { tipo: string; ids: string[] } | null;
}[] = [
  {
    nombre: "código: el campo que el sitio declara",
    html: `<form><input id="c" autocomplete="one-time-code" inputmode="numeric"></form>`,
    espera: { tipo: "uno", ids: ["c"] },
  },
  {
    nombre: "código: seis casillas de un carácter",
    html: `<form><div class="fila">${seis()}</div></form>`,
    espera: { tipo: "casillas", ids: ["c1", "c2", "c3", "c4", "c5", "c6"] },
  },
  {
    // Lo más común en la práctica: cada casilla en su caja, para dibujarle el borde.
    nombre: "código: seis casillas, cada una en su caja",
    html: `<form><div class="fila">${seis(true)}</div></form>`,
    espera: { tipo: "casillas", ids: ["c1", "c2", "c3", "c4", "c5", "c6"] },
  },
  {
    nombre: "código: por el nombre, si no hay contraseña y parece numérico",
    html: `<form><input id="c" name="totp" inputmode="numeric" maxlength="6"></form>`,
    espera: { tipo: "uno", ids: ["c"] },
  },
  {
    // «code» a secas no es un segundo factor, y es el caso que más se ve.
    nombre: "código: un código promocional no lo es",
    html: `<form><input id="p" name="promo_code" maxlength="6"></form>`,
    espera: null,
  },
  {
    nombre: "código: el de la tarjeta tampoco",
    html: `<form><input id="cvc" name="security_code" autocomplete="cc-csc" maxlength="3" inputmode="numeric"></form>`,
    espera: null,
  },
  {
    nombre: "código: ni el código postal",
    html: `<form><input id="cp" name="codigo_postal" maxlength="5" inputmode="numeric"></form>`,
    espera: null,
  },
  {
    // Cuatro casillas son un PIN, y un PIN no es un código de un solo uso.
    nombre: "código: cuatro casillas son un PIN",
    html: `<form><div>${[1, 2, 3, 4].map((n) => `<input id="c${n}" maxlength="1" style="width:32px">`).join("")}</div></form>`,
    espera: null,
  },
  {
    // Con una contraseña delante, un campo que se llama «mfa» puede ser cualquier
    // cosa: la regla del nombre es la más débil y solo vale sola.
    nombre: "código: por el nombre no, si hay una contraseña en la página",
    html: `<form><input id="u" name="usuario"><input id="p" type="password"><input id="c" name="mfa_token" inputmode="numeric"></form>`,
    espera: null,
  },
  {
    nombre: "código: un campo escondido no se rellena",
    html: `<div style="display:none"><input id="c" autocomplete="one-time-code"></div>`,
    espera: null,
  },
];

// **El de Cloudflare, copiado del de verdad** (2026-09-14, sacado de la consola en
// la pantalla de segundo factor). Seis casillas visibles que declaran todas
// `one-time-code`, **ninguna con `maxlength="1"`** —la primera acepta seis, para
// el autorrelleno del sistema— y un séptimo campo de 1×1 que guarda el valor.
// La 2.19.0 no lo detectaba por los dos lados: seis declarados era «no sé cuál»,
// y sin `maxlength="1"` no eran casillas. La señal que sí llevan todas es el
// `pattern` de una cifra.
casosDeCodigo.push({
  nombre: "código: el de Cloudflare, seis casillas sin maxlength y un campo escondido",
  html: `<form><div role="group">${[1, 2, 3, 4, 5, 6]
    .map(
      (n) =>
        `<input id="c${n}" type="text" autocomplete="one-time-code" inputmode="numeric"${
          n === 1 ? ' maxlength="6"' : ""
        } pattern="\\d{1}" style="width:65px;height:65px">`,
    )
    .join("")}<input id="oculto" type="text" autocomplete="one-time-code" inputmode="numeric" maxlength="6" pattern="\\d{6}" style="width:1px;height:1px"></div></form>`,
  espera: { tipo: "casillas", ids: ["c1", "c2", "c3", "c4", "c5", "c6"] },
});

for (const caso of casosDeCodigo) {
  test(caso.nombre, async ({ page }) => {
    expect(await codigoQueSeDetecta(page, caso.html)).toEqual(caso.espera);
  });
}

/**
 * Escribir en casillas pone **un carácter en cada una**, y no escribe nada si el
 * código no cabe: seis casillas y ocho cifras no es sitio para escribir a medias.
 */
test("código: se escribe una cifra por casilla, o nada si no cabe", async ({ page }) => {
  await page.setContent(`<!doctype html><meta charset="utf-8"><form><div>${seis()}</div></form>`);
  await page.addScriptTag({ content: modulo });
  const resultado = await page.evaluate(async () => {
    // @ts-expect-error el módulo se inyecta como global en la página
    const destino = Campos.buscarCodigo(document);
    const valores = () => destino.campos.map((c: HTMLInputElement) => c.value).join("");
    // @ts-expect-error el módulo se inyecta como global en la página
    const largo = await Campos.escribirCodigo(destino, "12345678");
    const trasLargo = valores();
    // @ts-expect-error el módulo se inyecta como global en la página
    const bueno = await Campos.escribirCodigo(destino, "482913");
    return { largo, trasLargo, bueno, valores: valores() };
  });
  expect(resultado.largo).toBe(false);
  expect(resultado.trasLargo).toBe("");
  expect(resultado.bueno).toBe(true);
  expect(resultado.valores).toBe("482913");
});
