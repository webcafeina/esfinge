# El formato de la bóveda

Hasta la A1 el formato vivía solo en el código de Go (`internal/boveda`). Con cuentas va a haber **dos
implementaciones** —Go en la aplicación y TypeScript en la extensión, fase E—, y este documento es el
contrato entre las dos. **Si el código y esto no coinciden, está mal uno de los dos**, y las pruebas
cruzadas de la fase E deciden cuál.

El contenedor `ESF1` (Argon2id + XChaCha20-Poly1305) no se describe aquí: está en `internal/cripto`,
congelado con vectores fijos (ADR 0022), y la versión de TypeScript se prueba contra esos mismos vectores.

## El fichero

JSON en claro, `formato: 1`. Los campos criptográficos son líneas `ESF1.…` corrientes.

```json
{
  "esfinge": "bóveda",
  "aviso": "Bóveda de Esfinge. …",
  "formato": 1,
  "id": "32 cifras hexadecimales, al azar al crearla; el mismo en todos sus equipos",
  "serie": 42,
  "cambiada": "RFC3339",
  "sobres": [
    { "tipo": "maestra", "creado": "RFC3339", "contenedor": "ESF1.…" },
    { "tipo": "recuperacion", "creado": "RFC3339", "contenedor": "ESF1.…", "codificacion": "crockford32-v1" }
  ],
  "sello": "ESF1.…",
  "cuerpo": "ESF1.…"
}
```

- **`serie`** cuenta los guardados de **este fichero** y sirve para no pisar lo que otro proceso haya
  escrito. No es la versión del servidor.
- Un `formato` mayor que el que se entiende **no se abre**, y uno menor se abre solo para leer.

## Las claves

- **La clave de bóveda** son 32 bytes al azar, manejados **como texto**: 43 caracteres base64url sin
  relleno. Ese texto es la «contraseña» con la que se sellan el sello y el cuerpo, con el coste mínimo
  (`PerfilLlave`: 8 MiB, 1 pasada, 1 hilo), porque ya es aleatoria.
- **Cada sobre** es la clave de bóveda (el texto de 43 caracteres) sellada en un `ESF1` con una llave de
  persona y el coste de siempre (`PerfilInteractivo`: 64 MiB, 3 pasadas, 4 hilos). La de recuperación se
  normaliza antes (`Normalizar`).
- Los tipos de sobre son una lista abierta. **`llavero-del-sistema` y `pin` son de un solo equipo** y no
  se suben nunca.

## El sello

`sello` es, cifrado con la clave de bóveda:

```json
{ "id": "…", "serie": 42, "sincro": 17, "huellas": { "maestra": "…", "recuperacion": "…" }, "cuerpo": "…" }
```

- `huellas[tipo]` es el SHA-256 en hexadecimal del texto `contenedor` de ese sobre, y `cuerpo`, el del
  texto `cuerpo` del fichero.
- **Al abrir se comprueba**: `id` y `serie` iguales a los de fuera, la huella del cuerpo, la de cada sobre
  que el sello conoce, y que no falte ningún sobre que el sello conozca. Un sobre que el sello no conoce se
  acepta (puede ser de una versión más nueva). Cualquier otra cosa es `ErrManipulada`.
- **`sincro`** es la versión del servidor que tiene o va a tener este documento; 0 si nunca se ha
  sincronizado. Al fundir, la versión que dice el servidor tiene que ser igual a ésta.

## El cuerpo

`cuerpo` es, cifrado con la clave de bóveda:

```json
{
  "entradas": [ … ],
  "sitiosExcluidos": ["dominio.com"],
  "lapidas": { "id de entrada": "RFC3339" }
}
```

**Las claves que no se conocen se conservan tal cual**, en el contenido y en cada entrada. Una versión que
no entiende algo no puede borrarlo al guardar.

### Una entrada

