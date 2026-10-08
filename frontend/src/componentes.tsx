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
// El monograma —inicial y tinte de cada sitio— vive en un módulo sin React desde
// la 2.20.0, porque **lo usa también el panel de la extensión** y tiene que dar el
// mismo color en los dos sitios. Se reexporta para que nada más cambie aquí.
import { dominioDe, inicialDe, tinteDe } from "./monograma";
export { dominioDe, inicialDe, tinteDe };

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
 *   de pruebas, y hay una prueba que cuenta cuántos hay. **Son siete desde que
 *   están las bóvedas de proyecto** (ADR 0050); seis desde la bóveda.
 *
 * **La fila de la bóveda dice si está abierta o cerrada** con un candado pequeño
 * a la derecha, apagado como los iconos, que lo pidió el cliente con la 2.24.3.
 * Va marcado como decorativo y el estado se dice en el `title`: si entrara en el
 * nombre accesible, la sección dejaría de llamarse «Bóveda» para quien la busca
 * por su nombre —las pruebas, y quien usa un lector de pantalla—. Sin bóveda
 * creada no sale nada: no hay nada que esté abierto ni cerrado.
 */
export function BarraLateral<T extends string>({
  valor,
  alCambiar,
  version,
  bovedaAbierta,
  nombreDelProyecto = "",
}: {
  valor: T;
  alCambiar: (v: T) => void;
  version: string;
  /** null: no hay bóveda. */
  bovedaAbierta?: boolean | null;
  /** El proyecto abierto (ADR 0050); vacío es la bóveda personal. */
  nombreDelProyecto?: string;
}) {
  const fila = (v: string, etiqueta: string) => {
    const candado = v === "boveda" && bovedaAbierta != null;
    // **Cuál** es la bóveda abierta, también desde la izquierda: con el proyecto
    // dicho solo en la barra de herramientas, la fila seguía poniendo «Bóveda» y
    // la pantalla se leía como la personal. Lo vio el cliente con la 2.38.0.
    const proyecto = v === "boveda" && nombreDelProyecto !== "";
    return (
      <button
        key={v}
        onClick={() => alCambiar(v as T)}
        aria-current={v === valor ? "page" : undefined}
        className={proyecto ? "con-proyecto" : undefined}
        title={
          candado
            ? bovedaAbierta
              ? proyecto
                ? `La bóveda de «${nombreDelProyecto}» está abierta`
                : "La bóveda está abierta"
              : "La bóveda está cerrada"
            : undefined
        }
      >
        <Icono nombre={v} />
        {/* El rótulo y, debajo, el proyecto. **El nombre accesible del botón sigue
            siendo «Bóveda»** —de ahí el `aria-hidden`—: hay pruebas que localizan
            la sección por él, y el nombre del proyecto ya lo dice el `title`. */}
        {proyecto ? (
          <span className="rotulo">
            {etiqueta}
            <span className="proyecto-activo" aria-hidden="true">
              {nombreDelProyecto}
            </span>
          </span>
        ) : (
          etiqueta
        )}
        {candado && (
          <span className="estado-boveda" data-abierta={bovedaAbierta ? "si" : "no"} aria-hidden="true">
            <Icono nombre={bovedaAbierta ? "candado-abierto" : "candado-cerrado"} />
          </span>
        )}
      </button>
    );
  };

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
        {fila("boveda", "Bóveda")}
        {fila("proyectos", "Proyectos")}
        {fila("historial", "Historial")}
      </nav>

      {/* **MCP va encima de Ajustes y lleva al mismo panel.** Lo pidió así el cliente
          el 2026-10-08, después de que instalarlo en Claude Code fallara por una
          bandera que faltaba: lo que no se encuentra no se usa, y quien quiere
          conectar su agente no se le ocurre mirar en Ajustes. Las dos filas abren la
          misma pantalla y **cada una se queda marcada**, que es lo que dice por dónde
          se entró; si no, el clic parecería no haber hecho nada. */}
      <nav className="abajo" aria-label="Configuración">
        {fila("mcp", "MCP")}
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
export function Icono({ nombre }: { nombre: string }) {
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
    // Los candados del estado de la bóveda, en la barra lateral. No son los de
    // «Cifrar» y «Descifrar»: a trece píxeles aquel arco apenas movido no se
    // distinguía del cerrado. Aquí el cuerpo es más bajo y, abierto, el arco sube y
    // se separa del todo: se lee sin tener que buscar la diferencia.
    "candado-cerrado": (
      <>
        <rect x="3.5" y="8" width="11" height="7.5" rx="2" />
        <path d="M6 8V5.5a3 3 0 0 1 6 0V8" />
      </>
    ),
    "candado-abierto": (
      <>
        <rect x="3.5" y="8" width="11" height="7.5" rx="2" />
        <path d="M12 8V4a3 3 0 0 0-6 0v.6" />
      </>
    ),
    // Chispa: lo que se genera sale de la nada.
    generar: (
      <path d="M9 2.5v13M2.5 9h13M4.6 4.6l8.8 8.8M13.4 4.6l-8.8 8.8" />
    ),
    // Una caja fuerte: el cuerpo, el disco y la manilla.
    boveda: (
      <>
        <rect x="2.8" y="3.6" width="12.4" height="10.8" rx="2" />
        <circle cx="8.2" cy="9" r="2.4" />
        <path d="M12.8 7.7v2.6" />
      </>
    ),
    // Una carpeta, con el canto de otra asomando por detrás.
    //
    // **Se dibujaron dos enteras primero y a 18 px no se leían**: dos solapas y dos
    // cuerpos a este tamaño son un borrón, y hubo que ampliarlo a cuatro aumentos
    // para ver que lo que se entendía era «una carpeta» a secas. Así que queda una,
    // que es lo que se lee, y el canto de detrás basta para que no sea la carpeta
    // genérica de guardar cosas. Lo demás lo dice el rótulo de al lado.
    proyectos: (
      <>
        <path d="M2.6 6.2h4l1.2 1.4h5.6v6.2a1.4 1.4 0 0 1-1.4 1.4H4a1.4 1.4 0 0 1-1.4-1.4V6.2Z" />
        <path d="M5 4.4h3.4l1 1.2h5" />
      </>
    ),
    // Reloj.
    historial: (
      <>
        <circle cx="9" cy="9" r="6.2" />
        <path d="M9 5.4V9l2.6 1.8" />
      </>
    ),
    // Las siete clases de la bóveda, más el «todo» que las junta.
    //
    // **Sustituyen a unos emoji**, que es lo que había y lo que este mismo
    // comentario prohibía cuatro líneas más arriba: son de color, no se tiñen, y
    // cada sistema los dibuja a su manera. Al lado de los seis de la barra
    // lateral se veía enseguida que no eran de la misma familia.
    todo: (
      <>
        <path d="M3 5.5h12M3 9h12M3 12.5h12" />
      </>
    ),
    // Una llave: el ojo y el paletón.
    credencial: (
      <>
        <circle cx="5.6" cy="8.6" r="2.9" />
        <path d="M8.2 7.2 15 4.6M13.2 5.4l1 2.1M11 6.2l.9 2" />
      </>
    ),
    // Una hoja con renglones y la esquina doblada, como el icono del documento.
    nota: (
      <>
        <path d="M4 2.8h6.4L14 6.4v8.8H4Z" />
        <path d="M10.2 3v3.4h3.4" />
        <path d="M6.4 9.4h5M6.4 12h3.4" />
      </>
    ),
    // Una tarjeta: el plástico y su banda.
    tarjeta: (
      <>
        <rect x="2.2" y="4.4" width="13.6" height="9.2" rx="1.8" />
        <path d="M2.2 7.6h13.6" />
        <path d="M5 11h2.6" />
      </>
    ),
    // Un documento con la foto de quien es, que es lo que distingue un carné de
    // una tarjeta: el retrato a un lado y los datos al otro.
    identidad: (
      <>
        <rect x="2.2" y="4" width="13.6" height="10" rx="1.8" />
        <circle cx="6.4" cy="7.9" r="1.5" />
        <path d="M4.2 11.6c.5-1.1 1.3-1.6 2.2-1.6s1.7.5 2.2 1.6" />
        <path d="M11 7.6h2.8M11 10.4h2.8" />
      </>
    ),
    // Un dato personal: quien es, sin el carné alrededor. Es lo que lo separa de
    // la identidad, que es el documento y no la persona.
    personal: (
      <>
        <circle cx="9" cy="6.1" r="2.9" />
        <path d="M3.5 15.2c.5-3 2.7-4.7 5.5-4.7s5 1.7 5.5 4.7" />
      </>
    ),
    // Una llave de acceso: la llave vista de frente, con su anilla arriba y los
    // dientes a un lado. **No es la de `credencial`**, que va tumbada y es la de
    // una contraseña: a quince píxeles lo que las separa es la orientación.
    llave: (
      <>
        <circle cx="9" cy="5.2" r="2.7" />
        <path d="M9 7.9v7.3M9 10.8h2.6M9 13.2h1.9" />
      </>
    ),
    // Una red wifi: las tres ondas y el punto, que es el dibujo que entiende todo el
    // mundo y el único de esta tira que no se confunde con nada. Las ondas son arcos
    // concéntricos abiertos hacia arriba, no semicírculos: a quince píxeles un
    // semicírculo se cierra y parece una ceja.
    wifi: (
      <>
        <path d="M2.4 7.3a9.5 9.5 0 0 1 13.2 0" />
        <path d="M4.9 10a6 6 0 0 1 8.2 0" />
        <path d="M7.3 12.6a2.5 2.5 0 0 1 3.4 0" />
        <circle cx="9" cy="14.9" r=".35" fill="currentColor" stroke="none" />
      </>
    ),
    // Los gestores de los que se sabe importar.
    //
    // **Son marcas de otras empresas, y por eso no son sus logotipos.** Nombrar un
    // producto con el que se interopera es legítimo —el nombre está al lado, en
    // texto, y es lo que de verdad identifica— pero calcar su marca en nuestro
    // binario es otra cosa. Éstos son glifos **inspirados en su forma**, en el
    // mismo trazo que todo lo demás: lo justo para distinguir cinco opciones de un
    // vistazo. Y a quince píxeles es además lo único que se lee: una marca real
    // reducida a esta escala se convierte en una mancha.
    //
    // Un escudo, un ojal, tres puntos, un disco partido y una onda.
    bitwarden: (
      <>
        <path d="M9 2.6l5.4 2v5.1c0 3-2.2 4.9-5.4 5.7-3.2-.8-5.4-2.7-5.4-5.7V4.6Z" />
        <path d="M9 5v7.6" />
      </>
    ),
    oneword: (
      <>
        <circle cx="9" cy="9" r="6.3" />
        <circle cx="9" cy="7.4" r="1.7" />
        <path d="M9 9.1v2.6" />
      </>
    ),
    lastpass: (
      <>
        <rect x="2.6" y="4.6" width="12.8" height="8.8" rx="2" />
        <path d="M6 9h.01M9 9h.01M12 9h.01" />
      </>
    ),
    // Los tres radios van **de la rueda interior al borde y ni un píxel más**: al
    // primer intento se salían del círculo y aquello parecía un gráfico de tarta
    // roto, no un disco partido en tres. Se vio ampliándolos, no en la interfaz.
    chrome: (
      <>
        <circle cx="9" cy="9" r="6.3" />
        <circle cx="9" cy="9" r="2.5" />
        <path d="M9 11.5v3.8M6.83 7.75 3.54 5.85M11.17 7.75 14.46 5.85" />
      </>
    ),
    dashlane: (
      <>
        <path d="M4 3.4h4.2c3.5 0 5.8 2.3 5.8 5.6s-2.3 5.6-5.8 5.6H4Z" />
        <path d="M7.2 6.6h1.2c1.5 0 2.5 1 2.5 2.4s-1 2.4-2.5 2.4H7.2Z" />
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
    // Una flecha que baja a una bandeja: lo que llega de fuera.
    descargar: (
      <>
        <path d="M9 3v8.5M5.6 8.1 9 11.5l3.4-3.4" />
        <path d="M3.5 13.5v1a1 1 0 0 0 1 1h9a1 1 0 0 0 1-1v-1" />
      </>
    ),
    // Una ventana de navegador: el marco y su barra.
    navegador: (
      <>
        <rect x="2.5" y="3.5" width="13" height="11" rx="2" />
        <path d="M2.5 7h13" />
        <circle cx="5" cy="5.2" r="0.6" />
      </>
    ),
    // La bóveda de esta sección es la caja fuerte de la barra lateral, y no se
    // repite el dibujo: `boveda` ya está arriba.
    //
    // **MCP, en nuestro trazo y no su logotipo**, por lo mismo que los glifos de los
    // gestores: nombrar un protocolo con el que se interopera es legítimo —el rótulo
    // va al lado— y meter la marca de otro dentro del binario es otra cosa. El
    // símbolo de MCP es marca de la Linux Foundation.
    //
    // Lo que se dibuja es lo que el de ellos cuenta y es lo que el protocolo hace:
    // **dos arcos que convergen en un punto**, el modelo y la herramienta
    // encontrándose. A dieciséis píxeles eso es lo único que se lee de aquella forma.
    mcp: (
      <>
        <path d="M2.5 13.5 8 6.2a2.6 2.6 0 0 1 4 3.3l-3.4 4" />
        <path d="M6.6 4.6a2.6 2.6 0 0 1 4 0l4.9 5.6" />
        <path d="M15.5 10.2 11 15.5" />
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

/**
 * Monograma es el cuadro que identifica una entrada de la bóveda.
 *
 * La inicial del sitio sobre uno de los ocho tintes, elegido por el propio sitio:
 * `banco.es` sale siempre del mismo color, y ese color es lo que permite recorrer
 * sesenta y cinco filas sin leerlas.
 *
 * Va `aria-hidden` porque **no añade nada**: el nombre de la entrada está al lado
 * en texto, y quien lea la pantalla en voz alta no necesita oír una letra suelta
 * antes de cada fila.
 */
export function Monograma({
  sitio,
  titulo,
  icono,
}: {
  sitio?: string;
  titulo: string;
  /** El icono del sitio, si se ha podido traer. Va como URI de datos. */
  icono?: string;
}) {
  // **El icono se pinta con una URI de datos y nunca con la dirección del sitio.**
  // Un `<img src="https://…">` haría que el propio webview saliera a internet, sin
  // pasar por el filtro de direcciones privadas ni por los topes de tamaño, y con
  // las cabeceras que le diera la gana. Los bytes los trae Go, ya comprobados.
  if (icono) {
    return <img className="monograma" src={icono} alt="" aria-hidden="true" />;
  }
  return <MonogramaDeLetra sitio={sitio} titulo={titulo} />;
}

function MonogramaDeLetra({ sitio, titulo }: { sitio?: string; titulo: string }) {
  // **La letra sale del nombre y el color del sitio**, y son dos cosas a
  // propósito. El color identifica el sitio y tiene que ser estable: dos entradas
  // del mismo banco salen del mismo color aunque se llamen distinto. La letra, en
  // cambio, tiene que cuadrar con lo que se lee justo al lado: sacándola del
  // dominio, «Hacienda» salía con una «A» —de agenciatributaria.gob.es— y parecía
  // un fallo. Se vio en una captura, no en una prueba.
  const dominio = dominioDe(sitio);
  return (
    <span
      className="monograma"
      data-tinte={tinteDe(dominio || titulo.trim().toLowerCase())}
      aria-hidden="true"
    >
      {inicialDe(titulo)}
    </span>
  );
}

/**
 * Control segmentado, que es como los dos sistemas agrupan modos excluyentes.
 *
 * Con `icono`, cada opción lleva un glifo delante del rótulo. **El rótulo sigue
 * estando en el DOM aunque la ventana lo esconda**: se recorta con `.solo-se-oye`
 * y no con `display: none`, porque `display: none` se lleva por delante el nombre
 * accesible del botón —y con él a quien lee la pantalla en voz alta y a los
 * localizadores de las pruebas, que buscan las pestañas por su nombre—.
 */
export function Segmentado<T extends string>({
  opciones,
  valor,
  alCambiar,
  conIconos,
  compacto,
}: {
  opciones: { valor: T; etiqueta: string; icono?: string }[];
  valor: T;
  alCambiar: (v: T) => void;
  conIconos?: boolean;
  /**
   * Deja que el rótulo se retire cuando la ventana se estrecha.
   *
   * **No va con `conIconos` a la fuerza, y es a propósito.** Las clases de la
   * bóveda se adivinan por su glifo —una llave, una tarjeta— pero los gestores no:
   * cinco marcas ajenas sin su nombre al lado no las reconoce nadie. Ahí el rótulo
   * se queda siempre.
   */
  compacto?: boolean;
}) {
  const clases = ["segmentado"];
  if (conIconos) clases.push("con-iconos");
  if (compacto) clases.push("compacto");
  return (
    <div className={clases.join(" ")} role="tablist">
      {opciones.map((o) => (
        <button
          key={o.valor}
          role="tab"
          aria-selected={o.valor === valor}
          onClick={() => alCambiar(o.valor)}
        >
          {o.icono && <Icono nombre={o.icono} />}
          <span className="rotulo">{o.etiqueta}</span>
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

  // **Tapada de partida y destapada a petición**, nunca al revés: quien teclea
  // una clave con alguien detrás no tiene que acordarse de esconderla primero.
  //
  // No es solo comodidad. De un campo de contraseña el navegador **se niega a
  // copiar** —lo hacen WebKit y Chromium, a propósito y sin avisar—, así que sin
  // esto la clave que acaba de fabricar «Generar una» no se podía sacar de aquí,
  // y es la única que no está apuntada en ningún otro sitio. Copiar de un campo
  // de contraseña se arregló además por su lado, en ordenes.ts.
  const [aLaVista, setALaVista] = useState(false);

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

      {/* El ojo va **dentro del campo**, a la derecha, que es donde lo pone todo
          el mundo y donde se busca sin pensar. Fuera competía con «Generar una»
          por el mismo sitio y parecía otra acción del formulario, cuando no es
          una acción: es una propiedad de lo que se está mirando. */}
      <div className="campo-con-ojo">
        <input
          id={id}
          type={aLaVista ? "text" : "password"}
          value={valor}
          autoComplete="off"
          /* Con la clave a la vista, el corrector y el autocompletado del sistema
             dejan de ser inocentes: los dos leen lo que hay escrito. */
          spellCheck={false}
          autoCorrect="off"
          autoCapitalize="off"
          className={aLaVista ? "a-la-vista" : undefined}
          onChange={(e) => alCambiar(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && alEnviar?.()}
        />
        {/* Con nombre de verdad y no solo un dibujo: quien lea la pantalla en voz
            alta tiene que oír qué hace, y «botón» a secas no lo dice. */}
        <button
          className="ojo"
          onClick={() => setALaVista(!aLaVista)}
          aria-controls={id}
          aria-pressed={aLaVista}
          aria-label={aLaVista ? "Ocultar la clave" : "Ver la clave"}
          title={aLaVista ? "Ocultar la clave" : "Ver la clave"}
        >
          <Ojo tachado={aLaVista} />
        </button>
      </div>

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

/**
 * Ojo dibuja el interruptor de ver la clave: el ojo abierto para destaparla y el
 * ojo tachado cuando ya se ve.
 *
 * El tachado va **por delante y con su propia línea de fondo**, del color del
 * campo: sin esa línea, la barra oblicua se confunde con el contorno del ojo y a
 * 18 px los dos estados se parecen demasiado.
 */
function Ojo({ tachado }: { tachado: boolean }) {
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
      <path d="M1.7 9S4.6 4.3 9 4.3 16.3 9 16.3 9s-2.9 4.7-7.3 4.7S1.7 9 1.7 9Z" />
      <circle cx="9" cy="9" r="2.2" />
      {tachado && (
        <>
          <path d="M3.4 15.2 14.6 2.8" stroke="var(--campo)" strokeWidth="3.2" />
          <path d="M3.4 15.2 14.6 2.8" />
        </>
      )}
    </svg>
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
        {/* Sin «justifyContent»: las notas dejaron de ser contenedores flexibles
            y aquí lo que centra es el «text-align» de la zona de soltar. */}
        <p className="nota" style={{ marginTop: 4 }}>
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
