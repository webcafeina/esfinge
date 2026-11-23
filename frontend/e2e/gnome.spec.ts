import { expect, test, type Locator, type Page } from "@playwright/test";
import { ICONOS_YARU } from "../src/iconos-yaru";

/**
 * El aspecto de GNOME con Yaru (ADR 0067), medido.
 *
 * # Qué comprueba esto y qué no
 *
 * **Mide**: tamaños, radios, colores calculados, dónde cae cada pieza y qué
 * dibujo lleva cada icono. Eso es lo que se puede contestar sin pantalla, y es
 * lo que impide que una hoja se desalinee sin que nadie se entere.
 *
 * **No dice si está bonito.** Eso se mira en las capturas (`make
 * capturas-gnome`) y, al final, en la Ubuntu del cliente. Una prueba que dijera
 * «se ve bien» estaría mintiendo. De hecho, **la mitad de lo que se arregló en
 * esta fase lo encontró una captura y no una prueba**: el botón de plegar
 * montado encima de la primera fila, el interruptor sin pomo, la lista de
 * ajustes partida en dos tarjetas y la fila del proyecto corrida. Lo que hacen
 * estas pruebas es que no vuelvan.
 *
 * El aspecto se pide por la URL, que es lo que sustituye a tener una Ubuntu
 * delante; que eso no exista en el binario del cliente lo vigila aparte
 * `frontend/herramientas/sin-simulacion.mjs`.
 */
const GNOME = "/?sistema=linux&escritorio=gnome";

test.beforeEach(async ({ request }) => {
  const r = await request.post("/api/ElegirModoLocal", { data: [] });
  expect(r.ok(), await r.text()).toBe(true);
});

function seccion(page: Page, nombre: string) {
  return page.locator(".lateral").getByRole("button", { name: nombre, exact: true });
}

/** Lo que cada página ha escrito en la consola, para mirarlo al acabar. */
const avisos = new WeakMap<Page, string[]>();

test.afterEach(async ({ page }) => {
  const quejas = avisos.get(page) ?? [];
  expect(quejas, quejas.join(" | ")).toEqual([]);
});

const caja = (l: Locator) => l.evaluate((e) => e.getBoundingClientRect().toJSON());
const alto = (l: Locator) => l.evaluate((e) => e.getBoundingClientRect().height);
const ancho = (l: Locator) => l.evaluate((e) => e.getBoundingClientRect().width);

