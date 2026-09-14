/**
 * Lo que Esfinge deja a la vista en un campo que acaba de rellenar.
 *
 * # Por qué existe, y qué decisión cambia
 *
 * Con el relleno automático, la contraseña aparece sola y **quien mira puede no
 * saber por qué**. El cliente pidió que se viera que ha sido Esfinge, y eso cambia
 * lo que decía la ADR 0028 —«en la página no se dibuja nada»—. La ADR 0031 lo
 * matiza con estas dos piezas, y con límites:
 *
 *   - **El filete** va en el propio campo de la web, sin añadir nada: un anillo
 *     de oro con un halo del mismo oro, más suave, alrededor. **Sin anillo de
 *     piedra por fuera**: lo llevó la 2.20.0 para que se viera en webs claras, y el
 *     cliente pidió quitarlo porque se leía como un borde negro. El precio, dicho:
 *     el oro como línea sobre una web clara da 1,37-1,68:1, por debajo de lo que
 *     pide un indicador; el halo lo ensancha, y quien dice que ha sido Esfinge es
 *     el aviso. Se quita con lo primero que escribe una persona.
 *   - **El aviso** «Rellenado por Esfinge» sí es un elemento nuestro en la página
 *     de otro. Para que eso sea poco: **no se puede pulsar**, va en un
 *     `shadowRoot` **cerrado** —la web no puede leerlo ni cambiar su estilo—, no
 *     lleva ningún texto que venga de la web ni de la bóveda, y **se va solo a los
 *     tres segundos**.
 *
 * Los colores son fijos y no tokens, porque aquí no hay tema nuestro: la web puede
 * ser clara u oscura. Están medidos en `internal/tema/extension_test.go`, que lee
 * este fichero.
 */
import siluetaSVG from "../../build/icono-barra.svg?raw";

const ORO = "#f2c14e";
const PIEDRA = "#2b2b31";
const BLANCO = "#ffffff";

/**
 * El filete: un anillo de oro y un halo del mismo oro, suave. No mueve nada de
 * sitio. **Sin piedra**: ver arriba.
 */
const FILETE = `0 0 0 2px ${ORO}, 0 0 0 5px rgb(242 193 78 / 0.3)`;

/** Cuánto dura el aviso a la vista. */
export const DURACION_DEL_AVISO = 3000;

type Marca = { sombra: string; prioridad: string; soltar: () => void };

/**
 * Los campos con filete, y lo que había antes en su `box-shadow` para devolverlo.
 * **Elementos, no valores**: aquí no hay ninguna contraseña.
 */
const marcados = new WeakMap<HTMLInputElement, Marca>();

/**
 * ponerFilete marca un campo hasta que una persona escriba en él.
 *
 * **Con `isTrusted`**, que es lo que distingue a una persona de Esfinge: los
 * eventos que dispara `escribir` al rellenar no son de confianza para el
 * navegador, así que no quitan el filete; una tecla de verdad, sí.
 */
export function ponerFilete(campo: HTMLInputElement) {
  if (marcados.has(campo)) return;
  const sombra = campo.style.getPropertyValue("box-shadow");
  const prioridad = campo.style.getPropertyPriority("box-shadow");
  campo.style.setProperty("box-shadow", FILETE, "important");

  const alEscribir = (e: Event) => {
    if (e.isTrusted) quitarFilete(campo);
  };
  campo.addEventListener("input", alEscribir, true);
  marcados.set(campo, {
    sombra,
    prioridad,
    soltar: () => campo.removeEventListener("input", alEscribir, true),
  });
}

/** quitarFilete devuelve el campo como estaba. */
export function quitarFilete(campo: HTMLInputElement) {
  const marca = marcados.get(campo);
  if (!marca) return;
  marca.soltar();
  if (marca.sombra) {
    campo.style.setProperty("box-shadow", marca.sombra, marca.prioridad);
  } else {
    campo.style.removeProperty("box-shadow");
  }
  marcados.delete(campo);
}

/**
 * El estilo del aviso, dentro de su sombra. `:host` con `all: initial` para que
 * nada de la web se herede hacia dentro.
 */
const ESTILO = `
:host { all: initial; }
.aviso {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px 5px 7px;
  border-radius: 999px;
  background: ${PIEDRA};
  color: ${BLANCO};
  font: 600 12px/1.2 -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
  box-shadow: 0 2px 10px rgb(0 0 0 / 0.28);
  white-space: nowrap;
  animation: entrar 160ms ease-out;
}
.marca { width: 16px; height: 16px; flex-shrink: 0; }
.marca svg { display: block; width: 100%; height: 100%; }
@keyframes entrar { from { opacity: 0; transform: translateY(-3px); } }
@media (prefers-reduced-motion: reduce) { .aviso { animation: none; } }
`;

/**
 * avisar pone «Rellenado por Esfinge» debajo de un campo, tres segundos.
 *
 * **Uno a la vez**: si llega otro, sustituye al anterior, que es lo que pasa cuando
 * se rellena la contraseña y enseguida el código.
 *
 * En coordenadas del documento y no de la ventana, para que acompañe al campo si
 * la página se desplaza mientras está a la vista.
 */
export function avisar(bajoDe: HTMLElement, texto: string): HTMLElement {
  document.querySelectorAll("esfinge-aviso").forEach((viejo) => viejo.remove());

  const anfitrion = document.createElement("esfinge-aviso");
  const caja = bajoDe.getBoundingClientRect();
  const fijar = (propiedad: string, valor: string) =>
    anfitrion.style.setProperty(propiedad, valor, "important");
  fijar("position", "absolute");
  fijar("left", `${Math.round(caja.left + window.scrollX)}px`);
  fijar("top", `${Math.round(caja.bottom + window.scrollY + 6)}px`);
  fijar("z-index", "2147483647");
  fijar("pointer-events", "none");
  fijar("display", "block");
  fijar("margin", "0");

  // **Cerrado**: la web no puede entrar a leerlo ni a cambiarle el estilo.
  const sombra = anfitrion.attachShadow({ mode: "closed" });
  const estilo = document.createElement("style");
  estilo.textContent = ESTILO;
  const aviso = document.createElement("div");
  aviso.className = "aviso";
  aviso.setAttribute("role", "status");
  const marca = document.createElement("span");
  marca.className = "marca";
  marca.setAttribute("aria-hidden", "true");
  // **La silueta rellena de la barra, no la marca a trazo.** La marca a trazo tiene
  // la cara hueca, y sobre el fondo de piedra del aviso el hueco se veía como una
  // cara negra (lo vio el cliente en la 2.20.0). La silueta lleva la cara en crema.
  // Un dibujo nuestro, no nada de la página.
  marca.innerHTML = siluetaSVG;
  const frase = document.createElement("span");
  frase.textContent = texto;
  aviso.append(marca, frase);
  sombra.append(estilo, aviso);

  document.documentElement.append(anfitrion);
  setTimeout(() => anfitrion.remove(), DURACION_DEL_AVISO);
  return anfitrion;
}
