import { useCallback, useEffect, useRef, useState } from "react";
import { obedecerEdicion } from "./ordenes";
import {
  alAbrirFichero,
  alDescargar,
  alHaberNovedad,
  alOrdenar,
  alPedirloUnAgente,
  alQuererAlgoUnAgente,
  alPedirloUnNavegador,
  alProgresar,
  alSoltarFicheros,
  enWails,
  esfinge,
  type Alfabeto,
  type Apertura,
  type Avance,
  type Entrada,
  type EstadoDelAgente,
  type EstadoDelNavegador,
  type EstadoDesbloqueo,
  type Medida,
  type Novedad,
  type Preferencias,
  type Progreso as TipoProgreso,
  type ResultadoFichero,
  CARACTERES_MINIMO,
  CARACTERES_MAXIMO,
  NUNCA,
  alCambiarLaBoveda,
  alCambiarElEstadoDeLaBoveda,
} from "./puente";
import {
  BandaNovedad,
  BarraLateral,
  CampoClave,
  Firma,
  Icono,
  Marca,
  nombreDe,
  PanelResultado,
  Progreso,
  Segmentado,
  ZonaFicheros,
} from "./componentes";
import { avisoDelSistemaAlGuardar, Boveda } from "./boveda";
import { Proyectos } from "./proyectos";
import { Asistente, Bienvenida, GrupoCuenta, usaCuenta, usaSincroAlVolver, type TipoAsistente } from "./cuenta";

// **«mcp» no es una pantalla, es una puerta a un bloque de Ajustes.** Se pidió así
// —«para no hacer una pantalla nueva»— y tiene su razón: lo que hay ahí dentro son
// ajustes, y partirlos en dos sitios obligaría a decidir cuál enseña el interruptor.
// Lo que gana es que **se encuentre**: quien busca cómo conectar su agente no mira en
// Ajustes, mira en la barra.
type Tarea =
  | "cifrar"
  | "descifrar"
  | "generar"
  | "boveda"
  | "proyectos"
  | "historial"
  | "mcp"
  | "ajustes";
type Modo = "texto" | "ficheros";

export default function App() {
  const [tarea, setTarea] = useState<Tarea>("cifrar");
  // Lo que se ha visitado alguna vez. Una sección se monta la primera vez que se
  // entra en ella y a partir de ahí se queda, solo que escondida.
  const [visitadas, setVisitadas] = useState<Set<Tarea>>(() => new Set<Tarea>(["cifrar"]));

  useEffect(() => {
    setVisitadas((antes) => (antes.has(tarea) ? antes : new Set(antes).add(tarea)));
  }, [tarea]);
  const [version, setVersion] = useState("");
  const [alArrancar, setAlArrancar] = useState<Apertura | null>(null);
  // Cambia cada vez que el sistema manda ficheros, para que la pantalla de
  // descifrar se rehaga con ellos aunque ya estuviera abierta.
  const [tanda, setTanda] = useState(0);

  // La clave que viene del generador. El sello es un contador y no el valor:
  // lo que dispara el efecto de abajo es que **cambie**, y pedir dos veces
  // seguidas la misma contraseña es perfectamente posible.
  const [claveGenerada, setClaveGenerada] = useState<{ valor: string; sello: number } | null>(null);

  // La cuenta (ADR 0035): sin elegir todavía, toca la bienvenida; y el asistente
  // para crearla o entrar, que va a ventana entera. `selloBoveda` rehace la
  // sección de la bóveda al terminar, que ya no es la de antes.
  const [cuenta, refrescarCuenta] = usaCuenta();
  const [asistente, setAsistente] = useState<{ tipo: TipoAsistente; hayBoveda: boolean } | null>(null);
  const [selloBoveda, setSelloBoveda] = useState(0);
  // Cada vez que alguien pide ver quién tiene acceso **desde dentro del proyecto**.
  // Contador y no booleano: lo que viaja es un gesto, y con un booleano el segundo
  // clic no haría nada hasta apagarlo.
  const [verAcceso, setVerAcceso] = useState(0);
  usaSincroAlVolver(cuenta?.modo === "cuenta");

  const abrirAsistente = (tipo: TipoAsistente) => {
    esfinge
      .estadoBoveda()
      .then((e) => setAsistente({ tipo, hayBoveda: e.existe }))
      .catch(() => setAsistente({ tipo, hayBoveda: false }));
  };

  // El candado de la barra lateral: se pregunta una vez y luego lo avisa Go en
  // cada cambio. Crear la bóveda la deja abierta y también avisa.
  const [bovedaAbierta, setBovedaAbierta] = useState<boolean | null>(null);
  // Y **cuál** es la que está abierta (ADR 0050): con varias bóvedas, «Bóveda» a
  // secas en la barra de herramientas deja de identificar nada. Vacío es la personal.
  // El nombre viene **dentro del estado de la bóveda**: con la referencia sola
  // habría que pedir la lista para traducirla, y pedirla con la bóveda cerrada es
  // un error seguro que acaba como un 400 en la consola del navegador.
  const [nombreDelProyecto, setNombreDelProyecto] = useState("");
  useEffect(() => {
    const mirar = (abierta?: boolean) =>
      esfinge
        .estadoBoveda()
        .then((e) => {
          setBovedaAbierta(e.existe ? e.abierta : null);
          setNombreDelProyecto(e.abierta ? e.nombreDelProyecto : "");
        })
        .catch(() => abierta !== undefined && setBovedaAbierta(abierta));
    void mirar();
    return alCambiarElEstadoDeLaBoveda((abierta) => void mirar(abierta));
  }, []);

  const enUnProyecto = nombreDelProyecto !== "";



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

  if (asistente) {
    return (
      <Asistente
        tipo={asistente.tipo}
        version={version}
        hayBoveda={asistente.hayBoveda}
        alVolver={() => setAsistente(null)}
        alTerminar={() => {
          setAsistente(null);
          refrescarCuenta();
          setSelloBoveda((n) => n + 1);
          setTarea("boveda");
        }}
      />
    );
  }

  if (cuenta?.modo === "") {
    return (
      <Bienvenida
        version={version}
        alElegirLocal={refrescarCuenta}
        alCrearCuenta={() => abrirAsistente({ que: "crear" })}
        alEntrar={() => abrirAsistente({ que: "entrar" })}
      />
    );
  }

  return (
    <div className="ventana">
      <BarraLateral
        valor={tarea}
        alCambiar={setTarea}
        version={version}
        bovedaAbierta={bovedaAbierta}
        nombreDelProyecto={nombreDelProyecto}
      />

      <div className="zona">
        <header className={enUnProyecto ? "herramientas en-proyecto" : "herramientas"}>
          {/* **En qué bóveda se está trabajando.** Con varias, saber que la bóveda
              está abierta no basta: hay que saber cuál. Y el nombre **es el título**,
              no un rótulo al lado: puesto al lado, lo que más se leía seguía diciendo
              «Bóveda» y la pantalla parecía la personal. Lo vio el cliente con la
              2.38.0 —«parecería que estoy en mi bóveda personal»—. */}
          <div className="titulo">
            {enUnProyecto && tarea === "boveda" && <span className="antetitulo">Proyecto</span>}
            <h1>{enUnProyecto && tarea === "boveda" ? nombreDelProyecto : TITULOS[tarea]}</h1>
          </div>
          {/* Y la salida, **siempre a la vista** mientras haya un proyecto abierto.
              Antes no estaba en ninguna parte: `VolverALaBovedaPersonal` existía en Go
              y no la llamaba nadie, así que la única forma de salir era bloquear la
              bóveda y desbloquear. */}
          {/* **Y quién tiene acceso, desde dentro.** La pantalla donde se da y se quita
              vive en la fila de «Proyectos», que es donde tiene que estar —dar acceso
              escribe en el fichero de ese proyecto—, pero **ahí no es donde se busca**:
              el cliente lo buscó dentro del proyecto, que es donde se está trabajando
              (2026-10-06). Esto no duplica la pantalla: lleva a ella y la abre. */}
          {enUnProyecto && tarea === "boveda" && (
            <button
              className="discreto"
              onClick={() => {
                setVerAcceso((n) => n + 1);
                setTarea("proyectos");
              }}
            >
              Quién tiene acceso…
            </button>
          )}
          {enUnProyecto && (tarea === "boveda" || tarea === "proyectos") && (
            <SalirDelProyecto
              alSalir={() => {
                // **A la lista de proyectos, que es lo que dice el botón.** Lo pidió
                // el cliente con esas palabras —«¿no sería más intuitivo por lenguaje
                // que me llevara al listado de proyectos?»— y encaja con que ahora la
                // personal vuelva abierta: salir es salir de ahí, no entrar en otro
                // sitio. Tu bóveda queda a un clic, y abierta.
                setSelloBoveda((n) => n + 1);
                setTarea("proyectos");
              }}
            />
          )}
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
          {/* Cada sección se queda montada desde la primera vez que se visita, y
              lo que hace se esconde en vez de desmontarse.

              Antes se desmontaba al salir, y con ella se iba lo escrito: se
              tecleaba el secreto, se iba uno a Generar a por una clave, y al
              volver el campo estaba vacío. Justo el camino que la propia
              aplicación propone desde que existe «Usar como clave».

              **Montadas la primera vez, no todas de golpe**: Generar saca una
              contraseña nada más montarse, y una herramienta que cifra no debería
              fabricar un secreto que nadie ha pedido solo por si acaso. */}
          <Panel activo={tarea === "cifrar"} visitado={visitadas.has("cifrar")}>
            <Trabajo accion="cifrar" claveGenerada={claveGenerada} />
          </Panel>

          <Panel activo={tarea === "descifrar"} visitado={visitadas.has("descifrar")}>
            {/* La clave lleva el número de tanda: si el sistema manda otro fichero
                con la pantalla ya abierta, se rehace con él en vez de quedarse con
                el de antes. */}
            <Trabajo
              key={`descifrar-${tanda}`}
              accion="descifrar"
              alArrancar={alArrancar ?? undefined}
            />
          </Panel>

          <Panel activo={tarea === "generar"} visitado={visitadas.has("generar")}>
            <Generar
              alUsarComoClave={(valor) => {
                setClaveGenerada((antes) => ({ valor, sello: (antes?.sello ?? 0) + 1 }));
                setTarea("cifrar");
              }}
            />
          </Panel>

          {/* La bóveda se entera de que la están mirando, y no le da igual: es lo
              que aplaza el bloqueo por inactividad y lo que hace que al volver se
              vea el estado de ahora y no el de hace media hora. */}
          <Panel activo={tarea === "boveda"} visitado={visitadas.has("boveda")}>
            <Boveda
              key={selloBoveda}
              activo={tarea === "boveda"}
              alVolverAEntrar={(correo) => abrirAsistente({ que: "entrar", correo, deNuevo: true })}
              alEntrarConLaNueva={(correo) => abrirAsistente({ que: "entrar", correo })}
              alRecuperar={(correo) => abrirAsistente({ que: "recuperar", correo })}
            />
          </Panel>

          <Panel activo={tarea === "proyectos"} visitado={visitadas.has("proyectos")}>
            <Proyectos
              activo={tarea === "proyectos"}
              verAcceso={verAcceso}
              alEntrar={() => {
                // Conmutar rehace la pantalla de la bóveda: lo que había montado era
                // la lista de la otra, con su búsqueda y su ficha abierta.
                setSelloBoveda((n) => n + 1);
                setTarea("boveda");
              }}
            />
          </Panel>

          <Panel activo={tarea === "historial"} visitado={visitadas.has("historial")}>
            <Historial recargar={tarea === "historial"} />
          </Panel>

          {/* **Las dos filas abren este panel**, y por eso `activo` mira las dos: si
              «mcp» no contara, entrar por ahí dejaría la pantalla en blanco. */}
          <Panel
            activo={tarea === "ajustes" || tarea === "mcp"}
            visitado={visitadas.has("ajustes") || visitadas.has("mcp")}
          >
            <Ajustes
              porElBloqueMCP={tarea === "mcp"}
              version={version}
              alEncontrar={setNovedad}
              alCrearCuenta={() => abrirAsistente({ que: "crear" })}
              alEntrar={(correo, deNuevo) => abrirAsistente({ que: "entrar", correo, deNuevo })}
            />
          </Panel>
        </main>
      </div>
    </div>
  );
}

