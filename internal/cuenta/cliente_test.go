package cuenta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// **Lo que autoriza a borrar una bóveda del disco** (ADR 0053), que es la decisión más
// cara de equivocar de todo el cliente: un `Revocado` de más borra la bóveda de un
// cliente en todos los equipos a la vez, y no tiene vuelta.
//
// Por eso hacen falta las tres cosas —el 403, un cuerpo que se entienda y que ese cuerpo
// lo diga— y por eso la tabla lleva los dos casos que **parecen** una revocación y no lo
// son: el otro 403 del Worker y el 403 de un portero que nadie puso ahí para esto.
func TestSoloBorraLoQueElServidorDiceQueEsUnaRevocacion(t *testing.T) {
	casos := []struct {
		nombre   string
		estado   int
		tipo     string
		cuerpo   string
		revocado bool
		sinAcc   bool
	}{
		{"el dueño te quitó el acceso", 403, "application/json",
			`{"error":"Ya no tienes acceso a esa bóveda.","codigo":"revocado"}`, true, true},
		{"solo puedes ver y has intentado subir", 403, "application/json",
			`{"error":"En esta bóveda solo puedes ver.","codigo":"solo-ver"}`, false, true},
		{"un 403 nuestro sin código", 403, "application/json",
			`{"error":"Esa clave no abre la bóveda de esta cuenta."}`, false, true},
		{"un portero delante, que contesta HTML", 403, "text/html",
			`<html><body>Access denied</body></html>`, false, true},
		{"y el mismo código con otro estado no vale", 401, "application/json",
			`{"error":"Vuelve a entrar.","codigo":"revocado"}`, false, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", c.tipo)
				w.WriteHeader(c.estado)
				_, _ = w.Write([]byte(c.cuerpo))
			}))
			defer s.Close()

			// Se baja **una compartida**, que es el camino donde el servidor manda el
			// código: el 403 de bajar es el único que no tiene dos lecturas posibles.
			_, _, _, err := Nuevo(s.URL).BajarCompartida(context.Background(), "s1.x.y", "0123456789abcdef", "aaaabbbbccccdddd", 0)
			if err == nil {
				t.Fatal("se esperaba un error")
			}
			if got := Revocado(err); got != c.revocado {
				t.Errorf("Revocado = %v, se esperaba %v (error: %v)", got, c.revocado, err)
			}
			if got := SinAcceso(err); got != c.sinAcc {
				t.Errorf("SinAcceso = %v, se esperaba %v", got, c.sinAcc)
			}
		})
	}
}
