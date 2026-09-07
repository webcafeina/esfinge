# ADR 0009 — El puente entre la interfaz y Go tiene dos caminos

**Fecha:** 2026-09-07 · **Estado:** aceptada

## Contexto

La máquina donde se desarrolla Esfinge no tiene entorno gráfico ni el webview del sistema, y no hay
`sudo` sin contraseña para instalarlo. Sin resolver eso, la interfaz solo se podría probar con datos
inventados, que es la clase de prueba que pasa siempre y no descubre nada.

## Decisión

La interfaz llama a Go a través de `frontend/src/puente.ts`, que tiene dos caminos y elige solo:

- **Wails**, cuando la aplicación está empaquetada. La llamada no sale del proceso.
- **HTTP**, durante el desarrollo, contra un servidor que publica los mismos métodos por reflexión.

Los diálogos del escritorio entran por la interfaz `app.Sistema`, con una implementación de Wails en
producción y otra de mentira en desarrollo. El servidor va tras la etiqueta de compilación `dev`.

## Alternativas descartadas

- **Probar solo con datos simulados.** No habría descubierto ninguno de los fallos que descubrió.
- **Instalar el entorno gráfico en la máquina de desarrollo.** Exige `sudo` con contraseña y no
  resuelve el caso de Windows y macOS de todas formas.

## Consecuencias

- Se puede recorrer la interfaz entera con Playwright contra el mismo Go que llevará la ventana. Eso
  ya destapó, entre otros, que el aviso de «si pierdes la clave» salía dos veces en la misma
  pantalla y que el resultado quedaba por debajo del borde con sus botones fuera de la vista.
- El servidor de desarrollo **no entra en lo que se distribuye**: un servidor HTTP en el binario del
  cliente, por local que sea, es una puerta que nadie ha pedido.
- Publicar por reflexión evita mantener una lista de métodos a mano, a cambio de que un cambio de
  firma falle al ejecutar y no al compilar.

## Verificación

Dieciséis pruebas con Playwright en tema claro y oscuro recorren cifrar, descifrar, generar, tandas
de ficheros e historial. Levantan ellas mismas el servidor de Go y Vite, así que corren igual aquí
que en integración continua.
