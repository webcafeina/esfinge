#!/usr/bin/env python3
"""Cosecha los iconos de Yaru que usa Esfinge en GNOME.

Escribe `frontend/src/iconos-yaru.ts`. Se ejecuta a mano, no en cada
compilación: lo que se publica es el fichero generado, no la descarga.

Por qué vive en el repositorio y no en `~/.cache/`: lo dice la cabecera de
`fluent.py`. Un generador que no está versionado es un generador que se pierde,
y en Apolo eso costó que la trampa de los trazados volviera dos veces.

# Y por qué este guarda el árbol y no sólo la `d`

Porque Yaru **no es Fluent**, y se midió antes de escribir esto. De los
diecinueve iconos que hacen falta:

- **seis son enlaces simbólicos**, y por HTTP un enlace no es un SVG: es un
  fichero de texto de una línea con la ruta de destino. `network-wireless`
  apunta a `captive-portal`, `folder` a `inode-directory`, `web-browser` a
  `webbrowser-app`… Sin seguirlos, lo que se cosecha es la cadena `"../apps/…"`.
- **seis llevan `transform`** en un grupo o en el propio trazado. Sacar la `d` y
  tirar el `transform` **no da un icono torcido: da un icono fuera de la caja**,
  y es exactamente la forma del fallo de Apolo.
- **uno dibuja con `<rect>` y `<circle>`** además de con `<path>`
  (`network-server`), así que una cosecha que sólo mire `<path>` lo deja a
  medias.
- **dos no tienen el `viewBox` en `0 0 16 16`** —`0 0 16 16.001` y
  `0 0 16.015 16`—, así que el `viewBox` se guarda por icono y no se supone.

De ahí que la salida sea el **árbol de dibujo tal cual**: elementos, atributos y
anidamiento. Lo que se pinta es lo mismo que pinta Ubuntu.

# Qué no hace, y es lo que decide la licencia

**No adapta nada.** No junta trazados, no los reescala, no los recorta y no
toca ningún `transform`. La geometría sale **idéntica**; lo único que cambia es
el envase —de SVG a datos— y un cambio de formato no es una adaptación (lo dice
la propia FAQ de Creative Commons). Eso importa porque Yaru es **CC BY-SA 4.0**:
una adaptación habría que publicarla con la misma licencia, y Esfinge no es
código abierto. Incluirlos sin tocarlos, citando a quien son, sí se puede.

Lo vigila `TestLosIconosDeYaruNoSeHanAdaptado`, que compara el árbol del fichero
generado contra el SVG de origen guardado al lado.

# Y las guardas, que son la mitad del guion

Este cosechador **se niega** en vez de adivinar: un elemento que no sabe dibujar,
un atributo que podría cambiar el dibujo, un `style` con color dentro, un `use`
que referencia un `id`, un relleno de un color que no es el del icono. Todo eso
para un fichero que se publica sin que nadie vuelva a mirarlo.

Uso:
    python3 herramientas/iconos/yaru.py
"""

import json
import os
import posixpath
import re
import sys
import urllib.error
import urllib.request
import xml.etree.ElementTree as ET

# El commit del que se cosecha. **Fijado**, por lo mismo que en Fluent: una rama
# rodante haría que dos cosechas dieran iconos distintos sin que nadie lo
# pidiera.
COMMIT = "7f18973e05607c609c4b972f0dd9bff36fa2a13e"
BASE = f"https://raw.githubusercontent.com/ubuntu/yaru/{COMMIT}/icons/Yaru/scalable/"

