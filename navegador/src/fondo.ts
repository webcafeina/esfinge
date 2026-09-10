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
    let hecho = false;
    const terminar = (r: Respuesta) => {
      if (hecho) return;
      hecho = true;
      resolver(r);
    };

    // **Un plazo, y aquí es donde faltaba.** El panel tenía el suyo, pero esto no:
    // si el proceso arranca y no contesta —o si el aviso de que no ha arrancado
    // se pierde por el camino— esta promesa no se resolvía nunca, el trabajador
    // no contestaba, y lo único que veía quien miraba era «el trabajador de fondo
    // no ha contestado», que apunta al sitio equivocado. Un plazo aquí dice **qué
    // tramo** es el que no responde.
    const plazo = setTimeout(() => {
      terminar({
        ok: false,
        motivo: "sin-esfinge",
        error:
          "El puente de Esfinge no ha contestado. Comprueba que Esfinge está " +
          "instalada y que el canal con el navegador está encendido en sus Ajustes.",
      });
    }, PLAZO_DEL_PUENTE);

    let puerto: chrome.runtime.Port;
    try {
      puerto = api.runtime.connectNative(HOST);
    } catch (e) {
      clearTimeout(plazo);
      return terminar(sinPuente(`${e}`));
    }

    puerto.onMessage.addListener((r: Respuesta) => {
      clearTimeout(plazo);
      puerto.disconnect();
      terminar(r);
    });
    // Se dispara cuando el proceso no está, no se puede ejecutar o se muere. El
    // motivo lo pone el navegador y **se enseña tal cual**: es lo único que dice
    // si el problema es el manifiesto, la ruta o los permisos.
    puerto.onDisconnect.addListener(() => {
      clearTimeout(plazo);
      terminar(sinPuente(api.runtime.lastError?.message));
    });

    try {
      puerto.postMessage(p);
    } catch (e) {
      clearTimeout(plazo);
      terminar(sinPuente(`${e}`));
    }
  });
}

/** Lo que se espera al proceso antes de darlo por perdido. */
const PLAZO_DEL_PUENTE = 4000;

function sinPuente(porque?: string): Respuesta {
  return {
    ok: false,
    motivo: "sin-esfinge",
    error:
      "No se puede hablar con Esfinge. Comprueba que está instalada y que el " +
      "canal con el navegador está encendido en sus Ajustes." +
      // **Lo que diga el navegador, tal cual.** Es lo único que distingue «no
      // encuentro el manifiesto» de «no puedo ejecutar eso» de «se ha muerto», y
      // sin ello los tres se ven igual desde fuera.
      (porque ? ` El navegador dice: ${porque}` : ""),
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

/**
 * El panel habla por un **puerto**, no con un mensaje suelto.
 *
 * **Y esto es la tercera versión de estas cuatro líneas**, así que conviene
 * dejar escrito por qué. Con `onMessage` hay que decir que la respuesta llega
 * después, y eso **no se dice igual en los dos navegadores**: Chrome quiere un
 * `return true` y una retrollamada; Firefox quiere que se devuelva la promesa.
 * Con el `return true` de Chrome, Firefox contesta con un error que lo dice todo
 * —«Promised response from onMessage listener went out of scope»— y el panel se
 * queda sin respuesta.
 *
 * Un puerto no tiene esa ambigüedad: no hay que prometer nada, la respuesta es
 * otro mensaje y punto. Es lo mismo en los dos navegadores y no hay que detectar
 * cuál es.
 */
api.runtime.onConnect.addListener((puerto) => {
  puerto.onMessage.addListener((p) => {
    const contestar = (r: Respuesta) => {
      // Quien preguntaba puede haberse ido mientras se preguntaba —el panel se
      // cierra al hacer clic fuera, la pestaña se recarga—, y entonces escribir en
      // el puerto lanza. No es un fallo: es que ya no hay nadie al otro lado.
      try {
        puerto.postMessage(r);
      } catch {
        /* se ha ido */
      }
    };

    // **El origen de una página lo dice el navegador, no la página**, y aquí es
    // donde eso deja de ser una frase y es una línea. Un puerto llamado «pagina»
    // viene del guion que Esfinge pone en las páginas, y lo que llegue por él en
    // el campo `origen` se tira y se pone `sender.tab.url`, que lo rellena el
    // navegador y es lo que se ve en la barra de direcciones.
    //
    // Sin esto, cualquier página con una vulnerabilidad que le dejara hablar por
    // este puerto podría pedir las cuentas de un banco diciendo que es el banco.
    // Con esto, lo peor que puede pedir es lo suyo.
    //
    // **Y si llega vacía, se dice.** No es una hipótesis: en Firefox los
    // `host_permissions` de MV3 no se conceden al instalar, hay que darlos en el
    // panel de extensiones. Sin ellos `sender.tab.url` llega `undefined` —sin
    // error—, el origen viaja vacío y Esfinge contesta que ahí no rellena. Desde
    // fuera eso parece un fallo de Esfinge y no un permiso que falta, que es
    // exactamente la clase de silencio que ya costó cinco versiones con
    // `storage`. Aquí se convierte en una frase que dice qué hacer.
    if (puerto.name === "pagina") {
      const donde = puerto.sender?.tab?.url ?? "";
      if (!donde) {
        contestar({
          ok: false,
          motivo: "origen",
          error:
            "La extensión no puede ver la dirección de esta pestaña. Dale permiso a " +
            "Esfinge para este sitio: en Firefox, en el botón de extensiones de la " +
            "barra; en Chrome, en «Gestionar extensión» → «Acceso a sitios web».",
        });
        return;
      }
      (p as Peticion).origen = donde;
    }

    // **Con `catch`, y es la lección de todo el día por tercera vez.** Sin él,
    // cualquier excepción aquí dentro es una promesa rechazada que nadie recoge:
    // el trabajador no contesta, el panel espera su plazo y lo que se ve es «el
    // trabajador de fondo no ha contestado», que **no dice nada del problema**.
    // Así pasó con un permiso que faltaba en el manifiesto: `api.storage` era
    // undefined, esto lanzaba en la primera línea, y desde fuera parecía un
    // problema del puente. Cinco versiones persiguiendo eso.
    pedir(p as Peticion)
      .then(contestar)
      .catch((e) =>
        contestar({
          ok: false,
          error: `La extensión ha fallado por dentro: ${e}`,
        }),
      );
  });
});
