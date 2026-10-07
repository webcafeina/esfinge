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
//   - ~~**Volver a la bóveda personal pide la contraseña maestra.**~~ **Ya no, desde el
//     2026-10-05**: vuelve abierta. Lo de antes se justificaba diciendo que la personal
//     no puede quedarse abierta por detrás —dos bóvedas a la vez es lo que se descartó—
//     y eso valía para tenerlas abiertas *a la vez*, no para abrir una **después** de
//     cerrar la otra. Lo preguntó el cliente viendo que «Cerrar la bóveda» y «Salir del
//     proyecto» hacían exactamente lo mismo. Ahora **salir** devuelve la tuya abierta y
//     lleva a esta lista; **cerrar** cierra todo y olvida la clave.
//
// Y una cosa que esta pantalla **no** hace: enseñar nombres con la bóveda cerrada.
// No es una decisión de aquí — es que no los tiene, porque viven dentro del cuerpo
// cifrado de la bóveda personal.

import { useCallback, useEffect, useState } from "react";
import { esfinge, type CompartidaEnLaLista, type Proyecto, type QuienTieneAcceso } from "./puente";
import { Icono } from "./componentes";
import { Ceremonia } from "./boveda";

export function Proyectos({
  activo,
  alEntrar,
  verAcceso = 0,
}: {
  /** Si la sección está a la vista: se recarga al entrar, como el historial. */
  activo: boolean;
  /** Se llama al conmutar, para llevar a la bóveda. */
  alEntrar: () => void;
  /**
   * Sube cada vez que alguien pide ver quién tiene acceso **desde dentro del
   * proyecto**, y entonces esta pantalla despliega ese bloque en la fila del proyecto
   * abierto.
   *
   * Es un contador y no un booleano porque lo que llega es **un gesto**, no un estado:
   * con un booleano habría que apagarlo después y el segundo clic no haría nada.
   */
  verAcceso?: number;
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
  // Y cuál tiene desplegado «Quién tiene acceso…».
  const [elAcceso, setElAcceso] = useState("");
  // Qué proyecto tiene desplegado «Al acabar…». Lo de terminar un proyecto se usa
  // una vez en su vida, así que no ocupa sitio en la fila del día a día.
  const [alAcabar, setAlAcabar] = useState("");
  // La clave de recuperación de una bóveda recién entregada: se enseña **una vez**
  // con la ceremonia de siempre y no se puede volver a pedir.
  const [entregada, setEntregada] = useState("");
  // Lo que me han compartido (ADR 0052). Lo trae la misma recarga que los proyectos.
  const [compartidas, setCompartidas] = useState<CompartidaEnLaLista[]>([]);

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
      // **Y lo compartido, en la misma pasada.** Pedirlo aparte desde su componente
      // significaba pedirlo también con la bóveda cerrada, y eso es un 400 seguro en
      // la consola: el mismo fallo que ya costó una vuelta con `POST /api/Proyectos`
      // el 2026-10-02. Lo que la pantalla necesita para dibujar va en lo que ya pide.
      setCompartidas(await esfinge.compartidas());
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

  // **Y si se ha pedido desde dentro del proyecto, se despliega al llegar aquí.**
  //
  // Se espera a tener la lista, no se hace en el clic: la lista **se vuelve a pedir al
  // entrar en la sección**, así que apuntar la referencia antes de que llegue es la
  // misma carrera que ya costó un ayudante de pruebas. Con la lista en la mano, el
  // proyecto abierto es el que viene marcado como activo.
  useEffect(() => {
    if (verAcceso === 0 || lista === null) return;
    const abierto = lista.find((p) => p.activo);
    if (abierto) setElAcceso(abierto.ref);
  }, [verAcceso, lista]);

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
                    /* **Los botones van juntos en un grupo, y no sueltos en la fila.**
                       Sueltos, cada uno es una columna más del `flex` del `li` y la fila
                       se parte por donde toque: con el proyecto abierto son tres, y «Al
                       acabar…» se caía solo a una segunda línea pegado a la izquierda,
                       debajo del nombre, como si fuera de otra cosa. Agrupados, o caben
                       los tres o bajan los tres, y bajan alineados con los de arriba.
                       Lo vio el cliente; aquí se vio en la captura (2026-10-06). */
                    <div className="acciones-proyecto">
                      <button
                        className="discreto"
                        onClick={() => {
                          setRenombrando(p.ref);
                          setOtroNombre(p.nombre);
                        }}
                      >
                        Cambiar el nombre
                      </button>
                      {/* **Quién tiene acceso solo sale en el proyecto abierto**, y
                          no es una limitación de la pantalla: dar acceso escribe una
                          ranura en el fichero de ese proyecto, y lo que sube ese
                          fichero es la sincronización de la bóveda abierta. Desde
                          aquí, la ranura se quedaría en este equipo y quien recibiera
                          el acceso se bajaría una bóveda que no puede abrir. */}
                      {p.activo && (
                        <button
                          className="discreto"
                          onClick={() => setElAcceso(elAcceso === p.ref ? "" : p.ref)}
                          aria-expanded={elAcceso === p.ref}
                        >
                          Quién tiene acceso…
                        </button>
                      )}
                      <button
                        className="discreto"
                        onClick={() => setAlAcabar(alAcabar === p.ref ? "" : p.ref)}
                        aria-expanded={alAcabar === p.ref}
                      >
                        Al acabar…
                      </button>
                    </div>
                  )}
                </>
              )}
              {/* **Y se cierra solo al salir del proyecto.** `elAcceso` guarda una
                  referencia, no «lo que está abierto», así que al salir el bloque se
                  quedaba desplegado con su campo de correo **y sin forma de cerrarlo**:
                  el botón que lo cierra es «Quién tiene acceso…», y ése solo sale con el
                  proyecto abierto. Lo vio el cliente al salir (2026-10-06). Se mira
                  `p.activo` aquí y no se limpia al salir porque esto no se entera de que
                  alguien salió: lo que cambia es la lista, y la lista llega con `activo`
                  ya puesto. */}
              {p.activo && elAcceso === p.ref && <ElAcceso alFallar={setError} />}
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

      {/* **Lo que me han compartido va en su propio grupo**, no mezclado con lo mío
          (ADR 0052): no son proyectos míos, no se entregan, no se borran del
          servidor y a algunos solo puedo mirarlos. Mezclarlos obligaría a explicar
          en cada fila de qué clase es. */}
      <CompartidasConmigo lista={compartidas} alEntrar={alEntrar} alFallar={setError} alCambiar={recargar} />

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


