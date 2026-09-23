# El servidor de cuentas

Guarda la bóveda **cifrada** de cada cuenta y la reparte entre sus equipos (ADR 0035 y 0036; el plan
entero en [`docs/cuentas.md`](../docs/cuentas.md)). **No puede leer nada**: la contraseña maestra, la
clave de la bóveda y las entradas no llegan nunca aquí.

Es un Worker de Cloudflare en TypeScript:

- **Un Durable Object por cuenta** (`src/cuenta.ts`), con su propia base SQLite: la bóveda en trozos
  con sus versiones, las sesiones, los equipos, los códigos y los frenos de esa cuenta.
- **Una base D1** (`migraciones/`): el índice correo → cuenta, las altas a medias, los contadores por
  día y la lista de admisión.
- Todo en la **UE**: D1 creada con `--jurisdiction eu` y los objetos pedidos con `jurisdiction("eu")`.

## Probar aquí

```sh
make comprobar       # también corre esto: tipos y pruebas dentro del motor de Workers
make servidor        # lo levanta en http://localhost:8790, con el buzón de pruebas
```

Las pruebas corren en `workerd`, el motor de verdad, sin conexión con Cloudflare. **El motor local no
implementa jurisdicciones**, así que ahí `JURISDICCION` va vacía; los dos Workers de verdad la llevan a
`eu` en `wrangler.jsonc`, y una prueba se pone roja si alguien la quita.

## El protocolo

JSON en todo salvo la bóveda, que viaja tal cual. Los errores son `{"error": "Frase en español."}`
con su estado HTTP, y **la frase se puede enseñar tal cual**. Los bytes van en base64url sin relleno.

El correo se normaliza **igual en los dos lados**: sin espacios alrededor, NFC y en minúsculas.

### Entrar

| Ruta | Envía | Devuelve |
|---|---|---|
| `GET /v1/salud` | — | `{registro: "cerrado"\|"lista"\|"abierto", protocolo: 1}` |
| `POST /v1/prelogin` | `{correo}` | `{sal, argon2: {memoria, pasadas, paralelismo}}`. **Igual con cuenta que sin ella** |
| `POST /v1/registro/inicio` | `{correo}` | `202`. Manda un código, o «ya tienes cuenta» si la hay |
| `POST /v1/registro/fin` | `{correo, codigo, sal, argon2, claveDeAcceso, posesion, dispositivo, confiar, llaves?}` | `201 {cuenta, sesion, dispositivo, confianza?}` |
| `POST /v1/sesion` | `{correo, claveDeAcceso, dispositivo, confianza?}` | `200 {sesion, dispositivo}` con un equipo de confianza; si no, `202 {reto}` y un código al correo |
| `POST /v1/sesion/codigo` | `{reto, codigo, confiar}` | `200 {sesion, dispositivo, confianza?}` |
| `DELETE /v1/sesion` | — | `200` |

- `sal`: 16 bytes que elige el cliente al registrarse. `claveDeAcceso` y `posesion`: 32 bytes cada una.
- `argon2`: **nunca por debajo de 64 MiB, 3 pasadas**; el servidor no lo acepta y el cliente no lo usa.
- La sesión va en `Authorization: Bearer s1.<cuenta>.<secreto>`. Caduca a los 30 días sin uso y a los 90
  en todo caso. El testigo de confianza (`c1.…`) dura 90 días y se renueva al usarlo.
- El código por correo llega **solo tras comprobar la contraseña**, y como mucho cinco por hora.

### La bóveda

| Ruta | Envía | Devuelve |
|---|---|---|
| `GET /v1/boveda` | `If-None-Match: "17"` opcional | `200` y la bóveda, con `ETag: "17"`; `304`; `404` si aún no hay |
| `PUT /v1/boveda` | la bóveda, con `If-Match: "17"` | `200 {version: 18}`; `412 {error, version}` si ya no es la 17 |
| `GET /v1/boveda/versiones` | — | `[{version, fecha, tamano}]`, la más nueva primero |
| `GET /v1/boveda/versiones/17` | — | esa versión |
| `PUT /v1/cuenta/clave` | `{posesion, sal, argon2, claveDeAcceso, version, documento}` | `200 {version, sesion?}` |

- **El `ETag` puede llegar débil** (`W/"17"`): Cloudflare lo debilita al comprimir la respuesta. El
  cliente tiene que leer las dos formas; el servidor acepta las dos de vuelta.
