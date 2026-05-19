package empty_interface

//go:generate mockgen -package empty_interface -destination mock.go -write_package_comment=false github.com/canonical/gomock/mockgen/internal/tests/package_comment Empty

type Empty any