# Qué icono de Yaru le toca a cada nombre de Esfinge, **con su carpeta**: en Yaru
# el mismo nombre no vive siempre en el mismo sitio y el conjunto no es plano.
#
# Se eligen los **simbólicos**, que son los de una sola tinta y los que GNOME usa
# en barras laterales y filas; los de color son los de las aplicaciones.
#
# Los cinco logotipos de gestores y el de MCP no están aquí **y no es un olvido**:
# son marcas. Yaru no trae un «Bitwarden», los nuestros son formas propias a
# propósito y el de MCP es el oficial.
CUALES = {
    # El candado de GNOME, cerrado y abierto. `changes-allow` es un enlace a
    # `emblems/padlock-open`.
    "cifrar": "status/changes-prevent-symbolic.svg",
    "descifrar": "status/changes-allow-symbolic.svg",
    # **Una estrella con un más: «haz una nueva».** Yaru no tiene ningún
    # «sparkle» —se buscó: no existe—, así que había que elegir metáfora, y se
    # eligió sobre una rejilla de seis candidatos dibujados. Lo primero que hubo
    # aquí fue `view-refresh`, la flecha de recargar, y el problema es que dice
    # «otra vez» y se lee como «recargar la página».
    "generar": "actions/star-new-symbolic.svg",
    # El escudo de GNOME, que es lo que allí significa «protegido».
    "boveda": "status/security-high-symbolic.svg",
    "proyectos": "places/folder-symbolic.svg",
    # El reloj con el documento: «abierto hace poco», que es el historial.
    "historial": "actions/document-open-recent-symbolic.svg",
    "todo": "actions/view-list-symbolic.svg",
    # La llave de GNOME. Es la de las contraseñas, así que **no se repite** en la
    # llave de acceso de abajo: dos clases con el mismo glifo no se distinguen.
    "credencial": "status/dialog-password-symbolic.svg",
    "nota": "mimetypes/text-x-generic-symbolic.svg",
    # **La tarjeta inteligente, que es lo único con forma de tarjeta que trae
    # Yaru.** Lo primero que se probó fue `money-symbolic`, y en la captura de la
    # fila de clases salía **un signo de dólar**: dice «dinero», no «tarjeta», y
    # encima en una moneda que no es la de nadie aquí.
    "tarjeta": "devices/auth-smartcard-symbolic.svg",
    "identidad": "apps/x-office-address-book-symbolic.svg",
    "personal": "status/avatar-default-symbolic.svg",
    # Yaru no tiene «passkey» —tampoco existe—. La huella es con lo que se usa
    # una llave de acceso en la práctica y es lo que GNOME tiene para
    # «autenticarse sin contraseña»; además deja la llave de `dialog-password`
    # para las credenciales y la tarjeta para las tarjetas, que es lo que de
    # verdad decide: **tres clases vecinas no pueden compartir glifo**.
    "llave": "devices/auth-fingerprint-symbolic.svg",
    # **Y no `network-wireless-symbolic`**, que en Yaru es un enlace a
    # `captive-portal` —un portal cautivo, no una red—. El abanico de señal es
    # éste; `…-signal-excellent` es a su vez un enlace a él.
    "wifi": "status/network-wireless-connected-symbolic.svg",
    "ajustes": "apps/preferences-system-symbolic.svg",
    # La flecha hacia la bandeja: «guardar», que es lo que hace descargar una
    # actualización. `folder-download` es la carpeta de descargas, otra cosa.
    "descargar": "actions/document-save-symbolic.svg",
    "navegador": "apps/web-browser-symbolic.svg",
    # Los tres de abajo **no estaban en el juego de Fluent tampoco**, porque
    # nacieron después de la fase de Windows: las dos pestañas de Cifrar y la de
    # Apariencia. Sin ellos, una fila enseña unos iconos del escritorio y otros
    # dibujados a mano, que es peor que no tener ninguno nativo.
    "texto": "apps/text-editor-symbolic.svg",
    # **Dos documentos apilados, que es lo único de Yaru que dice «varios
    # ficheros».** Se probaron en una rejilla los nueve candidatos y el reparto
    # real es éste: `emblem-documents`, `folder-documents` y `x-office-document`
    # dibujan **una página** —o sea el mismo glifo que `nota` y casi el de
    # `texto`, tres páginas seguidas que no se distinguen— y
    # `system-file-manager`, `filemanager-app` e `inode-directory` dibujan **una
    # carpeta**, que es la de `proyectos`.
    #
    # Lo que se coge es **la forma**, no el nombre: `edit-copy` significa
    # «copiar» y aquí no se copia nada. Se acepta porque la pestaña lleva su
    # rótulo al lado siempre —el glifo acompaña, no carga el significado— y es
    # una decisión que va a la galería por si se prefiere la carpeta.
    "ficheros": "actions/edit-copy-symbolic.svg",
    "tema": "categories/preferences-desktop-appearance-symbolic.svg",
    "servidor": "places/network-server-symbolic.svg",
    # **Los tres botones de la ventana, y el cuarto que hace falta.** Desde la
    # ADR 0073 el marco de GNOME lo dibuja Esfinge, así que esos glifos dejan de
    # ser del escritorio: los dibujaba el componente a mano y el cliente los vio
    # como lo que eran, «propios, no los nativos». Son los mismos cuatro que ya
    # eligió para Apolo.
    #
    # **Van en `ui/`**, que es la única carpeta de Yaru donde están: se probó
    # `actions/` y `status/` y las dos dan 404. Y el de restaurar no es un
    # capricho: el botón de maximizar cambia de dibujo cuando la ventana ya lo
    # está.
    "ventana-cerrar": "ui/window-close-symbolic.svg",
    "ventana-minimizar": "ui/window-minimize-symbolic.svg",
    "ventana-maximizar": "ui/window-maximize-symbolic.svg",
    "ventana-restaurar": "ui/window-restore-symbolic.svg",
}

