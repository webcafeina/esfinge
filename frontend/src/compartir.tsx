/**
 * Compartir una copia con otra cuenta, y el buzón de lo que llega (ADR 0043).
 *
 * Dos pantallas pequeñas y una idea que las gobierna: **la huella se enseña
 * siempre y se explica para qué sirve**. Es lo único que protege del servidor en
 * el primer envío, y una huella que nadie mira no protege nada.
 *
 * Por eso aquí no se dice «esta persona tiene cuenta» —el servidor contesta lo
 * mismo la tenga o no, a propósito— sino «compara esto con quien lo recibe».
 */

import { useEffect, useState } from "react";
import { esfinge, type EntradaBoveda, type EnvioRecibido } from "./puente";

/** La huella, en su tipografía y partida como se dicta. */
function Huella({ valor }: { valor: string }) {
  return <code className="huella seleccionable">{valor}</code>;
}

/**
 * Mandar una copia: el correo, la huella de quien la recibe y el aviso de qué
 * significa mandarla.
 */
export function Compartir({
  entrada,
  alVolver,
  alHecho,
}: {
  entrada: EntradaBoveda;
  alVolver: () => void;
  alHecho: (dicho: string) => void;
}) {
  const [correo, setCorreo] = useState("");
  const [huella, setHuella] = useState("");
  const [mia, setMia] = useState("");
  const [error, setError] = useState("");
  const [trabajando, setTrabajando] = useState(false);

  useEffect(() => {
    esfinge
      .miIdentidad()
      .then((i) => setMia(i.huella))
      .catch(() => setMia(""));
  }, []);

  const mirar = async () => {
    setError("");
    setHuella("");
    setTrabajando(true);
    try {
      setHuella((await esfinge.huellaDe(correo)).huella);
    } catch (e) {
      setError(String(e));
    } finally {
      setTrabajando(false);
    }
  };

  const mandar = async () => {
    setError("");
    setTrabajando(true);
    try {
      await esfinge.mandarCopia(entrada.id, correo);
      alHecho(`Copia de «${entrada.titulo || "Sin título"}» mandada a ${correo}.`);
    } catch (e) {
      setError(String(e));
      setTrabajando(false);
    }
  };

  return (
    <div className="panel">
      <div className="boveda-barra">
        <button onClick={alVolver}>← Volver</button>
      </div>

      <h2>Mandar una copia</h2>
      <p className="nota">
        {entrada.titulo || "Sin título"}
        {entrada.usuario ? ` · ${entrada.usuario}` : ""}
      </p>

      {/* Lo que hay que entender antes de pulsar, y no después. */}
      <p className="aviso">
        Se manda <strong>una copia</strong>, cifrada para quien la recibe. Al llegar es suya: si
        luego cambias la contraseña aquí, la suya se queda como está y hay que volver a mandarla.
      </p>

      <div className="grupo">
        <div>
          <label htmlFor="compartir-correo">Correo de quien la recibe</label>
          <input
            id="compartir-correo"
            type="email"
            autoComplete="off"
            spellCheck={false}
            value={correo}
            onChange={(e) => {
              setCorreo(e.target.value);
              setHuella("");
            }}
            onKeyDown={(e) => e.key === "Enter" && correo && !huella && mirar()}
          />
        </div>
      </div>

      {error && <p className="error">{error}</p>}

      {huella && (
        <>
          <p className="nota">Su huella es:</p>
          <p>
            <Huella valor={huella} />
          </p>
          <p className="aviso">
            <strong>Compárala con quien va a recibirla</strong>, por teléfono o en persona. Si no
            coincide, no la mandes: no sabrás quién está al otro lado. Esfinge no puede decirte si
            esa dirección tiene cuenta.
          </p>
        </>
      )}

      <div className="botones">
        {huella ? (
          <button className="principal" onClick={mandar} disabled={trabajando}>
            {trabajando ? "Mandando…" : "Mandar la copia"}
          </button>
        ) : (
          <button className="principal" onClick={mirar} disabled={!correo || trabajando}>
            {trabajando ? "Mirando…" : "Ver su huella"}
          </button>
        )}
      </div>

      {mia && (
        <p className="nota">
          La tuya, por si te la piden: <Huella valor={mia} />
        </p>
      )}
    </div>
  );
}

/** El buzón: lo que te han mandado, para aceptarlo o tirarlo. */
export function Buzon({
  envios,
  alVolver,
  alCambiar,
}: {
  envios: EnvioRecibido[];
  alVolver: () => void;
  alCambiar: (dicho: string) => void;
}) {
  const [trabajando, setTrabajando] = useState("");
  // Lo que se dice tras aceptar o tirar **se dice aquí**, no al volver: quien
  // acaba de pulsar sigue mirando esta pantalla.
  const [dicho, setDicho] = useState("");

  const hacer = async (id: string, que: "aceptar" | "tirar") => {
    setTrabajando(id);
    try {
      if (que === "aceptar") {
        await esfinge.aceptarDelBuzon(id);
        setDicho("Copia guardada en tu bóveda.");
        alCambiar("Copia guardada en tu bóveda.");
      } else {
        await esfinge.tirarDelBuzon(id);
        setDicho("Envío descartado.");
        alCambiar("Envío descartado.");
      }
    } catch (e) {
      setDicho(String(e));
      alCambiar(String(e));
    } finally {
      setTrabajando("");
    }
  };

  return (
    <div className="panel">
      <div className="boveda-barra">
        <button onClick={alVolver}>← Volver</button>
      </div>

      <h2>Te han mandado</h2>
      {envios.length === 0 ? (
        <p className="nota">Aquí no hay nada esperando.</p>
      ) : (
        <ul className="lista-papelera">
          {envios.map((e) => (
            <li key={e.id}>
              <span className="nombre">
                {e.error ? "No se puede abrir" : e.titulo || "Sin título"}
              </span>
              <span className="nota">
                {e.error ? e.error : <>De <Huella valor={e.huella} /></>}
              </span>
              <span className="acciones">
                {!e.error && (
                  <button
                    className="principal"
                    disabled={trabajando === e.id}
                    onClick={() => hacer(e.id, "aceptar")}
                  >
                    Guardar
                  </button>
                )}
                <button disabled={trabajando === e.id} onClick={() => hacer(e.id, "tirar")}>
                  Descartar
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}

      {dicho && <p className="exito">{dicho}</p>}

      {envios.length > 0 && (
        <p className="nota">
          Comprueba la huella con quien te lo manda antes de guardarlo. Lo que guardes entra en tu
          bóveda como una entrada tuya.
        </p>
      )}
    </div>
  );
}

