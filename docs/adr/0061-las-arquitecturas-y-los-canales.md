# 0061 · Las arquitecturas que faltaban, y los canales de distribución

**2026-11-12 a 2026-11-16.** Publicado por fases, de la **2.62.0** a la **2.66.1**.

## Contexto

Esfinge publicaba **28 adjuntos** y la mitad de la casa del cliente no estaba cubierta. La línea de
comandos ya cruzaba los seis objetivos y macOS era universal desde hacía tiempo, así que Apple
Silicon iba nativo; lo que faltaba era **la aplicación con ventana fuera de amd64** y los paquetes
que no son `.deb`:

- **Windows 11 ARM**: no había instalador.
- **Linux arm64**: no había aplicación, ni `.deb` —`nfpm.yaml` llevaba `arch: amd64` a fuego, así que
  `dpkg` lo rechazaba— ni `tar.gz`, **aunque el actualizador ya lo buscaba**.
- **Fedora y Arch**: nada.
- Y no había **ningún canal**: se instala bajando un fichero de GitHub.

El cliente tiene **todos sus Macs en Apple Silicon** y sobre ellos máquinas virtuales de Windows 11
ARM, Ubuntu, Fedora y Arch, todas ARM: cuatro de sus cinco entornos de prueba no tenían paquete.

Y había un fallo latente que se habría visto el primer día que existiera un `.rpm`:
`instaladoConPaquete()` miraba solo si el binario vivía en `/usr/bin`, así que **a una Fedora se le
habría ofrecido un `.deb`**.

## Decisión

**Decidido con el cliente** (2026-11-12 y 2026-11-16):

| Tema | Decisión |
|---|---|
| Alcance | La matriz entera **y los canales**: APT, RPM, AUR, Homebrew y Chocolatey |
| Windows | Los dos instaladores se **renombran** con su arquitectura, aceptando que las instalaciones anteriores pierdan la autoactualización una vez. Se gasta a propósito la garantía del buscar por sufijo: no hay instalaciones de Windows fuera de sus VM, y a cambio los nombres no mienten nunca más |
| Dónde | `esfinge.webcafeina.com`, con su CNAME proxificado en Cloudflare, como Apolo |
| La clave GPG | **Propia de Esfinge** (RSA 4096, sin caducidad), no la compartida con Apolo: si se compromete una, no se caen los dos productos |
| Licencia | Una **excepción expresa** en el `LICENSE` para los canales de la casa y para recetas que bajen el fichero oficial y lo instalen sin modificar. Sin eso, el AUR y Chocolatey contradecían la licencia |
| AppImage | **Autosuficiente** (WebKitGTK dentro, ~87 MiB) y **con la ruta del MCP arreglada**, que en Apolo es deuda abierta |
| Vocabulario | `amd64`/`arm64` en todo, como el `Makefile`. `x86_64`/`aarch64` solo donde lo nombra la herramienta |
| La ficha de Chocolatey (2026-11-18) | **En inglés primero y en español detrás**, que es lo que pidió su moderación al devolver el paquete. No es una excepción a la regla de la casa: el programa sigue estando en español, y **la ficha lo dice con todas las letras** para que nadie lo instale creyendo otra cosa. winget no lo necesita, porque allí la ficha va por idioma |

## Alternativas descartadas

- **Dejar los nombres de los adjuntos donde estaban.** Vivían en siete sitios, y la publicación
  pasaba de 28 a 37 ficheros. En esta casa una regla en N sitios ya había costado cinco fallos, así
  que el catálogo (ADR 0060) fue lo primero y no un refactor de paso.
- **Colgar los canales de `on: release: published`**, que es lo que hace Apolo y lo que parece
  natural. **Ese evento no se dispara cuando la publicación la crea `GITHUB_TOKEN`** —regla de
  GitHub—: Apolo puede porque su publicación sale como borrador y la publica una persona, y la de
  Esfinge sale ya publicada. Copiarlo habría dado tres flujos que **no se ejecutan nunca, en
  silencio**, con el tap sirviendo una versión de hace meses. Van con `needs:` dentro de
  `publicar.yml`.
