/**
 * El trabajador de fondo: lo único que habla con Esfinge.
 *
 * # Por qué existe, si el panel podría hablar directamente
 *
 * No podría: `runtime.connectNative` no está disponible en el panel, y
 * aunque lo estuviera, el panel **se muere al cerrarse** y con él se iría la
 * conversación a media respuesta. Aquí también hay una muerte, pero es la que se
 * puede manejar.
 *
 * # Lo que hay que tener presente
 *
 *   - **Este trabajador se muere solo, cada pocos minutos**, y con él el proceso
 *     que Esfinge lanzó. No es un fallo: es cómo funciona MV3. Así que **nada de
 *     lo que se guarde aquí puede hacer falta después**: el testigo va en
 *     `storage.local`, que sobrevive, y el puerto se vuelve a abrir cuando
 *     haga falta.
 *   - **El testigo se guarda, la contraseña no**, porque la contraseña no llega:
 *     Esfinge copia al portapapeles y por el canal solo vuelve cuánto tardará en
 *     borrarse.
 *   - **Y no se guarda nada de lo que se pregunta.** Ni el sitio, ni las cuentas
 *     que hay, ni cuándo. Un caché aquí sería la lista de sitios de la bóveda
 *     escrita en el perfil del navegador, sin cifrar, que es exactamente lo que
 *     Esfinge cifra en su disco.
 */
import { api } from "./api";
import {
  VERSION_DEL_PROTOCOLO,
  type Peticion,
  type Respuesta,
} from "./protocolo";

/** El nombre del host, el mismo que lleva el manifiesto que instala Esfinge. */
const HOST = "com.webcafeina.esfinge";

/** Dónde se guarda el permiso. Es lo único que sobrevive a este trabajador. */
const CLAVE_TESTIGO = "testigo";

/**
 * hablar manda una petición y espera la respuesta.
 *
 * **Un puerto por petición**, y no uno permanente: con MV3 el trabajador se
 * duerme y el puerto se cae de todas formas, así que un puerto de larga vida solo
 * añade estados intermedios que hay que recuperar. Abrirlo cuesta lanzar un
 * proceso pequeño, que es justo para lo que se hizo pequeño.
 */
function hablar(p: Peticion): Promise<Respuesta> {
  return new Promise((resolver) => {
    let puerto: chrome.runtime.Port;
    try {
      puerto = api.runtime.connectNative(HOST);
    } catch {
      return resolver(sinPuente());
    }

    // Si el proceso muere sin contestar —porque no está instalado, o porque el
    // manifiesto apunta a donde no hay nada— hay que contestar igual: dejar la
    // promesa colgada deja el panel girando para siempre.
    let contestado = false;
    puerto.onMessage.addListener((r: Respuesta) => {
      contestado = true;
      puerto.disconnect();
      resolver(r);
    });
    puerto.onDisconnect.addListener(() => {
      if (!contestado) resolver(sinPuente());
    });

    try {
      puerto.postMessage(p);
    } catch {
      resolver(sinPuente());
    }
  });
}

function sinPuente(): Respuesta {
  return {
    ok: false,
    motivo: "sin-esfinge",
    error:
      "No se puede hablar con Esfinge. Comprueba que está instalada y que el " +
      "canal con el navegador está encendido en sus Ajustes.",
  };
}

async function testigoGuardado(): Promise<string> {
  const g = await api.storage.local.get(CLAVE_TESTIGO);
  return typeof g[CLAVE_TESTIGO] === "string" ? g[CLAVE_TESTIGO] : "";
}

/**
 * pedir es lo que usa el panel: añade la versión y el testigo, y **vuelve a
 * intentarlo una vez si el permiso ya no vale**.
 *
 * Ese reintento es lo que hace que emparejar no sea un paso aparte: la primera
 * vez, y cada vez que el permiso se retira desde Esfinge, aquí se pide y se
 * guarda el que venga. Si la persona todavía no ha contestado en la ventana, lo
 * que vuelve es «sin-emparejar», que el panel sabe enseñar.
 */
async function pedir(p: Peticion): Promise<Respuesta> {
  const conTestigo = {
    ...p,
    version: VERSION_DEL_PROTOCOLO,
    testigo: await testigoGuardado(),
  };
  const r = await hablar(conTestigo);
  if (r.motivo !== "sin-emparejar") return r;

  const emparejado = await hablar({
    version: VERSION_DEL_PROTOCOLO,
    que: "emparejar",
    quien: nombreDelNavegador(),
  });
  if (!emparejado.ok || !emparejado.testigo) return emparejado;

  await api.storage.local.set({ [CLAVE_TESTIGO]: emparejado.testigo });
  return hablar({ ...conTestigo, testigo: emparejado.testigo });
}

/**
 * nombreDelNavegador es lo que Esfinge enseñará al preguntar si se permite.
 *
 * **Esfinge no se lo cree, y hace bien**: cualquiera que hable por el canal puede
 * decir que es Chrome. Sirve para que en el diálogo ponga algo reconocible, no
 * para decidir nada.
 */
function nombreDelNavegador(): string {
  const agente = navigator.userAgent;
  if (agente.includes("Edg/")) return "Edge";
  if (agente.includes("Firefox/")) return "Firefox";
  if (agente.includes("Chrome/")) return "Chrome";
  return "Un navegador";
}

api.runtime.onMessage.addListener((p: Peticion, _origen, contestar) => {
  pedir(p).then(contestar);
  return true; // la respuesta llega después
});
