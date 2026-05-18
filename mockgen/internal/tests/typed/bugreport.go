package typed

//go:generate mockgen -destination bugreport_mock.go -package typed github.com/canonical/gomock/mockgen/internal/tests/typed Source

import (
	"log"

	"github.com/canonical/gomock/mockgen/internal/tests/typed/faux"
)

// Source is an interface w/ an embedded foreign interface
type Source interface {
	faux.Foreign
}

func CallForeignMethod(s Source) {
	log.Println(s.Method())
}
