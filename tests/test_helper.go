package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type Helper struct {
	t *testing.T
}

func H(t *testing.T) Helper {
	t.Helper()
	return Helper{t}
}

func (h Helper) TypeEql(got, want interface{}) {
	h.t.Helper()
	// check obvious case
	if got == nil && want == nil {
		return
	}
	// check for type equality
	if strings.Compare(fmt.Sprintf("%T", got), fmt.Sprintf("%T", want)) != 0 {
		h.t.Fatalf("type equality assertion failed, got %q wanted %q", fmt.Sprintf("%T", got), fmt.Sprintf("%T", want))
	}
}

func (h Helper) IntEql(got, want int) {
	h.t.Helper()
	if got != want {
		h.t.Fatalf("int equality assertion failed, got %d wanted %d", got, want)
	}
}

func (h Helper) Int64Eql(got, want int64) {
	h.t.Helper()
	if got != want {
		h.t.Fatalf("int equality assertion failed, got %d wanted %d", got, want)
	}
}

func (h Helper) StringEql(got, want string) {
	h.t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		h.t.Errorf("string equality assertion failed (-got +want)\n%s", diff)
	}
}

func (h Helper) InterfaceEql(got, want interface{}) {
	h.t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		h.t.Errorf("string equality assertion failed (-got +want)\n%s", diff)
	}
}

func (h Helper) ErrEql(got, want error) {
	h.t.Helper()
	if got == nil && want == nil {
		return
	}
	if got != nil && want == nil {
		h.t.Fatalf("error equality assertion failed, got %q wanted nil", got)
	}
	if got != nil && want != nil {
		if got.Error() != want.Error() {
			h.t.Fatalf("error equality assertion failed, got %q wanted %q", got, want.Error())
		}
	}
}

func (h Helper) IsNil(any interface{}) {
	h.t.Helper()
	if any != nil {
		h.t.Fatalf("wanted not nil, got %v", any)
	}
}

func (h Helper) NotNil(any interface{}) {
	h.t.Helper()
	if any == nil {
		h.t.Fatalf("wanted not nil, got %v", any)
	}
}

func (h Helper) BoolEql(got, want bool) {
	h.t.Helper()
	if got != want {
		h.t.Fatalf("boolean equality assertion failed, got %t wanted %t", got, want)
	}
}

func (h Helper) BoolEqlWithMessage(got, want bool, message string) {
	h.t.Helper()
	if got != want {
		h.t.Fatalf("boolean equality assertion failed, got %t wanted %t - %s", got, want, message)
	}
}

func (h Helper) StringContains(got, want string) {
	h.t.Helper()

	if !strings.Contains(got, want) {
		h.t.Fatalf("wanted %v to contain %v", got, want)
	}
}
