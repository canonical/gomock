package overlap

//go:generate mockgen -package overlap -destination mock.go go.uber.org/mock/mockgen/internal/tests/overlapping_methods ReadWriteCloser

type ReadWriteCloser interface {
	ReadCloser
	WriteCloser
}
