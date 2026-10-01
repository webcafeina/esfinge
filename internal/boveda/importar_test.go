package boveda

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/webcafeina/esfinge/internal/cripto"
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
		es, lectura, err := Leer(conBOM, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !lectura.Columnas.tiene(CampoTitulo) {
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
	r, err := b.Importar(es, "Dashlane")
	if err != nil {
		t.Fatal(err)
	}
	if r.Metidas != 2 || r.Repetidas != 0 || r.Conflictos != 0 {
		t.Fatalf("%+v", r)
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
	if _, err := b.Importar(es, "Dashlane"); err != nil {
		t.Fatal(err)
	}

	csv2 := "title,url,username,password\nBanco,https://banco.es,yo@x.com,segunda\n"
	es2, _, _ := Leer([]byte(csv2), nil)
	r, err := b.Importar(es2, "Chrome")
	if err != nil {
		t.Fatal(err)
	}
	if r.Metidas != 1 || r.Conflictos != 1 || r.Repetidas != 0 {
		t.Errorf("%+v", r)
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
	if _, err := b.Importar(es, "Dashlane"); err != nil {
		t.Fatal(err)
	}
	// El paquete de la bóveda no conoce siquiera al del historial: la garantía es
	// estructural, no una promesa. Esto lo deja escrito por si alguien lo cambia.
	if strings.Contains(ruta, "historial") {
		t.Fatal("la bóveda no puede vivir en el historial")
	}
}

// Los cinco ficheros que exporta Dashlane, cada uno con su cabecera.
//
// **Ésta es la prueba que faltaba y por la que el cliente se quedó a medias.**
// Se importaron sus credenciales, salieron 65 entradas, y en Dashlane había más:
// lo que no entraba eran las tarjetas y los documentos, porque el importador solo
// sabía reconocer la forma de un CSV de credenciales y rechazaba los demás
// enteros —«no reconozco ninguna columna»—.
func TestLosCincoFicherosDeDashlane(t *testing.T) {
	casos := []struct {
		nombre  string
		csv     string
		forma   Forma
		tipo    Tipo
		revisar func(*testing.T, Entrada)
	}{{
		nombre: "credentials.csv",
		csv: "username,username2,username3,title,password,note,url,category,otpSecret\n" +
			"yo@ejemplo.com,,,Banco,s3cr3t0,una nota,https://banco.es,Finanzas,JBSWY3DP\n",
		forma: FormaCredencial,
		tipo:  TipoCredencial,
		revisar: func(t *testing.T, e Entrada) {
			if e.Titulo != "Banco" || e.Usuario != "yo@ejemplo.com" || e.Secreto != "s3cr3t0" {
				t.Errorf("credencial mal leída: %+v", e)
			}
			if e.TOTP != "JBSWY3DP" || e.Carpeta != "Finanzas" {
				t.Errorf("se ha perdido el segundo factor o la carpeta: %+v", e)
			}
		},
	}, {
		// **La columna del segundo factor se llama de las dos formas**, y por eso se
		// prueban las dos: el cliente recuerda `otp` a secas en su exportación y aquí
		// estaba escrito `otpSecret`. Las dos están en la tabla de alias, así que la
		// semilla entra igual — lo que no puede quedar es una prueba que diga que
		// comprueba la cabecera de Dashlane comprobando solo una de las dos. **Y la
		// forma no depende de este nombre**: `FormaCredencial` es el caso por defecto,
		// que es lo que hay que saber antes de preocuparse por cómo se llame.
		nombre: "credentials.csv con la columna llamada «otp»",
		csv: "username,username2,username3,title,password,note,url,category,otp\n" +
			"yo@ejemplo.com,,,Banco,s3cr3t0,una nota,https://banco.es,Finanzas,JBSWY3DP\n",
		forma: FormaCredencial,
		tipo:  TipoCredencial,
		revisar: func(t *testing.T, e Entrada) {
			if e.TOTP != "JBSWY3DP" {
				t.Errorf("con la columna «otp» se pierde el segundo factor: %+v", e)
			}
		},
	}, {
		nombre: "securenotes.csv",
		csv:    "title,note\nLa caja fuerte,la combinación es 1234\n",
		forma:  FormaCredencial,
		tipo:   TipoNota,
		revisar: func(t *testing.T, e Entrada) {
			if e.Notas != "la combinación es 1234" {
				t.Errorf("nota mal leída: %+v", e)
			}
		},
	}, {
		nombre: "payments.csv",
		csv: "type,account_name,account_holder,cc_number,code,expiration_month,expiration_year,country,issuing_bank\n" +
			"credit_card,Visa de la empresa,Yo Mismo,4111 1111 1111 1111,737,09,2029,ES,Banco Malo\n",
		forma: FormaTarjeta,
		tipo:  TipoTarjeta,
		revisar: func(t *testing.T, e Entrada) {
			if e.Numero != "4111 1111 1111 1111" || e.Verificacion != "737" {
				t.Errorf("tarjeta mal leída: %+v", e)
			}
			// Dashlane parte la caducidad en dos columnas y aquí es un campo.
			if e.Caduca != "09/2029" {
				t.Errorf("caducidad: %q, y quiero 09/2029", e.Caduca)
			}
			if e.Titular != "Yo Mismo" || e.Titulo != "Visa de la empresa" {
				t.Errorf("titular o título: %+v", e)
			}
			if e.Carpeta != "Banco Malo" {
				t.Errorf("el banco emisor hace de carpeta: %q", e.Carpeta)
			}
		},
	}, {
		nombre: "ids.csv",
		csv: "type,number,name,issue_date,expiration_date,place_of_issue,state\n" +
			"passport,ABC123456,Yo Mismo,2020-01-01,2030-01-01,Madrid,\n",
		forma: FormaIdentidad,
		tipo:  TipoIdentidad,
		revisar: func(t *testing.T, e Entrada) {
			if e.NumeroDocumento != "ABC123456" || e.Documento != "passport" {
				t.Errorf("documento mal leído: %+v", e)
			}
			if e.NombreCompleto != "Yo Mismo" {
				t.Errorf("el nombre no es el título de una cuenta: %+v", e)
			}
			// Lo que no tiene campo propio no se tira: se junta en las notas, y con
			// el nombre de su columna delante para que se sepa de qué es.
			if !strings.Contains(e.Notas, "Madrid") || !strings.Contains(e.Notas, "place_of_issue") {
				t.Errorf("notas: %q", e.Notas)
			}
		},
	}, {
		// **El quinto, que faltaba y le daba nombre a esta prueba** (ADR 0047).
		// Aquí va una fila para que la tabla diga la verdad; lo que este fichero
		// tiene de particular se ejercita entero en `TestPersonalinfoDeDashlane`.
		nombre: "personalinfo.csv",
		csv: "type,title,first_name,last_name,email,email_type,item_name,phone_number,place_of_birth\n" +
			"email,,,,alvaro@webcafeina.com,personal,Correo electrónico 1,,\n",
		forma: FormaPersonal,
		tipo:  TipoPersonal,
		revisar: func(t *testing.T, e Entrada) {
			if e.Correo != "alvaro@webcafeina.com" || e.Titulo != "Correo electrónico 1" {
				t.Errorf("dato personal mal leído: %+v", e)
			}
			// Un correo **no** es el usuario de una cuenta, y la tabla común dice
			// que sí. Si se cuela, esto entra como credencial sin sitio.
			if e.Usuario != "" {
				t.Errorf("el correo ha caído en el usuario: %+v", e)
			}
		},
	}}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if f := FormaDeLaCabecera(cabeceraDe(c.csv)); f != c.forma {
				t.Fatalf("forma %d, y quiero %d", f, c.forma)
			}
			entradas, _, err := Leer([]byte(c.csv), nil)
			if err != nil {
				t.Fatalf("no se ha podido leer: %v", err)
			}
			if len(entradas) != 1 {
				t.Fatalf("%d entradas", len(entradas))
			}
			if entradas[0].Tipo != c.tipo {
				t.Errorf("tipo %q, y quiero %q", entradas[0].Tipo, c.tipo)
			}
			c.revisar(t, entradas[0])
		})
	}
}

