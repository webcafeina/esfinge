import { expect, test } from "@playwright/test";

/**
 * No comprueba nada: recorre la interfaz sacando capturas para poder mirarla
 * antes de compilar. Se ejecuta a propósito, con
 * «pnpm exec playwright test capturas --project=claro».
 *
 * Vive aquí y no en un script suelto para aprovechar que la configuración ya
 * levanta el Go de verdad y Vite: así las capturas salen de la aplicación
 * funcionando, no de una maqueta.
 *
 * # El aspecto de cada sistema (ADR 0064)
 *
 * Con `CAPTURAS_SISTEMA=windows` el recorrido entero se hace **con el aspecto de
 * ese sistema**, pidiéndoselo por la URL (ver `frontend/src/plataforma.ts`), y las
 * capturas van a su propia carpeta. Eso es lo único que permite mirar Windows,
 * GNOME y KDE desde un servidor sin pantalla, que es donde se escribe esto, y es
 * sobre lo que el cliente elige: en Apolo las decisiones de aspecto se tomaron
 * así, generando **las dos opciones** y mirándolas.
 *
 * Y los dos temas salen gratis: son los dos proyectos de Playwright.
 */
const SISTEMA = process.env.CAPTURAS_SISTEMA ?? "";
const ESCRITORIO = process.env.CAPTURAS_ESCRITORIO ?? "";
const VIDRIO = process.env.CAPTURAS_VIDRIO === "si";

/** La carpeta: `capturas/windows-vidrio`, `capturas/linux-kde`, `capturas`… */
const CARPETA =
  process.env.CAPTURAS_EN ??
  (SISTEMA
    ? `capturas/${SISTEMA}${ESCRITORIO ? `-${ESCRITORIO}` : ""}${VIDRIO ? "-vidrio" : ""}`
    : "capturas");

