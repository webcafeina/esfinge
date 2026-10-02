package boveda

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/webcafeina/esfinge/internal/cripto"
	"github.com/webcafeina/esfinge/internal/wifi"
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

	// La red wifi (ADR 0049).
	CampoSSID      = "ssid"
	CampoSeguridad = "seguridad"
	CampoOculta    = "oculta"

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

	// Datos personales (ADR 0047). El nombre sí se compone —tres columnas en un
	// campo— y **la dirección no**: se guarda en sus nueve trozos, que es como la
	// da el gestor y como la pide un formulario. Componerla para leerla es de una
	// línea (`Entrada.Direccion`); volver a partirla sería adivinar.
	CampoCorreo       = "correo"
	CampoTelefono     = "telefono"
	CampoNacimiento   = "nacimiento"
	CampoNombrePila   = "nombre-pila"
	CampoNombreMedio  = "nombre-medio"
	CampoApellidos    = "apellidos"
	CampoCalle        = "calle"
	CampoCodigoPostal = "codigo-postal"
	CampoCiudad       = "ciudad"
	CampoProvincia    = "provincia"
	CampoPais         = "pais"
	CampoDestinatario = "destinatario"
	CampoEdificio     = "edificio"
	CampoPiso         = "piso"
	CampoPuerta       = "puerta"

	// CampoTipo solo se lee en el fichero que exporta Esfinge. Ver `tipoDe`.
	CampoTipo = "tipo"

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
	// FormaPersonal es `personalinfo.csv`, y es el **único** de Dashlane que se
	// declara fila a fila: trae una columna `type` que dice si esa línea es un
	// nombre, un correo, un teléfono o una dirección. No hace falta creerle
	// —dos clases cualesquiera no comparten ni una columna rellena—, pero sí hay
	// que saber que las 24 columnas son de seis cosas distintas y que **casi
	// todas vienen vacías en cada fila**.
	FormaPersonal
	// FormaWifi es `wifi.csv`, el **sexto** fichero de Dashlane (ADR 0049).
	//
	// Es el más pequeño y el que más miente: su columna de seguridad dice
	// `unsecured` en redes que tienen contraseña, así que lo que trae ahí **no se
	// copia**, se normaliza mirando si hay clave. Lo demás es directo: `ssid`,
	// `passphrase`, `name`, `note` y `hidden`.
	FormaWifi
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
	// Los datos personales, por columnas que no existen en ningún otro fichero de
	// ningún gestor. No se mira `type` ni `item_name`: `type` está en tres de los
	// cinco de Dashlane queriendo decir cosas distintas, que es el problema que
	// `Forma` vino a resolver.
	case hay["email_type"] || hay["address_door_code"] || hay["place_of_birth"] || hay["address_recipient"]:
		return FormaPersonal
	// La red, por dos columnas que no existen en ningún otro fichero de ningún
	// gestor. Va antes que las tarjetas porque no comparte ninguna con ellas y
	// cuanto más arriba esté lo inconfundible, menos depende del orden.
	case hay["ssid"] || hay["passphrase"]:
		return FormaWifi
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
		"correo":          CampoCorreo, "telefono": CampoTelefono,
		"nacimiento": CampoNacimiento, "destinatario": CampoDestinatario,
		"direccion": CampoCalle, "edificio": CampoEdificio,
		"piso": CampoPiso, "puerta": CampoPuerta,
		"codigo_postal": CampoCodigoPostal, "ciudad": CampoCiudad,
		"provincia": CampoProvincia, "pais": CampoPais,

		// **«type» se lee aquí y solo aquí**, y es la excepción a la regla de que
		// la clase se deduce de los campos: éste es nuestro propio fichero y no
		// miente. Hace falta desde que hay datos personales, porque un dato
		// personal que solo lleva un nombre **no tiene ningún campo que lo
		// distinga de nada**: exportado y vuelto a importar salía convertido en
		// una credencial sin usuario ni contraseña. Se usa en último lugar, cuando
		// los campos no deciden (ver `tipoDe`).
		"type": CampoTipo,
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
	FormaWifi: {
		// `wifi.csv` de Dashlane. **El título sale del nombre que le pusiera quien lo
		// guardó**, y cuando viene vacío —que es lo que pasa en el fichero del
		// cliente— lo pone `tituloDeReserva` con el SSID.
		"ssid": CampoSSID, "passphrase": CampoSecreto,
		"name": CampoTitulo, "note": CampoNotas,
		"hidden": CampoOculta,
		// **Escrito como lo escribe Dashlane, sin la «y»**, y también bien por si
		// algún día lo arreglan. Lo que traiga no se cree: `tipoDe` lo normaliza.
		"encription_type": CampoSeguridad, "encryption_type": CampoSeguridad,
	},
	FormaPersonal: {
		// El título: Dashlane deja `title` vacío en todas las filas y pone el
		// rótulo que se inventa —«Correo electrónico 1»— en `item_name`.
		//
		// **`title` tiene que salir de `titulo` o `item_name` no entra nunca**:
		// `Adivinar` se queda con la primera columna que reclama un campo y `title`
		// va antes en la cabecera, así que la reclamaba ella y se quedaba vacía. Y
		// no se tira, se manda a las notas: si algún día Dashlane la rellena, lo
		// que traiga se conserva en vez de desaparecer sin que nadie lo note.
		"title": CampoNotas, "item_name": CampoTitulo,
		// El nombre, en tres trozos.
		"first_name": CampoNombrePila, "middle_name": CampoNombreMedio,
		"last_name": CampoApellidos,
		// **`login` aquí no es un usuario de ninguna cuenta**: es el alias que esa
		// persona suele usar. Dejándolo en `usuario` —que es lo que dice la tabla
		// común— la fila salía clasificada como credencial, con un usuario y sin
		// sitio ni contraseña.
		"login": CampoNotas,
		"email": CampoCorreo, "email_type": CampoNotas,
		"phone_number":  CampoTelefono,
		"date_of_birth": CampoNacimiento, "place_of_birth": CampoNotas,
		"job_title": CampoNotas,
		// La dirección, en trozos con su sitio en el sobre.
		"address": CampoCalle, "zip": CampoCodigoPostal, "city": CampoCiudad,
		"state": CampoProvincia, "country": CampoPais,
		"address_recipient": CampoDestinatario,
		// El piso y la puerta **no pueden caer en el mismo campo**: `Adivinar` se
		// queda con la primera columna que lo reclama y la otra se perdería sin
		// decir nada.
		"address_building": CampoEdificio, "address_floor": CampoPiso,
		"address_apartment": CampoPuerta,
		// **El código del portal es un secreto y va a las notas**, que es lo único
		// que `vaciarLoSensible` limpia de lo que se escribe suelto.
		"address_door_code": CampoNotas,
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
	// **`fields` son los campos propios de Bitwarden, y hasta hoy se tiraban.** Una
	// columna sin alias se descarta entera, y ahí es donde Bitwarden guarda lo que su
	// dueño añadió a mano: el nombre de una red, un PIN, un número de contrato. Se
	// perdían en silencio desde el primer importador. Van a las notas, que es donde van
	// todas las columnas que valen algo y no tienen sitio propio.
	"fields": CampoNotas,
	// La red wifi. **`ssid` y `passphrase` van en la tabla común** porque no significan
	// otra cosa en ningún fichero de ningún gestor, y porque la exportación de Esfinge
	// las trae y tiene que poder volver a entrar: sin esto, una red exportada vuelve
	// como credencial —la columna no la reclama nadie, el SSID se pierde y `tipoDe` ya
	// no tiene por dónde reconocerla—. Lo cazó la prueba de la vuelta entera.
	"ssid": CampoSSID, "passphrase": CampoSecreto,
	"seguridad": CampoSeguridad, "oculta": CampoOculta,
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
	forma := FormaDeLaCabecera(cabecera)
	for _, fila := range filas[1:] {
		e := deFila(cabecera, fila, mapa, forma)
		// Una fila sin nada que guardar no es una entrada, es una línea en blanco.
		if e.Titulo == "" && e.Usuario == "" && e.Secreto == "" && e.Notas == "" &&
			e.Numero == "" && e.NumeroDocumento == "" && e.SSID == "" &&
			e.Correo == "" && e.Telefono == "" && e.Nacimiento == "" && !e.TieneDireccion() {
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
// Hasta la 2.12.4 lo que contaba era que hubiera **algo que merezca la pena
// guardar bajo llave** —una contraseña o una nota—, y por eso `payments.csv` e
// `ids.csv` de Dashlane se rechazaban enteros. Con la ADR 0047 el listón se
// mueve otra vez, y conviene decirlo en voz alta en vez de añadir cuatro campos
// a una lista: **un correo y un teléfono no son secretos**, y desde que hay
// datos personales lo que decide no es si algo hay que esconderlo sino si la
// bóveda lo guarda.
func (m Correspondencia) sirve() bool {
	return m.tiene(CampoSecreto) || m.tiene(CampoNotas) ||
		m.tiene(CampoNumero) || m.tiene(CampoNumeroDocumento) ||
		m.tiene(CampoCorreo) || m.tiene(CampoTelefono) ||
		m.tiene(CampoCalle) || m.tiene(CampoNacimiento) ||
		// Y el nombre de una red: un fichero que solo traiga redes tiene que servir,
		// y su clave llega por `secreto` pero puede venir vacía en una red abierta.
		m.tiene(CampoSSID)
}

func deFila(cabecera, fila []string, mapa Correspondencia, forma Forma) Entrada {
	var e Entrada
	var notas []string
	var mes, ano string
	var pila, medio, apellidos string
	var tipoDicho string

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
		case CampoCorreo:
			e.Correo = valor
		case CampoTelefono:
			e.Telefono = valor
		case CampoNacimiento:
			e.Nacimiento = valor
		case CampoNombrePila:
			pila = valor
		case CampoNombreMedio:
			medio = valor
		case CampoApellidos:
			apellidos = valor
		case CampoCalle:
			e.Calle = valor
		case CampoCodigoPostal:
			e.CodigoPostal = valor
		case CampoCiudad:
			e.Ciudad = valor
		case CampoProvincia:
			e.Provincia = valor
		case CampoPais:
			e.Pais = valor
		case CampoDestinatario:
			e.Destinatario = valor
		case CampoEdificio:
			e.Edificio = valor
		case CampoPiso:
			e.Piso = valor
		case CampoPuerta:
			e.Puerta = valor
		case CampoSSID:
			e.SSID = valor
		case CampoSeguridad:
			e.Seguridad = valor
		case CampoOculta:
			// Lo que un CSV escribe como sí: `true` de Dashlane, y lo que pueda venir
			// de otro gestor o de nuestra propia exportación.
			v := strings.ToLower(strings.TrimSpace(valor))
			e.Oculta = v == "true" || v == "sí" || v == "si" || v == "1" || v == "yes"
		case CampoTipo:
			tipoDicho = valor
		}
	}

	// Dashlane parte la caducidad en dos columnas; aquí es un campo. Se juntan
	// como se escriben en la tarjeta.
	if e.Caduca == "" && (mes != "" || ano != "") {
		e.Caduca = strings.TrimPrefix(mes+"/"+ano, "/")
		e.Caduca = strings.TrimSuffix(e.Caduca, "/")
	}
	if e.NombreCompleto == "" {
		e.NombreCompleto = juntarCon(" ", pila, medio, apellidos)
	}
	e.Notas = strings.Join(notas, "\n")

	// **Y una red puede venir escondida dentro del texto** (ADR 0049), que es como la
	// guardan los gestores que no tienen una columna para ella.
	if e.SSID == "" {
		e.SSID, e.Secreto, e.Seguridad, e.Notas = redEnElTexto(e.Notas, e.Secreto, e.Seguridad)
	}

	e.Tipo = tipoDe(e, forma, tipoDicho)
	// **La seguridad de una red se normaliza, no se copia** (ADR 0049). El `wifi.csv`
	// de Dashlane dice `unsecured` en redes que tienen contraseña, y guardándolo tal
	// cual el código QR saldría marcado como red abierta y el móvil no se conectaría:
	// el fallo parecería del código y estaría en el dato. Es la misma regla que ya
	// manda en el tipo de entrada — lo que el fichero diga de sí mismo no decide.
	if e.Tipo == TipoWifi {
		e.Seguridad = wifi.Normalizar(e.Seguridad, e.Secreto != "")
	}
	if e.Titulo == "" {
		e.Titulo = tituloDeReserva(e)
	}
	return e
}

// juntarCon pega los trozos que no están vacíos. Existe porque
// `strings.Join` de tres cosas con dos vacías deja dos separadores seguidos.
func juntarCon(sep string, partes ...string) string {
	var hay []string
	for _, p := range partes {
		if p = strings.TrimSpace(p); p != "" {
			hay = append(hay, p)
		}
	}
	return strings.Join(hay, sep)
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
func tipoDe(e Entrada, forma Forma, dicho string) Tipo {
	switch {
	// La red, por el nombre que emite: ningún otro fichero de ningún gestor trae esa
	// columna, así que no se puede confundir con nada.
	case e.SSID != "":
		return TipoWifi
	case e.Numero != "" || e.Verificacion != "":
		return TipoTarjeta
	case e.NumeroDocumento != "" || e.Documento != "":
		return TipoIdentidad
	case e.Correo != "" || e.Telefono != "" || e.Nacimiento != "" || e.TieneDireccion():
		return TipoPersonal
	// **Un nombre a secas no tiene ningún campo que lo distinga**, y es media
	// exportación de datos personales: la fila `name` de Dashlane solo trae
	// `first_name` y `last_name`. Los dos únicos sitios donde se puede saber qué
	// es son el fichero del que salió y lo que Esfinge escribió de sí misma al
	// exportarlo. Fuera de ahí sigue mandando lo que dicen los campos.
	case e.NombreCompleto != "" && (forma == FormaPersonal || dicho == string(TipoPersonal)):
		return TipoPersonal
	case e.Secreto == "" && e.Usuario == "" && e.Notas != "":
		return TipoNota
	default:
		return TipoCredencial
	}
}

// redEnElTexto saca una red wifi escrita **dentro** del texto de una nota.
//
// **Dashlane tiene una columna para la red; los demás no.** LastPass guarda sus notas
// seguras —y las de tipo «Wi-Fi Password» son una de ellas— metiendo todos sus campos
// dentro de la columna de texto, una línea por campo; Bitwarden hace lo mismo con sus
// campos propios, en su columna `fields`. En los dos casos la red llega como texto
// corrido donde una línea dice `SSID:` y otra la contraseña.
//
// **Por eso la señal es la línea `SSID:` y no el gestor del que venga.** Reconocer «el
// formato de LastPass» sería escribir de memoria el formato interno de otro programa
// —sus nombres de campo, su idioma, su versión—; reconocer una línea que declara el
// nombre de una red es lo que esos formatos tienen en común y es lo único que aquí se
// puede sostener. Si mañana otro gestor escribe igual, entra solo.
//
// **Y no se pierde nada de lo que no entienda**: las líneas que reconoce las saca del
// texto y **todo lo demás se queda en las notas**, con su contenido intacto. Una nota
// que no sea una red no se toca, porque sin `SSID:` esto no hace nada.
//
// **Lo que no está comprobado, y hay que decirlo**: no se ha visto ni un fichero de
// verdad de LastPass ni de Bitwarden. Los ejemplos de las pruebas están escritos a
// mano con la forma que esos gestores documentan, así que lo que esto demuestra es que
// **la regla funciona sobre esa forma**, no que esa sea exactamente la forma que sale
// de sus exportaciones. El día que haya un fichero de verdad, se comprueba contra él.
func redEnElTexto(texto, secreto, seguridad string) (string, string, string, string) {
	if texto == "" || !strings.Contains(strings.ToLower(texto), "ssid") {
		return "", secreto, seguridad, texto
	}

	var ssid, clave, cifrado string
	var resto []string
	for _, linea := range strings.Split(texto, "\n") {
		nombre, valor, hay := strings.Cut(linea, ":")
		if !hay {
			resto = append(resto, linea)
			continue
		}
		// **Y la primera línea puede venir con el nombre de su columna delante.** Las
		// columnas que van a las notas se etiquetan con el suyo, así que lo que
		// Bitwarden manda en `fields` llega como «fields: SSID: MiCasa» y el nombre
		// que importa es el segundo. Se recorta **una sola vez**: con más, una frase
		// cualquiera con dos puntos de más acabaría pareciendo la declaración de una
		// red.
		if !campoDeRed(nombre) {
			if n2, v2, hay2 := strings.Cut(valor, ":"); hay2 && campoDeRed(n2) {
				nombre, valor = n2, v2
			}
		}
		valor = strings.TrimSpace(valor)
		// El nombre del campo, sin espacios ni mayúsculas: los gestores escriben
		// «SSID», «Wireless Network Password» o «Network Password» según el sitio.
		switch quitarEspacios(nombre) {
		case "ssid", "networkname", "nombredered":
			if ssid == "" {
				ssid = valor
			}
		case "password", "networkpassword", "wirelessnetworkpassword", "contraseña":
			if clave == "" {
				clave = valor
			}
		case "encryption", "security", "cifrado", "seguridad":
			if cifrado == "" {
				cifrado = valor
			}
		default:
			resto = append(resto, linea)
		}
	}
	if ssid == "" {
		// Había la palabra «ssid» por ahí pero ninguna línea que la declare: no es una
		// red, es una nota que habla de una. Se devuelve el texto **tal cual estaba**.
		return "", secreto, seguridad, texto
	}
	if secreto == "" {
		secreto = clave
	}
	if seguridad == "" {
		seguridad = cifrado
	}
	return ssid, secreto, seguridad, strings.TrimSpace(strings.Join(resto, "\n"))
}

// campoDeRed dice si el nombre de una línea declara algo de una red. Está aparte
// porque se pregunta dos veces: por la línea y por lo que queda tras recortar el
// nombre de la columna.
func campoDeRed(nombre string) bool {
	switch quitarEspacios(nombre) {
	case "ssid", "networkname", "nombredered",
		"password", "networkpassword", "wirelessnetworkpassword", "contraseña",
		"encryption", "security", "cifrado", "seguridad":
		return true
	}
	return false
}

func quitarEspacios(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), ""))
}

