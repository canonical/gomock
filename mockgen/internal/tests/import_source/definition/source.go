package source

//go:generate mockgen -destination ../source_mock.go go.uber.org/mock/mockgen/internal/tests/import_source/definition S
//go:generate mockgen -package source -destination source_mock.go go.uber.org/mock/mockgen/internal/tests/import_source/definition S

type X struct{}

type S interface {
	F(X)
}
