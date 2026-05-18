package import_aliased

import (
	definitionAlias "context"
)

//go:generate mockgen -package import_aliased -destination source_mock.go github.com/canonical/gomock/mockgen/internal/tests/import_aliased S

type S interface {
	M(ctx definitionAlias.Context)
}
