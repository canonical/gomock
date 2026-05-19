// Copyright 2020 Google Inc.
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
	"testing"
)

type a struct {
	name string
}

func (testObj a) Name() string {
	return testObj.name
}

// c wraps *Call and implements CallHolder so InOrder can extract it.
type c struct {
	*Call
}

func (x *c) getCall() *Call {
	return x.Call
}

type mockTestReporter struct {
	errorCalls int
	fatalCalls int
}

func (o *mockTestReporter) Errorf(format string, args ...any) {
	o.errorCalls++
}

func (o *mockTestReporter) Fatalf(format string, args ...any) {
	o.fatalCalls++
}

func (o *mockTestReporter) Helper() {}

func TestCall_After(t *testing.T) {
	t.Run("SelfPrereqCallsFatalf", func(t *testing.T) {
		tr1 := &mockTestReporter{}

		c := &Call{t: tr1}
		c.After(c)

		if tr1.fatalCalls != 1 {
			t.Errorf(
				"number of fatal calls == %v, want 1",
				tr1.fatalCalls,
			)
		}
	})

	t.Run("LoopInCallOrderCallsFatalf", func(t *testing.T) {
		tr1 := &mockTestReporter{}
		tr2 := &mockTestReporter{}

		c1 := &Call{t: tr1}
		c2 := &Call{t: tr2}
		c1.After(c2)
		c2.After(c1)

		if tr1.errorCalls != 0 || tr1.fatalCalls != 0 {
			t.Error("unexpected errors")
		}

		if tr2.fatalCalls != 1 {
			t.Errorf(
				"number of fatal calls == %v, want 1",
				tr2.fatalCalls,
			)
		}
	})
}

func TestInOrder(t *testing.T) {
	t.Run("process only *Call or its wrappers", func(t *testing.T) {
		tr1 := &mockTestReporter{}
		tr2 := &mockTestReporter{}
		c1 := &Call{t: tr1}
		c2 := &c{Call: &Call{t: tr2}}
		InOrder(c1, c2)
		if len(c2.preReqs) != 1 {
			t.Fatalf(
				"expected 1 preReq in c2, found %d",
				len(c2.preReqs),
			)
		}
		if len(c1.preReqs) != 0 {
			t.Fatalf(
				"expected 0 preReq in c1, found %d",
				len(c1.preReqs),
			)
		}
	})
	t.Run(
		"panic when the argument isn't a *Call or has one embedded",
		func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected InOrder to panic")
				}
			}()
			tr := &mockTestReporter{}
			c := &Call{t: tr}
			a := &a{
				name: "Foo",
			}
			InOrder(c, a)
		},
	)
}
