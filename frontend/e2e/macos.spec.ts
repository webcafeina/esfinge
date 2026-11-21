import { expect, test, type Locator, type Page } from "@playwright/test";

/**
 * El aspecto de macOS 27 Golden Gate (ADR 0068), medido.
 *
 * # Qué comprueba esto y qué no
 *
 * **Mide**: tamaños, radios y colores calculados. Eso es lo que se puede
 * contestar sin un Mac delante, y es lo que impide que la hoja se desalinee sin
 * que nadie se entere.
 *
 * **No dice si está bonito**, y en macOS hay más de lo normal que no puede decir:
 * el material de la ventana no existe fuera de un Mac, y si el pomo del
 * interruptor sale píldora o elipse tampoco se juzga en Chromium. Eso se mira en
 * las capturas (`make capturas-macos`) y, al final, en su máquina.
 *
 * # Y por qué el sistema se llama «darwin»
 *
 * Porque es `runtime.GOOS`. Escribir «macos» aquí no da ningún error: es un
 * selector de atributo válido que no casa con nada, así que **la hoja entera deja
 * de aplicarse y las capturas salen del sistema real**, que es lo que pasó el
 * 2026-11-20 y pasó por macOS hasta que alguien midió. Lo vigila
 * `TestLosSistemasDeLasHojasExisten`, en `internal/tema`.
 */
const MACOS = "/?sistema=darwin";

// Sin esto, el servidor de desarrollo recién arrancado enseña la bienvenida de la
// cuenta y aquí no hay barra lateral que medir.
test.beforeEach(async ({ request }) => {
  const r = await request.post("/api/ElegirModoLocal", { data: [] });
  expect(r.ok(), await r.text()).toBe(true);
});

/** La sección, acotada a la barra lateral: «Cifrar» nombra dos cosas. */
function seccion(page: Page, nombre: string) {
  return page.locator(".lateral").getByRole("button", { name: nombre, exact: true });
}

const alto = (l: Locator) => l.evaluate((e) => e.getBoundingClientRect().height);
const ancho = (l: Locator) => l.evaluate((e) => e.getBoundingClientRect().width);

async function enMacos(page: Page) {
  // Sin animaciones: medir a mitad de una transición da números que no son de
  // nadie.
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto(MACOS);
  await expect(page.locator("html")).toHaveAttribute("data-sistema", "darwin");
  await expect(page.locator(".lateral")).toBeVisible();
}

test("la escala de macOS 27: cuerpo de 13 con interlineado de 16", async ({ page }) => {
  await enMacos(page);

  const letra = await page.evaluate(() => {
    const s = getComputedStyle(document.body);
    return { tamano: s.fontSize, interlineado: s.lineHeight };
  });
  expect(letra.tamano).toBe("13px");
  expect(letra.interlineado).toBe("16px");
});

test("los controles y los radios son los del kit", async ({ page }) => {
  await enMacos(page);

  // 24 de alto —el «medium» del kit— contra los 28 de la base, que son macOS 26.
  // **Se mide el botón de acción y no el primero que haya**: el primero es el del
  // control segmentado, que tiene su propia altura.
  const boton = page.locator(".contenido button.principal:visible").first();
  expect(await alto(boton)).toBe(24);
  expect(await boton.evaluate((b) => getComputedStyle(b).borderRadius)).toBe("6px");

  // El grupo redondea a 10, no a los 14 de la base ni a los 8 de Fluent.
  const grupo = page.locator(".contenido .grupo:visible").first();
  expect(await grupo.evaluate((g) => getComputedStyle(g).borderRadius)).toBe("10px");
  // Y **sin borde**: en macOS un grupo se separa por su superficie, no por una
  // línea. La base le pone medio píxel.
  expect(await grupo.evaluate((g) => getComputedStyle(g).borderTopWidth)).toBe("0px");

  // La barra lateral de borde a borde: 240 de ancho y filas de 36, el «large»
  // del kit. En la base son 225 y 38.
  expect(await ancho(page.locator(".lateral"))).toBe(240);
  expect(await alto(page.locator(".lateral nav button").first())).toBe(36);
});

test("la barra de herramientas unificada mide 52 y el título va a la izquierda", async ({
  page,
}) => {
  await enMacos(page);

  const barra = page.locator(".herramientas");
  expect(await alto(barra)).toBe(52);

  // **A la izquierda, no centrado**: centrado es la cabecera de GNOME, y así es
  // como se vio que la hoja no se estaba aplicando.
  //
  // **Y se afirma eso, no una distancia al borde.** La primera versión medía cuántos
  // píxeles separaban el título del canto de la barra, lo que daba por hecho que el
  // título era lo primero que hay dentro — y al entrar el botón de plegar dejó de
  // serlo y la prueba se puso roja **sin que nada estuviera mal**. Lo que importa es
  // que el título empiece en la mitad izquierda y acabe lejos del centro, que es lo
  // que distingue esta barra de la de GNOME a cualquier ancho de ventana.
  // **Lo que se afirma es que el título va pegado al botón de plegar**, no una
  // distancia al canto de la barra ni «a la izquierda del centro».
  //
  // Las dos versiones anteriores fallaron, y las dos por lo mismo. La primera medía
  // los píxeles hasta el borde de la barra, lo que daba por hecho que el título era
  // lo primero: al entrar el botón de plegar se puso roja **sin que nada estuviera
  // mal**. La segunda comprobaba que el título quedara a la izquierda del centro —y
  // **al mutarla siguió en verde**, porque con `space-between` y un título corto
  // como «Cifrar» el centro de la barra queda igualmente a su derecha—.
  //
  // Lo que distingue de verdad esta barra de la cabecera centrada de GNOME es que el
  // título **no se despega de lo que tiene delante**. Con el margen automático
  // quitado, el hueco pasa de unos pocos píxeles a más de doscientos.
  const hueco = await barra.evaluate((e) => {
    const b = e.querySelector(".boton-lateral")!.getBoundingClientRect();
    const h = e.querySelector("h1")!.getBoundingClientRect();
    return h.x - b.right;
  });
  expect(hueco, `el título está a ${Math.round(hueco)} px del botón de plegar`).toBeLessThan(24);
  expect(hueco).toBeGreaterThan(0);
});

