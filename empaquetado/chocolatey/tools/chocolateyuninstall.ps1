# Desinstala Esfinge con el desinstalador que deja su propio instalador de NSIS, en silencio. Lo de
# antes de la primera línea en blanco no llega al paquete.

$ErrorActionPreference = 'Stop'

$claves = @(Get-UninstallRegistryKey -SoftwareName 'Esfinge*')

if ($claves.Count -eq 1) {
  $desinstalador = $claves[0].UninstallString -replace '"', ''
  Uninstall-ChocolateyPackage -PackageName $env:ChocolateyPackageName `
    -FileType 'exe' -SilentArgs '/S' -File $desinstalador -ValidExitCodes @(0)
} elseif ($claves.Count -eq 0) {
  Write-Warning "Esfinge no está instalada; no hay nada que desinstalar."
} else {
  # Varias coincidencias: **no se adivina cuál**. Desinstalar la que no es deja a alguien sin su
  # gestor de contraseñas sin haberlo pedido.
  Write-Warning "Hay $($claves.Count) entradas que empiezan por «Esfinge»; desinstala a mano la que quieras:"
  $claves | ForEach-Object { Write-Warning "  $($_.DisplayName) — $($_.UninstallString)" }
}

# **Lo que no se borra: la bóveda.** Son las contraseñas de alguien, y un gestor de paquetes que las
# tira al desinstalar es la peor sorpresa posible. Viven en la carpeta de configuración del usuario
# y se quedan ahí; quien quiera irse del todo las borra él.
