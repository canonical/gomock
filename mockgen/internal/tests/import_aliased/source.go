package import_aliased

import (
	definitionAlias "context"
)

//go:generate mockgen -package import_aliased -destination source_mock.go go.uber.org/mock/mockgen/internal/tests/import_aliased S

type S interface {
	M(ctx definitionAlias.Context)
}
