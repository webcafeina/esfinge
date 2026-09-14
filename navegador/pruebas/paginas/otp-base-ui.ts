/**
 * Un formulario de segundo factor hecho con el componente que usa Cloudflare.
 *
 * `OTPField` de Base UI, en React, **el paquete de verdad** y no una imitación:
 * lo que hay que comprobar es si ese componente se entera de lo que escribe
 * Esfinge, y eso solo lo sabe el componente. Una imitación diría lo que se le haya
 * enseñado a decir.
 *
 * Sin JSX a propósito, para no meter un compilador de React en las pruebas de la
 * extensión por un fichero de veinte líneas.
 */
import * as React from "react";
import { createRoot } from "react-dom/client";
import { OTPField } from "@base-ui/react/otp-field";

function Formulario() {
  const [valor, setValor] = React.useState("");
  return React.createElement(
    "form",
    { onSubmit: (e: React.FormEvent) => e.preventDefault() },
    React.createElement(
      OTPField.Root,
      { length: 6, value: valor, onValueChange: (v: string) => setValor(v) },
      ...[0, 1, 2, 3, 4, 5].map((i) =>
        React.createElement(OTPField.Input, { key: i, style: { width: 65, height: 65 } }),
      ),
    ),
    // Lo que el componente cree que vale. **Esto es lo que se comprueba**, no lo
    // que se ve en las casillas.
    React.createElement("output", { id: "valor" }, valor),
    React.createElement("button", { id: "verificar", disabled: valor.length !== 6 }, "Verificar"),
  );
}

createRoot(document.getElementById("app")!).render(React.createElement(Formulario));