| Campo | Qué es |
|---|---|
| `id` | 32 cifras hexadecimales al azar. No cambia nunca |
| `tipo` | `credencial` · `nota` · `tarjeta` · `identidad` |
| `titulo`, `notas`, `etiquetas`, `carpeta` | Comunes |
| `creada`, `cambiada` | RFC3339, resolución de un segundo |
| `revision` | Cuántas veces ha cambiado. **La pone la bóveda al guardar**, nunca quien edita: 1 al crear, +1 al editar, al mandar a la papelera y al sacar |
| `papelera`, `borradaEn` | Borrado suave |
| `usuario`, `secreto`, `sitios`, `totp`, `historial` | Credencial. `historial`: `[{secreto, hasta}]`, lo más reciente primero, diez como mucho |
| `titular`, `numero`, `caduca`, `verificacion` | Tarjeta |
| `nombreCompleto`, `documento`, `numeroDocumento` | Identidad |

Los vacíos no se escriben.

### Lápidas

Borrar del todo —a mano desde la papelera, al vaciarla o sola a los treinta días— **quita la entrada y
deja su lápida**, que se va a los 180 días. Al abrir se purgan las dos cosas.

## Fundir

Lo que hace `Boveda.Fundir` con la bóveda de aquí (L), la del servidor (R) y la última común (B, puede
faltar). El porqué está en la ADR 0038.

**Forma canónica** de una entrada, para comparar y desempatar: JSON con **las claves en orden alfabético a
todos los niveles**, sin espacios, **sin escapar `<`, `>` ni `&`**, y los números tal cual. Dos entradas
son iguales si su forma canónica es igual.

**La mayor de dos entradas** (`mayor`): la de más `revision`; si empatan, la de `cambiada` mayor como
texto; si empatan, la de SHA-256 de la forma canónica mayor en hexadecimal.

**Entradas**, en el orden de R y luego las que solo están en L:

1. En L y en R: iguales → R. L igual a B → R. R igual a B → L. Si no, **fundir campos**.
2. Solo en un lado (P), falta en el otro:
   - si estaba en B: queda si P no es igual a B (**la edición gana al borrado**);
   - si no estaba en B y el otro lado no tiene lápida suya: queda (es nueva);
   - si el otro lado tiene lápida: queda si esa lápida ya estaba en B y el lado de P la había quitado, o
     si `P.cambiada >= lápida`.
3. Si una entrada queda, se quita su lápida. Si no queda y no tenía lápida, se le pone una con la hora de
   ahora.

**Fundir campos** de L y R:

- Con B, clave a clave de la forma JSON: iguales en L y R → ése; L igual a B → R; R igual a B → L; si no,
  el de la mayor. Sin B, la mayor entera.
- `historial`: la unión de los de L y R, más **el `secreto` de la que no es la mayor** si es distinto del
  que queda, con `hasta` = ahora; sin repetir secretos (se queda el `hasta` mayor), sin el secreto actual,
  lo más reciente primero y diez como mucho.
- `revision` = la mayor de las dos + 1; `cambiada` = la mayor de las dos.

**Lápidas**: la unión, con la fecha mayor; menos las de las entradas que quedan.

**Sitios excluidos**: con B, cada sitio queda si está en L y en R, o en uno y no en B; sin B, la unión.
Ordenados.

**Secciones del contenido que no se conocen**: como un valor entero, a tres bandas; si cambió en los dos
lados, gana R.

**Sobres**, por tipo: los de un solo equipo, los de aquí. Si el tipo solo está en un lado, ése. Iguales →
ése. Con B: L igual a B → R; R igual a B → L. Si no, el de `creado` mayor, y si empatan, el de
`contenedor` mayor.

**Después**:

- Si se van más de la mitad de las entradas vivas de L (con al menos cuatro), no se aplica sin permiso.
- Si cambia algo de L, se guarda; si se va alguna entrada viva, antes se copia el fichero a
  `<ruta>.antes-de-fundir`.
- Hay que subir si el resultado no es igual a R (contenido y sobres de todos los equipos, sin mirar el
  orden de los sobres).

## Lo que se sube

La bóveda tal cual, **sin los sobres de un solo equipo**, con el sello rehecho para esos sobres y con
`sincro` = la versión que va a tener en el servidor.
