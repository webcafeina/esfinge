// Vite sirve cualquier fichero importado con «?raw» como una cadena. TypeScript
// no lo sabe de serie, así que se le dice aquí.
declare module "*.svg?raw" {
  const contenido: string;
  export default contenido;
}
