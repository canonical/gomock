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
	"fmt"
)

// Call represents an expected call to a mock.
type Call struct {
	t TestHelper // for triggering test failures on invalid call setup

	receiver any    // the receiver of the method call
	method   string // the name of the method
	origin   string // file and line number of call setup

	preReqs []*Call // prerequisite calls

	// Expectations
	minCalls, maxCalls int

	numCalls int // actual number of calls made
}

// CallHolder is implemented by the typed CallN_M wrappers so that
// InOrder can extract the underlying *Call without reflection.
// *Call itself also implements this interface.
type CallHolder interface {
	getCall() *Call
}

// getCall implements CallHolder.
func (c *Call) getCall() *Call { return c }

// newCall creates an expected-call record.
// callerSkip is the number of extra stack frames above newCall to
// skip so that the reported origin points at the user's test code.
//
// Convention: when newCall is called from a generated NewCallN_M
// constructor, which is itself called by a generated recorder method,
// which is called by test code, the stack is:
//
//	callerInfo -> newCall -> NewCallN_M -> recorder.Method -> TestCode
//
// Pass callerSkip=2 so callerInfo(3) -> runtime.Caller(4) lands on
// TestCode.
func newCall(
	t TestHelper,
	receiver any,
	method string,
	callerSkip int,
) *Call {
	t.Helper()
	return &Call{
		t:        t,
		receiver: receiver,
		method:   method,
		origin:   callerInfo(callerSkip + 1),
		minCalls: 1,
		maxCalls: 1,
	}
}

// AnyTimes allows the expectation to be called 0 or more times.
func (c *Call) AnyTimes() *Call {
	c.minCalls, c.maxCalls = 0, 1e8 // close enough to infinity
	return c
}

// MinTimes requires the call to occur at least n times. If AnyTimes
// or MaxTimes have not been called, or if MaxTimes was previously
// called with 1, MinTimes also sets the maximum number of calls to
// infinity.
func (c *Call) MinTimes(n int) *Call {
	c.minCalls = n
	if c.maxCalls == 1 {
		c.maxCalls = 1e8
	}
	return c
}

// MaxTimes limits the number of calls to n times. If AnyTimes or
// MinTimes have not been called, or if MinTimes was previously called
// with 1, MaxTimes also sets the minimum number of calls to 0.
func (c *Call) MaxTimes(n int) *Call {
	c.maxCalls = n
	if c.minCalls == 1 {
		c.minCalls = 0
	}
	return c
}

// Times declares the exact number of times a function call is
// expected to be executed.
func (c *Call) Times(n int) *Call {
	c.minCalls, c.maxCalls = n, n
	return c
}

// After declares that the call may only match after preReq has been
// exhausted.
func (c *Call) After(preReq *Call) *Call {
	c.t.Helper()

	if c == preReq {
		c.t.Fatalf(
			"A call isn't allowed to be its own prerequisite",
		)
	}
	if preReq.isPreReq(c) {
		c.t.Fatalf(
			"Loop in call order: %v is a prerequisite to %v"+
				" (possibly indirectly).",
			c, preReq,
		)
	}

	c.preReqs = append(c.preReqs, preReq)
	return c
}

// isPreReq returns true if other is a direct or indirect prerequisite
// to c.
func (c *Call) isPreReq(other *Call) bool {
	for _, preReq := range c.preReqs {
		if other == preReq || preReq.isPreReq(other) {
			return true
		}
	}
	return false
}

// satisfied returns true if the minimum number of calls have been
// made.
func (c *Call) satisfied() bool {
	return c.numCalls >= c.minCalls
}

// exhausted returns true if the maximum number of calls have been
// made.
func (c *Call) exhausted() bool {
	return c.numCalls >= c.maxCalls
}

// String returns a human-readable description of the expected call.
func (c *Call) String() string {
	return fmt.Sprintf(
		"%T.%v(...) %s", c.receiver, c.method, c.origin,
	)
}

// dropPrereqs tells the expected Call to stop re-checking
// prerequisite calls and returns its current set.
func (c *Call) dropPrereqs() (preReqs []*Call) {
	preReqs = c.preReqs
	c.preReqs = nil
	return
}

// InOrder declares that the given calls should occur in order.
// It panics if any argument is neither *Call nor a type that
// implements CallHolder.
func InOrder(args ...any) {
	calls := make([]*Call, 0, len(args))
	for i := 0; i < len(args); i++ {
		if call := getCall(args[i]); call != nil {
			calls = append(calls, call)
			continue
		}
		panic(fmt.Sprintf(
			"invalid argument at position %d of type %T,"+
				" InOrder expects *gomock.Call or generated"+
				" mock types with an embedded *gomock.Call",
			i,
			args[i],
		))
	}
	for i := 1; i < len(calls); i++ {
		calls[i].After(calls[i-1])
	}
}

// getCall checks if arg is a *Call or implements CallHolder, and
// returns the underlying *Call. Returns nil if neither.
func getCall(arg any) *Call {
	if call, ok := arg.(*Call); ok {
		return call
	}
	if holder, ok := arg.(CallHolder); ok {
		return holder.getCall()
	}
	return nil
}