func cabeceraDe(csv string) []string {
	return strings.Split(strings.SplitN(csv, "\n", 2)[0], ",")
}

// **Varias tarjetas no son la misma tarjeta.** Con la huella de una credencial
// —sitio más usuario— todas tenían la misma, porque ninguna tiene ni sitio ni
// usuario: importar cinco marcaba cuatro como duplicadas. Lo mismo con las notas.
func TestVariasTarjetasYVariasNotasNoSonDuplicadas(t *testing.T) {
	b, _, _ := nueva(t)

	tarjetas := "type,account_name,cc_number,code,expiration_month,expiration_year\n" +
		"credit_card,La azul,4111111111111111,111,01,2030\n" +
		"credit_card,La negra,5555555555554444,222,02,2031\n" +
		"credit_card,La de la empresa,378282246310005,333,03,2032\n"

	entradas, _, err := Leer([]byte(tarjetas), nil)
	if err != nil {
		t.Fatal(err)
	}
	r, err := b.Importar(entradas, "Dashlane")
	if err != nil {
		t.Fatal(err)
	}
	if r.Metidas != 3 || r.Repetidas != 0 || r.Conflictos != 0 {
		t.Errorf("%+v; y son tres tarjetas distintas", r)
	}

	notas := "title,note\nUna,lo que sea\nOtra,otra cosa\n"
	entradas, _, err = Leer([]byte(notas), nil)
	if err != nil {
		t.Fatal(err)
	}
	if r, err = b.Importar(entradas, "Dashlane"); err != nil {
		t.Fatal(err)
	}
	if r.Repetidas != 0 || r.Conflictos != 0 {
		t.Errorf("%+v: son dos notas distintas", r)
	}

	// Y la misma tarjeta escrita de otra forma **sí** es la misma.
	otraVez := "type,account_name,cc_number,code,expiration_month,expiration_year\n" +
		"credit_card,La azul,4111 1111 1111 1111,111,01,2030\n"
	entradas, _, err = Leer([]byte(otraVez), nil)
	if err != nil {
		t.Fatal(err)
	}
	if r, err = b.Importar(entradas, "Dashlane"); err != nil {
		t.Fatal(err)
	}
	if r.Repetidas != 1 || r.Metidas != 0 {
		t.Errorf("%+v: la misma tarjeta con espacios es la misma tarjeta", r)
	}
}

