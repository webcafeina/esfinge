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

  // **A la izquierda, no centrado**: centrado es la cabecera de GNOME, y así es como
  // se vio que la hoja no se estaba aplicando. Lo que se mide es **el hueco desde el
  // canto de la barra**, que es exactamente lo que pone `--hueco-barra`.
  //
  // Van tres formulaciones y las dos anteriores fallaron por lo mismo: **medían algo
  // que dependía de lo que hubiera al lado**. La primera daba por hecho que el título
  // era lo primero dentro de la barra, y se puso roja sola al entrar el botón de
  // plegar; la segunda comprobaba que quedara a la izquierda del centro, y **al
  // mutarla siguió en verde**, porque con un título corto como «Cifrar» el centro de
  // la barra queda igualmente a su derecha. La tercera —ésta— volvió a ser posible en
  // cuanto el botón de plegar salió de la barra: ahora **nada le precede**, así que su
  // hueco es el del kit y nada más.
  const hueco = await page.locator(".herramientas").evaluate((e) => {
    const b = e.getBoundingClientRect();
    return e.querySelector("h1")!.getBoundingClientRect().x - b.x;
  });
  // El hueco del kit son 18; se deja margen por el redondeo y por si el `h1` trae lo
  // suyo. Centrado pasa de los doscientos, así que la frontera no es delicada.
  expect(hueco, `el título está a ${Math.round(hueco)} px del canto de la barra`).toBeLessThan(40);
  expect(hueco).toBeGreaterThan(8);
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

// **La selección del segmentado se desliza hasta la opción elegida.**
//
// Lo pidió el cliente comparando con Apolo: «cuando se cambia entre elementos de un
// menú horizontal debe haber animaciones». El fondo de la elegida es un `::before`
// del contenedor cuya posición escribe `segmentados.ts`, así que lo que se comprueba
// es **que esas medidas sigan a la opción elegida** — que es lo que el CSS anima—.
// Mirar la animación no se puede desde aquí; mirar que el fondo se muda, sí.
test("la selección del segmentado se desliza hasta la opción elegida", async ({ page }) => {
  await enMacos(page);
  await page.locator(".lateral").getByRole("button", { name: "Ajustes", exact: true }).click();

  const barra = page.locator(".panel:visible .segmentado").first();
  await expect(barra).toBeVisible();

  // Sin medir no hay fondo que deslizar, y entonces todo lo demás pasaría solo.
  await expect(barra).toHaveAttribute("data-medido", "si");

  const donde = () =>
    barra.evaluate((s) => {
      const b = s.querySelector<HTMLElement>('[aria-selected="true"]')!;
      const e = getComputedStyle(s);
      return {
        fondoX: e.getPropertyValue("--sel-x").trim(),
        fondoY: e.getPropertyValue("--sel-y").trim(),
        fondoAncho: e.getPropertyValue("--sel-w").trim(),
        botonX: `${b.offsetLeft}px`,
        botonY: `${b.offsetTop}px`,
        botonAncho: `${b.offsetWidth}px`,
        rotulo: (b.textContent || "").trim(),
      };
    });

  const antes = await donde();
  expect(antes.fondoX, "el fondo no está sobre la opción elegida").toBe(antes.botonX);
  expect(antes.fondoAncho).toBe(antes.botonAncho);

  // Y a otra pestaña: el fondo tiene que haberse mudado con ella.
  //
  // **Se mira dónde cae en los dos ejes, no sólo a lo ancho.** Estas pestañas se
  // parten en dos filas cuando no caben, así que la última puede quedar **debajo de
  // la primera y en su misma x**: comparando sólo la horizontal, el fondo parecería no
  // haberse movido. Pasó en cuanto entró la sexta pestaña.
  const otras = barra.getByRole("tab");
  const cuantas = await otras.count();
  expect(cuantas, "hace falta más de una pestaña para que haya nada que deslizar").toBeGreaterThan(1);
  await otras.nth(cuantas - 1).click();

  const despues = await donde();
  expect(despues.rotulo, "no se ha cambiado de pestaña").not.toBe(antes.rotulo);
  expect(despues.fondoX, "el fondo se ha quedado en la pestaña de antes").toBe(despues.botonX);
  expect(despues.fondoY).toBe(despues.botonY);
  expect(
    `${despues.fondoX},${despues.fondoY}`,
    "el fondo no se ha movido de donde estaba",
  ).not.toBe(`${antes.fondoX},${antes.fondoY}`);
});

