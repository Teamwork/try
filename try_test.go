package try

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	multierror "github.com/hashicorp/go-multierror"
)

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
