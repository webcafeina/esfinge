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
import { campoDeIdentificador, queSeEnvia, type Envio } from "./campos";

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

/** Lo que se espera a que se deje de teclear antes de avisar. */
const ESPERA_AL_TECLEAR = 300;

/**
 * vigilarIdentificador avisa de **lo que una persona teclea** en el usuario de una
 * página de solo usuario, para que la página siguiente —la de la contraseña— sepa
 * quién entra (`identidad.ts`).
 *
 * **Solo lo tecleado** (`isTrusted`): lo que escribe Esfinge al rellenar no cuenta,
 * y así, si se deja la cuenta que puso Esfinge y se pulsa «Siguiente», la página de la
 * contraseña se rellena como siempre. Se avisa al dejar de teclear, al cambiar de
 * campo y al pulsar Intro, porque **cómo se envía esa página no importa**: no hace
 * falta adivinar dónde está su botón.
 */
export function vigilarIdentificador(alEscribir: (usuario: string) => void, doc: Document = document) {
  let plazo: ReturnType<typeof setTimeout> | undefined;
  const avisar = (campo: HTMLInputElement, yaMismo: boolean) => {
    clearTimeout(plazo);
    const enviar = () => {
      const usuario = campo.value.trim();
      if (usuario) alEscribir(usuario);
    };
    if (yaMismo) enviar();
    else plazo = setTimeout(enviar, ESPERA_AL_TECLEAR);
  };

  doc.addEventListener(
    "input",
    (e) => {
      if (!e.isTrusted) return;
      const campo = campoDeIdentificador(e.target, doc);
      if (campo) avisar(campo, false);
    },
    true,
  );
  doc.addEventListener(
    "change",
    (e) => {
      if (!e.isTrusted) return;
      const campo = campoDeIdentificador(e.target, doc);
      if (campo) avisar(campo, true);
    },
    true,
  );
  doc.addEventListener(
    "keydown",
    (e) => {
      if (!e.isTrusted || e.key !== "Enter") return;
      const campo = campoDeIdentificador(e.target, doc);
      if (campo) avisar(campo, true);
    },
    true,
  );
}
