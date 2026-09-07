# ADR 0012 — No se firma ni se notariza con Apple

**Fecha:** 2026-09-04 · **Estado:** aceptada · **Revisar si** la aplicación pasa de un cliente

## Contexto

macOS y Windows avisan de que un programa descargado no está firmado por un desarrollador
identificado. Con una `.app` el aviso es más aparatoso que con un binario de terminal, y en macOS
reciente el «Abrir de todos modos» está más escondido.

## Decisión

No se firma. El paquete incluye instrucciones para quitar la cuarentena.

## Alternativas descartadas

- **Cuenta de Apple Developer**, 99 $ al año. Elimina el aviso del todo y además abriría la puerta a
  la actualización automática.

## Consecuencias

- Cada persona que instale Esfinge verá un aviso la primera vez y tendrá que dar un rodeo.
- La actualización automática queda descartada: exige firma.
- El binario sí lleva **firma ad-hoc**, que es obligatoria en Apple Silicon y que Go añade sola al
  compilar en cruz. Sin ella el sistema mata el proceso.

## Verificación

Comprobado leyendo las cabeceras Mach-O del paquete de macOS: las dos arquitecturas llevan
`LC_CODE_SIGNATURE`. Y comprobado por el humano, que abrió la aplicación en su Mac: descomprimiendo
el ZIP desde el Terminal no llegó ni a saltar el aviso, porque la cuarentena la propaga quien
descomprime, no el fichero.
