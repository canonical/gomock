// Package source makes sure output imports its. See #505.
package source

//go:generate mockgen -package source -destination=../output/source_mock.go go.uber.org/mock/mockgen/internal/tests/missing_import/source Bar

type Foo struct{}

type Bar interface {
	Baz(Foo)
}
