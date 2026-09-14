/**
 * Darse cuenta de que se ha enviado un formulario, y leerlo en ese momento.
 *
 * Tres caminos, porque los sitios envían de tres formas y ninguna sola basta:
 *
 *   - **El evento `submit`** de un `<form>` de verdad.
 *   - **Un clic en un botón** dentro del formulario o del contenedor que tiene la
 *     contraseña: media web moderna no usa `<form>` y envía con JavaScript.
 *   - **Intro en el campo de la contraseña**.
 *
 * **Solo cuenta lo que hace una persona** (`isTrusted`): un envío fabricado por la
 * propia página no se ofrece guardar. Y si llegan varios por el mismo envío —el
 * clic y el `submit` a la vez—, da igual: el último pisa al anterior con lo mismo.
 */
import { queSeEnvia, type Envio } from "./campos";

/** Hasta dónde se sube buscando el contenedor de la contraseña. */
const NIVELES = 8;

function contenedorConContrasena(desde: Element | null): ParentNode | null {
  let el: Element | null = desde;
  for (let i = 0; el && i < NIVELES; i++, el = el.parentElement) {
    if (el.querySelector('input[type="password"]')) return el;
  }
  return null;
}

export function vigilarEnvios(alEnviar: (e: Envio) => void, doc: Document = document) {
  const leer = (ambito: ParentNode | null) => {
    if (!ambito) return;
    const envio = queSeEnvia(ambito, doc);
    if (envio) alEnviar(envio);
  };

  doc.addEventListener(
    "submit",
    (e) => {
      if (e.isTrusted) leer(e.target as HTMLFormElement);
    },
    true,
  );

  doc.addEventListener(
    "click",
    (e) => {
      if (!e.isTrusted) return;
      const boton = (e.target as Element | null)?.closest?.(
        'button, input[type="submit"], [role="button"]',
      );
      if (!boton) return;
      leer((boton as HTMLButtonElement).form ?? contenedorConContrasena(boton));
    },
    true,
  );

  doc.addEventListener(
    "keydown",
    (e) => {
      if (!e.isTrusted || e.key !== "Enter") return;
      const campo = e.target as HTMLInputElement | null;
      if (campo?.tagName !== "INPUT" || campo.type !== "password") return;
      leer(campo.form ?? contenedorConContrasena(campo));
    },
    true,
  );
}
