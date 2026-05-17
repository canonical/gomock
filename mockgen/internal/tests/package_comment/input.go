package empty_interface

//go:generate mockgen -package empty_interface -destination mock.go -write_package_comment=false go.uber.org/mock/mockgen/internal/tests/package_comment Empty

type Empty interface{}
