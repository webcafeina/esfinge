import { useCallback, useEffect, useRef, useState } from "react";
import {
  alBloquearseLaBoveda,
  alCambiarElPortapapeles,
  esfinge,
  type EntradaBoveda,
  type EstadoBoveda,
  type ResumenImportacion,
  type TipoEntrada,
} from "./puente";
import { CampoClave, Segmentado } from "./componentes";

/**
 * La bóveda, asomada a la ventana.
 *
 * Va en su propio fichero y no dentro de App.tsx a propósito: aquella pantalla
 * ya pasa de novecientas líneas, y esto no es una sección más —es media docena
 * de estados que se turnan (no hay bóveda, está cerrada, se está creando, se
 * está enseñando la clave de recuperación, se está mirando una entrada)—.
 *
 * Dos reglas gobiernan todo lo de aquí, y las dos vienen de Go:
 *
 *   - **Los secretos se piden de uno en uno.** La lista llega sin contraseñas;
 *     la contraseña de una entrada solo se pide al abrirla. Eso hace más por que
 *     un secreto no acabe en un volcado de memoria que cualquier limpieza de
 *     variables, porque en cuanto algo cruza el puente vive en el montón del
 *     webview hasta que su recolector decida.
 *   - **Los relojes son de Go.** El bloqueo por inactividad y el borrado del
 *     portapapeles no se llevan con un setTimeout: el temporizador de un webview
 *     se pausa y muere al recargar, y un bloqueo que a veces no ocurre no es un
 *     bloqueo. Aquí solo se escucha lo que Go decide.
 */
export function Boveda({ activo }: { activo: boolean }) {
  const [estado, setEstado] = useState<EstadoBoveda | null>(null);
  const [error, setError] = useState("");
  // La clave de recuperación, cuando toca enseñarla. **Es la única vez que se
  // ve**, así que mientras esté puesta tapa todo lo demás: nada de que se pierda
  // por un clic en otro sitio.
  const [ceremonia, setCeremonia] = useState<{ clave: string; nueva: boolean } | null>(null);

  const refrescar = useCallback(async () => {
    try {
      setEstado(await esfinge.estadoBoveda());
    } catch (e) {
      setError(mensaje(e));
    }
  }, []);

  useEffect(() => {
    if (activo) refrescar();
  }, [activo, refrescar]);

  // Cuando Go la cierra por inactividad, la pantalla tiene que enterarse aunque
  // nadie esté mirando esta sección: si no, se vuelve y se ve la lista de antes,
  // que ya no se puede pedir.
  useEffect(() => alBloquearseLaBoveda(() => {
    setCeremonia(null);
    refrescar();
  }), [refrescar]);

  usaLatidoDeActividad(activo && (estado?.abierta ?? false));

  if (error) return <div className="panel"><p className="error">{error}</p></div>;
  if (!estado) return <div className="panel" />;

  if (ceremonia) {
    return (
      <Ceremonia
        clave={ceremonia.clave}
        nueva={ceremonia.nueva}
        alSeguir={() => {
          setCeremonia(null);
          refrescar();
        }}
      />
    );
  }

  if (!estado.existe) {
    return (
      <Crear
        alCrear={(clave) => {
          setCeremonia({ clave, nueva: true });
        }}
      />
    );
  }

  if (!estado.abierta) return <Cerrada estado={estado} alAbrir={refrescar} />;

  return (
    <Dentro
      estado={estado}
      alCambiar={refrescar}
      alRotar={(clave) => setCeremonia({ clave, nueva: false })}
    />
  );
}

/**
 * usaLatidoDeActividad le dice a Go que hay alguien delante.
 *
 * **Con moderación**: una vez cada diez segundos como mucho, y solo si se ha
 * tocado algo. Cruzar el puente en cada movimiento del ratón sería gastar por
 * gastar, y lo que hay que aplazar es un plazo de minutos.
 */
function usaLatidoDeActividad(encendido: boolean) {
  const ultimo = useRef(0);

  useEffect(() => {
    if (!encendido) return;

    const avisar = () => {
      const ahora = Date.now();
      if (ahora - ultimo.current < 10_000) return;
      ultimo.current = ahora;
      esfinge.actividad().catch(() => {});
    };

    avisar();
    window.addEventListener("pointerdown", avisar);
    window.addEventListener("keydown", avisar);
    return () => {
      window.removeEventListener("pointerdown", avisar);
      window.removeEventListener("keydown", avisar);
    };
  }, [encendido]);
}

