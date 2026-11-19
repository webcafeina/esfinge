import { expect, test, type Locator, type Page } from "@playwright/test";
import { ICONOS_FLUENT } from "../src/iconos-fluent";

/**
 * El aspecto de Windows 11 (ADR 0064), medido.
 *
 * # Qué comprueba esto y qué no
 *
 * **Mide**: tamaños, radios, colores calculados y qué trazado lleva cada icono.
 * Eso es lo que se puede contestar sin pantalla, y es lo que impide que una hoja
 * se desalinee sin que nadie se entere.
 *
 * **No dice si está bonito.** Eso se mira en las capturas (`make
 * capturas-windows`) y, al final, en la máquina del cliente. Una prueba que
 * dijera «se ve bien» estaría mintiendo.
 *
 * El aspecto se pide por la URL, que es lo que sustituye a tener un Windows
 * delante; que eso no exista en el binario del cliente lo vigila aparte
 * `frontend/herramientas/sin-simulacion.mjs`.
 */
const WINDOWS = "/?sistema=windows";

// Sin esto, el servidor de desarrollo recién arrancado enseña la bienvenida de la
// cuenta y aquí no hay barra lateral que medir. Es el mismo preámbulo que el
// resto de las pruebas de la interfaz.
test.beforeEach(async ({ request }) => {
  const r = await request.post("/api/ElegirModoLocal", { data: [] });
  expect(r.ok(), await r.text()).toBe(true);
});

const alto = (l: Locator) => l.evaluate((e) => e.getBoundingClientRect().height);
const ancho = (l: Locator) => l.evaluate((e) => e.getBoundingClientRect().width);

/**
 * El valor de una variable CSS, resuelto por el motor.
 *
 * Se compara **contra el token y no contra un hexadecimal escrito aquí**: así,
 * el día que la paleta de Windows cambie de neutros, esto sigue comprobando lo
 * que quería comprobar —«esta fila lleva el acento diluido»— en vez de ponerse
 * rojo por un color que ya no es el de nadie.
 */
async function variable(page: Page, nombre: string): Promise<string> {
  return page.evaluate((n) => {
    const d = document.createElement("i");
    d.style.background = `var(${n})`;
    document.body.append(d);
    const c = getComputedStyle(d).backgroundColor;
    d.remove();
    return c;
  }, nombre);
}

async function enWindows(page: Page) {
  // Sin animaciones: medir a mitad de una transición da números que no son de
  // nadie. Es la misma razón por la que las capturas esperan antes de retratar.
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto(WINDOWS);
  await expect(page.locator("html")).toHaveAttribute("data-sistema", "windows");
  await expect(page.locator(".lateral")).toBeVisible();
}

test("la escala de Fluent: cuerpo de 14 con interlineado de 20", async ({ page }) => {
  await enWindows(page);

  const letra = await page.evaluate(() => {
    const s = getComputedStyle(document.body);
    return { tamano: s.fontSize, interlineado: s.lineHeight };
  });
  // Body 14/20 de Fluent. En la base son 13 y 1,45, que es la escala de macOS:
  // con ella la ventana se lee apretada al lado de una aplicación de Windows.
  expect(letra.tamano).toBe("14px");
  expect(letra.interlineado).toBe("20px");
});

test("los controles y los radios son los de WinUI", async ({ page }) => {
  await enWindows(page);

  // 32 de alto, contra los 28 de la base. **Se mide el botón de acción y no el
  // primero que haya**: el primero es el del control segmentado, que tiene su
  // propia altura, y medirlo decía 24 — o sea que la prueba habría estado
  // comprobando otra cosa y además fallando por el motivo equivocado.
  const boton = page.locator(".contenido button.principal:visible").first();
  expect(await alto(boton)).toBe(32);
  expect(await boton.evaluate((b) => getComputedStyle(b).borderRadius)).toBe("4px");

  // La tarjeta redondea a 8, no a los 14 de macOS.
  const grupo = page.locator(".contenido .grupo:visible").first();
  expect(await grupo.evaluate((g) => getComputedStyle(g).borderRadius)).toBe("8px");

  // El NavigationView abierto: 240 de ancho y filas de 40.
  expect(await ancho(page.locator(".lateral"))).toBe(240);
  expect(await alto(page.locator(".lateral nav button").first())).toBe(40);
});

