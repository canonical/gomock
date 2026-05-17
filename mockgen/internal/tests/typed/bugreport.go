package typed

//go:generate mockgen -destination bugreport_mock.go -package typed go.uber.org/mock/mockgen/internal/tests/typed Source

import (
	"log"

	"go.uber.org/mock/mockgen/internal/tests/typed/faux"
)

// Source is an interface w/ an embedded foreign interface
type Source interface {
	faux.Foreign
}

func CallForeignMethod(s Source) {
	log.Println(s.Method())
}
