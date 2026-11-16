# Plantilla: `herramientas/rellenar` pone la versión, los nombres y las sumas de los dos
# instaladores. Lo de antes de la primera línea en blanco no llega al paquete.

$ErrorActionPreference = 'Stop'

$version = '{{VERSION}}'
$base = "https://github.com/webcafeina/esfinge/releases/download/v$version"

# **Win32_Processor.Architecture 12 es ARM64.** Chocolatey no tiene un parámetro propio para ARM, así
# que el instalador de ARM64 va como «64 bits» cuando la máquina lo es. Sin esto, una máquina ARM se
# llevaría el instalador de x64: arranca por emulación y funciona, pero más lento y sin motivo.
$arm64 = (Get-CimInstance Win32_Processor | Select-Object -First 1).Architecture -eq 12
if ($arm64) {
  $url = "$base/{{NOMBRE:app-windows-arm64}}"
  $checksum = '{{SHA:app-windows-arm64}}'
} else {
  $url = "$base/{{NOMBRE:app-windows-amd64}}"
  $checksum = '{{SHA:app-windows-amd64}}'
}

# `/S` es lo que entiende el instalador de NSIS, y es **lo mismo que usa Esfinge para actualizarse
# a sí misma** (`reemplazarseEnWindows`): una bandera distinta aquí sería un segundo camino que
# nadie prueba.
$packageArgs = @{
  packageName    = $env:ChocolateyPackageName
  fileType       = 'exe'
  url64bit       = $url
  checksum64     = $checksum
  checksumType64 = 'sha256'
  silentArgs     = '/S'
  validExitCodes = @(0)
  softwareName   = 'Esfinge*'
}

Install-ChocolateyPackage @packageArgs
