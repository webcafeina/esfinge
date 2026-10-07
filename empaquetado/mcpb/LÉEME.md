# El paquete instalable para Claude Desktop

Un `.mcpb` —antes `.dxt`— es **un zip con un manifiesto y el servidor dentro**, que Claude Desktop instala
con un clic desde Ajustes → Extensiones, o arrastrándolo a su ventana.

## Por qué existe

Porque la alternativa era pegar un bloque de JSON a mano en el fichero de configuración de otro programa, y
**eso solo sale bien una vez de cada dos**: hay que encontrar el fichero, no romper su sintaxis y acertar con
la ruta entera del binario —que ahí es obligatoria, porque ese cliente arranca los servidores desde un
directorio indefinido—.

Lo preguntó el cliente comparándolo con Cronos, que se instala con una URL. Y ahí está la diferencia que
conviene no olvidar: **Cronos es un servidor remoto y Esfinge no puede serlo**. La bóveda está en este
ordenador; un servidor remoto obligaría a elegir entre exponer el equipo o que el Worker vea los secretos
(ADR 0054). El `.mcpb` es la forma de que un servidor **local** se instale igual de fácil.

## Qué lleva dentro

```
manifest.json          el de aquí al lado, con la versión puesta al empaquetar
icon.png               la marca, 512×512
server/esfinge-mcp     el binario, el de esa plataforma
```

**Uno por plataforma**, porque el binario lo es: `esfinge-2.44.0-macos.mcpb`, `-windows.mcpb`, `-linux.mcpb`.
Lo arma `herramientas/armar-mcpb.sh` y lo cuelga de la publicación `publicar.yml`.

## Lo que hay que saber si se toca

- **El binario va dentro del paquete**, no se apunta al que instala Esfinge. Así la extensión funciona
  aunque Esfinge esté en otra carpeta, y actualizar Esfinge no la rompe. Lo que sí hace falta es que
  **Esfinge esté abierta**: el binario solo traduce, y la bóveda vive en la ventana.
- **`${__dirname}` es la carpeta donde Claude Desktop lo instala.** Sin eso, la ruta del comando no se puede
  escribir: no se sabe de antemano.
- **En Windows el binario lleva `.exe`** y el manifiesto no lo dice: lo añade el propio Claude Desktop.
- **No se firma.** Es la misma decisión que para el resto de Esfinge (ADR 0012), y aquí significa que la
  primera instalación avisa de que viene de fuera de la tienda.
