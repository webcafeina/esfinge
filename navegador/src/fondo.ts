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
import { aceptado, alAceptar } from "./consentimiento";
import {
  alCambiarLaBoveda,
  atenderAlPanel,
  atenderConCuenta,
  conCuenta,
  tic,
  type PeticionDeCuenta,
} from "./concuenta";
import { hostDe, queMostrar, TEXTO_DE_INSIGNIA, type QueMostrar } from "./insignia";
import {
  sirvePara,
  vigente,
  VIDA_DEL_USUARIO,
  type Pendiente,
  type UsuarioEscrito,
} from "./pendientes";
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
async function pedir(p: Peticion, dePersona: boolean): Promise<Respuesta> {
  // **Con cuenta no se habla con la aplicación** (ADR 0040): la bóveda está aquí
  // y la contesta la extensión, con las mismas reglas.
  if (await conCuenta()) return atenderConCuenta(p, dePersona);
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
/* ------------------------------------------------ el icono de la barra */

/**
 * consultar pregunta a Esfinge **sin pedir permiso si falta**.
 *
 * Es la diferencia con `pedir`, y no es un detalle: `pedir` se empareja solo
 * cuando el permiso no vale, y eso hace aparecer en la ventana de Esfinge «un
 * navegador pide permiso». Desde el panel está bien, porque alguien lo ha abierto;
 * desde el refresco del icono —cada minuto y al cambiar de pestaña— sería un aviso
 * en la ventana cada poco **sin que nadie hubiera tocado nada**.
 */
async function consultar(p: Omit<Peticion, "version" | "testigo">): Promise<Respuesta> {
  // Con cuenta, lo mismo que `pedir`, y **sin contar como actividad**: esto lo
  // hacen el refresco del icono y las páginas, no una persona.
  if (await conCuenta()) return atenderConCuenta({ ...p, version: VERSION_DEL_PROTOCOLO } as Peticion, false);
  return hablar({ ...p, version: VERSION_DEL_PROTOCOLO, testigo: await testigoGuardado() });
}

/** Las variantes del icono, que genera `make icono` desde `build/icono-barra*.svg`. */
const RUTAS_DEL_ICONO = (v: QueMostrar["icono"]) => ({
  16: `iconos/barra-${v}-16.png`,
  32: `iconos/barra-${v}-32.png`,
});

/**
 * Las pestañas donde Esfinge ha rellenado, con la dirección en la que lo hizo.
 *
 * **En memoria del trabajador y en ningún otro sitio**, como todo lo de aquí: el
 * trabajador se muere solo y con él se va esto, que es lo que tiene que pasar. La
 * insignia de cada pestaña la conserva el navegador.
 */
const rellenadas = new Map<number, string>();

/** Cuándo se preguntó por última vez por cada pestaña, y con qué dirección. */
const ultimaVez = new Map<number, { url: string; cuando: number }>();
const pendientes = new Map<number, ReturnType<typeof setTimeout>>();

/**
 * **El freno de sesenta preguntas por minuto es del canal entero**, y lo gastan
 * también el panel y las páginas. Así que el refresco agrupa las ráfagas —cambiar
 * de pestaña deprisa dispara muchos eventos seguidos— y no vuelve a preguntar por la
 * misma pestaña y la misma dirección antes de este plazo, salvo que se le obligue.
 */
const AGRUPAR = 300;
const NO_REPETIR = 5000;

function programarRefresco(tabId: number, obligar = false) {
  clearTimeout(pendientes.get(tabId));
  pendientes.set(
    tabId,
    setTimeout(() => {
      pendientes.delete(tabId);
      refrescar(tabId, obligar).catch(() => {});
    }, AGRUPAR),
  );
}

async function refrescar(tabId: number, obligar: boolean) {
  let pestana: chrome.tabs.Tab;
  try {
    pestana = await api.tabs.get(tabId);
  } catch {
    return; // la pestaña ya no está
  }
  const url = pestana.url ?? "";
  // **Sin aceptar el aviso de datos no se pregunta nada a Esfinge** (ADR 0033): ni
  // siquiera se lanza el puente. El icono dice dónde se empieza.
  if (!(await aceptado())) {
    pintar(tabId, queMostrar({ url, aceptado: false }));
    return;
  }
  const antes = ultimaVez.get(tabId);
  if (!obligar && antes && antes.url === url && Date.now() - antes.cuando < NO_REPETIR) return;
  ultimaVez.set(tabId, { url, cuando: Date.now() });
  if (rellenadas.has(tabId) && rellenadas.get(tabId) !== url) rellenadas.delete(tabId);

  const q = { url, rellenado: rellenadas.get(tabId) === url } as Parameters<typeof queMostrar>[0];
  if (url.startsWith("https:")) {
    q.estado = await consultar({ que: "estado" });
    if (q.estado.ok && q.estado.estado?.abierta) {
      q.cuentas = await consultar({ que: "cuentas", origen: url });
    }
  }
  pintar(tabId, queMostrar(q));
}

/** pintar pone el icono, la insignia y la frase de una pestaña. Nunca lanza. */
function pintar(tabId: number, q: QueMostrar) {
  const sinFallo = (promesa: unknown) => Promise.resolve(promesa).catch(() => {});
  try {
    sinFallo(api.action.setIcon({ tabId, path: RUTAS_DEL_ICONO(q.icono) }));
    sinFallo(api.action.setBadgeText({ tabId, text: q.insignia }));
    if (q.insignia) {
      sinFallo(api.action.setBadgeBackgroundColor({ tabId, color: q.fondoInsignia }));
      // No está en todos los navegadores viejos; donde falta, el texto sale blanco
      // igualmente, que es el que está medido.
      const accion = api.action as unknown as {
        setBadgeTextColor?: (d: { tabId: number; color: string }) => Promise<void>;
      };
      sinFallo(accion.setBadgeTextColor?.({ tabId, color: TEXTO_DE_INSIGNIA }));
    }
    sinFallo(api.action.setTitle({ tabId, title: q.titulo }));
  } catch {
    /* la pestaña se ha cerrado mientras tanto */
  }
}

async function refrescarLaActiva(obligar: boolean) {
  const [pestana] = await api.tabs.query({ active: true, lastFocusedWindow: true });
  if (pestana?.id !== undefined) programarRefresco(pestana.id, obligar);
}

api.tabs.onActivated.addListener(({ tabId }) => programarRefresco(tabId));
api.tabs.onUpdated.addListener((tabId, cambio) => {
  // Una dirección nueva obliga; una carga que termina en la misma, no.
  if (cambio.url) programarRefresco(tabId, true);
  else if (cambio.status === "complete") programarRefresco(tabId);
});
api.windows.onFocusChanged.addListener(() => {
  refrescarLaActiva(false).catch(() => {});
});

/**
 * **Y cada minuto, solo la pestaña activa**, porque Esfinge no puede avisar a la
 * extensión cuando la bóveda se cierra sola: es la extensión la que tiene que
 * preguntar. Lo decidió el cliente sabiendo el precio —un permiso más y arrancar el
 * puente una vez por minuto—.
 *
 * La alarma se crea solo si no existe: crearla cada vez que el trabajador despierta
 * la reiniciaría, y con un trabajador que despierta a menudo no llegaría a sonar.
 */
const ALARMA = "refrescar-el-icono";
Promise.resolve(api.alarms.get(ALARMA))
  .then((alarma) => {
    if (!alarma) api.alarms.create(ALARMA, { periodInMinutes: 1 });
  })
  .catch(() => {});
api.alarms.onAlarm.addListener((alarma) => {
  if (alarma.name !== ALARMA) return;
  // Con cuenta, el mismo reloj cierra la bóveda si lleva el plazo sin tocarse y
  // sincroniza; después se pinta el icono con lo que haya quedado.
  aceptado()
    .then((si) => (si ? tic() : undefined))
    .catch(() => {})
    .finally(() => refrescarLaActiva(true).catch(() => {}));
});

// Cuando llega algo de otro equipo, el icono de la pestaña activa se pone al día.
alCambiarLaBoveda(() => {
  refrescarLaActiva(true).catch(() => {});
});

// Y al aceptar el aviso de datos en el panel, el icono de la pestaña activa deja de
// pedir que se abra.
alAceptar(() => {
  refrescarLaActiva(true).catch(() => {});
});

/* ------------------------------------------------ guardar desde la página */

/**
 * Lo que se acaba de enviar en cada pestaña, esperando a que alguien decida si se
 * guarda (ADR 0032).
 *
 * **La contraseña vive aquí, en la memoria de este trabajador, y en ningún otro
 * sitio.** Hace falta tenerla un momento porque al pulsar «Entrar» la página cambia
 * y su guion muere, y la tarjeta sale en la siguiente. Pero la tarjeta de la página
 * siguiente **no la recibe**: recibe el sitio, el usuario y las cuentas, y al pulsar
 * manda la decisión. Es este trabajador el que la guarda. Caduca a los dos minutos,
 * se va al decidir y al cerrar la pestaña, y **nunca va a `storage`**.
 */
const ofertasPendientes = new Map<number, Pendiente>();
api.tabs.onRemoved.addListener((tabId) => ofertasPendientes.delete(tabId));

/**
 * El usuario que una persona tecleó en la página de solo usuario de cada pestaña
 * —la primera de Google—, para que la de la contraseña sepa quién entra
 * (`identidad.ts`). **No es una contraseña**, pero tampoco sale de aquí más que
 * hacia el mismo sitio, y como el pendiente vive solo en memoria.
 */
const usuariosEscritos = new Map<number, UsuarioEscrito>();
api.tabs.onRemoved.addListener((tabId) => usuariosEscritos.delete(tabId));

/** usuarioEscritoPara es lo tecleado en esa pestaña, si vale para esa dirección. */
function usuarioEscritoPara(tabId: number, url: string): string {
  const u = usuariosEscritos.get(tabId);
  if (!u) return "";
  if (!sirvePara(u, url, Date.now(), VIDA_DEL_USUARIO)) {
    usuariosEscritos.delete(tabId);
    return "";
  }
  return u.usuario;
}

type MensajeDeTarjeta =
  | { que: "envio"; forma: Pendiente["forma"]; usuario: string; secreto: string }
  | { que: "mirar" }
  | { que: "descartar" }
  | { que: "usuario-escrito"; usuario: string }
  | { que: "quien-entra" }
  | {
      que: "decidir";
      accion: "guardar" | "actualizar" | "nunca" | "ahora-no";
      titulo?: string;
      id?: string;
    };

/**
 * atenderTarjeta hace lo que pide el guion de la página. **La pestaña y su dirección
 * las pone el navegador**, como en todo lo demás.
 */
async function atenderTarjeta(tabId: number, url: string, m: MensajeDeTarjeta): Promise<unknown> {
  switch (m.que) {
    case "envio":
      if (url.startsWith("https:") && m.secreto) {
        ofertasPendientes.set(tabId, {
          origen: url,
          // Sin usuario en el formulario —la página de la contraseña de Google—, el
          // que se tecleó en la página anterior del mismo sitio.
          usuario: (m.usuario ?? "").trim() || usuarioEscritoPara(tabId, url),
          secreto: m.secreto,
          forma: m.forma,
          cuando: Date.now(),
        });
      }
      return { ok: true };

    case "usuario-escrito":
      if (url.startsWith("https:") && m.usuario?.trim()) {
        usuariosEscritos.set(tabId, { origen: url, usuario: m.usuario.trim(), cuando: Date.now() });
      }
      return { ok: true };

    case "quien-entra":
      return { usuario: usuarioEscritoPara(tabId, url) };

    case "descartar":
      ofertasPendientes.delete(tabId);
      return { ok: true };

    case "mirar": {
      const p = ofertasPendientes.get(tabId);
      if (!p || !sirvePara(p, url, Date.now())) {
        ofertasPendientes.delete(tabId);
        return { nada: true };
      }
      // **Con `consultar`, que no se empareja**: una página que se carga no es
      // alguien pidiendo permiso para el navegador.
      const r = await consultar({
        que: "ofrecer",
        origen: p.origen,
        usuario: p.usuario,
        secreto: p.secreto,
        forma: p.forma,
      });
      if (!r.ok) {
        // Con la bóveda cerrada se guarda el pendiente y se dice que la abra: es lo
        // que eligió el cliente. Con cualquier otro problema, se olvida.
        if (r.motivo === "cerrada") {
          return { cerrada: true, sitio: hostDe(p.origen), usuario: p.usuario, forma: p.forma };
        }
        ofertasPendientes.delete(tabId);
        return { nada: true };
      }
      if (!r.oferta || r.oferta.accion === "nada") {
        ofertasPendientes.delete(tabId);
        return { nada: true };
      }
      return { oferta: r.oferta, usuario: p.usuario, forma: p.forma };
    }

    case "decidir": {
      const p = ofertasPendientes.get(tabId);
      if (m.accion === "ahora-no") {
        ofertasPendientes.delete(tabId);
        return { ok: true };
      }
      if (!p || !vigente(p, Date.now())) {
        ofertasPendientes.delete(tabId);
        return {
          ok: false,
          error: "Esta oferta ha caducado. Vuelve a entrar en el sitio para guardarla.",
        };
      }
      // **El origen es el del envío**, no el de la página en la que está la tarjeta:
      // lo que se guarda es para el sitio del que salió, y Esfinge lo comprueba.
      const r =
        m.accion === "nunca"
          ? await consultar({ que: "nunca-aqui", origen: p.origen })
          : m.accion === "guardar"
            ? await consultar({
                que: "guardar-cuenta",
                origen: p.origen,
                usuario: p.usuario,
                secreto: p.secreto,
                titulo: m.titulo ?? "",
              })
            : await consultar({
                que: "actualizar-cuenta",
                origen: p.origen,
                id: m.id ?? "",
                usuario: p.usuario,
                secreto: p.secreto,
              });
      if (r.ok) {
        ofertasPendientes.delete(tabId);
        programarRefresco(tabId, true);
      }
      return { ok: r.ok, error: r.error };
    }
  }
}

api.runtime.onConnect.addListener((puerto) => {
  // **La tarjeta de guardar habla por su propio puerto**, y lo atiende otra función:
  // aquí no se reenvía nada a Esfinge tal cual, se decide con lo que tiene este
  // trabajador.
  if (puerto.name === "tarjeta") {
    puerto.onMessage.addListener((m) => {
      const pestana = puerto.sender?.tab;
      const contestar = (r: unknown) => {
        try {
          puerto.postMessage(r);
        } catch {
          /* la página se ha ido */
        }
      };
      if (pestana?.id === undefined || !pestana.url) {
        contestar({ ok: false, nada: true });
        return;
      }
      const { id, url } = pestana;
      aceptado()
        .then((si) => (si ? atenderTarjeta(id, url, m as MensajeDeTarjeta) : { ok: false, nada: true }))
        .then(contestar)
        .catch((e) => contestar({ ok: false, error: `La extensión ha fallado por dentro: ${e}` }));
    });
    return;
  }

  // **El guion de la página avisa de que ha rellenado**, sin decir qué: basta con
  // abrir el puerto. La pestaña y su dirección las pone el navegador.
  if (puerto.name === "relleno-hecho") {
    const pestana = puerto.sender?.tab;
    if (pestana?.id !== undefined && pestana.url) {
      const { id, url } = pestana;
      aceptado()
        .then((si) => {
          if (!si) return;
          rellenadas.set(id, url);
          pintar(
            id,
            queMostrar({
              url,
              estado: { ok: true, estado: { existe: true, abierta: true } },
              rellenado: true,
            }),
          );
        })
        .catch(() => {});
    }
    return;
  }

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

    // **Sin aceptar el aviso de datos, nada de lo que llegue por aquí va a Esfinge**
    // (ADR 0033), ni del panel ni de las páginas. El panel no llega a preguntar
    // antes de aceptarlo; esto es lo que lo hace cumplir si algo lo intentara.
    aceptado()
      .then((si) => {
        if (!si) {
          contestar({
            ok: false,
            motivo: "sin-consentimiento",
            error: "Falta aceptar el aviso de datos de la extensión: abre su panel.",
          });
          return;
        }

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
        // **Lo de la cuenta solo desde el panel.** Una página no puede entrar, salir
        // ni desbloquear: por su puerto, eso ni se mira.
        if ((p as PeticionDeCuenta).cuenta !== undefined) {
          if (puerto.name !== "panel") {
            contestar({ ok: false, motivo: "no-entiendo", error: "Eso solo se pide desde el panel" });
            return;
          }
          atenderAlPanel(p as PeticionDeCuenta)
            .then((r) => {
              contestar(r as unknown as Respuesta);
              refrescarLaActiva(true).catch(() => {});
            })
            .catch((e) => contestar({ ok: false, error: `La extensión ha fallado por dentro: ${e}` }));
          return;
        }

        pedir(p as Peticion, puerto.name === "panel")
          .then((r) => {
            contestar(r);
            // **Al abrir el panel, el icono se pone al día con lo que el panel acaba de
            // saber**, sin volver a preguntar: el panel pide las cuentas nada más abrirse.
            const peticion = p as Peticion;
            if (puerto.name === "panel" && peticion.que === "cuentas" && peticion.origen) {
              api.tabs
                .query({ active: true, currentWindow: true })
                .then(([pestana]) => {
                  if (pestana?.id === undefined) return;
                  pintar(
                    pestana.id,
                    queMostrar({
                      url: peticion.origen,
                      cuentas: r,
                      rellenado: rellenadas.get(pestana.id) === peticion.origen,
                    }),
                  );
                })
                .catch(() => {});
            }
          })
          .catch((e) =>
            contestar({
              ok: false,
              error: `La extensión ha fallado por dentro: ${e}`,
            }),
          );
      })
      .catch((e) => contestar({ ok: false, error: `La extensión ha fallado por dentro: ${e}` }));
  });
});
