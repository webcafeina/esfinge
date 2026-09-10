package boveda

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

// Importar credenciales de otro gestor.
//
// **Se mapea por el nombre de la columna, nunca por su posición.** Dashlane
// —como todos— cambia el orden y añade columnas entre versiones, y un importador
// que cuente columnas se rompe en silencio y mete la contraseña en el campo de
// las notas. Con un mapa de alias, el mismo código lee Dashlane, Bitwarden,
// 1Password, LastPass y el CSV de Chrome, que es el que acaba apareciendo tarde
// o temprano.
//
// Lo que este fichero cubre de los CSV del mundo real, todo aprendido a base de
// que falle:
//
//   - **La marca de orden de bytes (BOM)** al principio. Es el fallo clásico:
//     se pega al nombre de la primera columna y esa columna deja de reconocerse.
//   - Separador `;` en los CSV que han pasado por un Excel europeo.
//   - Saltos de línea **dentro** de una nota entrecomillada.
//   - Comillas a medias, que con `LazyQuotes` se toleran en vez de abortar.
//   - Filas con más o menos columnas que la cabecera.
//   - Bytes de Latin-1 de un fichero que pasó por Excel: si no es UTF-8 válido,
//     se relee como Windows-1252 en vez de meter caracteres rotos en la bóveda.

// Correspondencia dice qué columna del fichero va a qué campo de la entrada.
//
// Se devuelve para poder enseñarla antes de importar: adivinar bien casi siempre
// no es lo mismo que adivinar bien siempre, y aquí una columna mal puesta acaba
// con una contraseña en un campo que se ve en pantalla.
type Correspondencia map[string]string

// Campos a los que se puede mapear una columna.
const (
	CampoTitulo  = "titulo"
	CampoUsuario = "usuario"
	CampoSecreto = "secreto"
	CampoSitio   = "sitio"
	CampoNotas   = "notas"
	CampoTOTP    = "totp"
	CampoCarpeta = "carpeta"

	// Tarjetas.
	CampoTitular      = "titular"
	CampoNumero       = "numero"
	CampoCaduca       = "caduca"
	CampoVerificacion = "verificacion"
	// Y los dos trozos con los que Dashlane parte la caducidad, que hay que
	// volver a juntar.
	CampoCaducaMes = "caduca-mes"
	CampoCaducaAno = "caduca-ano"

	// Identidades y documentos.
	CampoNombre          = "nombre-completo"
	CampoDocumento       = "documento"
	CampoNumeroDocumento = "numero-documento"

	CampoIgnorar = ""
)

// Forma dice qué clase de fichero se está leyendo.
//
// **Hay que decidirla antes de mirar columna por columna, y no es una manía.**
// Dashlane no exporta un CSV: exporta cinco, uno por clase de dato, y una misma
// columna significa cosas distintas en cada uno. `number` es el número de una
// tarjeta en `payments.csv` y el de un pasaporte en `ids.csv`; `type` es la clase
// de documento allí y la clase de tarjeta aquí; `name` es el título de una cuenta
// en uno y el nombre de una persona en otro.
//
// Con una sola tabla de alias eso no se puede resolver: o se acierta en un
// fichero y se falla en el otro, o —lo que pasaba— las columnas de tarjetas y
// documentos no se reconocen y **el fichero entero se rechaza** diciendo que no
// parece una exportación de contraseñas. El cliente importó sus credenciales,
// vio 65 entradas y en Dashlane había más: lo que faltaba eran los otros cuatro
// ficheros.
type Forma int

const (
	// FormaCredencial es lo de siempre: usuario, contraseña y sitio. También las
	// notas seguras, que son este mismo fichero sin contraseña.
	FormaCredencial Forma = iota
	FormaTarjeta
	FormaIdentidad
	// FormaEsfinge es lo que exporta Esfinge: **una sola tabla con todas las
	// columnas**, justo lo contrario que Dashlane. Se reconoce sola para que lo
	// que sale de aquí pueda volver a entrar de una vez, que es la mitad de lo
	// que significa poder salir.
	FormaEsfinge
)