/**
 * El valor de una variable CSS, resuelto por el motor.
 *
 * Se compara **contra el token y no contra un hexadecimal escrito aquí**: así,
 * el día que la paleta de Yaru cambie de neutros, esto sigue comprobando lo que
 * quería comprobar en vez de ponerse rojo por un color que ya no es el de nadie.
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

async function enGnome(page: Page) {
  // **La consola, vigilada desde aquí también.** Lo que pintan los iconos de
  // Yaru puede hacer protestar a React sin romper nada que se vea —pasó con
  // `fill-rule`, que React quiere en camello—, y entonces lo que se pone rojo
  // son las ciento y pico pruebas del fichero grande, sin relación aparente con
  // los iconos. Aquí se ve al lado de lo que lo causa.
  const quejas: string[] = [];
  page.on("console", (m) => {
    const texto = m.text();
    // **Menos el ruido de red de esta máquina**, que no dice nada del aspecto:
    // `ERR_NETWORK_CHANGED` sale a rachas en este VPS y ya tiene su sitio en
    // `deuda.md`. Se descarta **por su nombre y no por su tipo**, para que un
    // error de verdad siga cayendo aquí.
    if (texto.includes("net::ERR_NETWORK_CHANGED")) return;
    if (m.type() === "error" || m.type() === "warning") quejas.push(texto);
  });
  page.on("pageerror", (e) => quejas.push(e.message));
  avisos.set(page, quejas);

  // Sin animaciones: medir a mitad de una transición da números que no son de
  // nadie.
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto(GNOME);
  await expect(page.locator("html")).toHaveAttribute("data-sistema", "linux");
  await expect(page.locator("html")).toHaveAttribute("data-escritorio", "gnome");
  await expect(page.locator(".lateral")).toBeVisible();
}

test("la escala de libadwaita va en puntos, no en píxeles", async ({ page }) => {
  await enGnome(page);

  const letra = await page.evaluate(() => {
    const s = getComputedStyle(document.body);
    return { tamano: s.fontSize, interlineado: s.lineHeight };
  });
  // **11 pt, que el motor calcula como 14.6667px.** Va en puntos a propósito: en
  // GNOME el tamaño de la interfaz lo pone el usuario en puntos y el escritorio
  // lo escala, así que escrito en píxeles Esfinge sería la única ventana que no
  // crece cuando alguien sube la letra del sistema. Si alguien lo devuelve a
  // píxeles, esto se pone rojo.
  expect(letra.tamano).toBe("14.6667px");
  // 1,4 de interlineado sobre esos 11 pt.
  expect(letra.interlineado).toBe("20.5333px");

  // Y el chico, 9 pt, que son 12 px clavados.
  const chico = await page.evaluate(() => {
    const d = document.createElement("i");
    d.style.fontSize = "var(--texto-chico)";
    document.body.append(d);
    const t = getComputedStyle(d).fontSize;
    d.remove();
    return t;
  });
  expect(chico).toBe("12px");
});

test("los controles y los radios son los de libadwaita", async ({ page }) => {
  await enGnome(page);

  // 34 de alto contra los 28 de la base, radio 9 y **en negrita**, que es el
  // botón de libadwaita.
  const boton = page.locator(".contenido button.principal:visible").first();
  expect(await alto(boton)).toBe(34);
  const estilo = await boton.evaluate((b) => {
    const s = getComputedStyle(b);
    return { radio: s.borderRadius, peso: s.fontWeight };
  });
  expect(estilo.radio).toBe("9px");
  expect(estilo.peso).toBe("700");

  // La tarjeta redondea a 12: ni los 8 de Fluent ni los 14 de macOS.
  const grupo = page.locator(".contenido .grupo:visible").first();
  expect(await grupo.evaluate((g) => getComputedStyle(g).borderRadius)).toBe("12px");

  // El AdwNavigationSplitView: 220 de ancho y filas de 36.
  expect(await ancho(page.locator(".lateral"))).toBe(220);
  expect(await alto(page.locator(".lateral nav button").first())).toBe(36);
});

test("la fila activa lleva el gris de GNOME y el oro va en su icono", async ({ page }) => {
  await enGnome(page);

  const activa = page.locator('.lateral nav button[aria-current="page"]');
  await expect(activa).toHaveCount(1);

  // **El fondo es el gris del tema, no el acento.** En GNOME una barra lateral
  // no se colorea: lo que dice «estás aquí» es el relleno gris. Se compara
  // contra el token, no contra un color escrito aquí.
  expect(await activa.evaluate((a) => getComputedStyle(a).backgroundColor)).toBe(
    await variable(page, "--barra-encima"),
  );

  // Y el oro, que es la marca, va en el icono (ADR 0021). Sin esto la fila
  // elegida perdería el único color propio que hay en una pantalla de trabajo
  // —que fue exactamente lo que pasó al escribir la hoja de macOS, porque la
  // regla general de la hoja le gana a la de la base—.
  expect(await activa.locator(".icono").evaluate((i) => getComputedStyle(i).color)).toBe(
    await variable(page, "--oro-lateral"),
  );
});

test("el botón de plegar vive en la cabecera y no se monta sobre la primera fila", async ({
  page,
}) => {
  await enGnome(page);

  const boton = page.locator(".boton-lateral");
  await expect(boton).toBeVisible();
  const b = await caja(boton);
  // El `sidebar-show` de libadwaita: 34 de lado, dentro de la banda de 46 de la
  // cabecera.
  expect(b.height).toBe(34);
  expect(b.width).toBe(34);
  expect(b.bottom).toBeLessThanOrEqual(46);

  // **Y no toca la primera fila**, que es el fallo que se vio en la primera
  // captura: con 12 px de hueco arriba, el botón caía encima de «Cifrar» y se
  // comía sus clics. Se compara **una caja contra la otra** y no contra una
  // coordenada escrita aquí: una aserción que mida píxeles hasta un canto se
  // cae sola en cuanto algo al lado se mueve.
  const primera = await caja(page.locator(".lateral nav button").first());
  expect(b.bottom).toBeLessThanOrEqual(primera.top);
});

test("la cabecera es una AdwHeaderBar: 46, el título centrado y en negrita", async ({ page }) => {
  await enGnome(page);

  const barra = page.locator(".herramientas");
  expect(await alto(barra)).toBe(46);

  const h1 = page.locator(".herramientas h1");
  expect(await h1.evaluate((t) => getComputedStyle(t).fontWeight)).toBe("700");

  // **Centrado de verdad**, que es lo primero que se reconoce de una aplicación
  // de GNOME. Se compara el centro del título con el centro de la barra, no con
  // una coordenada: con las órdenes a un lado, mirar sólo «está a la izquierda
  // de la mitad» pasa en verde con el título descentrado.
  const cb = await caja(barra);
  const ct = await caja(h1);
  const centroBarra = cb.x + cb.width / 2;
  const centroTitulo = ct.x + ct.width / 2;
  expect(Math.abs(centroTitulo - centroBarra)).toBeLessThan(2);
});

test("el título no se escribe encima de lo que haya en la cabecera", async ({ page }) => {
  await enGnome(page);

  // **Con la cabecera de un proyecto, montada aquí.** Crear un proyecto de
  // verdad cuesta medio minuto y lo que se mide es la regla, no el flujo: se le
  // pone a la cabecera la clase que lleva dentro de un proyecto y un botón que
  // ocupa lo que ocupan los dos que van ahí.
  //
  // **Y el botón se hace lo bastante ancho como para pasar del centro**, a
  // propósito. La primera versión ponía dos con los rótulos de verdad y
  // **pasaba en verde al mutar la regla**: con la letra de respaldo de esta
  // máquina se quedaban cuatro píxeles cortos, o sea reproducía el roce y no el
  // choque. Lo que la prueba tiene que sostener es que **cuando lo del
  // principio llega al centro, el título no se escribe encima**.
  const solapa = await page.evaluate(() => {
    const barra = document.querySelector(".herramientas") as HTMLElement;
    barra.classList.add("en-proyecto");

    const b = document.createElement("button");
    b.className = "discreto";
    b.textContent = "Quién tiene acceso…";
    // Hasta pasado el centro de la barra, midiendo y no suponiendo.
    b.style.width = `${Math.round(barra.getBoundingClientRect().width * 0.6)}px`;
    // **Detrás del título y delante de `.aparte`**, que es donde van de verdad:
    // añadido al final caería pasado su `margin-left: auto`, o sea a la derecha
    // del todo, y no se cruzaría con nada.
    const aparte = barra.querySelector(".aparte");
    if (aparte) barra.insertBefore(b, aparte);
    else barra.append(b);

    const t = document.querySelector(".titulo")!.getBoundingClientRect();
    const r = b.getBoundingClientRect();
    const choca = t.left < r.right && r.left < t.right && t.top < r.bottom && r.top < t.bottom;

    b.remove();
    barra.classList.remove("en-proyecto");
    return choca;
  });

  // En GNOME esto es la política `loose` de AdwHeaderBar: el título se centra si
  // cabe y, si no, se coloca detrás de lo que hay al principio. Centrado a la
  // fuerza, el nombre del proyecto se escribía **encima** de «Salir del
  // proyecto» — y como el título no recoge clics, el botón seguía funcionando y
  // nada se ponía rojo. Lo vio una captura.
  expect(solapa, "el título se escribe encima de un botón de la cabecera").toBe(false);
});

test("los ajustes son una lista en tarjeta: las filas seguidas se tocan", async ({ page }) => {
  await enGnome(page);
  await seccion(page, "Ajustes").click();
  await page.getByRole("tab", { name: "Bóveda", exact: true }).click();

  const filas = page.locator(".contenido .ajuste:visible");
  expect(await filas.count()).toBeGreaterThanOrEqual(2);

  const a = await caja(filas.nth(0));
  const b = await caja(filas.nth(1));
  // **Se tocan.** Con el hueco del bloque puesto salían como dos tarjetas
  // sueltas una debajo de otra, que es lo contrario de una `boxed list`; se vio
  // en la captura de Ajustes.
  expect(Math.abs(b.top - a.bottom)).toBeLessThan(1);

  // Y las esquinas sólo en los extremos de la tanda: 12 arriba en la primera y
  // 0 abajo, al revés en la última.
  const esquinas = (l: Locator) =>
    l.evaluate((e) => {
      const s = getComputedStyle(e);
      return { arriba: s.borderTopLeftRadius, abajo: s.borderBottomLeftRadius };
    });
  expect(await esquinas(filas.nth(0))).toEqual({ arriba: "12px", abajo: "0px" });
  expect(await esquinas(filas.last())).toEqual({ arriba: "0px", abajo: "12px" });
});

test("el interruptor es el de libadwaita y su pomo se ve", async ({ page }) => {
  await enGnome(page);
  await seccion(page, "Ajustes").click();
  await page.getByRole("tab", { name: "Bóveda", exact: true }).click();

  const pista = page.locator(".contenido .interruptor:visible input").first();
  const c = await caja(pista);
  expect(c.width).toBe(46);
  expect(c.height).toBe(26);

  // **El pomo está a `display: none` en la base y lo enciende cada hoja.** Sin
  // eso el interruptor sale como una pastilla gris vacía, que es como se vio en
  // la captura: la pista bien y dentro nada.
  const pomo = page.locator(".contenido .interruptor:visible .pomo").first();
  const p = await caja(pomo);
  expect(p.width).toBe(20);
  expect(p.height).toBe(20);
  expect(p.x - c.x).toBeCloseTo(3, 0);
});

test("el segmentado es un AdwToggleGroup: el elegido se levanta en blanco", async ({ page }) => {
  await enGnome(page);

  const grupo = page.locator(".contenido .segmentado:visible").first();
  const elegido = grupo.locator('button[aria-selected="true"]');
  await expect(elegido).toHaveCount(1);

  // La caja en el gris de los botones y el elegido en la superficie de tarjeta.
  // **Por `aria-selected` y no por `aria-checked`**: el segmentado de Esfinge es
  // una `tablist` (ADR 0019), que es la clase de detalle por la que una hoja
  // copiada de otro proyecto no viste nada.
  expect(await grupo.evaluate((g) => getComputedStyle(g).backgroundColor)).toBe(
    await variable(page, "--boton"),
  );
  expect(await elegido.evaluate((b) => getComputedStyle(b).backgroundColor)).toBe(
    await variable(page, "--tarjeta"),
  );
  expect(await alto(elegido)).toBe(28);
});

/** En cuántas filas cae una tira de pestañas, y cuántas hay en cada una. */
async function filasDe(pestanas: Locator): Promise<number[]> {
  const total = await pestanas.count();
  const filas = new Map<number, number>();
  for (let i = 0; i < total; i++) {
    const y = Math.round((await caja(pestanas.nth(i))).y);
    filas.set(y, (filas.get(y) ?? 0) + 1);
  }
  return [...filas.values()];
}

