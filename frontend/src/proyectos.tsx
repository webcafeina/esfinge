// El apartado «Proyectos»: una bóveda por cliente (ADR 0050).
//
// Lo que esta pantalla tiene que decir y es fácil olvidar, porque no se ve
// mirándola:
//
//   - **Un proyecto no tiene clave de recuperación propia.** Quien ha creado una
//     bóveda antes espera la ceremonia de la clave —pantalla entera, insistiendo en
//     que se apunte—, así que su ausencia sin explicar parece un olvido o un fallo.
//     Lo dice el formulario de crear, **antes** de crear.
//   - ~~**Lo que se guarde aquí todavía no sale de este equipo.**~~ Lo decía un aviso
//     en esta pantalla mientras la sincronización no estuvo desplegada. **Desde el
//     2026-10-02 sincronizan**, y el aviso se quitó: un texto que miente en la
//     pantalla donde alguien decide guardar algo es peor que no decir nada. Lo vio el
//     cliente leyéndolo en su Mac el mismo día del despliegue.
//   - **Volver a la bóveda personal pide la contraseña maestra**, porque la personal
//     no se queda abierta por detrás: dos bóvedas abiertas a la vez es justo lo que
//     se descartó. Sin decirlo, parecería que Esfinge se ha bloqueado solo.
//
// Y una cosa que esta pantalla **no** hace: enseñar nombres con la bóveda cerrada.
// No es una decisión de aquí — es que no los tiene, porque viven dentro del cuerpo
// cifrado de la bóveda personal.

import { useCallback, useEffect, useState } from "react";
import { esfinge, type Proyecto } from "./puente";
import { Icono } from "./componentes";
import { Ceremonia } from "./boveda";

