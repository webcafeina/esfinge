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
- **`boveda-principal` es la ranura de una bóveda de proyecto** ([ADR 0050](adr/0050-varias-bovedas.md)): la
  clave de ese proyecto sellada con **la clave de bóveda de la bóveda personal**, con `PerfilLlave` y no con
  el coste de una contraseña humana, porque el secreto son esos 43 caracteres al azar. **Ésta sí se sube**,
  al contrario que las dos de arriba: es lo único que permite que otro equipo abra el proyecto. Una bóveda
  de proyecto **no tiene ranura maestra ni de recuperación**: la contraseña maestra no la abre.

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
  "identidad": { "semilla": "…", "creada": "RFC3339", "suite": "…" },
  "envios": [ { "id": "…", "entrada": "…", "correo": "…", "huella": "…", "creado": "RFC3339" } ],
  "proyectos": [ { "ref": "16 hex", "nombre": "Acme", "creado": "RFC3339", "usado": "RFC3339", "archivado": false } ]
}
```

**Las claves que no se conocen se conservan tal cual**, en el contenido y en cada entrada. Una versión que
no entiende algo no puede borrarlo al guardar.

### Los proyectos

`proyectos` son las bóvedas de proyecto que abre **esta** bóveda ([ADR 0050](adr/0050-varias-bovedas.md)), y
solo lo tiene la personal. Cada una lleva su referencia —hex de 8 bytes, **y el nombre de su fichero**—, su
nombre, cuándo se creó, cuándo se abrió por última vez y si está archivada.

**No lleva la clave de ningún proyecto.** Lo que abre un proyecto es su ranura `boveda-principal`, que viaja
dentro de su propio fichero, así que un equipo que se baje el fichero lo abre sin consultar esta lista.
Guardarla aquí sería amontonar las llaves de todos los proyectos en un sitio más sin ganar nada.

Está aquí dentro, y no en un fichero al lado, porque **la lista de proyectos es la lista de clientes**: es el
mismo razonamiento que los sitios excluidos y los iconos ([ADR 0024](adr/0024-iconos-de-los-sitios.md)). El
registro local (`bovedas.json`) solo apunta qué referencias hay, sin nombres.

**Se funde a tres bandas por `ref`**, como los envíos: lo que estaba en la base y falta en un lado, lo quitó
ese lado. Y dos reglas finas, porque un proyecto **sí se edita en los dos equipos a la vez**:

- **`usado` es el mayor de los dos**, porque las dos aperturas ocurrieron. Es la misma regla que la fecha
  `usada` de una llave de acceso.
- **El nombre lo decide la base**: gana el lado que lo cambió, y si lo cambiaron los dos, el mayor por bytes
  — arbitrario, pero **igual en los dos equipos**, que es lo que hace falta ([ADR 0038](adr/0038-sincronizar-la-boveda.md)).
- Y si un equipo archiva y el otro desarchiva, **gana desarchivado**: es el estado que lo enseña en vez de
  esconderlo.

En el espejo de TypeScript esta sección vive en `extra` —el `Contenido` de allí no la declara— pero **se
funde con estas mismas reglas** (`navegador/src/nucleo/proyecto.ts`). Como sección opaca ganaría la del
servidor entera y se perdería un proyecto creado en la ventana.

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

### Lo que espera

`envios` son las copias mandadas a quien **todavía no tenía cuenta**, esperando a que la cree
([ADR 0043](adr/0043-la-identidad-para-compartir.md), entrega B3).

| Campo | Qué es |
|---|---|
| `id` | 16 bytes al azar en hexadecimal |
| `entrada` | a qué entrada apunta. Si ya no está, la nota se cae |
| `correo` | a quién se le mandó |
| `huella` | la de las llaves que dio el servidor entonces. **Es el disparo**: cuando cambie, hay a quién mandar |
| `creado` | RFC3339. A los **30 días** se deja de intentar |

**No lleva el secreto**, y no es un detalle: el secreto sigue en su entrada, y esto es una nota de a quién
se le debe una copia. Una por `entrada` y `correo`: volver a mandar lo mismo a la misma persona actualiza la
que había.

### Una entrada

| Campo | Qué es |
|---|---|
| `id` | 32 cifras hexadecimales al azar. No cambia nunca |
| `tipo` | `credencial` · `nota` · `tarjeta` · `identidad` · `personal` · `llave` · `wifi` |
| `titulo`, `notas`, `etiquetas`, `carpeta` | Comunes |
| `creada`, `cambiada` | RFC3339, resolución de un segundo. **`cambiada` la tocan también mandar a la papelera y restaurar**: sin base, «vive si se cambió después de borrarse» es la única regla que queda, y una entrada rescatada aquí perdía contra la purga de allí |
| `revision` | Cuántas veces ha cambiado. **La pone la bóveda al guardar**, nunca quien edita: 1 al crear, +1 al editar, al mandar a la papelera y al sacar |
| `papelera`, `borradaEn` | Borrado suave |
| `usuario`, `secreto`, `sitios`, `totp`, `historial` | Credencial. `historial`: `[{secreto, hasta}]`, lo más reciente primero, diez como mucho |
| `titular`, `numero`, `caduca`, `verificacion` | Tarjeta |
| `nombreCompleto`, `documento`, `numeroDocumento` | Identidad |
| `rpId`, `idCredencial`, `idUsuario`, `nombreVisible`, `algoritmo`, `clavePrivada`, `confirmada` | Llave de acceso ([ADR 0048](adr/0048-las-llaves-de-acceso.md)). **No hay contador de firmas**: se firma siempre con cero, y por eso esta clase no necesita regla de fusión propia. `clavePrivada` es **PKCS#8** en base64url: el escalar a secas no se puede importar en WebCrypto, que exige también la parte pública y no sabe multiplicar por el generador. **Dos fechas, y dicen cosas distintas**: `confirmada` la pone el navegador **la primera vez que el sitio nombra esa llave** —lo único que prueba que la tiene registrada— y `usada`, **la última vez que se firmó con ella**, que solo prueba que alguien la eligió. Van separadas porque la fuerte casi nunca llega: donde el sitio no nombra ninguna llave, `usada` es todo lo que se sabe. `usada` se reescribe en cada firma, y no se reescribe con la misma fecha |
| `ssid`, `seguridad`, `oculta` | Red wifi ([ADR 0049](adr/0049-las-redes-wifi.md)). La clave va en `secreto`, como cualquier otra contraseña, con su ojo y su historial. `ssid` es el nombre que emite la red y va aparte del título: el título es cómo se llama la entrada —«La oficina»— y el SSID es lo que el móvil tiene que encontrar. `seguridad` es `wpa`, `wep` o `abierta`, y **no se copia de lo que diga el fichero importado**: Dashlane marca `unsecured` redes que tienen contraseña. `oculta` es el **segundo campo de sí o no** del formato, el primero después de `papelera` |
| `correo`, `telefono`, `nacimiento` | Dato personal ([ADR 0047](adr/0047-los-datos-personales.md)). `nombreCompleto` se comparte con la identidad: es el mismo dato |
| `destinatario`, `calle`, `edificio`, `piso`, `puerta`, `codigoPostal`, `ciudad`, `provincia`, `pais` | La dirección de un dato personal, **por trozos**: es como la da un gestor y como la pide un formulario. Componerla para leerla es de una línea; partirla sería adivinar. **La 2.30.0 escribía `direccion`, un solo texto**: al leerlo se trae entero a `calle` y se deja de escribir, y eso lo hacen los dos lados igual |

Los vacíos no se escriben.

### Lápidas

Borrar del todo —a mano desde la papelera, al vaciarla o sola a los treinta días— **quita la entrada y
deja su lápida**, que se va a los 180 días. Al abrir se purgan las dos cosas.

## El sobre de un envío

Lo que viaja cuando se manda una copia de una entrada a otra cuenta
([ADR 0043](adr/0043-la-identidad-para-compartir.md)). **No es parte de la bóveda**, pero se describe aquí
porque es el otro sitio donde las dos implementaciones tienen que escribir los mismos bytes.

```json
{
  "esfinge": "envío", "version": 1,
  "suite": "DHKEM(X25519)/HKDF-SHA256/ChaCha20-Poly1305",
  "de": { "cifrado": "base64", "firma": "base64" },
  "para": "base64 de la llave de cifrado de quien lo recibe",
  "enc": "base64 del encapsulado de HPKE",
  "cuerpo": "base64 de la entrada cifrada",
  "firma": "base64 de la firma Ed25519"
}
```

- **El cuerpo** es la entrada en forma canónica, **sin `id`, sin `historial`, sin papelera y con
  `revision` a cero**: es una copia, no la misma entrada en dos bóvedas.
- **Cifrado con HPKE** en modo base, `info = "esfinge/envio/v1"`, hacia `para`.
- **Lo autenticado** (los datos asociados del cifrado) es la cabecera **sin el cuerpo y sin la firma**,
  como `json.Marshal` de un mapa: claves en orden alfabético, sin espacios y los bytes en **base64
  estándar con relleno**. Al cerrar, el cuerpo todavía no existe; por eso no entra.
- **Lo firmado** es `{"cabecera":<lo autenticado>,"cuerpo":"base64"}`, con Ed25519 y la llave de firma de
  quien manda. **La firma se comprueba antes de descifrar.**
- Al abrir se exige, por este orden: versión conocida, que `suite` y `para` sean los de esta bóveda, que la
  firma cuadre, y por último que el cifrado abra.

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

**Lo que espera** (`envios`): como un conjunto **por `id`**, con la regla de los sitios excluidos —lo que
estaba en B y falta en un lado, lo quitó ese lado—, y con lo de R cuando está en los dos. **Ordenado por
`id`**, que es el único orden que los dos equipos calculan igual. No hay desempates finos a propósito:
perder una nota entregada o conservarla de más cuesta lo mismo, que esa copia llegue dos veces.

**Los proyectos**, por `ref`: igual que `envios` para decidir quién queda, y con las tres reglas de arriba
—`usado` el mayor, el nombre lo decide la base, y desarchivado gana— para lo que queda en los dos. Ordenados
por `ref`.

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
