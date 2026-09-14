/**
 * La tarjeta «¿Guardar en Esfinge?», en la página de otro.
 *
 * # Lo que esto es, dicho antes que nada
 *
 * **El primer elemento de Esfinge que se puede pulsar dentro de una web ajena**
 * (ADR 0032). La ADR 0028 decía que en la página no se dibujaba nada y la 0031 dejó
 * un filete y un aviso que no se pulsan; esto va más allá, y lo decidió el cliente
 * sabiendo el precio. Lo que lo acota:
 *
 *   - **Sombra cerrada.** La web no puede leer lo que hay dentro, ni cambiarle el
 *     estilo, ni alcanzar sus botones.
 *   - **Solo cuentan los clics de verdad** (`isTrusted`), por si algún día algo
 *     llegara a los botones sin ser una persona.
 *   - **Aquí no hay ninguna contraseña.** La tarjeta enseña el sitio, el usuario y
 *     las cuentas, y al pulsar manda la decisión; la contraseña la tiene el
 *     trabajador de fondo, que es quien guarda.
 *   - **No lleva nada que venga de la web** salvo el anfitrión y el usuario que se
 *     escribió, y los dos se escriben con `textContent`.
 *
 * Los colores son fijos, medidos en `internal/tema/extension_test.go`, que lee este
 * fichero.
 */
import siluetaSVG from "../../build/icono-barra.svg?raw";
import { dibujar } from "./dibujo";
import type { Cuenta, Oferta } from "./protocolo";

const PIEDRA = "#2b2b31";
const BLANCO = "#ffffff";
const ORO = "#f2c14e";
/** El texto secundario sobre la piedra. */
const SUAVE = "#d8d8de";

/** Lo que dura a la vista la confirmación de que se ha guardado. */
export const DURACION_DE_LA_CONFIRMACION = 2000;

export type Decision =
  | { accion: "guardar"; titulo: string }
  | { accion: "actualizar"; id: string }
  | { accion: "nunca" }
  | { accion: "ahora-no" };

export type Resultado = { ok: boolean; error?: string };

export type EstadoDeLaTarjeta =
  | { tipo: "oferta"; oferta: Oferta; usuario: string }
  | { tipo: "cerrada"; sitio: string; usuario: string };

export type Tarjeta = {
  /** Cambia lo que enseña: al abrir la bóveda, la oferta sustituye al aviso. */
  poner: (estado: EstadoDeLaTarjeta) => void;
  cerrar: () => void;
  /**
   * La raíz de la sombra, **solo para las pruebas**. Vive en el mundo aislado del
   * guion de la extensión, donde la página no llega.
   */
  raiz: ShadowRoot;
};

const ESTILO = `
:host { all: initial; }
.tarjeta {
  box-sizing: border-box;
  width: 320px;
  padding: 14px 16px 16px;
  border-radius: 14px;
  background: ${PIEDRA};
  color: ${BLANCO};
  font: 14px/1.35 -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
  box-shadow: 0 10px 34px rgb(0 0 0 / 0.35);
  animation: entrar 180ms ease-out;
}
.cabecera { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
.marca { width: 34px; height: 34px; flex-shrink: 0; }
.marca svg { display: block; width: 100%; height: 100%; }
h1 { margin: 0; font-size: 15px; font-weight: 700; }
.sitio { margin: 0; font-size: 12px; color: ${SUAVE}; overflow-wrap: anywhere; }
.dato { margin: 8px 0 0; color: ${BLANCO}; overflow-wrap: anywhere; }
.etiqueta { display: block; margin: 10px 0 2px; font-size: 12px; color: ${SUAVE}; }
.aviso { margin: 10px 0 0; color: ${SUAVE}; }
.sin-margen { margin-top: 0; }
label { display: block; margin: 10px 0 4px; font-size: 12px; color: ${SUAVE}; }
input, select {
  box-sizing: border-box;
  width: 100%;
  padding: 7px 9px;
  border: 0;
  border-radius: 8px;
  background: ${BLANCO};
  color: ${PIEDRA};
  font: inherit;
}
.botones { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; margin-top: 14px; }
button {
  appearance: none;
  padding: 7px 12px;
  border: 1px solid rgb(255 255 255 / 0.35);
  border-radius: 8px;
  background: transparent;
  color: ${BLANCO};
  font: 600 13px/1.2 -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
  cursor: pointer;
}
button.principal { border-color: ${ORO}; background: ${ORO}; color: ${PIEDRA}; }
/* «Nunca en este sitio» va en su propia línea, debajo y a la izquierda: en la fila
   de los otros dos no cabe a 320 píxeles, y es la decisión que menos se toma. Va
   subrayado, como el botón discreto de la ventana. */
.pie { margin-top: 8px; }
button.discreto {
  padding: 4px 0;
  border: 0;
  color: ${SUAVE};
  font-weight: 400;
  font-size: 12px;
  text-decoration: underline;
  text-underline-offset: 2px;
}
button:focus-visible, input:focus-visible, select:focus-visible { outline: 2px solid ${ORO}; outline-offset: 2px; }
.resultado { margin: 10px 0 0; font-weight: 600; }
@keyframes entrar { from { opacity: 0; transform: translateY(-6px); } }
@media (prefers-reduced-motion: reduce) { .tarjeta { animation: none; } }
`;

