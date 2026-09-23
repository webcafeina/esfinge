import { useEffect, useState } from "react";
import { Ceremonia } from "./boveda";
import { CampoClave, Firma, Marca, Segmentado } from "./componentes";
import {
  alCambiarLaSincro,
  esfinge,
  type EquipoDeCuenta,
  type EstadoCuenta,
  type EstadoSincro,
} from "./puente";

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

export type TipoAsistente =
  | { que: "crear" }
  | { que: "entrar"; correo?: string; deNuevo?: boolean }
  | { que: "recuperar"; correo?: string };

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
  const [recuperando, setRecuperando] = useState(false);
  return (
    <Portada version={version}>
      <div className="asistente">
        {tipo.que === "crear" ? (
          <CrearCuenta hayBoveda={hayBoveda} alTerminar={alTerminar} alVolver={alVolver} />
        ) : tipo.que === "recuperar" || recuperando ? (
          <RecuperarCuenta
            correoInicial={tipo.correo ?? ""}
            alTerminar={alTerminar}
            alVolver={tipo.que === "recuperar" ? alVolver : () => setRecuperando(false)}
          />
        ) : (
          <EntrarEnCuenta
            correoFijo={tipo.correo}
            deNuevo={tipo.deNuevo ?? false}
            alTerminar={alTerminar}
            alVolver={alVolver}
            alOlvidarla={() => setRecuperando(true)}
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

      {/* **Las condiciones y la política, antes de crear la cuenta y no después.**
          Crear una cuenta es contratar un servicio a distancia, y el artículo 27 de
          la LSSI pide que se puedan leer antes; hasta la 2.25.7 no se enlazaban
          desde ninguna parte de la ventana (revisión de los textos, 2026-09-23). */}
      <p className="nota">
        Al crear la cuenta aceptas las{" "}
        <a href="https://webcafeina.github.io/esfinge/condiciones.html" target="_blank" rel="noreferrer">
          condiciones de uso
        </a>{" "}
        y la{" "}
        <a href="https://webcafeina.github.io/esfinge/privacidad.html" target="_blank" rel="noreferrer">
          política de privacidad
        </a>
        .
      </p>
    </>
  );
}

function EntrarEnCuenta({
  correoFijo,
  deNuevo,
  alTerminar,
  alVolver,
  alOlvidarla,
}: {
  correoFijo?: string;
  deNuevo: boolean;
  alTerminar: () => void;
  alVolver: () => void;
  alOlvidarla: () => void;
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
        ) : null}
        {paso === "datos" ? (
          <button className="discreto" onClick={alOlvidarla}>
            ¿Has olvidado la contraseña?
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
 * Recuperar la cuenta **sin ningún equipo a mano** (A3): código por correo, la clave
 * de recuperación y una contraseña nueva. Sin la clave de recuperación no hay forma:
 * nosotros no podemos abrir la bóveda.
 */
function RecuperarCuenta({
  correoInicial,
  alTerminar,
  alVolver,
}: {
  correoInicial: string;
  alTerminar: () => void;
  alVolver: () => void;
}) {
  const [paso, setPaso] = useState<"correo" | "datos" | "hecho">("correo");
  const [correo, setCorreo] = useState(correoInicial);
  const [codigo, setCodigo] = useState("");
  const [clave, setClave] = useState("");
  const [nueva, setNueva] = useState("");
  const [repetida, setRepetida] = useState("");
  const [apartada, setApartada] = useState("");
  const [trabajando, setTrabajando] = useState(false);
  const [error, setError] = useState("");
  const nivel = usaNivel(nueva);

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

  const pedir = () =>
    hacer(async () => {
      await esfinge.empezarRecuperacion(correo);
      setPaso("datos");
    });
  const recuperar = () =>
    hacer(async () => {
      const r = await esfinge.terminarRecuperacion(correo, codigo, clave, nueva);
      if (r.apartada) {
        setApartada(r.apartada);
        setPaso("hecho");
      } else if (r.hayOtraBoveda) {
        setError("En este ordenador hay otra bóveda. Entra en la cuenta con la contraseña nueva para decidir qué hacer con ella.");
      } else alTerminar();
    });

  const distintas = repetida !== "" && repetida !== nueva;
  const listo =
    codigo.length === 6 && clave.trim() !== "" && nueva !== "" && repetida === nueva && (nivel ?? 0) >= 3;

  if (paso === "hecho") {
    return (
      <>
        <Cabecera titulo="Cuenta recuperada" texto="Ya puedes entrar con tu contraseña nueva en todos tus equipos." />
        <p className="nota seleccionable">
          La bóveda que había en este ordenador no se ha borrado: se ha guardado aparte, en {apartada}.
        </p>
        <div className="botones">
          <button className="principal" onClick={alTerminar}>
            Continuar
          </button>
        </div>
      </>
    );
  }

  return (
    <>
      <Cabecera
        titulo="Recuperar tu cuenta"
        texto="Con el código que te mandaremos y tu clave de recuperación, pondrás una contraseña nueva."
      />
      {paso === "correo" ? (
        <div className="grupo">
          <div>
            <label htmlFor="recuperar-correo">Tu correo</label>
            <input
              id="recuperar-correo"
              type="email"
              autoComplete="email"
              value={correo}
              onChange={(e) => setCorreo(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && correo && pedir()}
            />
          </div>
        </div>
      ) : (
        <div className="grupo">
          <div>
            <label htmlFor="recuperar-codigo">Código que te ha llegado a {correo}</label>
            <input
              id="recuperar-codigo"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={7}
              value={codigo}
              onChange={(e) => setCodigo(e.target.value.replace(/[^\d]/g, ""))}
            />
          </div>
          <div>
            <label htmlFor="recuperar-clave">Clave de recuperación</label>
            <input
              id="recuperar-clave"
              type="text"
              autoComplete="off"
              spellCheck={false}
              placeholder="ESF-…"
              value={clave}
              onChange={(e) => setClave(e.target.value)}
            />
            <p className="nota">La que apuntaste al crear la bóveda. Da igual en mayúsculas o minúsculas.</p>
          </div>
          <CampoClave id="recuperar-nueva" etiqueta="Contraseña maestra nueva" valor={nueva} alCambiar={setNueva} />
          <div>
            <label htmlFor="recuperar-nueva-2">Repítela</label>
            <input
              id="recuperar-nueva-2"
              type="password"
              autoComplete="off"
              value={repetida}
              onChange={(e) => setRepetida(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && listo && recuperar()}
            />
            {distintas && <p className="error">Las dos no coinciden</p>}
          </div>
        </div>
      )}
      {paso === "datos" && (
        <p className="aviso">
          La contraseña nueva tiene que ser <strong>al menos «Buena»</strong>. Tus otros equipos te la
          pedirán al volver a entrar, y lo que tuvieran sin subir no se pierde.
        </p>
      )}
      {error && <p className="error">{error}</p>}
      <div className="botones">
        {paso === "correo" ? (
          <button className="principal" onClick={pedir} disabled={!correo || trabajando}>
            {trabajando ? "Mandando…" : "Mandarme el código"}
          </button>
        ) : (
          <button className="principal" onClick={recuperar} disabled={!listo || trabajando}>
            {trabajando ? "Recuperando…" : "Recuperar la cuenta"}
          </button>
        )}
        <button className="discreto" onClick={paso === "correo" ? alVolver : () => setPaso("correo")}>
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

/**
 * usaSincroAlVolver pide una pasada cuando se vuelve a la ventana de Esfinge, como
 * mucho una cada treinta segundos. Es lo natural: se cambia algo en otro equipo, se
 * vuelve a éste, y ya está. **No cuenta como actividad**: SincronizarAhora no toca
 * el reloj del bloqueo.
 */
export function usaSincroAlVolver(conCuenta: boolean) {
  useEffect(() => {
    if (!conCuenta) return;
    let ultima = 0;
    const pedir = () => {
      if (document.visibilityState === "hidden") return;
      const ahora = Date.now();
      if (ahora - ultima < 30_000) return;
      ultima = ahora;
      // Con la bóveda cerrada contesta que no hay nada que sincronizar: da igual.
      esfinge.sincronizarAhora().catch(() => {});
    };
    window.addEventListener("focus", pedir);
    document.addEventListener("visibilitychange", pedir);
    return () => {
      window.removeEventListener("focus", pedir);
      document.removeEventListener("visibilitychange", pedir);
    };
  }, [conCuenta]);
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
      {/* **La parada por muchos borrados tiene salida** (revisión del 2026-09-23).
          La comprobación existía desde la ADR 0038 y no había forma de decir que sí:
          la bóveda se quedaba sin sincronizar, sin subir lo de aquí, y lo único que
          se leía era que estaba parada. */}
      {e.estado === "muchos-borrados" && (
        <>
          {" "}
          <button
            className="discreto"
            onClick={() => {
              esfinge.sincronizarAunqueBorre().catch(() => {});
            }}
          >
            Juntarlo igual
          </button>
        </>
      )}
      {/* **Sincronizar a mano**, sin esperar a la pasada de cada minuto: lo pidió el
          cliente al ver que un cambio entre la aplicación y la extensión tardaba
          hasta un minuto (2.25.2). La pasada sola se queda. */}
      {e.estado !== "hay-que-entrar" && e.estado !== "apagada" && <BotonSincronizar sincro={e} />}
    </p>
  );
}

/**
 * Las dos flechas que giran mientras sincroniza. **Lo pidió el cliente así** con la
 * 2.25.2: un botón con la palabra y sin nada que se moviera no decía si había hecho
 * algo. Gira desde el clic hasta que llega el resultado de esa pasada —una fecha
 * igual o posterior al clic, o un fallo—, y al menos medio segundo, para que se vea
 * aunque vaya rapidísimo. Con «reducir movimiento», no gira: se atenúa.
 */
function BotonSincronizar({ sincro }: { sincro: EstadoSincro }) {
  const [desde, setDesde] = useState<number | null>(null);
  useEffect(() => {
    if (desde === null) return;
    const ultima = sincro.ultima ? Date.parse(sincro.ultima) : 0;
    const acabada =
      ultima >= Math.floor(desde / 1000) * 1000 ||
      sincro.estado === "error" ||
      sincro.estado === "sin-conexion" ||
      sincro.estado === "sin-red";
    const resto = Math.max(0, 500 - (Date.now() - desde));
    // Si no llega nada, se para igual: un botón que gira para siempre miente.
    const t = setTimeout(() => setDesde(null), acabada ? resto : 20_000);
    return () => clearTimeout(t);
  }, [desde, sincro]);
  const girando = desde !== null;
  return (
    <button
      type="button"
      className={girando ? "boton-sincro girando" : "boton-sincro"}
      aria-label="Sincronizar ahora"
      title="Sincronizar ahora"
      aria-busy={girando}
      disabled={girando}
      onClick={() => {
        setDesde(Date.now());
        esfinge.sincronizarAhora().catch(() => setDesde(null));
      }}
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M21 12a9 9 0 0 1-15.5 6.2L3 16" />
        <path d="M3 21v-5h5" />
        <path d="M3 12a9 9 0 0 1 15.5-6.2L21 8" />
        <path d="M21 3v5h-5" />
      </svg>
    </button>
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

  // Los equipos, exportar y borrar necesitan la sesión, que solo existe con la
  // bóveda abierta y sin caducar.
  const conSesion = cuenta.sincro.estado !== "apagada" && cuenta.sincro.estado !== "hay-que-entrar";

  return (
    <>
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
    {conSesion && <EquiposDeLaCuenta />}
    {/* **Siempre a la vista con cuenta**, aunque sin sesión no se pueda usar: con la
        bóveda cerrada el grupo no salía, y el cliente no encontraba dónde exportar. */}
    <BorrarLaCuenta alBorrar={refrescar} conSesion={conSesion} />
    </>
  );
}

/** Los equipos con la cuenta abierta, y quitarle la sesión a uno (A3). */
function EquiposDeLaCuenta() {
  const [equipos, setEquipos] = useState<EquipoDeCuenta[] | null>(null);
  const [error, setError] = useState("");
  const traer = () => {
    esfinge
      .dispositivosDeCuenta()
      .then(setEquipos)
      .catch((e) => setError(mensaje(e)));
  };
  useEffect(traer, []);

  async function olvidar(e: EquipoDeCuenta) {
    setError("");
    try {
      await esfinge.olvidarDispositivo(e.id);
      traer();
    } catch (err) {
      setError(mensaje(err));
    }
  }

  return (
    <div className="grupo">
      <h2>Equipos con tu cuenta</h2>
      {equipos === null ? (
        <p className="nota">Cargando…</p>
      ) : (
        <ul className="lista-equipos">
          {equipos.map((e) => (
            <li key={e.id}>
              <span className="equipo">
                <strong>{e.nombre}</strong>
                <span className="equipo-visto">
                  {e.actual ? "Este equipo" : `Usado ${haceCuanto(e.visto)}`}
                </span>
              </span>
              {!e.actual && (
                <button className="discreto" onClick={() => olvidar(e)}>
                  Olvidar
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      <p className="nota">
        Olvidar un equipo —uno perdido, uno que ya no usas— le cierra la sesión: tendrá que volver a
        entrar con tu contraseña y un código. Lo que tenga en su disco se queda allí, cifrado.
      </p>
      {error && <p className="error">{error}</p>}
    </div>
  );
}

/** Exportar lo que hay de la cuenta y borrarla del servidor (A3). */
function BorrarLaCuenta({ alBorrar, conSesion }: { alBorrar: () => void; conSesion: boolean }) {
  const [paso, setPaso] = useState<"nada" | "codigo">("nada");
  const [codigo, setCodigo] = useState("");
  const [maestra, setMaestra] = useState("");
  const [exportado, setExportado] = useState("");
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

  return (
    <div className="grupo peligro">
      <h2>Tus datos en el servidor</h2>
      <p className="nota">
        Puedes llevarte lo que tenemos de tu cuenta —el correo, los equipos, los avisos y tu bóveda tal
        como está allí: cifrada— o borrarla del servidor.
      </p>
      <div className="botones">
        <button
          onClick={() =>
            hacer(async () => {
              const donde = await esfinge.exportarDatosDeCuenta();
              if (donde) setExportado(donde);
            })
          }
          disabled={trabajando || !conSesion}
        >
          Exportar los datos de la cuenta…
        </button>
      </div>
      {!conSesion && (
        <p className="nota">
          Para exportar o borrar la cuenta, abre la bóveda y, si lo pide, vuelve a entrar en la cuenta.
        </p>
      )}
      {exportado && <p className="exito seleccionable">Guardado en {exportado}</p>}

      {paso === "nada" ? (
        <div className="botones">
          <button
            className="discreto"
            onClick={() =>
              hacer(async () => {
                await esfinge.pedirCodigoParaBorrarCuenta();
                setPaso("codigo");
              })
            }
            disabled={trabajando || !conSesion}
          >
            Borrar la cuenta…
          </button>
        </div>
      ) : (
        <>
          <p className="aviso">
            <strong>No tiene vuelta atrás.</strong> Se borran del servidor tu bóveda, sus versiones,
            tus equipos y tu correo. La bóveda de este ordenador —y la de tus otros equipos— se queda
            donde está, y deja de sincronizarse.
          </p>
          <div>
            <label htmlFor="borrar-codigo">Código que te ha llegado al correo</label>
            <input
              id="borrar-codigo"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={7}
              value={codigo}
              onChange={(e) => setCodigo(e.target.value.replace(/[^\d]/g, ""))}
            />
          </div>
          <CampoClave id="borrar-maestra" etiqueta="Contraseña maestra" medir={false} valor={maestra} alCambiar={setMaestra} />
          <div className="botones">
            <button
              onClick={() =>
                hacer(async () => {
                  await esfinge.borrarCuenta(maestra, codigo);
                  setPaso("nada");
                  alBorrar();
                })
              }
              disabled={codigo.length !== 6 || !maestra || trabajando}
            >
              {trabajando ? "Borrando…" : "Borrar la cuenta del servidor"}
            </button>
            <button className="discreto" onClick={() => setPaso("nada")}>
              Cancelar
            </button>
          </div>
        </>
      )}
      {error && <p className="error">{error}</p>}
    </div>
  );
}

function mensaje(e: unknown): string {
  if (e instanceof Error) return e.message;
  return String(e);
}
