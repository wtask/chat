package history

import (
	"reflect"
	"testing"
)

func TestStack(test *testing.T) {
	if _, err := NewStack(0); err == nil {
		test.Error("NewStack(0):", "excpected error got nil")
	}
	if _, err := NewStack(-1); err == nil {
		test.Error("NewStack(-1):", "excpected error got nil")
	}

	s, _ := NewStack(2)
	s.Push("1")
	s.Push("2")
	s.Push("3")
	if s.Len() != 2 {
		test.Error("Unexpected Stack len", s.Len())
	}

	if t := s.Tail(0); !reflect.DeepEqual(t, []string{}) {
		test.Error("Unexpected Tail(0) result", t)
	}
	if t := s.Tail(2); !reflect.DeepEqual(t, []string{"2", "3"}) {
		test.Error("Unexpected Tail(2) result", t)
	}
	if t := s.Tail(-2); !reflect.DeepEqual(t, []string{"2", "3"}) {
		test.Error("Unexpected Tail(-2) result", t)
	}
	if t := s.Tail(100); !reflect.DeepEqual(t, []string{"2", "3"}) {
		test.Error("Unexpected Tail(100) result", t)
	}
}
