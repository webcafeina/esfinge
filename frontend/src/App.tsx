import { useCallback, useEffect, useState } from "react";
import { obedecerEdicion } from "./ordenes";
import {
  alAbrirFichero,
  alDescargar,
  alHaberNovedad,
  alOrdenar,
  alProgresar,
  alSoltarFicheros,
  enWails,
  esfinge,
  type Alfabeto,
  type Apertura,
  type Avance,
  type Entrada,
  type Medida,
  type Novedad,
  type Preferencias,
  type Progreso as TipoProgreso,
  type ResultadoFichero,
  CARACTERES_MINIMO,
  CARACTERES_MAXIMO,
} from "./puente";
import {
  BandaNovedad,
  BarraLateral,
  CampoClave,
  nombreDe,
  PanelResultado,
  Progreso,
  Segmentado,
  ZonaFicheros,
} from "./componentes";

type Tarea = "cifrar" | "descifrar" | "generar" | "historial" | "ajustes";
type Modo = "texto" | "ficheros";

export default function App() {
  const [tarea, setTarea] = useState<Tarea>("cifrar");
  const [version, setVersion] = useState("");
  const [alArrancar, setAlArrancar] = useState<Apertura | null>(null);
  // Cambia cada vez que el sistema manda ficheros, para que la pantalla de
  // descifrar se rehaga con ellos aunque ya estuviera abierta.
  const [tanda, setTanda] = useState(0);

  const [novedad, setNovedad] = useState<Novedad | null>(null);
  const [avance, setAvance] = useState<Avance | undefined>();
  const [falloAlBajar, setFalloAlBajar] = useState("");
  const [instalando, setInstalando] = useState(false);

  useEffect(() => {
    esfinge.version().then(setVersion).catch(() => setVersion("?"));

    // En qué sistema estamos. La estructura de fondo es la misma en los tres
    // —barra lateral y contenido— y lo que cambia son las formas y las
    // densidades, que en cada escritorio son las suyas. Como con el vidrio, va
    // en un atributo de la raíz y el CSS cuelga de él.
    esfinge
      .plataforma()
      .then((s) => {
        document.documentElement.dataset.sistema = s;
      })
      .catch(() => {});

    // El vidrio del sistema. Cuando lo hay, la ventana es transparente y el
    // fondo lo pinta el CSS; cuando no —una versión de Windows sin Mica—, todo
    // se queda como siempre. De ahí que sea un atributo y no un estilo suelto:
    // el CSS entero del vidrio cuelga de él.
    esfinge
      .vidrio()
      .then((hay) => {
        if (hay) document.documentElement.dataset.vidrio = "si";
      })
      .catch(() => {});

    // La comprobación de versiones sale a la red desde Go, en su propia
    // gorrutina, y puede contestar segundos después de abrirse la ventana. Por
    // eso se escucha el evento y además se pregunta: quien llega tarde al primero
    // se entera por lo segundo.
    const dejarDeEscuchar = alHaberNovedad((n) => n.hay && setNovedad(n));

    // Lo que se pide desde el menú del sistema. Las órdenes de edición las
    // resuelve ordenes.ts, que es quien sabe mirar el campo con el foco; aquí
    // quedan las que cambian de pantalla.
    const dejarDeObedecer = alOrdenar((o) => {
      if (obedecerEdicion(o)) return;
      if (o.que.startsWith("ir:")) {
        setTarea(o.que.slice("ir:".length) as Tarea);
        return;
      }
      if (o.que === "actualizar:buscar") {
        setTarea("ajustes");
        esfinge
          .comprobarActualizacion()
          .then((n) => n.hay && setNovedad(n))
          .catch(() => {});
      }
    });
    esfinge
      .novedadPendiente()
      .then((n) => n.hay && setNovedad(n))
      .catch(() => {});

    // Doble clic en un .esf: la aplicación se abre directamente en descifrar,
    // con el fichero puesto. Quien hace ese gesto quiere abrir ese fichero, no
    // buscarlo otra vez desde dentro.
    //
    // El orden importa. Primero se escucha, porque un fichero puede llegar en
    // cualquier momento —doble clic con Esfinge ya abierta—, y solo después se
    // pregunta por los que llegaron antes de que hubiera nadie escuchando. Al
    // revés queda un hueco por el que se pierde el fichero, que es lo que hacía
    // que la ventana se abriera vacía.
    const abrir = (a: Apertura) => {
      if (!a.modo) return;
      setAlArrancar(a);
      setTanda((n) => n + 1);
      setTarea("descifrar");
    };

    const dejarDeAbrir = alAbrirFichero(abrir);
    esfinge.aperturaDeArranque().then(abrir).catch(() => {});

    return () => {
      dejarDeEscuchar();
      dejarDeObedecer();
      dejarDeAbrir();
    };
  }, []);

  async function descargar() {
    setFalloAlBajar("");
    setAvance({ bytes: 0, total: novedad?.bytes ?? 0, hecho: false });

    const dejarDeEscuchar = alDescargar(setAvance);
    try {
      await esfinge.descargarActualizacion();
    } catch (e) {
      setAvance(undefined);
      setFalloAlBajar(mensaje(e));
    } finally {
      dejarDeEscuchar();
    }
  }

  return (
    <div className="ventana">
      <BarraLateral valor={tarea} alCambiar={setTarea} />

      <div className="zona">
        <header className="herramientas">
          <h1>{TITULOS[tarea]}</h1>
          {!enWails() && <span className="aparte">Modo desarrollo</span>}
        </header>

        {novedad && (
          <BandaNovedad
            novedad={novedad}
            avance={avance}
            error={falloAlBajar}
            instalando={instalando}
            alDescargar={descargar}
            alInstalar={() => {
              setInstalando(true);
              esfinge.instalarActualizacion().catch((e) => {
                setInstalando(false);
                setFalloAlBajar(mensaje(e));
              });
            }}
            alCerrar={() => setNovedad(null)}
          />
        )}

        <main className="contenido">
          {tarea === "cifrar" && <Trabajo key="cifrar" accion="cifrar" />}
          {tarea === "descifrar" && (
            // La clave lleva el número de tanda: si el sistema manda otro fichero
            // con la pantalla ya abierta, se rehace con él en vez de quedarse con
            // el de antes.
            <Trabajo
              key={`descifrar-${tanda}`}
              accion="descifrar"
              alArrancar={alArrancar ?? undefined}
            />
          )}
          {tarea === "generar" && <Generar />}
          {tarea === "historial" && <Historial />}
          {tarea === "ajustes" && <Ajustes version={version} alEncontrar={setNovedad} />}
        </main>
      </div>
    </div>
  );
}