// FormaDeLaCabecera mira los nombres de las columnas y dice qué se está leyendo.
//
// Se decide por las columnas que **solo** aparecen en una clase de fichero, no
// por las que podrían aparecer en varias.
func FormaDeLaCabecera(cabecera []string) Forma {
	hay := map[string]bool{}
	for _, c := range cabecera {
		hay[normalizarColumna(c)] = true
	}

	switch {
	// Lo nuestro primero: es el único fichero que lleva a la vez columnas de
	// tarjeta y de documento, porque es el único que no separa por clases.
	case hay["cc_number"] && hay["document_number"]:
		return FormaEsfinge
	case hay["cc_number"] || hay["card_number"] || hay["cardnumber"]:
		return FormaTarjeta
	case hay["number"] && (hay["code"] || hay["expiration_month"] || hay["issuing_bank"]):
		return FormaTarjeta
	case hay["document_type"] || (hay["number"] && (hay["issue_date"] || hay["place_of_issue"])):
		return FormaIdentidad
	default:
		return FormaCredencial
	}
}

func normalizarColumna(c string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(c)), " ", "")
}

// aliasDe devuelve la tabla que toca para esa forma: lo común más lo suyo.
func aliasDe(f Forma) map[string]string {
	tabla := map[string]string{}
	for k, v := range alias {
		tabla[k] = v
	}
	for k, v := range aliasPorForma[f] {
		tabla[k] = v
	}
	return tabla
}

// aliasPorForma es lo que cambia de significado según el fichero. Lo de aquí
// **pisa** a la tabla común, que es justo lo que hace falta para `number`,
// `type` y `name`.
var aliasPorForma = map[Forma]map[string]string{
	FormaEsfinge: {
		"cardholder": CampoTitular, "cc_number": CampoNumero,
		"expiration_date": CampoCaduca, "cvv": CampoVerificacion,
		"full_name": CampoNombre, "document_type": CampoDocumento,
		"document_number": CampoNumeroDocumento,
		// «type» no se lee: la clase de entrada se deduce de los campos que vengan
		// rellenos, que es lo que no puede mentir.
	},
	FormaTarjeta: {
		"cc_number": CampoNumero, "card_number": CampoNumero,
		"cardnumber": CampoNumero, "number": CampoNumero,
		"numero": CampoNumero, "número": CampoNumero,
		"account_holder": CampoTitular, "cardholder": CampoTitular,
		"cardholdername": CampoTitular, "titular": CampoTitular, "name": CampoTitular,
		"code": CampoVerificacion, "cvv": CampoVerificacion, "cvc": CampoVerificacion,
		"security_code": CampoVerificacion, "verificacion": CampoVerificacion,
		"expiration_month": CampoCaducaMes, "exp_month": CampoCaducaMes,
		"expiration_year": CampoCaducaAno, "exp_year": CampoCaducaAno,
		"expiration_date": CampoCaduca, "expiry": CampoCaduca, "caduca": CampoCaduca,
		"expirationdate": CampoCaduca, "valid_until": CampoCaduca,
		"account_name": CampoTitulo, "issuing_bank": CampoCarpeta,
		// La clase de tarjeta no es un título ni un documento: es una nota.
		"type": CampoNotas, "country": CampoNotas,
	},
	FormaIdentidad: {
		"number": CampoNumeroDocumento, "numero": CampoNumeroDocumento,
		"document_type": CampoDocumento, "type": CampoDocumento,
		"name": CampoNombre, "full_name": CampoNombre, "fullname": CampoNombre,
		"nombrecompleto": CampoNombre, "nombre": CampoNombre,
		"expiration_date": CampoCaduca, "expiry": CampoCaduca,
		"issue_date": CampoNotas, "place_of_issue": CampoNotas,
		"state": CampoNotas, "country": CampoNotas,
	},
}

