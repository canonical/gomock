// Copyright 2011 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package gomock_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/canonical/gomock/gomock"
)

type ErrorReporter struct {
	t          *testing.T
	log        []string
	failed     bool
	fatalToken struct{}
}

func NewErrorReporter(t *testing.T) *ErrorReporter {
	return &ErrorReporter{t: t}
}

func (e *ErrorReporter) reportLog() {
	for _, entry := range e.log {
		e.t.Log(entry)
	}
}

func (e *ErrorReporter) assertPass(msg string) {
	if e.failed {
		e.t.Errorf("Expected pass, but got failure(s): %s", msg)
		e.reportLog()
	}
}

func (e *ErrorReporter) assertFail(msg string) {
	if !e.failed {
		e.t.Errorf("Expected failure, but got pass: %s", msg)
	}
}

// assertFatal checks that fn triggers a fatal test failure.
func (e *ErrorReporter) assertFatal(fn func(), expectedErrMsgs ...string) {
	defer func() {
		err := recover()
		if err == nil {
			var actual string
			if e.failed {
				actual = "non-fatal failure"
			} else {
				actual = "pass"
			}
			e.t.Error("Expected fatal failure, but got a", actual)
		} else if token, ok := err.(*struct{}); ok && token == &e.fatalToken {
			if expectedErrMsgs != nil {
				actualErrMsg := e.log[len(e.log)-1]
				for _, expected := range expectedErrMsgs {
					if !strings.Contains(actualErrMsg, expected) {
						e.t.Errorf(
							"Error message:\ngot: %q\nwant to contain: %q\n",
							actualErrMsg, expected,
						)
					}
				}
			}
			return
		} else {
			panic(err)
		}
	}()
	fn()
}

// recoverUnexpectedFatal can be used as a deferred call in test cases
// to recover from and display a call to Fatalf().
func (e *ErrorReporter) recoverUnexpectedFatal() {
	err := recover()
	if err == nil {
		// No panic.
	} else if token, ok := err.(*struct{}); ok && token == &e.fatalToken {
		e.t.Error(
			"Got unexpected fatal error(s). All errors up to this point:",
		)
		e.reportLog()
		return
	} else {
		panic(err)
	}
}

func (e *ErrorReporter) Log(args ...any) {
	e.log = append(e.log, fmt.Sprint(args...))
}

func (e *ErrorReporter) Logf(format string, args ...any) {
	e.log = append(e.log, fmt.Sprintf(format, args...))
}

func (e *ErrorReporter) Errorf(format string, args ...any) {
	e.Logf(format, args...)
	e.failed = true
}

func (e *ErrorReporter) Fatalf(format string, args ...any) {
	e.Logf(format, args...)
	e.failed = true
	panic(&e.fatalToken)
}

func (e *ErrorReporter) Helper() {}

type HelperReporter struct {
	gomock.TestReporter
	helper int
}

func (h *HelperReporter) Helper() {
	h.helper++
}

// Subject is a type purely for use as a receiver in testing the Controller.
type Subject struct{}

func (s *Subject) FooMethod(arg string) int { return 0 }
func (s *Subject) BarMethod(arg string) int { return 0 }

// TestStruct is used by ActOnTestStructMethod.
type TestStruct struct {
	Number  int
	Message string
}

func (s *Subject) ActOnTestStructMethod(arg TestStruct, arg1 int) int { return 0 }

func assertEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %+v, but got %+v", expected, actual)
	}
}

func createFixtures(t *testing.T) (*ErrorReporter, *gomock.Controller) {
	reporter := NewErrorReporter(t)
	ctrl := gomock.NewController(reporter)
	return reporter, ctrl
}

// addExpect is a helper that appends a typed call to its expects slice
// and registers it with the controller for lifecycle tracking.
func addExpect[A1, R1 any](
	ctrl *gomock.Controller,
	expects *[]*gomock.Call1_1[A1, R1],
	call *gomock.Call1_1[A1, R1],
) {
	*expects = append(*expects, call)
	ctrl.Track(call.Call)
}

func TestNoCalls(t *testing.T) {
	reporter, _ := createFixtures(t)
	reporter.assertPass("No calls expected or made.")
}

