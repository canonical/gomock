package overlap

//go:generate mockgen -package overlap -destination mock.go github.com/canonical/gomock/mockgen/internal/tests/overlapping_methods ReadWriteCloser

type ReadWriteCloser interface {
	ReadCloser
	WriteCloser
}