// alias reconocidos, en minúsculas y sin espacios. La lista es explícita para
// que añadir un gestor sea una decisión y no una expresión regular que crece.
var alias = map[string]string{
	// Título
	"title": CampoTitulo, "name": CampoTitulo, "nombre": CampoTitulo,
	"título": CampoTitulo, "titulo": CampoTitulo, "item": CampoTitulo,
	// Usuario
	"username": CampoUsuario, "user": CampoUsuario, "login": CampoUsuario,
	"usuario": CampoUsuario, "email": CampoUsuario, "correo": CampoUsuario,
	"login_username": CampoUsuario, "account": CampoUsuario,
	// Contraseña
	"password": CampoSecreto, "contraseña": CampoSecreto, "contrasena": CampoSecreto,
	"login_password": CampoSecreto, "secreto": CampoSecreto, "pass": CampoSecreto,
	// Sitio
	"url": CampoSitio, "urls": CampoSitio, "website": CampoSitio, "site": CampoSitio,
	"sitio": CampoSitio, "login_uri": CampoSitio, "web": CampoSitio,
	// Notas
	"note": CampoNotas, "notes": CampoNotas, "notas": CampoNotas,
	"comentarios": CampoNotas, "extra": CampoNotas, "comment": CampoNotas,
	// Segundo factor
	"otpsecret": CampoTOTP, "totp": CampoTOTP, "otp": CampoTOTP,
	"otpauth": CampoTOTP, "login_totp": CampoTOTP, "authenticator": CampoTOTP,
	// Carpeta
	"folder": CampoCarpeta, "grouping": CampoCarpeta, "category": CampoCarpeta,
	"carpeta": CampoCarpeta, "categoria": CampoCarpeta, "categoría": CampoCarpeta,
	"group": CampoCarpeta,
}

// ErrSinColumnas dice que el fichero no tiene ninguna columna reconocible.
var ErrSinColumnas = errors.New("Este fichero no parece una exportación de contraseñas: no reconozco ninguna columna")

// Leer analiza un CSV y devuelve las entradas junto con la correspondencia que
// ha adivinado, para poder enseñarla y corregirla antes de meter nada.
func Leer(datos []byte, mapa Correspondencia) ([]Entrada, Lectura, error) {
	texto := aTexto(datos)

	r := csv.NewReader(strings.NewReader(texto))
	r.Comma = separadorDe(texto)
	// Las dos banderas que hacen la diferencia entre importar un fichero real y
	// abortar a la primera: filas de longitud variable y comillas mal cerradas.
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	filas, err := r.ReadAll()
	if err != nil {
		return nil, Lectura{}, fmt.Errorf("No he podido leer el fichero: %w", err)
	}
	if len(filas) < 2 {
		return nil, Lectura{}, ErrSinColumnas
	}

	cabecera := filas[0]
	if mapa == nil {
		mapa = Adivinar(cabecera)
	}
	l := Lectura{Columnas: mapa, Filas: len(filas) - 1}
	if !mapa.sirve() {
		// **Con las columnas que traía dentro del mensaje.** Un «no lo reconozco» a
		// secas deja a quien lo ve sin nada que hacer ni nada que contar; con la
		// cabecera delante, el fichero se puede añadir a la tabla de alias en cinco
		// minutos. Los nombres de las columnas no son datos de nadie: son la forma
		// del fichero.
		return nil, l, fmt.Errorf("%w. Las que trae son: %s",
			ErrSinColumnas, strings.Join(cabecera, ", "))
	}

	var out []Entrada
	for _, fila := range filas[1:] {
		e := deFila(cabecera, fila, mapa)
		// Una fila sin nada que guardar no es una entrada, es una línea en blanco.
		if e.Titulo == "" && e.Usuario == "" && e.Secreto == "" && e.Notas == "" &&
			e.Numero == "" && e.NumeroDocumento == "" {
			continue
		}
		out = append(out, e)
	}
	l.Vacias = l.Filas - len(out)
	if len(out) == 0 {
		return nil, l, errors.New("El fichero se lee bien pero no tiene ninguna entrada")
	}
	return out, l, nil
}

// Lectura es lo que se ha entendido del fichero, aparte de las entradas.
//
// **`Filas` existe para poder contestar «¿están todas?»**, que es la pregunta que
// se hace cualquiera después de importar y que hasta ahora no tenía respuesta:
// se veían 65 entradas dentro y no había forma de saber si el fichero traía 65 u
// 80. Contar las líneas por fuera tampoco vale, porque una nota con saltos de
// línea ocupa varias.
type Lectura struct {
	// Columnas es lo que se ha adivinado, para poder enseñarlo y corregirlo.
	Columnas Correspondencia
	// Filas son las del fichero sin contar la cabecera.
	Filas int
	// Vacias son las que no llevaban nada que guardar.
	Vacias int
}