// tituloDeReserva: una entrada sin título es una entrada que no se encuentra.
func tituloDeReserva(e Entrada) string {
	if t := primerNoVacio(e.Sitios...); t != "" {
		return t
	}
	// El nombre antes que el usuario: en un dato personal es lo único que hay, y
	// `personalinfo.csv` **deja `title` vacío en todas las filas** —lo que se
	// parece a un título está en `item_name`, y solo en algunas clases—. Sin
	// esto, la fila del nombre entraba sin título, y una entrada sin título es
	// una entrada que no se encuentra.
	// Y el nombre de la red va el **primero** de todos: el `wifi.csv` del cliente trae
	// `name` vacío en todas sus filas, así que sin esto sus redes entrarían con el
	// título de otra cosa o sin ninguno.
	return primerNoVacio(e.SSID, e.NombreCompleto, e.Usuario, e.Correo, e.Telefono, e.Titular, e.Documento, e.Calle)
}

// siNo escribe un booleano como lo escribe un CSV: `true` o nada. Nada y no `false`
// porque una columna vacía es lo que cualquier gestor lee como un no, y porque el
// fichero se lee con los ojos.
func siNo(v bool) string {
	if v {
		return "true"
	}
	return ""
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
		e.Revision = 1
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
		e.Correo, e.Telefono, e.Nacimiento,
		e.Destinatario, e.Calle, e.Edificio, e.Piso, e.Puerta,
		e.CodigoPostal, e.Ciudad, e.Provincia, e.Pais,
		e.RPID, e.IDCredencial, e.IDUsuario, e.NombreVisible, e.ClavePrivada,
		e.SSID, e.Seguridad, fmt.Sprint(e.Oculta),
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
	// El dato personal se identifica por lo suyo, igual que los demás: un correo
	// y un teléfono son únicos por sí solos, y una dirección lo es de sobra. Sin
	// esto, las seis filas de `personalinfo.csv` no tienen ni sitio ni usuario y
	// **caían todas en la huella del título**, que es donde ya pasó lo de las
	// tarjetas. El nombre no entra aquí a propósito: se queda con la huella del
	// título, que para un nombre *es* el nombre.
	// La llave de acceso, por lo que la identifica de verdad: el identificador que
	// emitió el sitio. Va **antes que el correo** porque una llave puede llevar un
	// nombre visible que sea un correo, y entonces dos llaves distintas del mismo
	// sitio compartirían huella.
	// La red, por el nombre que emite y **nunca por su clave**: el cliente tiene dos
	// redes distintas con la misma contraseña, así que una huella que mirara el secreto
	// daría la segunda por repetida y no entraría.
	case e.SSID != "":
		return "wifi\x00" + strings.ToLower(strings.TrimSpace(e.SSID))
	case e.IDCredencial != "":
		return "llave\x00" + strings.ToLower(strings.TrimSpace(e.RPID)) + "\x00" + e.IDCredencial
	case e.Correo != "":
		return "correo\x00" + strings.ToLower(strings.TrimSpace(e.Correo))
	case e.Telefono != "":
		return "telefono\x00" + soloCifras(e.Telefono)
	case e.TieneDireccion():
		return "direccion\x00" + strings.ToLower(strings.Join(strings.Fields(e.Direccion()), " "))
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

// CuantasLlaves dice cuántas llaves de acceso vivas hay.
//
// Existe para poder **preguntar antes de preguntar**: sin esto, exportar abría el
// diálogo del sistema y descubría que no había nada al ir a escribir, así que
// pedía un sitio para un fichero que no iba a existir. Lo vio el cliente con la
// 2.32.0 recién instalada, que es justo el caso más probable: una bóveda sin
// llaves todavía.
func (b *Boveda) CuantasLlaves() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, e := range b.cont.Entradas {
		if !e.Papelera && e.Tipo == TipoLlave {
			n++
		}
	}
	return n
}

