import { useEffect, useRef, useState } from "react";
import { esfinge, type Avance, type Fuerza, type Novedad } from "./puente";

// La marca se importa **en crudo desde build/**, donde vive el resto del dibujo
// de la casa, y no se copia a frontend/: una segunda copia del logotipo es
// exactamente lo que este proyecto ya ha pagado caro tres veces —el icono del
// volumen y el del documento salieron los dos de duplicar sin querer—.
//
// Va en línea y no como <img> porque así hereda «currentColor», que es lo que
// permite que la misma pieza sea la tinta en el lockup y el filete en el vacío
// del historial. Ver build/marca.svg, que explica cómo está dibujada.
import marcaSVG from "../../build/marca.svg?raw";

/**
 * Marca dibuja la esfinge, monocroma y del color que herede.
 *
 * El tamaño se pasa en píxeles; el grosor del trazo lo puede ajustar el CSS de
 * cada sitio, porque en el SVG es un atributo y no está clavado.
 */
export function Marca({ lado, clase }: { lado: number; clase?: string }) {
  return (
    <span
      className={clase ? `marca-esfinge ${clase}` : "marca-esfinge"}
      style={{ width: lado, height: lado }}
      dangerouslySetInnerHTML={{ __html: marcaSVG }}
    />
  );
}

/**
 * BarraLateral es la navegación de la ventana, a la izquierda.
 *
 * Es la estructura de cualquier aplicación de macOS con secciones —Ajustes,
 * Correo, App Store—: una columna con las secciones, la activa resaltada, y lo
 * que configura la aplicación separado abajo del todo.
 *
 * Arriba queda un hueco vacío a propósito: es el de los semáforos. Como la
 * ventana no tiene barra de título, los botones del sistema caen ahí encima.
 *
 * **Y aquí vive la marca** (ADR 0021): el lockup de Esfinge debajo de ese hueco
 * y la firma de la casa al fondo. Ninguna de las dos es interactiva, y eso no es
 * pereza sino dos decisiones:
 *
 * - `.lateral` es zona de arrastre de la ventana. Un enlace ahí obligaría a
 *   marcarlo `no-drag` y dejaría una tira muerta justo donde la gente agarra la
 *   ventana. El enlace a la casa va en la ficha de Ajustes, que no tiene ese
 *   problema.
 * - Las pruebas localizan las secciones con «.lateral + getByRole("button")».
 *   Un botón o un enlace de más aquí rompería ese localizador en todo el fichero
 *   de pruebas, y hay una prueba que cuenta que siguen siendo cinco.
 */
export function BarraLateral<T extends string>({
  valor,
  alCambiar,
  version,
}: {
  valor: T;
  alCambiar: (v: T) => void;
  version: string;
}) {
  const fila = (v: string, etiqueta: string) => (
    <button
      key={v}
      onClick={() => alCambiar(v as T)}
      aria-current={v === valor ? "page" : undefined}
    >
      <Icono nombre={v} />
      {etiqueta}
    </button>
  );

  return (
    <aside className="lateral">
      <div className="semaforos" />

      <div className="marca">
        <Marca lado={26} />
        <span>Esfinge</span>
      </div>

      <nav aria-label="Secciones">
        {fila("cifrar", "Cifrar")}
        {fila("descifrar", "Descifrar")}
        {fila("generar", "Generar")}
        {fila("historial", "Historial")}
      </nav>

      <nav className="abajo" aria-label="Configuración">
        {fila("ajustes", "Ajustes")}
      </nav>

      <Firma version={version} />
    </aside>
  );
}

/**
 * Firma es quién hizo esto y qué versión es: «Webcafeína ▍ 2.11.0».
 *
 * El glifo de bloque es el de la casa —el mismo del pie del LÉEME del DMG, del
 * README y de la línea de comandos, donde vive en Go como `salida.GlifoBarra`—
 * pero aquí hace de separador entre el nombre y la versión, que es lo que se
 * pidió al ver la 2.11.0. Se repite en vez de pedírselo a Go porque es un
 * rótulo, no un dato: cruzar el puente por dos cadenas constantes es un viaje
 * para nada.
 *
 * **La versión estaba solo en Ajustes**, que es un sitio al que hay que ir. Aquí
 * se ve siempre, y es lo primero que se pregunta cuando algo va raro.
 *
 * El glifo va marcado como decorativo para que quien lea la pantalla en voz alta
 * no oiga el nombre del carácter de bloque en medio de la frase.
 */