- **Marcadores de plantilla escritos a mano**, como `{{SHA_DEB_AMD64}}` en Apolo, con una tabla al
  lado que dice qué fichero es cada uno. Eso es una segunda lista de nombres. Aquí el marcador **es**
  la clave del catálogo.
- **Firmar el `.deb`**. En Debian lo que se firma es el índice del repositorio (`InRelease`), no cada
  paquete; firmar los dos no añade nada y sí un paso que puede fallar.
- **Que el `zap` de Homebrew se lleve la carpeta de la bóveda.** Son las contraseñas de alguien: un
  gestor de paquetes que las borra al desinstalar es la peor sorpresa posible.
- **Automatizar el `choco push` sin decir lo que no se puede prometer.** Se automatiza, pero la
  **aprobación es de su moderación** y tarda días: el flujo comprueba que la subida se acepte y nada
  más.

## Consecuencias

- **37 adjuntos** en cada publicación, y **doce trabajos** en el flujo: seis de compilación, el humo
  en los dos arcos, publicar, tiendas, la web con los repositorios y los canales.
- **Cinco claves más que custodiar** (`~/.config/esfinge-paquetes/claves/`, y una copia fuera del
  equipo): la GPG con su frase, la del AUR, la del tap y la de Chocolatey. **Si se pierde la GPG,
  quien tenga los repositorios puestos tiene que importar la nueva.**
- **Cada despliegue de la web rehace /apt y /rpm** desde la última publicación. Un despliegue que no
  los montara publicaría un sitio **sin** ellos, y eso no es «no actualizarlos», es borrárselos a
  quien los tenga puestos.
- **El `.rpm` va firmado**, porque el `esfinge.repo` lleva `gpgcheck=1`. Es lo que hace que `dnf` lo
  acepte, y lo que lo rechazaba cuando no lo estaba.
- **Cada paquete deja escrito su gestor** en `/usr/share/esfinge/gestor`, y con `pacman` la banda
  dice «la instala tu gestor de paquetes» en vez de ofrecer un `.deb` que ese sistema rechaza.
- **El suelo de glibc es el de `ubuntu-24.04`** (glibc 2.39): Debian 13, Ubuntu 24.04 y Fedora 40
  en adelante, no Debian 12. GitHub retiró las de 22.04, así que no había elección.

  **Y esto estaba escrito mal el 2026-11-16 por la mañana**, con las dos mitades falsas: decía «el
  de `ubuntu-latest` (24.04)», y `ubuntu-latest` **ya es 26.04** —glibc 2.43—. Lo destapó el
  cliente probando la AppImage en su VM. De ahí salieron dos cosas: que **la AppImage de amd64
  pedía GLIBC_2.43**, o sea que solo servía en Ubuntu 26.04 —justo lo contrario de para lo que
  existe—, y que **las dos arquitecturas se empaquetaban en Ubuntu distintos**, porque no hay
  `ubuntu-latest-arm` y la pata de ARM ya corría en 24.04: llevaban dentro glib distinta, y la de
  ARM cargaba el `gvfs` de su sistema de 26.04 en nuestra glib de 24.04 y reventaba con un símbolo
  que no existe. **Las dos patas van fijadas a 24.04 desde la 2.67.0.**

  Lo que **no** cambia: nuestros propios binarios piden `GLIBC_2.34` el de la ventana y nada los
  tres de la línea de comandos, que son estáticos. El `.deb` y el `.rpm` nunca tuvieron este
  problema, y se comprobó mirando los símbolos de lo publicado.
- **El AUR sigue sin cuenta**, porque su registro de usuarios está cerrado. Arch se sirve del
  PKGBUILD que publica la web, con `makepkg -si`, y el trabajo del AUR **avisa y sale bien** mientras
  sea así —perdonando ese error y ninguno más—.

## Verificación

Lo que **está comprobado**, y con qué:

- **El catálogo**: los 24 casos de seis objetivos por cuatro gestores, desde el VPS; y en producción
  la puerta de listas diciendo `✓ 37 adjuntos, los del catálogo y ninguno más`.
