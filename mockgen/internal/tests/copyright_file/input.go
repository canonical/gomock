package empty_interface

//go:generate mockgen -package empty_interface -destination mock.go -copyright_file=mock_copyright_header go.uber.org/mock/mockgen/internal/tests/copyright_file Empty

type Empty interface{} // migrating interface{} -> any does not resolve to an interface type dropping test generation added in b391ab3
