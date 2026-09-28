# El cartero

Recibe un correo **ya compuesto** por el servidor de cuentas y lo entrega por el relé
SMTP de Google Workspace. Vive en el VPS y es unas ciento cincuenta líneas.

## Por qué existe

El servidor de cuentas vive en Cloudflare y **no puede entregar el correo él mismo**.
No por una limitación de la plataforma —`connect()` abre TCP saliente y el 465 y el
587 funcionan— sino porque **Google rechaza la autenticación SMTP cuando la conexión
sale de Cloudflare**: contesta `535 5.7.8 … p=BadCredentials` a las mismas
credenciales que acepta desde cualquier otra máquina.

Eso se comprobó, no se dedujo (ADR 0045):

| Hipótesis | Cómo se descartó |
|---|---|
| El secreto guardado no es el que creemos | La misma huella SHA-256 en Cloudflare y en el Mac |
| Un espacio pegado al copiarlo | Dieciséis caracteres, sin espacios |
| Nuestro `AUTH PLAIN` está mal | Con `AUTH LOGIN` Google acepta el usuario y rechaza la clave igual |
| La entrada nueva del relé está mal | `smtp.gmail.com`, que no la usa, falla igual |
| Google bloquea por ubicación | Diría `p=WebLoginRequired`, y no hubo alerta de seguridad |

La IPv4 del VPS **sí** está autorizada en el relé —es la que usa Cronos— y ahí no
hacen falta ni usuario ni contraseña. De ahí esta pieza.

## Qué hace y qué no

Recibe un mensaje RFC 822 entero, lo revisa y lo entrega. **No compone el correo**
—eso sigue en el Worker, donde están la maqueta y sus pruebas—, no guarda nada y no
sabe nada de cuentas.

La revisión no es ceremonia. Este servicio **manda correo como webcafeína a quien se
le diga**, y un correo creíble desde el propio dominio de un gestor de contraseñas es
exactamente la pieza que le falta a quien quiera pescar a sus usuarios. El secreto
compartido lo cierra, pero un secreto se filtra, así que detrás hay dos reglas:

- **El remitente es el nuestro, la cabecera entera y exacta.** Comparar solo la
  dirección dejaría pasar `Banco Santander <esfinge@webcafeina.com>`.
- **Un solo destinatario**, y ni `Cc` ni `Bcc`.

## Cómo se prueba

```sh
pnpm install
pnpm run comprobar   # tipos y pruebas
```

Las pruebas de `relevo.ts` se **mutaron** una a una antes de darlas por buenas: si se
compara el remitente solo por la dirección, si se dejan de mirar las copias, si gana
la última cabecera en vez de la primera o si el secreto se compara con `===`, se ponen
rojas. Lo que **no** prueban: que la comparación del secreto sea de tiempo constante
—eso una prueba no lo mide— ni el diálogo SMTP de verdad, que es de nodemailer.

## Cómo se pone en el VPS

Todo esto es **a mano y una vez**. Los cinco primeros pasos son los de
`~/produccion/webcafeina-local/sites/LEEME.md`, que es de donde salen.

1. **El secreto compartido**, en hexadecimal —viaja en una cabecera, pero la costumbre
   de la casa es que un secreto no lleve caracteres que rompan una URL—:

   ```sh
   openssl rand -hex 32
   ```

   El mismo valor va en dos sitios: `cartero/.env` aquí y
   `wrangler secret put CARTERO_SECRETO` en los dos Workers.

2. **El `.env`**, al lado del `docker-compose.yml` y fuera de git:

   ```sh
   CARTERO_SECRETO=el-de-arriba
   SMTP_FROM=Esfinge <esfinge@webcafeina.com>
   ```

3. **La red de borde y el sitio de Caddy**:

   ```sh
   docker network create esfinge-borde
   docker network connect esfinge-borde webcafeina-local-caddy-1
   cp esfinge.caddy ~/produccion/webcafeina-local/sites/esfinge.caddy
   docker exec webcafeina-local-caddy-1 caddy validate --config /etc/caddy/Caddyfile
   docker exec webcafeina-local-caddy-1 caddy reload  --config /etc/caddy/Caddyfile
   ```

   Y declarar `esfinge-borde` en el `docker-compose` de Caddy, para que sobreviva a
   recrearlo.

4. **El DNS**: `cartero.webcafeina.com` apuntando al VPS. **Sin proxy de Cloudflare**
   no hace falta —aquí no hay conexiones largas—, así que con proxy está bien y además
   esconde la IP de origen.

5. **Levantarlo**:

   ```sh
   docker compose up -d --build
   curl https://cartero.webcafeina.com/salud    # {"salud":"bien"}
   ```

## Lo que no se puede comprobar desde la máquina de desarrollo

Que el relé acepte de verdad. Eso solo se ve entregando, desde el VPS y contra Google.
Por eso el paso de comprobación manual del plan no es opcional.
