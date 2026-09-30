/**
 * El banner de las llaves de acceso (ADR 0048).
 *
 * **Es la primera vez que Esfinge sustituye un diálogo del navegador**, y no un
 * adorno sobre la página: es la pantalla donde alguien decide identificarse. De ahí
 * las tres diferencias con `tarjeta.ts`, que por lo demás se copia entera —sombra
 * cerrada, solo clics `isTrusted`, rebote antes de hacer caso, `textContent` para
 * todo, `dibujar()` con `DOMParser` y los colores medidos en
 * `internal/tema/extension_test.go`—:
 *
 *  - **Va centrado y con velo**, no arriba a la derecha. Es una decisión de
 *    identidad, no un «¿guardo esto?», y además la esquina superior derecha es
 *    donde algunos Chrome ponen su propia burbuja de WebAuthn: ahí se confundirían.
 *  - **Atrapa el foco**, y `Esc` cede al navegador.
 *  - **El rebote de «Aceptar» es mayor.** La tarjeta arriesga una contraseña
 *    guardada de más; esto arriesga una identificación.
 *
 * **Y no lleva ningún campo de contraseña. Nunca.** Ni siquiera con la bóveda
 * cerrada: ahí se dice que se abra Esfinge, y la maestra se teclea en el panel de
 * la extensión, fuera de la página. Es lo único que protege de que la propia página
 * dibuje un banner igual que éste, y por eso va escrito en el pie donde se lee.
 *
 * No se imita al navegador: se contrasta con él. Paleta nuestra, la silueta siempre
 * a la vista, y una línea que dice quién pregunta.
 */

import siluetaSVG from "../../build/icono-barra.svg?raw";
import { dibujar } from "./dibujo";
import type { LlaveParaElBanner } from "./protocolo";

const PIEDRA = "#2b2b31";
const BLANCO = "#ffffff";
const ORO = "#f2c14e";
/** El texto secundario sobre la piedra. */
const SUAVE = "#d8d8de";

export type DecisionDelBanner =
  | { accion: "aceptar"; id: string }
  | { accion: "ahora-no" }
  | { accion: "otra-llave" };

export type EstadoDelBanner =
  | { tipo: "elegir"; rpId: string; llaves: LlaveParaElBanner[] }
  | { tipo: "cerrada"; rpId: string }
  /** Crear una llave nueva (P3). `cuenta` es el `user.name` que dice el sitio. */
  | { tipo: "crear"; rpId: string; cuenta: string };

export type Banner = {
  poner: (estado: EstadoDelBanner) => void;
  cerrar: () => void;
  /** La raíz de la sombra, **solo para las pruebas**: la página no llega aquí. */
  raiz: ShadowRoot;
};

/** Lo que se espera antes de hacer caso a un clic en «Aceptar». */
export const ESPERA_DE_ACEPTAR = 400;
/** Y en los demás, que arriesgan menos. */
export const ESPERA = 250;

const ESTILO = `
:host { all: initial; }
.velo {
  position: fixed;
  inset: 0;
  background: rgb(0 0 0 / 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
}
.banner {
  box-sizing: border-box;
  width: 340px;
  max-width: calc(100vw - 32px);
  padding: 18px 20px 16px;
  border-radius: 16px;
  background: ${PIEDRA};
  color: ${BLANCO};
  font: 14px/1.4 -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
  box-shadow: 0 18px 50px rgb(0 0 0 / 0.5);
  animation: entrar 160ms ease-out;
}
.cabecera { display: flex; align-items: center; gap: 10px; }
.marca { width: 30px; height: 30px; flex-shrink: 0; }
.marca svg { display: block; width: 100%; height: 100%; }
h1 { margin: 0; font-size: 14px; font-weight: 700; letter-spacing: 0.01em; }
.sitio { margin: 14px 0 0; font-size: 19px; font-weight: 700; overflow-wrap: anywhere; }
.etiqueta { display: block; margin: 12px 0 2px; font-size: 12px; color: ${SUAVE}; }
.dato { margin: 0; overflow-wrap: anywhere; }
.aviso { margin: 12px 0 0; color: ${SUAVE}; }
select {
  box-sizing: border-box;
  width: 100%;
  margin-top: 2px;
  padding: 7px 9px;
  border: 0;
  border-radius: 8px;
  background: ${BLANCO};
  color: ${PIEDRA};
  font: inherit;
}
.botones { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; margin-top: 18px; }
button {
  appearance: none;
  padding: 8px 14px;
  border: 1px solid rgb(255 255 255 / 0.35);
  border-radius: 8px;
  background: transparent;
  color: ${BLANCO};
  font: 600 13px/1.2 -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
  cursor: pointer;
}
button.principal { border-color: ${ORO}; background: ${ORO}; color: ${PIEDRA}; }
button.discreto {
  padding: 4px 0;
  border: 0;
  color: ${SUAVE};
  font-weight: 400;
  font-size: 12px;
  text-decoration: underline;
  text-underline-offset: 2px;
}
.pie { margin: 14px 0 0; padding-top: 10px; border-top: 1px solid rgb(255 255 255 / 0.15); font-size: 11px; color: ${SUAVE}; }
button:focus-visible, select:focus-visible { outline: 2px solid ${ORO}; outline-offset: 2px; }
@keyframes entrar { from { opacity: 0; transform: scale(0.98); } }
@media (prefers-reduced-motion: reduce) { .banner { animation: none; } }
`;