- La primera subida va sobre la versión `0`. La versión la asigna el servidor, y es siempre la anterior + 1.
- **Se conservan las diez últimas versiones y la última de cada uno de los treinta días anteriores**.
- Tope: 8 MB por bóveda y 60 subidas por hora.
- Cambiar la contraseña es **todo o nada**: bóveda nueva, verificador nuevo y las demás sesiones fuera.
  Se demuestra con la `posesion`, no con la contraseña vieja.

### Recuperar, equipos y borrar

| Ruta | Envía | Devuelve |
|---|---|---|
| `POST /v1/recuperacion/inicio` | `{correo}` | `202`, siempre |
| `POST /v1/recuperacion/codigo` | `{correo, codigo}` | `{reto, sobre: {contenedor, codificacion}}`: **solo el sobre de recuperación** |
| `POST /v1/recuperacion/fin` | `{reto, posesion, dispositivo}` | `{sesion, dispositivo}`, **restringida**: solo baja la bóveda y cambia la contraseña |
| `GET /v1/dispositivos` | — | `[{id, nombre, creado, visto, actual}]` |
| `DELETE /v1/dispositivos/<id>` | — | `200`, y ese equipo se queda sin sesión |
| `POST /v1/cuenta/borrado` | — | `202 {reto}`, y un código al correo |
| `DELETE /v1/cuenta` | `{claveDeAcceso, reto, codigo}` | `200`. No tiene vuelta atrás |
| `GET /v1/cuenta/exportacion` | — | lo que hay de la cuenta, con la bóveda tal cual: cifrada |

## Desplegarlo: lo que tiene que hacer quien administra Cloudflare

Una vez, en este orden. **Ningún secreto pasa por el chat ni por el repositorio.**

1. **Crear las dos bases D1 en la UE**. La jurisdicción **solo se puede elegir al crearlas**: en el
   panel de Cloudflare, *Storage & databases → D1 → Create*, con la ubicación en **jurisdicción
   `EU`** (no una «preferencia de ubicación», que no garantiza nada). Nombres: `esfinge-cuentas` y
   `esfinge-cuentas-pruebas`. Sus identificadores no son secretos: van en `wrangler.jsonc`.
2. **Un token de la API** para GitHub, en *My Profile → API Tokens*, con la plantilla «Edit Cloudflare
   Workers» más *Account · D1 · Edit*, limitado a la cuenta WebCafeína y a la zona `webcafeina.com`.
   Se guarda en GitHub como secreto `CLOUDFLARE_API_TOKEN`, y el identificador de la cuenta como
   `CLOUDFLARE_ACCOUNT_ID`. Basta con eso también para el dominio propio (comprobado al desplegar).
3. **Desplegar el de pruebas**: *Actions → Servidor de cuentas → Run workflow → pruebas*.
4. **Sus secretos**, en el panel del Worker (*Settings → Variables and Secrets*, tipo *Secret*). Cada
   uno se genera en el terminal con `openssl rand -hex 32`, **distinto para cada uno y para cada
   Worker**:
   - `PIMIENTA`: firma los verificadores. **No se cambia nunca**: cambiarla deja fuera a todas las
     cuentas.
   - `SECRETO_PRELOGIN`: las sales inventadas y las IP.
   - `RESEND_API_KEY`: solo en producción. Una clave de Resend con permiso **solo de envío** y **solo
     para `webcafeina.com`**.

   Sin ellos el servidor contesta `503` a todo: no arranca con un secreto que falte.
5. **Producción**, lo mismo con `produccion`: el entorno de GitHub solo deja desplegar desde `main`.
   Vive en `https://esfinge-cuentas.webcafeina.com`.

**Hecho todo el 2026-09-18.** Bases: `esfinge-cuentas` (`0122dfb4-…`) y `esfinge-cuentas-pruebas`
(`62a934bf-…`), las dos con jurisdicción `eu`. El de pruebas vive en
`https://esfinge-cuentas-pruebas.webcafe-na.workers.dev`.

**Abrir el registro** (`REGISTRO: "abierto"`) es una decisión de producto (ADR 0035); pide la política y
las condiciones de uso publicadas. Hasta entonces, quién entra se decide en la tabla `admision`:

```sh
pnpm exec wrangler d1 execute BD --remote --command "INSERT INTO admision (patron) VALUES ('@webcafeina.com')"
```