/** El nombre de cada sección, que es lo que pone la barra de herramientas. */
const TITULOS: Record<Tarea, string> = {
  cifrar: "Cifrar",
  descifrar: "Descifrar",
  generar: "Generar una contraseña",
  historial: "Historial",
  ajustes: "Ajustes",
};

/**
 * Trabajo sirve para cifrar y para descifrar, que son la misma pantalla con los
 * verbos cambiados. Tenerlas separadas duplicaría el manejo de ficheros, el de
 * la clave y el de los errores para ganar dos palabras distintas.
 */
function Trabajo({
  accion,
  alArrancar,
}: {
  accion: "cifrar" | "descifrar";
  alArrancar?: Apertura;
}) {
  const cifrando = accion === "cifrar";

  // Lo que manda el sistema decide en qué modo se abre esta pantalla: un .esf
  // que lleva un fichero cifrado se abre en ficheros, y uno que lleva el
  // contenedor de una línea se abre en texto, con la línea puesta. Lo mira Go
  // por dentro; aquí solo se obedece.
  const [modo, setModo] = useState<Modo>(alArrancar?.modo === "ficheros" ? "ficheros" : "texto");
  const [texto, setTexto] = useState(alArrancar?.texto ?? "");
  const [clave, setClave] = useState("");
  const [ficheros, setFicheros] = useState<string[]>(alArrancar?.rutas ?? []);

  const [trabajando, setTrabajando] = useState(false);
  const [progreso, setProgreso] = useState<TipoProgreso | null>(null);
  const [resultado, setResultado] = useState<{ texto: string; aviso: string } | null>(null);
  const [hechos, setHechos] = useState<ResultadoFichero[]>([]);
  const [error, setError] = useState("");
  const [guardadoEn, setGuardadoEn] = useState("");
  const [copiadoSolo, setCopiadoSolo] = useState(false);

  // Lo que se suelte sobre la ventana entra por aquí, con su ruta absoluta.
  useEffect(() => {
    return alSoltarFicheros((rutas) => {
      setModo("ficheros");
      setFicheros((antes) => [...new Set([...antes, ...rutas])]);
      setError("");
    });
  }, []);

  useEffect(() => alProgresar(setProgreso), []);

  const limpiar = () => {
    setResultado(null);
    setHechos([]);
    setError("");
    setGuardadoEn("");
    setCopiadoSolo(false);
    setProgreso(null);
  };

  const elegir = useCallback(async () => {
    try {
      const rutas = cifrando
        ? await esfinge.elegirFicheros(true)
        : await esfinge.elegirCifrados();
      if (rutas?.length) {
        setFicheros((antes) => [...new Set([...antes, ...rutas])]);
        setError("");
      }
    } catch (e) {
      setError(mensaje(e));
    }
  }, [cifrando]);

  async function ejecutar() {
    limpiar();
    setTrabajando(true);
    try {
      if (modo === "texto") {
        const r = cifrando
          ? await esfinge.cifrarTexto(texto, clave)
          : await esfinge.descifrarTexto(texto, clave);
        setResultado(r);

        // Al cifrar, lo siguiente que se hace con el resultado es pegarlo en
        // algún sitio, siempre. Al descifrar no: eso es el secreto en claro, y
        // dejarlo en el portapapeles sin que nadie lo pida es meterlo donde
        // puede leerlo cualquier cosa.
        if (cifrando) {
          try {
            await navigator.clipboard.writeText(r.texto);
            setCopiadoSolo(true);
          } catch {
            /* sin portapapeles queda el botón de guardar */
          }
        }
      } else {
        const r = cifrando
          ? await esfinge.cifrarFicheros(ficheros, clave)
          : await esfinge.descifrarFicheros(ficheros, clave);
        setHechos(r);
      }
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  const listo =
    clave !== "" && (modo === "texto" ? texto.trim() !== "" : ficheros.length > 0);
  const verbo = cifrando ? "Cifrar" : "Descifrar";

  return (
    <div className="panel">
      <p className="entradilla">
        {cifrando
          ? "Lo que salga solo se abre con la clave que pongas. Vale cualquier fichero."
          : "Hace falta la misma clave con la que se cifró."}
      </p>

      <div className="grupo">
        <Segmentado<Modo>
          valor={modo}
          alCambiar={(m) => {
            setModo(m);
            limpiar();
          }}
          opciones={[
            { valor: "texto", etiqueta: "Texto" },
            { valor: "ficheros", etiqueta: "Ficheros" },
          ]}
        />

        {modo === "texto" ? (
          <div>
            <label htmlFor="texto">
              {cifrando ? "Qué quieres cifrar" : "El texto cifrado"}
            </label>
            <textarea
              id="texto"
              value={texto}
              onChange={(e) => setTexto(e.target.value)}
              placeholder={
                cifrando
                  ? "Una contraseña, un token, lo que sea"
                  : "Pega aquí el ESF1.… que te han pasado"
              }
            />
          </div>
        ) : (
          <ZonaFicheros
            ficheros={ficheros}
            alElegir={elegir}
            alQuitar={(r) => setFicheros((a) => a.filter((x) => x !== r))}
            texto={
              cifrando
                ? "Arrastra aquí los ficheros que quieras cifrar"
                : "Arrastra aquí los ficheros cifrados"
            }
            admite={
              cifrando
                ? "vale cualquier fichero"
                : "ficheros .esf, o de texto con un ESF1.…"
            }
          />
        )}

        <CampoClave valor={clave} alCambiar={setClave} alEnviar={() => listo && ejecutar()} />
      </div>

      {/* El aviso desaparece cuando ya hay resultado: el propio resultado trae
          el suyo, y aquí no cambiamos de pantalla, así que se verían los dos a la
          vez diciendo lo mismo. */}
      {cifrando && !resultado && hechos.length === 0 && (
        <p className="aviso">
          Si pierdes la clave, se pierde el contenido. No hay forma de recuperarlo.
        </p>
      )}

      <div className="botones">
        <button className="principal" onClick={ejecutar} disabled={!listo || trabajando}>
          {trabajando ? "Trabajando…" : verbo}
        </button>
      </div>

      {trabajando && progreso && progreso.total > 1 && <Progreso {...progreso} />}
      {error && <p className="error">{error}</p>}

      {resultado && (
        <PanelResultado
          texto={resultado.texto}
          aviso={resultado.aviso}
          exito={
            copiadoSolo
              ? "Copiado al portapapeles · ya lo puedes pegar donde haga falta"
              : guardadoEn
                ? `Guardado en ${guardadoEn}`
                : undefined
          }
          nombreSugerido={cifrando ? "secreto.esf" : "secreto.txt"}
          alGuardar={setGuardadoEn}
        />
      )}

      {hechos.length > 0 && <Tanda hechos={hechos} />}
    </div>
  );
}

/** Tanda enseña cómo le ha ido a cada fichero. */
function Tanda({ hechos }: { hechos: ResultadoFichero[] }) {
  const bien = hechos.filter((h) => !h.error);
  const mal = hechos.filter((h) => h.error);

  return (
    <div>
      {bien.length > 0 && (
        <p className="exito">
          {bien.length === 1
            ? "Un fichero listo"
            : `${bien.length} ficheros listos`}
        </p>
      )}
      {mal.length > 0 && (
        <p className="error">
          {mal.length === 1 ? "Uno ha fallado" : `${mal.length} han fallado`}
        </p>
      )}

      <ul className="lista-ficheros" style={{ marginTop: 12 }}>
        {hechos.map((h) => (
          <li key={h.origen}>
            <span className="nombre" title={h.destino || h.origen}>
              {nombreDe(h.origen)}
            </span>
            <span className={h.error ? "error" : "nota"}>
              {h.error ? h.error : `→ ${nombreDe(h.destino)}`}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function Generar() {
  const [alfabetos, setAlfabetos] = useState<Alfabeto[]>([]);
  const [alfabeto, setAlfabeto] = useState("hex");

  // Una contraseña se mide de dos maneras: en caracteres, cuando hay que
  // pegarla en un formulario que limita la longitud, y en bits, cuando lo que
  // importa es lo cara que sea de adivinar. Se puede mover cualquiera de las
  // dos, y la otra se recalcula: quien conoce una no tiene por qué conocer la
  // otra. Las cuentas las hace Go, que es quien genera la contraseña.
  const [caracteres, setCaracteres] = useState(32);
  const [medida, setMedida] = useState<Medida | null>(null);

  const [contrasena, setContrasena] = useState("");
  const [error, setError] = useState("");
  const [guardadoEn, setGuardadoEn] = useState("");

  useEffect(() => {
    esfinge.alfabetos().then(setAlfabetos).catch(() => {});
  }, []);

  const generar = useCallback(async () => {
    setError("");
    setGuardadoEn("");
    try {
      const m = await esfinge.medirPorCaracteres(caracteres, alfabeto);
      setMedida(m);
      setContrasena(await esfinge.generarContrasena(m.bytes, alfabeto));
    } catch (e) {
      setError(mensaje(e));
    }
  }, [caracteres, alfabeto]);

  useEffect(() => {
    generar();
  }, [generar]);

  const elegido = alfabetos.find((a) => a.nombre === alfabeto);
  const bits = medida?.bits ?? 0;
  const salen = medida?.caracteres ?? caracteres;

  // La misma escala que el medidor de claves, para no dar dos opiniones
  // distintas sobre lo mismo.
  const nivel = bits >= 128 ? 4 : bits >= 100 ? 3 : bits >= 80 ? 2 : bits >= 60 ? 1 : 0;
  const juicio =
    nivel >= 4 ? "Excelente" : nivel === 3 ? "Buena" : nivel === 2 ? "Aceptable" : "Corta";

  return (
    <div className="panel">
      <p className="entradilla">Al azar, con la entropía del sistema.</p>

      <div className="grupo">
        <div>
          <label>Qué caracteres</label>
          <Segmentado
            valor={alfabeto}
            alCambiar={setAlfabeto}
            opciones={alfabetos.map((a) => ({ valor: a.nombre, etiqueta: a.etiqueta }))}
          />
        </div>

        <div>
          <div className="fila">
            <label htmlFor="largo" style={{ marginBottom: 0 }}>
              Longitud
            </label>
            <span className="cifra">
              {salen} caracteres · {bits} bits
            </span>
          </div>
          <input
            id="largo"
            type="range"
            min={CARACTERES_MINIMO}
            max={CARACTERES_MAXIMO}
            step={2}
            value={caracteres}
            onChange={(e) => setCaracteres(Number(e.target.value))}
          />
          <div className="medidor" data-nivel={nivel} aria-hidden="true">
            <span />
            <span />
            <span />
            <span />
            <span />
          </div>
          <p className="nota" style={{ marginTop: 6 }}>
            {juicio} · cuantos más caracteres, más cara de adivinar
          </p>
        </div>

        {elegido?.aviso ? (
          <p className="aviso">{elegido.aviso}</p>
        ) : (
          <p className="nota">Segura dentro de una URL</p>
        )}
      </div>

      {error && <p className="error">{error}</p>}

      {contrasena && (
        <PanelResultado
          texto={contrasena}
          exito={guardadoEn ? `Guardado en ${guardadoEn}` : undefined}
          nombreSugerido="contrasena.txt"
          alGuardar={setGuardadoEn}
          extra={<button onClick={generar}>Generar otra</button>}
        />
      )}
    </div>
  );
}

/**
 * Ajustes es donde se cuenta la única cosa que Esfinge hace fuera de esta
 * máquina, y donde se apaga.
 *
 * Que esté a la vista no es cortesía: la portada dice que nada sale del
 * ordenador, y a partir de la comprobación de versiones sale una petición. Si se
 * hace, se dice, y se deja apagar.
 */
function Ajustes({
  version,
  alEncontrar,
}: {
  version: string;
  alEncontrar: (n: Novedad) => void;
}) {
  const [prefs, setPrefs] = useState<Preferencias | null>(null);
  const [vidrio, setVidrio] = useState<boolean | null>(null);
  const [buscando, setBuscando] = useState(false);
  const [dicho, setDicho] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    esfinge.verPreferencias().then(setPrefs).catch(() => {});
    esfinge.vidrio().then(setVidrio).catch(() => {});
  }, []);

  async function cambiar(buscarActualizaciones: boolean) {
    if (!prefs) return;
    const siguiente = { ...prefs, buscarActualizaciones };
    setPrefs(siguiente);
    try {
      await esfinge.guardarPreferencias(siguiente);
    } catch (e) {
      setError(mensaje(e));
    }
  }

  async function buscarAhora() {
    setBuscando(true);
    setError("");
    setDicho("");
    try {
      const n = await esfinge.comprobarActualizacion();
      if (n.hay) {
        alEncontrar(n);
        setDicho(`Hay una versión nueva: Esfinge ${n.version}.`);
      } else {
        setDicho("Ya tienes la última versión.");
      }
      esfinge.verPreferencias().then(setPrefs).catch(() => {});
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setBuscando(false);
    }
  }

  return (
    <div className="panel">
      <p className="entradilla">Tienes instalada la versión {version}.</p>

      <div className="grupo">
        <label className="fila-ajuste">
          <input
            type="checkbox"
            checked={prefs?.buscarActualizaciones ?? true}
            onChange={(e) => cambiar(e.target.checked)}
          />
          <span>Avisarme cuando haya una versión nueva</span>
        </label>

        <p className="nota">
          Es lo único que Esfinge hace fuera de tu ordenador: una vez al día le pregunta a
          GitHub cuál es la última versión publicada. No manda nada de lo que cifras, ni quién
          eres, ni cuántas veces la usas. En la petición viaja el número de versión que tienes,
          que es lo que se compara, y GitHub ve tu dirección IP, como cualquier página que
          visites.
        </p>

        {prefs?.ultimaComprobacion && (
          <p className="nota">Se miró por última vez el {fecha(prefs.ultimaComprobacion)}.</p>
        )}

        <div className="botones">
          <button onClick={buscarAhora} disabled={buscando}>
            {buscando ? "Buscando…" : "Buscar ahora"}
          </button>
        </div>

        {dicho && <p className="exito">{dicho}</p>}
        {error && <p className="error">{error}</p>}
      </div>

      {vidrio !== null && (
        <p className="nota">
          {vidrio
            ? "Esta ventana usa el vidrio del sistema: la barra y el pie dejan ver lo que hay detrás."
            : "Esta ventana es opaca: tu sistema no ofrece el vidrio, o esta versión no lo trae."}
        </p>
      )}

      <p className="nota">
        Al actualizar no hay que desinstalar nada: en macOS se arrastra encima de la anterior,
        en Windows el asistente la sustituye y en Linux lo hace el paquete. Tu historial y estos
        ajustes se quedan donde están.
      </p>
    </div>
  );
}

function Historial() {
  const [entradas, setEntradas] = useState<Entrada[]>([]);
  const [donde, setDonde] = useState("");

  const recargar = useCallback(() => {
    esfinge.verHistorial().then(setEntradas).catch(() => {});
    esfinge.dondeVive().then(setDonde).catch(() => {});
  }, []);

  useEffect(recargar, [recargar]);

  async function vaciar() {
    await esfinge.vaciarHistorial();
    recargar();
  }

  return (
    <div className="panel">
      <p className="entradilla">
        Solo qué fichero y cuándo. Nunca el contenido, ni la clave, ni el texto cifrado.
      </p>

      {entradas.length === 0 ? (
        <p className="nota">Todavía no has hecho nada.</p>
      ) : (
        <ul className="historial">
          {entradas.map((e, i) => (
            <li key={`${e.cuando}-${i}`}>
              <span style={{ flex: 1 }}>
                {e.accion === "cifrar" ? "Cifrado" : "Descifrado"} · {e.nombre}
              </span>
              <span className="cuando">{fecha(e.cuando)}</span>
            </li>
          ))}
        </ul>
      )}

      <div className="botones">
        <button onClick={vaciar} disabled={entradas.length === 0}>
          Vaciar historial
        </button>
      </div>

      {donde && <p className="nota seleccionable">Se guarda en {donde}</p>}
    </div>
  );
}

function fecha(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(undefined, {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function mensaje(e: unknown): string {
  if (e instanceof Error) return e.message;
  return String(e);
}
