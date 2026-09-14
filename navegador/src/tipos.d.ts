/**
 * Lo que Vite sabe importar y TypeScript no.
 *
 * El panel trae la marca **en crudo desde `build/marca.svg`**, igual que la
 * ventana (`frontend/src/componentes.tsx`), para que no haya una segunda copia
 * del dibujo que pueda quedarse atrás. Ese `?raw` lo resuelve Vite al compilar;
 * aquí solo se le dice al comprobador de tipos qué devuelve.
 */
declare module "*.svg?raw" {
  const contenido: string;
  export default contenido;
}