export function Firma({ version }: { version: string }) {
  return (
    <p className="firma">
      Webcafeína
      <span className="barra" aria-hidden="true">
        ▍
      </span>
      {version || "…"}
    </p>
  );
}

/**
 * Icono dibuja los glifos de la barra lateral.
 *
 * Van en SVG y no como emoji ni como SF Symbols. Los emoji son de color y
 * desentonan —las aplicaciones del sistema usan trazo monocromo— y de los SF
 * Symbols ya se sabe lo que pasa: no se puede comprobar desde CSS si la fuente
 * está, y cuando no lo está salen cuadrados vacíos. Dibujarlos es lo único que
 * se ve igual en los tres sistemas.
 */
function Icono({ nombre }: { nombre: string }) {
  const trazos: Record<string, React.ReactNode> = {
    // Candado cerrado.
    cifrar: (
      <>
        <rect x="3.5" y="7" width="11" height="7.5" rx="2" />
        <path d="M6 7V5a3 3 0 0 1 6 0v2" />
      </>
    ),
    // El mismo, con el arco abierto hacia un lado.
    descifrar: (
      <>
        <rect x="3.5" y="7" width="11" height="7.5" rx="2" />
        <path d="M12 7V5a3 3 0 0 0-6 0" />
      </>
    ),
    // Chispa: lo que se genera sale de la nada.
    generar: (
      <path d="M9 2.5v13M2.5 9h13M4.6 4.6l8.8 8.8M13.4 4.6l-8.8 8.8" />
    ),
    // Reloj.
    historial: (
      <>
        <circle cx="9" cy="9" r="6.2" />
        <path d="M9 5.4V9l2.6 1.8" />
      </>
    ),
    // Deslizadores, que es como el sistema dibuja los ajustes.
    ajustes: (
      <>
        <path d="M3 5.5h12M3 12.5h12" />
        <circle cx="7" cy="5.5" r="1.8" />
        <circle cx="11.5" cy="12.5" r="1.8" />
      </>
    ),
  };

  return (
    <svg
      className="icono"
      viewBox="0 0 18 18"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {trazos[nombre]}
    </svg>
  );
}

/** Control segmentado, que es como los dos sistemas agrupan modos excluyentes. */
export function Segmentado<T extends string>({
  opciones,
  valor,
  alCambiar,
}: {
  opciones: { valor: T; etiqueta: string }[];
  valor: T;
  alCambiar: (v: T) => void;
}) {
  return (
    <div className="segmentado" role="tablist">
      {opciones.map((o) => (
        <button
          key={o.valor}
          role="tab"
          aria-selected={o.valor === valor}
          onClick={() => alCambiar(o.valor)}
        >
          {o.etiqueta}
        </button>
      ))}
    </div>
  );
}

/**
 * CampoClave es el campo de la clave con su medidor.
 *
 * La valoración la hace Go, no el navegador: es la misma que usa la línea de
 * comandos, y tener dos opiniones distintas sobre la misma clave sería peor que
 * no tener ninguna.
 */