test("las pestañas de Ajustes caben en una fila, con el glifo encima", async ({ page }) => {
  await enGnome(page);
  await seccion(page, "Ajustes").click();

  const pestanas = page.locator(".contenido .segmentado.envuelve:visible button");
  expect(await pestanas.count()).toBeGreaterThan(1);

  // **Una sola fila.** Lo eligió el cliente sobre capturas: a 11 pt con el glifo
  // al lado las seis piden 749 px y la columna da 736, así que salían dos filas
  // de tres. Con el glifo **encima**, cada pestaña mide lo que mide su palabra y
  // caben en un renglón sin tocar el ancho de la ventana.
  const filas = await filasDe(pestanas);
  expect(filas.length, `se han partido en ${filas.join(" + ")}`).toBe(1);

  // Y el glifo va **arriba**, no al lado: se compara una caja con la otra, que
  // es lo que dice la colocación sin depender de ninguna coordenada.
  const primera = pestanas.first();
  const icono = await caja(primera.locator(".icono"));
  const rotulo = await caja(primera.locator(".rotulo"));
  expect(icono.bottom).toBeLessThanOrEqual(rotulo.top + 1);
});

test("y si la ventana se estrecha se parten sin dejar una fila coja", async ({ page }) => {
  await enGnome(page);
  await seccion(page, "Ajustes").click();

  // **El mínimo de la ventana**, que es donde la fila ya no cabe: ahí la columna
  // da 516 contra los 600 de siempre. La regla de la casa es que antes de tirar
  // un rótulo o un glifo la fila se parte —«Cuenta» y «Bóveda» sin su nombre no
  // las distingue nadie—; lo que no sale solo es el reparto.
  await page.setViewportSize({ width: 760, height: 680 });
  await page.waitForTimeout(150);

  const pestanas = page.locator(".contenido .segmentado.envuelve:visible button");
  const filas = await filasDe(pestanas);
  expect(filas.length, "a 760 de ventana deberían partirse").toBeGreaterThan(1);
  expect(
    Math.max(...filas) - Math.min(...filas),
    `el reparto deja una fila coja: ${filas.join(" + ")}`,
  ).toBeLessThanOrEqual(1);
});

