// Lo que Vite hace y TypeScript no sabe de serie.

// Vite sirve cualquier fichero importado con «?raw» como una cadena.
declare module "*.svg?raw" {
  const contenido: string;
  export default contenido;
}

// `import.meta.env.DEV` lo **sustituye Vite al construir** por una constante:
// `true` sirviendo en desarrollo y `false` en `vite build`, que es lo que deja
// fuera del paquete la simulación de plataforma (ver `plataforma.ts`). Se
// declara sólo lo que se usa, en vez de traerse los tipos enteros de Vite: así
// esto no se convierte en una puerta para apoyarse en cualquier otra cosa suya
// sin darse cuenta.
interface ImportMeta {
  readonly env: { readonly DEV: boolean };
}
