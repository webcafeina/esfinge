# Cómo compilar la extensión Esfinge {{VERSION}}

Instrucciones para reproducir el paquete de Firefox a partir de este código fuente.
El resultado tiene que ser **idéntico, byte a byte**, al que se ha subido a
addons.mozilla.org. / *Instructions to rebuild the Firefox package from this source.
The result must be byte-for-byte identical to the uploaded package.*

## Entorno / Environment

- Linux o macOS, con `bash`. / *Linux or macOS, with `bash`.*
- **Node.js 22** (probado con 22.23.1): https://nodejs.org/
- **pnpm 11.20.0**, fijado en `navegador/package.json` (`packageManager`). La forma más
  sencilla es activarlo con corepack, que viene con Node. / *Pinned in
  `navegador/package.json`; the easiest way is corepack, bundled with Node:*

  ```sh
  corepack enable
  corepack prepare pnpm@11.20.0 --activate
  ```

  O sin corepack: `npm install -g pnpm@11.20.0`.

Las demás herramientas —Vite y TypeScript— las instala pnpm con las versiones exactas
de `navegador/pnpm-lock.yaml`. Todas son de código abierto y funcionan sin red una vez
instaladas. / *Everything else (Vite, TypeScript) is installed by pnpm with the exact
versions in the lockfile.*

## Compilar / Build

Desde la raíz de este código fuente: / *From the root of this source tree:*

```sh
cd navegador
pnpm install --frozen-lockfile
NAVEGADOR=firefox VERSION={{VERSION}} pnpm run build
```

El paquete queda en `navegador/dist/firefox/`: `manifest.json`, `fondo.js`, `pagina.js`,
`panel.html`, `panel.js`, `panel.css` y `iconos/`. / *The package is written to
`navegador/dist/firefox/`.*

## Qué hay aquí / What is here

- `navegador/`: la extensión. `src/` es el código en TypeScript; `vite.config.ts`,
  `vite.fondo.config.ts` y `vite.pagina.config.ts` lo empaquetan en `panel.js`, `fondo.js`
  y `pagina.js`. **No se ofusca**: Vite solo agrupa y minimiza.
- `build/marca.svg` y `build/icono-barra.svg`: dibujos de la marca que la extensión incluye
  en línea.
- `frontend/src/monograma.ts` y `frontend/src/tokens.css`: el cuadro con la inicial de cada
  sitio y los colores, compartidos con la aplicación de escritorio.

La extensión habla **solo con la aplicación Esfinge instalada en el mismo ordenador**, por
native messaging. No descarga código ni se conecta a ningún servidor. / *The extension
talks only to the Esfinge desktop app on the same computer, through native messaging. It
downloads no code and connects to no server.*