test("el menú de la ventana está en la cabecera y lleva lo que no está en otro sitio", async ({
  page,
}) => {
  await enGnome(page);

  // En GNOME ninguna aplicación lleva barra de menús (ADR 0071): lo que hay es
  // este botón. Quién lo lleva lo decide Go y llega como `data-menu`.
  await expect(page.locator("html")).toHaveAttribute("data-menu", "propio");
  const boton = page.getByRole("button", { name: "Menú principal" });
  await expect(boton).toBeVisible();
  await expect(boton).toHaveAttribute("aria-expanded", "false");

  // Dentro va lo que la barra tenía **y no está en ninguna otra parte**: los dos
  // enlaces de ayuda. Sin ellos, quitar la barra habría dejado la documentación
  // sin forma de llegar.
  await boton.click();
  await expect(boton).toHaveAttribute("aria-expanded", "true");
  const menu = page.getByRole("menu");
  await expect(menu.getByRole("menuitem", { name: "Documentación de Esfinge" })).toBeVisible();
  await expect(menu.getByRole("menuitem", { name: "Qué protege y qué no" })).toBeVisible();
  await expect(menu.getByRole("menuitem", { name: "Buscar actualizaciones…" })).toBeVisible();

  // Y los enlaces salen fuera, al navegador del sistema.
  await expect(menu.getByRole("menuitem", { name: "Qué protege y qué no" })).toHaveAttribute(
    "target",
    "_blank",
  );

  // Se cierra con Escape, como cualquier menú.
  await page.keyboard.press("Escape");
  await expect(menu).toHaveCount(0);
  await expect(boton).toHaveAttribute("aria-expanded", "false");
});