// **La cápsula de plegar vive sobre la cabeza de la barra lateral, y viaja.**
//
// En macOS 27 ese botón no es un hijo de la barra de herramientas: flota pegado al
// canto derecho de la barra lateral y, al plegarla, **se muda junto a los
// semáforos**. El cliente lo dijo comparándolo con Apolo —«el botón de colapso no
// está en el mismo sitio»—, y metido en la fila se parecía sin serlo.
//
// Se mide **dónde acaba**, no dónde empieza: lo que lo define es que esté pegado al
// canto de la barra, y eso no cambia aunque un día la cápsula mida otra cosa.
test("la cápsula de plegar va al canto de la barra lateral y se muda al plegarla", async ({
  page,
}) => {
  await enMacos(page);

  const capsula = page.locator(".boton-lateral");
  const caja = () => capsula.evaluate((e) => e.getBoundingClientRect());
  const anchoLateral = await page
    .locator(".lateral")
    .evaluate((e) => e.getBoundingClientRect().width);

  const abierta = await caja();
  expect(abierta.width, "la cápsula no mide los 36 del kit").toBe(36);
  expect(abierta.height).toBe(36);
  // Pegada al canto: su final queda a 8 px del borde de la barra lateral.
  expect(
    anchoLateral - abierta.right,
    `la cápsula está a ${Math.round(anchoLateral - abierta.right)} px del canto de la barra`,
  ).toBeLessThan(12);

  // Y al plegar, se muda junto a los semáforos y el título le deja sitio.
  //
  // **El hueco se mide dentro de la barra, no en la pantalla.** Al plegar, la columna
  // de la barra lateral se queda en cero y **la barra de herramientas entera se corre
  // a la izquierda**, así que comparar posiciones absolutas dice lo contrario de lo
  // que pasa: el título «retrocede» de 258 a 136 estando más adentro de su barra que
  // antes. Lo descubrió esta misma prueba fallando por el motivo equivocado.
  const hueco = async () =>
    page.locator(".herramientas").evaluate((e) => {
      const b = e.getBoundingClientRect();
      return e.querySelector("h1")!.getBoundingClientRect().x - b.x;
    });

  const huecoAntes = await hueco();
  await capsula.click();
  await expect(page.locator(".ventana")).toHaveAttribute("data-panel", "compacto");
  // La transición son 250 ms; se espera a que llegue en vez de dormir un rato.
  await expect.poll(async () => Math.round((await caja()).x)).toBe(88);
  await expect.poll(hueco).toBeGreaterThan(huecoAntes);

  // Y lo que de verdad importa: que no se pisen. Los semáforos los dibuja el sistema
  // encima de todo, así que el título tiene que empezar después de la cápsula.
  const finCapsula = (await caja()).right;
  const tituloX = await page.locator(".herramientas h1").evaluate((e) => e.getBoundingClientRect().x);
  expect(tituloX, "el título se ha quedado encima de la cápsula").toBeGreaterThan(finCapsula);
});

// **Con vidrio, elegir un tema distinto del que tiene el Mac le devuelve el fondo a
// la barra lateral.**
//
// Bajo vidrio la barra lateral no lleva fondo propio: el material del sistema **es**
// el fondo (ADR 0017). Pero ese material lo elige macOS por la apariencia de la
// ventana, que sigue a la del equipo, y Wails sólo deja elegirla **al arrancar**. Así
// que al elegir aquí un tema distinto del que tiene el Mac, el contenido se pinta del
// elegido y la barra lateral se queda del otro. Lo vio el cliente con la 2.76.0: «el
// modo oscuro no coge la parte de la barra lateral izquierda».
//
// Se ejercita **por la pantalla**, eligiendo en Ajustes como se elige de verdad: la
// regla de cuándo el material no cuadra vive en `tema.ts`, y una prueba que la
// repitiera aquí estaría comprobándose a sí misma.
test("con vidrio, un tema distinto del equipo le devuelve el fondo a la barra lateral", async ({
  page,
}, info) => {
  await page.goto("/?sistema=darwin&vidrio=si");
  await expect(page.locator(".lateral")).toBeVisible();

  const delEquipo = info.project.name === "oscuro" ? "Oscuro" : "Claro";
  const elOtro = info.project.name === "oscuro" ? "Claro" : "Oscuro";
  const fondo = () =>
    page.locator(".lateral").evaluate((e) => getComputedStyle(e).backgroundColor);
  const elegir = (n: string) =>
    page.locator(".panel:visible").getByRole("tab", { name: n, exact: true }).click();

  await page.locator(".lateral").getByRole("button", { name: "Ajustes", exact: true }).click();
  await page.locator(".panel:visible").getByRole("tab", { name: "Apariencia" }).click();

  // El del equipo: el material sí cuadra, así que la barra sigue siendo vidrio.
  await elegir(delEquipo);
  await expect(page.locator("html")).not.toHaveAttribute("data-material", /.*/);
  expect(await fondo(), "con el tema del equipo la barra lateral tapa el material").toBe(
    "rgba(0, 0, 0, 0)",
  );

  // Y el otro: el material se queda del revés, así que la barra recupera su fondo.
  await elegir(elOtro);
  await expect(page.locator("html")).toHaveAttribute("data-material", "ajeno");
  expect(
    await fondo(),
    "con un tema distinto del equipo la barra lateral se queda enseñando el material",
  ).not.toBe("rgba(0, 0, 0, 0)");
});
