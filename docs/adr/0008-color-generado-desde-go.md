# ADR 0008 — El color se genera desde Go, no se escribe en el CSS

**Fecha:** 2026-09-07 · **Estado:** aceptada

## Contexto

La instrucción de la casa dice que el contraste se mide siempre, con la herramienta del propio
proyecto, después de tocar la paleta. En la 1.x eso era fácil: la paleta vivía en Go y la interfaz
también. Con una interfaz en CSS, el color podía acabar viviendo en dos sitios.

## Decisión

`internal/tema` es la única fuente de verdad y **genera** `frontend/src/tokens.css`. El CSS no se
edita a mano.

## Alternativas descartadas

- **Escribir los colores en el CSS y medirlos con un script aparte**, como hace Tempero con
  `contraste.mjs`. Es lo que ya funciona en otro proyecto, pero aquí la paleta la necesitan también
  la línea de comandos y los tests de Go: tenerla en CSS obligaría a duplicarla.

## Consecuencias

- Cambiar un color es cambiar Go y ejecutar `make tokens`.
- Un test compara el fichero del repositorio con lo que Go genera ahora mismo y falla si se ha
  quedado atrás. Sin él, cambiar un color y olvidar regenerar dejaría la interfaz pintando los
  colores de ayer, y los tests de contraste seguirían en verde porque miden el tema, no el fichero.
- Los espaciados, radios y tipografías van por el mismo camino, por el mismo motivo.

## Verificación

`TestContrasteDeLosDosTemas` mide las parejas reales; `TestLosTokensEstanAlDia` compara el CSS con
lo generado; `TestSePublicanTodosLosColores` comprueba que ningún color del tema se queda sin
publicar. Los tres corren en cada compilación.