test("la fila activa se tiñe del acento y lleva su barra al lado", async ({ page }) => {
  await enWindows(page);

  const activa = page.locator('.lateral nav button[aria-current="page"]');
  await expect(activa).toHaveCount(1);

  // **Teñida, no rellena**: en Fluent la selección es el acento muy diluido y
  // quien dice «estás aquí» es la barra de la izquierda. Esto se comprueba
  // contra el token, no contra un color escrito a mano.
  expect(await activa.evaluate((a) => getComputedStyle(a).backgroundColor)).toBe(
    await variable(page, "--relleno-tenue"),
  );

  // Y la barra: 3 × 16, en el acento. Sin ella la fila activa no se distingue,
  // porque el tinte solo no basta — y el oro a 3 px sobre un fondo claro da
  // 1,37:1, así que aquí manda la piedra.
  const barra = await activa.evaluate((a) => {
    const s = getComputedStyle(a, "::before");
    return { ancho: s.width, alto: s.height, color: s.backgroundColor };
  });
  expect(barra.ancho).toBe("3px");
  expect(barra.alto).toBe("16px");
  expect(barra.color).toBe(await variable(page, "--acento"));
});

test("los iconos son los de Fluent, no los dibujados de la casa", async ({ page }) => {
  await enWindows(page);

  // Se compara el trazado pintado contra el del módulo cosechado: así, si
  // alguien vuelve a cosechar y un icono cambia de forma, esto lo dice. Y si el
  // componente deja de mirar el sistema, también.
  // **El primero**, y no «el svg de esa fila»: cuando la bóveda está cerrada la
  // fila lleva además el candado, así que sin esto la prueba pasa sola y cae en
  // la tanda completa —donde alguna prueba de antes ha dejado la bóveda abierta o
  // cerrada—. Es la fuga de estado de siempre, vista desde el otro lado.
  const boveda = page
    .locator(".lateral nav button", { hasText: "Bóveda" })
    .locator("svg")
    .first()
    .locator("path");
  const esperado = ICONOS_FLUENT.boveda["16"] ?? ICONOS_FLUENT.boveda["20"];
  await expect(boveda.first()).toHaveAttribute("d", esperado[0]);

  // Los de Fluent son de relleno; los nuestros, de trazo. Si esto se invirtiera,
  // el icono se vería como una mancha o no se vería.
  const svg = page.locator(".lateral nav button", { hasText: "Bóveda" }).locator("svg").first();
  expect(await svg.evaluate((s) => s.getAttribute("fill"))).toBe("currentColor");

  // Y la reserva de tamaño sigue en pie: lo que esto impide es el icono de
  // 200 px que una vez se comió una fila entera.
  expect(await ancho(page.locator(".contenido .icono").first())).toBeLessThanOrEqual(24);
});

test("la marca sigue siendo nuestra: el botón de acción es oro con piedra encima", async ({
  page,
}) => {
  await enWindows(page);

  // El acento **no es el de Windows** (ADR 0021): el azul del sistema se
  // descartó a propósito, y eso no cambia porque la ventana se vista de Fluent.
  const accion = page.locator(".contenido button.principal:visible").first();
  const colores = await accion.evaluate((b) => {
    const s = getComputedStyle(b);
    return { fondo: s.backgroundColor, letra: s.color };
  });
  expect(colores.fondo).toBe(await variable(page, "--relleno"));
  expect(colores.letra).toBe(await variable(page, "--sobre-acento"));
});

// **Aquí vivía una prueba que comparaba el árbol del DOM entre dos cargas**, para
// afirmar que el marcado es el mismo en todos los sistemas. Pasaba sola y caía en
// la tanda completa, y el motivo merece quedarse: comparar dos cargas es comparar
// **dos momentos**. Entre una y otra había aparecido la banda de aviso de versión
// nueva y el candado de la bóveda, que son estado, no sistema.
//
// Esa afirmación se hace ahora donde es estable: leyendo el fuente, en
// `TestElAspectoNoSeDecideEnElMarcado` (internal/app/puente_test.go). Lo que hay
// que impedir es un `{esWindows() && <div/>}` en el marcado, y eso se ve mejor en
// el código que en una pantalla.