test("donde el sistema pone su barra de menús, el botón no existe", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/?sistema=darwin");
  await expect(page.locator("html")).toHaveAttribute("data-menu", "sistema");

  // **Y no existe, no es que esté apagado**: escondido con `display: none` se
  // queda fuera del árbol de accesibilidad y del orden de tabulación, que es lo
  // que hace que donde no se usa no estorbe. Si saliera, macOS tendría dos menús.
  await expect(page.getByRole("button", { name: "Menú principal" })).toHaveCount(0);
});

test("los atajos de las secciones siguen funcionando sin barra de menús", async ({ page }) => {
  await enGnome(page);

  // Ctrl+1 … Ctrl+6 colgaban del menú «Ver». Al quitar la barra en GNOME se
  // habrían perdido sin que nada se pusiera rojo: la pantalla sigue ahí, sólo
  // que no se llega con el teclado. Van contra el mismo `obedecer` que el menú.
  await expect(seccion(page, "Cifrar")).toHaveAttribute("aria-current", "page");

  await page.keyboard.press("Control+3");
  await expect(seccion(page, "Generar")).toHaveAttribute("aria-current", "page");

  await page.keyboard.press("Control+1");
  await expect(seccion(page, "Cifrar")).toHaveAttribute("aria-current", "page");
});