// Adivinar propone una correspondencia a partir de los nombres de las columnas.
func Adivinar(cabecera []string) Correspondencia {
	tabla := aliasDe(FormaDeLaCabecera(cabecera))

	mapa := Correspondencia{}
	for _, c := range cabecera {
		campo, hay := tabla[normalizarColumna(c)]
		if !hay {
			continue
		}
		// La primera columna que reclama un campo se lo queda: Dashlane exporta
		// «username», «username2» y «username3», y el bueno es el primero.
		//
		// Las notas son la excepción: en las tarjetas y en los documentos hay
		// varias columnas sueltas —el país, la fecha de expedición, dónde se
		// expidió— que no tienen sitio propio y que juntas sí valen algo. Se
		// acumulan en vez de quedarse con la primera.
		if campo == CampoNotas || !mapa.tiene(campo) {
			mapa[c] = campo
		}
	}
	return mapa
}

func (m Correspondencia) tiene(campo string) bool {
	for _, v := range m {
		if v == campo {
			return true
		}
	}
	return false
}

// sirve dice si con este mapa se puede sacar algo aprovechable.
//
// Lo que cuenta es que haya **algo que merezca la pena guardar bajo llave**: una
// contraseña, una nota, el número de una tarjeta o el de un documento. Hasta la
// 2.12.4 solo valían las dos primeras, y por eso `payments.csv` e `ids.csv` de
// Dashlane se rechazaban enteros.
func (m Correspondencia) sirve() bool {
	return m.tiene(CampoSecreto) || m.tiene(CampoNotas) ||
		m.tiene(CampoNumero) || m.tiene(CampoNumeroDocumento)
}

func deFila(cabecera, fila []string, mapa Correspondencia) Entrada {
	var e Entrada
	var notas []string
	var mes, ano string

	for i, col := range cabecera {
		if i >= len(fila) {
			break
		}
		valor := strings.TrimSpace(fila[i])
		if valor == "" {
			continue
		}
		switch mapa[col] {
		case CampoTitulo:
			e.Titulo = valor
		case CampoUsuario:
			e.Usuario = valor
		case CampoSecreto:
			e.Secreto = valor
		case CampoSitio:
			e.Sitios = append(e.Sitios, valor)
		case CampoNotas:
			// Con el nombre de la columna delante cuando hay varias, que si no se
			// quedan cuatro datos sueltos sin decir de qué son.
			notas = append(notas, etiquetar(col, valor, mapa))
		case CampoTOTP:
			e.TOTP = valor
		case CampoCarpeta:
			e.Carpeta = valor
		case CampoTitular:
			e.Titular = valor
		case CampoNumero:
			e.Numero = valor
		case CampoVerificacion:
			e.Verificacion = valor
		case CampoCaduca:
			e.Caduca = valor
		case CampoCaducaMes:
			mes = valor
		case CampoCaducaAno:
			ano = valor
		case CampoNombre:
			e.NombreCompleto = valor
		case CampoDocumento:
			e.Documento = valor
		case CampoNumeroDocumento:
			e.NumeroDocumento = valor
		}
	}

	// Dashlane parte la caducidad en dos columnas; aquí es un campo. Se juntan
	// como se escriben en la tarjeta.
	if e.Caduca == "" && (mes != "" || ano != "") {
		e.Caduca = strings.TrimPrefix(mes+"/"+ano, "/")
		e.Caduca = strings.TrimSuffix(e.Caduca, "/")
	}
	e.Notas = strings.Join(notas, "\n")

	e.Tipo = tipoDe(e)
	if e.Titulo == "" {
		e.Titulo = tituloDeReserva(e)
	}
	return e
}

// etiquetar pone delante el nombre de la columna cuando varias caen en las
// notas. Con una sola no hace falta: es «la nota», y ya.
func etiquetar(col, valor string, mapa Correspondencia) string {
	cuantas := 0
	for _, campo := range mapa {
		if campo == CampoNotas {
			cuantas++
		}
	}
	if cuantas < 2 {
		return valor
	}
	return col + ": " + valor
}

// tipoDe decide qué clase de entrada es por lo que se ha podido rellenar, no por
// lo que decía el fichero: un CSV puede mentir sobre sí mismo, y los campos no.
func tipoDe(e Entrada) Tipo {
	switch {
	case e.Numero != "" || e.Verificacion != "":
		return TipoTarjeta
	case e.NumeroDocumento != "" || e.Documento != "":
		return TipoIdentidad
	case e.Secreto == "" && e.Usuario == "" && e.Notas != "":
		return TipoNota
	default:
		return TipoCredencial
	}
}