/**
 * Las bóvedas de otras personas a las que tengo acceso (ADR 0052).
 *
 * Tres cosas que esta lista tiene que decir y que no se ven mirándola:
 *
 *   - **De quién es cada una.** Una bóveda compartida se parece a un proyecto propio
 *     y no lo es: no se entrega, no se borra del servidor y a veces solo se mira.
 *   - **Si solo puedo verla**, antes de entrar. Enterarse al ir a guardar es la regla
 *     que costó `ExportarLlaves`: lo que hace falta para decidir se mira antes.
 *   - **Que dejar de verla no se la quita a nadie más.** Es mi lista, no la suya.
 */
function CompartidasConmigo({
  lista,
  alEntrar,
  alFallar,
  alCambiar,
}: {
  /** **La trae el padre**, en la misma pasada que los proyectos. Pidiéndola aquí se
   * pedía también con la bóveda cerrada, y eso es un 400 en la consola: el fallo que
   * ya costó una vuelta con `POST /api/Proyectos`. */
  lista: CompartidaEnLaLista[];
  alEntrar: () => void;
  alFallar: (m: string) => void;
  alCambiar: () => void | Promise<void>;
}) {
  const [trabajando, setTrabajando] = useState("");

  if (lista.length === 0) return null;

  async function entrar(c: CompartidaEnLaLista) {
    setTrabajando(c.dueno + c.ref);
    try {
      if (!c.enEsteEquipo) await esfinge.bajarCompartida(c.dueno, c.ref);
      await esfinge.abrirCompartida(c.dueno, c.ref);
      alEntrar();
    } catch (e) {
      alFallar(mensaje(e));
    } finally {
      setTrabajando("");
    }
  }

  return (
    <div className="compartidas">
      <h3>Compartido conmigo</h3>
      <ul className="proyectos">
        {lista.map((c) => (
          <li key={c.dueno + c.ref} className={c.retirada ? "retirada" : undefined}>
            {/* **Una retirada ya no es una bóveda: es el aviso de una que se fue**
                (ADR 0053). Así que no se puede pulsar — el fichero no está—, y lo que
                dice es qué pasó y cuándo. La fila se queda, tachada, porque sin ella
                no hay forma de contar cuál era: es lo que hace el resto del proyecto
                al cerrar algo. */}
            {c.retirada ? (
              <p className="aviso">
                <strong>{c.nombre}</strong> ya no está: quien te la compartió te quitó el acceso{" "}
                {cuando(c.retirada)}. Se ha borrado de este equipo.
              </p>
            ) : (
              <button className="abrir-proyecto" onClick={() => entrar(c)} disabled={trabajando !== ""}>
                {/* El mismo glifo que un proyecto propio, **a propósito**: es un
                    proyecto, de otra persona. Lo que lo distingue va en el texto —de
                    quién es y si solo se puede mirar—, que es lo que de verdad hay que
                    leer antes de entrar. Un glifo nuevo aquí sería una cosa más que
                    aprender para decir lo mismo peor. */}
                <Icono nombre="proyectos" />
                <span className="nombre">{c.nombre}</span>
                <span className="aparte">
                  {trabajando === c.dueno + c.ref
                    ? "Abriendo…"
                    : !c.enEsteEquipo
                      ? "Se traerá a este equipo"
                      : c.permiso === "ver"
                        ? "Solo puedes ver"
                        : "Puedes editar"}
                </span>
              </button>
            )}
            <button
              className="discreto"
              onClick={async () => {
                try {
                  await esfinge.dejarDeVerCompartida(c.dueno, c.ref);
                  await alCambiar();
                } catch (e) {
                  alFallar(mensaje(e));
                }
              }}
            >
              {c.retirada ? "Quitarla de la lista" : "Dejar de verla"}
            </button>
          </li>
        ))}
      </ul>
      <p className="nota">
        Estas bóvedas son de otras personas. <strong>Dejar de verlas no se las quita a nadie</strong>: solo
        las saca de tu lista y borra la copia de este equipo.
      </p>
    </div>
  );
}


