// Package cruzada deja a las pruebas de Go ejecutar el núcleo de la extensión
// (ADR 0040): la bóveda, la fusión, los códigos y las claves de la cuenta, que
// desde la fase E existen dos veces, en Go y en TypeScript.
//
// **Dos implementaciones probadas por separado pueden no estar de acuerdo**, y en
// la fusión eso es el riesgo número uno de las cuentas: si Go y la extensión
// funden distinto, cada una sube lo suyo y los equipos se pasan la bóveda sin fin.
// Así que las pruebas cruzadas mandan lo mismo a los dos lados y exigen los mismos
// bytes.
//
// Como las de la cuenta con el servidor, **un `go test` suelto se las salta**:
// hacen falta Node y las dependencias de `navegador/` instaladas. Las activa
// `ESFINGE_CRUZADA=1`, que ponen `make comprobar` y la puerta de publicación
// después de instalar la extensión.
package cruzada

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Activa dice si hay que correr las pruebas cruzadas, y si no, las salta.
func Activa(t testing.TB) {
	t.Helper()
	if os.Getenv("ESFINGE_CRUZADA") == "" {
		t.Skip("Prueba cruzada con la extensión: se corre con ESFINGE_CRUZADA=1 (make comprobar)")
	}
}

// Pedir manda una petición al núcleo de la extensión y lee su respuesta.
func Pedir(t testing.TB, peticion any, respuesta any) {
	t.Helper()
	Activa(t)
	guion := filepath.Join(raiz(t), "navegador", "herramientas", "cruzada.mjs")
	entrada, err := json.Marshal(peticion)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("node", guion)
	cmd.Stdin = bytes.NewReader(entrada)
	var salida, errores bytes.Buffer
	cmd.Stdout, cmd.Stderr = &salida, &errores
	if err := cmd.Run(); err != nil {
		t.Fatalf("la extensión no ha podido contestar: %v\n%s", err, errores.String())
	}
	if err := json.Unmarshal(salida.Bytes(), respuesta); err != nil {
		t.Fatalf("la extensión contesta algo que no se entiende: %v\n%s", err, salida.String())
	}
}

// raiz es la carpeta del repositorio: la que tiene go.mod.
func raiz(t testing.TB) string {
	t.Helper()
	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		arriba := filepath.Dir(d)
		if arriba == d {
			t.Fatal("no se encuentra la raíz del repositorio")
		}
		d = arriba
	}
}
