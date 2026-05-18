package empty_interface

//go:generate mockgen -package empty_interface -destination mock.go github.com/canonical/gomock/mockgen/internal/tests/empty_interface Empty

type Empty interface{} // migrating interface{} -> any does not resolve to an interface type.
