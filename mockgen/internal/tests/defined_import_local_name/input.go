package defined_import_local_name

import (
	"bytes"
	"context"
)

//go:generate mockgen -package defined_import_local_name -destination mock.go github.com/canonical/gomock/mockgen/internal/tests/defined_import_local_name WithImports

type WithImports interface {
	Method1() bytes.Buffer
	Method2() context.Context
}