// --------------------------------------------------------------- crear y abrir

/** El mínimo de la contraseña maestra. Corta abajo, no es una opinión de estilo. */
const MINIMO_MAESTRA = 10;

function Crear({ alCrear }: { alCrear: (recuperacion: string) => void }) {
  const [maestra, setMaestra] = useState("");
  const [repetida, setRepetida] = useState("");
  const [trabajando, setTrabajando] = useState(false);
  const [error, setError] = useState("");

  const cortas = maestra.length < MINIMO_MAESTRA;
  const distintas = repetida !== "" && repetida !== maestra;
  const listo = !cortas && !distintas && repetida !== "";

  async function crear() {
    setTrabajando(true);
    setError("");
    try {
      alCrear(await esfinge.crearBoveda(maestra));
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  return (
    <div className="panel">
      <p className="entradilla">
        Un sitio cifrado donde guardar contraseñas, notas, tarjetas y documentos. Vive en este
        ordenador y no sale de aquí.
      </p>

      <div className="grupo">
        <CampoClave
          id="boveda-maestra"
          etiqueta="Contraseña maestra"
          valor={maestra}
          alCambiar={setMaestra}
        />

        <div>
          <label htmlFor="boveda-maestra-2">Repítela</label>
          <input
            id="boveda-maestra-2"
            type="password"
            autoComplete="off"
            value={repetida}
            onChange={(e) => setRepetida(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && listo && crear()}
          />
          {distintas && <p className="error">Las dos no coinciden</p>}
          {!distintas && cortas && maestra !== "" && (
            <p className="nota">Al menos {MINIMO_MAESTRA} caracteres</p>
          )}
        </div>
      </div>

      <p className="aviso">
        Esta contraseña abre todo lo que guardes aquí. No se guarda en ninguna parte, así que
        nadie —tampoco nosotros— puede recuperarla.
      </p>
      <p className="nota">
        Por eso, al crearla, verás <strong>una vez</strong> una clave de recuperación para
        apuntar en papel. Es la única segunda puerta que vas a tener.
      </p>

      {error && <p className="error">{error}</p>}

      <div className="botones">
        <button className="principal" onClick={crear} disabled={!listo || trabajando}>
          {trabajando ? "Creando…" : "Crear la bóveda"}
        </button>
      </div>
    </div>
  );
}

/**
 * La ceremonia de la clave de recuperación.
 *
 * Se enseña una vez y no se puede volver a pedir. Por eso hay una casilla que
 * hay que marcar para seguir: no es burocracia, es la última oportunidad.
 */
function Ceremonia({
  clave,
  nueva,
  alSeguir,
}: {
  clave: string;
  nueva: boolean;
  alSeguir: () => void;
}) {
  const [guardada, setGuardada] = useState(false);
  const [copiado, setCopiado] = useState(false);
  const [guardadoEn, setGuardadoEn] = useState("");
  const [error, setError] = useState("");

  async function copiar() {
    // **Aquí sí se copia por el navegador y no con esfinge.copiar.** El de Go
    // arma el borrado a los treinta segundos, que para una contraseña que se va a
    // pegar enseguida está bien y para esto es una trampa: quien copia esta clave
    // para dejarla en el gestor de la empresa puede tardar más, y no hay forma de
    // volver a verla.
    try {
      await navigator.clipboard.writeText(clave);
      setCopiado(true);
    } catch {
      setError("No he podido usar el portapapeles; guárdala en un fichero");
    }
  }

  async function guardar() {
    try {
      const donde = await esfinge.guardarTexto("clave-de-recuperacion-esfinge.txt", clave);
      if (donde) setGuardadoEn(donde);
    } catch (e) {
      setError(mensaje(e));
    }
  }

  return (
    <div className="panel">
      <h2>{nueva ? "Apunta esta clave de recuperación" : "Ésta es tu clave nueva"}</h2>
      <p className="entradilla">
        Abre la bóveda si olvidas la contraseña maestra. <strong>Se enseña una sola vez</strong>:
        no se guarda en ninguna parte y no se puede volver a pedir.
      </p>

      <div className="clave-recuperacion seleccionable">{clave}</div>

      {copiado && <p className="exito">Copiada al portapapeles</p>}
      {guardadoEn && <p className="exito">Guardada en {guardadoEn}</p>}
      {error && <p className="error">{error}</p>}

      <p className="aviso">
        Guárdala en otro sitio distinto de donde guardes la maestra. Quien la tenga entra en la
        bóveda: es una segunda llave, no una copia de seguridad.
      </p>
      {!nueva && <p className="nota">La clave anterior ya no vale.</p>}

      <div className="botones">
        <button className="principal" onClick={copiar}>
          Copiar
        </button>
        <button onClick={guardar}>Guardar como…</button>
      </div>

      <label className="fila-ajuste">
        <input
          type="checkbox"
          checked={guardada}
          onChange={(e) => setGuardada(e.target.checked)}
        />
        <span>La he apuntado en un sitio seguro</span>
      </label>

      <div className="botones">
        <button className="principal" onClick={alSeguir} disabled={!guardada}>
          Continuar
        </button>
      </div>
    </div>
  );
}

function Cerrada({ estado, alAbrir }: { estado: EstadoBoveda; alAbrir: () => void }) {
  const [llave, setLlave] = useState("");
  const [trabajando, setTrabajando] = useState(false);
  const [error, setError] = useState("");

  async function abrir() {
    setTrabajando(true);
    setError("");
    try {
      await esfinge.abrirBoveda(llave);
      setLlave("");
      alAbrir();
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  return (
    <div className="panel">
      <p className="entradilla">La bóveda está cerrada.</p>

      <div className="grupo">
        <div>
          <label htmlFor="boveda-llave">Contraseña maestra o clave de recuperación</label>
          <input
            id="boveda-llave"
            type="password"
            autoComplete="off"
            value={llave}
            onChange={(e) => setLlave(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && llave && abrir()}
          />
        </div>
      </div>

      {error && <p className="error">{error}</p>}

      <div className="botones">
        <button className="principal" onClick={abrir} disabled={!llave || trabajando}>
          {trabajando ? "Abriendo…" : "Abrir la bóveda"}
        </button>
      </div>

      {estado.minutosParaBloquear > 0 && (
        <p className="nota">
          Se cierra sola tras {estado.minutosParaBloquear} minutos sin tocar nada. Se cambia en
          Ajustes.
        </p>
      )}
      <p className="nota seleccionable">Se guarda en {estado.ruta}</p>
    </div>
  );
}

// ---------------------------------------------------------------------- dentro

function Dentro({
  estado,
  alCambiar,
  alRotar,
}: {
  estado: EstadoBoveda;
  alCambiar: () => void;
  alRotar: (clave: string) => void;
}) {
  const [q, setQ] = useState("");
  const [lista, setLista] = useState<EntradaBoveda[]>([]);
  const [mirando, setMirando] = useState<EntradaBoveda | null>(null);
  const [editando, setEditando] = useState<EntradaBoveda | null>(null);
  const [error, setError] = useState("");

  const buscar = useCallback(async (texto: string) => {
    try {
      setLista(await esfinge.buscarEnBoveda(texto));
    } catch (e) {
      setError(mensaje(e));
    }
  }, []);

  // La búsqueda cruza el puente, así que se espera a que se deje de teclear. No
  // es por coste: es que cada pulsación devolvería una lista y las respuestas
  // pueden llegar desordenadas.
  useEffect(() => {
    const t = setTimeout(() => buscar(q), 120);
    return () => clearTimeout(t);
  }, [q, buscar]);

  async function ver(id: string) {
    setError("");
    try {
      setMirando(await esfinge.verDeBoveda(id));
    } catch (e) {
      setError(mensaje(e));
    }
  }

  async function cerrar() {
    await esfinge.cerrarBoveda();
    alCambiar();
  }

  if (editando) {
    return (
      <Editor
        entrada={editando}
        alGuardar={async (e) => {
          await esfinge.guardarEnBoveda(e);
          setEditando(null);
          setMirando(null);
          await buscar(q);
          alCambiar();
        }}
        alDejarlo={() => setEditando(null)}
      />
    );
  }

  if (mirando) {
    return (
      <Detalle
        entrada={mirando}
        alVolver={() => setMirando(null)}
        alEditar={() => setEditando(mirando)}
        alBorrar={async () => {
          await esfinge.borrarDeBoveda(mirando.id);
          setMirando(null);
          await buscar(q);
          alCambiar();
        }}
      />
    );
  }

  return (
    <div className="panel">
      <div className="boveda-barra">
        <input
          id="boveda-buscar"
          type="text"
          value={q}
          placeholder="Buscar"
          aria-label="Buscar en la bóveda"
          onChange={(e) => setQ(e.target.value)}
        />
        <button onClick={() => setEditando(entradaNueva())}>Nueva</button>
        <button onClick={cerrar}>Cerrar la bóveda</button>
      </div>

      {error && <p className="error">{error}</p>}

      {lista.length === 0 ? (
        <p className="nota">
          {q ? "Nada encaja con esa búsqueda." : "La bóveda está vacía. Añade algo o importa lo que ya tengas."}
        </p>
      ) : (
        <ul className="lista-boveda">
          {lista.map((e) => (
            <li key={e.id}>
              <button onClick={() => ver(e.id)}>
                <span className="que">{ICONO_TIPO[e.tipo] ?? "•"}</span>
                <span className="nombre">{e.titulo || "Sin título"}</span>
                <span className="nota">{deQuien(e)}</span>
              </button>
            </li>
          ))}
        </ul>
      )}

      <p className="nota">
        {estado.cuantas === 1 ? "Una entrada" : `${estado.cuantas} entradas`}
        {estado.minutosParaBloquear > 0 &&
          ` · se cierra sola tras ${estado.minutosParaBloquear} minutos sin tocar nada`}
      </p>

      <Traer alTraer={async () => {
        await buscar(q);
        alCambiar();
      }} />

      <Seguridad alRotar={alRotar} />
    </div>
  );
}

/** El glifo de cada clase de entrada, para distinguirlas de un vistazo. */
const ICONO_TIPO: Record<TipoEntrada, string> = {
  credencial: "🔑",
  nota: "📝",
  tarjeta: "💳",
  identidad: "🪪",
};

const NOMBRE_TIPO: Record<TipoEntrada, string> = {
  credencial: "Credencial",
  nota: "Nota",
  tarjeta: "Tarjeta",
  identidad: "Identidad",
};

/** Lo que va debajo del título en la lista: de quién es esta entrada. */
function deQuien(e: EntradaBoveda): string {
  return e.usuario || e.sitios?.[0] || e.titular || e.nombreCompleto || "";
}

function entradaNueva(): EntradaBoveda {
  return { id: "", tipo: "credencial", titulo: "", creada: "", cambiada: "" };
}

// --------------------------------------------------------------------- detalle

function Detalle({
  entrada,
  alVolver,
  alEditar,
  alBorrar,
}: {
  entrada: EntradaBoveda;
  alVolver: () => void;
  alEditar: () => void;
  alBorrar: () => void;
}) {
  // Borrar pide una segunda pulsación en vez de un diálogo. El diálogo del
  // sistema pararía la ventana entera para una pregunta que se contesta aquí.
  const [seguro, setSeguro] = useState(false);
  const [verHistorial, setVerHistorial] = useState(false);

  return (
    <div className="panel">
      <div className="boveda-barra">
        <button onClick={alVolver}>← Volver</button>
        <span className="crece" />
        <button onClick={alEditar}>Editar</button>
        <button
          className={seguro ? "principal" : undefined}
          onClick={() => (seguro ? alBorrar() : setSeguro(true))}
        >
          {seguro ? "Sí, borrar" : "Borrar"}
        </button>
      </div>

      <h2>{entrada.titulo || "Sin título"}</h2>
      <p className="nota">
        {NOMBRE_TIPO[entrada.tipo]}
        {entrada.carpeta ? ` · ${entrada.carpeta}` : ""}
      </p>

      <div className="grupo">
        <Dato etiqueta="Usuario" valor={entrada.usuario} />
        <Secreto etiqueta="Contraseña" valor={entrada.secreto} />
        <Dato etiqueta="Sitios" valor={entrada.sitios?.join("\n")} />
        <Secreto etiqueta="Semilla del código de un solo uso" valor={entrada.totp} />

        <Dato etiqueta="Titular" valor={entrada.titular} />
        <Secreto etiqueta="Número de la tarjeta" valor={entrada.numero} />
        <Dato etiqueta="Caduca" valor={entrada.caduca} />
        <Secreto etiqueta="Código de verificación" valor={entrada.verificacion} />

        <Dato etiqueta="Nombre completo" valor={entrada.nombreCompleto} />
        <Dato etiqueta="Documento" valor={entrada.documento} />
        <Secreto etiqueta="Número del documento" valor={entrada.numeroDocumento} />

        <Dato etiqueta="Notas" valor={entrada.notas} />
        <Dato etiqueta="Etiquetas" valor={entrada.etiquetas?.join(", ")} />
      </div>

      {entrada.historial && entrada.historial.length > 0 && (
        <div className="grupo">
          <div className="fila">
            <label style={{ marginBottom: 0 }}>Contraseñas anteriores</label>
            <button className="discreto" onClick={() => setVerHistorial(!verHistorial)}>
              {verHistorial ? "Ocultar" : `Ver las ${entrada.historial.length}`}
            </button>
          </div>
          {/* Se dice en voz alta porque es una consecuencia de guardarlas, no un
              detalle: un secreto sustituido sigue dentro de la bóveda. */}
          <p className="nota">
            Siguen dentro de la bóveda hasta que se borre la entrada.
          </p>
          {verHistorial &&
            entrada.historial.map((h, i) => (
              <Secreto
                key={i}
                etiqueta={`Hasta el ${fecha(h.hasta)}`}
                valor={h.secreto}
              />
            ))}
        </div>
      )}

      <p className="nota">Cambiada el {fecha(entrada.cambiada)}</p>
    </div>
  );
}

/** Un dato que no es un secreto: se ve y se puede seleccionar. */
function Dato({ etiqueta, valor }: { etiqueta: string; valor?: string }) {
  if (!valor) return null;
  return (
    <div>
      <label>{etiqueta}</label>
      <p className="dato seleccionable">{valor}</p>
    </div>
  );
}

/**
 * Un dato que sí lo es: tapado, con un botón para verlo y otro para copiarlo.
 *
 * Copiar va por Go —`esfinge.copiar`— y no por el navegador, que es lo que arma
 * el borrado pasado el plazo.
 */
function Secreto({ etiqueta, valor }: { etiqueta: string; valor?: string }) {
  const [visible, setVisible] = useState(false);
  const [dicho, setDicho] = useState("");
  const [quedan, setQuedan] = useState(0);

  useEffect(() => alCambiarElPortapapeles(setQuedan), []);

  // La cuenta atrás la lleva la pantalla; el borrado, Go. Aquí solo se enseña.
  useEffect(() => {
    if (quedan <= 0) return;
    const t = setTimeout(() => setQuedan((n) => n - 1), 1000);
    return () => clearTimeout(t);
  }, [quedan]);

  if (!valor) return null;

  async function copiar() {
    try {
      await esfinge.copiar(valor!);
      setDicho("Copiado");
    } catch (e) {
      setDicho(mensaje(e));
    }
  }

  return (
    <div>
      <div className="fila">
        <label style={{ marginBottom: 0 }}>{etiqueta}</label>
        <span>
          <button className="discreto" onClick={() => setVisible(!visible)}>
            {visible ? "Ocultar" : "Ver"}
          </button>
          <button className="discreto" onClick={copiar}>
            Copiar
          </button>
        </span>
      </div>
      <p className={visible ? "dato secreto seleccionable" : "dato secreto"}>
        {visible ? valor : "••••••••••••"}
      </p>
      {dicho && (
        <p className="exito">
          {dicho}
          {quedan > 0 && ` · se borra del portapapeles en ${quedan} s`}
        </p>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------- editor

function Editor({
  entrada,
  alGuardar,
  alDejarlo,
}: {
  entrada: EntradaBoveda;
  alGuardar: (e: EntradaBoveda) => Promise<void>;
  alDejarlo: () => void;
}) {
  const [e, setE] = useState<EntradaBoveda>(entrada);
  const [trabajando, setTrabajando] = useState(false);
  const [error, setError] = useState("");
  const nueva = entrada.id === "";

  const pon = (cambio: Partial<EntradaBoveda>) => setE((a) => ({ ...a, ...cambio }));

  async function guardar() {
    setTrabajando(true);
    setError("");
    try {
      await alGuardar(e);
    } catch (err) {
      setError(mensaje(err));
      setTrabajando(false);
    }
  }

  return (
    <div className="panel">
      <div className="boveda-barra">
        <button onClick={alDejarlo}>← Dejarlo</button>
        <span className="crece" />
        <button className="principal" onClick={guardar} disabled={!e.titulo.trim() || trabajando}>
          {trabajando ? "Guardando…" : "Guardar"}
        </button>
      </div>

      <div className="grupo">
        {nueva && (
          <div>
            <label>Qué es</label>
            <Segmentado<TipoEntrada>
              valor={e.tipo}
              alCambiar={(t) => pon({ tipo: t })}
              opciones={(Object.keys(NOMBRE_TIPO) as TipoEntrada[]).map((t) => ({
                valor: t,
                etiqueta: NOMBRE_TIPO[t],
              }))}
            />
          </div>
        )}

        <Campo id="boveda-titulo" etiqueta="Título" valor={e.titulo} alCambiar={(v) => pon({ titulo: v })} />

        {e.tipo === "credencial" && (
          <>
            <Campo id="boveda-usuario" etiqueta="Usuario" valor={e.usuario} alCambiar={(v) => pon({ usuario: v })} />
            <CampoClave
              id="boveda-secreto"
              etiqueta="Contraseña"
              valor={e.secreto ?? ""}
              alCambiar={(v) => pon({ secreto: v })}
              alGenerar={async () => {
                const m = await esfinge.medirPorCaracteres(24, "alnum");
                pon({ secreto: await esfinge.generarContrasena(m.bytes, "alnum") });
              }}
            />
            <Campo
              id="boveda-sitios"
              etiqueta="Sitios"
              valor={e.sitios?.join("\n")}
              alCambiar={(v) => pon({ sitios: enLineas(v) })}
              largo
              pista="Uno por línea"
            />
            <Campo
              id="boveda-totp"
              etiqueta="Semilla del código de un solo uso"
              valor={e.totp}
              alCambiar={(v) => pon({ totp: v })}
              pista="La cadena en base32 que da el servicio, no el código de seis cifras"
            />
          </>
        )}

        {e.tipo === "tarjeta" && (
          <>
            <Campo id="boveda-titular" etiqueta="Titular" valor={e.titular} alCambiar={(v) => pon({ titular: v })} />
            <Campo id="boveda-numero" etiqueta="Número" valor={e.numero} alCambiar={(v) => pon({ numero: v })} />
            <Campo id="boveda-caduca" etiqueta="Caduca" valor={e.caduca} alCambiar={(v) => pon({ caduca: v })} pista="MM/AA" />
            <Campo
              id="boveda-verificacion"
              etiqueta="Código de verificación"
              valor={e.verificacion}
              alCambiar={(v) => pon({ verificacion: v })}
            />
          </>
        )}

        {e.tipo === "identidad" && (
          <>
            <Campo
              id="boveda-nombre"
              etiqueta="Nombre completo"
              valor={e.nombreCompleto}
              alCambiar={(v) => pon({ nombreCompleto: v })}
            />
            <Campo
              id="boveda-documento"
              etiqueta="Documento"
              valor={e.documento}
              alCambiar={(v) => pon({ documento: v })}
              pista="DNI, pasaporte, lo que sea"
            />
            <Campo
              id="boveda-num-documento"
              etiqueta="Número del documento"
              valor={e.numeroDocumento}
              alCambiar={(v) => pon({ numeroDocumento: v })}
            />
          </>
        )}

        <Campo id="boveda-notas" etiqueta="Notas" valor={e.notas} alCambiar={(v) => pon({ notas: v })} largo />
        <Campo
          id="boveda-etiquetas"
          etiqueta="Etiquetas"
          valor={e.etiquetas?.join(", ")}
          alCambiar={(v) => pon({ etiquetas: enComas(v) })}
          pista="Separadas por comas"
        />
      </div>

      {error && <p className="error">{error}</p>}
    </div>
  );
}

function Campo({
  id,
  etiqueta,
  valor,
  alCambiar,
  largo,
  pista,
}: {
  id: string;
  etiqueta: string;
  valor?: string;
  alCambiar: (v: string) => void;
  largo?: boolean;
  pista?: string;
}) {
  return (
    <div>
      <label htmlFor={id}>{etiqueta}</label>
      {largo ? (
        <textarea id={id} value={valor ?? ""} onChange={(ev) => alCambiar(ev.target.value)} />
      ) : (
        <input
          id={id}
          type="text"
          autoComplete="off"
          value={valor ?? ""}
          onChange={(ev) => alCambiar(ev.target.value)}
        />
      )}
      {pista && <p className="nota">{pista}</p>}
    </div>
  );
}

const enLineas = (v: string) => v.split("\n").map((s) => s.trim()).filter(Boolean);
const enComas = (v: string) => v.split(",").map((s) => s.trim()).filter(Boolean);

// -------------------------------------------------------- importar y exportar

/** Los gestores de los que se sabe leer. Los nombres los reconoce Go por columnas. */
const GESTORES = ["Dashlane", "Bitwarden", "1Password", "LastPass", "Chrome"];

function Traer({ alTraer }: { alTraer: () => Promise<void> }) {
  const [deDonde, setDeDonde] = useState(GESTORES[0]);
  const [resumen, setResumen] = useState<ResumenImportacion | null>(null);
  const [borrado, setBorrado] = useState(false);
  const [salida, setSalida] = useState("");
  const [seguro, setSeguro] = useState(false);
  const [error, setError] = useState("");
  const [trabajando, setTrabajando] = useState(false);

  async function importar() {
    setError("");
    setResumen(null);
    setBorrado(false);
    setTrabajando(true);
    try {
      const r = await esfinge.importarEnBoveda(deDonde);
      if (!r.fichero) return; // lo ha cancelado, que no es un error
      setResumen(r);
      await alTraer();
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  async function exportar() {
    setError("");
    setSalida("");
    try {
      const donde = await esfinge.exportarBoveda();
      if (donde) setSalida(donde);
      setSeguro(false);
    } catch (e) {
      setError(mensaje(e));
    }
  }

  return (
    <div className="grupo">
      <div>
        <label>Traer de otro gestor</label>
        <Segmentado
          valor={deDonde}
          alCambiar={setDeDonde}
          opciones={GESTORES.map((g) => ({ valor: g, etiqueta: g }))}
        />
      </div>

      {/* **Esto no es una nota de cortesía.** Dashlane no exporta un fichero,
          exporta uno por clase de dato, y quien trae solo el de las credenciales
          se queda con la mitad de su gestor dentro y la otra mitad fuera sin que
          nada se lo diga. Pasó de verdad: 65 entradas importadas y más cosas en
          Dashlane, sin ningún error por medio. */}
      <p className="nota">
        Algunos gestores exportan <strong>varios ficheros</strong>: Dashlane saca uno por clase de
        cosa —credenciales, notas seguras, tarjetas y documentos—. Tráelos uno a uno con este mismo
        botón, y cada uno se reconoce por sus columnas.
      </p>

      <div className="botones">
        <button onClick={importar} disabled={trabajando}>
          {trabajando ? "Leyendo…" : `Elegir la exportación de ${deDonde}…`}
        </button>
        <button onClick={() => (seguro ? exportar() : setSeguro(true))}>
          {seguro ? "Sí, exportar en claro" : "Exportar la bóveda…"}
        </button>
      </div>

      {seguro && (
        <p className="aviso">
          Lo que sale es una lista de contraseñas <strong>sin cifrar</strong>. Piensa dónde la
          dejas y bórrala cuando termines.
        </p>
      )}

      {salida && <p className="exito seleccionable">Exportada en {salida}</p>}

      {resumen && (
        <>
          <p className="exito">
            {resumen.metidas === 1 ? "Una entrada" : `${resumen.metidas} entradas`} de{" "}
            {resumen.deDonde}
            {resumen.duplicadas > 0 &&
              ` · ${resumen.duplicadas} ya estaban y se han dejado como estaban`}
          </p>
          {!borrado && (
            <>
              {/* Se ofrece con insistencia porque ese fichero es una lista de
                  contraseñas en claro en el disco, y quien lo exportó de su gestor
                  rara vez se acuerda de quitarlo. */}
              <p className="aviso seleccionable">
                {resumen.fichero} sigue en el disco con las contraseñas en claro.
              </p>
              <div className="botones">
                <button
                  className="principal"
                  onClick={async () => {
                    try {
                      await esfinge.borrarElCSVImportado(resumen.fichero);
                      setBorrado(true);
                    } catch (e) {
                      setError(mensaje(e));
                    }
                  }}
                >
                  Borrarlo
                </button>
              </div>
              <p className="nota">
                En un disco de estado sólido, borrar no lo elimina de verdad: deja de estar en el
                índice, pero puede seguir en el disco un tiempo.
              </p>
            </>
          )}
          {borrado && <p className="exito">Borrado</p>}
        </>
      )}

      {error && <p className="error">{error}</p>}
    </div>
  );
}

// -------------------------------------------------------------------- llaves

function Seguridad({ alRotar }: { alRotar: (clave: string) => void }) {
  const [abierto, setAbierto] = useState(false);
  const [vieja, setVieja] = useState("");
  const [nueva, setNueva] = useState("");
  const [repetida, setRepetida] = useState("");
  const [dicho, setDicho] = useState("");
  const [error, setError] = useState("");
  const [trabajando, setTrabajando] = useState(false);

  const listo = vieja !== "" && nueva.length >= MINIMO_MAESTRA && nueva === repetida;

  async function cambiar() {
    setTrabajando(true);
    setError("");
    setDicho("");
    try {
      await esfinge.cambiarMaestraDeBoveda(vieja, nueva);
      setVieja("");
      setNueva("");
      setRepetida("");
      setDicho("Contraseña maestra cambiada. La clave de recuperación sigue valiendo.");
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  if (!abierto) {
    return (
      <div className="botones">
        <button className="discreto" onClick={() => setAbierto(true)}>
          Contraseña maestra y clave de recuperación
        </button>
      </div>
    );
  }

  return (
    <div className="grupo">
      <div className="fila">
        <label style={{ marginBottom: 0 }}>Contraseña maestra</label>
        <button className="discreto" onClick={() => setAbierto(false)}>
          Cerrar
        </button>
      </div>

      <div>
        <label htmlFor="boveda-vieja">La de ahora</label>
        <input
          id="boveda-vieja"
          type="password"
          autoComplete="off"
          value={vieja}
          onChange={(e) => setVieja(e.target.value)}
        />
        {/* Pedirla aunque la bóveda esté abierta no es un trámite: una bóveda
            abierta encima de una mesa es una bóveda a la que cualquiera que pase
            puede cambiarle la contraseña y dejar fuera a su dueño. */}
        <p className="nota">Se pide aunque la bóveda esté abierta.</p>
      </div>

      <CampoClave id="boveda-nueva" etiqueta="La nueva" valor={nueva} alCambiar={setNueva} />

      <div>
        <label htmlFor="boveda-nueva-2">Repítela</label>
        <input
          id="boveda-nueva-2"
          type="password"
          autoComplete="off"
          value={repetida}
          onChange={(e) => setRepetida(e.target.value)}
        />
      </div>

      <div className="botones">
        <button className="principal" onClick={cambiar} disabled={!listo || trabajando}>
          {trabajando ? "Cambiando…" : "Cambiar la contraseña"}
        </button>
      </div>

      {dicho && <p className="exito">{dicho}</p>}
      {error && <p className="error">{error}</p>}

      <div>
        <label>Clave de recuperación</label>
        <p className="nota">
          Generar otra deja la anterior inservible. Hazlo si crees que alguien más la tiene.
        </p>
      </div>
      <div className="botones">
        <button
          onClick={async () => {
            try {
              alRotar(await esfinge.rotarRecuperacionDeBoveda());
            } catch (e) {
              setError(mensaje(e));
            }
          }}
        >
          Generar otra…
        </button>
      </div>
    </div>
  );
}

function fecha(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(undefined, {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}

function mensaje(e: unknown): string {
  if (e instanceof Error) return e.message;
  return String(e);
}
