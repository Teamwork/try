package try

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"context"
	"time"

	"github.com/hashicorp/go-multierror"
)

type exam struct {
	description string
	fn          func() func() error
	ctx         context.Context
	validate    func([]error, bool, *testing.T)
}

var exams = []exam{
	{
		"length of errs should be zero",
		func() func() error {
			return func() error {
				return nil
			}
		},
		context.Background(),
		func(e []error, b bool, t *testing.T) {
			if len(e) != 0 {
				t.Errorf("expected length of error be 0 but it's %d", len(e))
			}
			if !b {
				t.Errorf("channel should be closed but it is not")
			}
		},
	},
	{"should return 2 error",
		func() func() error {
			x := 0
			return func() error {
				if x != 2 {
					x++
					return errors.New("test")
				}

				return nil
			}
		},
		context.Background(),
		func(e []error, b bool, t *testing.T) {
			if len(e) != 2 {
				t.Errorf("expected length of error be 2 but it's %d", len(e))
			}
			if !b {
				t.Errorf("channel should be closed but it is not")
			}
		},
	}, {"should cancel",
		func() func() error {
			return func() error {
				time.Sleep(time.Millisecond)
				return errors.New("test")
			}
		},
		func() context.Context {
			c, cl := context.WithTimeout(context.Background(), time.Duration(0))
			defer cl()
			return c
		}(),
		func(e []error, b bool, t *testing.T) {
			if len(e) != 0 {
				t.Errorf("expected length of error be 0 but it's %d", len(e))
			}
			if !b {
				t.Errorf("channel should be closed but it is not")
			}
		},
	},
}

func TestWithCancel(t *testing.T) {
	for _, c := range exams {
		t.Log(c.description)
		cErr := WithCancel(c.ctx, time.Millisecond, 20, c.fn())
		errs := make([]error, 0)
		to := time.After(20 * time.Millisecond)
	loop:
		for {
			select {
			case err, open := <-cErr:
				if !open {
					c.validate(errs, true, t)
					break loop
				} else {
					errs = append(errs, err)
				}
			case <-to:
				c.validate(errs, false, t)
				break loop
			}
		}
	}

}

func TestLimited(t *testing.T) {
	exams := []struct {
		fn    func() error
		count int
	}{
		{func() error { return errors.New("test") }, 5},
		{func() error { return nil }, 0},
	}
next:
	for _, v := range exams {
		limit := 5
		var counter int
		cErr := Limited(limit, time.Millisecond, 1, v.fn)
		to := time.After(2 * time.Second)
		for {
			select {
			case _, open := <-cErr:
				if !open {
					if counter != v.count {
						t.Errorf("expected limited should invoke fn %d times but did it %d time(s)", v.count, counter)
					}
					break next
				} else {
					counter++
				}
			case <-to:
				t.Errorf("expected limited should be done but it didn't")
				break next
			}
		}
	}
}

func TestFibonacci(t *testing.T) {
	cases := []uint{0, 1, 1, 2, 3, 5, 8, 12, 12, 12, 12, 12}
	f := fibonacci(12)
	for i := 0; i < len(cases); i++ {
		if x := f(); x != cases[i] {
			t.Errorf("expected %d but value is %d", cases[i], x)
		}
	}
}

func TestTryExample(t *testing.T) {
	SomeFunction := func() (string, error) {
		return "", nil
	}
	err := Do(func(attempt int) (bool, error) {
		var err error
		_, err = SomeFunction()
		return attempt < 5, err // try 5 times
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTryDoSuccessful(t *testing.T) {
	callCount := 0
	err := Do(func(attempt int) (bool, error) {
		callCount++
		return attempt < 5, nil
	})

	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected callCount: %d, got: %d", 1, callCount)
	}
}

func TestTryDoSuccessfulAfterFailure(t *testing.T) {
	callCount := 0
	err := Do(func(attempt int) (bool, error) {
		callCount++

		if attempt < 3 {
			return true, fmt.Errorf("failure %d", attempt)
		}
		return false, nil
	})

	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected callCount: %d, got: %d", 3, callCount)
	}
}

func TestTryDoFailed(t *testing.T) {
	var errs error
	for i := 1; i <= 5; i++ {
		errs = multierror.Append(errs, fmt.Errorf("err attempt: %d", i))
	}
	callCount := 0
	err := Do(func(attempt int) (bool, error) {
		callCount++
		return attempt < 5, fmt.Errorf("err attempt: %d", attempt)
	})

	if !reflect.DeepEqual(err, errs) {
		t.Errorf("expected err: \n%v\ngot:\n%v", errs, err)
	}
	if callCount != 5 {
		t.Errorf("expected callCount: %d, got: %d", 5, callCount)
	}
}

func TestTryPanics(t *testing.T) {
	theErr := errors.New("something went wrong")
	errs := multierror.Append(theErr, errors.New("panic: I don't like three"))
	callCount := 0
	err := Do(func(attempt int) (retry bool, err error) {
		retry = attempt < 2
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		callCount++
		if attempt == 2 {
			panic("I don't like three")
		}
		err = theErr
		return
	})

	if !reflect.DeepEqual(err, errs) {
		t.Errorf("expected err: \n%v\ngot:\n%v", errs, err)
	}
	if callCount != 2 {
		t.Errorf("expected callCount: %d, got: %d", 5, callCount)
	}
}

func TestRetryLimit(t *testing.T) {
	err := Do(func(attempt int) (bool, error) {
		return true, errors.New("nope")
	})

	if err == nil {
		t.Errorf("expected err, got: <nil>")
	}
	if !IsMaxRetries(err) {
		t.Errorf("expected IsMaxRetries to be true")
	}
}

func TestIsMaxRetries(t *testing.T) {
	tests := []struct {
		in  error
		out bool
	}{
		{nil, false},
		{errors.New("nope"), false},
		{errMaxRetriesReached, true},
		{multierror.Append(nil), false},
		{multierror.Append(nil, errMaxRetriesReached), true},
		{multierror.Append(errors.New("something"), errMaxRetriesReached), true},
	}
	for _, tt := range tests {
		if IsMaxRetries(tt.in) != tt.out {
			t.Errorf("isMaxRetries(%v) expected to be %t", tt.in, tt.out)
		}
	}
}
