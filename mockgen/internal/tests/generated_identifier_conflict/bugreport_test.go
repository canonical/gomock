package bugreport

import (
	"testing"

	"github.com/canonical/gomock/gomock"
)

func TestExample_Method(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockExample(ctrl)
	m.EXPECT().Method(1, 2, 3, 4)

	m.Method(1, 2, 3, 4)
}

func TestExample_VarargMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockExample(ctrl)
	m.EXPECT().VarargMethod(1, 2, 3, 4, 6, 7)

	m.VarargMethod(1, 2, 3, 4, 6, 7)
}

// TestExample_VarargMethod_NilExpectsNoVarargs verifies that passing nil
// for the variadic argument requires the call to have no variadic args.
func TestExample_VarargMethod_NilExpectsNoVarargs(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockExample(ctrl)
	m.EXPECT().VarargMethod(1, 2, 3, 4, nil)

	m.VarargMethod(1, 2, 3, 4)
}

// TestExample_VarargMethod_SingleMatcher verifies that a single Matcher is
// tried against the first vararg element; if it matches, the remainder is
// ignored. If it does not match, the whole tail is tried as a slice.
func TestExample_VarargMethod_SingleMatcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockExample(ctrl)
	m.EXPECT().VarargMethod(1, 2, 3, 4, gomock.Any())

	// Any() matches the first element (99); 100 and 101 are ignored.
	m.VarargMethod(1, 2, 3, 4, 99, 100, 101)
}

// TestExample_VarargMethod_BareValue verifies that a bare value is wrapped
// with Eq and matched against the first variadic element.
func TestExample_VarargMethod_BareValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := NewMockExample(ctrl)
	m.EXPECT().VarargMethod(1, 2, 3, 4, 42)

	m.VarargMethod(1, 2, 3, 4, 42)
}
