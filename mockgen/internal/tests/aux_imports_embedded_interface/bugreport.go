package bugreport

// Reproduce the issue described in the README.md with
//go:generate mockgen -destination bugreport_mock.go -package bugreport github.com/canonical/gomock/mockgen/internal/tests/aux_imports_embedded_interface Source

import (
	"log"

	"github.com/canonical/gomock/mockgen/internal/tests/aux_imports_embedded_interface/faux"
)

// Source is an interface w/ an embedded foreign interface
type Source interface {
	faux.Foreign
}

func CallForeignMethod(s Source) {
	log.Println(s.Method())
}
