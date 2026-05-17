package const_length

import "math"

//go:generate mockgen -package const_length -destination mock.go github.com/canonical/gomock/mockgen/internal/tests/const_array_length I

const C = 2

type I interface {
	Foo() [C]int
	Bar() [2]int
	Baz() [math.MaxInt8]int
	Qux() [1 + 2]int
	Quux() [(1 + 2)]int
	Corge() [math.MaxInt8 - 120]int
}