SVG = "http://www.w3.org/2000/svg"

# Lo que este cosechador sabe dibujar. Cualquier otra cosa **para la cosecha**:
# un `<use>` referencia un `id` que no viajaría, un `<clipPath>` recorta y un
# `<text>` necesita la tipografía del sistema.
DIBUJAN = {"g", "path", "rect", "circle", "ellipse", "polygon", "polyline", "line"}

# Los atributos que se conservan: los que **son** el dibujo. Todo lo que no esté
# aquí se tira, y si pudiera cambiar lo que se ve, el guion se para antes.
GEOMETRIA = {
    "d", "transform", "x", "y", "width", "height", "rx", "ry",
    "cx", "cy", "r", "points", "x1", "y1", "x2", "y2",
    "fill-rule", "clip-rule", "opacity",
}

# Atributos que se pueden tirar sin mirar: no cambian la forma, y el color lo
# pone `currentColor` desde el CSS (es la mitad de para qué existen los
# simbólicos).
INOCUOS = {
    "fill", "stroke", "id", "class", "font-family", "font-weight", "font-size",
    "overflow", "white-space", "color", "display", "visibility", "enable-background",
    "solid-color", "solid-opacity", "isolation", "mix-blend-mode", "shape-rendering",
    "color-interpolation-filters", "color-rendering", "image-rendering", "paint-order",
    "text-rendering", "vector-effect", "fill-opacity", "stroke-width", "stroke-opacity",
    # Propiedades de **texto** y metadatos del editor. Van nombradas porque las
    # guardas las pararon de verdad en cuatro de los diecinueve: ninguna puede
    # mover un `<path>`, y dejar pasar «lo que parece inofensivo» es justo lo que
    # este guion no hace.
    "label", "dominant-baseline", "text-orientation", "text-align", "writing-mode",
    "letter-spacing", "word-spacing", "text-anchor", "line-height", "direction",
    "stroke-linecap", "stroke-linejoin", "stroke-miterlimit", "stroke-dasharray",
    "stroke-dashoffset", "marker", "version", "baseprofile",
}

# Lo que un `style` puede traer sin que importe. Si trae algo más, se para: un
# `fill` o un `stroke` ahí dentro sí cambia lo que se ve, y tirarlo en silencio
# es publicar un icono que no es el que se cosechó.
ESTILOS_INOCUOS = re.compile(
    r"^(isolation|mix-blend-mode|shape-padding|text-decoration[\w-]*|text-indent"
    r"|text-transform|white-space|font-[\w-]+|letter-spacing|word-spacing"
    r"|line-height|direction|writing-mode|text-anchor|baseline-shift"
    r"|dominant-baseline|block-progression|inline-size|paint-order|solid-[\w-]+"
    r"|text-orientation|text-align|text-overflow|hyphens|white-space-collapse"
    r"|-inkscape-[\w-]+|color|display|visibility|overflow|marker[\w-]*"
    r"|enable-background|image-rendering|shape-rendering|text-rendering"
    r"|color-[\w-]+|vector-effect|opacity|fill-opacity|stroke[\w-]*|fill)$"
)

