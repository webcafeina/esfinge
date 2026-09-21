import { useEffect, useState } from "react";
import { Ceremonia } from "./boveda";
import { CampoClave, Firma, Marca, Segmentado } from "./componentes";
import { alCambiarLaSincro, esfinge, type EstadoCuenta, type EstadoSincro } from "./puente";

/**
 * La cuenta, en la ventana (ADR 0035).
 *
 * Tres piezas:
 *
 *   - **La bienvenida**, la primera vez que se abre Esfinge sin nada: elegir entre
 *     trabajar en este ordenador o con cuenta, con lo bueno y lo malo de cada una
 *     dicho al lado. Se puede cambiar de idea en Ajustes.
 *   - **El asistente**, para crear la cuenta o entrar en una. Va a ventana entera,
 *     con el mismo marco que la bienvenida, porque es un momento de una vez y no
 *     una pantalla de trabajo.
 *   - **Lo que se enseña de la sincronización**: una línea en la bóveda y un grupo
 *     en Ajustes.
 *
 * **La marca sale aquí y no en las pantallas de trabajo** (ADR 0021): la
 * bienvenida y el asistente son la puerta, y ahí la esfinge grande tiene sitio.
 * Los colores son todos parejas ya medidas en `contraste_test.go`.
 */

// ------------------------------------------------------------------ la bienvenida

export function Bienvenida({
  version,
  alElegirLocal,
  alCrearCuenta,
  alEntrar,
}: {
  version: string;
  alElegirLocal: () => void;
  alCrearCuenta: () => void;
  alEntrar: () => void;
}) {
  const [error, setError] = useState("");

  async function local() {
    try {
      await esfinge.elegirModoLocal();
      alElegirLocal();
    } catch (e) {
      setError(mensaje(e));
    }
  }

  return (
    <Portada version={version}>
      <header className="portada-cabeza">
        <Marca lado={88} clase="portada-marca" />
        <h1>Te damos la bienvenida a Esfinge</h1>
        <p className="entradilla">¿Dónde quieres guardar tus contraseñas?</p>
      </header>

      <div className="eleccion">
        <section className="opcion" aria-labelledby="opcion-local">
          <h2 id="opcion-local">En este ordenador</h2>
          <p className="opcion-lema">Todo se queda aquí, sin cuentas ni servidores.</p>
          <ul className="pros-contras">
            <li className="pro">Nada sale de este ordenador</li>
            <li className="pro">Funciona sin conexión, siempre</li>
            <li className="contra">Solo la tienes aquí</li>
            <li className="contra">Si pierdes el equipo, te queda la copia que hayas hecho tú</li>
          </ul>
          <div className="botones">
            <button className="principal" onClick={local}>
              Usar en este ordenador
            </button>
          </div>
        </section>

        <section className="opcion" aria-labelledby="opcion-cuenta">
          <h2 id="opcion-cuenta">Con cuenta</h2>
          <p className="opcion-lema">La misma bóveda en todos tus equipos.</p>
          <ul className="pros-contras">
            <li className="pro">Se cifra antes de salir: el servidor no puede leerla</li>
            <li className="pro">Cambias algo en un equipo y aparece en los demás</li>
            <li className="contra">Tu bóveda cifrada se guarda también en un servidor de Webcafeína, en la UE</li>
            <li className="contra">Pide una contraseña fuerte y un código por correo en cada equipo nuevo</li>
          </ul>
          <div className="botones">
            <button className="principal" onClick={alCrearCuenta}>
              Crear una cuenta
            </button>
            <button onClick={alEntrar}>Ya tengo cuenta</button>
          </div>
          <p className="nota">Por ahora, las cuentas son por invitación.</p>
        </section>
      </div>

      {error && <p className="error">{error}</p>}
      <p className="nota portada-pie">Puedes cambiar de idea cuando quieras, en Ajustes.</p>
    </Portada>
  );
}

/**
 * Portada es el marco de la bienvenida y del asistente: a ventana entera, sin
 * barra lateral y con la firma de la casa al pie.
 *
 * **Se puede arrastrar la ventana desde ella**: no hay barra de título, y sin
 * declararla zona de arrastre la ventana se quedaría clavada (CLAUDE.md).
 */
