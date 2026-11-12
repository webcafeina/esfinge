//go:build !unix

package app

import "os"

// En Windows no hay inodo que mirar, y la prueba que lo usa se salta ella sola. Esta pareja
// existe porque una etiqueta //go:build que excluye un sistema **es una promesa**, y aquí ya
// costó dos objetivos de Windows quedarse sin su función sin que nada lo dijera.
func inodo(os.FileInfo) (uint64, bool) { return 0, false }