/** La raíz con lo que se le pide a la interfaz, para que `goto` sea siempre igual. */
function raiz(extra = ""): string {
  const q = new URLSearchParams(extra);
  if (SISTEMA) q.set("sistema", SISTEMA);
  if (ESCRITORIO) q.set("escritorio", ESCRITORIO);
  if (VIDRIO) q.set("vidrio", "si");
  const cola = q.toString();
  return cola ? `/?${cola}` : "/";
}
test.describe("Capturas", () => {
  test.skip(!process.env.CAPTURAS, "Solo cuando se piden con CAPTURAS=1");

  test("recorrido", async ({ page }, info) => {
    const tema = info.project.name;
    const donde = CARPETA;
    const clave = (p: typeof page) => p.locator("input[type=password]:visible");

    const foto = (n: string) => page.screenshot({ path: `${donde}/${n}-${tema}.png` });

    await page.setViewportSize({ width: 980, height: 680 });
    await page.goto(raiz());

    // Con el servidor de desarrollo recién arrancado, lo primero es la bienvenida
    // de la cuenta. Se retrata y se elige «En este ordenador», que es lo que
    // recorre el resto.
    const bienvenida = page.getByRole("button", { name: "Usar en este ordenador" });
    // Se espera a que haya algo: preguntar en el acto si se ve, con la página aún
    // pintándose, decía que no y el recorrido se quedaba en la bienvenida.
    await bienvenida.or(page.getByLabel("Qué quieres cifrar")).first().waitFor();
    // **Se pregunta por el número y no por la visibilidad**: entre el `waitFor` y
    // el `isVisible` cabe el final de la transición, y entonces se contestaba que no
    // está, no se pulsaba, y el recorrido se quedaba en la bienvenida —con el fallo
    // apareciendo tres pantallas más adelante—. `click` ya espera a que se pueda.
    if (await bienvenida.count()) {
      await foto("00-bienvenida");
      await bienvenida.click();
    }

    // El rótulo de «Modo desarrollo» es cierto aquí y mentira en la portada del
    // repositorio, que es donde acaban estas capturas.
    if (process.env.CAPTURAS_SIN_DEV) {
      await page.addStyleTag({ content: ".herramientas .aparte{visibility:hidden}" });
    }

    await page.getByLabel("Qué quieres cifrar").fill("postgres://usuario:secreto@host/basededatos");
    await clave(page).fill("caballo grapa batería correcto");
    await page.waitForTimeout(500);
    await foto("1-cifrar");

    await page.locator(".contenido").getByRole("button", { name: "Cifrar", exact: true }).click();
    await page.waitForSelector(".resultado", { timeout: 20_000 });
    await page.waitForTimeout(600);
    await foto("2-resultado");

    await page.getByRole("tab", { name: "Ficheros" }).click();
    await page.locator(".soltar").click();
    await page.waitForTimeout(700);
    await foto("3-ficheros");

    await page.locator(".lateral").getByRole("button", { name: "Generar", exact: true }).click();
    await page.waitForSelector(".resultado", { timeout: 20_000 });
    await page.waitForTimeout(400);
    await foto("4-generar");

    await page.locator(".lateral").getByRole("button", { name: "Historial", exact: true }).click();
    await page.waitForTimeout(500);
    await foto("5-historial");

    // **Y Ajustes, pestaña a pestaña.** Es donde vive la mitad de los textos de la
    // aplicación y donde van a caer las tarjetas de ajuste de cada sistema, así que
    // sin esto la galería no enseña justo lo que hay que mirar. Hasta hoy el
    // recorrido se paraba en el historial.
    await page.locator(".lateral").getByRole("button", { name: "Ajustes", exact: true }).click();
    await page.waitForTimeout(400);
    for (const pestana of ["Cuenta", "Bóveda", "Navegador", "Agentes", "Actualizaciones"]) {
      // Las pestañas son `role="tab"`, no botones: pedirlas como botones no
      // encuentra ninguna y el bucle se saltaba Ajustes entero, en silencio.
      const boton = page.locator(".contenido").getByRole("tab", { name: pestana, exact: true });
      if (!(await boton.count())) continue;
      await boton.first().click();
      await page.waitForTimeout(350);
      await foto(`6-ajustes-${pestana.toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "")}`);
    }
  });

  /**
   * La pantalla de desbloquear con la huella (fase C, ADR 0044).
   *
   * Va aparte del recorrido porque necesita una bóveda, y sobre todo porque **hay
   * que mirarla**: el latido y el halo no los comprueba ninguna aserción, y esta
   * pantalla se rehízo justo por eso —el cliente vio un botón donde tenía que
   * haber una huella—.
   */
  test("desbloquear con la huella", async ({ page }, info) => {
    const tema = info.project.name;
    const donde = CARPETA;
    const foto = (n: string) => page.screenshot({ path: `${donde}/${n}-${tema}.png` });
    const maestra = "caballo grapa batería correcto";
    const dentro = page.locator(".contenido");
    const boton = (n: string) => dentro.getByRole("button", { name: n, exact: true });

    await page.setViewportSize({ width: 980, height: 680 });
    await page.goto(raiz());

    const bienvenida = page.getByRole("button", { name: "Usar en este ordenador" });
    await bienvenida.or(page.getByLabel("Qué quieres cifrar")).first().waitFor();
    if (await bienvenida.count()) await bienvenida.click();

    await page.locator(".lateral").getByRole("button", { name: "Bóveda", exact: true }).click();
    // **Se espera a que haya algo antes de preguntar cuál hay.** `isVisible` no
    // espera: con la pantalla aún pintándose contesta que no a las dos, no se crea
    // ni se abre nada, y lo que falla es la línea de después. Es la misma trampa
    // que ya está escrita arriba, en la bienvenida del recorrido.
    await boton("Crear la bóveda")
      .or(boton("Abrir la bóveda"))
      .or(page.locator("#boveda-buscar"))
      .first()
      .waitFor({ timeout: 20_000 });
    if (await boton("Crear la bóveda").isVisible().catch(() => false)) {
      await page.locator("#boveda-maestra").fill(maestra);
      await page.locator("#boveda-maestra-2").fill(maestra);
      await boton("Crear la bóveda").click();
      await page.getByText("La he apuntado en un sitio seguro").click();
      await boton("Continuar").click();
    } else if (await boton("Abrir la bóveda").isVisible().catch(() => false)) {
      await page.locator("#boveda-llave").fill(maestra);
      await boton("Abrir la bóveda").click();
    }
    await page.locator("#boveda-buscar").waitFor({ timeout: 20_000 });

    // **La sugerencia se pide, no se espera: esta prueba se prepara su propio punto
    // de partida.**
    //
    // «La bóveda lo ofrece sola la primera vez» es verdad **una vez por bóveda**: la
    // marca es `DesbloqueoSugeridoPara`, va por bóveda y la pone cualquiera que
    // conteste —activar desde Ajustes cuenta como contestar—. Y el Go es uno, con una
    // sola bóveda y una sola carpeta de configuración para los dos temas, así que el
    // tema claro contesta y **en oscuro la tarjeta no vuelve a salir**: fallaba
    // siempre en el segundo, y nunca corriéndolo solo. Se borra la marca y se quita el
    // desbloqueo, que son las dos mitades de lo que `Sugerir` pide. No hace falta
    // ninguna costura nueva: las preferencias ya cruzan el puente enteras.
    // **El orden importa: primero se quita el desbloqueo y después se borra la
    // marca.** Al revés no sirve, porque quitarlo vuelve a apuntar que ya se
    // contestó — y eso costó una pasada: con la marca borrada antes, la tarjeta
    // seguía sin salir y parecía que el borrado no funcionaba.
    const puesto = page.locator("#desbloqueo-del-sistema");
    await page.locator(".lateral").getByRole("button", { name: "Ajustes", exact: true }).click();
    await page.getByRole("tab", { name: "Bóveda" }).click();
    await puesto.waitFor({ state: "visible", timeout: 20_000 });
    if (await puesto.isChecked()) {
      await puesto.click();
      await page.waitForTimeout(1200);
    }
    // Por el mismo puente que usa la ventana, con los argumentos en una lista.
    const antes = await (await page.request.post("/api/VerPreferencias", { data: [] })).json();
    await page.request.post("/api/GuardarPreferencias", {
      data: [{ ...antes, desbloqueoSugeridoPara: "" }],
    });
    // **Y se comprueba que la preparación ha funcionado, antes de esperar nada.**
    // Sin esto, lo único que se ve cuando falla es «la tarjeta no sale», que es el
    // síntoma y no dice cuál de las cuatro condiciones de `Sugerir` falta. Puesto
    // aquí, dijo a la primera que la preparación estaba bien **y el fallo era otro**.
    const desbloqueo = await (
      await page.request.post("/api/EstadoDelDesbloqueo", { data: [] })
    ).json();
    expect(desbloqueo, JSON.stringify(desbloqueo)).toMatchObject({ sugerir: true });

    // **Y hay que recargar, porque la tarjeta no vuelve a preguntar.** Se dibuja con
    // lo que Go contestó **al montarse** y las secciones se esconden en vez de
    // desmontarse, así que ir a Ajustes y volver no la hace preguntar de nuevo: con
    // la marca ya borrada seguía creyendo que no había nada que ofrecer. La bóveda
    // sigue abierta tras recargar —eso vive en Go—, así que no hay que teclear nada;
    // lo que sí hay que hacer es **volver a entrar en Bóveda**, porque al recargar la
    // aplicación abre en «Cifrar».
    await page.reload();
    await page.locator(".lateral").getByRole("button", { name: "Bóveda", exact: true }).click();
    await page.locator("#boveda-buscar").waitFor({ timeout: 20_000 });

    await page.locator("#sugerencia-activar").waitFor({ state: "visible", timeout: 20_000 });
    await page.waitForTimeout(400);
    await foto("6a-sugerencia");
    await page.locator("#sugerencia-activar").screenshot({ path: `${donde}/6b-sugerencia-cerca-${tema}.png`, scale: "css" });
    await page.locator(".sugerencia").screenshot({ path: `${donde}/6c-sugerencia-tarjeta-${tema}.png`, scale: "css" });

    // **La casilla de desbloquear va PRIMERO, y el orden no es un gusto: es el
    // único que el producto permite.**
    //
    // Esto estaba escrito al revés —activar en Ajustes, quitarlo, y entonces
    // retratar la casilla— con el argumento de que «la casilla solo sale si no está
    // puesto». Eso es media condición. `Sugerir` pide dos cosas: que no esté puesto
    // **y que nadie haya contestado todavía** —`DesbloqueoSugeridoPara`, por bóveda—,
    // y **activarlo desde Ajustes es contestar**: `ActivarDesbloqueo` llama a
    // `noVolverAOfrecerA`. Quitarlo después recupera la primera mitad y no la
    // segunda, así que la casilla no volvía a salir nunca y el `waitFor` agotaba sus
    // 20 s. Las dos pantallas **no pueden coexistir en el tiempo**, y por eso se
    // retrata la de desbloquear antes de tocar el interruptor.
    await page.locator(".lateral").getByRole("button", { name: "Bóveda", exact: true }).click();
    await boton("Cerrar la bóveda").click();
    await page.locator("#boveda-activar-al-abrir").waitFor({ state: "visible", timeout: 20_000 });
    await page.waitForTimeout(300);
    await foto("6d-casilla-al-abrir");
    await page.locator("#boveda-llave").fill(maestra);
    await boton("Abrir la bóveda").click();
    await page.locator("#boveda-buscar").waitFor({ timeout: 20_000 });

    // Y ahora el interruptor de Ajustes, que es lo que deja el desbloqueo puesto
    // para las capturas de abajo.
    await page.locator(".lateral").getByRole("button", { name: "Ajustes", exact: true }).click();
    // **Y la pestaña, que es lo otro que faltaba.** Ajustes se recorre por pestañas
    // desde la 2.46.0 y el interruptor del desbloqueo vive en «Bóveda», que no es la
    // que abre por defecto: esperarlo sin abrirla agotaba los 20 s y dejaba este
    // recorrido entero sin sacar ni una foto. Estaba apuntado en `deuda.md` como
    // «probablemente se quedaron atrás cuando Ajustes pasó a pestañas», y era eso.
    await page.getByRole("tab", { name: "Bóveda" }).click();
    const interruptor = page.locator("#desbloqueo-del-sistema");
    await interruptor.waitFor({ state: "visible", timeout: 20_000 });
    if (!(await interruptor.isChecked())) {
      await interruptor.click();
      await page.waitForTimeout(1200);
    }
    await foto("6-ajustes-desbloqueo");

    // **En reposo**: el llavero dice que no, así que la pantalla se queda con la
    // huella quieta. Si dijera que sí abriría sola y no habría nada que retratar.
    await page.request.get("/api/_llavero?dice=no");
    await page.locator(".lateral").getByRole("button", { name: "Bóveda", exact: true }).click();
    await boton("Cerrar la bóveda").click();
    await page.locator("#boveda-con-el-sistema").waitFor({ state: "visible", timeout: 20_000 });
    await page.waitForTimeout(500);
    await foto("7-huella-reposo");
    // Y de cerca, que es donde se juzga el trazo: a tamaño de pantalla completa
    // esto son setenta píxeles y no se ve si se lee como una huella o como un
    // arcoíris. La primera versión era lo segundo.
    await page.locator("#boveda-con-el-sistema").screenshot({ path: `${donde}/7b-huella-cerca-${tema}.png`, scale: "css" });

    // **Esperando**: con el diálogo del sistema delante. Aquí no hay diálogo, así
    // que se congela la clase a mano para poder ver el latido a medio camino.
    await page.locator("#boveda-con-el-sistema").evaluate((b) => b.classList.add("esperando"));
    await page.waitForTimeout(900);
    await foto("8-huella-esperando");

    // Se deja como estaba.
    await page.request.get("/api/_llavero?dice=si");
    await page.locator("#boveda-llave").fill(maestra);
    await boton("Abrir la bóveda").click();
    await page.locator("#boveda-buscar").waitFor({ timeout: 20_000 });
    await page.locator(".lateral").getByRole("button", { name: "Ajustes", exact: true }).click();
    await page.locator("#desbloqueo-del-sistema").click();
    await page.waitForTimeout(1200);
  });

  /**
   * El dato personal (ADR 0047): el glifo nuevo en las pestañas, el formulario y
   * la ficha.
   *
   * **Va dentro del `describe`, y eso no es orden sino lo único que la apaga.** El
   * `test.skip(!CAPTURAS)` de arriba es del bloque, así que una prueba escrita
   * debajo del cierre corre siempre — y ésta corre **la primera de toda la tanda**,
   * porque `capturas.spec.ts` va antes por orden alfabético. Escrita fuera, creaba
   * la bóveda del 5173 con la contraseña maestra de aquí, que no es la de
   * `esfinge.spec.ts`; como Go deja la bóveda abierta entre pruebas, cuarenta y
   * cinco no se enteraban y **solo caían las dos que de verdad teclean la maestra**
   * —borrar la bóveda y abrirla con el sistema—, con un «esa no es la contraseña»
   * que no señalaba a ningún sitio. Tiró la publicación de la 2.30.0.
   *
   * Va aquí y no en una aserción porque **lo que hay que saber de un dibujo
   * nuevo no se comprueba, se mira**: el glifo del dato personal convive con el
   * de la identidad en la misma fila y son los dos una persona, así que lo único
   * que dice si se distinguen a quince píxeles es verlos juntos. Ya pasó con la
   * huella dactilar, que salía de 72 de ancho y 0 de alto sin que nada fallara.
   */
  test("un dato personal", async ({ page }, info) => {
    const tema = info.project.name;
    const donde = CARPETA;
    const foto = (n: string) => page.screenshot({ path: `${donde}/${n}-${tema}.png` });
    const maestra = "caballo grapa batería correcto";
    const dentro = page.locator(".contenido");
    const boton = (n: string) => dentro.getByRole("button", { name: n, exact: true });
    await page.setViewportSize({ width: 980, height: 680 });
    await page.goto(raiz());
    const bienvenida = page.getByRole("button", { name: "Usar en este ordenador" });
    await bienvenida.or(page.getByLabel("Qué quieres cifrar")).first().waitFor();
    if (await bienvenida.count()) await bienvenida.click();
    await page.locator(".lateral").getByRole("button", { name: "Bóveda", exact: true }).click();
    await boton("Crear la bóveda")
      .or(boton("Abrir la bóveda"))
      .or(page.locator("#boveda-buscar"))
      .first()
      .waitFor({ timeout: 20_000 });
    if (await boton("Crear la bóveda").isVisible().catch(() => false)) {
      await page.locator("#boveda-maestra").fill(maestra);
      await page.locator("#boveda-maestra-2").fill(maestra);
      await boton("Crear la bóveda").click();
      await page.getByText("La he apuntado en un sitio seguro").click();
      await boton("Continuar").click();
    } else if (await boton("Abrir la bóveda").isVisible().catch(() => false)) {
      await page.locator("#boveda-llave").fill(maestra);
      await boton("Abrir la bóveda").click();
    }
    await page.locator("#boveda-buscar").waitFor({ timeout: 20_000 });
    // Las pestañas, que es donde el glifo nuevo tiene que distinguirse del de la
    // identidad. De cerca y con la escala de CSS, que a tamaño de pantalla no se
    // ve lo que se ve en un Mac.
    // La segunda `.boveda-barra`: la primera es la de «Nueva» y «Cerrar la bóveda».
    await page.locator(".boveda-barra").nth(1).screenshot({ path: `${donde}/8a-clases-${tema}.png`, scale: "css" });
    // Y con la de llaves activa, que es el glifo nuevo y el que convive con el de
    // credencial: los dos son una llave y lo que los separa es la orientación.
    await page.getByRole("tab", { name: "Llaves de acceso", exact: true }).click();
    await page.locator(".boveda-barra").nth(1).screenshot({ path: `${donde}/8d-clases-llave-${tema}.png`, scale: "css" });
    await page.getByRole("tab", { name: "Todo", exact: true }).click();
    await boton("Nueva").click();
    await dentro.getByRole("tab", { name: "Dato personal", exact: true }).click();
    await page.locator("#boveda-titulo").fill("Correo electrónico 1");
    await page.locator("#boveda-personal-nombre").fill("Álvaro Cabezas");
    await page.locator("#boveda-correo").fill("alvaro@webcafeina.com");
    await page.locator("#boveda-telefono").fill("+34 600 11 22 33");
    await page.locator("#boveda-nacimiento").fill("1980-01-01");
    for (const [id, valor] of [
      ["destinatario", "Álvaro Cabezas"],
      ["calle", "Calle Mayor 1"],
      ["edificio", "Portal B"],
      ["piso", "3"],
      ["puerta", "B"],
      ["codigo-postal", "28001"],
      ["ciudad", "Madrid"],
      ["provincia", "Madrid"],
      ["pais", "España"],
    ]) {
      await page.locator(`#boveda-${id}`).fill(valor);
    }
    await foto("8b-dato-personal-formulario");
    await boton("Guardar").click();
    await page.getByText("Correo electrónico 1").first().click();
    await page.waitForTimeout(300);
    await foto("8c-dato-personal-ficha");

    // La red wifi (ADR 0049): el formulario y, sobre todo, **la ficha con el código**.
    // Es lo que no dice ninguna aserción: si el QR se ve del tamaño que una cámara lee,
    // si el blanco del código pelea con el fondo en tema oscuro y si el aviso de que ese
    // dibujo es la contraseña se lee antes de que alguien lo enseñe a nadie.
    // **Se vuelve recargando, no con un botón.** La ficha no tiene ninguno que lleve a
    // la lista, y un `click()` sobre uno que no existe se come treinta segundos de plazo
    // antes de decirlo. La bóveda sigue abierta: la tiene Go, no la página.
    const aLaLista = async () => {
      await page.reload();
      await page.locator(".lateral").getByRole("button", { name: "Bóveda", exact: true }).click();
      await page.locator("#boveda-buscar").waitFor({ timeout: 20_000 });
    };
    await aLaLista();
    await boton("Nueva").click();
    await dentro.getByRole("tab", { name: "Wi-Fi", exact: true }).click();
    await page.locator("#boveda-titulo").fill("La oficina");
    await page.locator("#boveda-ssid").fill("WEBCAFEINA");
    await page.locator("#boveda-clave-wifi").fill("una-clave-de-ejemplo");
    await foto("8e-wifi-formulario");
    await boton("Guardar").click();
    await page.getByText("La oficina").first().click();
    await page.waitForTimeout(300);
    await foto("8f-wifi-ficha");

    // Y la tira de clases con la de Wi-Fi activa, que es el glifo nuevo.
    await aLaLista();
    await page.getByRole("tab", { name: "Wi-Fi", exact: true }).click();
    await page.locator(".boveda-barra").nth(1).screenshot({ path: `${donde}/8g-clases-wifi-${tema}.png`, scale: "css" });

    // El apartado de proyectos (ADR 0050): vacío, creando y con uno dentro.
    await page.locator(".lateral").getByRole("button", { name: "Proyectos", exact: true }).click();
    await page.waitForTimeout(300);
    await foto("9a-proyectos-vacio");
    await boton("Nueva bóveda de proyecto").click();
    // **El nombre lleva el tema dentro.** El Go es uno y los proyectos se quedan
    // entre tandas y **entre temas**, así que con «Acme» a secas el segundo tema
    // creaba un segundo proyecto con el mismo nombre y el clic de abajo fallaba por
    // modo estricto —dos botones y ninguno elegible—. Es la misma familia que «una
    // prueba que conmuta de bóveda tiene que dejar abierta la de siempre»: lo que el
    // primer tema deja, el segundo se lo encuentra.
    const proyecto = `Acme ${tema}`;
    await page.locator("#proyecto-nombre").fill(proyecto);
    await foto("9b-proyectos-creando");
    await boton("Crear").click();
    await page.waitForTimeout(400);
    await foto("9c-proyectos-con-uno");

    // Lo que se hace con un proyecto al acabarlo (ADR 0051).
    await page.locator(".panel:visible .proyectos li").first().getByRole("button", { name: "Al acabar…" }).click();
    await page.waitForTimeout(250);
    await foto("9e-al-acabar");
    await boton("Entregársela al cliente").click();
    await page.locator('.panel:visible input[type="password"]').first().fill("la del cliente");
    await page.waitForTimeout(250);
    await foto("9f-entregar");

    // Y llevar una entrada a esa bóveda, desde la ficha de la entrada.
    await page.locator(".lateral").getByRole("button", { name: "Bóveda", exact: true }).click();
    await page.locator(".panel:visible .lista-boveda li").first().locator("button").click();
    await boton("Llevar a otra bóveda").click();
    await page.waitForTimeout(300);
    await foto("9d-llevar-a-otra-boveda");

    // **Y dentro del proyecto**, que es lo que el cliente leyó como su bóveda personal
    // (2026-10-02): hay que mirar si el título dice dónde estás, si la fila de la barra
    // lateral cabe en dos líneas y si el botón de salir se lee como una salida y no como
    // otra acción del formulario. Nada de eso lo dice una aserción.
    await aLaLista();
    await page.locator(".lateral").getByRole("button", { name: "Proyectos", exact: true }).click();
    await page.locator(".panel:visible .proyectos").getByRole("button", { name: new RegExp(proyecto) }).click();
    await page.locator("#boveda-buscar").waitFor({ timeout: 20_000 });
    await page.waitForTimeout(400);
    await foto("9g-dentro-del-proyecto");
    // **Y la lista con el proyecto abierto**, que es donde la fila lleva más cosas:
    // el nombre y tres botones, uno de ellos «Quién tiene acceso…». El cliente dice
    // que ahí se rompe el salto de línea (2026-10-06), y eso no lo dice ninguna
    // aserción: se mira.
    await page.locator(".lateral").getByRole("button", { name: "Proyectos", exact: true }).click();
    await page.locator(".panel:visible .proyectos li").first().waitFor({ timeout: 20_000 });
    await page.waitForTimeout(300);
    await page
      .locator(".panel:visible .proyectos")
      .screenshot({ path: `${donde}/9j-lista-con-abierto-${tema}.png`, scale: "css" });
    // La barra de cerca, con la escala del CSS: el antetítulo son once píxeles y a
    // tamaño de pantalla no se ve si se lee o si pelea con el nombre.
    await page.locator(".herramientas").screenshot({ path: `${donde}/9h-barra-proyecto-${tema}.png`, scale: "css" });
    // La barra lateral tiene **dos** `nav` —las secciones y Ajustes abajo—, así que
    // sin nombrar cuál Playwright falla por modo estricto.
    await page
      .locator('.lateral nav[aria-label="Secciones"]')
      .screenshot({ path: `${donde}/9i-lateral-proyecto-${tema}.png`, scale: "css" });
    // Y el segundo clic, que es el que dice lo que va a pasar.
    await page.locator(".herramientas .salir-proyecto").click();
    await page.waitForTimeout(200);
    await page.locator(".herramientas").screenshot({ path: `${donde}/9j-barra-salir-${tema}.png`, scale: "css" });

    // **Se sale antes de acabar.** Una captura que deje la ventana dentro de un proyecto
    // deja a las pruebas siguientes mirando otra bóveda, y eso ya tiró una publicación.
    await page.locator(".herramientas .salir-proyecto").click();
    await page.locator(".lateral").getByRole("button", { name: "Bóveda", exact: true }).click();
    // **Salir devuelve tu bóveda abierta**, y aquí había escrito lo contrario.
    //
    // Hasta el 2026-10-05 salir de un proyecto pedía la contraseña maestra, y esto
    // la tecleaba para dejar la bóveda abierta a las pruebas siguientes. Ese día se
    // quitó —lo preguntó el cliente: «Salir del proyecto» y «Cerrar la bóveda»
    // hacían lo mismo—, así que «Abrir la bóveda» **no vuelve a salir nunca** y el
    // `waitFor` agotaba sus 20 s. Es la familia de «al cambiar una decisión, releer
    // lo que la daba por supuesta»: la prueba seguía al día en todo menos en esto.
    //
    // Se espera al buscador, que es la señal de que la bóveda está abierta, y se
    // deja el camino de abrirla por si alguna pasada la encontrara cerrada.
    const abierta = page.locator("#boveda-buscar");
    await abierta.or(boton("Abrir la bóveda")).first().waitFor({ timeout: 20_000 });
    if (await boton("Abrir la bóveda").isVisible().catch(() => false)) {
      await page.locator("#boveda-llave").fill(maestra);
      await boton("Abrir la bóveda").click();
    }
    await abierta.waitFor({ timeout: 20_000 });
  });
});