/**
 * Quién tiene acceso a la bóveda abierta, y cómo se da o se quita (ADR 0052).
 *
 * Lo que esta pantalla tiene que decir y no se ve mirándola:
 *
 *   - **La huella se enseña antes de dar el acceso**, y se explica para qué sirve.
 *     Es lo único que protege del servidor en el primer envío, y una huella que nadie
 *     mira no protege nada (ADR 0043).
 *   - **«Ver» lo impone el servidor, no el cifrado.** Quien puede ver tiene con qué
 *     descifrar, así que puede escribir en su copia; lo que no puede es subirla. Se
 *     dice con las mismas palabras con que Esfinge dice que Touch ID es un cerrojo.
 *   - **Quitar el acceso borra la bóveda en el equipo del otro** (ADR 0053), y la
 *     pantalla dice **lo que eso no es**: pasa cuando esa persona abre Esfinge, así que
 *     si no lo abre no pasa, y no protege de quien haya querido guardarse una copia.
 *     Hasta la 2.41.1 no se borraba nada y aquí ponía que no había forma; sí la había,
 *     la misma que usan Dashlane y Bitwarden, y lo que no se puede es **prometerlo**.
 *   - **Quien recibe acceso ve el correo de los demás que lo tienen.** No hay forma
 *     de que no lo vea si la lista se puede leer, así que se dice.
 */