// tituloDeReserva: una entrada sin título es una entrada que no se encuentra.
func tituloDeReserva(e Entrada) string {
	if t := primerNoVacio(e.Sitios...); t != "" {
		return t
	}
	return primerNoVacio(e.Usuario, e.NombreCompleto, e.Titular, e.Documento)
}

func primerNoVacio(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// aTexto quita la marca de orden de bytes y arregla la codificación.
func aTexto(b []byte) string {
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	if utf8.Valid(b) {
		return string(b)
	}
	// No es UTF-8: casi siempre es Windows-1252, de un fichero que pasó por
	// Excel. Se convierte a mano para no arrastrar una dependencia por esto.
	var sb strings.Builder
	for _, c := range b {
		sb.WriteRune(rune(c))
	}
	return sb.String()
}

// separadorDe mira la primera línea y decide. Un CSV guardado por un Excel en
// español viene con punto y coma.
func separadorDe(texto string) rune {
	linea, _, _ := strings.Cut(texto, "\n")
	comas := strings.Count(linea, ",")
	puntoYComa := strings.Count(linea, ";")
	tabuladores := strings.Count(linea, "\t")

	if puntoYComa > comas && puntoYComa >= tabuladores {
		return ';'
	}
	if tabuladores > comas {
		return '\t'
	}
	return ','
}

// Importar mete las entradas leídas en la bóveda, en su propia carpeta.
//
// **No escribe nada en el historial**, y es una regla absoluta: la ADR 0010
// decidió que el historial guarda nombres de fichero, y
// «credenciales-dashlane.csv» ahí es una señal de tráfico que apunta a lo que
// alguien acaba de exportar en claro.
//
// Los duplicados —mismo sitio y mismo usuario— se marcan pero no se fusionan
// solos: dos contraseñas distintas para la misma cuenta significan que una de
// las dos está mal, y adivinar cuál no es cosa de un importador.
func (b *Boveda) Importar(entradas []Entrada, deDonde string) (r Resumen, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return Resumen{}, ErrCerrada
	}
	carpeta := fmt.Sprintf("Importado de %s · %s", deDonde, time.Now().Format("2006-01-02"))
	ahora := time.Now().UTC().Format(time.RFC3339)

	// Dos índices, y son dos preguntas distintas: «¿es esta misma entrada?» y
	// «¿es esta misma cuenta con otra contraseña?».
	//
	// **Lo que está en la papelera no cuenta**, y desde que la papelera guarda lo
	// borrado entero eso dejó de ser un detalle: sin esta línea, volver a importar
	// el CSV de una entrada que se borró la daría por repetida y no volvería a
	// entrar, con la única copia escondida en la papelera y a punto de caducar.
	// Se veía como «la borré, la reimporté y no ha vuelto».
	iguales := map[string]bool{}
	cuentas := map[string]bool{}
	for _, e := range b.cont.Entradas {
		if e.Papelera {
			continue
		}
		iguales[huellaDeContenido(e)] = true
		cuentas[huellaDeCuenta(e)] = true
	}

	for _, e := range entradas {
		// **Idéntica: no se mete, y esto es lo que hace que reimportar el mismo
		// fichero no cambie nada.** Antes se metía marcada, así que pasar dos veces
		// el CSV dejaba la bóveda con el doble de entradas y sesenta y cinco copias
		// que había que borrar a mano. Lo contó el cliente después de hacerlo.
		if iguales[huellaDeContenido(e)] {
			r.Repetidas++
			continue
		}
		if e.Carpeta == "" {
			e.Carpeta = carpeta
		}
		// Misma cuenta y distinta contraseña **sí** entra, y marcada. Aquí sigue
		// valiendo el argumento de siempre: dos contraseñas distintas para la misma
		// cuenta significan que una de las dos está mal, y adivinar cuál no es cosa
		// de un importador.
		if cuentas[huellaDeCuenta(e)] {
			r.Conflictos++
			e.Etiquetas = append(e.Etiquetas, "duplicada")
		}

		id, err := azarHex()
		if err != nil {
			return r, err
		}
		e.ID = id
		e.Creada, e.Cambiada = ahora, ahora
		b.cont.Entradas = append(b.cont.Entradas, e)
		iguales[huellaDeContenido(e)] = true
		cuentas[huellaDeCuenta(e)] = true
		r.Metidas++
	}

	if r.Metidas == 0 {
		return r, nil // no hay nada que guardar, y guardar de más es reescribir la bóveda
	}
	b.cuerpoSucio = true
	return r, b.guardar()
}

