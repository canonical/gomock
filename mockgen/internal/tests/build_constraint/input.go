package empty_interface

//go:generate mockgen -package empty_interface -destination mock.go "-build_constraint=(linux && 386) || (darwin && !cgo) || usertag" github.com/canonical/gomock/mockgen/internal/tests/build_constraint Empty

type Empty interface{}
