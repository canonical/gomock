package empty_interface

//go:generate mockgen -package empty_interface -destination mock.go -copyright_file=mock_copyright_header github.com/canonical/gomock/mockgen/internal/tests/copyright_file Empty

type Empty any // migrating interface{} -> any does not resolve to an interface type dropping test generation added in b391ab3
