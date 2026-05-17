package gomock_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/canonical/gomock/gomock"
)

type Foo interface {
	Bar(string) string
	String() string
}

func ExampleCall_DoAndReturn_latency() {
	t := &testing.T{} // provided by test
	ctrl := gomock.NewController(t)
	mockIndex := NewMockFoo(ctrl)

	mockIndex.EXPECT().Bar(gomock.Any()).DoAndReturn(
		func(arg string) string {
			time.Sleep(1 * time.Millisecond)
			return "I'm sleepy"
		},
	)

	r := mockIndex.Bar("foo")
	fmt.Println(r)
	// Output: I'm sleepy
}

func ExampleCall_DoAndReturn_captureArguments() {
	t := &testing.T{} // provided by test
	ctrl := gomock.NewController(t)
	mockIndex := NewMockFoo(ctrl)
	var s string

	mockIndex.EXPECT().Bar(gomock.AssignableToTypeOf(s)).DoAndReturn(
		func(arg string) string {
			s = arg
			return "I'm sleepy"
		},
	)

	r := mockIndex.Bar("foo")
	fmt.Printf("%s %s", r, s)
	// Output: I'm sleepy foo
}

func ExampleCall_DoAndReturn_withOverridableExpectations() {
	t := &testing.T{} // provided by test
	ctrl := gomock.NewController(t, gomock.WithOverridableExpectations())
	mockIndex := NewMockFoo(ctrl)
	var s string

	mockIndex.EXPECT().Bar(gomock.AssignableToTypeOf(s)).DoAndReturn(
		func(arg string) string {
			s = arg
			return "I'm sleepy"
		},
	)

	r := mockIndex.Bar("foo")
	fmt.Printf("%s %s", r, s)
	// Output: I'm sleepy foo
}