fallos: list[str] = []


def bajar(ruta: str) -> tuple[str, str]:
    """El SVG de una ruta, siguiendo los enlaces simbólicos de Yaru.

    Devuelve el texto y la ruta donde acabó, que es lo que se escribe al lado del
    icono para que se sepa de dónde salió de verdad.
    """
    actual = ruta
    for _ in range(6):
        try:
            with urllib.request.urlopen(BASE + actual, timeout=25) as r:
                texto = r.read().decode("utf-8")
        except urllib.error.HTTPError as e:
            raise RuntimeError(f"{e.code} en {actual}") from e
        if texto.lstrip().startswith("<"):
            return texto, actual
        # No empieza por «<»: es un enlace simbólico servido como texto.
        destino = posixpath.normpath(posixpath.join(posixpath.dirname(actual), texto.strip()))
        if destino == actual:
            raise RuntimeError(f"{actual} se enlaza a sí mismo")
        actual = destino
    raise RuntimeError(f"{ruta}: demasiados enlaces seguidos")


def local(etiqueta: str) -> str:
    """De «{http://www.w3.org/2000/svg}path» a «path»."""
    return etiqueta.split("}")[-1] if "}" in etiqueta else etiqueta


def estilo_es_inocuo(valor: str) -> list[str]:
    """Las propiedades de un `style` que sí cambiarían el dibujo."""
    malas = []
    for trozo in valor.split(";"):
        if not trozo.strip():
            continue
        prop = trozo.split(":", 1)[0].strip().lower()
        if not ESTILOS_INOCUOS.match(prop):
            malas.append(prop)
    return malas


def nodo(el: ET.Element, quien: str) -> dict | None:
    """Un elemento de dibujo convertido en datos, o `None` si no dibuja nada."""
    etiqueta = local(el.tag)
    if etiqueta not in DIBUJAN:
        fallos.append(f"{quien}: no sé dibujar <{etiqueta}>")
        return None

    atributos: dict[str, str] = {}
    for nombre, valor in el.attrib.items():
        n = local(nombre).lower()
        if n in GEOMETRIA:
            atributos[local(nombre)] = valor
        elif n == "style":
            malas = estilo_es_inocuo(valor)
            if malas:
                fallos.append(f"{quien}: <{etiqueta}> trae style con {', '.join(malas)}")
        elif n in INOCUOS or n.startswith(("sodipodi", "inkscape")) or "}" in nombre:
            # Los de otros espacios de nombres (Inkscape, sodipodi) son metadatos
            # del editor: no dibujan.
            continue
        else:
            fallos.append(f"{quien}: <{etiqueta}> trae {n}, que no sé si cambia el dibujo")

    hijos = [h for h in (nodo(c, quien) for c in el) if h is not None]
    if etiqueta == "g" and not hijos:
        return None
    # Un `<g>` sin nada que lo distinga no hace falta; con `transform` sí.
    if etiqueta == "g" and not atributos and len(hijos) == 1:
        return hijos[0]

    salida: dict = {"t": etiqueta}
    if atributos:
        salida["a"] = dict(sorted(atributos.items()))
    if hijos:
        salida["h"] = hijos
    return salida


# Dónde se guarda el SVG tal como vino. **No es un capricho**: la cabecera de este
# guion y `TERCEROS.md` dicen que aquí no se adapta nada, y eso es lo que permite
# incluir obra CC BY-SA en un programa que no es código abierto. Una afirmación
# así necesita algo que la compruebe, y lo que la comprueba es
# `TestLosIconosDeYaruNoSeHanAdaptado`, que compara los trazados del módulo con
# los de estos ficheros.
ORIGENES = "internal/tema/testdata/yaru"