func TestNoRecordedCallsForAReceiver(t *testing.T) {
	reporter, ctrl := createFixtures(t)
	subject := new(Subject)

	var expects []*gomock.Call1_1[string, int]
	reporter.assertFatal(func() {
		gomock.Dispatch1_1(&expects, ctrl, subject, "NotRecordedMethod", "argument")
	}, "Unexpected call to")
}

func TestExpectedMethodCall(t *testing.T) {
	reporter, ctrl := createFixtures(t)
	subject := new(Subject)

	var expects []*gomock.Call1_1[string, int]
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	))
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	reporter.assertPass("Expected method call made.")
}

func TestUnexpectedMethodCall(t *testing.T) {
	reporter, ctrl := createFixtures(t)
	subject := new(Subject)

	var expects []*gomock.Call1_1[string, int]
	reporter.assertFatal(func() {
		gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	})
}

func TestRepeatedCall(t *testing.T) {
	reporter, ctrl := createFixtures(t)
	subject := new(Subject)

	var expects []*gomock.Call1_1[string, int]
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).Times(3))
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	reporter.assertPass("After expected repeated method calls.")
	reporter.assertFatal(func() {
		gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	})
	reporter.assertFail("After calling one too many times.")
}

func TestAnyTimes(t *testing.T) {
	reporter, ctrl := createFixtures(t)
	subject := new(Subject)

	var expects []*gomock.Call1_1[string, int]
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).AnyTimes())
	for i := 0; i < 100; i++ {
		gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	}
	reporter.assertPass("After 100 method calls.")
}

func TestMinTimes1(t *testing.T) {
	// It fails if there are no calls.
	reporter, ctrl := createFixtures(t)
	subject := new(Subject)
	var expects []*gomock.Call1_1[string, int]
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).MinTimes(1))
	reporter.assertFatal(func() { ctrl.Finish() })

	// It succeeds if there is one call.
	_, ctrl = createFixtures(t)
	subject = new(Subject)
	expects = nil
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).MinTimes(1))
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	ctrl.Finish()

	// It succeeds if there are many calls.
	_, ctrl = createFixtures(t)
	subject = new(Subject)
	expects = nil
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).MinTimes(1))
	for i := 0; i < 100; i++ {
		gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	}
	ctrl.Finish()
}

func TestMaxTimes1(t *testing.T) {
	// It succeeds if there are no calls.
	_, ctrl := createFixtures(t)
	subject := new(Subject)
	var expects []*gomock.Call1_1[string, int]
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		t, subject, "FooMethod", gomock.Eq("argument"),
	).MaxTimes(1))
	ctrl.Finish()

	// It succeeds if there is one call.
	_, ctrl = createFixtures(t)
	subject = new(Subject)
	expects = nil
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		t, subject, "FooMethod", gomock.Eq("argument"),
	).MaxTimes(1))
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	ctrl.Finish()

	// It fails if there are more calls than the maximum.
	reporter, ctrl := createFixtures(t)
	subject = new(Subject)
	expects = nil
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).MaxTimes(1))
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	reporter.assertFatal(func() {
		gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	})
	ctrl.Finish()
}

func TestMinMaxTimes(t *testing.T) {
	// It fails if there are fewer calls than specified.
	reporter, ctrl := createFixtures(t)
	subject := new(Subject)
	var expects []*gomock.Call1_1[string, int]
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).MinTimes(2).MaxTimes(2))
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	reporter.assertFatal(func() { ctrl.Finish() })

	// It fails if there are more calls than specified.
	reporter, ctrl = createFixtures(t)
	subject = new(Subject)
	expects = nil
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).MinTimes(2).MaxTimes(2))
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	reporter.assertFatal(func() {
		gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	})

	// It succeeds if there is exactly the right number of calls.
	_, ctrl = createFixtures(t)
	subject = new(Subject)
	expects = nil
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).MaxTimes(2).MinTimes(2))
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	ctrl.Finish()

	// If MaxTimes is called after MinTimes(1), MaxTimes takes precedence.
	reporter, ctrl = createFixtures(t)
	subject = new(Subject)
	expects = nil
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).MinTimes(1).MaxTimes(2))
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	reporter.assertFatal(func() {
		gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	})

	// If MinTimes is called after MaxTimes(1), MinTimes takes precedence.
	_, ctrl = createFixtures(t)
	subject = new(Subject)
	expects = nil
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	).MaxTimes(1).MinTimes(2))
	for i := 0; i < 100; i++ {
		gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
	}
	ctrl.Finish()
}

