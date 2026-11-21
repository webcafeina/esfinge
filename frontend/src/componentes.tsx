import { useEffect, useRef, useState, type ReactNode } from "react";
import { esfinge, ventana, type Avance, type Fuerza, type Novedad } from "./puente";
import { ICONOS_FLUENT } from "./iconos-fluent";
import { esWindows } from "./plataforma";

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
  compacto = false,
}: {
  valor: T;
  alCambiar: (v: T) => void;
  version: string;
  /** null: no hay bóveda. */
  bovedaAbierta?: boolean | null;
  /** El proyecto abierto (ADR 0050); vacío es la bóveda personal. */
  nombreDelProyecto?: string;
  /**
   * El panel está plegado y de cada fila solo se ve su icono.
   *
   * Lo único que cambia aquí es que **el nombre pasa al `title`**: esconder el
   * rótulo lo hace el CSS, que es quien sabe en qué sistema se pliega. Sin esto,
   * una columna de ocho dibujos no dice adónde lleva ninguno.
   */
  compacto?: boolean;
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
            : compacto
              ? etiqueta
              : undefined
        }
      >
        <Icono nombre={v} />
        {/* El rótulo y, debajo, el proyecto. **El nombre accesible del botón sigue
            siendo «Bóveda»** —de ahí el `aria-hidden`—: hay pruebas que localizan
            la sección por él, y el nombre del proyecto ya lo dice el `title`. */}
        <span className="rotulo">
          {etiqueta}
          {proyecto && (
            <span className="proyecto-activo" aria-hidden="true">
              {nombreDelProyecto}
            </span>
          )}
        </span>
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

      {/* **Al pie va el número de versión y nada más** (ADR 0065). Aquí estaban el
          lockup de Esfinge arriba y la firma «Webcafeína ▍ versión» abajo; el cliente
          los vio en la galería del 2026-11-19 y los quitó de los cuatro sistemas. Lo
          que se queda es lo único de ese pie que sirve para algo: **la versión es lo
          primero que se pregunta cuando algo va raro**, y en Ajustes hay que ir a
          buscarla. La marca sigue en el icono de la ventana, en las portadas y en el
          historial vacío. */}
      <p className="version">{version || "…"}</p>
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
  // **El de MCP es el oficial, y es el único que no se dibuja aquí.** Lo pidió el
  // cliente tras ver el nuestro: un glifo inventado para un protocolo no lo reconoce
  // nadie, y el parecido razonable no basta — o es el símbolo o es ruido. Se le dijo
  // lo que arrastra: es marca de la Linux Foundation, y eso lo decide quien encarga.
  //
  // Va aparte porque **es relleno y no trazo**, así que no cabe en el `<svg>` común:
  // metido ahí con `fill="none"` no se vería nada. Y lleva su propio `viewBox`, que
  // es el del original: reescalarlo a mano es la forma de romper una curva.
  if (nombre === "mcp") {
    return (
      <svg className="icono" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <path d="M13.85 0a4.16 4.16 0 0 0-2.95 1.217L1.456 10.66a.835.835 0 0 0 0 1.18.835.835 0 0 0 1.18 0l9.442-9.442a2.49 2.49 0 0 1 3.541 0 2.49 2.49 0 0 1 0 3.541L8.59 12.97l-.1.1a.835.835 0 0 0 0 1.18.835.835 0 0 0 1.18 0l.1-.098 7.03-7.034a2.49 2.49 0 0 1 3.542 0l.049.05a2.49 2.49 0 0 1 0 3.54l-8.54 8.54a1.96 1.96 0 0 0 0 2.755l1.753 1.753a.835.835 0 0 0 1.18 0 .835.835 0 0 0 0-1.18l-1.753-1.753a.266.266 0 0 1 0-.394l8.54-8.54a4.185 4.185 0 0 0 0-5.9l-.05-.05a4.16 4.16 0 0 0-2.95-1.218c-.2 0-.401.02-.6.048a4.17 4.17 0 0 0-1.17-3.552A4.16 4.16 0 0 0 13.85 0m0 3.333a.84.84 0 0 0-.59.245L6.275 10.56a4.186 4.186 0 0 0 0 5.902 4.186 4.186 0 0 0 5.902 0L19.16 9.48a.835.835 0 0 0 0-1.18.835.835 0 0 0-1.18 0l-6.985 6.984a2.49 2.49 0 0 1-3.54 0 2.49 2.49 0 0 1 0-3.54l6.983-6.985a.835.835 0 0 0 0-1.18.84.84 0 0 0-.59-.245" />
      </svg>
    );
  }

  // **Los de Windows, que son los de Fluent** (ADR 0064). Se consulta el sistema
  // al dibujar y no al cargar el módulo, y eso es correcto justamente porque
  // `main.tsx` resuelve la plataforma antes del primer pintado: cuando esto corre,
  // el atributo ya está puesto.
  //
  // Son **de relleno** y llevan **su propio viewBox**, así que no caben en el
  // `<svg>` común de abajo: metidos ahí con `fill="none"` no se vería nada, y
  // reescalarlos a mano es la forma de romper una curva. Cada trazado va en su
  // `<path>`, tal como vino: no se juntan (ver el cosechador).
  const fluent = esWindows() ? ICONOS_FLUENT[nombre] : undefined;
  if (fluent) {
    // El tamaño pintado lo decide el CSS (`--icono`); aquí se elige **el dibujo
    // más cercano**, que no es el mismo reescalado: Fluent dibuja el de 16 y el de
    // 20 por separado, y eso es la mitad de por qué sus iconos se ven nítidos.
    const caja = fluent["16"] ? "16" : "20";
    return (
      <svg
        className="icono"
        viewBox={`0 0 ${caja} ${caja}`}
        fill="currentColor"
        aria-hidden="true"
      >
        {fluent[caja].map((d, i) => (
          <path key={i} d={d} />
        ))}
      </svg>
    );
  }

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
    // Un círculo con media cara en sombra: el glifo del aspecto, el mismo que usa
    // Ajustes del Sistema para claro y oscuro. La media luna va rellena a propósito
    // —es la única forma de que a 16 px se lea como una mitad y no como un gajo—
    // aunque los demás glifos de la casa sean de trazo.
    tema: (
      <>
        <circle cx="9" cy="9" r="6" />
        <path d="M9 3a6 6 0 0 1 0 12z" fill="currentColor" stroke="none" />
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
    // Un servidor: dos bandejas apiladas, cada una con su piloto. Es la forma con la
    // que se dibuja una máquina sin pantalla desde siempre.
    //
    // **Lo que lo hace reconocible a dieciséis píxeles son las dos bandejas**, no los
    // pilotos: mirando la captura, un punto de 1,4 al lado de un trazo de 1,4 casi no
    // está. Se quedan porque a tamaños mayores son lo que lo separa de una lista, y
    // porque es lo que hace el icono del navegador de aquí al lado.
    servidor: (
      <>
        <rect x="2.5" y="3.5" width="13" height="5" rx="1.5" />
        <rect x="2.5" y="9.5" width="13" height="5" rx="1.5" />
        <path d="M5 6h0.01M5 12h0.01" />
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
  envuelve,
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
  /**
   * Deja que la fila se parta en dos antes que perder un rótulo.
   *
   * **Es la salida cuando `compacto` no vale**, y lo que decide cuál usar es qué se
   * pierde al quitar el rótulo: una llave o una tarjeta se adivinan por su glifo;
   * «Cuenta» y «Bóveda» no.
   */
  envuelve?: boolean;
}) {
  const clases = ["segmentado"];
  if (conIconos) clases.push("con-iconos");
  if (compacto) clases.push("compacto");
  if (envuelve) clases.push("envuelve");
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
      <label htmlFor={id}>{etiqueta}</label>

      {/* El ojo va **dentro del campo**, a la derecha, que es donde lo pone todo
          el mundo y donde se busca sin pensar. Fuera competía con «Generar una»
          por el mismo sitio y parecía otra acción del formulario, cuando no es
          una acción: es una propiedad de lo que se está mirando.
          **Y desde el 2026-11-20 la llave está al lado, dentro también**, por lo
          mismo y porque lo pidió el cliente: como botón de texto encima del campo
          competía con el rótulo y pesaba más que la acción de la pantalla. Lleva
          su nombre accesible, igual que el ojo: un dibujo solo no dice qué hace. */}
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
        {alGenerar && (
          <button
            className="ojo llave"
            onClick={alGenerar}
            aria-label="Generar una clave"
            title="Generar una clave"
          >
            {/* **`credencial` y no `llave`**: aquí la llave es la de una
                contraseña —tumbada, con su ojo y su paletón— y `llave` es la de
                acceso, que es otra cosa y otra clase de la bóveda. Lo dice el
                comentario del propio catálogo. */}
            <Icono nombre="credencial" />
          </button>
        )}
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
  // Y donde no la instala nadie —la AppImage— las palabras del instalador son falsas:
  // lo que se baja **es la aplicación**, así que «abrir el instalador» abriría una
  // segunda Esfinge y la que se está usando seguiría siendo la vieja.
  const aMano = novedad.comoSeInstala === "a mano";
  // Y a quien lo instaló con su gestor de paquetes no se le ofrece nada que bajar: en Arch
  // porque el `.deb` publicado no se instala con `pacman -U`, y en Chocolatey y winget porque
  // el paquete **es** este mismo instalador y adelantarnos dejaría al gestor viendo una
  // versión vieja. La banda informa y se calla.
  //
  // **Lo decide Go**, que es quien sabe de qué gestor viene esta copia. Antes se deducía aquí
  // —«no hay fichero y el gestor es pacman»— y eso es la misma regla escrita en dos sitios,
  // que en este proyecto ya ha costado seis fallos: el día que apareció un gestor con fichero
  // publicado, esta copia habría contestado que no.
  const loActualizaSuGestor = novedad.loActualizaSuGestor;
  // Y en Linux, instalarla **saca el diálogo de la contraseña del sistema**. Se dice antes de
  // que aparezca, que es la misma regla del llavero en macOS: un diálogo que pide la
  // contraseña sin avisar invita a cancelarlo, y cancelarlo deja la actualización sin hacer.
  const pedira = novedad.pideAutorizacion;

  return (
    <div className="novedad" role="status">
      <div className="dice">
        {error ? (
          <span className="error">{error}</span>
        ) : instalando ? (
          <span>
            Instalando Esfinge {novedad.version}.{" "}
            {pedira && "Tu sistema te pedirá la contraseña. "}
            La ventana se cerrará y volverá a abrirse…
          </span>
        ) : lista ? (
          <span>
            {sola
              ? pedira
                ? `Esfinge ${novedad.version} está lista. Tu sistema te pedirá la contraseña para instalarla, y después la ventana volverá a abrirse.`
                : `Esfinge ${novedad.version} está lista. Se instalará y volverá a abrirse.`
              : aMano
                ? `Esfinge ${novedad.version} está descargada. Reemplaza con ella la copia que usas.`
                : `Esfinge ${novedad.version} está lista para instalarse.`}
          </span>
        ) : bajando ? (
          <span>
            Descargando Esfinge {novedad.version}
            {avance.total > 0 && ` · ${Math.round((avance.bytes / avance.total) * 100)} %`}
          </span>
        ) : loActualizaSuGestor ? (
          <span>
            Hay una versión nueva, <strong>Esfinge {novedad.version}</strong>: la instala tu gestor
            de paquetes.
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
            {sola ? "Instalar y reiniciar" : aMano ? "Abrir la carpeta" : "Abrir el instalador"}
          </button>
        ) : (
          !bajando && (
            <>
              {novedad.fichero ? (
                <button className="principal" onClick={alDescargar}>
                  Descargar
                </button>
              ) : loActualizaSuGestor ? (
                // Nada que pulsar: ni descarga —no hay fichero— ni «ver la publicación», que
                // mandaría a una página llena de ficheros que no debe instalar.
                null
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

/**
 * Un interruptor: el control de un ajuste, dibujado entero.
 *
 * **El pomo es un hermano y no un `::before`.** En WebKit un `<input>` no pinta
 * pseudoelementos, así que ahí el interruptor saldría sin la bolita **en la
 * ventana de macOS** mientras aquí, en Chromium, se vería perfecto: o sea el
 * fallo que solo aparece en la máquina del cliente. Lleva `aria-hidden` porque
 * no es información, es la pieza que se mueve.
 *
 * **El marcado es el mismo en los cuatro sistemas y lo que cambia es el CSS**:
 * la casilla cuadrada de 18 px en la base y la píldora de 40 × 20 de Fluent en
 * Windows. Y **sigue siendo una casilla**, no un `role="switch"`: cambiarlo
 * renombraría el control para quien usa un lector de pantalla y para las
 * pruebas, que lo buscan por `getByRole("checkbox")`.
 */
export function Interruptor({
  id,
  puesto,
  trabajando,
  alCambiar,
  rotulo,
}: {
  id?: string;
  puesto: boolean;
  trabajando?: boolean;
  alCambiar: (v: boolean) => void;
  /** Para los que no tienen rótulo al lado porque lo dice el título del bloque. */
  rotulo?: string;
}) {
  return (
    <span className="interruptor">
      <input
        type="checkbox"
        id={id}
        checked={puesto}
        disabled={trabajando}
        onChange={(e) => alCambiar(e.target.checked)}
        aria-label={rotulo}
      />
      <span className="pomo" aria-hidden="true" />
    </span>
  );
}

/**
 * Una tarjeta de ajuste: qué se decide, qué cuesta y con qué se cambia.
 *
 * Es la anatomía de la SettingsCard de WinUI (ADR 0066) y de la fila agrupada de
 * los Ajustes del Sistema de macOS: **el texto a un lado y el control al otro**.
 * Aquí hay una sola forma en el marcado y una gramática por sistema en el CSS,
 * que es lo que impide que mantener cuatro aspectos sea mantener cuatro
 * interfaces.
 *
 * **La nota va fuera del `label`**, y eso no es cosmético: el nombre accesible
 * del control es lo que dice el rótulo y nada más. Metiéndola dentro, el control
 * pasaría a llamarse «Descargar el icono de cada sitio de la bóveda Una de las
 * dos salidas a internet…» y dejarían de encontrarlo tanto las pruebas como
 * quien lo busca con un lector de pantalla.
 *
 * **No es para cualquier casilla**: una casilla dentro de un formulario —«La he
 * apuntado en un sitio seguro»— sigue siendo una casilla en los cuatro sistemas,
 * y ésas se quedan en `.fila-ajuste`.
 */
export function TarjetaAjuste({
  id,
  titulo,
  nota,
  puesto,
  trabajando,
  alCambiar,
  control,
  sangrada,
  avisa,
}: {
  id: string;
  titulo: ReactNode;
  nota?: ReactNode;
  puesto?: boolean;
  trabajando?: boolean;
  alCambiar?: (v: boolean) => void;
  /** Cuando lo que cambia el ajuste no es un interruptor, sino un desplegable. */
  control?: ReactNode;
  /** Cuelga de la de arriba: es una cosa más sobre algo que ya está decidido. */
  sangrada?: boolean;
  /** La nota cuenta un riesgo que ya está corriendo, no lo que pasaría. */
  avisa?: boolean;
}) {
  return (
    <div
      className={["ajuste", control ? "con-control" : "", sangrada ? "sangrada" : ""]
        .filter(Boolean)
        .join(" ")}
    >
      <label htmlFor={id}>{titulo}</label>
      {control ?? (
        <Interruptor id={id} puesto={puesto ?? false} trabajando={trabajando} alCambiar={alCambiar ?? (() => {})} />
      )}
      {nota ? <p className={avisa ? "aviso" : "nota"}>{nota}</p> : null}
    </div>
  );
}

/**
 * La barra de título de la ventana, cuando la dibuja Esfinge (ADR 0066).
 *
 * # Por qué está siempre en el marcado
 *
 * Se dibuja en los cuatro sistemas y **la esconde el CSS** donde la barra la
 * pinta el sistema. Lo contrario —montarla solo si el marco es nuestro— parece
 * lo evidente y rompe dos cosas: la regla de la ADR 0064 de que el marcado es el
 * mismo en todos los sistemas, y, más caro en el día a día, **la simulación por
 * URL**, que cambia de aspecto sin recargar nada. Escondida con `display: none`
 * no está en el árbol de accesibilidad, no entra en el orden de tabulación y
 * ninguna prueba la encuentra, así que donde no se usa no existe.
 *
 * # Y por qué el doble clic va a mano
 *
 * Wails no lo trae: no hay un solo `dblclick` ni en su runtime ni en su capa de
 * Windows —mirado en el fuente, no deducido—. Lo que sí hace es **no empezar a
 * arrastrar cuando hay más de un clic** (`e.detail !== 1`), que es justo el
 * hueco por donde esto cabe.
 */
export function BarraDeTitulo({
  panelAbierto,
  alMenu,
}: {
  /** Para decir en el botón si lo que va a hacer es abrir o cerrar el panel. */
  panelAbierto?: boolean;
  alMenu?: () => void;
}) {
  const [maximizada, setMaximizada] = useState(false);

  /**
   * Cómo está el botón de maximizar, **dicho desde fuera**.
   *
   * Sobre ese botón hay una ventana invisible de Windows, que es lo único que
   * hace salir el menú de Snap (ADR 0066), y se queda con el ratón: desde que
   * está ahí **la página no se entera de que alguien pasa por encima**, así que
   * el `:hover` del CSS no se dispara nunca y el botón se quedaría muerto
   * mientras los otros dos se iluminan. Lo dice Go, que es quien lo sabe.
   *
   * Fuera de Windows ese aviso no llega nunca y esto se queda en «fuera», que
   * es lo correcto: allí manda el `:hover` de siempre.
   */
  const [elMaximizar, setElMaximizar] = useState<"fuera" | "encima" | "pulsado">("fuera");
  useEffect(() => esfinge.alCambiarElBotonDeMaximizar(setElMaximizar), []);

  // Se le pregunta a la ventana en vez de llevar la cuenta: maximizar también se
  // hace arrastrando a la parte de arriba y con el teclado, y una cuenta propia
  // se desalinearía con el sistema sin que nada avisara.
  const mirar = () => void ventana.estaMaximizada().then(setMaximizada);
  useEffect(() => {
    mirar();
    window.addEventListener("resize", mirar);
    return () => window.removeEventListener("resize", mirar);
  }, []);

  const alternar = () => {
    ventana.alternarMaximizada();
    mirar();
  };

  return (
    <div className="barra-titulo" onDoubleClick={alternar}>
      {/* **La hamburguesa va aquí y no encima del panel**, que es lo que une las
          dos cosas: en una ventana con NavigationView, Windows pone la barra de
          título alta —48 y no 32— y el botón que pliega el panel dentro de ella,
          alineado con la columna de iconos. Con el botón encima del panel, la
          barra y el panel se leen como dos piezas pegadas en vez de una.
          Lleva `no-drag` como los de la derecha: si no, pulsarlo arrastraría. */}
      {alMenu && (
        <button
          type="button"
          className="boton-menu"
          onClick={alMenu}
          aria-expanded={panelAbierto ?? true}
          aria-label={panelAbierto ? "Plegar el menú" : "Desplegar el menú"}
          title={panelAbierto ? "Plegar el menú" : "Desplegar el menú"}
        >
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path d="M1.5 4h13M1.5 8h13M1.5 12h13" stroke="currentColor" fill="none" />
          </svg>
        </button>
      )}

      {/* El nombre de la ventana, que es lo que dibujaría el sistema si no lo
          dibujáramos nosotros. No es la marca volviendo al cromo (ADR 0065): es
          el título de la ventana, y en Windows lo lleva cualquier aplicación. */}
      <span className="titulo-ventana">Esfinge</span>

      <div className="botones-ventana">
        <button type="button" aria-label="Minimizar la ventana" onClick={ventana.minimizar}>
          <svg viewBox="0 0 10 10" aria-hidden="true">
            <path d="M0 5h10" stroke="currentColor" fill="none" />
          </svg>
        </button>
        <button
          type="button"
          aria-label={maximizada ? "Restaurar la ventana" : "Maximizar la ventana"}
          onClick={alternar}
          data-estado={elMaximizar === "fuera" ? undefined : elMaximizar}
        >
          {/* Restaurar lleva dos marcos escalonados, como en Windows: el de
              detrás abierto por abajo, que es lo que lo distingue del cuadrado
              de maximizar a este tamaño. */}
          <svg viewBox="0 0 10 10" aria-hidden="true">
            {maximizada ? (
              <path d="M2.5 2.5V0.5h7v7h-2M0.5 2.5h7v7h-7z" stroke="currentColor" fill="none" />
            ) : (
              <path d="M0.5 0.5h9v9h-9z" stroke="currentColor" fill="none" />
            )}
          </svg>
        </button>
        <button
          type="button"
          className="cerrar"
          aria-label="Cerrar la ventana"
          onClick={ventana.cerrar}
        >
          <svg viewBox="0 0 10 10" aria-hidden="true">
            <path d="M0.5 0.5l9 9M9.5 0.5l-9 9" stroke="currentColor" fill="none" />
          </svg>
        </button>
      </div>
    </div>
  );
}

/**
 * Si la ventana da de sí para tener el panel de navegación abierto al lado.
 *
 * 860 px no es un número redondo: es **donde la columna de contenido deja de
 * crecer**. Por debajo, abrir el panel le quita sitio a lo que se está
 * haciendo; por encima, cabe de sobra. La misma medida que ya manda en el resto
 * de la hoja.
 *
 * Se suscribe al medio y no se mira una vez: la ventana se redimensiona, y una
 * lectura al montar deja el panel decidido con el tamaño de entonces.
 */
export function laVentanaEsAncha(): boolean {
  const medio = "(min-width: 860px)";
  const [ancha, setAncha] = useState(() => window.matchMedia(medio).matches);
  useEffect(() => {
    const mq = window.matchMedia(medio);
    const alCambiar = () => setAncha(mq.matches);
    mq.addEventListener("change", alCambiar);
    alCambiar();
    return () => mq.removeEventListener("change", alCambiar);
  }, []);
  return ancha;
}
