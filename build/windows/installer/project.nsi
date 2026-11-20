# El instalador de Esfinge para Windows.
#
# **Es la plantilla de Wails v2.15.0 con una sola diferencia: lleva el puente del
# navegador** (`esfinge-puente.exe`) junto a Esfinge. Sin él la extensión no puede
# funcionar en Windows: el navegador lanza lo que dice el manifiesto, y Esfinge lo
# busca al lado de su propio ejecutable (`rutaDelPuente`). Wails usa este fichero
# tal cual si está en el proyecto; `wails_tools.nsh` lo regenera en cada compilación
# y no se sube. El puente lo compila `publicar.yml` en esta misma carpeta antes de
# llamar a Wails, que es desde donde `makensis` resuelve la ruta relativa.
#
Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them with the values from ProjectInfo.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows to use this project.nsi manually
## from outside of Wails for debugging and development of the installer.
##
## For development first make a wails nsis build to populate the "wails_tools.nsh":
## > wails build --target windows/amd64 --nsis
## Then you can call makensis on this file with specifying the path to your binary:
## For a AMD64 only installer:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## For a ARM64 only installer:
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## For a installer with both architectures:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## The following information is taken from the ProjectInfo file, but they can be overwritten here.
####
## !define INFO_PROJECTNAME    "MyProject" # Default "{{.Name}}"
## !define INFO_COMPANYNAME    "MyCompany" # Default "{{.Info.CompanyName}}"
## !define INFO_PRODUCTNAME    "MyProduct" # Default "{{.Info.ProductName}}"
## !define INFO_PRODUCTVERSION "1.0.0"     # Default "{{.Info.ProductVersion}}"
## !define INFO_COPYRIGHT      "Copyright" # Default "{{.Info.Copyright}}"
###
## !define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
## !define UNINST_KEY_NAME     "UninstKeyInRegistry"  # Default "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
####
## !define REQUEST_EXECUTION_LEVEL "admin"            # Default "admin"  see also https://nsis.sourceforge.io/Docs/Chapter4.html
####
## Include the wails tools
####
!include "wails_tools.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"

# ── El instalador es de Esfinge, no el de nadie ──────────────────────────────
#
# Hasta la 2.70.0 esto era la plantilla de Wails tal cual: **en inglés**, con la
# imagen azul de serie de NSIS y con «Nullsoft Install System» al pie, en un
# producto que está entero en español. Lo vio el cliente instalándola.
#
# Las dos imágenes salen de `build/windows/instalador/*.svg` y las convierte a
# BMP `frontend/herramientas/bmp.mjs`, con el mismo Chromium que rasteriza los
# iconos. **Van sin una sola letra dentro**: NSIS las estira hasta llenar su
# control y el control crece con el DPI, así que a 200 % se dibujan al doble sin
# que la documentación diga con qué calidad. Lo que sí sale nítido son los
# textos, que los pinta Windows —de ahí `ManifestDPIAware` de arriba—.
#
# Los tamaños son los que recomienda MUI2 (164×314 y 150×57) y **no hay forma
# soportada de dar una versión al doble**: no existe ninguna macro que acepte un
# segundo fichero por DPI. Comprobado en su documentación antes de dibujarlas.
!define MUI_WELCOMEFINISHPAGE_BITMAP "..\instalador\lado.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "..\instalador\lado.bmp"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_RIGHT
!define MUI_HEADERIMAGE_BITMAP "..\instalador\cabecera.bmp"
!define MUI_HEADERIMAGE_UNBITMAP "..\instalador\cabecera.bmp"

BrandingText "Esfinge · Webcafeína"

# Las cuatro frases que se leen enteras, dichas por nosotros en vez de por la
# traducción genérica de NSIS. El resto —los botones, los avisos— lo pone el
# idioma, que para eso está.
#
# **Por `MUI_*` y no por `LangString`**, que es la vía documentada para esto: con
# `LangString` las mismas cadenas quedan puestas dos veces y `makensis` saca
# cuatro avisos de «set multiple times, wasting space». Funcionaba, y un aviso
# que nadie va a leer ya ha costado bastante en esta casa.
!define MUI_WELCOMEPAGE_TITLE "Esfinge"
!define MUI_WELCOMEPAGE_TEXT "Cifra y descifra contraseñas y ficheros con una clave.$\r$\n$\r$\nSe instalará en este equipo; tarda unos segundos.$\r$\n$\r$\nDe Webcafeína."
!define MUI_FINISHPAGE_TITLE "Esfinge está lista"
!define MUI_FINISHPAGE_TEXT "Ya puedes abrirla desde el menú de inicio.$\r$\n$\r$\nLo que cifres solo se abre con la clave que pongas: si la pierdes, se pierde el contenido."
!define MUI_UNCONFIRMPAGE_TEXT_TOP "Se quitará Esfinge de este equipo. Tus ficheros cifrados y tu bóveda no se tocan: viven fuera de la carpeta del programa y siguen donde estén."

!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

# Y al terminar, abrirla. Marcado de fábrica: quien acaba de instalar un
# programa lo que quiere es verlo.
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "Abrir Esfinge"

!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # Adds a EULA page to the installer
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES # Uinstalling page

# **En español**, que es el idioma del producto. El fichero de idioma viene con
# NSIS; lo que hace falta es que el guion sea Unicode, y lo es (arriba).
!insertmacro MUI_LANGUAGE "Spanish"



## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
!ifdef WAILS_INSTALL_SCOPE
  !if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
  !else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
  !endif
!else
  InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!endif # Default installing folder ($PROGRAMFILES is Program Files folder).
ShowInstDetails show # This will always show the installation details.

!include "FileFunc.nsh"

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    !insertmacro wails.files

    # El puente del navegador, junto a Esfinge (ver la cabecera).
    File "esfinge-puente.exe"

    # Y el servidor MCP, que es lo que lanza un agente de IA (ADR 0054). Junto a
    # Esfinge por lo mismo: la configuración que Ajustes enseña apunta aquí con la
    # ruta entera.
    File "esfinge-mcp.exe"

    # **De qué gestor viene esta copia** (ADR 0063).
    #
    # En Linux lo deja el `postinstall` del paquete en /usr/share/esfinge/gestor; en Windows
    # no hay un sitio compartido, así que lo escribe el instalador **al lado del programa** y
    # sólo cuando quien lo lanza se lo pide con `/GESTOR=`. Un doble clic no pasa nada y no
    # deja fichero, que es justo lo que lo distingue.
    #
    # Sin esto, a quien instaló con `choco` o con `winget` se le ofrecía actualizarse por su
    # cuenta y el gestor se quedaba viendo una versión vieja para siempre.
    ${GetParameters} $R0
    ${GetOptions} $R0 "/GESTOR=" $R1
    ${If} $R1 != ""
        FileOpen $R2 "$INSTDIR\gestor" w
        FileWrite $R2 "$R1"
        FileClose $R2
    ${EndIf}

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath

    # RMDir /r ya se lo lleva con la carpeta, pero se dice: que desinstalar quita el
    # puente no debería depender de dónde esté instalado.
    Delete "$INSTDIR\esfinge-puente.exe"
    Delete "$INSTDIR\esfinge-mcp.exe"

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
SectionEnd
