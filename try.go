// The MIT License (MIT)

// Copyright (c) 2015 Mat Ryer

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package try

import (
	"errors"

	multierror "github.com/hashicorp/go-multierror"
)

// MaxRetries is the maximum number of retries before bailing.
var MaxRetries = 10

var errMaxRetriesReached = errors.New("exceeded retry limit")

// Func represents functions that can be retried.
type Func func(attempt int) (retry bool, err error)

// Do keeps trying the function until the second argument returns false, or no
// error is returned, attempt is started at 1.
//
// A *multierror.Error combining all attempt errors on failure.
// If the function does not return true before MaxRetries then the combination
// of all errors that occurred will be returned which IsMaxRetries() will return
// true for.
func Do(fn Func) error {
	err := do(fn)
	if merr, ok := err.(*multierror.Error); ok {
		return merr.ErrorOrNil()
	}
	return err
}

func do(fn Func) error {
	var errs error
	attempt := 1
	for {
		cont, err := fn(attempt)
		if err == nil {
			return nil
		}

		errs = multierror.Append(errs, err)
		if !cont {
			return errs
		}

		attempt++
		if attempt > MaxRetries {
			return multierror.Append(errs, errMaxRetriesReached)
		}
	}
}

// IsMaxRetries checks whether the error is due to hitting the
// maximum number of retries or not.
func IsMaxRetries(err error) bool {
	if merr, ok := err.(*multierror.Error); ok {
		if len(merr.Errors) == 0 {
			return false
		}
		return merr.Errors[len(merr.Errors)-1] == errMaxRetriesReached
	}
	return err == errMaxRetriesReached
}