/**
 * Lo que dice quién pregunta, **y la regla que protege de un banner falso**.
 *
 * No se cambia sin pensarlo: es la única defensa real contra que la propia página
 * dibuje una copia de esto para que alguien teclee la maestra. Enseña la regla, no
 * solo la marca.
 */
const PIE = "Esto lo pregunta la extensión Esfinge, no el navegador. La contraseña maestra solo se teclea en el panel de Esfinge.";

export function mostrarBanner(
  estado: EstadoDelBanner,
  alDecidir: (d: DecisionDelBanner) => void,
  doc: Document = document,
): Banner {
  doc.querySelectorAll("esfinge-llave").forEach((viejo) => viejo.remove());

  const anfitrion = doc.createElement("esfinge-llave");
  const fijar = (p: string, v: string) => anfitrion.style.setProperty(p, v, "important");
  fijar("position", "fixed");
  fijar("inset", "0");
  fijar("z-index", "2147483647");
  fijar("display", "block");
  fijar("margin", "0");

  const raiz = anfitrion.attachShadow({ mode: "closed" });
  const estilo = doc.createElement("style");
  estilo.textContent = ESTILO;
  const velo = doc.createElement("div");
  velo.className = "velo";
  const banner = doc.createElement("div");
  banner.className = "banner";
  banner.setAttribute("role", "dialog");
  banner.setAttribute("aria-modal", "true");
  banner.setAttribute("aria-label", "Esfinge · Llave de acceso");
  velo.append(banner);
  raiz.append(estilo, velo);
  doc.documentElement.append(anfitrion);

  let cerrado = false;
  const cerrar = () => {
    if (cerrado) return;
    cerrado = true;
    doc.removeEventListener("keydown", teclas, true);
    anfitrion.remove();
  };

  const decidir = (d: DecisionDelBanner) => {
    if (cerrado) return;
    cerrar();
    alDecidir(d);
  };

  /**
   * Cuándo apareció lo que se está enseñando. Un clic que llega antes de que dé
   * tiempo a leerlo no cuenta: `isTrusted` demuestra que hubo una persona, no que
   * supiera dónde pulsaba, y la página de debajo puede poner algo justo donde va a
   * salir esto y quedarse con la decisión.
   */
  let desde = Date.now();

  const boton = (texto: string, clase: string, espera: number, alPulsar: () => void) => {
    const b = doc.createElement("button");
    b.type = "button";
    b.textContent = texto;
    if (clase) b.className = clase;
    b.addEventListener("click", (e) => {
      if (!e.isTrusted || Date.now() - desde < espera) return;
      alPulsar();
    });
    return b;
  };

  const parrafo = (clase: string, texto: string) => {
    const p = doc.createElement("p");
    p.className = clase;
    p.textContent = texto;
    return p;
  };

  /**
   * **El foco se queda dentro, y `Esc` cede.**
   *
   * En captura y sobre el documento: un sitio que se pelee por el foco con su
   * propio manejador no puede sacarlo de aquí sin que esto lo vea primero. Y ceder
   * con `Esc` es lo que hace cualquier diálogo, así que es lo que se espera.
   */
  const teclas = (e: KeyboardEvent) => {
    if (cerrado) return;
    if (e.key === "Escape") {
      e.preventDefault();
      decidir({ accion: "ahora-no" });
      return;
    }
    if (e.key !== "Tab") return;
    const dentro = Array.from(raiz.querySelectorAll<HTMLElement>("button, select"));
    if (dentro.length === 0) return;
    const activo = raiz.activeElement as HTMLElement | null;
    const i = activo ? dentro.indexOf(activo) : -1;
    const siguiente = e.shiftKey ? (i <= 0 ? dentro.length - 1 : i - 1) : (i === dentro.length - 1 ? 0 : i + 1);
    e.preventDefault();
    dentro[siguiente].focus();
  };
  doc.addEventListener("keydown", teclas, true);

  const poner = (nuevo: EstadoDelBanner) => {
    banner.replaceChildren();
    desde = Date.now();

    const cabecera = doc.createElement("div");
    cabecera.className = "cabecera";
    const marca = doc.createElement("span");
    marca.className = "marca";
    dibujar(marca, siluetaSVG);
    const h = doc.createElement("h1");
    h.textContent = "Esfinge · Llave de acceso";
    cabecera.append(marca, h);
    banner.append(cabecera);

    // **El sitio, en grande y con `textContent`.** Es lo que hay que leer antes de
    // aceptar, y es lo único de aquí que viene de fuera.
    banner.append(parrafo("sitio", nuevo.rpId));

    const botones = doc.createElement("div");
    botones.className = "botones";

    // **Crear es la pantalla más consecuente de Esfinge en la web de otro** (P3), y
    // el texto lo dice en vez de disimularlo: lo que se decide aquí es dónde va a
    // vivir la única forma de entrar en esa cuenta. Una contraseña olvidada se
    // recupera por correo; una llave de acceso perdida, no.
    //
    // Y por eso lleva **la frase del respaldo**: la llave se guarda cifrada y
    // sincronizada, que es la respuesta a la pregunta que cualquiera se hace al leer
    // lo de arriba. Sin ella, lo honesto daría miedo y lo que da miedo se cancela.
    if (nuevo.tipo === "crear") {
      if (nuevo.cuenta) {
        const eti = doc.createElement("span");
        eti.className = "etiqueta";
        eti.textContent = "Cuenta";
        banner.append(eti, parrafo("dato", nuevo.cuenta));
      }
      banner.append(
        parrafo(
          "aviso",
          "Esfinge creará tu llave de acceso y la guardará en tu bóveda, cifrada y sincronizada " +
            "con tus equipos. Será la forma de entrar en esta cuenta.",
        ),
      );
      botones.append(boton("Ahora no", "", ESPERA, () => decidir({ accion: "ahora-no" })));
      botones.append(
        boton("Crear la llave", "principal", ESPERA_DE_ACEPTAR, () => decidir({ accion: "aceptar", id: "" })),
      );
    } else if (nuevo.tipo === "cerrada") {
      banner.append(
        parrafo("aviso", "Abre Esfinge para usar tu llave de acceso. Cuando la tengas abierta, vuelve a intentarlo."),
      );
      botones.append(boton("Ahora no", "", ESPERA, () => decidir({ accion: "ahora-no" })));
      botones.append(boton("Usar otra llave", "", ESPERA, () => decidir({ accion: "otra-llave" })));
    } else if (nuevo.llaves.length === 1) {
      const eti = doc.createElement("span");
      eti.className = "etiqueta";
      eti.textContent = "Cuenta";
      banner.append(eti, parrafo("dato", nuevo.llaves[0].nombre));
      const id = nuevo.llaves[0].id;
      botones.append(boton("Usar otra llave", "discreto", ESPERA, () => decidir({ accion: "otra-llave" })));
      botones.append(boton("Ahora no", "", ESPERA, () => decidir({ accion: "ahora-no" })));
      botones.append(boton("Aceptar", "principal", ESPERA_DE_ACEPTAR, () => decidir({ accion: "aceptar", id })));
    } else {
      const eti = doc.createElement("label");
      eti.className = "etiqueta";
      eti.textContent = "Con qué cuenta";
      const sel = doc.createElement("select");
      for (const l of nuevo.llaves) {
        const o = doc.createElement("option");
        o.value = l.id;
        o.textContent = l.nombre;
        sel.append(o);
      }
      banner.append(eti, sel);
      botones.append(boton("Usar otra llave", "discreto", ESPERA, () => decidir({ accion: "otra-llave" })));
      botones.append(boton("Ahora no", "", ESPERA, () => decidir({ accion: "ahora-no" })));
      botones.append(boton("Aceptar", "principal", ESPERA_DE_ACEPTAR, () => decidir({ accion: "aceptar", id: sel.value })));
    }

    banner.append(botones, parrafo("pie", PIE));
  };

  poner(estado);
  return { poner, cerrar, raiz };
}
