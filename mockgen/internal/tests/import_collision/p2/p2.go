package p2

//go:generate mockgen -destination=mocks/mocks.go -package=internalpackage . Mything

import (
	"github.com/canonical/gomock/mockgen/internal/tests/import_collision/internalpackage"
)

type Mything interface {
	// issue here, is that the variable has the same name as an imported package.
	DoThat(internalpackage int) internalpackage.FooExported
}
