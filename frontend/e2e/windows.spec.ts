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

/**
 * La sección, acotada a la barra lateral por lo de siempre: «Cifrar» nombra a la
 * vez una sección y el botón que cifra. Está copiada de `esfinge.spec.ts`, que
 * la tiene privada; son dos líneas y compartirlas obligaría a un módulo de
 * ayudantes que hoy no hace falta.
 */
function seccion(page: Page, nombre: string) {
  return page.locator(".lateral").getByRole("button", { name: nombre, exact: true });
}

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

test("un ajuste es una tarjeta: el texto a la izquierda y el control a la derecha", async ({
  page,
}) => {
  await enWindows(page);
  await seccion(page, "Ajustes").click();
  await page.getByRole("tab", { name: "Bóveda" }).click();

  const tarjeta = page.locator(".ajuste:visible").first();
  const rotulo = tarjeta.locator("label");
  const control = tarjeta.locator(".interruptor, select").first();

  // La SettingsCard de WinUI: 68 de alto mínimo, radio 4 y su borde de 1 px.
  const caja = await tarjeta.evaluate((t) => {
    const s = getComputedStyle(t);
    return { alto: t.getBoundingClientRect().height, radio: s.borderRadius, borde: s.borderTopWidth };
  });
  expect(caja.alto).toBeGreaterThanOrEqual(68);
  expect(caja.radio).toBe("4px");
  expect(caja.borde).toBe("1px");

  // **Y la anatomía, que es lo que se estaba cambiando**: en la base el control
  // va delante del rótulo —es la casilla de macOS— y aquí va al otro extremo. Se
  // compara dónde acaba uno y dónde empieza el otro, que es lo único que
  // distingue las dos gramáticas: el marcado es el mismo en los dos sistemas.
  const dondeElRotulo = (await rotulo.boundingBox())!;
  const dondeElControl = (await control.boundingBox())!;
  expect(dondeElControl.x).toBeGreaterThan(dondeElRotulo.x + dondeElRotulo.width);
});

test("el interruptor es la píldora de Fluent, y lo que se mueve es el pomo", async ({ page }) => {
  await enWindows(page);
  await seccion(page, "Ajustes").click();
  await page.getByRole("tab", { name: "Navegador" }).click();

  const casilla = page.getByRole("checkbox", { name: /llaves de acceso en el navegador/ });
  const pomo = page.locator(".ajuste:visible .pomo").first();

  // 40 × 20, que es la del ToggleSwitch. En la base son 18 × 18.
  const caja = await casilla.evaluate((c) => c.getBoundingClientRect());
  expect(caja.width).toBe(40);
  expect(caja.height).toBe(20);

  // **El pomo existe y es un hermano**: puesto como `::before` del `<input>`, en
  // la ventana de macOS no se pintaría y aquí sí, que es el fallo que solo se ve
  // en la máquina del cliente.
  await expect(pomo).toBeVisible();

  // Y se mueve de un extremo al otro, que es lo único que dice si está puesto.
  const izquierda = (await pomo.boundingBox())!.x;
  await casilla.click();
  await expect(casilla).not.toBeChecked();
  const derecha = (await pomo.boundingBox())!.x;
  expect(izquierda).not.toBe(derecha);

  // Encendido, el oro con la piedra encima (ADR 0021), no el azul del sistema.
  await casilla.click();
  await expect(casilla).toBeChecked();
  expect(await casilla.evaluate((c) => getComputedStyle(c).backgroundColor)).toBe(
    await variable(page, "--relleno"),
  );
  expect(await pomo.evaluate((p) => getComputedStyle(p).backgroundColor)).toBe(
    await variable(page, "--sobre-acento"),
  );
});

/**
 * Pone un runtime de Wails de mentira **antes de que cargue la página**.
 *
 * Apunta cuántas veces se llama a cada cosa, y eso es lo que se mira: un doble
 * que solo guardara el argumento no distinguiría «no me han llamado» de «me han
 * llamado». Y **no trae `EventsOn`** a propósito, porque el puente elige el canal
 * de eventos por esa propiedad: poniéndola, la ventana se quedaría esperando
 * avisos que nadie manda.
 */
async function conVentanaDeMentira(page: Page, maximizada = false) {
  await page.addInitScript((max) => {
    const llamadas: Record<string, number> = {
      WindowMinimise: 0,
      WindowToggleMaximise: 0,
      Quit: 0,
    };
    (window as unknown as { __llamadas: typeof llamadas }).__llamadas = llamadas;
    window.runtime = {
      WindowMinimise: () => void llamadas.WindowMinimise++,
      WindowToggleMaximise: () => void llamadas.WindowToggleMaximise++,
      WindowIsMaximised: () => Promise.resolve(max),
      Quit: () => void llamadas.Quit++,
    } as Window["runtime"];
  }, maximizada);
}

