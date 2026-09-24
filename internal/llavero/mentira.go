package llavero

import "sync"

// DeMentira es un llavero en memoria, para las pruebas y para el servidor de
// desarrollo. **No se compila fuera con ninguna etiqueta**: es un tipo más, y lo
// que lo mantiene fuera del binario del cliente es que nadie lo construya ahí.
//
// Sirve para lo que aquí no se puede: en esta máquina no hay Touch ID ni Hello,
// así que sin esto la fase C no tendría ninguna prueba que no fuera mirar el
// código.
type DeMentira struct {
	mu       sync.Mutex
	guardado map[string][]byte
	// ComoSeLlama es lo que devuelve Nombre. Por defecto, «el sistema».
	ComoSeLlama string
	// NoHay simula un equipo sin biometría configurada.
	NoHay bool
	// DiceQueNo simula que la persona cancela el diálogo o que no la reconoce.
	DiceQueNo bool
	// Lecturas cuenta cuántas veces se ha pedido el secreto: es lo que distingue
	// «abrió con el sistema» de «abrió con la maestra».
	Lecturas int
}

func (l *DeMentira) Nombre() string {
	if l.ComoSeLlama == "" {
		return "el sistema"
	}
	return l.ComoSeLlama
}

func (l *DeMentira) Hay() bool { return !l.NoHay }

func (l *DeMentira) Guardar(id string, secreto []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.NoHay {
		return ErrNoHay
	}
	if l.guardado == nil {
		l.guardado = map[string][]byte{}
	}
	l.guardado[id] = append([]byte(nil), secreto...)
	return nil
}

func (l *DeMentira) Leer(id, _ string) ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Lecturas++
	if l.NoHay {
		return nil, ErrNoHay
	}
	if l.DiceQueNo {
		return nil, ErrNoQuiso
	}
	s, hay := l.guardado[id]
	if !hay {
		return nil, ErrNoEsta
	}
	return append([]byte(nil), s...), nil
}

func (l *DeMentira) Borrar(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.guardado, id)
	return nil
}
