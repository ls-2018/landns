package landns

import (
	"fmt"
	"testing"
)

func TestErrorType(t *testing.T) {
	t.Parallel()

	for i, tt := range []string{"UnknownError", "InternalError", "ExternalError", "ArgumentError", "ExpirationError", "UnknownError"} {
		if s := ErrorType(i).String(); s != tt {
			t.Errorf("unexpected error string: expected %#v but got %#v", tt, s)
		}
	}

	for _, tt := range []struct {
		Type   ErrorType
		Expect string
	}{
		{TypeInternalError, "InternalError"},
		{TypeExternalError, "ExternalError"},
		{TypeArgumentError, "ArgumentError"},
		{TypeExpirationError, "ExpirationError"},
	} {
		if s := tt.Type.String(); s != tt.Expect {
			t.Errorf("unexpected error string: expected %#v but got %#v", tt.Expect, s)
		}
	}
}

func TestError(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		Err    Error
		Expect string
	}{
		{Error{TypeInternalError, nil, "some error"}, "some error"},
		{Error{TypeExternalError, fmt.Errorf("world"), "hello"}, "hello: world"},
	} {
		if tt.Err.Unwrap() != tt.Err.Original {
			t.Errorf("failed to get original error: expected %#v but got %#v", tt.Err.Original, tt.Err.Unwrap())
		}

		if tt.Err.Error() != tt.Expect {
			t.Errorf("unexpected error string:\nexpected: %#v\nbut got:  %#v", tt.Expect, tt.Err.Error())
		}

		if tt.Err.String() != tt.Expect {
			t.Errorf("unexpected string:\nexpected: %#v\nbut got:  %#v", tt.Expect, tt.Err.Error())
		}
	}
}

func TestNewError(t *testing.T) {
	t.Parallel()

	orig := fmt.Errorf("original")
	err := newError(TypeInternalError, orig, "hello %s", "world")

	expected := "hello world: original"
	if err.Error() != expected {
		t.Errorf("failed to create Error:\nexpected: %#v\nbut got:  %#v", expected, err.Error())
	}
}

func TestWrapError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Original error
		Type     ErrorType
		Message  string
		Expect   string
	}{
		{fmt.Errorf("original"), TypeInternalError, "hello world", "hello world: original"},
		{nil, TypeExternalError, "hello world", ""},
	}

	for _, tt := range tests {
		err := wrapError(tt.Original, tt.Type, tt.Message)
		if err == nil && tt.Expect != "" {
			t.Errorf("failed to create Error:\nexpected: %#v\nbut got:  %#v", tt.Expect, err)
		}
		if err != nil && err.Error() != tt.Expect {
			t.Errorf("failed to create Error:\nexpected: %#v\nbut got:  %#v", tt.Expect, err)
		}
	}
}

func TestErrorSet(t *testing.T) {
	t.Parallel()

	err := ErrorSet{
		fmt.Errorf("hello"),
		fmt.Errorf("world"),
	}
	expected := "hello\nworld"

	if err.Error() != expected {
		t.Errorf("unexpected error string:\nexpected:\n%s\n\nbut got:\n%s\n", expected, err.Error())
	}
}