// ExportarLlaves saca las llaves de acceso **cifradas**, y nunca en claro.
//
// Es la excepción a «una bóveda de la que no se puede salir es una trampa», y
// está pensada para seguir cumpliéndola sin lo que costaría cumplirla del todo:
// una fila de CSV con una clave privada dentro es lo más peligroso que Esfinge
// escribiría nunca en el disco, y además **no le sirve a ningún gestor**, porque
// ninguno sabe leerla. Así que se puede salir, pero el fichero que sale está en
// un contenedor `ESF1` con su propia clave, que es la misma forma con la que
// Esfinge cifra cualquier cosa desde la 1.0.
//
// Lo de dentro es JSON y no CSV a propósito: esto no lo va a leer una hoja de
// cálculo, lo va a leer otra Esfinge.
func (b *Boveda) ExportarLlaves(w io.Writer, clave string) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.llave == nil {
		return 0, ErrCerrada
	}
	if clave == "" {
		return 0, errors.New("Hace falta una clave para cifrar el fichero")
	}

	var llaves []Entrada
	for _, e := range b.cont.Entradas {
		if e.Papelera || e.Tipo != TipoLlave {
			continue
		}
		llaves = append(llaves, e)
	}
	if len(llaves) == 0 {
		return 0, errors.New("No hay ninguna llave de acceso que exportar")
	}

	datos, err := json.Marshal(struct {
		Esfinge string    `json:"esfinge"`
		Version int       `json:"version"`
		Llaves  []Entrada `json:"llaves"`
	}{"llaves de acceso", 1, llaves})
	if err != nil {
		return 0, err
	}

	sellado, err := cripto.Sellar(datos, []byte(clave), cripto.PerfilInteractivo)
	if err != nil {
		return 0, err
	}
	if _, err := w.Write(sellado); err != nil {
		return 0, err
	}
	return len(llaves), nil
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
		"correo", "telefono", "nacimiento",
		"destinatario", "direccion", "edificio", "piso", "puerta",
		"codigo_postal", "ciudad", "provincia", "pais",
		"ssid", "seguridad", "oculta",
	}); err != nil {
		return err
	}
	for _, e := range b.cont.Entradas {
		if e.Papelera {
			continue
		}
		// **Las llaves de acceso no salen por aquí** (ADR 0048). Una fila de CSV con
		// una clave privada dentro es lo más peligroso que Esfinge escribiría nunca
		// en claro, y además no le sirve a ningún gestor: ninguno sabe leerla. Salen
		// por `ExportarLlaves`, en un contenedor ESF1 con su clave, y quien exporta
		// **lo ve dicho en la pantalla**, no lo descubre contando filas.
		if e.Tipo == TipoLlave {
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
			e.Correo, e.Telefono, e.Nacimiento,
			e.Destinatario, e.Calle, e.Edificio, e.Piso, e.Puerta,
			e.CodigoPostal, e.Ciudad, e.Provincia, e.Pais,
			// **La red sí sale en claro**, al contrario que la llave de acceso: su clave
			// es una contraseña como las demás, y poder llevársela a otro gestor es la
			// mitad de lo que significa poder salir. Lo que no sale es `oculta` cuando es
			// falso, que es lo que escribe un CSV vacío para un no.
			e.SSID, e.Seguridad, siNo(e.Oculta),
		}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}
