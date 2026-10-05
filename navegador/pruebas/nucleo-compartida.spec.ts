import { expect, test } from "@playwright/test";
import { Boveda } from "../src/nucleo/boveda";

/**
 * Abrir una bóveda a la que me han dado acceso, en el espejo (ADR 0052).
 *
 * **Casi todo lo de aquí vive en las cruzadas de Go**, y no por pereza: la ranura va
 * sellada, y sellar no está en el espejo a propósito (ver la cabecera de
 * `nucleo/compartida.ts`). Un sellador escrito aquí para las pruebas sería **una
 * segunda implementación**: el día que se desviara de Go, estas pruebas seguirían
 * verdes y las bóvedas compartidas no se abrirían en el navegador. Así que quien sella
 * es Go, en `TestCruzadaUnSelladoDeGoLoAbreLaExtension` y
 * `TestCruzadaUnaCompartidaSeAbreEnLaExtension`.
 *
 * Y queda lo que allí no se ve: **lo que abrir una bóveda ajena no tiene que hacerle a
 * la mía**.
 */

const MAESTRA = "una maestra larga para las pruebas del núcleo";
const TITULAR = "1111222233334444";

/**
 * Una bóveda ajena con una ranura de acceso para `TITULAR`. El contenedor no vale para
 * nada, y da igual: lo que se comprueba no es que abra.
 *
 * La ranura se inyecta en el documento a mano porque **ponerla es cosa de quien da el
 * acceso**, y eso es la ventana.
 */
async function ajenaConRanuraInservible(): Promise<string> {
  const { boveda } = await Boveda.crearProyecto("la-clave-de-otra-persona");
  const doc = JSON.parse(await boveda.guardar());
  doc.sobres.push({
    tipo: `acceso:${TITULAR}`,
    creado: "2026-10-05T12:00:00Z",
    contenedor: "ACC1.AAAA.BBBB",
    codificacion: "sobre-x25519-v1",
  });
  return JSON.stringify(doc);
}

/**
 * **Abrir una bóveda ajena no le crea identidad a la mía**, y eso no es una
 * delicadeza.
 *
 * `semillaDeIdentidad()` crea la identidad si no la hay **y guarda**, así que usarla
 * aquí tendría dos costes: la bóveda personal sube de serie sin que nadie haya
 * cambiado nada —o sea una subida al servidor por haber mirado un proyecto de otro— y,
 * peor, haría creer que el fallo es otro: con identidad recién creada la ranura sigue
 * sin abrirse, porque está sellada hacia la que había cuando dieron el acceso.
 *
 * Una personal sin identidad es una que nunca ha compartido ni recibido nada, así que
 * tampoco puede tener acceso a nada. Rendirse es la respuesta correcta.
 */
test("compartida: abrir una ajena no escribe en la bóveda personal", async () => {
  const { boveda: personal } = await Boveda.crear(MAESTRA);
  const antes = await personal.guardar();
  const serieAntes = personal.serie;

  await expect(Boveda.abrirCompartida(await ajenaConRanuraInservible(), TITULAR, personal)).rejects.toThrow(
    /No tienes acceso/,
  );

  expect(personal.serie, "la serie de la personal ha cambiado al abrir una bóveda ajena").toBe(serieAntes);
  expect(personal.documento(), "la bóveda personal se ha escrito al abrir una bóveda ajena").toBe(antes);
});

/**
 * Y con identidad, que es el caso que de verdad corre: la ranura no es para ésta, así
 * que tampoco abre — **y la personal sigue intacta**, que es la mitad que se olvida.
 */
test("compartida: con identidad y una ranura que no es suya, tampoco abre ni escribe", async () => {
  const { boveda: personal } = await Boveda.crear(MAESTRA);
  await personal.semillaDeIdentidad();
  const antes = await personal.guardar();
  const serieAntes = personal.serie;

  await expect(Boveda.abrirCompartida(await ajenaConRanuraInservible(), TITULAR, personal)).rejects.toThrow(
    /No tienes acceso/,
  );
  expect(personal.serie).toBe(serieAntes);
  expect(personal.documento()).toBe(antes);
});

/**
 * Sin ranura para mí no se prueba ninguna otra.
 *
 * Importa porque una bóveda compartida **tiene más ranuras**: la del dueño y una por
 * cada persona con acceso. Probarlas todas no abriría ninguna —están selladas hacia
 * otras identidades— pero sí gastaría tiempo para acabar en el mismo error, y es la
 * clase de «por si acaso» que convierte un error claro en uno lento.
 */
test("compartida: sin ranura para ese titular no abre", async () => {
  const { boveda: personal } = await Boveda.crear(MAESTRA);
  await personal.semillaDeIdentidad();
  const ajena = await ajenaConRanuraInservible();

  for (const titular of ["", "9999888877776666", "acceso:" + TITULAR]) {
    await expect(Boveda.abrirCompartida(ajena, titular, personal), `titular ${titular || "vacío"}`).rejects.toThrow(
      /No tienes acceso/,
    );
  }
});