function ElAcceso({ alFallar }: { alFallar: (m: string) => void }) {
  const [gente, setGente] = useState<QuienTieneAcceso[] | null>(null);
  const [correo, setCorreo] = useState("");
  const [permiso, setPermiso] = useState<"ver" | "editar">("editar");
  const [huella, setHuella] = useState("");
  const [trabajando, setTrabajando] = useState(false);
  const [quitando, setQuitando] = useState("");

  const recargar = useCallback(async () => {
    try {
      setGente(await esfinge.quienTiene());
    } catch (e) {
      alFallar(mensaje(e));
      setGente([]);
    }
  }, [alFallar]);
  useEffect(() => {
    void recargar();
  }, [recargar]);

  async function mirar() {
    setTrabajando(true);
    try {
      setHuella((await esfinge.huellaDe(correo.trim())).huella);
    } catch (e) {
      alFallar(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  async function dar() {
    setTrabajando(true);
    try {
      await esfinge.darAcceso(correo.trim(), permiso);
      setCorreo("");
      setHuella("");
      await recargar();
    } catch (e) {
      alFallar(mensaje(e));
    } finally {
      setTrabajando(false);
    }
  }

  return (
    <div className="el-acceso">
      {gente !== null && gente.length > 0 && (
        <ul className="titulares">
          {gente.map((t) => (
            <li key={t.titular}>
              <span className="nombre">{t.correo}</span>
              <span className="aparte">{t.permiso === "ver" ? "Solo puede ver" : "Puede editar"}</span>
              {!t.enElServidor && (
                /* Las dos listas no cuadran, y eso se enseña en vez de disimularlo:
                   es lo que permite arreglarlo en vez de descubrirlo el día que esa
                   persona no entra. */
                <span className="aparte">
                  <strong>El servidor no le deja entrar</strong>
                </span>
              )}
              <button
                className={quitando === t.titular ? "principal" : "discreto"}
                disabled={trabajando}
                onClick={async () => {
                  if (quitando !== t.titular) {
                    setQuitando(t.titular);
                    return;
                  }
                  setTrabajando(true);
                  try {
                    await esfinge.quitarAcceso(t.titular);
                    setQuitando("");
                    await recargar();
                  } catch (e) {
                    alFallar(mensaje(e));
                  } finally {
                    setTrabajando(false);
                  }
                }}
              >
                {quitando === t.titular ? "Sí, quitarle el acceso" : "Quitar"}
              </button>
            </li>
          ))}
        </ul>
      )}

      {quitando !== "" && (
        <p className="aviso">
          Dejará de recibir cambios, y la bóveda <strong>se borrará de su ordenador</strong> la próxima vez
          que abra Esfinge. <strong>Si no lo abre, no se borra</strong>, y esto tampoco protege de quien
          haya querido guardarse una copia a mano: si lo de dentro no puede estar en sus manos, lo que hay
          que cambiar son las contraseñas, no esta bóveda.
        </p>
      )}

      <div>
        <label htmlFor="acceso-correo">Dar acceso a</label>
        <input
          id="acceso-correo"
          type="email"
          autoComplete="off"
          placeholder="correo@ejemplo.com"
          value={correo}
          onChange={(e) => {
            setCorreo(e.target.value);
            setHuella("");
          }}
        />
      </div>

      <div className="botones">
        <button className="discreto" onClick={mirar} disabled={correo.trim() === "" || trabajando}>
          {trabajando ? "Mirando…" : "Ver su huella"}
        </button>
      </div>

      {huella !== "" && (
        <>
          <p className="nota">
            Su huella es <code className="huella">{huella}</code>. Compárala con esa persona por otro
            camino —una llamada— antes de darle acceso: es lo único que prueba que las contraseñas van a
            quien crees.
          </p>
          <div>
            <label htmlFor="acceso-permiso">Qué puede hacer</label>
            <select
              id="acceso-permiso"
              value={permiso}
              onChange={(e) => setPermiso(e.target.value === "ver" ? "ver" : "editar")}
            >
              <option value="editar">Ver y editar</option>
              <option value="ver">Solo ver</option>
            </select>
          </div>
          <p className="nota">
            {permiso === "ver" ? (
              <>
                <strong>«Solo ver» lo decide el servidor, no el cifrado.</strong> Quien pueda ver esta
                bóveda tiene con qué descifrarla, así que puede escribir en la copia de su equipo. Lo que
                no puede es subirla: el servidor la rechaza, y por eso nadie más verá lo que escriba.
              </>
            ) : (
              <>
                Quien pueda editar puede <strong>cambiar y borrar</strong> lo de aquí, y{" "}
                <strong>dar acceso a más gente</strong>.
              </>
            )}{" "}
            Verá también el correo de las demás personas con acceso.
          </p>
          <div className="botones">
            <button className="principal" onClick={dar} disabled={trabajando}>
              {trabajando ? "Dando acceso…" : "Dar acceso"}
            </button>
          </div>
        </>
      )}
    </div>
  );
}
