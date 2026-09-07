package cripto

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// TamSegmento es el trozo de texto claro que se sella de una vez. 64 KiB mantiene
// el consumo de memoria constante por grande que sea el fichero.
const TamSegmento = 64 * 1024

// nonceSegmento compone el nonce de un segmento: 19 bytes aleatorios de la
// cabecera, el número de segmento y una marca de final.
//
// Esa marca es lo que cierra el ataque de truncado. Sin ella, quien corte el
// fichero por la mitad obtiene un descifrado que va bien y termina sin quejarse,
// y el destinatario se queda con medio secreto creyendo que lo tiene entero.
func nonceSegmento(base [tamNonce]byte, n uint32, ultimo bool) []byte {
	nonce := make([]byte, tamNonce)
	copy(nonce, base[:19])
	binary.BigEndian.PutUint32(nonce[19:23], n)
	if ultimo {
		nonce[23] = 1
	}
	return nonce
}

// SellarFlujo cifra todo lo que salga de r y lo escribe en w como contenedor ESF1
// en modo segmentos.
func SellarFlujo(w io.Writer, r io.Reader, clave []byte, p Parametros) error {
	cab, err := nuevaCabecera(ModoFlujo, p)
	if err != nil {
		return err
	}

	k := derivar(clave, cab.Sal[:], p)
	defer Borrar(k)

	aead, err := chacha20poly1305.NewX(k)
	if err != nil {
		return err
	}

	cb := cab.bytes()
	if _, err := w.Write(cb); err != nil {
		return err
	}

	// Se lleva un segmento de ventaja para saber cuál es el último antes de
	// sellarlo: la marca de final tiene que ir dentro de la parte autenticada.
	pendiente := make([]byte, TamSegmento)
	siguiente := make([]byte, TamSegmento)
	nPend, err := leerSegmento(r, pendiente)
	if err != nil {
		return err
	}

	salida := make([]byte, 0, TamSegmento+tamEtiqueta)
	for i := uint32(0); ; i++ {
		nSig, err := leerSegmento(r, siguiente)
		if err != nil {
			return err
		}
		ultimo := nSig == 0 // no queda nada por leer detrás

		salida = aead.Seal(salida[:0], nonceSegmento(cab.Nonce, i, ultimo), pendiente[:nPend], cb)
		if _, err := w.Write(salida); err != nil {
			return err
		}
		if ultimo {
			return nil
		}
		if i == ^uint32(0) {
			return errors.New("El contenido excede el tamaño máximo del contenedor")
		}
		pendiente, siguiente = siguiente, pendiente
		nPend = nSig
	}
}

// leerSegmento llena buf hasta donde llegue el lector. Devuelve cuántos bytes ha
// leído; 0 significa que no queda nada.
func leerSegmento(r io.Reader, buf []byte) (int, error) {
	n, err := io.ReadFull(r, buf)
	switch {
	case err == nil, errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return n, nil
	default:
		return n, err
	}
}

// AbrirFlujo descifra un contenedor ESF1 en modo segmentos de r hacia w.
//
// No escribe ni un byte en w hasta que el segmento correspondiente ha pasado la
// comprobación de autenticidad, así que un fichero manipulado no llega a salir a
// disco a medias.
func AbrirFlujo(w io.Writer, r io.Reader, clave []byte) error {
	cb := make([]byte, tamCabecera)
	if _, err := io.ReadFull(r, cb); err != nil {
		return ErrFormato
	}
	cab, err := leerCabecera(cb)
	if err != nil {
		return err
	}
	if cab.Modo != ModoFlujo {
		return errors.New("Este contenedor no es un flujo por segmentos: usa el modo texto")
	}

	k := derivar(clave, cab.Sal[:], cab.Par)
	defer Borrar(k)

	aead, err := chacha20poly1305.NewX(k)
	if err != nil {
		return err
	}

	cifrado := make([]byte, TamSegmento+tamEtiqueta)
	claro := make([]byte, 0, TamSegmento)
	for i := uint32(0); ; i++ {
		n, err := leerSegmento(r, cifrado)
		if err != nil {
			return err
		}
		if n == 0 {
			// Se acabaron los bytes sin haber visto nunca la marca de final.
			return ErrTruncado
		}
		if n < tamEtiqueta {
			return ErrFormato
		}

		claro, ultimo, err := abrirSegmento(aead, claro, cifrado[:n], cab.Nonce, i, cb)
		if err != nil {
			// Si el primer segmento abrió, la clave es buena y lo que falla es el
			// contenido. Decir «la clave no es correcta» ante un fichero que llegó
			// cortado manda a quien lo recibe a comprobar la clave durante media
			// hora, que es exactamente el rato que se pierde por un mensaje mal
			// elegido.
			if i > 0 {
				return ErrDanado
			}
			return err
		}
		if _, err := w.Write(claro); err != nil {
			return err
		}

		if ultimo {
			// Después del segmento final no puede quedar nada.
			if resto, _ := leerSegmento(r, cifrado); resto != 0 {
				return ErrFormato
			}
			return nil
		}
	}
}

// abrirSegmento intenta el segmento como intermedio y, si no cuadra, como final,
// y dice cuál de los dos era. Solo uno de los dos nonces puede validar la
// etiqueta, así que el resultado es inequívoco y no hay nada que un atacante
// pueda elegir aquí.
func abrirSegmento(aead cipher.AEAD, dst, cifrado []byte, base [tamNonce]byte, i uint32, aad []byte) ([]byte, bool, error) {
	if out, err := aead.Open(dst[:0], nonceSegmento(base, i, false), cifrado, aad); err == nil {
		return out, false, nil
	}
	out, err := aead.Open(dst[:0], nonceSegmento(base, i, true), cifrado, aad)
	if err != nil {
		return nil, false, ErrClaveIncorrecta
	}
	return out, true, nil
}
