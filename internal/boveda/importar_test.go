package boveda

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// Los formatos que de verdad va a traer alguien. Se mapea por nombre de columna
// y no por posición, así que el mismo código los lee todos.
func TestLeeLosCSVDeLosGestoresDeVerdad(t *testing.T) {
	casos := map[string]struct {
		csv     string
		titulo  string
		usuario string
		secreto string
		sitio   string
	}{
		"Dashlane": {
			"username,username2,username3,title,password,note,url,category,otpSecret\n" +
				"yo@ejemplo.com,,,Banco,s3cr3t0,mi nota,https://banco.es,Finanzas,JBSWY3DP\n",
			"Banco", "yo@ejemplo.com", "s3cr3t0", "https://banco.es",
		},
		"Chrome": {
			"name,url,username,password\n" +
				"Banco,https://banco.es,yo@ejemplo.com,s3cr3t0\n",
			"Banco", "yo@ejemplo.com", "s3cr3t0", "https://banco.es",
		},
		"Bitwarden": {
			"folder,favorite,type,name,notes,fields,login_uri,login_username,login_password,login_totp\n" +
				"Finanzas,,login,Banco,mi nota,,https://banco.es,yo@ejemplo.com,s3cr3t0,\n",
			"Banco", "yo@ejemplo.com", "s3cr3t0", "https://banco.es",
		},
		"LastPass": {
			"url,username,password,totp,extra,name,grouping,fav\n" +
				"https://banco.es,yo@ejemplo.com,s3cr3t0,,mi nota,Banco,Finanzas,0\n",
			"Banco", "yo@ejemplo.com", "s3cr3t0", "https://banco.es",
		},
		"1Password": {
			"Title,Url,Username,Password,OTPAuth,Notes\n" +
				"Banco,https://banco.es,yo@ejemplo.com,s3cr3t0,,mi nota\n",
			"Banco", "yo@ejemplo.com", "s3cr3t0", "https://banco.es",
		},
	}

	for nombre, c := range casos {
		t.Run(nombre, func(t *testing.T) {
			es, _, err := Leer([]byte(c.csv), nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(es) != 1 {
				t.Fatalf("entradas: %d", len(es))
			}
			e := es[0]
			if e.Titulo != c.titulo || e.Usuario != c.usuario || e.Secreto != c.secreto {
				t.Errorf("mal leído: %+v", e)
			}
			if len(e.Sitios) == 0 || e.Sitios[0] != c.sitio {
				t.Errorf("sitios: %v", e.Sitios)
			}
		})
	}
}

// Las trampas de los CSV del mundo real, todas encontradas a base de que fallen.
func TestLasTrampasDeLosCSVReales(t *testing.T) {
	t.Run("con marca de orden de bytes", func(t *testing.T) {
		// El fallo clásico: la BOM se pega al nombre de la primera columna y esa
		// columna deja de reconocerse. Si no se quita, «name» pasa a ser la misma palabra con tres bytes
		// invisibles delante, y deja de encontrarse en la tabla de alias.
		conBOM := append([]byte{0xEF, 0xBB, 0xBF}, []byte("name,url,username,password\nBanco,https://b.es,yo,s3cr3t0\n")...)
		es, mapa, err := Leer(conBOM, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !mapa.tiene(CampoTitulo) {
			t.Error("la BOM se ha comido la primera columna")
		}
		if es[0].Titulo != "Banco" {
			t.Errorf("título: %q", es[0].Titulo)
		}
	})

	t.Run("con punto y coma, del Excel europeo", func(t *testing.T) {
		es, _, err := Leer([]byte("name;url;username;password\nBanco;https://b.es;yo;s3cr3t0\n"), nil)
		if err != nil {
			t.Fatal(err)
		}
		if es[0].Secreto != "s3cr3t0" {
			t.Errorf("mal separado: %+v", es[0])
		}
	})

	t.Run("con saltos de línea dentro de una nota", func(t *testing.T) {
		csv := "name,password,note\nBanco,s3cr3t0,\"primera línea\nsegunda línea\"\n"
		es, _, err := Leer([]byte(csv), nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(es) != 1 {
			t.Fatalf("ha partido la nota en %d entradas", len(es))
		}
		if !strings.Contains(es[0].Notas, "segunda línea") {
			t.Errorf("notas: %q", es[0].Notas)
		}
	})

	t.Run("con CRLF", func(t *testing.T) {
		es, _, err := Leer([]byte("name,password\r\nBanco,s3cr3t0\r\n"), nil)
		if err != nil {
			t.Fatal(err)
		}
		if es[0].Secreto != "s3cr3t0" {
			t.Errorf("el retorno de carro se ha colado: %q", es[0].Secreto)
		}
	})

	t.Run("con filas de longitud distinta", func(t *testing.T) {
		csv := "name,url,username,password\nBanco,https://b.es,yo,s3cr3t0\nCorto,https://c.es\n"
		es, _, err := Leer([]byte(csv), nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(es) != 2 {
			t.Fatalf("entradas: %d", len(es))
		}
	})

	t.Run("en Latin-1, de un fichero que pasó por Excel", func(t *testing.T) {
		// «contraseña» con la ñ en Windows-1252 (0xF1), que no es UTF-8 válido.
		crudo := append([]byte("name,password\nContrase"), 0xF1, 'a')
		crudo = append(crudo, []byte(",s3cr3t0\n")...)
		es, _, err := Leer(crudo, nil)
		if err != nil {
			t.Fatal(err)
		}
		if es[0].Titulo != "Contraseña" {
			t.Errorf("título: %q; se esperaba que se releyera como Windows-1252", es[0].Titulo)
		}
	})

	t.Run("sin ninguna columna reconocible", func(t *testing.T) {
		_, _, err := Leer([]byte("alfa,beta,gamma\n1,2,3\n"), nil)
		if !errors.Is(err, ErrSinColumnas) {
			t.Errorf("quiero ErrSinColumnas, tengo %v", err)
		}
	})

	t.Run("Dashlane exporta tres usuarios y el bueno es el primero", func(t *testing.T) {
		csv := "username,username2,username3,title,password\nbueno@x.com,otro@x.com,tercero@x.com,Banco,s3cr3t0\n"
		es, _, err := Leer([]byte(csv), nil)
		if err != nil {
			t.Fatal(err)
		}
		if es[0].Usuario != "bueno@x.com" {
			t.Errorf("usuario: %q", es[0].Usuario)
		}
	})
}

// Una bóveda de la que no se puede salir es una trampa.
func TestIdaYVueltaDeLaImportacion(t *testing.T) {
	b, _, _ := nueva(t)

	entrada := "title,url,username,password,note,folder\n" +
		"Banco,https://banco.es,yo@ejemplo.com,s3cr3t0,una nota,Finanzas\n" +
		"Correo,https://correo.es,otro@ejemplo.com,otra clave,,Personal\n"

	es, _, err := Leer([]byte(entrada), nil)
	if err != nil {
		t.Fatal(err)
	}
	metidas, duplicadas, err := b.Importar(es, "Dashlane")
	if err != nil {
		t.Fatal(err)
	}
	if metidas != 2 || duplicadas != 0 {
		t.Fatalf("metidas %d, duplicadas %d", metidas, duplicadas)
	}

	var salida bytes.Buffer
	if err := b.Exportar(&salida); err != nil {
		t.Fatal(err)
	}

	// Y lo exportado se vuelve a leer, que es lo que hace de esto una salida de
	// verdad y no un fichero para mirar.
	otra, _, err := Leer(salida.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(otra) != 2 {
		t.Fatalf("de vuelta: %d entradas", len(otra))
	}
	porTitulo := map[string]Entrada{}
	for _, e := range otra {
		porTitulo[e.Titulo] = e
	}
	if porTitulo["Banco"].Secreto != "s3cr3t0" {
		t.Errorf("la contraseña no ha sobrevivido a la ida y vuelta: %+v", porTitulo["Banco"])
	}
	if porTitulo["Banco"].Notas != "una nota" {
		t.Errorf("las notas no han sobrevivido: %q", porTitulo["Banco"].Notas)
	}
}

// Dos contraseñas distintas para la misma cuenta significan que una está mal, y
// adivinar cuál no es cosa de un importador.
func TestLosDuplicadosSeMarcanPeroNoSeFusionan(t *testing.T) {
	b, _, _ := nueva(t)
	csv := "title,url,username,password\nBanco,https://banco.es,yo@x.com,primera\n"
	es, _, _ := Leer([]byte(csv), nil)
	if _, _, err := b.Importar(es, "Dashlane"); err != nil {
		t.Fatal(err)
	}

	csv2 := "title,url,username,password\nBanco,https://banco.es,yo@x.com,segunda\n"
	es2, _, _ := Leer([]byte(csv2), nil)
	metidas, duplicadas, err := b.Importar(es2, "Chrome")
	if err != nil {
		t.Fatal(err)
	}
	if metidas != 1 || duplicadas != 1 {
		t.Errorf("metidas %d, duplicadas %d", metidas, duplicadas)
	}
	if b.Cuantas() != 2 {
		t.Errorf("ha fusionado: quedan %d entradas", b.Cuantas())
	}
}

// La ADR 0010 decidió que el historial guarda nombres de fichero.
// «credenciales-dashlane.csv» ahí sería una señal de tráfico apuntando a lo que
// alguien acaba de exportar en claro.
func TestImportarNoDejaRastroEnElHistorial(t *testing.T) {
	b, _, ruta := nueva(t)
	es, _, _ := Leer([]byte("title,password\nBanco,s3cr3t0\n"), nil)
	if _, _, err := b.Importar(es, "Dashlane"); err != nil {
		t.Fatal(err)
	}
	// El paquete de la bóveda no conoce siquiera al del historial: la garantía es
	// estructural, no una promesa. Esto lo deja escrito por si alguien lo cambia.
	if strings.Contains(ruta, "historial") {
		t.Fatal("la bóveda no puede vivir en el historial")
	}
}
