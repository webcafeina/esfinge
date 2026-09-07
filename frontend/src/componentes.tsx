import { useEffect, useRef, useState } from "react";
import { esfinge, type Avance, type Fuerza, type Novedad } from "./puente";

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
}: {
  valor: string;
  alCambiar: (v: string) => void;
  etiqueta?: string;
  medir?: boolean;
  alEnviar?: () => void;
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
      <label htmlFor="clave">{etiqueta}</label>
      <input
        id="clave"
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
