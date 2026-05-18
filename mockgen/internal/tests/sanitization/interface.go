package sanitization

import (
	"github.com/canonical/gomock/mockgen/internal/tests/sanitization/any"
)

//go:generate mockgen -destination mockout/mock.go -package mockout . AnyMock

type AnyMock interface {
	Do(a *any.Any, b int)
}