export function CampoClave({
  valor,
  alCambiar,
  etiqueta = "Clave",
  medir = true,
  alEnviar,
  alGenerar,
  id = "clave",
}: {
  valor: string;
  alCambiar: (v: string) => void;
  etiqueta?: string;
  medir?: boolean;
  alEnviar?: () => void;
  /**
   * Si se pasa, aparece un botón para sacar una clave al azar sin salir de esta
   * pantalla.
   *
   * Solo tiene sentido al cifrar: al descifrar la clave no se elige, se
   * recuerda, y un botón de generar ahí no significa nada.
   */
  alGenerar?: () => void;
  /**
   * Distinto en cada pantalla: cifrar y descifrar están montadas a la vez —para
   * que cambiar de sección no borre lo escrito— y dos campos con el mismo
   * identificador dejarían la etiqueta apuntando a cualquiera de los dos.
   */
  id?: string;
}) {
  const [fuerza, setFuerza] = useState<Fuerza | null>(null);

  useEffect(() => {
    if (!medir || !valor) {
      setFuerza(null);
      return;
    }
    let vigente = true;
    esfinge
      .evaluarClave(valor)
      .then((f) => vigente && setFuerza(f))
      .catch(() => {});
    return () => {
      vigente = false;
    };
  }, [valor, medir]);

  return (
    <div>
      <div className="fila">
        <label htmlFor={id} style={{ marginBottom: 0 }}>
          {etiqueta}
        </label>
        {alGenerar && (
          <button className="discreto" onClick={alGenerar}>
            Generar una
          </button>
        )}
      </div>
      <input
        id={id}
        type="password"
        value={valor}
        autoComplete="off"
        onChange={(e) => alCambiar(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && alEnviar?.()}
      />
      {fuerza && (
        <>
          <div className="medidor" data-nivel={fuerza.nivel} aria-hidden="true">
            <span />
            <span />
            <span />
            <span />
            <span />
          </div>
          <p className="nota" style={{ marginTop: 4 }}>
            {fuerza.etiqueta}
            {fuerza.sugerencia ? ` · ${fuerza.sugerencia}` : ""}
          </p>
        </>
      )}
    </div>
  );
}

/** Zona donde soltar ficheros, con su lista. */
export function ZonaFicheros({
  ficheros,
  alElegir,
  alQuitar,
  texto,
  admite,
}: {
  ficheros: string[];
  alElegir: () => void;
  alQuitar: (ruta: string) => void;
  texto: string;
  admite: string;
}) {
  // El resaltado al arrastrar por encima lo lleva el navegador. Wails entrega
  // las rutas cuando se suelta, pero no avisa de que hay algo encima, y sin esa
  // señal no se sabe si la ventana va a aceptar lo que se lleva en la mano.
  const [encima, setEncima] = useState(false);

  useEffect(() => {
    const entra = (e: DragEvent) => {
      e.preventDefault();
      setEncima(true);
    };
    const sale = (e: DragEvent) => {
      if (e.relatedTarget === null) setEncima(false);
    };
    const soltar = () => setEncima(false);

    window.addEventListener("dragover", entra);
    window.addEventListener("dragleave", sale);
    window.addEventListener("drop", soltar);
    return () => {
      window.removeEventListener("dragover", entra);
      window.removeEventListener("dragleave", sale);
      window.removeEventListener("drop", soltar);
    };
  }, []);

  return (
    <div>
      <div className={`soltar${encima ? " encima" : ""}`} onClick={alElegir}>
        <p className="icono" aria-hidden="true">
          ⇱
        </p>
        <p>{texto}</p>
        <p className="nota" style={{ justifyContent: "center", marginTop: 4 }}>
          O haz clic para elegirlos · {admite}
        </p>
      </div>

      {ficheros.length > 0 && (
        <ul className="lista-ficheros" style={{ marginTop: 12 }}>
          {ficheros.map((f) => (
            <li key={f}>
              <span className="nombre" title={f}>
                {nombreDe(f)}
              </span>
              <button className="discreto" onClick={() => alQuitar(f)}>
                Quitar
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

/** Panel de resultado, con lo que se puede hacer con él. */
export function PanelResultado({
  texto,
  aviso,
  exito,
  nombreSugerido,
  alGuardar,
  extra,
}: {
  texto: string;
  aviso?: string;
  exito?: string;
  nombreSugerido: string;
  alGuardar: (donde: string) => void;
  extra?: React.ReactNode;
}) {
  const [copiado, setCopiado] = useState(false);
  const [error, setError] = useState("");
  const caja = useRef<HTMLDivElement>(null);

  // El resultado aparece debajo del formulario, no en otra pantalla. Sin llevar
  // la vista hasta él, lo que se ve es el formulario de siempre y hay que
  // adivinar que abajo ha pasado algo.
  useEffect(() => {
    caja.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [texto]);

  async function copiar() {
    try {
      await navigator.clipboard.writeText(texto);
      setCopiado(true);
      setTimeout(() => setCopiado(false), 2500);
    } catch {
      setError("No he podido usar el portapapeles; guárdalo en un fichero");
    }
  }

  async function guardar() {
    try {
      const donde = await esfinge.guardarTexto(nombreSugerido, texto);
      if (donde) alGuardar(donde);
    } catch (e) {
      setError(String(e instanceof Error ? e.message : e));
    }
  }

  return (
    <div className="bloque-resultado" ref={caja}>
      <div className="resultado seleccionable">{texto}</div>
      {exito && <p className="exito">{exito}</p>}
      {copiado && <p className="exito">Copiado al portapapeles</p>}
      {aviso && <p className="aviso">{aviso}</p>}
      {error && <p className="error">{error}</p>}
      <div className="botones">
        <button className="principal" onClick={copiar}>
          Copiar
        </button>
        <button onClick={guardar}>Guardar como…</button>
        {extra}
      </div>
    </div>
  );
}

/** Barra de progreso de una tanda. */
export function Progreso({ hechos, total, actual }: { hechos: number; total: number; actual: string }) {
  const porcentaje = total > 0 ? Math.round((hechos / total) * 100) : 0;
  return (
    <div>
      <div className="barra-progreso">
        <i style={{ width: `${porcentaje}%` }} />
      </div>
      <p className="nota" style={{ marginTop: 6 }}>
        {hechos} de {total}
        {actual ? ` · ${actual}` : ""}
      </p>
    </div>
  );
}

/** nombreDe se queda con el nombre del fichero, sin la carpeta. */
export function nombreDe(ruta: string): string {
  const trozos = ruta.split(/[/\\]/);
  return trozos[trozos.length - 1] || ruta;
}

/** usaMontado evita tocar el estado de un componente que ya no está. */
export function usaMontado() {
  const montado = useRef(true);
  useEffect(() => {
    montado.current = true;
    return () => {
      montado.current = false;
    };
  }, []);
  return montado;
}

/**
 * BandaNovedad es el aviso de que hay una versión nueva.
 *
 * Va entre la barra y el contenido, no dentro del panel: dentro se repetiría en
 * las cuatro pestañas y se mezclaría con los avisos de lo que se está cifrando,
 * que son de otra cosa. Y no se llama «aviso» porque esa clase ya es la de los
 * avisos del propio trabajo.
 *
 * Los tres estados son el mismo sitio contando tres momentos: hay algo nuevo,
 * se está bajando, está listo para instalar.
 */
export function BandaNovedad({
  novedad,
  avance,
  error,
  instalando,
  alDescargar,
  alInstalar,
  alCerrar,
}: {
  novedad: Novedad;
  avance?: Avance;
  error?: string;
  instalando?: boolean;
  alDescargar: () => void;
  alInstalar: () => void;
  alCerrar: () => void;
}) {
  const bajando = avance !== undefined && !avance.hecho;
  const lista = avance?.hecho === true;
  // Donde Esfinge puede reemplazarse sola no hay nada que arrastrar, y el botón
  // no puede prometer lo contrario.
  const sola = novedad.comoSeInstala === "sola";

  return (
    <div className="novedad" role="status">
      <div className="dice">
        {error ? (
          <span className="error">{error}</span>
        ) : instalando ? (
          <span>Instalando Esfinge {novedad.version}. La ventana se cerrará y volverá a abrirse…</span>
        ) : lista ? (
          <span>
            {sola
              ? `Esfinge ${novedad.version} está lista. Se instalará y volverá a abrirse.`
              : `Esfinge ${novedad.version} está lista para instalarse.`}
          </span>
        ) : bajando ? (
          <span>
            Descargando Esfinge {novedad.version}
            {avance.total > 0 && ` · ${Math.round((avance.bytes / avance.total) * 100)} %`}
          </span>
        ) : (
          <span>
            Hay una versión nueva: <strong>Esfinge {novedad.version}</strong>
          </span>
        )}

        {bajando && (
          <div className="barra-progreso">
            <i style={{ width: anchoDe(avance) }} />
          </div>
        )}
      </div>

      <div className="botones">
        {instalando ? null : lista ? (
          <button className="principal" onClick={alInstalar}>
            {sola ? "Instalar y reiniciar" : "Abrir el instalador"}
          </button>
        ) : (
          !bajando && (
            <>
              {novedad.fichero ? (
                <button className="principal" onClick={alDescargar}>
                  Descargar
                </button>
              ) : (
                <a className="discreto" href={novedad.pagina} target="_blank" rel="noreferrer">
                  Ver la publicación
                </a>
              )}
              <button className="discreto" onClick={alCerrar}>
                Ahora no
              </button>
            </>
          )
        )}
      </div>
    </div>
  );
}

/**
 * anchoDe evita la barra vacía cuando todavía no se sabe el tamaño total: sin
 * esto, una descarga sin «Content-Length» se vería como una barra que no avanza,
 * que se lee como que algo va mal.
 */
function anchoDe(a: Avance): string {
  if (a.total <= 0) return "100%";
  return `${Math.min(100, (a.bytes / a.total) * 100)}%`;
}
