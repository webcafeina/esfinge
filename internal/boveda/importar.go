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
	CampoIgnorar = ""
)

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
func Leer(datos []byte, mapa Correspondencia) ([]Entrada, Correspondencia, error) {
	texto := aTexto(datos)

	r := csv.NewReader(strings.NewReader(texto))
	r.Comma = separadorDe(texto)
	// Las dos banderas que hacen la diferencia entre importar un fichero real y
	// abortar a la primera: filas de longitud variable y comillas mal cerradas.
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	filas, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("No he podido leer el fichero: %w", err)
	}
	if len(filas) < 2 {
		return nil, nil, ErrSinColumnas
	}

	cabecera := filas[0]
	if mapa == nil {
		mapa = Adivinar(cabecera)
	}
	if !mapa.sirve() {
		return nil, mapa, ErrSinColumnas
	}

	var out []Entrada
	for _, fila := range filas[1:] {
		e := deFila(cabecera, fila, mapa)
		// Una fila sin nada que guardar no es una entrada, es una línea en blanco.
		if e.Titulo == "" && e.Usuario == "" && e.Secreto == "" && e.Notas == "" {
			continue
		}
		if e.Titulo == "" {
			e.Titulo = primerNoVacio(e.Sitios...)
		}
		if e.Titulo == "" {
			e.Titulo = e.Usuario
		}
		out = append(out, e)
	}
	if len(out) == 0 {
		return nil, mapa, errors.New("El fichero se lee bien pero no tiene ninguna entrada")
	}
	return out, mapa, nil
}

// Adivinar propone una correspondencia a partir de los nombres de las columnas.
func Adivinar(cabecera []string) Correspondencia {
	mapa := Correspondencia{}
	for _, c := range cabecera {
		clave := strings.ToLower(strings.TrimSpace(c))
		clave = strings.ReplaceAll(clave, " ", "")
		if campo, hay := alias[clave]; hay {
			// La primera columna que reclama un campo se lo queda: Dashlane exporta
			// «username», «username2» y «username3», y el bueno es el primero.
			if !mapa.tiene(campo) {
				mapa[c] = campo
			}
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

// sirve dice si con este mapa se puede sacar algo aprovechable. Sin contraseña
// ni notas no hay nada que guardar.
func (m Correspondencia) sirve() bool {
	return m.tiene(CampoSecreto) || m.tiene(CampoNotas)
}

func deFila(cabecera, fila []string, mapa Correspondencia) Entrada {
	var e Entrada
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
			e.Notas = valor
		case CampoTOTP:
			e.TOTP = valor
		case CampoCarpeta:
			e.Carpeta = valor
		}
	}
	e.Tipo = TipoCredencial
	if e.Secreto == "" && e.Usuario == "" && e.Notas != "" {
		e.Tipo = TipoNota
	}
	return e
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
func (b *Boveda) Importar(entradas []Entrada, deDonde string) (metidas, duplicadas int, err error) {
	if b.llave == nil {
		return 0, 0, ErrCerrada
	}
	carpeta := fmt.Sprintf("Importado de %s · %s", deDonde, time.Now().Format("2006-01-02"))
	ahora := time.Now().UTC().Format(time.RFC3339)

	conocidas := map[string]bool{}
	for _, e := range b.cont.Entradas {
		conocidas[huellaDeCuenta(e)] = true
	}

	for _, e := range entradas {
		if e.Carpeta == "" {
			e.Carpeta = carpeta
		}
		if conocidas[huellaDeCuenta(e)] {
			duplicadas++
			e.Etiquetas = append(e.Etiquetas, "duplicada")
		}
		id, err := azarHex()
		if err != nil {
			return metidas, duplicadas, err
		}
		e.ID = id
		e.Creada, e.Cambiada = ahora, ahora
		b.cont.Entradas = append(b.cont.Entradas, e)
		conocidas[huellaDeCuenta(e)] = true
		metidas++
	}

	b.cuerpoSucio = true
	return metidas, duplicadas, b.Guardar()
}

func huellaDeCuenta(e Entrada) string {
	sitio := ""
	if len(e.Sitios) > 0 {
		sitio = strings.ToLower(e.Sitios[0])
	}
	return sitio + "\x00" + strings.ToLower(e.Usuario)
}

// Exportar escribe las entradas en CSV, **en claro**.
//
// Existe porque una bóveda de la que no se puede salir es una trampa, y porque
// es lo que permite probar esto sin apostar nada. Quien llame a esto tiene que
// haber avisado antes, en grande: lo que sale por aquí no está cifrado.
func (b *Boveda) Exportar(w io.Writer) error {
	if b.llave == nil {
		return ErrCerrada
	}
	c := csv.NewWriter(w)
	if err := c.Write([]string{"title", "url", "username", "password", "otpSecret", "note", "folder"}); err != nil {
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
		if err := c.Write([]string{e.Titulo, sitio, e.Usuario, e.Secreto, e.TOTP, e.Notas, e.Carpeta}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}
