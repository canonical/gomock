// Copyright 2010 Google Inc.
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

package gomock

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

// A TestReporter is something that can be used to report test
// failures. It is satisfied by the standard library's *testing.T.
type TestReporter interface {
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// TestHelper is a TestReporter that has the Helper method. It is
// satisfied by the standard library's *testing.T.
type TestHelper interface {
	TestReporter
	Helper()
}

// cleanuper is used to check if TestHelper also has the `Cleanup`
// method. A common pattern is to pass in a `*testing.T` to
// `NewController(t TestReporter)`. In Go 1.14+, `*testing.T` has a
// cleanup method. This can be utilised to call `Finish()` so the
// caller of this library does not have to.
type cleanuper interface {
	Cleanup(func())
}

// A Controller represents the top-level control of a mock ecosystem.
// It defines the scope and lifetime of mock objects, as well as their
// expectations. It is safe to call Controller's methods from multiple
// goroutines. Each test should create a new Controller.
//
//	func TestFoo(t *testing.T) {
//	  ctrl := gomock.NewController(t)
//	  // ..
//	}
type Controller struct {
	// T should only be called within a generated mock. It is not
	// intended to be used in user code and may be changed in future
	// versions. T is the TestReporter passed in when creating the
	// Controller via NewController. If the TestReporter does not
	// implement a TestHelper it will be wrapped with a nopTestHelper.
	T           TestHelper
	mu          sync.Mutex
	calls       []*Call
	finished    bool
	overridable bool
}

// NewController returns a new Controller. It is the preferred way to
// create a Controller.
//
// Passing [*testing.T] registers a cleanup function to automatically
// call [Controller.Finish] when the test and all its subtests
// complete.
func NewController(
	t TestReporter,
	opts ...ControllerOption,
) *Controller {
	h, ok := t.(TestHelper)
	if !ok {
		h = &nopTestHelper{t}
	}
	ctrl := &Controller{
		T: h,
	}
	for _, opt := range opts {
		opt.apply(ctrl)
	}
	if c, ok := isCleanuper(ctrl.T); ok {
		c.Cleanup(func() {
			ctrl.T.Helper()
			ctrl.finish(true, nil)
		})
	}
	return ctrl
}

// ControllerOption configures how a Controller should behave.
type ControllerOption interface {
	apply(*Controller)
}

type overridableExpectationsOption struct{}

func (overridableExpectationsOption) apply(ctrl *Controller) {
	ctrl.overridable = true
}

// WithOverridableExpectations allows expectations to be overridden
// by later registrations for the same method. The most recently
// registered expectation takes precedence during dispatch.
func WithOverridableExpectations() ControllerOption {
	return overridableExpectationsOption{}
}

type cancelReporter struct {
	t      TestHelper
	cancel func()
}

func (r *cancelReporter) Errorf(format string, args ...any) {
	r.t.Errorf(format, args...)
}

func (r *cancelReporter) Fatalf(format string, args ...any) {
	defer r.cancel()
	r.t.Fatalf(format, args...)
}

func (r *cancelReporter) Helper() {
	r.t.Helper()
}

// WithContext returns a new Controller and a Context, which is
// cancelled on any fatal failure.
func WithContext(
	ctx context.Context,
	t TestReporter,
) (*Controller, context.Context) {
	h, ok := t.(TestHelper)
	if !ok {
		h = &nopTestHelper{t: t}
	}
	ctx, cancel := context.WithCancel(ctx)
	return NewController(
		&cancelReporter{t: h, cancel: cancel},
	), ctx
}

type nopTestHelper struct {
	t TestReporter
}

func (h *nopTestHelper) Errorf(format string, args ...any) {
	h.t.Errorf(format, args...)
}

func (h *nopTestHelper) Fatalf(format string, args ...any) {
	h.t.Fatalf(format, args...)
}

func (h nopTestHelper) Helper() {}

// removeCall removes a *Call from ctrl.calls.
func (ctrl *Controller) removeCall(c *Call) {
	for i, existing := range ctrl.calls {
		if existing == c {
			ctrl.calls = append(
				ctrl.calls[:i], ctrl.calls[i+1:]...,
			)
			return
		}
	}
}

// Track registers c with this controller for Finish() lifecycle
// checking. Called by generated recorder methods.
func (ctrl *Controller) Track(c *Call) {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()
	ctrl.calls = append(ctrl.calls, c)
}

// Finish checks to see if all the methods that were expected to be
// called were called. It is not idempotent and therefore can only be
// invoked once.
//
// Note: If you pass a *testing.T into [NewController], you no longer
// need to call ctrl.Finish() in your test methods.
func (ctrl *Controller) Finish() {
	// If we're currently panicking, probably because this is a
	// deferred call. This must be recovered in the deferred function.
	err := recover()
	ctrl.finish(false, err)
}

// Satisfied returns whether all expected calls bound to this
// Controller have been satisfied. Calling Finish is then guaranteed
// to not fail due to missing calls.
func (ctrl *Controller) Satisfied() bool {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()
	for _, call := range ctrl.calls {
		if !call.satisfied() {
			return false
		}
	}
	return true
}

func (ctrl *Controller) finish(cleanup bool, panicErr any) {
	ctrl.T.Helper()

	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()

	if ctrl.finished {
		if _, ok := isCleanuper(ctrl.T); !ok {
			ctrl.T.Fatalf(
				"Controller.Finish was called more than once." +
					" It has to be called exactly once.",
			)
		}
		return
	}
	ctrl.finished = true

	if panicErr != nil {
		panic(panicErr)
	}

	var failures []*Call
	for _, call := range ctrl.calls {
		if !call.satisfied() {
			failures = append(failures, call)
		}
	}
	for _, call := range failures {
		ctrl.T.Errorf("missing call(s) to %v", call)
	}
	if len(failures) != 0 {
		if !cleanup {
			ctrl.T.Fatalf("aborting test due to missing call(s)")
			return
		}
		ctrl.T.Errorf("aborting test due to missing call(s)")
	}
}

// callerInfo returns the file:line of the call site. skip is the
// number of stack frames to skip. 0 identifies callerInfo's own
// call site.
func callerInfo(skip int) string {
	if _, file, line, ok := runtime.Caller(skip + 1); ok {
		return fmt.Sprintf("%s:%d", file, line)
	}
	return "unknown file"
}

// isCleanuper checks if t's base TestReporter has a Cleanup method.
func isCleanuper(t TestReporter) (cleanuper, bool) {
	tr := unwrapTestReporter(t)
	c, ok := tr.(cleanuper)
	return c, ok
}

// unwrapTestReporter unwraps TestReporter to the base implementation.
func unwrapTestReporter(t TestReporter) TestReporter {
	tr := t
	switch nt := t.(type) {
	case *cancelReporter:
		tr = nt.t
		if h, check := tr.(*nopTestHelper); check {
			tr = h.t
		}
	case *nopTestHelper:
		tr = nt.t
	default:
		// not wrapped
	}
	return tr
}