export function Proyectos({
  activo,
  alEntrar,
}: {
  /** Si la sección está a la vista: se recarga al entrar, como el historial. */
  activo: boolean;
  /** Se llama al conmutar, para llevar a la bóveda. */
  alEntrar: () => void;
}) {
  const [lista, setLista] = useState<Proyecto[] | null>(null);
  const [cerrada, setCerrada] = useState(false);
  const [error, setError] = useState("");
  const [busca, setBusca] = useState("");
  const [creando, setCreando] = useState(false);
  const [nombre, setNombre] = useState("");
  const [trabajando, setTrabajando] = useState(false);
  const [renombrando, setRenombrando] = useState("");
  const [otroNombre, setOtroNombre] = useState("");
  const [verArchivados, setVerArchivados] = useState(false);
  const [bajando, setBajando] = useState("");
  // Qué proyecto tiene desplegado «Al acabar…». Lo de terminar un proyecto se usa
  // una vez en su vida, así que no ocupa sitio en la fila del día a día.
  const [alAcabar, setAlAcabar] = useState("");
  // La clave de recuperación de una bóveda recién entregada: se enseña **una vez**
  // con la ceremonia de siempre y no se puede volver a pedir.
  const [entregada, setEntregada] = useState("");

  const recargar = useCallback(async () => {
    // **Se mira si hay bóveda abierta antes de pedir la lista.** No es una
    // optimización: pedirla con la bóveda cerrada es un error seguro, y un error
    // seguro deja un 400 en la consola del navegador —que es ruido en el sitio donde
    // se mira cuando algo va mal— y en la ventana, un viaje al puente para nada.
    // Es la regla de `ExportarLlaves`: lo que hace falta para decidir se mira antes
    // de pedirle a alguien que decida.
    try {
      const e = await esfinge.estadoBoveda();
      if (!e.abierta) {
        setLista(null);
        setCerrada(true);
        return;
      }
      setLista(await esfinge.proyectos());
      setCerrada(false);
      setError("");
    } catch (e) {
      setLista(null);
      setCerrada(true);
    }
  }, []);

  useEffect(() => {
    if (activo) void recargar();
  }, [activo, recargar]);

  // **La clave de recuperación de lo entregado, a pantalla completa y una sola
  // vez.** Es la misma ceremonia que al crear una bóveda, sin tocarla: quien reciba
  // el fichero la necesita tanto como su contraseña, y aquí no hay «vuélvemela a
  // enseñar».
  if (entregada) {
    return <Ceremonia clave={entregada} nueva={true} alSeguir={() => setEntregada("")} />;
  }

  if (cerrada) {
    return (
      <div className="panel">
        <p className="aviso">
          Abre tu bóveda para ver tus proyectos. Sus nombres viven dentro de ella, cifrados.
        </p>
      </div>
    );
  }

  const vivos = (lista ?? []).filter((p) => !p.archivado);
  const archivados = (lista ?? []).filter((p) => p.archivado);
  const filtra = (p: Proyecto) => p.nombre.toLowerCase().includes(busca.trim().toLowerCase());
  const visibles = vivos.filter(filtra);

  async function crear() {
    const n = nombre.trim();
    if (!n) return;
    setTrabajando(true);
    setError("");
    try {
      await esfinge.crearProyecto(n);
      setNombre("");
      setCreando(false);
      await recargar();
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  async function entrar(ref: string) {
    setError("");
    try {
      await esfinge.abrirProyecto(ref);
      alEntrar();
    } catch (e) {
      setError(mensaje(e));
    }
  }

  async function bajar(ref: string) {
    setBajando(ref);
    setError("");
    try {
      await esfinge.bajarProyecto(ref);
      await recargar();
    } catch (e) {
      setError(mensaje(e));
    } finally {
      setBajando("");
    }
  }

  async function archivar(ref: string, si: boolean) {
    setError("");
    try {
      await esfinge.archivarProyecto(ref, si);
      setAlAcabar("");
      await recargar();
    } catch (e) {
      setError(mensaje(e));
    }
  }

  async function renombrar(ref: string) {
    const n = otroNombre.trim();
    if (!n) return;
    setError("");
    try {
      await esfinge.renombrarProyecto(ref, n);
      setRenombrando("");
      await recargar();
    } catch (e) {
      setError(mensaje(e));
    }
  }

  return (
    <div className="panel">
      <p className="entradilla">
        Una bóveda por proyecto o por cliente, con lo mismo que la tuya dentro. Se abren con tu misma
        contraseña maestra y <strong>solo una está abierta a la vez</strong>.
      </p>

      {error && <p className="error">{error}</p>}

      <div className="botones">
        {!creando && (
          <button className="principal" onClick={() => setCreando(true)}>
            Nueva bóveda de proyecto
          </button>
        )}
      </div>

      {creando && (
        <div className="grupo">
          <div>
            <label htmlFor="proyecto-nombre">Nombre</label>
            <input
              id="proyecto-nombre"
              type="text"
              autoComplete="off"
              autoFocus
              value={nombre}
              onChange={(e) => setNombre(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && nombre.trim() !== "" && crear()}
            />
          </div>

          {/* **La ausencia de la ceremonia, explicada antes de crear.** */}
          <p className="nota">
            Se abre con tu misma contraseña maestra, así que{" "}
            <strong>no tiene clave de recuperación propia</strong>: la de tu bóveda la recupera igual.
          </p>

          <div className="botones">
            <button className="principal" onClick={crear} disabled={nombre.trim() === "" || trabajando}>
              {trabajando ? "Creando…" : "Crear"}
            </button>
            <button
              className="discreto"
              onClick={() => {
                setCreando(false);
                setNombre("");
                setError("");
              }}
            >
              Cancelar
            </button>
          </div>
        </div>
      )}

      {vivos.length > 2 && (
        <div>
          <label htmlFor="proyecto-buscar">Buscar</label>
          <input
            id="proyecto-buscar"
            type="search"
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
            placeholder="Nombre del proyecto"
          />
        </div>
      )}

      {lista != null && vivos.length === 0 && !creando && (
        <p className="nota">
          Todavía no tienes ninguna. Una bóveda de proyecto guarda lo mismo que la tuya —cuentas, notas,
          tarjetas, llaves de acceso, redes— y se puede entregar entera el día que el proyecto acabe.
        </p>
      )}

      {visibles.length > 0 && (
        <ul className="proyectos">
          {visibles.map((p) => (
            <li key={p.ref} data-activo={p.activo ? "si" : undefined}>
              {renombrando === p.ref ? (
                <div className="renombrar">
                  <label htmlFor={`renombrar-${p.ref}`}>Nombre</label>
                  <input
                    id={`renombrar-${p.ref}`}
                    type="text"
                    autoComplete="off"
                    autoFocus
                    value={otroNombre}
                    onChange={(e) => setOtroNombre(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && otroNombre.trim() !== "" && renombrar(p.ref)}
                  />
                  <button
                    className="principal"
                    onClick={() => renombrar(p.ref)}
                    disabled={otroNombre.trim() === ""}
                  >
                    Guardar
                  </button>
                  <button className="discreto" onClick={() => setRenombrando("")}>
                    Cancelar
                  </button>
                </div>
              ) : (
                <>
                  <button
                    className="abrir-proyecto"
                    onClick={() => entrar(p.ref)}
                    disabled={!p.enEsteEquipo || p.activo}
                    title={p.activo ? "Es la bóveda que estás usando" : undefined}
                  >
                    <Icono nombre="proyectos" />
                    <span className="nombre">{p.nombre}</span>
                    <span className="aparte">{segundaLinea(p)}</span>
                  </button>
                  {/* **Un proyecto dormido se baja desde aquí.** Sin este botón, un
                      equipo nuevo ve sus proyectos en la lista y no puede abrir
                      ninguno, que es peor que no verlos. */}
                  {!p.enEsteEquipo ? (
                    <button className="discreto" onClick={() => bajar(p.ref)} disabled={bajando === p.ref}>
                      {bajando === p.ref ? "Bajando…" : "Bajar a este equipo"}
                    </button>
                  ) : (
                    <>
                      <button
                        className="discreto"
                        onClick={() => {
                          setRenombrando(p.ref);
                          setOtroNombre(p.nombre);
                        }}
                      >
                        Cambiar el nombre
                      </button>
                      <button
                        className="discreto"
                        onClick={() => setAlAcabar(alAcabar === p.ref ? "" : p.ref)}
                        aria-expanded={alAcabar === p.ref}
                      >
                        Al acabar…
                      </button>
                    </>
                  )}
                </>
              )}
              {alAcabar === p.ref && (
                <AlAcabarElProyecto
                  proyecto={p}
                  alEntregar={(clave) => {
                    setAlAcabar("");
                    setEntregada(clave);
                  }}
                  alArchivar={() => archivar(p.ref, true)}
                  alBorrar={async () => {
                    setAlAcabar("");
                    await recargar();
                  }}
                  alFallar={setError}
                />
              )}
            </li>
          ))}
        </ul>
      )}

      {archivados.length > 0 && (
        <div className="archivados">
          <button className="discreto" onClick={() => setVerArchivados((v) => !v)}>
            {verArchivados ? "Ocultar los archivados" : `Archivados (${archivados.length})`}
          </button>
          {verArchivados && (
            <ul className="proyectos">
              {archivados.filter(filtra).map((p) => (
                <li key={p.ref}>
                  <button
                    className="abrir-proyecto"
                    onClick={() => entrar(p.ref)}
                    disabled={!p.enEsteEquipo}
                  >
                    <Icono nombre="proyectos" />
                    <span className="nombre">{p.nombre}</span>
                    <span className="aparte">{segundaLinea(p)}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * La segunda línea de una fila dice **lo que la distingue**, en este orden: si no
 * está aquí, si es la que se está usando, y cuándo se abrió por última vez.
 *
 * «Dormido en este equipo» no se escribe como un error, porque no lo es: el proyecto
 * existe y este ordenador no tiene su fichero todavía.
 */
function segundaLinea(p: Proyecto): string {
  if (!p.enEsteEquipo) return "Dormido en este equipo";
  if (p.activo) return "La estás usando";
  if (p.usado) return `Abierta ${cuando(p.usado)}`;
  return "Sin abrir todavía";
}

function cuando(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "alguna vez";
  const dias = Math.floor((Date.now() - d.getTime()) / 86_400_000);
  if (dias <= 0) return "hoy";
  if (dias === 1) return "ayer";
  if (dias < 30) return `hace ${dias} días`;
  return d.toLocaleDateString();
}

function mensaje(e: unknown): string {
  return (e instanceof Error ? e.message : String(e)).replace(/^Error:\s*/, "");
}

/**
 * Llevar una entrada a otra bóveda (ADR 0050, E3).
 *
 * Se elige el destino de una lista y se decide **mover o copiar**, y las dos cosas
 * se dicen con esas palabras porque no significan lo mismo: copiar deja dos
 * contraseñas que a partir de ahí se cambian por separado.
 *
 * Lo que no se ve y hay que decir: **el secreto no cruza el puente**. La ventana
 * manda dos identificadores y Go hace el viaje por dentro.
 */
export function LlevarEntrada({
  id,
  titulo,
  alVolver,
  alHecho,
}: {
  id: string;
  titulo: string;
  alVolver: () => void;
  /** Se llama cuando la entrada ya no está aquí, para rehacer la lista. */
  alHecho: (dicho: string) => void;
}) {
  const [lista, setLista] = useState<Proyecto[] | null>(null);
  const [activa, setActiva] = useState("");
  const [error, setError] = useState("");
  const [trabajando, setTrabajando] = useState(false);

  useEffect(() => {
    void (async () => {
      try {
        const [e, l] = await Promise.all([esfinge.estadoBoveda(), esfinge.proyectos()]);
        setActiva(e.proyecto);
        setLista(l);
      } catch (e) {
        setError(mensaje(e));
      }
    })();
  }, []);

  // Los destinos posibles: las bóvedas que están en este equipo y no son ésta.
  // La personal entra en la lista **solo si no es la activa**.
  const destinos: { ref: string; nombre: string }[] = [
    ...(activa !== "" ? [{ ref: "", nombre: "Tu bóveda" }] : []),
    ...(lista ?? [])
      .filter((p) => p.ref !== activa && p.enEsteEquipo && !p.archivado)
      .map((p) => ({ ref: p.ref, nombre: p.nombre })),
  ];

  async function llevar(ref: string, nombre: string, copiar: boolean) {
    setTrabajando(true);
    setError("");
    try {
      await esfinge.llevarAOtraBoveda(id, ref, copiar);
      alHecho(
        copiar
          ? `«${titulo}» también está ahora en ${nombre}.`
          : `«${titulo}» se ha ido a ${nombre}. Aquí queda en la papelera 30 días.`,
      );
    } catch (e) {
      setError(mensaje(e));
      setTrabajando(false);
    }
  }

  return (
    <div className="panel">
      <div className="boveda-barra">
        <button onClick={alVolver}>← Volver</button>
      </div>

      <h2>Llevar «{titulo}» a otra bóveda</h2>
      <p className="nota">
        Al moverla, aquí queda en la papelera durante 30 días. La contraseña no sale de Esfinge en ningún
        momento.
      </p>

      {error && <p className="error">{error}</p>}

      {lista != null && destinos.length === 0 && (
        <p className="nota">
          No hay ninguna otra bóveda en este ordenador. Crea una en «Proyectos», o baja una que esté dormida.
        </p>
      )}

      {destinos.length > 0 && (
        <ul className="proyectos">
          {destinos.map((d) => (
            <li key={d.ref || "personal"}>
              <span className="abrir-proyecto">
                <Icono nombre={d.ref === "" ? "boveda" : "proyectos"} />
                <span className="nombre">{d.nombre}</span>
              </span>
              <button className="discreto" onClick={() => llevar(d.ref, d.nombre, true)} disabled={trabajando}>
                Copiar
              </button>
              <button className="principal" onClick={() => llevar(d.ref, d.nombre, false)} disabled={trabajando}>
                Mover
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

/**
 * Lo que se hace con un proyecto **cuando se acaba** (ADR 0051): entregárselo al
 * cliente, archivarlo o borrarlo.
 *
 * Va detrás de un botón y no en la fila porque esto se usa una vez en la vida de
 * un proyecto, y porque las tres son decisiones: una entrega una copia de todas
 * sus contraseñas, otra se lleva el fichero de este equipo y la tercera no tiene
 * vuelta atrás.
 */
function AlAcabarElProyecto({
  proyecto,
  alEntregar,
  alArchivar,
  alBorrar,
  alFallar,
}: {
  proyecto: Proyecto;
  /** Recibe la clave de recuperación de lo entregado, para la ceremonia. */
  alEntregar: (clave: string) => void;
  alArchivar: () => void;
  alBorrar: () => void;
  alFallar: (mensaje: string) => void;
}) {
  const [que, setQue] = useState<"" | "entregar" | "borrar">("");
  const [clave, setClave] = useState("");
  const [trabajando, setTrabajando] = useState(false);

  async function entregar() {
    setTrabajando(true);
    try {
      alEntregar(await esfinge.entregarProyecto(proyecto.ref, clave));
      setClave("");
    } catch (e) {
      alFallar(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  async function borrar() {
    setTrabajando(true);
    try {
      await esfinge.borrarProyecto(proyecto.ref, clave);
      setClave("");
      alBorrar();
    } catch (e) {
      alFallar(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  if (que === "entregar") {
    return (
      <div className="al-acabar">
        <h3>Entregar «{proyecto.nombre}»</h3>
        <p className="nota">
          Se guarda una copia con la contraseña que le pongas aquí. A partir de ahí es suya: no se
          sincroniza con la tuya ni la puedes abrir.
        </p>
        <label htmlFor={`entregar-${proyecto.ref}`}>Contraseña para quien la reciba</label>
        <input
          id={`entregar-${proyecto.ref}`}
          type="password"
          autoComplete="off"
          autoFocus
          value={clave}
          onChange={(e) => setClave(e.target.value)}
        />
        {clave !== "" && clave.length < MINIMO && <p className="nota">Al menos {MINIMO} caracteres</p>}
        <p className="aviso">
          <strong>Dísela por otro camino</strong>, no en el mismo correo que el fichero. Y verás una vez su
          clave de recuperación: va para quien la reciba.
        </p>
        <div className="botones">
          <button className="principal" onClick={entregar} disabled={clave.length < MINIMO || trabajando}>
            {trabajando ? "Entregando…" : "Elegir dónde guardarla"}
          </button>
          <button className="discreto" onClick={() => setQue("")}>
            Cancelar
          </button>
        </div>
      </div>
    );
  }

  if (que === "borrar") {
    return (
      <div className="al-acabar">
        <h3>Borrar «{proyecto.nombre}»</h3>
        {/* **Lo único irreversible que hay aquí**, y se dice con todas las letras
            antes de pedir nada: no va a la papelera, no está en otro equipo y no se
            puede recuperar con la clave de recuperación. */}
        <p className="aviso">
          <strong>Esto no se puede deshacer.</strong> Se borra de este ordenador, de tus otros equipos y del
          servidor, con todo lo que tenga dentro. Si quieres conservarla, entrégala antes.
        </p>
        <label htmlFor={`borrar-${proyecto.ref}`}>Tu contraseña maestra</label>
        <input
          id={`borrar-${proyecto.ref}`}
          type="password"
          autoComplete="off"
          autoFocus
          value={clave}
          onChange={(e) => setClave(e.target.value)}
        />
        <div className="botones">
          <button className="principal" onClick={borrar} disabled={clave === "" || trabajando}>
            {trabajando ? "Borrando…" : "Borrarla para siempre"}
          </button>
          <button className="discreto" onClick={() => setQue("")}>
            Cancelar
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="al-acabar">
      <div className="botones">
        <button onClick={() => setQue("entregar")}>Entregársela al cliente</button>
        <button onClick={alArchivar}>Archivarla</button>
        <button className="discreto" onClick={() => setQue("borrar")}>
          Borrarla
        </button>
      </div>
      <p className="nota">
        Archivarla la saca de la lista y <strong>borra su copia de este ordenador</strong>; sigue en tu
        cuenta y se puede traer cuando haga falta.
      </p>
    </div>
  );
}

/** El mismo mínimo que al crear una bóveda. */
const MINIMO = 10;
