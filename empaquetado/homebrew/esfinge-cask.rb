# Plantilla del cask de Esfinge. `canales.yml` la rellena en cada versión y la empuja a
# webcafeina/homebrew-esfinge como Casks/esfinge.rb.
#
# **Esfinge no está firmada ni notarizada** (ADR 0012, los 99 $/año no compensan para un cliente).
# Homebrew solo rechaza eso en su repositorio oficial; en un tap propio la instala, y quien la
# bloquearía al abrirla es macOS. Por eso se le quita la cuarentena al instalar — igual que en
# Apolo, y con la forma de hoy: el `postflight do` de antes está obsoleto desde Homebrew 7 y
# `--no-quarantine` ya no existe.
cask "esfinge" do
  version "{{VERSION}}"
  sha256 "{{SHA:app-darwin}}"

  url "https://github.com/webcafeina/esfinge/releases/download/v#{version}/{{NOMBRE:app-darwin}}"
  name "Esfinge"
  desc "Cifra y descifra contraseñas y ficheros con una clave"
  homepage "https://esfinge.webcafeina.com/"

  livecheck do
    url :url
    strategy :github_latest
  end

  # Esfinge se actualiza sola en macOS, con su banda de aviso (ADR 0014): Homebrew no tiene que
  # hacerlo por ella, y si lo intentara se pisarían.
  auto_updates true
  # **Sin mínimo, y eso es decir lo mismo que antes.** La intención era no dejar fuera a nadie: el
  # `LSMinimumSystemVersion` de `build/darwin/Info.plist` pone 10.13, y Homebrew no corre en nada
  # anterior a **Big Sur**, así que cualquier mínimo que declaremos por debajo de eso es redundante
  # — y **declararlo rompe el cask entero**: la forma de cadena (`">= :high_sierra"`) está en
  # retirada y el símbolo de una versión anterior a Big Sur está **desactivado**, «there is no
  # replacement». `depends_on :macos` es la forma documentada de decir «sólo macOS, sin mínimo»; el
  # día que haga falta uno de verdad, va como símbolo (`depends_on macos: :monterey`).
  depends_on :macos

  app "Esfinge.app"

  postflight_steps do
    run "/usr/bin/xattr",
        args:         ["-dr", "com.apple.quarantine", "#{appdir}/Esfinge.app"],
        must_succeed: false
  end

  # El identificador lo compone el Info.plist con «com.webcafeina.» y el nombre pasado por
  # `safeBundleID` de Wails, que **lo pasa a minúsculas** —leído en su fuente, que está en el
  # disco—. Con la caja mal, en un disco sensible a mayúsculas el «zap» no encontraría nada y
  # tampoco lo diría.
  #
  # (Y aquí no se escriben las llaves dobles de esa plantilla a propósito: `rellenar` se pone rojo
  # ante cualquier marcador que no sepa sustituir, y eso es justo lo que se quiere de él.)
  #
  # **La bóveda NO se va con «zap», y eso es a propósito.** Lo que hay en esa carpeta son las
  # contraseñas de alguien; un gestor de paquetes que las borra al desinstalar es la peor sorpresa
  # posible. Se van las cachés y el estado del webview, que se rehacen solos.
  zap trash: [
    "~/Library/Caches/com.webcafeina.esfinge",
    "~/Library/WebKit/com.webcafeina.esfinge",
    "~/Library/Saved Application State/com.webcafeina.esfinge.savedState",
  ]
end