/**
 * mostrarTarjeta pone la tarjeta arriba a la derecha y la deja **hasta que se
 * decida algo o se cambie de página**, que es lo que eligió el cliente.
 */
export function mostrarTarjeta(
  estado: EstadoDeLaTarjeta,
  alDecidir: (d: Decision) => Promise<Resultado>,
  alReintentar: () => void,
  doc: Document = document,
): Tarjeta {
  doc.querySelectorAll("esfinge-tarjeta").forEach((vieja) => vieja.remove());

  const anfitrion = doc.createElement("esfinge-tarjeta");
  const fijar = (p: string, v: string) => anfitrion.style.setProperty(p, v, "important");
  fijar("position", "fixed");
  fijar("top", "16px");
  fijar("right", "16px");
  fijar("z-index", "2147483647");
  fijar("display", "block");
  fijar("margin", "0");

  const raiz = anfitrion.attachShadow({ mode: "closed" });
  const estilo = doc.createElement("style");
  estilo.textContent = ESTILO;
  const tarjeta = doc.createElement("div");
  tarjeta.className = "tarjeta";
  tarjeta.setAttribute("role", "dialog");
  tarjeta.setAttribute("aria-label", "Esfinge");
  raiz.append(estilo, tarjeta);
  doc.documentElement.append(anfitrion);

  const cerrar = () => anfitrion.remove();

  /** Un botón que solo hace caso a los clics de una persona. */
  const boton = (texto: string, clase: string, alPulsar: () => void) => {
    const b = doc.createElement("button");
    b.type = "button";
    b.textContent = texto;
    if (clase) b.className = clase;
    b.addEventListener("click", (e) => {
      if (!e.isTrusted) return;
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

  const usuario = (nombre: string) => {
    const d = doc.createElement("div");
    d.append(parrafo("etiqueta", "Usuario"), parrafo("dato", nombre));
    d.lastElementChild!.classList.add("sin-margen");
    return d;
  };

  const cabecera = (titulo: string, sitio: string) => {
    const c = doc.createElement("div");
    c.className = "cabecera";
    const marca = doc.createElement("span");
    marca.className = "marca";
    marca.setAttribute("aria-hidden", "true");
    dibujar(marca, siluetaSVG); // un dibujo nuestro
    const textos = doc.createElement("div");
    const h = doc.createElement("h1");
    h.textContent = titulo;
    textos.append(h);
    if (sitio) textos.append(parrafo("sitio", sitio));
    c.append(marca, textos);
    return c;
  };

  const pie = (b: HTMLButtonElement) => {
    const d = doc.createElement("div");
    d.className = "pie";
    d.append(b);
    return d;
  };

  /** decidir manda la decisión y enseña cómo ha ido. */
  const decidir = async (d: Decision, botones: HTMLButtonElement[], hecho: string) => {
    botones.forEach((b) => (b.disabled = true));
    const r = await alDecidir(d);
    if (d.accion === "ahora-no") {
      cerrar();
      return;
    }
    if (r.ok) {
      tarjeta.replaceChildren(cabecera(hecho, ""));
      setTimeout(cerrar, DURACION_DE_LA_CONFIRMACION);
      return;
    }
    botones.forEach((b) => (b.disabled = false));
    tarjeta.querySelector(".resultado")?.remove();
    const error = parrafo("resultado", r.error ?? "No se ha podido guardar.");
    const debajo = tarjeta.querySelector(".botones");
    if (debajo) debajo.before(error);
    else tarjeta.append(error);
  };

  const poner = (e: EstadoDeLaTarjeta) => {
    tarjeta.replaceChildren();

    if (e.tipo === "cerrada") {
      tarjeta.append(cabecera("¿Guardar en Esfinge?", e.sitio));
      if (e.usuario) tarjeta.append(usuario(e.usuario));
      tarjeta.append(parrafo("aviso", "Abre la bóveda en Esfinge para guardar esta cuenta."));
      const fila = doc.createElement("div");
      fila.className = "botones";
      fila.append(
        boton("Ahora no", "", () => decidir({ accion: "ahora-no" }, [], "")),
        boton("Ya la he abierto", "principal", alReintentar),
      );
      tarjeta.append(fila);
      return;
    }

    const { oferta } = e;
    const fila = doc.createElement("div");
    fila.className = "botones";

    if (oferta.accion === "guardar") {
      tarjeta.append(cabecera("¿Guardar en Esfinge?", oferta.sitio));
      const etiqueta = doc.createElement("label");
      etiqueta.textContent = "Título";
      const campo = doc.createElement("input");
      campo.value = oferta.titulo ?? "";
      campo.setAttribute("aria-label", "Título");
      etiqueta.append(campo);
      tarjeta.append(etiqueta);
      if (e.usuario) tarjeta.append(usuario(e.usuario));

      const nunca = boton("Nunca en este sitio", "discreto", () =>
        decidir({ accion: "nunca" }, botones, "No volverá a preguntar aquí"),
      );
      const ahoraNo = boton("Ahora no", "", () => decidir({ accion: "ahora-no" }, [], ""));
      const guardar = boton("Guardar", "principal", () =>
        decidir({ accion: "guardar", titulo: campo.value.trim() }, botones, "✓ Guardado en Esfinge"),
      );
      const botones = [nunca, ahoraNo, guardar];
      campo.addEventListener("keydown", (ev) => {
        // Intro en el título guarda, como en cualquier formulario. Solo si lo ha
        // pulsado una persona.
        if (ev.isTrusted && ev.key === "Enter") {
          ev.preventDefault();
          decidir({ accion: "guardar", titulo: campo.value.trim() }, botones, "✓ Guardado en Esfinge");
        }
      });
      fila.append(ahoraNo, guardar);
      tarjeta.append(fila, pie(nunca));
      return;
    }

    // Actualizar.
    const cuentas: Cuenta[] = oferta.cuentas ?? [];
    tarjeta.append(cabecera("¿Actualizar la contraseña?", oferta.sitio));
    let elegir: () => string;
    if (cuentas.length === 1) {
      const c = cuentas[0];
      tarjeta.append(parrafo("dato", [c.titulo, c.usuario].filter(Boolean).join(" · ")));
      elegir = () => c.id;
    } else {
      const etiqueta = doc.createElement("label");
      etiqueta.textContent = "Cuenta";
      const lista = doc.createElement("select");
      lista.setAttribute("aria-label", "Cuenta");
      for (const c of cuentas) {
        const opcion = doc.createElement("option");
        opcion.value = c.id;
        opcion.textContent = [c.titulo || "Sin título", c.usuario].filter(Boolean).join(" · ");
        lista.append(opcion);
      }
      etiqueta.append(lista);
      tarjeta.append(etiqueta);
      elegir = () => lista.value;
    }
    const nunca = boton("Nunca en este sitio", "discreto", () =>
      decidir({ accion: "nunca" }, botones, "No volverá a preguntar aquí"),
    );
    const ahoraNo = boton("Ahora no", "", () => decidir({ accion: "ahora-no" }, [], ""));
    const actualizar = boton("Actualizar", "principal", () =>
      decidir({ accion: "actualizar", id: elegir() }, botones, "✓ Contraseña actualizada"),
    );
    const botones = [nunca, ahoraNo, actualizar];
    fila.append(ahoraNo, actualizar);
    tarjeta.append(fila, pie(nunca));
  };

  poner(estado);
  return { poner, cerrar, raiz };
}