func TestReturn(t *testing.T) {
	_, ctrl := createFixtures(t)
	subject := new(Subject)

	// Unspecified return should produce zero result.
	var eZero []*gomock.Call1_1[string, int]
	addExpect(ctrl, &eZero, gomock.NewCall1_1[string, int](
		t, subject, "FooMethod", gomock.Eq("zero"),
	))

	// Explicit Return should produce the given value.
	var eFive []*gomock.Call1_1[string, int]
	fiveCall := gomock.NewCall1_1[string, int](t, subject, "FooMethod", gomock.Eq("five"))
	addExpect(ctrl, &eFive, fiveCall)
	fiveCall.Return(5)

	assertEqual(t, 0, gomock.Dispatch1_1(&eZero, ctrl, subject, "FooMethod", "zero"))
	assertEqual(t, 5, gomock.Dispatch1_1(&eFive, ctrl, subject, "FooMethod", "five"))
}

func TestUnorderedCalls(t *testing.T) {
	reporter, ctrl := createFixtures(t)
	defer reporter.recoverUnexpectedFatal()
	subjectOne := new(Subject)
	subjectTwo := new(Subject)

	var eFoo1, eBar2, eFoo3, eBar4 []*gomock.Call1_1[string, int]
	addExpect(ctrl, &eFoo1, gomock.NewCall1_1[string, int](reporter, subjectOne, "FooMethod", gomock.Eq("1")))
	addExpect(ctrl, &eBar2, gomock.NewCall1_1[string, int](reporter, subjectOne, "BarMethod", gomock.Eq("2")))
	addExpect(ctrl, &eFoo3, gomock.NewCall1_1[string, int](reporter, subjectTwo, "FooMethod", gomock.Eq("3")))
	addExpect(ctrl, &eBar4, gomock.NewCall1_1[string, int](reporter, subjectTwo, "BarMethod", gomock.Eq("4")))

	// Make the calls in a different order, which should be fine.
	gomock.Dispatch1_1(&eBar2, ctrl, subjectOne, "BarMethod", "2")
	gomock.Dispatch1_1(&eFoo3, ctrl, subjectTwo, "FooMethod", "3")
	gomock.Dispatch1_1(&eBar4, ctrl, subjectTwo, "BarMethod", "4")
	gomock.Dispatch1_1(&eFoo1, ctrl, subjectOne, "FooMethod", "1")

	reporter.assertPass("After making all calls in different order")
	ctrl.Finish()
	reporter.assertPass("After finish")
}

func TestOrderedCallsCorrect(t *testing.T) {
	reporter, ctrl := createFixtures(t)
	subjectOne := new(Subject)
	subjectTwo := new(Subject)

	call1 := gomock.NewCall1_1[string, int](reporter, subjectOne, "FooMethod", gomock.Any()).AnyTimes()
	call2 := gomock.NewCall1_1[string, int](reporter, subjectTwo, "FooMethod", gomock.Any())
	call3 := gomock.NewCall1_1[string, int](reporter, subjectTwo, "BarMethod", gomock.Any())
	gomock.InOrder(call1, call2, call3)

	var e1, e2, e3 []*gomock.Call1_1[string, int]
	addExpect(ctrl, &e1, call1)
	addExpect(ctrl, &e2, call2)
	addExpect(ctrl, &e3, call3)

	gomock.Dispatch1_1(&e1, ctrl, subjectOne, "FooMethod", "1")
	gomock.Dispatch1_1(&e2, ctrl, subjectTwo, "FooMethod", "2")
	gomock.Dispatch1_1(&e3, ctrl, subjectTwo, "BarMethod", "3")

	ctrl.Finish()
	reporter.assertPass("After finish")
}

func TestPanicOverridesExpectationChecks(t *testing.T) {
	ctrl := gomock.NewController(t)
	reporter := NewErrorReporter(t)

	reporter.assertFatal(func() {
		var expects []*gomock.Call1_1[string, int]
		addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
			t, new(Subject), "FooMethod", gomock.Any(),
		))
		defer ctrl.Finish()
		reporter.Fatalf("Intentional panic")
	})
}

