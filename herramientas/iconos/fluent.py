#!/usr/bin/env python3
"""Cosecha los iconos de Fluent System Icons que usa Esfinge en Windows.

Escribe `frontend/src/iconos-fluent.ts`. Se ejecuta a mano, no en cada
compilación: lo que se publica es el fichero generado, no la descarga.

# Por qué esto vive en el repositorio

Porque en Apolo los generadores de iconos se quedaron en `~/.cache/` y **la
trampa de los trazados volvió dos veces**: cuando hizo falta cosechar un icono
más, la herramienta ya no estaba y se rehízo a mano. Un generador que no está
versionado es un generador que se pierde.

# Y por qué no se juntan los trazados

Cada icono de Fluent puede traer varios `<path>`. Aquí se guardan **tal como
vienen, uno por elemento**, en vez de pegarlos en una sola `d`. Dos razones y
las dos pesan:

- Juntarlos obliga a reescribir la `m` inicial de cada trazado siguiente, y esa
  es exactamente la trampa que en Apolo dejó a Ubuntu sin la montaña del icono
  del Estudio durante cinco versiones. Si no se juntan, no hay nada que
  reescribir.
- Y juntarlos, recortarlos o reescalarlos es **adaptar** la obra. Fluent es MIT
  y eso daría igual; pero el mismo generador sirve de patrón para Yaru
  (CC BY-SA) y Breeze (LGPL), donde adaptar sí tiene consecuencias. La regla es
  la misma en los tres: lo que entra, entra como está.

Uso:
    python3 herramientas/iconos/fluent.py
"""

import json
import re
import sys
import urllib.error
import urllib.parse
import urllib.request

# El commit del que se cosecha. **Fijado**: una etiqueta rodante haría que dos
# cosechas dieran iconos distintos sin que nadie lo pidiera, y Microsoft no
# publica etiquetas para los SVG.
COMMIT = "6c2c9075cf8952ef48369b2475233b6e9761efed"
BASE = "https://raw.githubusercontent.com/microsoft/fluentui-system-icons"

# Qué icono de Fluent le toca a cada nombre de Esfinge.
#
# Se eligen los **regular** y no los **filled**: la barra lateral de WinUI usa
# los de trazo, y los rellenos son para el estado activo de una barra de
# pestañas, que aquí no existe.
#
# Los cinco logotipos de gestores (Dashlane, Bitwarden…) no están aquí **y no es
# un olvido**: son marcas, Fluent no trae un «Bitwarden», y los nuestros ya son
# formas propias a propósito. Tampoco el de MCP, que es el oficial.
CUALES = {
    "cifrar": "lock_closed",
    "descifrar": "lock_open",
    "generar": "sparkle",
    "boveda": "shield_keyhole",
    "proyectos": "folder",
    "historial": "history",
    "todo": "apps_list",
    "credencial": "key",
    "nota": "note",
    "tarjeta": "payment",
    "identidad": "contact_card",
    "personal": "person",
    "llave": "person_passkey",
    "wifi": "wifi_1",
    "ajustes": "settings",
    "descargar": "arrow_download",
    "navegador": "globe",
    "servidor": "server",
    # Los tres de abajo nacieron después de esta fase —las dos pestañas de
    # Cifrar y la de Apariencia— y se quedaron fuera, así que en Windows esas
    # filas enseñaban unos iconos de Fluent y otros dibujados a mano. Se vio al
    # cosechar los de Yaru, comparando los nombres que usa la interfaz con los
    # dos juegos.
    "texto": "text_description",
    "ficheros": "document_multiple",
    "tema": "dark_theme",
}

# Los dos tamaños de Fluent que se usan: 16 en las filas y 20 en las tarjetas.
# Cada uno está **dibujado para su tamaño**, no es el otro escalado, que es la
# mitad de por qué los iconos de Windows se ven nítidos.
TAMANOS = (16, 20)


# Las carpetas cuyo nombre no sale de la regla de abajo. Van **nombradas una a
# una**: la alternativa es que el cosechador «pruebe cosas» hasta acertar, y
# entonces el día que Microsoft renombre una, cosecharía otro icono parecido sin
# decirlo.
CARPETAS = {
    "wifi_1": "WiFi 1",
}