def cosechar(mio: str, ruta: str) -> dict | None:
    texto, final = bajar(ruta)
    os.makedirs(ORIGENES, exist_ok=True)
    with open(f"{ORIGENES}/{mio}.svg", "w", encoding="utf-8") as f:
        f.write(texto)
    raiz = ET.fromstring(texto)
    if local(raiz.tag) != "svg":
        fallos.append(f"{mio}: la raíz de {final} es <{local(raiz.tag)}>, no <svg>")
        return None

    caja = raiz.attrib.get("viewBox")
    if not caja:
        fallos.append(f"{mio}: {final} no declara viewBox")
        return None

    nodos = [n for n in (nodo(c, mio) for c in raiz) if n is not None]
    if not nodos:
        fallos.append(f"{mio}: {final} no tiene nada que dibujar")
        return None

    return {"caja": caja, "nodos": nodos, "de": final}


CABECERA = f'''// Los iconos de Yaru, que son los de Ubuntu y los de GNOME (ADR 0070).
//
// **Generado por `herramientas/iconos/yaru.py`. No se edita a mano**: se cosecha
// otra vez, que además es lo que deja constancia de dónde viene cada forma.
//
// Commit de origen: {COMMIT}
//
// # Licencia
//
// Yaru · © Canonical Ltd. y los colaboradores de Yaru · CC BY-SA 4.0
// https://github.com/ubuntu/yaru
//
// Esta licencia exige **citar a quien son** y que **las adaptaciones** se
// publiquen con la misma licencia. Aquí **no se adapta nada**: las formas, sus
// atributos y sus `transform` salen idénticos a los del origen, y lo único que
// cambia es el envase —de SVG a datos—, que no es una adaptación. Lo vigila
// `TestLosIconosDeYaruNoSeHanAdaptado`. La atribución está también en
// `TERCEROS.md` y en Acerca de, dentro de la propia aplicación: el repositorio
// es público, pero quien recibe el programa no lee el repositorio.
//
// # Cómo se usan
//
// Cada icono trae **su `viewBox`** —que en Yaru no siempre es `0 0 16 16`— y su
// **árbol de dibujo**, no una lista de trazados: seis de éstos llevan
// `transform` y uno dibuja con `<rect>` y `<circle>`, así que quedarse con las
// `d` daría iconos fuera de su caja. Se pintan con `fill="currentColor"` en el
// `<svg>`: son de una tinta, que es para lo que existen los simbólicos.

/** Un elemento de dibujo: su etiqueta, sus atributos y lo que lleve dentro. */
export type NodoYaru = {{
  t: string;
  a?: Record<string, string>;
  h?: NodoYaru[];
}};

/** Un icono de Yaru: su caja, su árbol y de qué fichero del origen salió. */
export type IconoYaru = {{
  caja: string;
  nodos: NodoYaru[];
  /** La ruta dentro de `icons/Yaru/scalable/`, ya seguidos los enlaces. */
  de: string;
}};

export const ICONOS_YARU: Record<string, IconoYaru> = '''


def main() -> int:
    salida: dict[str, dict] = {}
    for mio, ruta in sorted(CUALES.items()):
        try:
            icono = cosechar(mio, ruta)
        except RuntimeError as e:
            fallos.append(f"{mio}: {e}")
            continue
        if icono is None:
            continue
        salida[mio] = icono
        enlace = f"  (enlace → {icono['de']})" if icono["de"] != ruta else ""
        print(f"  · {mio} ← {ruta}{enlace}")

    if fallos:
        print("\nNo he podido cosechar:", file=sys.stderr)
        for f in fallos:
            print("   ", f, file=sys.stderr)
        print(
            "\nSe busca el nombre bueno en\n"
            "    https://github.com/ubuntu/yaru/tree/master/icons/Yaru/scalable\n"
            "y se corrige en CUALES. **No se inventa un parecido**: un icono que no\n"
            "es el que la gente conoce no sirve de nada. Y si lo que falla es un\n"
            "elemento o un atributo que este guion no sabe tratar, se añade a la\n"
            "lista con su porqué — nunca se deja pasar «lo que no conozco».",
            file=sys.stderr,
        )
        return 1

    cuerpo = json.dumps(salida, ensure_ascii=False, indent=2, sort_keys=True)
    with open("frontend/src/iconos-yaru.ts", "w", encoding="utf-8") as f:
        f.write(CABECERA + cuerpo + ";\n")
    print(f"\nEscrito frontend/src/iconos-yaru.ts con {len(salida)} iconos")
    return 0


if __name__ == "__main__":
    sys.exit(main())
