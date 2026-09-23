# El formato de la bóveda

Hasta la A1 el formato vivía solo en el código de Go (`internal/boveda`). Desde la E1 (2026-09-22) hay
**dos implementaciones** —Go en la aplicación y TypeScript en la extensión, `navegador/src/nucleo/`—, y
este documento es el contrato entre las dos. **Si el código y esto no coinciden, está mal uno de los dos**,
y las pruebas cruzadas (`cruzada_test.go`, ADR 0040) deciden cuál.

**La forma canónica tiene una trampa que no se ve**: Go escapa U+2028 y U+2029 aunque no escape el HTML, y
`JSON.stringify` no. La extensión la escribe a mano (`canon.ts`).

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
{
  "id": "…", "serie": 42,
  "huellas": { "maestra": "…", "recuperacion": "…" },
  "sobres":  { "maestra": "…", "recuperacion": "…" },
  "sincro": 17, "cuerpo": "…"
}
```

- `huellas[tipo]` es el SHA-256 en hexadecimal del texto `contenedor` de ese sobre, y `cuerpo`, el del
  texto `cuerpo` del fichero.
- **`sobres[tipo]` es la huella del sobre entero**: el SHA-256 de `tipo`, `creado`, `codificacion` (vacía
  si no la lleva) y `contenedor`, en ese orden y pegados con saltos de línea —que no pueden aparecer dentro
  de ninguno de los cuatro—. Es **una cadena y no JSON** a propósito, para que las dos implementaciones no
  tengan que ponerse de acuerdo en nada más. Existe porque `creado` decide qué ranura gana al fundir sin
  base, y sin sellarlo un servidor podía envejecer una ranura sin tocar su contenedor (revisión del
  2026-09-23).
- **Al abrir se comprueba**: `id` y `serie` iguales a los de fuera, la huella del cuerpo, la de cada sobre
  que el sello conoce —y la del sobre entero, si el sello la trae—, y que no falte ningún sobre que el
  sello conozca. Un sobre que el sello no conoce se acepta (puede ser de una versión más nueva). Cualquier
  otra cosa es `ErrManipulada`.
- **`sobres` puede no estar**: las bóvedas escritas antes de la 2.25.7 no lo traen y se abren igual. No se
  puede quitar para esquivar la comprobación, porque el sello va cifrado con la clave de bóveda.
- **`sincro`** es la versión del servidor que tiene o va a tener este documento; 0 si nunca se ha
  sincronizado. Al fundir, la versión que dice el servidor tiene que ser igual a ésta.

## El cuerpo

`cuerpo` es, cifrado con la clave de bóveda:

```json
{
  "entradas": [ … ],
  "sitiosExcluidos": ["dominio.com"],
  "lapidas": { "id de entrada": "RFC3339" },
  "identidad": { "semilla": "…", "creada": "RFC3339", "suite": "…" }
}
```

**Las claves que no se conocen se conservan tal cual**, en el contenido y en cada entrada. Una versión que
no entiende algo no puede borrarlo al guardar.

### La identidad

`identidad` es la de esta bóveda **para compartir copias** ([ADR 0043](adr/0043-la-identidad-para-compartir.md)):
una **semilla de 32 bytes** en base64url, cuándo nació y con qué conjunto de HPKE se cifra hacia ella. De la
semilla salen, con HKDF-SHA256:

| Etiqueta | Llave |
|---|---|
| `esfinge/identidad/cifrado/v1` | **X25519**, para que otros cifren hacia ti |
| `esfinge/identidad/firma/v1` | **Ed25519**, para firmar lo que mandas |

La **huella** que se compara por teléfono es el SHA-256 de `suite`, `0x00`, la llave de cifrado, `0x00` y la
de firma; de ahí, los primeros 28 símbolos del alfabeto de la clave de recuperación, en grupos de cuatro
separados por guiones.

**Se crea una vez y no cambia.** Una versión que no la conozca la conserva como sección desconocida.

### Una entrada

| Campo | Qué es |
|---|---|
| `id` | 32 cifras hexadecimales al azar. No cambia nunca |
| `tipo` | `credencial` · `nota` · `tarjeta` · `identidad` |
| `titulo`, `notas`, `etiquetas`, `carpeta` | Comunes |
| `creada`, `cambiada` | RFC3339, resolución de un segundo. **`cambiada` la tocan también mandar a la papelera y restaurar**: sin base, «vive si se cambió después de borrarse» es la única regla que queda, y una entrada rescatada aquí perdía contra la purga de allí |
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
- **Papelera contra edición**: si un lado, respecto a B, **no ha hecho más que mandarla a la papelera** y
  el otro ha tocado contenido, la entrada **se queda fuera de la papelera** (`papelera` y `borradaEn`, los
  del lado que tocó contenido). La edición gana al borrado también cuando el borrado es suave. «No ha hecho
  más» se mira sin `papelera`, `borradaEn`, `revision` ni `cambiada`, porque mandar a la papelera toca las
  cuatro.
- `historial`: la unión de los de L y R, más **el `secreto` de la que no es la mayor** si es distinto del
  que queda, con `hasta` = ahora; sin repetir secretos (se queda el `hasta` mayor), sin el secreto actual,
  lo más reciente primero y diez como mucho.
- `revision` = la mayor de las dos + 1; `cambiada` = la mayor de las dos.

**Lápidas**: la unión, con la fecha mayor; menos las de las entradas que quedan.

**Sitios excluidos**: con B, cada sitio queda si está en L y en R, o en uno y no en B; sin B, la unión.
Ordenados.

**Secciones del contenido que no se conocen**: como un valor entero, a tres bandas; si cambió en los dos
lados, gana R.

**La identidad**, en cambio, **no se funde: se elige una**, y las dos implementaciones tienen que elegir la
misma. Gana la de `creada` menor; si empatan, la de `semilla` menor como texto. Solo puede haber dos si dos
equipos crearon la suya antes de verse.

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