function Portada({ version, children }: { version: string; children: React.ReactNode }) {
  return (
    <div className="portada">
      <div className="portada-cuerpo">{children}</div>
      <Firma version={version} />
    </div>
  );
}

// ------------------------------------------------------------------ el asistente

export type TipoAsistente = { que: "crear" } | { que: "entrar"; correo?: string; deNuevo?: boolean };

/**
 * El asistente para crear la cuenta o entrar en ella. Al terminar, `alTerminar`:
 * la bóveda de la cuenta queda abierta en este equipo y sincronizándose.
 */
export function Asistente({
  tipo,
  version,
  hayBoveda,
  alTerminar,
  alVolver,
}: {
  tipo: TipoAsistente;
  version: string;
  /** Si en este equipo ya hay una bóveda: entonces la cuenta se crea con ella. */
  hayBoveda: boolean;
  alTerminar: () => void;
  alVolver: () => void;
}) {
  return (
    <Portada version={version}>
      <div className="asistente">
        {tipo.que === "crear" ? (
          <CrearCuenta hayBoveda={hayBoveda} alTerminar={alTerminar} alVolver={alVolver} />
        ) : (
          <EntrarEnCuenta
            correoFijo={tipo.correo}
            deNuevo={tipo.deNuevo ?? false}
            alTerminar={alTerminar}
            alVolver={alVolver}
          />
        )}
      </div>
    </Portada>
  );
}

function Cabecera({ titulo, texto }: { titulo: string; texto: string }) {
  return (
    <header className="asistente-cabeza">
      <Marca lado={44} clase="portada-marca" />
      <h1>{titulo}</h1>
      <p className="entradilla">{texto}</p>
    </header>
  );
}

