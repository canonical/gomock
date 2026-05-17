package source

//go:generate mockgen -destination ../source_mock.go github.com/canonical/gomock/mockgen/internal/tests/import_source/definition S
//go:generate mockgen -package source -destination source_mock.go github.com/canonical/gomock/mockgen/internal/tests/import_source/definition S

type X struct{}

type S interface {
	F(X)
}