const llamadas = (page: Page) =>
  page.evaluate(() => (window as unknown as { __llamadas: Record<string, number> }).__llamadas);

test("la ventana lleva su propia barra de título, con los tres botones del sistema", async ({
  page,
}) => {
  await enWindows(page);

  const barra = page.locator(".barra-titulo");
  await expect(barra).toBeVisible();
  // 32, que es el alto de la barra de Windows 11. **Lo dice Go** y llega como
  // propiedad de CSS: si alguien lo escribiera también aquí, serían dos.
  expect(await alto(barra)).toBe(32);

  // Los tres, con su nombre y a la derecha del título de la ventana.
  const cerrar = page.getByRole("button", { name: "Cerrar la ventana" });
  for (const nombre of ["Minimizar la ventana", "Maximizar la ventana", "Cerrar la ventana"]) {
    const b = page.getByRole("button", { name: nombre });
    await expect(b).toBeVisible();
    expect(await ancho(b)).toBe(46);
    expect(await alto(b)).toBe(32);
  }
  const titulo = (await barra.locator(".titulo-ventana").boundingBox())!;
  expect((await cerrar.boundingBox())!.x).toBeGreaterThan(titulo.x + titulo.width);

  // **Y por dónde se agarra la ventana**, que sin esto se queda clavada en la
  // pantalla. La propiedad se hereda, así que lo que hay que comprobar es que los
  // botones estén desmarcados: si no, pulsar «Cerrar» arrastraría.
  const agarre = (l: Locator) =>
    l.evaluate((e) => getComputedStyle(e).getPropertyValue("--wails-draggable").trim());
  expect(await agarre(barra)).toBe("drag");
  expect(await agarre(cerrar)).toBe("no-drag");
});

test("cada botón de la ventana llama a lo suyo, y el doble clic maximiza", async ({ page }) => {
  await conVentanaDeMentira(page);
  await enWindows(page);

  await page.getByRole("button", { name: "Minimizar la ventana" }).click();
  expect((await llamadas(page)).WindowMinimise).toBe(1);

  await page.getByRole("button", { name: "Maximizar la ventana" }).click();
  expect((await llamadas(page)).WindowToggleMaximise).toBe(1);

  // **El doble clic lo ponemos nosotros**: Wails no lo trae —no hay un `dblclick`
  // en todo su runtime— y es la mitad de lo que la gente hace con una barra de
  // título. Se hace sobre el título, que es banda de arrastre y no un botón.
  await page.locator(".barra-titulo .titulo-ventana").dblclick();
  expect((await llamadas(page)).WindowToggleMaximise).toBe(2);

  await page.getByRole("button", { name: "Cerrar la ventana" }).click();
  expect((await llamadas(page)).Quit).toBe(1);
});

test("maximizada, el botón de en medio ofrece restaurar", async ({ page }) => {
  // Lo que dice si está maximizada es la ventana, no una cuenta nuestra: también
  // se maximiza arrastrando arriba y con el teclado, y llevando la cuenta aquí el
  // botón acabaría ofreciendo lo contrario de lo que hace.
  await conVentanaDeMentira(page, true);
  await enWindows(page);

  await expect(page.getByRole("button", { name: "Restaurar la ventana" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Maximizar la ventana" })).toHaveCount(0);
});

test("las cinco pestañas de Ajustes caben en una fila, con sus iconos de Fluent", async ({
  page,
}) => {
  await enWindows(page);
  await seccion(page, "Ajustes").click();

  // **Windows es el caso que aprieta**: su cuerpo es de 14 y el de la base de 13,
  // así que una barra que cabe aquí cabe en los cuatro. La columna está topada en
  // 560 y no crece con la ventana, que es la razón de que estas pestañas
  // estuvieran un tiempo sin iconos.
  const filas = await page
    .locator(".panel:visible .segmentado")
    .first()
    .evaluate((barra) =>
      [...barra.querySelectorAll('[role="tab"]')].map((t) =>
        Math.round(t.getBoundingClientRect().y),
      ),
    );
  expect(filas).toHaveLength(5);
  expect(new Set(filas).size, `las pestañas caen a alturas distintas: ${filas}`).toBe(1);

  // Y el glifo es el de Fluent, no el dibujado de la casa: es lo que el cliente
  // pidió —«con los correspondientes de cada sistema, claro»—.
  const boveda = page.getByRole("tab", { name: "Bóveda", exact: true }).locator("svg path");
  const esperado = ICONOS_FLUENT.boveda["16"] ?? ICONOS_FLUENT.boveda["20"];
  await expect(boveda.first()).toHaveAttribute("d", esperado[0]);
});
