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
import { campoDeIdentificador, queSeEnvia, usuarioDeLaPagina, type Envio } from "./campos";

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
 * vigilarIdentificador avisa del usuario que hay en una página de solo usuario —la
 * primera de Google o de Microsoft—, para que la página siguiente, la de la
 * contraseña, sepa quién entra (`identidad.ts`).
 *
 * **Cuenta lo ponga quien lo ponga**: tecleado, puesto por el navegador, recordado por
 * el sitio o escrito por Esfinge. Es el usuario que se va a enviar, y eso es lo único
 * que importa. (En la primera versión solo contaba lo tecleado, y el cliente preguntó
 * lo evidente: ¿y si el correo se pone solo?)
 *
 * Se lee **cuando el campo cambia**, y otra vez **al pulsar Intro o hacer clic** en
 * cualquier sitio de la página —el botón «Siguiente», esté donde esté—, porque un
 * sitio puede poner el valor sin avisar a nadie. El clic y la tecla sí tienen que ser
 * de una persona: son la señal de que se sigue adelante.
 */
export function vigilarIdentificador(alEscribir: (usuario: string) => void, doc: Document = document) {
  let plazo: ReturnType<typeof setTimeout> | undefined;
  const leer = () => {
    clearTimeout(plazo);
    const usuario = usuarioDeLaPagina(doc);
    if (usuario) alEscribir(usuario);
  };

  doc.addEventListener(
    "input",
    (e) => {
      if (!campoDeIdentificador(e.target, doc)) return;
      clearTimeout(plazo);
      plazo = setTimeout(leer, ESPERA_AL_TECLEAR);
    },
    true,
  );
  doc.addEventListener(
    "change",
    (e) => {
      if (campoDeIdentificador(e.target, doc)) leer();
    },
    true,
  );
  doc.addEventListener(
    "keydown",
    (e) => {
      if (e.isTrusted && e.key === "Enter") leer();
    },
    true,
  );
  doc.addEventListener(
    "click",
    (e) => {
      if (e.isTrusted) leer();
    },
    true,
  );
}