// Lo que sale de la bóveda tiene que poder volver a entrar **de una vez**, con
// tarjetas y documentos incluidos. Es la mitad de lo que significa poder salir.
func TestLoExportadoVuelveAEntrarConTodo(t *testing.T) {
	b, _, _ := nueva(t)

	todo := []Entrada{
		{Tipo: TipoCredencial, Titulo: "Banco", Usuario: "yo", Secreto: "s3cr3t0",
			Sitios: []string{"https://banco.es"}, TOTP: "JBSWY3DP"},
		{Tipo: TipoTarjeta, Titulo: "La azul", Titular: "Yo Mismo",
			Numero: "4111111111111111", Caduca: "01/2030", Verificacion: "111"},
		{Tipo: TipoIdentidad, Titulo: "Pasaporte", NombreCompleto: "Yo Mismo",
			Documento: "passport", NumeroDocumento: "ABC123456"},
		{Tipo: TipoNota, Titulo: "La caja fuerte", Notas: "la combinación es 1234"},
		{Tipo: TipoPersonal, Titulo: "Correo electrónico 1", Correo: "yo@ejemplo.com"},
		{Tipo: TipoPersonal, Titulo: "Casa", Telefono: "600111222", Nacimiento: "1980-01-01",
			Destinatario: "Yo Mismo", Calle: "Calle Mayor 1", Edificio: "Portal B",
			Piso: "3", Puerta: "B", CodigoPostal: "28001", Ciudad: "Madrid",
			Provincia: "Madrid", Pais: "España"},
		// **Un nombre a secas no tiene ningún campo que diga qué es**, y es media
		// exportación de datos personales. Volvía convertido en una credencial sin
		// usuario ni contraseña: por eso la forma de Esfinge lee su columna `type`,
		// que es lo que nosotros mismos escribimos y no puede mentir.
		{Tipo: TipoPersonal, Titulo: "Álvaro Cabezas", NombreCompleto: "Álvaro Cabezas"},
		// La red wifi: **con `oculta` a verdadero**, porque un booleano que se escribe
		// como columna vacía cuando es falso solo se comprueba de verdad en el otro caso.
		{Tipo: TipoWifi, Titulo: "La oficina", SSID: "WEBCAFEINA",
			Secreto: "la-clave", Seguridad: "wpa", Oculta: true},
	}
	if _, err := b.Importar(todo, "una prueba"); err != nil {
		t.Fatal(err)
	}

	var salida bytes.Buffer
	if err := b.Exportar(&salida); err != nil {
		t.Fatal(err)
	}

	// Se reconoce como nuestro y no como un CSV de tarjetas, que es lo que
	// pasaría mirando solo la columna del número.
	if f := FormaDeLaCabecera(cabeceraDe(salida.String())); f != FormaEsfinge {
		t.Fatalf("lo nuestro se lee como forma %d", f)
	}

	vueltas, _, err := Leer(salida.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(vueltas) != len(todo) {
		t.Fatalf("salieron %d y volvieron %d", len(todo), len(vueltas))
	}

	porTitulo := map[string]Entrada{}
	for _, e := range vueltas {
		porTitulo[e.Titulo] = e
	}
	for _, quiero := range todo {
		tengo, hay := porTitulo[quiero.Titulo]
		if !hay {
			t.Errorf("«%s» no ha vuelto", quiero.Titulo)
			continue
		}
		if tengo.Tipo != quiero.Tipo {
			t.Errorf("«%s» vuelve como %q y salió como %q", quiero.Titulo, tengo.Tipo, quiero.Tipo)
		}
		if tengo.Secreto != quiero.Secreto || tengo.Numero != quiero.Numero ||
			tengo.NumeroDocumento != quiero.NumeroDocumento || tengo.Notas != quiero.Notas {
			t.Errorf("«%s» ha vuelto distinta: %+v", quiero.Titulo, tengo)
		}
	}
}

// **Pasar el mismo fichero dos veces no puede cambiar nada.**
//
// Es la prueba que faltaba y la que se echó de menos de la peor forma: el cliente
// importó su `credentials.csv`, lo volvió a importar por si acaso, y se encontró
// con ciento treinta entradas donde había sesenta y cinco. Los repetidos se
// marcaban —esa parte funcionaba— pero se metían igual, y marcar no sirve de nada
// cuando lo que hay que hacer es no meterlos.
func TestPasarElMismoFicheroDosVecesNoCambiaNada(t *testing.T) {
	b, _, _ := nueva(t)

	csv := "username,title,password,note,url,category\n" +
		"yo@ejemplo.com,Banco,s3cr3t0,una nota,https://banco.es,Finanzas\n" +
		"otro@ejemplo.com,Correo,otra clave,,https://correo.es,\n" +
		"tercero@ejemplo.com,Tienda,y otra más,,https://tienda.es,Compras\n"

	entradas, _, err := Leer([]byte(csv), nil)
	if err != nil {
		t.Fatal(err)
	}

	primera, err := b.Importar(entradas, "Dashlane")
	if err != nil {
		t.Fatal(err)
	}
	if primera.Metidas != 3 {
		t.Fatalf("la primera pasada: %+v", primera)
	}

	// La segunda, con el mismo fichero recién leído otra vez, como haría una
	// persona: mismo CSV, mismo botón.
	entradas, _, err = Leer([]byte(csv), nil)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := b.Importar(entradas, "Dashlane")
	if err != nil {
		t.Fatal(err)
	}
	if segunda.Metidas != 0 || segunda.Repetidas != 3 || segunda.Conflictos != 0 {
		t.Errorf("la segunda pasada: %+v", segunda)
	}
	if b.Cuantas() != 3 {
		t.Errorf("quedan %d entradas donde hay tres", b.Cuantas())
	}

	// Y una tercera con **una contraseña cambiada** sí entra, marcada: eso no es
	// una repetición, es una cuenta con dos contraseñas y una de las dos está mal.
	cambiado := strings.Replace(csv, "s3cr3t0", "la nueva", 1)
	entradas, _, err = Leer([]byte(cambiado), nil)
	if err != nil {
		t.Fatal(err)
	}
	tercera, err := b.Importar(entradas, "Dashlane")
	if err != nil {
		t.Fatal(err)
	}
	if tercera.Metidas != 1 || tercera.Conflictos != 1 || tercera.Repetidas != 2 {
		t.Errorf("con una contraseña cambiada: %+v", tercera)
	}
	if b.Cuantas() != 4 {
		t.Errorf("quedan %d entradas donde hay cuatro", b.Cuantas())
	}
}

// Y lo mismo con las otras clases, que se identifican por otra cosa: una tarjeta
// por su número y un documento por el suyo.
func TestReimportarTarjetasYDocumentosTampocoDuplica(t *testing.T) {
	b, _, _ := nueva(t)

	ficheros := []string{
		"type,account_name,account_holder,cc_number,code,expiration_month,expiration_year\n" +
			"credit_card,La azul,Yo Mismo,4111111111111111,111,01,2030\n",
		"type,number,name,issue_date,expiration_date,place_of_issue\n" +
			"passport,ABC123456,Yo Mismo,2020-01-01,2030-01-01,Madrid\n",
		"title,note\nLa caja fuerte,la combinación es 1234\n",
	}

	for vuelta := 1; vuelta <= 2; vuelta++ {
		for _, csv := range ficheros {
			entradas, _, err := Leer([]byte(csv), nil)
			if err != nil {
				t.Fatal(err)
			}
			r, err := b.Importar(entradas, "Dashlane")
			if err != nil {
				t.Fatal(err)
			}
			if vuelta == 2 && (r.Metidas != 0 || r.Repetidas != 1) {
				t.Errorf("segunda vuelta de un fichero: %+v", r)
			}
		}
	}
	if b.Cuantas() != 3 {
		t.Errorf("quedan %d entradas donde hay tres", b.Cuantas())
	}
}

// **«¿Están todas?» tiene que poder contestarse con lo que devuelve Leer.**
//
// Es la pregunta que se hace cualquiera después de importar, y hasta ahora no
// tenía respuesta: se veían 65 entradas dentro y no había forma de saber si el
// fichero traía 65 u 80. Contar las líneas por fuera tampoco vale —una nota con
// saltos de línea ocupa varias— y por eso el número lo tiene que dar quien ya ha
// leído el CSV de verdad.
func TestLeerCuentaLasFilasDelFichero(t *testing.T) {
	csv := "title,password,note\n" +
		"Banco,s3cr3t0,\n" +
		"Correo,otra,\"una nota\ncon dos líneas\"\n" +
		",,\n" + // una fila vacía, de las que dejan los exportadores al final
		"Tienda,y otra,\n"

	entradas, lectura, err := Leer([]byte(csv), nil)
	if err != nil {
		t.Fatal(err)
	}

	// Cuatro filas de datos, aunque el fichero tenga cinco saltos de línea: la
	// nota con dos líneas es **una** fila.
	if lectura.Filas != 4 {
		t.Errorf("filas %d, y el fichero trae cuatro", lectura.Filas)
	}
	if lectura.Vacias != 1 {
		t.Errorf("vacías %d, y hay una", lectura.Vacias)
	}
	if len(entradas) != 3 {
		t.Errorf("%d entradas, y hay tres", len(entradas))
	}
	// La cuenta tiene que cerrar, que es lo que la ventana enseña.
	if len(entradas)+lectura.Vacias != lectura.Filas {
		t.Errorf("la cuenta no cierra: %d + %d ≠ %d",
			len(entradas), lectura.Vacias, lectura.Filas)
	}
}

// **`personalinfo.csv`, el sexto fichero de Dashlane** (ADR 0047).
//
// La cabecera es la de verdad, entera y con las columnas en su orden: son 24 para
// seis clases de dato, y **en cada fila vienen casi todas vacías**. Las dos
// primeras filas son las del cliente tal cual las exportó él; las otras dos
// ejercitan las clases que él no tenía guardadas.
//
// Tres cosas que este fichero hace y ningún otro:
//
//   - **`title` viene vacío en todas las filas.** Lo que se parece a un título está
//     en `item_name`, y solo en algunas clases: la del nombre no lo trae. Sin
//     `tituloDeReserva` mirando el nombre, esa fila entraba sin título.
//   - **`login` no es el usuario de ninguna cuenta**, y dejarlo en el campo usuario
//     —que es lo que dice la tabla común— clasificaba la fila como credencial.
//   - **La dirección viene en nueve columnas y en el orden de Dashlane**, que no es
//     el del sobre: `address, country, state, city, zip`.
func TestPersonalinfoDeDashlane(t *testing.T) {
	const cabecera = "type,title,first_name,middle_name,last_name,login,date_of_birth," +
		"place_of_birth,email,email_type,item_name,phone_number,address,country,state,city," +
		"zip,address_recipient,address_building,address_apartment,address_floor," +
		"address_door_code,job_title,url"
	const fichero = cabecera + "\n" +
		"name,,Álvaro,,Cabezas,alvarocabezas,,,,,,,,,,,,,,,,,,\n" +
		"email,,,,,,,,alvaro@webcafeina.com,personal,Correo electrónico 1,,,,,,,,,,,,,\n" +
		"phone,,,,,,,,,,Teléfono 1,+34 600 11 22 33,,,,,,,,,,,,\n" +
		"address,,,,,,,,,,Casa,,Calle Mayor 1,España,Madrid,Madrid,28001,Álvaro Cabezas,Portal B,B,3,1234,,\n"

	if f := FormaDeLaCabecera(cabeceraDe(fichero)); f != FormaPersonal {
		t.Fatalf("forma %d, y quiero FormaPersonal (%d)", f, FormaPersonal)
	}

	entradas, lectura, err := Leer([]byte(fichero), nil)
	if err != nil {
		t.Fatalf("no se ha podido leer: %v", err)
	}
	if len(entradas) != 4 || lectura.Vacias != 0 {
		t.Fatalf("%d entradas y %d vacías, de 4 filas", len(entradas), lectura.Vacias)
	}
	for _, e := range entradas {
		if e.Tipo != TipoPersonal {
			t.Errorf("«%s» ha entrado como %q", e.Titulo, e.Tipo)
		}
		if e.Titulo == "" {
			t.Errorf("una entrada sin título es una entrada que no se encuentra: %+v", e)
		}
		if e.Usuario != "" || e.Secreto != "" {
			t.Errorf("«%s» ha salido con usuario o contraseña: %+v", e.Titulo, e)
		}
	}

	nombre, correo, telefono, casa := entradas[0], entradas[1], entradas[2], entradas[3]

	// El nombre: tres columnas en una, y el título sale de él porque no hay otro.
	if nombre.NombreCompleto != "Álvaro Cabezas" || nombre.Titulo != "Álvaro Cabezas" {
		t.Errorf("el nombre: %+v", nombre)
	}
	// `login` es el alias que esa persona usa, no la cuenta de ningún sitio.
	if !strings.Contains(nombre.Notas, "alvarocabezas") {
		t.Errorf("el login se ha perdido: %q", nombre.Notas)
	}

	if correo.Correo != "alvaro@webcafeina.com" || correo.Titulo != "Correo electrónico 1" {
		t.Errorf("el correo: %+v", correo)
	}
	if telefono.Telefono != "+34 600 11 22 33" || telefono.Titulo != "Teléfono 1" {
		t.Errorf("el teléfono: %+v", telefono)
	}

	// **La dirección entra en sus nueve trozos**, que es como la da Dashlane y como
	// la pide un formulario (ADR 0047, revisada en la 2.31.0). Se comprueba campo a
	// campo porque el fallo que esto evita es silencioso: dos columnas que caen en
	// el mismo campo y una se pierde, como le pasó al piso y a la puerta.
	if casa.Destinatario != "Álvaro Cabezas" || casa.Calle != "Calle Mayor 1" ||
		casa.Edificio != "Portal B" || casa.Piso != "3" || casa.Puerta != "B" ||
		casa.CodigoPostal != "28001" || casa.Ciudad != "Madrid" ||
		casa.Provincia != "Madrid" || casa.Pais != "España" {
		t.Errorf("la dirección, por trozos: %+v", casa)
	}

	// Y compuesta para leerla, **en el orden del sobre**: por orden de columna
	// saldría «Calle Mayor 1, España, Madrid, Madrid, 28001», que no es una
	// dirección sino una lista de campos. La provincia no se repite cuando se llama
	// igual que la ciudad, que en media España es lo normal.
	quiero := "Álvaro Cabezas\nCalle Mayor 1, Portal B\n3, B\n28001 Madrid\nEspaña"
	if casa.Direccion() != quiero {
		t.Errorf("la dirección compuesta es\n%q\ny la quiero\n%q", casa.Direccion(), quiero)
	}
	// **El código del portal es un secreto**, y lo único que se vacía de lo que se
	// escribe suelto son las notas.
	if !strings.Contains(casa.Notas, "1234") {
		t.Errorf("el código del portal no está en las notas: %q", casa.Notas)
	}
	if strings.Contains(casa.Direccion(), "1234") {
		t.Errorf("el código del portal está escrito en la dirección: %q", casa.Direccion())
	}
}

// **Cuatro datos personales no son el mismo dato personal.**
//
// Es el fallo de las tarjetas con otra cara y por eso hay prueba: ninguna de estas
// filas tiene sitio ni usuario, así que con la huella de una credencial todas
// tienen la misma, y con la del título tampoco se salvan —«Correo electrónico 1»
// y «Correo electrónico 2» sí, pero dos exportaciones del mismo Dashlane traen el
// mismo rótulo—. Lo que las distingue es lo suyo: el correo, el teléfono, la
// dirección.
func TestVariosDatosPersonalesNoSonDuplicados(t *testing.T) {
	b, _, _ := nueva(t)
	entradas := []Entrada{
		{Tipo: TipoPersonal, Titulo: "Correo electrónico 1", Correo: "uno@ejemplo.com"},
		{Tipo: TipoPersonal, Titulo: "Correo electrónico 1", Correo: "dos@ejemplo.com"},
		{Tipo: TipoPersonal, Titulo: "Teléfono 1", Telefono: "600111222"},
		{Tipo: TipoPersonal, Titulo: "Teléfono 1", Telefono: "600333444"},
		{Tipo: TipoPersonal, Titulo: "Casa", Calle: "Calle Mayor 1", CodigoPostal: "28001", Ciudad: "Madrid"},
		{Tipo: TipoPersonal, Titulo: "Casa", Calle: "Calle Menor 2", CodigoPostal: "08001", Ciudad: "Barcelona"},
	}
	r, err := b.Importar(entradas, "Dashlane")
	if err != nil {
		t.Fatal(err)
	}
	// **`Conflictos` es lo que hay que mirar aquí, y no `Metidas`.** Una huella que
	// choca no impide que la entrada entre: la marca como «la misma cuenta con otro
	// secreto» y la mete igual. Así que contando solo las metidas, esta prueba
	// pasaba en verde con la huella del dato personal quitada —comprobado
	// mutándola—, y lo que el cliente habría visto es «6 conflictos» al importar
	// seis datos que no tienen nada que ver entre sí.
	if r.Metidas != 6 || r.Repetidas != 0 || r.Conflictos != 0 {
		t.Fatalf("%+v, y las seis son distintas", r)
	}

	// Y lo de siempre por el otro lado: pasar el mismo fichero dos veces no lo
	// duplica. Sin esto, la huella podría ser «distinta cada vez» y la prueba de
	// arriba pasaría igual.
	r2, err := b.Importar(entradas, "Dashlane")
	if err != nil {
		t.Fatal(err)
	}
	if r2.Metidas != 0 || r2.Repetidas != 6 {
		t.Fatalf("a la segunda: %d metidas y %d repetidas", r2.Metidas, r2.Repetidas)
	}
}

// **Dos datos personales con el mismo rótulo no son la misma cuenta.**
//
// `claveDeCuenta` es una lista de campos escrita a mano, y lo que no esté en ella
// no distingue: con los cuatro campos nuevos fuera, «Correo electrónico 1» con dos
// direcciones distintas era la misma cuenta y **quitar repetidas borraba una**.
// Es el camino de «Juntar» al entrar en una cuenta (ADR 0039), no el del
// importador, y por eso no lo cubre la prueba de arriba.
func TestDosDatosPersonalesConElMismoRotuloNoSeFusionan(t *testing.T) {
	uno := Entrada{Tipo: TipoPersonal, Titulo: "Correo electrónico 1", Correo: "uno@ejemplo.com"}
	dos := Entrada{Tipo: TipoPersonal, Titulo: "Correo electrónico 1", Correo: "dos@ejemplo.com"}
	if claveDeCuenta(uno) == claveDeCuenta(dos) {
		t.Fatal("dos correos distintos con el mismo rótulo salen como la misma cuenta")
	}
	// Y el teléfono, la dirección y la fecha por separado: con uno solo en la
	// lista, la prueba pasaría dejando fuera los otros tres.
	for _, par := range [][2]Entrada{
		{{Tipo: TipoPersonal, Telefono: "600111222"}, {Tipo: TipoPersonal, Telefono: "600333444"}},
		{{Tipo: TipoPersonal, Calle: "Mayor 1"}, {Tipo: TipoPersonal, Calle: "Menor 2"}},
		{{Tipo: TipoPersonal, CodigoPostal: "28001"}, {Tipo: TipoPersonal, CodigoPostal: "08001"}},
		{{Tipo: TipoPersonal, Ciudad: "Madrid"}, {Tipo: TipoPersonal, Ciudad: "Barcelona"}},
		{{Tipo: TipoPersonal, Piso: "3"}, {Tipo: TipoPersonal, Piso: "4"}},
		{{Tipo: TipoPersonal, Nacimiento: "1980-01-01"}, {Tipo: TipoPersonal, Nacimiento: "1990-01-01"}},
	} {
		if claveDeCuenta(par[0]) == claveDeCuenta(par[1]) {
			t.Errorf("no distingue: %+v y %+v", par[0], par[1])
		}
	}
}

// **Las llaves de acceso no salen en la exportación en claro** (ADR 0048).
//
// Una fila de CSV con una clave privada dentro es lo más peligroso que Esfinge
// escribiría nunca en el disco, y además no le sirve a ningún gestor: ninguno
// sabe leerla. Lo que esta prueba vigila es que salga **todo lo demás**: dejar
// fuera una clase entera es fácil de hacer de más.
func TestLaExportacionEnClaroNoLlevaLlaves(t *testing.T) {
	b, _, _ := nueva(t)
	todo := []Entrada{
		{Tipo: TipoCredencial, Titulo: "Banco", Usuario: "yo", Secreto: "s3cr3t0"},
		{Tipo: TipoLlave, Titulo: "GitHub", RPID: "github.com", IDCredencial: "Y3JlZC0x",
			NombreVisible: "yo@ejemplo.com", Algoritmo: -7, ClavePrivada: "no-tiene-que-salir"},
		{Tipo: TipoPersonal, Titulo: "Correo electrónico 1", Correo: "yo@ejemplo.com"},
	}
	if _, err := b.Importar(todo, "una prueba"); err != nil {
		t.Fatal(err)
	}

	var salida bytes.Buffer
	if err := b.Exportar(&salida); err != nil {
		t.Fatal(err)
	}
	texto := salida.String()
	if strings.Contains(texto, "no-tiene-que-salir") {
		t.Error("la clave privada está escrita en claro en el CSV")
	}
	if strings.Contains(texto, "GitHub") {
		t.Error("la llave sale en la exportación en claro")
	}
	// Y lo demás sí, que es la otra mitad.
	if !strings.Contains(texto, "s3cr3t0") || !strings.Contains(texto, "Correo electrónico 1") {
		t.Errorf("se ha llevado por delante lo que sí tenía que salir:\n%s", texto)
	}
}

// **Y salen por su puerta, cifradas.**
//
// Es la excepción a «una bóveda de la que no se puede salir es una trampa», y lo
// que la sostiene: se puede salir, pero lo que sale está en un contenedor ESF1
// con su clave. Lo que se comprueba es lo que importa de verdad: que **la clave
// privada no aparece en el fichero** —si el cifrado no se hubiera aplicado, ahí
// estaría en claro— y que lo que sale se vuelve a abrir entero.
func TestLasLlavesSalenCifradas(t *testing.T) {
	b, _, _ := nueva(t)
	if _, err := b.Importar([]Entrada{
		{Tipo: TipoCredencial, Titulo: "Banco", Secreto: "s3cr3t0"},
		{Tipo: TipoLlave, Titulo: "GitHub", RPID: "github.com", IDCredencial: "Y3JlZC0x",
			NombreVisible: "yo@ejemplo.com", Algoritmo: -7, ClavePrivada: "la-privada-de-github"},
		{Tipo: TipoLlave, Titulo: "Google", RPID: "google.com", IDCredencial: "Y3JlZC0y",
			NombreVisible: "yo@ejemplo.com", Algoritmo: -7, ClavePrivada: "la-privada-de-google"},
	}, "una prueba"); err != nil {
		t.Fatal(err)
	}

	var fuera bytes.Buffer
	cuantas, err := b.ExportarLlaves(&fuera, "la clave del fichero")
	if err != nil {
		t.Fatal(err)
	}
	if cuantas != 2 {
		t.Errorf("dice que ha sacado %d y son 2", cuantas)
	}
	if bytes.Contains(fuera.Bytes(), []byte("la-privada-de-github")) {
		t.Fatal("la clave privada está en claro dentro del fichero «cifrado»")
	}
	if bytes.Contains(fuera.Bytes(), []byte("github.com")) {
		t.Error("hasta el sitio sale en claro: eso es una lista de dónde tienes llaves")
	}

	// Se abre con su clave y está entero.
	dentro, err := cripto.Abrir(fuera.Bytes(), []byte("la clave del fichero"))
	if err != nil {
		t.Fatalf("no se puede volver a abrir: %v", err)
	}
	var leido struct {
		Esfinge string    `json:"esfinge"`
		Version int       `json:"version"`
		Llaves  []Entrada `json:"llaves"`
	}
	if err := json.Unmarshal(dentro, &leido); err != nil {
		t.Fatal(err)
	}
	if leido.Esfinge != "llaves de acceso" || leido.Version != 1 || len(leido.Llaves) != 2 {
		t.Fatalf("lo de dentro no cuadra: %+v", leido)
	}
	if leido.Llaves[0].ClavePrivada == "" {
		t.Error("ha salido sin la clave privada, que es lo único que no se puede rehacer")
	}

	// Y con otra clave no se abre, que es lo que significa que estaba cifrado.
	if _, err := cripto.Abrir(fuera.Bytes(), []byte("otra cualquiera")); err == nil {
		t.Error("se abre con cualquier clave")
	}
}

// **Varias llaves de acceso no son la misma llave**, ni siquiera dos del mismo
// sitio para la misma persona.
//
// Es el fallo de las tarjetas con la peor cara posible: una llave no tiene ni
// sitio ni usuario en los campos que mira la huella de una credencial, así que
// sin huella propia todas caen en la del título — y dos llaves de GitHub se
// titulan igual. Lo que hay que mirar es **`Conflictos`**, no `Metidas`: una
// huella que choca no impide que la entrada entre, la marca y la mete igual, así
// que contando las metidas esta prueba pasaría en verde con la huella rota.
func TestVariasLlavesNoSonLaMisma(t *testing.T) {
	b, _, _ := nueva(t)
	llaves := []Entrada{
		{Tipo: TipoLlave, Titulo: "GitHub", RPID: "github.com", IDCredencial: "Y3JlZC0x",
			NombreVisible: "yo@ejemplo.com", Algoritmo: -7, ClavePrivada: "una"},
		// La misma persona, el mismo sitio, **otra llave**: pasa cada vez que se
		// registra un equipo nuevo.
		{Tipo: TipoLlave, Titulo: "GitHub", RPID: "github.com", IDCredencial: "Y3JlZC0y",
			NombreVisible: "yo@ejemplo.com", Algoritmo: -7, ClavePrivada: "otra"},
		{Tipo: TipoLlave, Titulo: "Google", RPID: "google.com", IDCredencial: "Y3JlZC0z",
			NombreVisible: "yo@ejemplo.com", Algoritmo: -7, ClavePrivada: "tercera"},
	}
	r, err := b.Importar(llaves, "una prueba")
	if err != nil {
		t.Fatal(err)
	}
	if r.Metidas != 3 || r.Repetidas != 0 || r.Conflictos != 0 {
		t.Fatalf("%+v, y las tres son distintas", r)
	}

	// Y por el otro lado: la misma, dos veces, sí es la misma.
	r2, err := b.Importar(llaves, "una prueba")
	if err != nil {
		t.Fatal(err)
	}
	if r2.Metidas != 0 || r2.Repetidas != 3 {
		t.Fatalf("a la segunda: %+v", r2)
	}

	// Y al juntar dos bóvedas (ADR 0039), que es otra lista y otro camino: dos
	// llaves distintas no pueden salir como la misma cuenta, o quitar repetidas
	// **borraría una llave**, y eso no se restablece por correo.
	if claveDeCuenta(llaves[0]) == claveDeCuenta(llaves[1]) {
		t.Error("dos llaves del mismo sitio salen como la misma cuenta")
	}
}

// **`wifi.csv`, el sexto fichero de Dashlane** (ADR 0049).
//
// La cabecera es la de verdad, con su `encription_type` mal escrito en origen. Las dos
// filas imitan el fichero del cliente en lo que importa: **dicen `unsecured` teniendo
// contraseña**, **traen `name` y `note` vacíos** y **comparten la misma clave**. Las tres
// cosas rompen algo distinto si se hace lo obvio.
func TestWifiDeDashlane(t *testing.T) {
	const cabecera = "ssid,passphrase,name,note,hidden,encription_type"
	const fichero = cabecera + "\n" +
		"WEBCAFEINA_PLUS,unaclavecompartida,,,false,unsecured\n" +
		"WEBCAFEINA,unaclavecompartida,,,false,unsecured\n"

	if f := FormaDeLaCabecera(cabeceraDe(fichero)); f != FormaWifi {
		t.Fatalf("forma %d, y quiero FormaWifi (%d)", f, FormaWifi)
	}

	entradas, lectura, err := Leer([]byte(fichero), nil)
	if err != nil {
		t.Fatalf("no se ha podido leer: %v", err)
	}
	if len(entradas) != 2 || lectura.Vacias != 0 {
		t.Fatalf("%d entradas y %d vacías, de 2 filas", len(entradas), lectura.Vacias)
	}

	for _, e := range entradas {
		if e.Tipo != TipoWifi {
			t.Errorf("«%s» ha entrado como %q", e.Titulo, e.Tipo)
		}
		// **El título sale del nombre de la red**: `name` viene vacío en todas las filas,
		// y una entrada sin título es una entrada que no se encuentra.
		if e.Titulo != e.SSID {
			t.Errorf("el título es %q y la red se llama %q", e.Titulo, e.SSID)
		}
		// **Y la seguridad se corrige.** Esto es lo que separa un código que funciona de
		// uno que no: creyendo al fichero, el QR saldría como red abierta y el móvil
		// intentaría entrar sin clave.
		if e.Seguridad != "wpa" {
			t.Errorf("«%s» dice que su seguridad es %q, y tiene contraseña", e.SSID, e.Seguridad)
		}
		if e.Oculta {
			t.Errorf("«%s» ha entrado como oculta y el fichero dice que no", e.SSID)
		}
		if e.Secreto != "unaclavecompartida" {
			t.Errorf("«%s» ha entrado con la clave %q", e.SSID, e.Secreto)
		}
	}

	// **Y las dos entran**: comparten contraseña y son redes distintas. Mirando
	// `Conflictos` y no solo `Metidas`, que es lo que ya pasó con las tarjetas — una
	// huella común las habría marcado como duplicadas y la segunda no entraría.
	b, _, _ := nueva(t)
	r, err := b.Importar(entradas, "dashlane")
	if err != nil {
		t.Fatal(err)
	}
	if r.Metidas != 2 || r.Repetidas != 0 || r.Conflictos != 0 {
		t.Errorf("%d metidas, %d repetidas y %d conflictos, de dos redes distintas",
			r.Metidas, r.Repetidas, r.Conflictos)
	}
}

// Una red abierta de verdad: sin contraseña, la seguridad es «abierta» diga lo que diga.
func TestUnaRedSinClaveEsAbierta(t *testing.T) {
	const fichero = "ssid,passphrase,name,note,hidden,encription_type\n" +
		"Wifi del bar,,El bar de abajo,la de la terraza,true,wpa2\n"
	entradas, _, err := Leer([]byte(fichero), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(entradas) != 1 {
		t.Fatalf("%d entradas de una fila", len(entradas))
	}
	e := entradas[0]
	if e.Seguridad != "abierta" {
		t.Errorf("sin contraseña, la seguridad es %q", e.Seguridad)
	}
	if !e.Oculta {
		t.Error("el fichero dice que es oculta y no ha entrado así")
	}
	// Con `name` relleno, el título es el suyo y el SSID se queda aparte.
	if e.Titulo != "El bar de abajo" || e.SSID != "Wifi del bar" {
		t.Errorf("título %q y red %q", e.Titulo, e.SSID)
	}
	if e.Notas != "la de la terraza" {
		t.Errorf("las notas son %q", e.Notas)
	}
}
