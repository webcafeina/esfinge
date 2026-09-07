// Hace de API de publicaciones de GitHub durante las pruebas.
//
// Existe para que la prueba del aviso de versión nueva recorra el camino de
// verdad —Go pregunta, compara, elige el fichero de este sistema y lo devuelve
// por el puente— sin tocar internet ni depender de lo que haya publicado hoy el
// repositorio.
import { createServer } from "node:http";

const VERSION = "9.9.9";

const publicacion = {
  tag_name: `v${VERSION}`,
  html_url: `https://github.com/webcafeina/esfinge/releases/tag/v${VERSION}`,
  assets: [
    { name: `Esfinge-${VERSION}.dmg`, browser_download_url: "http://127.0.0.1:34444/f", size: 1024 },
    {
      name: `Esfinge-${VERSION}-windows-instalador.exe`,
      browser_download_url: "http://127.0.0.1:34444/f",
      size: 1024,
    },
    {
      name: `esfinge_${VERSION}_amd64.deb`,
      browser_download_url: "http://127.0.0.1:34444/f",
      size: 1024,
    },
    {
      name: `esfinge-${VERSION}-linux-amd64.tar.gz`,
      browser_download_url: "http://127.0.0.1:34444/f",
      size: 1024,
    },
  ],
};

createServer((peticion, respuesta) => {
  if (peticion.url === "/salud") {
    respuesta.writeHead(200).end("bien");
    return;
  }
  respuesta.writeHead(200, { "Content-Type": "application/json" });
  respuesta.end(JSON.stringify(publicacion));
}).listen(34444, "127.0.0.1");
