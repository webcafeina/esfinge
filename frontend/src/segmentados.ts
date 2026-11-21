// **La selección que se desliza en los controles segmentados de macOS 27.**
//
// Al elegir otra opción, el fondo de la elegida **se desliza** hasta ella en vez de
// aparecer de golpe, que es lo que hace macOS y lo que el cliente echó en falta
// comparándolo con Apolo: «cuando se cambia entre elementos de un menú horizontal
// debe haber animaciones».
//
// El CSS no sabe dónde está la opción elegida, así que se mide aquí y la hoja de
// macOS dibuja el fondo con esas medidas. **Sólo lo usa macOS**: en los demás
// sistemas el fondo sigue viviendo en el botón, y las variables que esto escribe no
// las lee nadie.
//
// **Y se sigue en los cuatro, no sólo en macOS**, al revés que en Apolo: aquí el
// sistema se puede cambiar por la URL **sin recargar** (ADR 0064), así que mirarlo
// una vez al arrancar dejaría el deslizamiento muerto en cuanto alguien simulara
// macOS desde otro. Lo que cuesta es un vigilante parado, y lo que ahorra es que la
// simulación siga valiendo para lo que existe.
//
// Ojo con el atributo: el segmentado de Esfinge son **pestañas** —`role="tab"` con
// `aria-selected`—, no `aria-checked`, que es lo que usa Apolo. Copiarlo de allí ya
// costó una vez que la selección no se quedara marcada.

function medir(s: HTMLElement) {
  const b = s.querySelector<HTMLElement>('[aria-selected="true"]');
  // `offsetParent` nulo es que está escondido: sus medidas serían todas cero y el
  // fondo se iría a la esquina, para volver con un salto al enseñarlo.
  if (!b || !s.offsetParent) return;
  s.style.setProperty("--sel-x", `${b.offsetLeft}px`);
  s.style.setProperty("--sel-y", `${b.offsetTop}px`);
  s.style.setProperty("--sel-w", `${b.offsetWidth}px`);
  s.style.setProperty("--sel-h", `${b.offsetHeight}px`);
  s.style.setProperty("--sel-r", getComputedStyle(b).borderRadius);
  // La primera vez **no se desliza**: el fondo aparece ya en su sitio. Sin esto,
  // cada vez que se entra en una pantalla la selección entraría corriendo desde la
  // izquierda, que es un efecto que nadie ha pedido y se ve como un fallo.
  if (!s.dataset.medido) requestAnimationFrame(() => (s.dataset.medido = "si"));
}

export function seguirSegmentados(): void {
  const vistos = new WeakSet<HTMLElement>();
  // Un segmentado cambia de ancho sin que cambie la selección: la ventana se
  // estrecha y los rótulos se esconden, o se pliega la barra lateral. Entonces hay
  // que volver a medir, o el fondo se queda del tamaño de antes.
  const tamano = new ResizeObserver((e) => e.forEach((x) => medir(x.target as HTMLElement)));
  const todos = () =>
    document.querySelectorAll<HTMLElement>(".segmentado").forEach((s) => {
      if (!vistos.has(s)) {
        vistos.add(s);
        tamano.observe(s);
      }
      medir(s);
    });
  // Las secciones se esconden, no se desmontan, así que `hidden` cuenta tanto como
  // la selección: al volver a una pantalla hay que medir otra vez.
  new MutationObserver(todos).observe(document.body, {
    subtree: true,
    childList: true,
    attributes: true,
    attributeFilter: ["aria-selected", "hidden"],
  });
  todos();
}