test("la fila elegida de la barra lateral no se resalta al pasar por encima", async ({ page }) => {
  await enMacos(page);

  // En macOS lo que marca el estado es la selección. Un gris al pasar por encima
  // compite con ella, y la base lo pone porque allí sí se hunde.
  const otra = seccion(page, "Descifrar");
  const antes = await otra.evaluate((e) => getComputedStyle(e).backgroundColor);
  await otra.hover();
  const despues = await otra.evaluate((e) => getComputedStyle(e).backgroundColor);
  expect(despues).toBe(antes);
});

test("el interruptor es la píldora del kit, más ancha que alta", async ({ page }) => {
  await enMacos(page);
  await seccion(page, "Ajustes").click();
  // **Y la pestaña, que es donde vive el control.** Ajustes se recorre por pestañas
  // y la que abre por defecto —Cuenta— no tiene ningún interruptor: esperarlo ahí
  // dice «no existe», que es el síntoma y no la causa.
  await page.getByRole("tab", { name: "Actualizaciones" }).click();

  // **Por su envoltorio y no por un `role`**: el interruptor de Esfinge es una
  // casilla a propósito, para no renombrar el control a los lectores de pantalla.
  // Buscarlo por `[role="switch"]` no encuentra nada — y eso es lo que destapó que
  // la hoja también lo estaba vistiendo con ese selector, o sea con ninguno.
  const interruptor = page.locator(".contenido .interruptor input:visible").first();
  await expect(interruptor).toBeVisible();
  // 44 × 20, el tamaño «small» del kit, el de Ajustes del Sistema. En Fluent son
  // 40 × 20, que es lo bastante parecido como para que sólo lo diga una medida.
  expect(await ancho(interruptor)).toBe(44);
  expect(await alto(interruptor)).toBe(20);

  // **El pomo es una píldora, no un círculo**, y eso es lo que lo distingue del de
  // Fluent. Va como hermano del `<input>` porque en WebKit un `<input>` no pinta
  // `::before` — y WebKit es justo el motor donde esto se va a ver.
  const pomo = page.locator(".contenido .interruptor > .pomo").first();
  expect(await ancho(pomo)).toBe(26);
  expect(await alto(pomo)).toBe(16);
  expect(await ancho(pomo)).toBeGreaterThan(await alto(pomo));
});

test("los ajustes son filas agrupadas, no tarjetas sueltas", async ({ page }) => {
  await enMacos(page);
  await seccion(page, "Ajustes").click();
  await page.getByRole("tab", { name: "Bóveda" }).click();

  const filas = page.locator(".contenido .ajuste:visible");
  await expect(filas.first()).toBeVisible();

  // Fila de 42 con 10 de relleno, como en Ajustes del Sistema. En Windows son 68
  // con 16, que es la SettingsCard.
  const caja = await filas.first().evaluate((e) => {
    const s = getComputedStyle(e);
    return {
      alto: e.getBoundingClientRect().height,
      relleno: s.paddingLeft,
      borde: s.borderTopWidth,
    };
  });
  expect(caja.alto).toBeGreaterThanOrEqual(42);
  expect(caja.relleno).toBe("10px");
  // **Sin borde**: lo que agrupa es que se tocan y el filete sangrado entre
  // ellas, no un marco por fila.
  expect(caja.borde).toBe("0px");

  // Y **se tocan**: entre una fila y la siguiente no hay hueco. Es lo que las
  // convierte en un grupo en vez de en una lista de tarjetas.
  if ((await filas.count()) >= 2) {
    const [a, b] = await Promise.all([
      filas.nth(0).evaluate((e) => e.getBoundingClientRect().bottom),
      filas.nth(1).evaluate((e) => e.getBoundingClientRect().top),
    ]);
    expect(Math.abs(b - a)).toBeLessThan(1);
  }
});

test("el control segmentado es la cápsula de macOS, con carril", async ({ page }) => {
  await enMacos(page);

  // **El carril es lo que lo hace de macOS.** En Windows esto es la SelectorBar de
  // Fluent, que no lo lleva: si alguien iguala los dos «por consistencia», esto
  // se pone rojo.
  const segmentado = page.locator(".contenido .segmentado:visible").first();
  await expect(segmentado).toBeVisible();
  const fondo = await segmentado.evaluate((e) => getComputedStyle(e).backgroundColor);
  expect(fondo).not.toBe("rgba(0, 0, 0, 0)");
  expect(await segmentado.evaluate((e) => getComputedStyle(e).borderRadius)).toBe("12px");
});

test("en macOS la barra de título la dibuja el sistema", async ({ page }) => {
  await enMacos(page);

  // La barra que Esfinge dibuja en Windows (ADR 0066) está en el marcado de los
  // cuatro sistemas **y aquí la esconde el CSS**: montarla sólo donde se usa
  // rompería que el marcado sea el mismo y la simulación por URL.
  await expect(page.locator(".barra-titulo")).toBeHidden();
});