function CrearCuenta({
  hayBoveda,
  alTerminar,
  alVolver,
}: {
  hayBoveda: boolean;
  alTerminar: () => void;
  alVolver: () => void;
}) {
  const [paso, setPaso] = useState<"correo" | "codigo" | "clave">("correo");
  const [correo, setCorreo] = useState("");
  const [codigo, setCodigo] = useState("");
  const [maestra, setMaestra] = useState("");
  const [repetida, setRepetida] = useState("");
  // La que sustituye a la de la bóveda si ésa no llega a «Buena».
  const [nueva, setNueva] = useState("");
  const [nuevaRepetida, setNuevaRepetida] = useState("");
  const [recuperacion, setRecuperacion] = useState("");
  const nivel = usaNivel(maestra);
  const nivelNueva = usaNivel(nueva);
  const [trabajando, setTrabajando] = useState(false);
  const [error, setError] = useState("");

  async function hacer(f: () => Promise<void>) {
    setTrabajando(true);
    setError("");
    try {
      await f();
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  const pedirCodigo = () =>
    hacer(async () => {
      await esfinge.empezarRegistro(correo);
      setPaso("codigo");
    });

  const crear = () =>
    hacer(async () => {
      const rec = await esfinge.terminarRegistro(correo, codigo, maestra, floja ? nueva : "");
      if (rec) setRecuperacion(rec);
      else alTerminar();
    });

  if (recuperacion) {
    return <Ceremonia clave={recuperacion} nueva alSeguir={alTerminar} />;
  }

  // Con la bóveda de siempre, si su contraseña no llega a «Buena» se pide una nueva
  // aquí mismo, y se cambia a la vez que se crea la cuenta. Sin bóveda, la que se
  // escribe tiene que llegar ya.
  const floja = hayBoveda && maestra !== "" && nivel !== null && nivel < 3;
  const distintas = !hayBoveda && repetida !== "" && repetida !== maestra;
  const nuevasDistintas = floja && nuevaRepetida !== "" && nuevaRepetida !== nueva;
  const listo =
    codigo.trim().length === 6 &&
    maestra !== "" &&
    nivel !== null &&
    (hayBoveda
      ? !floja || (nueva !== "" && nuevaRepetida === nueva && (nivelNueva ?? 0) >= 3)
      : repetida === maestra && nivel >= 3);

  return (
    <>
      <Cabecera
        titulo="Crear una cuenta"
        texto={
          hayBoveda
            ? "Tu bóveda de siempre pasa a estar en la cuenta, y la podrás abrir en tus otros equipos."
            : "Crea tu bóveda en la cuenta, y la podrás abrir en tus otros equipos."
        }
      />

      {paso === "correo" ? (
        <div className="grupo">
          <div>
            <label htmlFor="cuenta-correo">Tu correo</label>
            <input
              id="cuenta-correo"
              type="email"
              autoComplete="email"
              value={correo}
              onChange={(e) => setCorreo(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && correo && pedirCodigo()}
            />
            <p className="nota">Te mandaremos un código para comprobar que es tuyo.</p>
          </div>
        </div>
      ) : (
        <div className="grupo">
          <div>
            <label htmlFor="cuenta-codigo">Código que te ha llegado a {correo}</label>
            <input
              id="cuenta-codigo"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={7}
              value={codigo}
              onChange={(e) => setCodigo(e.target.value.replace(/[^\d]/g, ""))}
            />
            <p className="nota">Caduca en diez minutos. Si no lo ves, mira en el correo no deseado.</p>
          </div>
          <CampoClave
            id="cuenta-maestra"
            etiqueta={hayBoveda ? "La contraseña maestra de tu bóveda" : "Contraseña maestra"}
            valor={maestra}
            alCambiar={setMaestra}
            medir={!hayBoveda}
            alEnviar={() => hayBoveda && listo && crear()}
          />
          {floja && (
            <>
              <p className="aviso">
                Tu contraseña de ahora no llega a «Buena», y con cuenta hace falta. Pon una nueva: a
                partir de ahora abrirá tu bóveda y tu cuenta. La de recuperación sigue valiendo.
              </p>
              <CampoClave
                id="cuenta-maestra-nueva"
                etiqueta="Contraseña maestra nueva"
                valor={nueva}
                alCambiar={setNueva}
              />
              <div>
                <label htmlFor="cuenta-maestra-nueva-2">Repítela</label>
                <input
                  id="cuenta-maestra-nueva-2"
                  type="password"
                  autoComplete="off"
                  value={nuevaRepetida}
                  onChange={(e) => setNuevaRepetida(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && listo && crear()}
                />
                {nuevasDistintas && <p className="error">Las dos no coinciden</p>}
              </div>
            </>
          )}
          {!hayBoveda && (
            <div>
              <label htmlFor="cuenta-maestra-2">Repítela</label>
              <input
                id="cuenta-maestra-2"
                type="password"
                autoComplete="off"
                value={repetida}
                onChange={(e) => setRepetida(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && listo && crear()}
              />
              {distintas && <p className="error">Las dos no coinciden</p>}
            </div>
          )}
        </div>
      )}

      {paso !== "correo" && (
        <p className="aviso">
          Con cuenta, la contraseña maestra tiene que ser <strong>al menos «Buena»</strong>: tu
          bóveda cifrada vive también en el servidor, y es lo único que la protege. Nadie —tampoco
          nosotros— puede recuperarla.
        </p>
      )}

      {error && <p className="error">{error}</p>}

      <div className="botones">
        {paso === "correo" ? (
          <button className="principal" onClick={pedirCodigo} disabled={!correo || trabajando}>
            {trabajando ? "Mandando…" : "Mandarme el código"}
          </button>
        ) : (
          <button className="principal" onClick={crear} disabled={!listo || trabajando}>
            {trabajando ? "Creando…" : "Crear la cuenta"}
          </button>
        )}
        <button className="discreto" onClick={paso === "correo" ? alVolver : () => setPaso("correo")}>
          Volver
        </button>
      </div>
    </>
  );
}

function EntrarEnCuenta({
  correoFijo,
  deNuevo,
  alTerminar,
  alVolver,
}: {
  correoFijo?: string;
  deNuevo: boolean;
  alTerminar: () => void;
  alVolver: () => void;
}) {
  const [paso, setPaso] = useState<"datos" | "codigo" | "otra" | "hecho">("datos");
  const [correo, setCorreo] = useState(correoFijo ?? "");
  const [maestra, setMaestra] = useState("");
  const [codigo, setCodigo] = useState("");
  const [juntar, setJuntar] = useState(true);
  const [maestraLocal, setMaestraLocal] = useState("");
  const [apartada, setApartada] = useState("");
  const [trabajando, setTrabajando] = useState(false);
  const [error, setError] = useState("");

  async function hacer(f: () => Promise<void>) {
    setTrabajando(true);
    setError("");
    try {
      await f();
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  function seguirCon(r: { necesitaCodigo: boolean; hayOtraBoveda: boolean; listo: boolean; apartada?: string }) {
    if (r.necesitaCodigo) setPaso("codigo");
    else if (r.hayOtraBoveda) setPaso("otra");
    else if (r.apartada) {
      setApartada(r.apartada);
      setPaso("hecho");
    } else alTerminar();
  }

  const entrar = () => hacer(async () => seguirCon(await esfinge.entrarEnCuenta(correo, maestra)));
  const confirmar = () => hacer(async () => seguirCon(await esfinge.confirmarEntrada(codigo)));
  const resolver = () =>
    hacer(async () => seguirCon(await esfinge.resolverOtraBoveda(juntar, juntar ? maestraLocal : "")));

  if (paso === "hecho") {
    return (
      <>
        <Cabecera titulo="Ya estás en la cuenta" texto="La bóveda de la cuenta está en este equipo." />
        <p className="nota seleccionable">
          La bóveda que había aquí no se ha borrado: se ha guardado aparte, en {apartada}.
        </p>
        <div className="botones">
          <button className="principal" onClick={alTerminar}>
            Continuar
          </button>
        </div>
      </>
    );
  }

  if (paso === "otra") {
    return (
      <>
        <Cabecera
          titulo="En este ordenador ya hay una bóveda"
          texto="Es distinta de la de tu cuenta. ¿Qué hacemos con ella?"
        />
        <div className="grupo">
          <Segmentado<"juntar" | "apartar">
            valor={juntar ? "juntar" : "apartar"}
            alCambiar={(v) => setJuntar(v === "juntar")}
            opciones={[
              { valor: "juntar", etiqueta: "Juntarlas" },
              { valor: "apartar", etiqueta: "Solo la de la cuenta" },
            ]}
          />
          {juntar ? (
            <>
              <p className="nota">
                Lo que tiene la de este ordenador pasa a la cuenta, y lo verás en todos tus equipos.
              </p>
              <CampoClave
                id="cuenta-maestra-local"
                etiqueta="La contraseña de la bóveda de este ordenador"
                medir={false}
                valor={maestraLocal}
                alCambiar={setMaestraLocal}
              />
              <p className="nota">Déjala vacía si es la misma que la de la cuenta.</p>
            </>
          ) : (
            <p className="nota">Te quedas con la de la cuenta tal como está.</p>
          )}
        </div>
        <p className="nota">En los dos casos, la de este ordenador no se borra: se guarda aparte.</p>
        {error && <p className="error">{error}</p>}
        <div className="botones">
          <button className="principal" onClick={resolver} disabled={trabajando}>
            {trabajando ? "Un momento…" : "Seguir"}
          </button>
        </div>
      </>
    );
  }

  return (
    <>
      <Cabecera
        titulo={deNuevo ? "Vuelve a entrar en la cuenta" : "Entrar en tu cuenta"}
        texto={
          deNuevo
            ? "La sesión ha caducado. Escribe tu contraseña para seguir sincronizando."
            : "Tu bóveda de la cuenta se abrirá en este equipo."
        }
      />

      {paso === "datos" ? (
        <div className="grupo">
          <div>
            <label htmlFor="entrar-correo">Tu correo</label>
            <input
              id="entrar-correo"
              type="email"
              autoComplete="email"
              value={correo}
              readOnly={!!correoFijo}
              onChange={(e) => setCorreo(e.target.value)}
            />
          </div>
          <CampoClave
            id="entrar-maestra"
            etiqueta="Contraseña maestra"
            medir={false}
            valor={maestra}
            alCambiar={setMaestra}
            alEnviar={() => correo && maestra && entrar()}
          />
        </div>
      ) : (
        <div className="grupo">
          <div>
            <label htmlFor="entrar-codigo">Código que te ha llegado a {correo}</label>
            <input
              id="entrar-codigo"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={7}
              value={codigo}
              onChange={(e) => setCodigo(e.target.value.replace(/[^\d]/g, ""))}
              onKeyDown={(e) => e.key === "Enter" && codigo.length === 6 && confirmar()}
            />
            <p className="nota">
              Solo la primera vez en cada equipo. Caduca en diez minutos; si no lo ves, mira en el
              correo no deseado.
            </p>
          </div>
        </div>
      )}

      {error && <p className="error">{error}</p>}

      <div className="botones">
        {paso === "datos" ? (
          <button className="principal" onClick={entrar} disabled={!correo || !maestra || trabajando}>
            {trabajando ? "Entrando…" : "Entrar"}
          </button>
        ) : (
          <button className="principal" onClick={confirmar} disabled={codigo.length !== 6 || trabajando}>
            {trabajando ? "Comprobando…" : "Confirmar"}
          </button>
        )}
        <button className="discreto" onClick={paso === "datos" ? alVolver : () => setPaso("datos")}>
          Volver
        </button>
      </div>
    </>
  );
}

/**
 * usaNivel pregunta a Go cuánto vale una contraseña (0 muy débil … 4 excelente),
 * con el mismo medidor que el resto de la ventana. Null mientras no contesta.
 */
function usaNivel(clave: string): number | null {
  const [nivel, setNivel] = useState<number | null>(null);
  useEffect(() => {
    if (!clave) {
      setNivel(null);
      return;
    }
    let vigente = true;
    const t = setTimeout(() => {
      esfinge
        .evaluarClave(clave)
        .then((f) => vigente && setNivel(f.nivel))
        .catch(() => {});
    }, 150);
    return () => {
      vigente = false;
      clearTimeout(t);
    };
  }, [clave]);
  return nivel;
}

// ------------------------------------------------------------------ la sincronización a la vista

/** Lo que dice cada estado, en una frase. */
export function frase(e: EstadoSincro): string {
  switch (e.estado) {
    case "al-dia":
      return e.ultima ? `Sincronizada ${haceCuanto(e.ultima)}` : "Sincronizada";
    case "sincronizando":
      return "Sincronizando…";
    case "sin-conexion":
      return "Sin conexión: se sube al volver";
    case "sin-red":
      return "Sin red: Esfinge tiene la red apagada";
    case "hay-que-entrar":
      return "Hay que volver a entrar en la cuenta";
    case "muchos-borrados":
      return "Parada: los cambios de otro equipo borrarían media bóveda";
    case "error":
      return e.mensaje ?? "La sincronización ha fallado";
    default:
      return "Sin sincronizar";
  }
}

function haceCuanto(iso: string): string {
  const segundos = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
  if (segundos < 60) return "hace un momento";
  const minutos = Math.round(segundos / 60);
  if (minutos < 60) return minutos === 1 ? "hace un minuto" : `hace ${minutos} minutos`;
  const horas = Math.round(minutos / 60);
  return horas === 1 ? "hace una hora" : `hace ${horas} horas`;
}

/** usaCuenta trae el estado de la cuenta y lo tiene al día con los avisos de Go. */
export function usaCuenta(activo = true): [EstadoCuenta | null, () => void] {
  const [cuenta, setCuenta] = useState<EstadoCuenta | null>(null);
  const refrescar = () => {
    esfinge
      .estadoDeCuenta()
      .then(setCuenta)
      .catch(() => {});
  };
  useEffect(() => {
    if (activo) refrescar();
  }, [activo]);
  useEffect(() => alCambiarLaSincro((sincro) => setCuenta((c) => (c ? { ...c, sincro } : c))), []);
  // La frase «hace N minutos» envejece sola.
  const [, setLatido] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setLatido((n) => n + 1), 30_000);
    return () => clearInterval(t);
  }, []);
  return [cuenta, refrescar];
}

/** La línea de la sincronización, en la bóveda. Nada si no hay cuenta. */
export function LineaSincro({ alVolverAEntrar }: { alVolverAEntrar: (correo: string) => void }) {
  const [cuenta] = usaCuenta();
  if (!cuenta || cuenta.modo !== "cuenta") return null;
  const e = cuenta.sincro;
  const mal = e.estado === "hay-que-entrar" || e.estado === "error" || e.estado === "muchos-borrados";
  return (
    <p className={mal ? "linea-sincro aviso" : "linea-sincro nota"} role="status">
      {frase(e)}
      {e.estado === "hay-que-entrar" && (
        <>
          {" "}
          <button className="discreto" onClick={() => alVolverAEntrar(cuenta.correo ?? "")}>
            Volver a entrar
          </button>
        </>
      )}
    </p>
  );
}

/** El grupo de la cuenta en Ajustes. */
export function GrupoCuenta({
  alCrearCuenta,
  alEntrar,
}: {
  alCrearCuenta: () => void;
  alEntrar: (correo?: string, deNuevo?: boolean) => void;
}) {
  const [cuenta, refrescar] = usaCuenta();
  const [error, setError] = useState("");
  const [saliendo, setSaliendo] = useState(false);
  const [maestra, setMaestra] = useState("");
  if (!cuenta) return null;

  async function salir() {
    setError("");
    try {
      await esfinge.salirDeCuenta(maestra);
      setSaliendo(false);
      setMaestra("");
      refrescar();
    } catch (e) {
      setError(mensaje(e));
    }
  }

  if (cuenta.modo !== "cuenta") {
    return (
      <div className="grupo">
        <h2>Cuenta y sincronización</h2>
        <p className="nota">
          Tu bóveda está solo en este ordenador. Con una cuenta, la tendrías también en tus otros
          equipos, cifrada antes de salir de aquí.
        </p>
        <div className="botones">
          <button onClick={alCrearCuenta}>Crear una cuenta</button>
          <button onClick={() => alEntrar()}>Entrar en una cuenta</button>
        </div>
      </div>
    );
  }

  return (
    <div className="grupo">
      <h2>Cuenta y sincronización</h2>
      <dl className="datos-cuenta">
        <dt>Cuenta</dt>
        <dd className="seleccionable">{cuenta.correo}</dd>
        <dt>Este equipo</dt>
        <dd>{cuenta.equipo}</dd>
        <dt>Sincronización</dt>
        <dd role="status">{frase(cuenta.sincro)}</dd>
      </dl>
      {error && <p className="error">{error}</p>}
      <div className="botones">
        {cuenta.sincro.estado === "hay-que-entrar" ? (
          <button onClick={() => alEntrar(cuenta.correo, true)}>Volver a entrar</button>
        ) : (
          <button
            onClick={() => {
              setError("");
              esfinge.sincronizarAhora().catch((e) => setError(mensaje(e)));
            }}
            disabled={cuenta.sincro.estado === "apagada"}
          >
            Sincronizar ahora
          </button>
        )}
      </div>
      {cuenta.sincro.estado === "apagada" && (
        <p className="nota">Se sincroniza con la bóveda abierta.</p>
      )}
      {saliendo ? (
        <>
          <p className="nota">
            Tu bóveda se queda en este ordenador tal como está y deja de sincronizarse. La cuenta
            sigue en el servidor, y tus otros equipos no cambian.
          </p>
          <CampoClave
            id="salir-maestra"
            etiqueta="Contraseña maestra"
            medir={false}
            valor={maestra}
            alCambiar={setMaestra}
            alEnviar={() => maestra && salir()}
          />
          <div className="botones">
            <button onClick={salir} disabled={!maestra}>
              Dejar la cuenta en este equipo
            </button>
            <button className="discreto" onClick={() => setSaliendo(false)}>
              Cancelar
            </button>
          </div>
        </>
      ) : (
        <div className="botones">
          <button className="discreto" onClick={() => setSaliendo(true)}>
            Dejar la cuenta en este equipo…
          </button>
        </div>
      )}
      <p className="nota">
        Tu bóveda sube cifrada a {new URL(cuenta.servidor).host}, en la UE: el servidor no puede
        leerla.
      </p>
    </div>
  );
}

function mensaje(e: unknown): string {
  if (e instanceof Error) return e.message;
  return String(e);
}