- **El `.deb` de arm64 y el `.rpm` de las dos arquitecturas**, disecados aquí: arquitectura
  declarada, binarios ELF del arco correcto y el `postinst` dejando escrito su gestor.
- **El humo, en los dos arcos y en cada publicación**: Fedora 41 instala el `.rpm` —con
  `rpm --checksig` diciendo «digests signatures OK»— y Debian 13 el `.deb`, sin ninguna biblioteca
  sin resolver.
- **La AppImage**: armada y abierta en un Ubuntu 24.04 de verdad, y en el corredor pelado un `ldd`
  donde lo único sin resolver es `libEGL`, que tiene que ser del equipo que la ejecuta.
- **Los canales, el 2026-11-16 con la 2.66.1**: `apt install esfinge` y `dnf install esfinge`
  **desde los repositorios**, con la firma verificada y la clave importada de `gpgkey=`; el tap
  sirviendo la fórmula y el cask; el PKGBUILD de la web construido con `makepkg` e instalado con
  `pacman` en un contenedor de Arch; y la subida a Chocolatey aceptada.
- **La cadena de la firma GPG**: importar desde el secreto, firmar y verificar **con la pública
  sola**, como hace `apt`.

- **Y en máquinas de verdad, el 2026-11-17**, que es lo que el 16 faltaba: `apt install` desde el
  repositorio en su Ubuntu ARM, `dnf install` en su Fedora ARM —con la clave nuestra importada, vista
  con `rpm -q gpg-pubkey`—, el PKGBUILD de la web en su Arch ARM, y el instalador en su Windows 11
  ARM, **con el icono de los `.esf` puesto en los tres sistemas**, que era lo último sin verificar de
  todo el proyecto.

- **Y el Mac, el 2026-11-19**, con lo que cierra la verificación por sistemas entera: la fórmula
  (`esfinge 2.69.0`, binario arm64 nativo) y el cask, con la aplicación abriéndose **sin que macOS la
  bloquee** —el `xattr` de la cuarentena contesta «No such xattr», que es lo que sustituye a la firma
  que no se paga— y `brew outdated --cask` **sin listarla**. No hizo falta `brew trust`: `brew
  install` confía en lo que se le pide explícitamente.

  **Y destapó dos fallos del cask que habían pasado todas las puertas**, los dos de la misma
  familia: `depends_on macos: ">= :high_sierra"` —forma en retirada, y un mínimo por debajo de Big
  Sur que Homebrew ha **desactivado**— y `#{appdir}` dentro de `postflight_steps`, donde ese método
  no existe. Cualquiera de los dos deja el cask «unreadable», **y eso tumba también `brew install`
  de la fórmula**, porque cargar el tap evalúa las dos recetas. El tap los sirvió desde el
  2026-11-16. Lo que lo cierra es una puerta nueva: un trabajo en **macOS** monta un tap de mentira
  y hace `brew info` de las dos recetas **antes** de empujarlas, y falla también si Homebrew sólo
  **avisa** —su `depends_on` de cadena estuvo meses en «deprecated» antes de morir—. El tap depende
  de esa puerta y lo afirma una prueba, porque quitar la dependencia es una línea que no se nota.

Lo que **no** está comprobado, y hay que decirlo:
- **Que Chocolatey apruebe el paquete**, que es de su moderación, y ahí hay algo aprendido: el
  2026-11-18 **lo devolvió**, y no por el paquete sino **por el idioma de la ficha** —«the
  description is only in Spanish»—, con las dos sumas ya comprobadas por su moderador. Corregida y
  reempujada **la misma versión**, que es lo que piden; `choco install` sigue sin poderse probar
  hasta que se apruebe.
- **Un Windows amd64 nativo**: nadie de la casa tiene uno. Su instalador lo ejercita solo la máquina
  de GitHub.
- **Si Cloudflare cachea los índices de /apt**, que está en `deuda.md`: por los nombres sin extensión
  no debería, pero no está medido.