def carpeta(nombre: str) -> str:
    """De «lock_closed» a «Lock Closed», que es como se llama la carpeta."""
    if nombre in CARPETAS:
        return CARPETAS[nombre]
    return " ".join(p.capitalize() for p in nombre.split("_"))


def bajar(nombre: str, tam: int) -> str | None:
    # El nombre de la carpeta lleva espacios, y una URL no los admite tal cual.
    ruta = urllib.parse.quote(f"assets/{carpeta(nombre)}/SVG/ic_fluent_{nombre}_{tam}_regular.svg")
    url = f"{BASE}/{COMMIT}/{ruta}"
    try:
        with urllib.request.urlopen(url, timeout=20) as r:
            return r.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        print(f"  ✕ {nombre} {tam}: {e.code} {url}", file=sys.stderr)
        return None


def trazados(svg: str) -> list[str]:
    """Las `d` de cada `<path>`, tal cual vienen."""
    return re.findall(r'<path[^>]*\sd="([^"]+)"', svg)


def main() -> int:
    salida: dict[str, dict[int, list[str]]] = {}
    fallos = []
    for mio, suyo in sorted(CUALES.items()):
        salida[mio] = {}
        for tam in TAMANOS:
            svg = bajar(suyo, tam)
            if svg is None:
                # Que falte **un** tamaño es normal: no todos los iconos de Fluent
                # están dibujados en los dos. Lo que no puede faltar es el icono.
                continue
            ds = trazados(svg)
            if not ds:
                fallos.append(f"{mio} ({suyo}) a {tam}: sin trazados")
                continue
            salida[mio][tam] = ds
        if salida[mio]:
            print(f"  · {mio} ← {suyo} ({', '.join(str(t) for t in sorted(salida[mio]))})")
        else:
            fallos.append(f"{mio} ({suyo}): no está en ningún tamaño")

    if fallos:
        print("\nNo he podido cosechar:", file=sys.stderr)
        for f in fallos:
            print("   ", f, file=sys.stderr)
        print(
            "\nEl nombre de Fluent será otro. Se busca en\n"
            "    https://github.com/microsoft/fluentui-system-icons/tree/main/assets\n"
            "y se corrige en CUALES. **No se inventa un parecido**: un icono que no\n"
            "es el que la gente conoce no sirve de nada.",
            file=sys.stderr,
        )
        return 1

    cuerpo = ",\n".join(
        "  "
        + json.dumps(mio)
        + ": { "
        + ", ".join(f'"{t}": {json.dumps(ds)}' for t, ds in sorted(tams.items()))
        + " }"
        for mio, tams in sorted(salida.items())
    )

    with open("frontend/src/iconos-fluent.ts", "w", encoding="utf-8") as f:
        f.write(
            f"""// Los iconos de Fluent System Icons, que son los de Windows 11 (ADR 0064).
//
// **Generado por `herramientas/iconos/fluent.py`. No se edita a mano**: se
// cosecha otra vez, que además es lo que deja constancia de dónde viene cada
// trazado.
//
// Commit de origen: {COMMIT}
//
// # Licencia
//
// Fluent System Icons · © Microsoft Corporation · MIT
// https://github.com/microsoft/fluentui-system-icons
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// # Cómo se usan
//
// Cada icono trae **sus dos tamaños dibujados aparte** —16 para las filas, 20
// para las tarjetas— y **sus trazados sin juntar**, tal como vienen. Los `d` se
// pintan uno por `<path>`, con `fill="currentColor"` y el `viewBox` de su
// tamaño: estos son de relleno, no de trazo como los dibujados de la casa.

/** Los trazados de un icono, por tamaño de caja. */
export type IconoFluent = Record<string, string[]>;

export const ICONOS_FLUENT: Record<string, IconoFluent> = {{
{cuerpo},
}};
"""
        )
    print(f"\n✓ Escrito frontend/src/iconos-fluent.ts con {len(salida)} iconos")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