/**
 * Panel guarda una sección: la monta la primera vez que se visita y luego la
 * esconde en vez de quitarla, para que no se pierda lo que hubiera dentro.
 *
 * El «hidden» necesita ayuda del CSS: la hoja del navegador lo resuelve con
 * «display: none», pero «.panel» declara «display: flex» y gana por ser de
 * autor. En estilos.css hay un «[hidden] { display: none !important }» que lo
 * arregla, y sin él esconder no escondería nada.
 */
function Panel({
  activo,
  visitado,
  children,
}: {
  activo: boolean;
  visitado: boolean;
  children: React.ReactNode;
}) {
  if (!visitado) return null;
  return <div hidden={!activo}>{children}</div>;
}

/**
 * Salir del proyecto y volver a la bóveda personal (ADR 0050).
 *
 * **Pide la maestra otra vez**, y eso no se adivina del rótulo: la personal no se
 * queda abierta por detrás mientras hay un proyecto abierto —dos bóvedas abiertas a
 * la vez es justo lo que se descartó—, así que volver es abrirla. Por eso hay un
 * segundo clic que lo dice, el mismo idioma que «Borrar» → «Sí, a la papelera»: la
 * acción no destruye nada, pero sin avisar se vive como que Esfinge se ha bloqueado
 * solo.
 */
function SalirDelProyecto({ alSalir }: { alSalir: () => void }) {
  const [seguro, setSeguro] = useState(false);
  const [trabajando, setTrabajando] = useState(false);

  // El segundo clic no se queda esperando para siempre: con el botón armado en una
  // pantalla a la que nadie vuelve, el clic siguiente —días después— saldría del
  // proyecto sin que esa persona haya pedido nada.
  useEffect(() => {
    if (!seguro) return;
    const t = setTimeout(() => setSeguro(false), 8000);
    return () => clearTimeout(t);
  }, [seguro]);

  return (
    <button
      className="discreto salir-proyecto"
      disabled={trabajando}
      title="Vuelve a tu bóveda. Habrá que teclear la contraseña maestra otra vez."
      onClick={async () => {
        if (!seguro) {
          setSeguro(true);
          return;
        }
        setTrabajando(true);
        try {
          await esfinge.volverALaBovedaPersonal();
          alSalir();
        } finally {
          setTrabajando(false);
          setSeguro(false);
        }
      }}
    >
      {trabajando ? "Saliendo…" : seguro ? "Sí, salir y cerrar" : "Salir del proyecto"}
    </button>
  );
}