// Resumen es lo que se cuenta después de importar.
type Resumen struct {
	// Metidas son las que han entrado.
	Metidas int
	// Repetidas son las que ya estaban **exactamente igual** y no se han metido.
	Repetidas int
	// Conflictos son las de una cuenta que ya estaba **con otra contraseña**: esas
	// sí entran, marcadas, porque una de las dos está mal y no es el importador
	// quien decide cuál.
	Conflictos int
}

// huellaDeContenido dice si dos entradas son la misma cosa, no solo la misma
// cuenta.
//
// Se comparan **los campos que vienen del fichero**, nunca el identificador, las
// fechas ni la carpeta: esos los pone el importador, y con ellos dentro dos
// pasadas del mismo CSV no se parecerían en nada.
func huellaDeContenido(e Entrada) string {
	return strings.Join([]string{
		e.Titulo, e.Usuario, e.Secreto, e.Notas, e.TOTP,
		strings.Join(e.Sitios, "\x1f"),
		e.Titular, soloCifras(e.Numero), e.Caduca, e.Verificacion,
		e.NombreCompleto, e.Documento, e.NumeroDocumento,
	}, "\x00")
}

// huellaDeCuenta es lo que hace que dos entradas sean «la misma cuenta».
//
// **Cada clase de entrada se identifica por lo suyo**, y eso no es un refinamiento
// sino lo que impide un desastre callado: con la huella de una credencial —sitio
// más usuario— todas las tarjetas del mundo tienen la misma, porque ninguna tiene
// sitio ni usuario. Importar cinco tarjetas marcaba cuatro como duplicadas. Lo
// mismo pasaba con las notas seguras.
func huellaDeCuenta(e Entrada) string {
	switch {
	case e.Numero != "":
		return "tarjeta\x00" + soloCifras(e.Numero)
	case e.NumeroDocumento != "":
		return "documento\x00" + strings.ToLower(e.NumeroDocumento)
	}

	sitio := ""
	if len(e.Sitios) > 0 {
		sitio = strings.ToLower(e.Sitios[0])
	}
	huella := sitio + "\x00" + strings.ToLower(e.Usuario)
	if huella == "\x00" {
		// Sin sitio ni usuario —una nota— lo único que distingue una de otra es su
		// título. Peor huella, pero infinitamente mejor que una común a todas.
		return "titulo\x00" + strings.ToLower(e.Titulo)
	}
	return huella
}

// soloCifras compara los números de tarjeta sin importar cómo estén escritos:
// «4111 1111 1111 1111» y «4111111111111111» son la misma tarjeta.
func soloCifras(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// Exportar escribe las entradas en CSV, **en claro**.
//
// Existe porque una bóveda de la que no se puede salir es una trampa, y porque
// es lo que permite probar esto sin apostar nada. Quien llame a esto tiene que
// haber avisado antes, en grande: lo que sale por aquí no está cifrado.
func (b *Boveda) Exportar(w io.Writer) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return ErrCerrada
	}
	c := csv.NewWriter(w)
	// Una sola tabla con todas las columnas, no una por clase de entrada. Es
	// justo lo contrario de lo que hace Dashlane, y a propósito: lo que sale de
	// aquí tiene que poder volver a entrar de una vez, y un fichero con columnas
	// vacías se lee mejor que cuatro ficheros que hay que traer uno a uno.
	if err := c.Write([]string{
		"title", "url", "username", "password", "otpSecret", "note", "folder",
		"type", "cardholder", "cc_number", "expiration_date", "cvv",
		"full_name", "document_type", "document_number",
	}); err != nil {
		return err
	}
	for _, e := range b.cont.Entradas {
		if e.Papelera {
			continue
		}
		sitio := ""
		if len(e.Sitios) > 0 {
			sitio = e.Sitios[0]
		}
		if err := c.Write([]string{
			e.Titulo, sitio, e.Usuario, e.Secreto, e.TOTP, e.Notas, e.Carpeta,
			string(e.Tipo), e.Titular, e.Numero, e.Caduca, e.Verificacion,
			e.NombreCompleto, e.Documento, e.NumeroDocumento,
		}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}
