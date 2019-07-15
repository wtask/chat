package history

import (
	"errors"
	"sync"
)

// Stack - accumulates a limited number of strings in style of LIFO queue.
// When stack length is reached max value, it drops first item on every push.
type Stack struct {
	max  int
	mu   sync.RWMutex
	data []string
}

// NewStack - build history stack.
func NewStack(max int) (*Stack, error) {
	if max <= 0 {
		return nil, errors.New("NewLimitedStack: max (%d) must be greater than 0")
	}
	return &Stack{max: max, data: []string{}}, nil
}

// Len - returns number of currently
func (s *Stack) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// Push - adds item to history.
func (s *Stack) Push(item string) {
	l := s.Len()
	s.mu.Lock()
	defer s.mu.Unlock()
	if l == s.max {
		s.data = s.data[1:]

	}
	s.data = append(s.data, item)
}

// Tail - makes copy of last n-items from stack into resulting slice.
// The first item in resulting slice is the most older.
func (s *Stack) Tail(n int) []string {
	if n == 0 {
		return []string{}
	}
	if n < 0 {
		n *= -1
	}
	l := s.Len()
	if l == 0 {
		return []string{}
	}
	if n > l {
		n = l
	}
	tail := make([]string, n)
	s.mu.Lock()
	defer s.mu.Unlock()
	copy(tail, s.data[l-n:])
	return tail
}
