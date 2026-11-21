import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { resolverPlataforma } from "./plataforma";
import { seguirSegmentados } from "./segmentados";
import { resolverTema } from "./tema";
import "./estilos.css";

// **Primero se resuelve dónde estamos y luego se pinta**, no al revés: de
// `data-sistema` y `data-escritorio` cuelga todo el CSS de cada escritorio, así
// que poniéndolos después del primer pintado se ve un instante la base y luego
// el sistema. Ver `plataforma.ts`, que además explica qué carrera cierra esto.
//
// Si la plataforma no se puede resolver, se pinta igual con la base neutra: una
// ventana sin atributos se ve, y una ventana que no se pinta, no.
// El tema se pregunta **a la vez** que la plataforma, no después: son dos preguntas
// independientes y encadenarlas retrasaría el primer pintado por nada.
void Promise.all([resolverPlataforma(), resolverTema()]).finally(() => {
  createRoot(document.getElementById("raiz")!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  );
  // Y después de pintar, la selección deslizante de los segmentados de macOS 27.
  // Va aquí y no dentro de un componente porque vigila el documento entero: los
  // segmentados viven en seis pantallas y las secciones se esconden en vez de
  // desmontarse, así que un efecto por componente no vería volver a ninguna.
  seguirSegmentados();
});
