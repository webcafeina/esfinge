package actualizacion

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Avance es lo que se va sabiendo mientras baja el fichero.
type Avance struct {
	Bytes int64 `json:"bytes"`
	Total int64 `json:"total"`
	// Hecho pasa a cierto en el último aviso, cuando el fichero ya está en disco
	// y comprobado.
	Hecho bool `json:"hecho"`
}

// Descargar trae el fichero de la novedad, comprueba que llegó entero y devuelve
// dónde ha quedado.
//
// El resumen se calcula **mientras** se descarga, no releyendo el fichero
// después: así no se pasa dos veces por unos cuantos megas y, sobre todo, no hay
// un hueco entre comprobar y usar.
//
// Lo que esa comprobación protege es una descarga cortada o corrompida. **No**
// protege contra quien controle la publicación, porque el resumen sale del mismo
// sitio que el fichero. Lo que sostiene la confianza es el TLS contra GitHub.
func (c *Comprobador) Descargar(n Novedad, avisar func(Avance)) (string, error) {
	if n.URL == "" {
		return "", fmt.Errorf("Esta versión no trae fichero para este sistema")
	}

	carpeta, err := Carpeta()
	if err != nil {
		return "", err
	}
	// Lo anterior se tira: son megas de un fichero que ya no sirve.
	Limpiar(carpeta, n.Fichero)

	destino := filepath.Join(carpeta, n.Fichero)

	cliente := c.Cliente
	if cliente == nil {
		cliente = &http.Client{Timeout: 30 * time.Minute}
	}
	pet, err := http.NewRequest(http.MethodGet, n.URL, nil)
	if err != nil {
		return "", err
	}
	pet.Header.Set("User-Agent", "Esfinge/"+c.Version)

	resp, err := cliente.Do(pet)
	if err != nil {
		return "", fmt.Errorf("No se ha podido descargar: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("La descarga contestó %s", resp.Status)
	}

	total := n.Bytes
	if total == 0 {
		total = resp.ContentLength
	}

	// Se escribe en un fichero de paso y se renombra al final: si algo se corta a
	// medias, lo que queda no tiene el nombre del bueno y nadie lo instalará.
	parcial := destino + ".parcial"
	f, err := os.OpenFile(parcial, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}

	suma := sha256.New()
	cuenta := &contador{total: total, avisar: avisar}
	_, err = io.Copy(io.MultiWriter(f, suma, cuenta), resp.Body)
	cerrar := f.Close()
	if err == nil {
		err = cerrar
	}
	if err != nil {
		os.Remove(parcial)
		return "", fmt.Errorf("La descarga se ha cortado: %w", err)
	}

	if n.Resumen != "" {
		tengo := hex.EncodeToString(suma.Sum(nil))
		if !strings.EqualFold(tengo, n.Resumen) {
			os.Remove(parcial)
			return "", fmt.Errorf(
				"Lo descargado no coincide con lo publicado. Se ha borrado; vuelve a intentarlo")
		}
	}

	if err := os.Rename(parcial, destino); err != nil {
		os.Remove(parcial)
		return "", err
	}

	if avisar != nil {
		avisar(Avance{Bytes: total, Total: total, Hecho: true})
	}
	return destino, nil
}

// Carpeta es donde se dejan las descargas: la de caché del usuario, no la de
// Descargas. Es un fichero de paso, no algo que nadie quiera conservar.
func Carpeta() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	carpeta := filepath.Join(base, "Esfinge", "descargas")
	if err := os.MkdirAll(carpeta, 0o700); err != nil {
		return "", err
	}
	return carpeta, nil
}

// Limpiar tira todo lo que haya en la carpeta menos el fichero que se va a
// bajar ahora.
func Limpiar(carpeta, salvo string) {
	entradas, err := os.ReadDir(carpeta)
	if err != nil {
		return
	}
	for _, e := range entradas {
		if e.IsDir() || e.Name() == salvo {
			continue
		}
		os.Remove(filepath.Join(carpeta, e.Name()))
	}
}

// contador cuenta los bytes que pasan y va avisando.
//
// No avisa en cada escritura —serían cientos de eventos por segundo cruzando el
// puente— sino cuando ha pasado un trozo o ha cambiado el porcentaje.
type contador struct {
	bytes  int64
	total  int64
	ultimo time.Time
	avisar func(Avance)
}

func (c *contador) Write(p []byte) (int, error) {
	c.bytes += int64(len(p))
	if c.avisar == nil {
		return len(p), nil
	}
	if ahora := time.Now(); ahora.Sub(c.ultimo) >= 100*time.Millisecond {
		c.ultimo = ahora
		c.avisar(Avance{Bytes: c.bytes, Total: c.total})
	}
	return len(p), nil
}
