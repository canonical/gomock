package exclude

//go:generate mockgen -destination=mock.go -package=exclude -exclude_interfaces=IgnoreMe,IgnoreMe2 go.uber.org/mock/mockgen/internal/tests/exclude GenerateMockForMe

type IgnoreMe interface {
	A() bool
}

type IgnoreMe2 interface {
	~int
}

type GenerateMockForMe interface {
	B() int
}
