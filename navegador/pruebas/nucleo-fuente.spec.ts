import { expect, test } from "@playwright/test";
import { Boveda } from "../src/nucleo/boveda";
import { atender } from "../src/nucleo/fuente";
import type { Entrada } from "../src/nucleo/entrada";
import type { Peticion } from "../src/protocolo";

/**
 * Los verbos contestados con la bóveda del navegador (ADR 0040), con las reglas de
 * `fuenteDelNavegador` en Go. Lo que importa de verdad está en tres: que **nunca**
 * salga una contraseña hacia otro sitio, qué se ofrece guardar o actualizar, y el
 * freno de rellenos.
 */

const MAESTRA = "una maestra larga para las pruebas de los verbos";
const cred = (titulo: string, usuario: string, secreto: string, sitio: string, extra: Partial<Entrada> = {}): Entrada => ({
  id: "",
  tipo: "credencial",
  titulo,
  usuario,
  secreto,
  sitios: [sitio],
  creada: "",
  cambiada: "",
  ...extra,
});
const p = (x: Partial<Peticion>): Peticion => ({ version: 1, que: "cuentas", ...x }) as Peticion;

async function bovedaDePrueba() {
  const { boveda } = await Boveda.crear(MAESTRA);
  const banco = await boveda.poner(cred("Banco", "yo", "clave-banco", "https://www.banco.es/entrar", { totp: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" }));
  await boveda.poner(cred("Correo", "yo@correo.es", "clave-correo", "correo.es"));
  await boveda.poner({ id: "", tipo: "nota", titulo: "Nota del banco", notas: "x", sitios: ["banco.es"], creada: "", cambiada: "" });
  return { b: boveda, banco };
}

test("verbos: cuentas del sitio, solo credenciales, y con el código dicho", async () => {
  const { b } = await bovedaDePrueba();
  const r = await atender(p({ que: "cuentas", origen: "https://banco.es/x" }), { existe: true, boveda: b });
  expect(r.cuentas?.map((c) => [c.titulo, c.tieneCodigo])).toEqual([["Banco", true]]);
});

test("verbos: una contraseña no sale nunca hacia otro sitio", async () => {
  const { b, banco } = await bovedaDePrueba();
  for (const origen of ["https://banco.es.malo.com/", "https://correo.es/", "http://banco.es/"]) {
    const r = await atender(p({ que: "rellenar", id: banco.id, origen }), { existe: true, boveda: b });
    expect(r.ok, origen).toBe(false);
    expect(r.relleno).toBeUndefined();
  }
  const bien = await atender(p({ que: "rellenar", id: banco.id, origen: "https://banco.es/" }), { existe: true, boveda: b });
  expect(bien.relleno).toEqual({ usuario: "yo", secreto: "clave-banco" });
});

test("verbos: cerrada lo dice, y el origen se mira antes que la bóveda", async () => {
  expect((await atender(p({ que: "cuentas", origen: "https://banco.es" }), { existe: true, boveda: null })).motivo).toBe("cerrada");
  expect((await atender(p({ que: "cuentas", origen: "https://banco.es" }), { existe: false, boveda: null })).motivo).toBe("sin-boveda");
  expect((await atender(p({ que: "cuentas", origen: "http://banco.es" }), { existe: true, boveda: null })).motivo).toBe("origen");
});

test("verbos: qué se ofrece tras un envío", async () => {
  const { b, banco } = await bovedaDePrueba();
  const ofrece = async (usuario: string, secreto: string) =>
    (await atender(p({ que: "ofrecer", origen: "https://banco.es/entrar", usuario, secreto }), { existe: true, boveda: b })).oferta;
  expect((await ofrece("yo", "clave-banco"))?.accion).toBe("nada"); // ya está
  expect(await ofrece("YO", "otra")).toMatchObject({ accion: "actualizar", cuentas: [{ id: banco.id }] });
  expect(await ofrece("otra persona", "otra")).toMatchObject({ accion: "guardar", titulo: "Banco", sitio: "banco.es" });
  expect(await ofrece("", "otra")).toMatchObject({ accion: "actualizar", cuentas: [{ id: banco.id }] });
  await b.excluir("banco.es");
  expect((await ofrece("otra persona", "otra"))?.accion).toBe("nada"); // «Nunca en este sitio»
});

test("verbos: guardar lo pone para el sitio del envío, y actualizar deja la anterior en el historial", async () => {
  const { b, banco } = await bovedaDePrueba();
  const g = await atender(p({ que: "guardar-cuenta", origen: "https://login.nuevo.es/entrar", usuario: "yo", secreto: "s", titulo: "" }), {
    existe: true,
    boveda: b,
  });
  expect(g.guardada?.titulo).toBe("Nuevo");
  const nueva = b.buscar("Nuevo")[0];
  expect(nueva.sitios).toEqual(["https://login.nuevo.es"]);

  await atender(p({ que: "actualizar-cuenta", origen: "https://banco.es/", id: banco.id, secreto: "clave-nueva" }), { existe: true, boveda: b });
  const v = b.ver(banco.id)!;
  expect(v.secreto).toBe("clave-nueva");
  expect(v.historial?.[0].secreto).toBe("clave-banco");
});

test("verbos: el freno de rellenos, doce por minuto", async () => {
  const { b, banco } = await bovedaDePrueba();
  const t = Date.now() + 10 * 60_000; // una ventana nueva, sin lo de las otras pruebas
  let bien = 0;
  for (let i = 0; i < 15; i++) {
    const r = await atender(p({ que: "rellenar", id: banco.id, origen: "https://banco.es/" }), { existe: true, boveda: b }, t);
    if (r.ok) bien++;
    else expect(r.motivo).toBe("demasiado");
  }
  expect(bien).toBe(12);
});

// --------------------------------------------------------- las llaves de acceso

/**
 * Los dos verbos de las llaves de acceso (ADR 0048).
 *
 * Lo que de verdad importa aquí es **uno solo**: que un sitio no pueda firmar por
 * otro. Lo demás son comodidades; eso es la única vía por la que esta fase puede
 * entregarle algo a un atacante, y por eso hay tres pruebas sobre ello y una sobre
 * lo que se enseña.
 */
async function conLlaves() {
  const { boveda } = await Boveda.crear(MAESTRA);
  const { crearLlave } = await import("../src/nucleo/llaves");
  const { aBase64Url } = await import("../src/nucleo/afirmacion");
  const par = await crearLlave();
  const github = await boveda.poner({
    id: "",
    tipo: "llave",
    titulo: "GitHub",
    rpId: "github.com",
    idCredencial: "Y3JlZC1kZS1naXRodWI",
    idUsuario: "dXN1YXJpbw",
    nombreVisible: "yo@ejemplo.com",
    algoritmo: -7,
    clavePrivada: aBase64Url(par.privada),
    creada: "",
    cambiada: "",
  } as Entrada);
  await boveda.poner({
    id: "",
    tipo: "llave",
    titulo: "Otro sitio",
    rpId: "ejemplo.com",
    idCredencial: "Y3JlZC1kZS1lamVtcGxv",
    nombreVisible: "yo@ejemplo.com",
    algoritmo: -7,
    clavePrivada: aBase64Url(par.privada),
    creada: "",
    cambiada: "",
  } as Entrada);
  return { b: boveda, github, publica: par.publica };
}

test("llaves: solo las de ese sitio, y solo lo que el banner enseña", async () => {
  const { b } = await conLlaves();
  const r = await atender(p({ que: "llaves", origen: "https://github.com/login", rpId: "github.com" }), {
    existe: true,
    boveda: b,
  });
  expect(r.llaves).toEqual([{ id: expect.any(String), nombre: "yo@ejemplo.com" }]);
  // **Ni el identificador de credencial ni el de usuario**: no hacen falta para
  // elegir, y lo que no hace falta no sale.
  expect(JSON.stringify(r.llaves)).not.toContain("Y3JlZC1kZS1naXRodWI");
});

test("llaves: con allowCredentials, solo la que el sitio dice querer", async () => {
  const { b } = await conLlaves();
  const sinLaSuya = await atender(
    p({ que: "llaves", origen: "https://github.com/", rpId: "github.com", permitidas: ["otra-cualquiera"] }),
    { existe: true, boveda: b },
  );
  expect(sinLaSuya.llaves).toEqual([]);
  const conLaSuya = await atender(
    p({ que: "llaves", origen: "https://github.com/", rpId: "github.com", permitidas: ["Y3JlZC1kZS1naXRodWI"] }),
    { existe: true, boveda: b },
  );
  expect(conLaSuya.llaves).toHaveLength(1);
});

/** **Lo único que puede entregar algo a un atacante**, y por eso va con su tabla. */
test("llaves: un sitio no puede pedir la llave de otro", async () => {
  const { b, github } = await conLlaves();
  const casos: [string, string][] = [
    ["https://malo.com/", "github.com"],
    ["https://github.com.malo.com/", "github.com"],
    ["https://malogithub.com/", "github.com"],
    ["https://foo.github.io/", "github.io"],
    ["http://github.com/", "github.com"],
  ];
  for (const [origen, rpId] of casos) {
    const listar = await atender(p({ que: "llaves", origen, rpId }), { existe: true, boveda: b });
    expect(listar.llaves ?? [], `listar desde ${origen} pidiendo ${rpId}`).toEqual([]);
    const firmar = await atender(p({ que: "firmar-llave", origen, rpId, id: github.id, reto: "AAAA" }), {
      existe: true,
      boveda: b,
    });
    expect(firmar.ok, `firmar desde ${origen} pidiendo ${rpId}`).toBe(false);
  }
});

test("llaves: y tampoco firmar con una llave que no es de ese sitio", async () => {
  const { b, github } = await conLlaves();
  // El origen y el rpId cuadran, pero la llave que se pide es de otro sitio.
  const r = await atender(
    p({ que: "firmar-llave", origen: "https://ejemplo.com/", rpId: "ejemplo.com", id: github.id, reto: "AAAA" }),
    { existe: true, boveda: b },
  );
  expect(r.ok).toBe(false);
});

/**
 * Lo que sale de firmar: la forma, el origen y que no lleve nada de dentro.
 *
 * **Aquí no se verifica la firma, y hay que decir por qué**: sale en DER, que es lo
 * que WebAuthn exige, y **WebCrypto solo verifica en P1363**. Comprobarla desde
 * aquí obligaría a escribir un descodificador de DER que producción no usa. La
 * verificación de verdad la hace Go, en `TestCruzadaAfirmarLlave`.
 */
test("llaves: lo que sale de firmar tiene la forma buena y no lleva la privada", async () => {
  const { b, github } = await conLlaves();
  const r = await atender(
    p({ que: "firmar-llave", origen: "https://github.com/login", rpId: "github.com", id: github.id, reto: "cmV0bw" }),
    { existe: true, boveda: b },
  );
  expect(r.ok).toBe(true);
  const a = r.afirmacion!;
  expect(a.idCredencial).toBe("Y3JlZC1kZS1naXRodWI");

  const { deBase64Url } = await import("../src/nucleo/afirmacion");
  const cliente = deBase64Url(a.datosDelCliente);
  // El origen que se firma lo pone este lado, del que dio el navegador.
  expect(new TextDecoder().decode(cliente)).toContain(String.raw`"origin":"https://github.com"`);
  // El autenticador son 37 bytes: el hash del sitio, las banderas y el contador.
  expect(deBase64Url(a.datosDelAutenticador).length).toBe(37);
  // Y la firma, DER: empieza por 0x30 y no son los 64 crudos de WebCrypto.
  const firma = deBase64Url(a.firma);
  expect(firma[0]).toBe(0x30);
  expect(firma.length).toBeGreaterThan(64);
  // Y la privada no aparece por ningún lado de lo que sale.
  expect(JSON.stringify(a)).not.toContain("clavePrivada");
});

/**
 * **Sin `rpId`, lo que este origen podría usar.** Es la pregunta que hay que hacer
 * *antes* de que el sitio diga nada, para saber si Esfinge se ofrece. Y no es la
 * misma que preguntar por el anfitrión: una llave guardada como `ejemplo.com`
 * sirve en `login.ejemplo.com`, y preguntando por el anfitrión no saldría.
 */
test("llaves: sin rpId salen las que este origen puede usar, incluida la del dominio de arriba", async () => {
  const { boveda } = await Boveda.crear(MAESTRA);
  const { crearLlave } = await import("../src/nucleo/llaves");
  const { aBase64Url } = await import("../src/nucleo/afirmacion");
  const par = await crearLlave();
  const llave = (rpId: string, nombre: string) =>
    ({
      id: "", tipo: "llave", titulo: nombre, rpId, idCredencial: "c-" + rpId,
      nombreVisible: nombre, algoritmo: -7, clavePrivada: aBase64Url(par.privada),
      creada: "", cambiada: "",
    }) as Entrada;
  await boveda.poner(llave("ejemplo.com", "la del dominio"));
  await boveda.poner(llave("login.ejemplo.com", "la del subdominio"));
  await boveda.poner(llave("otro.com", "la de otro sitio"));

  const desdeElSubdominio = await atender(p({ que: "llaves", origen: "https://login.ejemplo.com/" }), {
    existe: true,
    boveda,
  });
  expect(desdeElSubdominio.llaves?.map((l) => l.nombre).sort()).toEqual(["la del dominio", "la del subdominio"]);

  // Y desde el dominio de arriba **no** sale la del subdominio: un `rpId` tiene que
  // ser el anfitrión o un sufijo suyo, no al revés.
  const desdeElDominio = await atender(p({ que: "llaves", origen: "https://ejemplo.com/" }), { existe: true, boveda });
  expect(desdeElDominio.llaves?.map((l) => l.nombre)).toEqual(["la del dominio"]);

  // **Y la lista de dominios, que es otra cosa y por eso se mira aparte** (ADR 0048):
  // no son las llaves de este sitio sino **todas**, porque lo que el navegador tiene
  // que poder decir con la bóveda cerrada es «aquí hay algo» en cualquier sitio, no
  // solo en el que estaba abierto cuando se apuntó. Un filtro por origen aquí la
  // dejaría sirviendo únicamente para el sitio desde el que se preguntó.
  expect(desdeElDominio.dominios).toEqual(["ejemplo.com", "login.ejemplo.com", "otro.com"]);

  // Y al preguntar por un sitio concreto no viaja: ahí ya se sabe de qué se habla, y
  // repetirla sería mandar la lista entera de la bóveda en cada firma.
  const alFirmar = await atender(p({ que: "llaves", origen: "https://ejemplo.com/", rpId: "ejemplo.com" }), {
    existe: true,
    boveda,
  });
  expect(alFirmar.dominios).toBeUndefined();
});

/**
 * **Una llave sin clave privada dentro no cuenta**, ni para ofrecerse ni para la
 * lista. No es un caso inventado: es lo que `sinSecretos` deja al cruzar una entrada
 * hacia el panel, y una bóveda traída de otra versión puede tener la clase sin el
 * campo. Ofrecerse con ella sería sacar un banner que no puede firmar nada.
 */
test("llaves: una llave sin clave privada no sale ni en la lista de dominios", async () => {
  const { boveda } = await Boveda.crear(MAESTRA);
  await boveda.poner({
    id: "", tipo: "llave", titulo: "media llave", rpId: "vacia.com", idCredencial: "c-vacia",
    nombreVisible: "media llave", algoritmo: -7, clavePrivada: "", creada: "", cambiada: "",
  } as Entrada);

  const r = await atender(p({ que: "llaves", origen: "https://vacia.com/" }), { existe: true, boveda });
  expect(r.llaves).toEqual([]);
  expect(r.dominios).toEqual([]);
});

/**
 * **Una llave queda confirmada cuando el sitio la nombra, y firmar no la confirma**
 * (ADR 0048), espejo de `TestUnaLlaveSeConfirmaCuandoElSitioLaNombra` de Go.
 *
 * Es una prueba de **comportamiento y no de formato**, y eso es a propósito: quitar
 * `confirmada` de `CAMPOS` **no rompe ninguna cruzada** —vuelve por `extra` y los bytes
 * salen idénticos, que es la trampa ya escrita— y tampoco `TestCruzadaLoQueSeVacia`,
 * porque no es un secreto. Lo único que se rompe es esto: el núcleo no sabría leerlo ni
 * escribirlo, así que la llave **no quedaría confirmada nunca** y se reescribiría en
 * cada visita. Comprobado quitándolo.
 */
test("llaves: el sitio la nombra y queda confirmada; firmar no la confirma", async () => {
  const { boveda } = await Boveda.crear(MAESTRA);
  const { crearLlave } = await import("../src/nucleo/llaves");
  const { aBase64Url } = await import("../src/nucleo/afirmacion");
  const par = await crearLlave();
  await boveda.poner({
    id: "", tipo: "llave", titulo: "GitHub", rpId: "github.com",
    idCredencial: "Y3JlZC0x", nombreVisible: "yo@ejemplo.com",
    algoritmo: -7, clavePrivada: aBase64Url(par.privada), creada: "", cambiada: "",
  } as Entrada);
  const laLlave = () => {
    const x = boveda.buscar("").find((e) => e.tipo === "llave");
    return boveda.ver(x!.id)!;
  };
  const estado = { existe: true, boveda };

  expect(laLlave().confirmada ?? "", "recién creada no puede estar confirmada").toBe("");

  // El aviso de la página no confirma: ahí no hay lista ni `rpId`.
  await atender(p({ que: "llaves", origen: "https://github.com/login" }), estado);
  expect(laLlave().confirmada ?? "", "preguntar qué hay ha confirmado la llave").toBe("");

  // **Y firmar tampoco**, que es la mitad que se puede equivocar: sin lista, el sitio
  // no dice qué tiene y Esfinge ofrece la suya, así que una huérfana se firmaría igual.
  await atender(p({ que: "llaves", origen: "https://github.com/login", rpId: "github.com" }), estado);
  const firmada = await atender(
    p({ que: "firmar-llave", origen: "https://github.com/login", rpId: "github.com", id: laLlave().id, reto: "cmV0bw" }),
    estado,
  );
  expect(firmada.ok, firmada.error).toBe(true);
  expect(laLlave().confirmada ?? "", "firmar ha confirmado la llave").toBe("");

  // **Pero firmar sí apunta que se ha usado**, que es la otra señal y la única que llega
  // en el flujo donde el sitio no nombra ninguna llave. Separadas a propósito: mezclarlas
  // haría pasar la débil por la fuerte.
  const usada = laLlave().usada ?? "";
  expect(usada, "se ha firmado y no se ha apuntado cuándo").not.toBe("");
  // **Y en segundos, como las escribe Go**: con `toISOString()` a secas entran los
  // milisegundos y lo que escribe este lado deja de ser lo que escribiría el otro.
  expect(usada, "la fecha de uso lleva milisegundos").toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/);

  // **Y se actualiza**, que es lo que la separa de `confirmada`: lo útil de una fecha de
  // uso es la última. Se envejece a mano, que es más rápido y menos frágil que mover el
  // reloj.
  await boveda.poner({ ...laLlave(), usada: "2020-01-01T00:00:00Z" });
  const otra = await atender(
    p({ que: "firmar-llave", origen: "https://github.com/login", rpId: "github.com", id: laLlave().id, reto: "b3Ry" }),
    estado,
  );
  expect(otra.ok, otra.error).toBe(true);
  expect(laLlave().usada, "la fecha de uso no se ha actualizado").not.toBe("2020-01-01T00:00:00Z");

  // Y con la misma fecha no se escribe: dos firmas en el mismo segundo no pueden costar
  // dos escrituras y dos subidas.
  const antesDeFirmar = laLlave();
  const tercera = await atender(
    p({ que: "firmar-llave", origen: "https://github.com/login", rpId: "github.com", id: laLlave().id, reto: "eW90" }),
    estado,
  );
  expect(tercera.ok, tercera.error).toBe(true);
  if (laLlave().usada === antesDeFirmar.usada) {
    expect(laLlave().revision, "se ha escrito en la bóveda sin que la fecha de uso cambiara").toBe(
      antesDeFirmar.revision,
    );
  }

  // Lo que sí la confirma.
  await atender(
    p({ que: "llaves", origen: "https://github.com/login", rpId: "github.com", permitidas: ["Y3JlZC0x"] }),
    estado,
  );
  const primera = laLlave().confirmada ?? "";
  expect(primera, "el sitio la ha nombrado y no se ha confirmado").not.toBe("");

  // Y no se vuelve a escribir: la marca es de la primera vez.
  const antes = laLlave().revision;
  await atender(
    p({ que: "llaves", origen: "https://github.com/login", rpId: "github.com", permitidas: ["Y3JlZC0x"] }),
    estado,
  );
  expect(laLlave().confirmada).toBe(primera);
  expect(laLlave().revision, "se ha vuelto a escribir con la llave ya confirmada").toBe(antes);
});
