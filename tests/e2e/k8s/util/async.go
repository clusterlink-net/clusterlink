// Copyright (c) The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"sync"
)

// AsyncRunner allows asynchronous running of functions with error tracking.
type AsyncRunner struct {
	wg sync.WaitGroup

	lock sync.RWMutex
	err  error
}

// Error returns the (combined) errors collected.
func (r *AsyncRunner) Run(f func() error) {
	if r.Error() != nil {
		return
	}

	r.wg.Add(1)
	go func() {
		r.SetError(f())
		r.wg.Done()
	}()
}

// SetError collects a new error.
func (r *AsyncRunner) SetError(err error) {
	if err == nil {
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	if r.err == nil {
		r.err = err
	}
}

// Error returns the (combined) errors collected.
func (r *AsyncRunner) Error() error {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.err
}

// Wait for async operations to finish.
func (r *AsyncRunner) Wait() error {
	r.wg.Wait()
	return r.Error()
}
