package red

import "testing"

func TestSinRedSoloCuandoSePide(t *testing.T) {
	olvidar()
	if SinRed() {
		t.Error("dice que no hay red sin que nadie lo haya pedido")
	}

	olvidar()
	t.Setenv(laVariable, "1")
	if !SinRed() {
		t.Errorf("%s=1 y sigue creyendo que puede salir", laVariable)
	}

	// **Se lee una sola vez.** Un programa que cambia de opinión sobre si puede
	// usar la red según cuándo se le pregunte es peor que cualquiera de las dos
	// respuestas: lo que decide es el entorno con el que arrancó.
	t.Setenv(laVariable, "")
	if !SinRed() {
		t.Error("ha cambiado de opinión a mitad de ejecución")
	}
	olvidar()
}

// Cualquier valor vale: quien pone la variable quiere apagarla, no negociar.
func TestCualquierValorLaApaga(t *testing.T) {
	for _, v := range []string{"1", "sí", "0", "false", "x"} {
		olvidar()
		t.Setenv(laVariable, v)
		if !SinRed() {
			t.Errorf("con %q puesto sigue saliendo a la red", v)
		}
	}
	olvidar()
}