test("la regla general de botón no pisa a los botones compuestos", async ({ page }) => {
  await enGnome(page);

  // `.abrir-proyecto` es una **fila entera**, no una variante de botón: si la
  // regla general de la hoja le gana, se convierte en una pastilla gris de 34 y
  // la fila del proyecto sale corrida —se vio en la captura de Proyectos—. Es
  // el mismo fallo que costó tres rondas en Windows, y lo que lo cierra es el
  // peso `(0,0,2)` de `:where(...) body button`, no una lista de exclusiones.
  //
  // **Con un botón de usar y tirar**, no con los de la pantalla: así no depende
  // de que la bóveda tenga un proyecto dentro, y una prueba que se salta cuando
  // no lo hay no vigila nada. Es la técnica que ya usa la prueba de que ningún
  // botón cambia de tamaño al pasar el puntero.
  const fondos = await page.evaluate(() => {
    const donde = document.querySelector(".contenido") ?? document.body;
    const medir = (clase: string) => {
      const b = document.createElement("button");
      b.className = clase;
      b.textContent = "x";
      donde.append(b);
      const s = getComputedStyle(b);
      const r = { fondo: s.backgroundColor, alto: s.height };
      b.remove();
      return r;
    };
    return { suelto: medir(""), compuesto: medir("abrir-proyecto") };
  });

  // El botón de verdad sí lo viste la hoja…
  expect(fondos.suelto.alto).toBe("34px");
  // …y la fila compuesta se queda con lo suyo: fondo transparente y alto por
  // contenido, que es lo que la base le da por clase.
  expect(fondos.compuesto.fondo).toBe("rgba(0, 0, 0, 0)");
  expect(fondos.compuesto.alto).not.toBe("34px");
});

test("los iconos son los de Yaru, con su dibujo intacto", async ({ page }) => {
  await enGnome(page);

  // Se compara **el árbol que llega al DOM con el del módulo**, que es lo que
  // dice que lo que se pinta es lo que se cosechó. Importa por la licencia
  // —CC BY-SA: lo que se puede incluir sin publicar nada es lo **no adaptado**—
  // y porque seis de estos llevan un `transform`: quedarse con la `d` daría un
  // icono fuera de su caja, que es la trampa de Yaru que en Apolo volvió dos
  // veces.
  // **Se elige uno que lleve un grupo con `transform` dentro**, y no el primero
  // que haya. Con `cifrar`, que es un solo `<path>` suelto, esta prueba pasaba
  // en verde aunque `pintarNodo` dejara de pintar los hijos — o sea que no
  // vigilaba justo lo que dice vigilar. `ajustes` tiene grupo, transformación y
  // tres trazados, y está en la barra lateral.
  const esperado = ICONOS_YARU["ajustes"];
  expect(JSON.stringify(esperado.nodos), "el icono elegido ya no lleva grupo con transform").toContain(
    "transform",
  );
  const icono = seccion(page, "Ajustes").locator("svg.icono");
  await expect(icono).toHaveAttribute("viewBox", esperado.caja);

  const dibujado = await icono.evaluate((s) =>
    [...s.querySelectorAll("path")].map((p) => p.getAttribute("d")),
  );
  const delModulo: string[] = [];
  const recorrer = (n: { t: string; a?: Record<string, string>; h?: unknown[] }) => {
    if (n.t === "path" && n.a?.d) delModulo.push(n.a.d);
    (n.h ?? []).forEach((h) => recorrer(h as never));
  };
  esperado.nodos.forEach(recorrer);
  expect(dibujado).toEqual(delModulo);

  // **Y la transformación llega al DOM.** Es la mitad que de verdad importa: sin
  // ella el dibujo no sale torcido, sale **fuera de su caja** —la trampa de Yaru
  // que en Apolo costó dos vueltas— y las `d` seguirían siendo las mismas.
  const transformaciones = await icono.evaluate((s) =>
    [...s.querySelectorAll("g")].map((g) => g.getAttribute("transform")),
  );
  expect(transformaciones.filter(Boolean).length).toBeGreaterThan(0);

  // Y **16 px**, que es la reserva de `--icono`. Es la lección del `<svg>` sin
  // dimensiones que acabó midiendo doscientos.
  expect(await ancho(icono)).toBe(16);
});
