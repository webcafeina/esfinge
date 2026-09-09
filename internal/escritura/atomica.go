// Package escritura deja ficheros en disco de forma que una interrupción no
// pueda dejar a medias lo que había.
//
// La receta —temporal en el mismo directorio, permisos antes de escribir un
// byte, y renombrar al final— vivía dentro de `conSalida`, en la línea de
// comandos, enredada con sus estilos y sus opciones. Se saca aquí porque la
// bóveda de contraseñas necesita exactamente lo mismo y con más motivo: en un
// `.esf` suelto una escritura a medias es un susto, y en la bóveda es todo.
//
// **Y de paso gana tres cosas que allí no tenía**, las tres aprendidas de cómo
// se pierde un fichero de verdad:
//
//  1. `Sync` antes de cerrar. Sin él, el renombrado puede llegar al disco antes
//     que los datos, y tras un corte de luz queda un fichero de cero bytes con
//     el nombre bueno. Es el peor resultado posible: parece que está.
//  2. `Sync` del directorio después de renombrar, en los sistemas donde eso
//     significa algo. Sin él, el propio renombrado puede perderse.
//  3. Una generación anterior, opcional. Es el seguro contra un fallo de nuestro
//     propio serializador, que ninguna atomicidad cubre: si lo que escribimos
//     está mal, se escribe mal de forma perfectamente atómica.
package escritura

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Prefijo de los temporales. Va a la vista para que se puedan reconocer y
// limpiar: un fichero suelto que empieza por punto en la carpeta de trabajo de
// alguien, sin nombre que lo explique, es basura que nadie se atreve a borrar.
const Prefijo = ".esfinge-"

// Opciones afina lo que hace Atomica. El cero vale para el caso corriente.
type Opciones struct {
	// Permisos del fichero. Si es cero, 0600: lo que escribe Esfinge es siempre
	// un secreto o está a un paso de serlo.
	Permisos os.FileMode

	// Anterior guarda una copia de lo que había con este sufijo antes de pisarlo.
	// Vacío, no se guarda nada.
	Anterior string

	// CrearCarpeta crea el directorio de destino si falta, con permisos 0700.
	//
	// Va apagado por defecto **a propósito**: en la línea de comandos, un
	// «-o carpeta/que/no/existe/fichero.esf» tiene que fallar y decirlo, no
	// inventarse tres directorios en silencio. Lo enciende quien escribe en un
	// sitio suyo, como la bóveda en la carpeta de configuración.
	CrearCarpeta bool
}

// Atomica escribe en destino lo que ponga la función, de una sola pieza.
//
// O queda el fichero nuevo entero, o queda el viejo entero. Nunca uno a medias
// con el nombre del bueno.
func Atomica(destino string, opt Opciones, escribir func(io.Writer) error) error {
	if opt.Permisos == 0 {
		opt.Permisos = 0o600
	}

	dir := filepath.Dir(destino)
	if opt.CrearCarpeta {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("No puedo preparar %s: %w", dir, err)
		}
	}

	// La copia de la generación anterior va **antes** de tocar nada, para que el
	// seguro exista incluso si lo que viene después falla.
	if opt.Anterior != "" {
		if err := copiar(destino, destino+opt.Anterior, opt.Permisos); err != nil {
			return err
		}
	}

	tmp, err := os.CreateTemp(dir, Prefijo+"*")
	if err != nil {
		return fmt.Errorf("No puedo escribir en %s: %w", dir, err)
	}
	nombre := tmp.Name()
	// No hace nada si el renombrado funcionó; lo limpia todo si no.
	defer os.Remove(nombre)

	// Los permisos, antes de escribir un solo byte: entre crear el fichero y
	// ajustarlos hay una ventana en la que un secreto sería legible por cualquiera
	// de la máquina.
	if err := tmp.Chmod(opt.Permisos); err != nil {
		tmp.Close()
		return err
	}

	w := bufio.NewWriter(tmp)
	if err := escribir(w); err != nil {
		tmp.Close()
		return err
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		return err
	}
	// **Aquí está la diferencia con lo que había.** Sin este Sync, el renombrado
	// puede llegar al disco antes que los datos.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := renombrar(nombre, destino); err != nil {
		return fmt.Errorf("No puedo dejar el resultado en %s: %w", destino, err)
	}
	sincronizarDirectorio(dir)
	return nil
}

// renombrar reintenta una vez en Windows.
//
// Allí `os.Rename` sobre un fichero que existe funciona, pero **falla si otro
// proceso lo tiene abierto**, y un antivirus escaneando lo que se acaba de
// escribir cuenta como otro proceso. Un respiro y otro intento resuelve el caso
// real sin esconder un fallo de verdad.
func renombrar(de, a string) error {
	err := os.Rename(de, a)
	if err == nil || runtime.GOOS != "windows" {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return os.Rename(de, a)
}

// sincronizarDirectorio asienta el propio renombrado.
//
// En Windows no se puede abrir un directorio como fichero y esto no significa
// nada, así que se salta. El error se ignora a propósito en todas partes: si
// falla, el fichero ya está escrito y renombrado, y no hay nada mejor que hacer
// que seguir.
func sincronizarDirectorio(dir string) {
	if runtime.GOOS == "windows" {
		return
	}
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	defer d.Close()
	_ = d.Sync()
}

func copiar(de, a string, permisos os.FileMode) error {
	origen, err := os.Open(de)
	if err != nil {
		// Que no haya nada que copiar es lo normal la primera vez.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer origen.Close()

	destino, err := os.OpenFile(a, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permisos)
	if err != nil {
		return err
	}
	defer destino.Close()

	if _, err := io.Copy(destino, origen); err != nil {
		return err
	}
	return destino.Sync()
}

// LimpiarHuerfanos borra los temporales que quedaron de una interrupción.
//
// Solo los del prefijo propio y **solo los que llevan un rato**: uno recién
// creado puede ser de otro Esfinge que esté escribiendo ahora mismo, y borrarlo
// sería provocar justo el fallo que este paquete existe para evitar.
func LimpiarHuerfanos(dir string, edad time.Duration) {
	entradas, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	corte := time.Now().Add(-edad)
	for _, e := range entradas {
		if e.IsDir() || len(e.Name()) < len(Prefijo) || e.Name()[:len(Prefijo)] != Prefijo {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().After(corte) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
	}
}
