// Package source makes sure output imports its. See #505.
package source

//go:generate mockgen -package source -destination=../output/source_mock.go github.com/canonical/gomock/mockgen/internal/tests/missing_import/source Bar

type Foo struct{}

type Bar interface {
	Baz(Foo)
}