func TestTimes0(t *testing.T) {
	reporter, ctrl := createFixtures(t)
	subject := new(Subject)

	var expects []*gomock.Call1_1[string, int]
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("arg"),
	).Times(0))
	reporter.assertFatal(func() {
		gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "arg")
	})
}

func TestNoHelper(t *testing.T) {
	ctrlNoHelper := gomock.NewController(NewErrorReporter(t))
	// doesn't panic
	ctrlNoHelper.T.Helper()
}

func TestWithHelper(t *testing.T) {
	withHelper := &HelperReporter{TestReporter: NewErrorReporter(t)}
	ctrlWithHelper := gomock.NewController(withHelper)
	ctrlWithHelper.T.Helper()
	if withHelper.helper == 0 {
		t.Fatal("expected Helper to be invoked")
	}
}

func (e *ErrorReporter) Cleanup(f func()) {
	e.t.Helper()
	e.t.Cleanup(f)
}

func TestMultipleDefers(t *testing.T) {
	reporter := NewErrorReporter(t)
	reporter.Cleanup(func() {
		reporter.assertPass("No errors for multiple calls to Finish")
	})
	_ = gomock.NewController(reporter)
}

func TestDeferNotNeededPass(t *testing.T) {
	reporter := NewErrorReporter(t)
	subject := new(Subject)
	var ctrl *gomock.Controller
	reporter.Cleanup(func() {
		reporter.assertPass("Expected method call made.")
	})
	ctrl = gomock.NewController(reporter)
	var expects []*gomock.Call1_1[string, int]
	addExpect(ctrl, &expects, gomock.NewCall1_1[string, int](
		reporter, subject, "FooMethod", gomock.Eq("argument"),
	))
	gomock.Dispatch1_1(&expects, ctrl, subject, "FooMethod", "argument")
}

func TestOrderedCallsInCorrect(t *testing.T) {
	reporter := NewErrorReporter(t)
	subjectOne := new(Subject)
	subjectTwo := new(Subject)
	var ctrl *gomock.Controller
	reporter.Cleanup(func() {
		call1 := gomock.NewCall1_1[string, int](reporter, subjectOne, "FooMethod", gomock.Any()).AnyTimes()
		call2 := gomock.NewCall1_1[string, int](reporter, subjectTwo, "FooMethod", gomock.Any())
		call3 := gomock.NewCall1_1[string, int](reporter, subjectTwo, "BarMethod", gomock.Any())
		gomock.InOrder(call1, call2, call3)

		var e1, e2, e3 []*gomock.Call1_1[string, int]
		addExpect(ctrl, &e1, call1)
		addExpect(ctrl, &e2, call2)
		addExpect(ctrl, &e3, call3)

		reporter.assertFatal(func() {
			gomock.Dispatch1_1(&e1, ctrl, subjectOne, "FooMethod", "1")
			// BarMethod should only be called after FooMethod("2").
			gomock.Dispatch1_1(&e3, ctrl, subjectTwo, "BarMethod", "3")
		}, "Unexpected call to")
	})
	ctrl = gomock.NewController(reporter)
}

func TestCallAfterLoopPanic(t *testing.T) {
	reporter := NewErrorReporter(t)
	subject := new(Subject)
	var ctrl *gomock.Controller
	reporter.Cleanup(func() {
		firstCall := gomock.NewCall1_1[string, int](reporter, subject, "FooMethod", gomock.Any())
		secondCall := gomock.NewCall1_1[string, int](reporter, subject, "FooMethod", gomock.Any())
		thirdCall := gomock.NewCall1_1[string, int](reporter, subject, "FooMethod", gomock.Any())

		var e1, e2, e3 []*gomock.Call1_1[string, int]
		addExpect(ctrl, &e1, firstCall)
		addExpect(ctrl, &e2, secondCall)
		addExpect(ctrl, &e3, thirdCall)
		gomock.InOrder(firstCall, secondCall, thirdCall)

		defer func() {
			err := recover()
			if err == nil {
				t.Error("Call.After creation of dependency loop did not panic.")
			}
		}()

		// This should panic due to the dependency loop.
		firstCall.After(thirdCall)
	})
	ctrl = gomock.NewController(reporter)
}
