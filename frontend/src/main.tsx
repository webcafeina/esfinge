import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { resolverPlataforma } from "./plataforma";
import "./estilos.css";

// **Primero se resuelve dónde estamos y luego se pinta**, no al revés: de
// `data-sistema` y `data-escritorio` cuelga todo el CSS de cada escritorio, así
// que poniéndolos después del primer pintado se ve un instante la base y luego
// el sistema. Ver `plataforma.ts`, que además explica qué carrera cierra esto.
//
// Si la plataforma no se puede resolver, se pinta igual con la base neutra: una
// ventana sin atributos se ve, y una ventana que no se pinta, no.
void resolverPlataforma().finally(() => {
  createRoot(document.getElementById("raiz")!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  );
});