/** El nombre de cada sección, que es lo que pone la barra de herramientas. */
const TITULOS: Record<Tarea, string> = {
  cifrar: "Cifrar",
  descifrar: "Descifrar",
  generar: "Generar una contraseña",
  boveda: "Bóveda",
  proyectos: "Proyectos",
  historial: "Historial",
  mcp: "Ajustes",
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
  claveGenerada,
}: {
  accion: "cifrar" | "descifrar";
  alArrancar?: Apertura;
  /** Lo que manda el generador. Ver el efecto de más abajo. */
  claveGenerada?: { valor: string; sello: number } | null;
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

  // Si la clave viene del generador hay que decirlo, porque es distinta de una
  // que la persona sabe: ésta no está en ninguna parte. Se apaga en cuanto se
  // teclea encima, que entonces ya es otra cosa.
  const [claveDelGenerador, setClaveDelGenerador] = useState(false);

  // La clave que llega del generador se aplica **sin rehacer la pantalla**. El
  // patrón de remontar con «key» que usan los ficheros del sistema aquí borraría
  // el texto que se estuviera escribiendo, que es justo lo que se iba a cifrar.
  useEffect(() => {
    if (!claveGenerada) return;
    setClave(claveGenerada.valor);
    setClaveDelGenerador(true);
  }, [claveGenerada?.sello]);

  // Sacar una clave al azar sin salir de aquí. Los valores por defecto son los
  // de la pantalla de Generar: 32 caracteres en hexadecimal, que es el único
  // alfabeto que sobrevive dentro de una URL (ADR 0004).
  const generarClave = useCallback(async () => {
    try {
      const m = await esfinge.medirPorCaracteres(32, "hex");
      setClave(await esfinge.generarContrasena(m.bytes, "hex"));
      setClaveDelGenerador(true);
    } catch (e) {
      setError(mensaje(e));
    }
  }, []);

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
            <label htmlFor={`texto-${accion}`}>
              {cifrando ? "Qué quieres cifrar" : "El texto cifrado"}
            </label>
            <textarea
              id={`texto-${accion}`}
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

        <CampoClave
          valor={clave}
          alCambiar={(v) => {
            setClave(v);
            setClaveDelGenerador(false);
          }}
          alEnviar={() => listo && ejecutar()}
          alGenerar={cifrando ? generarClave : undefined}
          id={`clave-${accion}`}
        />
      </div>

      {/* El aviso desaparece cuando ya hay resultado: el propio resultado trae
          el suyo, y aquí no cambiamos de pantalla, así que se verían los dos a la
          vez diciendo lo mismo. */}
      {cifrando && !resultado && hechos.length === 0 && (
        <p className="aviso">
          {claveDelGenerador
            ? "Esta clave acaba de generarse y no está guardada en ninguna parte. Cópiala o guárdala antes de cifrar, o el contenido se perderá."
            : "Si pierdes la clave, se pierde el contenido. No hay forma de recuperarlo."}
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

function Generar({ alUsarComoClave }: { alUsarComoClave: (clave: string) => void }) {
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
          extra={
            <>
              <button
                onClick={async () => {
                  // Se copia además de llevarla, para que quepa pegarla en un
                  // gestor de contraseñas sin volver atrás. Es la misma decisión
                  // que al cifrar un texto (ADR 0011), con la misma pega: el
                  // portapapeles lo ve cualquier programa que lo vigile.
                  try {
                    await navigator.clipboard.writeText(contrasena);
                  } catch {
                    // Si el portapapeles falla, llevarla sigue valiendo.
                  }
                  alUsarComoClave(contrasena);
                }}
              >
                Usar como clave
              </button>
              <button onClick={generar}>Generar otra</button>
            </>
          }
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
/**
 * Desbloquear la bóveda con el sistema: Touch ID o Windows Hello (fase C).
 *
 * **Lo que esta pantalla tiene que decir, y por eso lleva tanto texto**: sin
 * firmar la aplicación —que es la decisión de la ADR 0012— esto es un cerrojo y
 * no una llave. Protege de quien se siente delante de tu ordenador desbloqueado;
 * no protege de un programa que corra como tú, que es de lo que sí protege la
 * contraseña maestra. Quien lo activa tiene derecho a saberlo aquí y no en un
 * documento.
 *
 * Y donde no hay biometría —Linux, un Mac sin Touch ID— **no se enseña un botón
 * apagado**: se dice que este equipo no tiene, que es una respuesta y no un
 * misterio.
 */
function DesbloqueoDelSistema() {
  const [estado, setEstado] = useState<EstadoDesbloqueo | null>(null);
  const [trabajando, setTrabajando] = useState(false);
  const [error, setError] = useState("");

  // **Lo mismo que las preferencias, y por lo mismo** (ver `cambiosHechos` más
  // abajo): esto se lee al montar y otra vez cada vez que la bóveda se abre o se
  // cierra, así que **una lectura pedida antes de un cambio puede llegar
  // después** y dejar el interruptor enseñando lo de antes. Con el interruptor,
  // además, el siguiente clic parte de ahí y deshace lo que se acababa de hacer.
  //
  // Y devuelve la promesa a propósito: quien cambia el interruptor la espera
  // antes de volver a dejarlo pulsable, para que no se pueda pulsar dos veces
  // sobre un estado que todavía no se sabe.
  const cambiosHechos = useRef(0);
  const mirar = useCallback(() => {
    const cuandoSePidio = cambiosHechos.current;
    return esfinge
      .estadoDelDesbloqueo()
      .then((e) => {
        if (cambiosHechos.current !== cuandoSePidio) return; // lo leído ya es viejo
        setEstado(e);
      })
      .catch(() => {});
  }, []);
  useEffect(() => {
    void mirar();
  }, [mirar]);
  // Activarlo exige la bóveda abierta, así que el estado se vuelve a mirar cuando
  // se abre o se cierra: si no, el interruptor se queda diciendo lo de antes.
  useEffect(() => alCambiarElEstadoDeLaBoveda(() => void mirar()), [mirar]);

  async function cambiar(activar: boolean) {
    setTrabajando(true);
    setError("");
    cambiosHechos.current++;
    try {
      if (activar) await esfinge.activarDesbloqueo();
      else await esfinge.quitarDesbloqueo();
    } catch (e) {
      setError(mensaje(e));
    } finally {
      await mirar();
      setTrabajando(false);
    }
  }

  if (!estado) return null;

  if (!estado.hay) {
    return (
      <p className="nota">
        <strong>Este equipo no tiene desbloqueo del sistema.</strong> Donde lo hay —Touch ID en un
        Mac, Windows Hello— la bóveda se puede abrir con la huella en vez de con la contraseña
        maestra. Aquí siempre se escribe.
      </p>
    );
  }

  return (
    <>
      <label className="fila-ajuste">
        <input
          id="desbloqueo-del-sistema"
          type="checkbox"
          checked={estado.puesto}
          disabled={trabajando}
          onChange={(e) => cambiar(e.target.checked)}
        />
        <span>Abrir la bóveda con {estado.nombre}</span>
      </label>

      {error && <p className="error">{error}</p>}

      {/* Lo que el sistema va a preguntar, dicho antes de que lo pregunte. */}
      {!estado.puesto && avisoDelSistemaAlGuardar() && (
        <p className="nota">{avisoDelSistemaAlGuardar()}</p>
      )}

      <p className="nota">
        Para activarlo, la bóveda tiene que estar abierta. <strong>La contraseña maestra sigue
        abriendo siempre</strong>, y la clave de recuperación también: esto se añade, no sustituye a
        nada.
      </p>
      <p className="aviso">
        <strong>Protege de quien se siente delante de tu ordenador desbloqueado, no de un programa
        que corra en él.</strong> Esfinge no está firmada, así que {estado.nombre} guarda la llave
        sin poder atarla solo a Esfinge. Si eso te importa, deja esto apagado y escribe la
        contraseña.
      </p>
    </>
  );
}

/**
 * ComoConectarlo: lo que hay que hacer en cada cliente para que vea la bóveda.
 *
 * **Son cuatro caminos distintos y no uno con variantes**, que es lo que había hasta
 * la 2.44.0 —un párrafo, una orden y un JSON «para lo demás»— y por lo que instalarlo
 * en Claude Code falló el primer día. Dos cosas lo demuestran:
 *
 *   - **Claude Code quiere `--scope user`.** Sin eso el servidor queda registrado solo
 *     en la carpeta donde se pegó la orden, y desde cualquier otro proyecto Esfinge
 *     no está. Sin ningún error.
 *   - **VS Code no lee `mcpServers`, lee `servers`.** Pegarle el bloque de Claude
 *     tampoco da error: no carga nada.
 *
 * Un bloque genérico acierta en la mitad de los sitios y en la otra mitad falla
 * callado. Así que cada uno con su nombre, y se elige con el mismo control segmentado
 * que las clases de la bóveda — que ya existe y tiene sus colores medidos.
 */
function ComoConectarlo({ elAgente }: { elAgente: EstadoDelAgente }) {
  const [cual, setCual] = useState<"desktop" | "code" | "cursor" | "vscode">("desktop");
  const [guardado, setGuardado] = useState("");
  const [fallo, setFallo] = useState("");
  const [guardando, setGuardando] = useState(false);

  async function guardarElPaquete() {
    setFallo("");
    setGuardado("");
    setGuardando(true);
    try {
      const donde = await esfinge.guardarPaqueteMCP();
      // Vacío es que se canceló el diálogo, que no es un fallo ni es nada que decir.
      if (donde) setGuardado(donde);
    } catch (e) {
      setFallo(mensaje(e));
    } finally {
      setGuardando(false);
    }
  }

  return (
    <div className="conectar">
      <Segmentado
        opciones={[
          { valor: "desktop", etiqueta: "Claude Desktop" },
          { valor: "code", etiqueta: "Claude Code" },
          { valor: "cursor", etiqueta: "Cursor" },
          { valor: "vscode", etiqueta: "VS Code" },
        ]}
        valor={cual}
        alCambiar={setCual}
      />

      {cual === "desktop" && (
        <div className="receta">
          <p className="nota">
            Un paquete que se instala abriéndolo. Lo arma Esfinge con{" "}
            <strong>la versión que tienes puesta</strong>, así que no hay que acertar con
            ninguna descarga.
          </p>
          <div className="botones">
            <button className="principal" onClick={guardarElPaquete} disabled={guardando}>
              {guardando ? "Guardando…" : "Guardar el paquete…"}
            </button>
          </div>
          {guardado && (
            <p className="nota seleccionable">
              Guardado en {guardado}. <strong>Ábrelo con doble clic</strong> y Claude Desktop
              lo instalará.
            </p>
          )}
          {fallo && <p className="error">{fallo}</p>}
        </div>
      )}

      {cual === "code" && (
        <div className="receta">
          <p className="nota">Pega esto en una terminal. No hay ningún fichero que tocar.</p>
          <ParaCopiar id="config-mcp-orden" valor={elAgente.orden} />
          <p className="nota">
            <code>--scope user</code> es lo que hace que esté en todos tus proyectos. Sin eso
            queda registrado solo en la carpeta desde donde lo ejecutes.
          </p>
          {/* **Y esto cubre también los IDE, que es lo que no se ve.** La extensión de
              Claude Code es el mismo Claude Code, y lee la misma configuración: con la
              orden de arriba, Esfinge ya está dentro de tu editor. Sin decirlo, quien
              usa Claude dentro de Cursor se va a la pestaña de Cursor y acaba
              configurando **el agente de Cursor**, que es otro programa. */}
          <p className="nota">
            <strong>Vale también dentro de tu editor</strong>: la extensión de Claude Code
            para VS Code, Cursor o JetBrains es el mismo Claude Code y lee esta misma
            configuración. Con la orden de arriba no hay nada más que hacer.
          </p>
        </div>
      )}

      {cual === "cursor" && (
        <div className="receta">
          {/* **Esto es para el agente de Cursor, no para Claude dentro de Cursor.** Son
              dos agentes distintos en el mismo editor y cada uno lee su configuración;
              confundirlos es configurar el que no se está usando. */}
          <p className="nota">
            Esto es para <strong>el agente propio de Cursor</strong>. Si lo que usas ahí es la
            extensión de Claude Code, lo tuyo es la pestaña anterior.
          </p>
          <p className="nota">
            Va en <code>~/.cursor/mcp.json</code>, o en <code>.cursor/mcp.json</code> si lo
            quieres solo en un proyecto.
          </p>
          <textarea
            id="config-mcp"
            className="seleccionable"
            readOnly
            rows={7}
            value={elAgente.configuracion}
            onFocus={(e) => e.currentTarget.select()}
          />
        </div>
      )}

      {cual === "vscode" && (
        <div className="receta">
          <p className="nota">
            Esto es para <strong>el agente propio de VS Code</strong>. Si ahí usas la extensión
            de Claude Code, lo tuyo es la pestaña de Claude Code.
          </p>
          <p className="nota">
            Va en <code>.vscode/mcp.json</code>, o en tu perfil con{" "}
            <strong>MCP: Add Server</strong> desde la paleta de órdenes.
          </p>
          <textarea
            id="config-mcp-vscode"
            className="seleccionable"
            readOnly
            rows={8}
            value={elAgente.vscode}
            onFocus={(e) => e.currentTarget.select()}
          />
          <p className="aviso">
            Ojo: VS Code usa <code>servers</code> y los demás usan <code>mcpServers</code>.
            Con el bloque equivocado <strong>no da ningún error y no carga nada</strong>.
          </p>
        </div>
      )}

      <details>
        <summary>En otro agente</summary>
        <p className="nota">
          Casi todos esperan el bloque de arriba —el de Cursor— con la ruta entera. Si el
          tuyo solo pide una orden, es ésta:
        </p>
        <ParaCopiar id="config-mcp-ruta" valor={elAgente.ruta} />
      </details>
    </div>
  );
}

/**
 * ParaCopiar es una línea que se copia: un campo de solo lectura que se selecciona
 * entero al enfocarlo.
 *
 * **No lleva botón de copiar a propósito.** De un campo normal el navegador sí deja
 * copiar —lo que no deja es de uno de contraseña, que es otra historia y está resuelta
 * por Go—, así que un botón aquí sería un camino más que mantener para ganar un clic.
 */
function ParaCopiar({ id, valor }: { id: string; valor: string }) {
  return (
    <input
      id={id}
      className="seleccionable para-copiar"
      readOnly
      value={valor}
      onFocus={(e) => e.currentTarget.select()}
    />
  );
}

function Ajustes({
  version,
  alEncontrar,
  alCrearCuenta,
  alEntrar,
  porElBloqueMCP = false,
}: {
  version: string;
  alEncontrar: (n: Novedad) => void;
  alCrearCuenta: () => void;
  alEntrar: (correo?: string, deNuevo?: boolean) => void;
  /**
   * Se ha entrado por la fila «MCP» de la barra lateral, no por «Ajustes».
   *
   * Es la misma pantalla —lo pidió así el cliente, «para no hacer una pantalla
   * nueva»— y lo único que cambia es **por dónde se abre**: el bloque de los agentes
   * está abajo del todo y llegar ahí a ciegas es no llegar.
   */
  porElBloqueMCP?: boolean;
}) {
  const [cuenta] = usaCuenta();
  const [prefs, setPrefs] = useState<Preferencias | null>(null);
  const [navegador, setNavegador] = useState<EstadoDelNavegador | null>(null);
  const [vidrio, setVidrio] = useState<boolean | null>(null);
  const [buscando, setBuscando] = useState(false);
  const [dicho, setDicho] = useState("");
  const [error, setError] = useState("");

  // Lo último que se ha decidido guardar, que **no es lo mismo que el estado**.
  //
  // Go recibe el objeto entero, así que cada cambio manda también lo que no se ha
  // tocado. Si eso se lee del estado, dos cambios seguidos —bajar el bloqueo y
  // acto seguido el portapapeles— salen los dos del mismo valor de partida,
  // porque entre el primero y el segundo React todavía no ha vuelto a dibujar: el
  // segundo guardado **deshace el primero**, y en pantalla los dos se ven puestos.
  // Lo encontró una prueba, no una persona, y en un ajuste que apaga el bloqueo de
  // la bóveda eso no puede quedarse así.
  const ultimasPrefs = useRef<Preferencias | null>(null);

  // Cuántos cambios se han hecho aquí dentro. **Es lo que impide que una lectura
  // que llega tarde deshaga uno de ellos.**
  //
  // Las preferencias se leen más de una vez —al montar el panel, y otra vez
  // después de buscar actualizaciones— y una respuesta pedida antes de un cambio
  // puede llegar después: trae lo que había, se aplica encima, y **el cambio
  // siguiente parte de ahí y borra el anterior sin que nada lo diga**. Pasó
  // exactamente así: bajar el bloqueo a cinco minutos y acto seguido el
  // portapapeles a diez dejaba el bloqueo otra vez en quince.
  const cambiosHechos = useRef(0);

  const leerPreferencias = useCallback(() => {
    const cuandoSePidio = cambiosHechos.current;
    esfinge
      .verPreferencias()
      .then((p) => {
        if (cambiosHechos.current !== cuandoSePidio) return; // lo leído ya es viejo
        ultimasPrefs.current = p;
        setPrefs(p);
      })
      .catch(() => {});
  }, []);

  const leerNavegador = useCallback(() => {
    esfinge.estadoDelNavegador().then(setNavegador).catch(() => {});
  }, []);

  const [elAgente, setElAgente] = useState<EstadoDelAgente | null>(null);
  const leerAgente = useCallback(() => {
    esfinge.estadoDelAgente().then(setElAgente).catch(() => {});
  }, []);

  // Los sitios en los que la extensión no ofrece guardar, que viven dentro de la
  // bóveda. Se vuelven a leer cuando el navegador apunta uno nuevo.
  const [excluidos, setExcluidos] = useState<string[]>([]);
  const leerExcluidos = useCallback(() => {
    esfinge.sitiosExcluidos().then(setExcluidos).catch(() => {});
  }, []);
  useEffect(() => {
    leerExcluidos();
    return alCambiarLaBoveda(leerExcluidos);
  }, [leerExcluidos]);

  useEffect(() => {
    leerPreferencias();
    leerNavegador();
    esfinge.vidrio().then(setVidrio).catch(() => {});
    // Cuando un navegador pide permiso hay que enterarse **sin que nadie
    // recargue nada**: quien lo está pidiendo está mirando la otra ventana.
    const dejarDeOir = alPedirloUnNavegador(leerNavegador);
    // Y lo mismo con los agentes: quien lo está pidiendo está mirando su terminal,
    // no esta ventana, así que esto tiene que aparecer **sin que nadie recargue nada**.
    const dejarDeOirAgente = alPedirloUnAgente(leerAgente);
    // Y cuando pide algo que hay que aprobar, por lo mismo: quien lo pide está mirando
    // su terminal, no esta ventana.
    const dejarDeOirQuiere = alQuererAlgoUnAgente(leerAgente);
    leerAgente();
    return () => {
      dejarDeOir();
      dejarDeOirAgente();
      dejarDeOirQuiere();
    };
  }, [leerPreferencias, leerNavegador, leerAgente]);

  async function cambiar(cambio: Partial<Preferencias>) {
    const base = ultimasPrefs.current ?? prefs;
    if (!base) return;
    const siguiente = { ...base, ...cambio };
    cambiosHechos.current++;
    ultimasPrefs.current = siguiente;
    setPrefs(siguiente);
    try {
      await esfinge.guardarPreferencias(siguiente);
      leerNavegador();
      leerAgente();
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
      leerPreferencias();
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setBuscando(false);
    }
  }

  // **Mientras no hayan llegado, los controles no se pueden tocar.**
  //
  // Se dibujan con su valor de siempre —la casilla marcada, quince minutos— para
  // que la pantalla no dé un salto al cargar, y eso está bien; lo que no vale es
  // dejar que se pulsen, porque `cambiar` no tiene de dónde partir y **el clic no
  // hace nada, en silencio**: la casilla vuelve sola a como estaba. Dura lo que
  // tarda una llamada al proceso de al lado, pero una prueba lo pilló.
  const cargando = prefs === null;

  // **Entrar por «MCP» lleva a su bloque.** Sin esto, la fila de la barra abre
  // Ajustes por arriba y lo que se buscaba queda a una pantalla de distancia hacia
  // abajo: el clic parecería no haber hecho nada.
  //
  // Va con `useEffect` y no en el clic porque el panel **puede no estar montado
  // todavía** —las secciones se montan la primera vez que se visitan—, y entonces no
  // hay a qué saltar. El salto es suave salvo que se haya pedido lo contrario, que
  // es un movimiento grande y no decorativo.
  const bloqueMCP = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (!porElBloqueMCP) return;
    const quieto = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
    bloqueMCP.current?.scrollIntoView({
      behavior: quieto ? "auto" : "smooth",
      block: "start",
    });
  }, [porElBloqueMCP]);

  return (
    <div className="panel">
      {/* La ficha de producto. Aquí es donde la ADR 0007 prometía que estaría la
          marca —«en el icono y en Acerca de»— y donde el menú «Acerca de
          Esfinge» lleva desde siempre, porque no hay diálogo aparte: la orden
          navega a esta pantalla. Hasta la 2.11.0 lo que había era una línea con
          el número de versión, así que la promesa estaba a medias.

          Sustituye a la entradilla, que decía lo mismo con menos. */}
      <div className="ficha">
        <Marca lado={44} />
        <div>
          <h2>Esfinge</h2>
          <p className="nota">Cifra y descifra secretos con una clave.</p>
          <Firma version={version} />
        </div>
      </div>

      <GrupoCuenta alCrearCuenta={alCrearCuenta} alEntrar={alEntrar} />

      {/* Los dos relojes de la bóveda.
       *
       * Van aquí y no dentro de la bóveda porque son ajustes de la aplicación y
       * porque el del portapapeles no es solo de la bóveda: también borra lo que
       * copia «Usar como clave», que hasta la 2.11.x se quedaba ahí para siempre.
       *
       * «Nunca» viaja como -1 y no como 0. El cero es «no lo he dicho», que es lo
       * que llega cuando alguien guarda un objeto a medias: si significara
       * «nunca», ese descuido apagaría el bloqueo de la bóveda sin que nadie lo
       * pidiera. Lo cuenta entero internal/app/preferencias.go. */}
      <section className="bloque">
        <header className="bloque-cabecera">
          <Icono nombre="boveda" />
          <div>
            <h3>La bóveda</h3>
            <p className="nota">Cuándo se cierra sola, cómo se abre y qué se borra al copiar.</p>
          </div>
        </header>
        <div>
          <label htmlFor="bloqueo">Cerrar la bóveda sola</label>
          <select
            id="bloqueo"
            disabled={cargando}
            value={prefs?.minutosParaBloquear ?? 15}
            onChange={(e) => cambiar({ minutosParaBloquear: Number(e.target.value) })}
          >
            {[1, 5, 15, 30, 60, 240, NUNCA].map((m) => (
              <option key={m} value={m}>
                {m === NUNCA ? "Nunca" : `Tras ${m} ${m === 1 ? "minuto" : "minutos"} sin tocar nada`}
              </option>
            ))}
          </select>
          <p className="nota">
            Cerrarla obliga a volver a escribir la contraseña maestra. Con «nunca» se queda
            abierta hasta que se cierre a mano o se cierre la aplicación.
          </p>
        </div>

        <DesbloqueoDelSistema />

        <label className="fila-ajuste">
          <input
            type="checkbox"
            checked={prefs?.descargarIconos ?? true}
            disabled={cargando}
            onChange={(e) => cambiar({ descargarIconos: e.target.checked, iconosAvisados: true })}
          />
          <span>Descargar el icono de cada sitio de la bóveda</span>
        </label>

        <p className="nota">
          Es la segunda cosa que Esfinge hace fuera de tu ordenador. Le pide el icono a cada sitio
          de tu bóveda, directamente y nunca a un intermediario, y lo guarda cifrado junto a ella.
          Quien pueda mirar tu red verá a qué sitios pregunta. Sin esto, cada entrada sale con un
          cuadro de color y su inicial.
        </p>

        <div>
          <label htmlFor="portapapeles">Borrar del portapapeles lo que se copie</label>
          <select
            id="portapapeles"
            disabled={cargando}
            value={prefs?.segundosDePortapapeles ?? 30}
            onChange={(e) => cambiar({ segundosDePortapapeles: Number(e.target.value) })}
          >
            {[10, 30, 60, 120, NUNCA].map((s) => (
              <option key={s} value={s}>
                {s === NUNCA ? "Nunca" : `A los ${s} segundos`}
              </option>
            ))}
          </select>
          <p className="nota">
            Vale para las contraseñas de la bóveda y para las que se generan aquí. Nunca se pisa
            lo que hayas copiado tú después.
          </p>
        </div>
      </section>

      {/* **El aviso de los iconos, una vez.**
       *
       * Es la segunda cosa que Esfinge hace fuera de este ordenador y viene
       * encendida, así que quien actualice empezará a preguntar por sus sitios sin
       * haber pedido nada. La costumbre de esta casa para eso está escrita desde la
       * ADR 0014: si se hace, se dice, y se deja apagar. Aquí se dice una vez, con
       * el «no, gracias» al lado y con el dato incómodo delante —que el nombre del
       * sitio viaja en claro aunque el icono venga cifrado—. */}
      {prefs && prefs.descargarIconos && !prefs.iconosAvisados && (
        <div className="grupo peligro">
          <label>Esfinge va a pedir el icono de cada sitio de tu bóveda</label>
          <p className="aviso">
            Se lo pide <strong>a cada sitio directamente</strong>, nunca a un intermediario. Aun
            así, quien pueda mirar tu red verá <strong>a qué sitios pregunta</strong>: el nombre
            viaja en claro antes de que empiece el cifrado.
          </p>
          <p className="nota">
            Va poco a poco y espaciado, no de golpe. Los iconos se guardan cifrados, junto a la
            bóveda.
          </p>
          <div className="botones">
            <button
              className="principal"
              onClick={() => cambiar({ iconosAvisados: true })}
            >
              De acuerdo
            </button>
            <button onClick={() => cambiar({ descargarIconos: false, iconosAvisados: true })}>
              No, gracias
            </button>
          </div>
        </div>
      )}



      {/* **El canal con el navegador.**
       *
       * Va aquí, con las otras dos cosas que Esfinge hace fuera de sí misma, y con
       * una diferencia que hay que decir: las otras dos **salen** a la red y ésta
       * **abre una puerta** a este ordenador. Por eso viene apagada, al revés que
       * los iconos.
       *
       * Y cuando un navegador pide permiso, la respuesta se da aquí y no en el
       * navegador: es lo único de todo esto que la página que estás mirando no
       * puede tocar. */}
      <section className="bloque">
        <header className="bloque-cabecera">
          <Icono nombre="navegador" />
          <div>
            <h3>El navegador</h3>
            <p className="nota">La extensión de Esfinge: rellenar, guardar y los códigos.</p>
          </div>
          {/* **El interruptor de la sección vive en su cabecera**, y solo ahí. Dejar
              también la casilla de antes sería preguntar dos veces lo mismo y dejar
              sin respuesta única a quien la busque por su nombre. El rótulo se
              conserva tal cual en `aria-label`: es como la localizan las pruebas y
              quien usa un lector de pantalla. */}
          <label className="interruptor">
            <input
              type="checkbox"
              checked={prefs?.puenteDelNavegador ?? false}
              disabled={cargando}
              onChange={(e) => cambiar({ puenteDelNavegador: e.target.checked })}
              aria-label="Dejar que la extensión del navegador consulte la bóveda"
            />
          </label>
        </header>

        <p className="nota">
          Abre un canal <strong>dentro de este ordenador</strong>, no en la red: no hay puerto al
          que nadie pueda conectarse desde fuera. Por él salen las cuentas del sitio que estés
          mirando y, cuando las pides, una contraseña cada vez. Nunca la contraseña maestra.
        </p>

        {/* **Con cuenta, este canal sobra, y hay que decirlo donde se ve.** Desde la
            2.25.0 la extensión entra con la cuenta en su propio panel y va siempre
            por ella, esté Esfinge abierta o cerrada (ADR 0040). El interruptor se
            queda —en local sigue siendo la única forma— y quien estrene Esfinge lo
            tiene apagado, que es como viene de fábrica. Lo que no se hace es
            apagarlo solo al entrar en una cuenta: en un navegador donde todavía no
            se haya entrado con la cuenta, eso dejaría de rellenar sin avisar. */}
        {cuenta?.modo === "cuenta" && (
          <p className="nota">
            <strong>Con cuenta no hace falta.</strong> Entra con tu cuenta en el panel de la
            extensión y funcionará sola, también con Esfinge cerrada. Esto solo sirve si prefieres
            que el navegador le pregunte a esta aplicación; si no lo usas, déjalo apagado.
          </p>
        )}

        {navegador?.error && <p className="error">{navegador.error}</p>}

        {navegador?.escuchando && (
          <>
            <p className="nota seleccionable">Escucha en {navegador.donde}</p>
            {/* **A quién se ha avisado.** Sin esto, un navegador al que no se le
                dejó el manifiesto se ve igual que uno al que sí: el interruptor
                puesto y nada más. Costó un viaje al Mac. */}
            {navegador.avisados.length > 0 ? (
              <p className="nota">Avisados: {navegador.avisados.join(", ")}.</p>
            ) : (
              <p className="aviso">
                No se ha avisado a ningún navegador. Si tienes uno instalado,
                cuéntamelo: el canal está abierto pero ninguno sabe que existe.
              </p>
            )}
          </>
        )}

        {/* **El freno de las llaves de acceso** (ADR 0048).
            Va aquí, con lo del navegador, porque es de lo que apaga: no toca nada
            de la bóveda ni de la ventana, solo lo que la extensión puede hacer
            dentro de una página. Viene encendido, que es como se decidió
            publicarlo, y existe porque hay código de Esfinge dentro de cada página
            `https`: si un sitio cambia y deja de entrar, esto se apaga y se sigue
            trabajando sin esperar a una versión. */}
        <label className="fila-ajuste">
          <input
            type="checkbox"
            checked={prefs?.llavesDeAccesoEnElNavegador ?? true}
            disabled={cargando}
            onChange={(e) => cambiar({ llavesDeAccesoEnElNavegador: e.target.checked })}
          />
          <span>Usar tus llaves de acceso en el navegador</span>
        </label>

        <p className="nota">
          Cuando un sitio pida una llave de acceso, Esfinge se ofrecerá a poner la tuya. Si lo
          apagas, el navegador preguntará como si Esfinge no estuviera y tus llaves seguirán
          guardadas aquí. Apágalo si algún sitio deja de dejarte entrar.
        </p>

        {/* Lo que pide permiso. Va en «peligro» a propósito: es la única pregunta
            de esta pantalla cuya respuesta le abre la bóveda a otro programa. */}
        {navegador?.pide && (
          <div className="grupo peligro">
            <label>{navegador.pide} quiere consultar tu bóveda</label>
            <p className="aviso">
              Si no has sido tú al abrir el navegador, <strong>di que no</strong>. Con permiso podrá
              preguntar qué cuentas tienes de cada sitio que visites y pedir su contraseña.
            </p>
            <div className="botones">
              <button
                className="principal"
                onClick={async () => {
                  await esfinge.permitirNavegador();
                  leerNavegador();
                }}
              >
                Permitirlo
              </button>
              <button onClick={leerNavegador}>Ahora no</button>
            </div>
          </div>
        )}

        {navegador && navegador.permitidos?.length > 0 && (
          <div>
            <label>Navegadores permitidos</label>
            <ul className="lista-papelera">
              {navegador.permitidos.map((n) => (
                <li key={n.desde}>
                  <span className="nombre">{n.quien}</span>
                  <span className="nota">Desde el {fecha(n.desde)}</span>
                  <span className="acciones">
                    <button
                      className="discreto"
                      onClick={async () => {
                        await esfinge.olvidarNavegador(n.desde);
                        leerNavegador();
                      }}
                    >
                      Retirar
                    </button>
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </section>

      {/* **El canal con los agentes de IA** (ADR 0054), en sección propia.

          Hasta la 2.44.0 compartía tarjeta con el del navegador y se leía como una
          lista larga: el cliente dijo que ahí se pierde, y tenía razón. Son la misma
          clase de cosa —una puerta hacia dentro de este ordenador— pero no la misma
          cosa, y cada una tiene su interruptor, su socket y su lista de permitidos a
          propósito: apagar una no puede apagar la otra.

          El `id` es a donde salta la fila «MCP» de la barra lateral. */}
      <section className="bloque" id="ajustes-mcp" ref={bloqueMCP}>
        <header className="bloque-cabecera">
          <Icono nombre="mcp" />
          <div>
            <h3>Agentes de IA</h3>
            <p className="nota">
              Claude, Cursor y los demás, por el protocolo MCP.
            </p>
          </div>
          <label className="interruptor">
            <input
              type="checkbox"
              checked={prefs?.canalDeAgentes ?? false}
              disabled={cargando}
              onChange={(e) => cambiar({ canalDeAgentes: e.target.checked })}
              aria-label="Dejar que un agente de IA consulte la bóveda"
            />
          </label>
        </header>

        
        <p className="nota">
          Abre otro canal <strong>dentro de este ordenador</strong>, para que programas como Claude
          puedan buscar en tu bóveda y ayudarte a ordenarla. <strong>No les da tus contraseñas</strong>
          : lo que se usa se copia al portapapeles, y cada uso se aprueba aquí.
        </p>

        <p className="aviso">
          Lo que un agente lea <strong>acaba en la conversación de su modelo</strong>, con quien lo
          sirva. Y si ese agente puede ejecutar órdenes en tu equipo —como el de una terminal—,
          puede leer el portapapeles: ahí lo que te protege no es la copia,{" "}
          <strong>es que cada uso te lo pregunte y quede apuntado</strong>.
        </p>

        {elAgente?.error && <p className="error">{elAgente.error}</p>}

        {elAgente?.escuchando && (
          <>
            <p className="nota seleccionable">Escucha en {elAgente.donde}</p>
            <ComoConectarlo elAgente={elAgente} />
          </>
        )}

        {/* Lo que pide permiso. En «peligro» por lo mismo que el del navegador: su
            respuesta le abre la bóveda a otro programa. */}
        {elAgente?.pide && (
          <div className="grupo peligro">
            <label>{elAgente.pide} quiere consultar tu bóveda</label>
            <p className="aviso">
              Si no has sido tú al abrir ese programa, <strong>di que no</strong>. Con permiso podrá
              ver qué cuentas tienes, de qué sitios y cuáles están mal; para usar una contraseña
              tendrá que pedírtelo cada vez.
            </p>
            <div className="botones">
              <button
                className="principal"
                onClick={async () => {
                  await esfinge.permitirAgente();
                  leerAgente();
                }}
              >
                Permitirlo
              </button>
              <button onClick={leerAgente}>Ahora no</button>
            </div>
          </div>
        )}

        {/* **Lo que un agente quiere ahora mismo.** En «peligro» como la de emparejar,
            y por una razón de más: aquí lo que se aprueba es **una contraseña
            concreta**, y el sí vale una vez y solo para ella.

            Dice **qué** se pide, no solo que se pide algo: «un agente quiere una
            contraseña» no es una pregunta que se pueda contestar. */}
        {elAgente?.quiere && (
          <div className="grupo peligro">
            <label>
              {elAgente.quiere.quien} quiere{" "}
              {elAgente.quiere.que === "codigo"
                ? `el código de un solo uso de «${elAgente.quiere.titulo}»`
                : `la contraseña de «${elAgente.quiere.titulo}»`}
            </label>
            {/* **Las dos peticiones no son lo mismo y no se dicen igual.** Una se copia
                y el agente no la ve; la otra **se la enseñas**, y eso se queda en la
                conversación de su modelo. Decirlas con el mismo texto sería esconder
                justo la diferencia que hace falta para contestar. */}
            {elAgente.quiere.que === "codigo" ? (
              <p className="aviso">
                Esas seis cifras <strong>sí las verá</strong>, y se quedarán en la conversación de
                su modelo. Caducan en medio minuto y no sirven sin la contraseña, pero si no le has
                pedido nada que lo necesite, <strong>di que no</strong>.
              </p>
            ) : (
              <p className="aviso">
                Se copiará al portapapeles de este ordenador: <strong>el agente no la ve</strong>.
                Si no le has pedido nada que la necesite, <strong>di que no</strong>: lo que un
                agente lee por ahí puede decirle qué pedir.
              </p>
            )}
            <div className="botones">
              <button
                className="principal"
                onClick={async () => {
                  await esfinge.aprobarLoQuePideElAgente(false);
                  leerAgente();
                }}
              >
                {elAgente.quiere.que === "codigo" ? "Darle el código" : "Solo ésta"}
              </button>
              {/* **El «un rato» no es el botón principal**, y eso es a propósito: es
                  lo único de esta pantalla que quita una pregunta, así que no puede
                  ser lo que se pulsa sin mirar.

                  **Y no sale para el código**, porque la válvula no lo cubre: ofrecer
                  aquí un botón que no va a hacer lo que dice sería peor que no
                  ofrecerlo. */}
              {elAgente.quiere.que !== "codigo" && (
                <button
                  onClick={async () => {
                    await esfinge.aprobarLoQuePideElAgente(true);
                    leerAgente();
                  }}
                >
                  Todo lo suyo, 5 minutos
                </button>
              )}
              <button
                onClick={async () => {
                  await esfinge.denegarLoQuePideElAgente();
                  leerAgente();
                }}
              >
                No
              </button>
            </div>
          </div>
        )}

        {/* **El contador de la válvula, mientras está abierta.** Es lo que hace
            soportable haber dicho «durante cinco minutos»: se ve lo que se le va
            dando y se puede cortar sin esperar a que caduque. */}
        {elAgente?.valvula?.abierta && (
          <div className="grupo peligro">
            <label>
              Dándole lo que pida durante {Math.max(0, Math.ceil(elAgente.valvula.quedan / 60))} min
            </label>
            <p className="nota">
              Van {elAgente.valvula.usadas} de {elAgente.valvula.tope}
              {elAgente.valvula.ultimos && elAgente.valvula.ultimos.length > 0 && (
                <> · {elAgente.valvula.ultimos.slice(-3).join(", ")}</>
              )}
              . Al llegar al tope vuelve a preguntar.
            </p>
            <div className="botones">
              <button
                className="principal"
                onClick={async () => {
                  await esfinge.cortarAlAgente();
                  leerAgente();
                }}
              >
                Cortar
              </button>
            </div>
          </div>
        )}

        {elAgente && elAgente.permitidos?.length > 0 && (
          <div>
            <label>Agentes permitidos</label>
            <ul className="lista-papelera">
              {elAgente.permitidos.map((g) => (
                <li key={g.desde}>
                  <span className="nombre">{g.quien}</span>
                  <span className="nota">Desde el {fecha(g.desde)}</span>
                  <span className="acciones">
                    <button
                      className="discreto"
                      onClick={async () => {
                        await esfinge.olvidarAgente(g.desde);
                        leerAgente();
                      }}
                    >
                      Retirar
                    </button>
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* **Lo que se le ha dado a un agente**, que es la otra mitad de dejarle
            entrar: hasta la ADR 0054, Esfinge no registraba qué entradas se abrían, y
            con una persona delante eso se sostiene —lo que has mirado lo has mirado
            tú—. Con un programa pidiendo cosas, «¿qué le di la semana pasada?» es una
            pregunta que se hace sola.

            Vive **dentro de la bóveda cifrada**, así que esto solo se ve con ella
            abierta. Se guardan noventa días. */}
        {(elAgente?.dado?.length ?? 0) > 0 && (
          <div>
            <label>Lo que les has dado</label>
            <ul className="lista-papelera">
              {(elAgente?.dado ?? []).map((d) => (
                <li key={d.id}>
                  <span className="nombre">{d.titulo || "Una entrada"}</span>
                  <span className="nota">
                    {d.quien} · {fecha(d.cuando)} ·{" "}
                    {d.resultado === "hecho" ? "copiada" : d.resultado === "negado" ? "dijiste que no" : d.resultado}
                  </span>
                </li>
              ))}
            </ul>
            <p className="nota">Se guardan noventa días.</p>
          </div>
        )}

        {/* «Nunca en este sitio» se decide en la tarjeta de la página y se deshace
            aquí (ADR 0032). La lista está dentro de la bóveda, cifrada, y por eso
            solo se ve con la bóveda abierta. */}
        {excluidos.length > 0 && (
          <div>
            <label>Sitios en los que no se ofrece guardar</label>
            <ul className="lista-papelera">
              {excluidos.map((d) => (
                <li key={d}>
                  <span className="nombre">{d}</span>
                  <span className="acciones">
                    <button
                      className="discreto"
                      onClick={async () => {
                        await esfinge.quitarSitioExcluido(d);
                        leerExcluidos();
                      }}
                    >
                      Quitar
                    </button>
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </section>


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

      <section className="bloque">
        <header className="bloque-cabecera">
          <Icono nombre="descargar" />
          <div>
            <h3>Actualizaciones</h3>
            <p className="nota">La única salida a internet que Esfinge hace por sí sola.</p>
          </div>
        </header>
        <label className="fila-ajuste">
          <input
            type="checkbox"
            checked={prefs?.buscarActualizaciones ?? true}
            disabled={cargando}
            onChange={(e) => cambiar({ buscarActualizaciones: e.target.checked })}
          />
          <span>Avisarme cuando haya una versión nueva</span>
        </label>

        <p className="nota">
          Una de las dos cosas que Esfinge hace fuera de tu ordenador: una vez al día le pregunta
          a GitHub cuál es la última versión publicada. No manda nada de lo que cifras, ni quién
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
      </section>
    </div>
  );
}

function Historial({ recargar: aLaVista }: { recargar: boolean }) {
  const [entradas, setEntradas] = useState<Entrada[]>([]);
  const [donde, setDonde] = useState("");

  const recargar = useCallback(() => {
    esfinge.verHistorial().then(setEntradas).catch(() => {});
    esfinge.dondeVive().then(setDonde).catch(() => {});
  }, []);

  // Se recarga cada vez que se entra, no solo al montarse. Desde que las
  // secciones se quedan montadas, montarse pasa una sola vez: sin esto, el
  // historial enseñaría lo que había la primera vez que se miró y no lo que se
  // acaba de cifrar.
  useEffect(() => {
    if (aLaVista) recargar();
  }, [aLaVista, recargar]);

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
        // El único sitio de la aplicación donde la marca se permite ser grande,
        // y solo porque aquí no hay nada que estorbar (ADR 0021). En cuanto haya
        // una entrada, desaparece.
        //
        // Se tiñe con «--filete», que es un color que ya existe y que ya está
        // medido contra el lienzo en los dos temas. Con «opacity» habría que
        // inventar dos valores, uno por tema, y ninguno tendría prueba.
        <div className="vacio">
          <Marca lado={112} />
          <p className="nota">Todavía no has hecho nada.</p>
        </div>
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
