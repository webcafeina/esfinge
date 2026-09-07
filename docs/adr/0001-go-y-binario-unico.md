# ADR 0001 — Esfinge se escribe en Go y se entrega como un binario por plataforma

**Fecha:** 2026-09-04 · **Estado:** aceptada

## Contexto

La herramienta la iban a usar dos perfiles muy distintos: quien la encarga, cómodo en un terminal, y
un cliente que no vive ahí. Había que elegir con qué escribirla sabiendo que el segundo no iba a
instalar un intérprete ni a pelearse con dependencias.

## Decisión

Go, compilado sin cgo, con un binario autocontenido por sistema y arquitectura. Nada que instalar:
se descarga el fichero y se ejecuta.

## Alternativas descartadas

- **Python con Rich o Textual.** Desarrollo más rápido y ecosistema criptográfico sólido, pero
  obliga al cliente a tener Python, y no se puede compilar en cruz desde Linux hacia macOS.
- **Node con Ink.** Mismo problema, y el cliente tendría que instalar Node.

## Consecuencias

- Los seis objetivos —Linux, macOS y Windows, en amd64 y arm64— salen de una sola máquina Linux
  mientras no haya cgo. Eso se mantuvo cierto para la línea de comandos y **dejó de serlo** para la
  aplicación con ventana, que necesita el webview de cada sistema: ver [ADR 0006](0006-de-terminal-a-ventana.md).
- Los binarios rondan los 6 MB, que es lo que pesa el runtime de Go.

## Verificación

Los seis objetivos se compilan en cada publicación desde `make publicar`, y se comprobó a mano que
el binario de macOS arm64 lleva la firma ad-hoc que Apple exige (Go la añade al compilar en cruz).
