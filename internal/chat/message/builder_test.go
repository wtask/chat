package message

import (
	"testing"
	"unicode/utf8"
)

func TestLastValidRune(test *testing.T) {
	cases := []struct {
		data       []byte
		expI, expS int
	}{
		{[]byte{}, -1, 0},
		{[]byte("⌘"), 0, 3},                          // "⌘": []byte{226, 140, 152}
		{[]byte{226, 140}, -1, 0},                    // invalid sequence
		{[]byte{226, 140, 226, 140, 152}, 2, 3},      // there are invalid sequences
		{[]byte{226, 140, 226, 140, 152, 226}, 2, 3}, // there are invalid sequences
		{[]byte{226, 140, '!'}, 2, 1},
		{[]byte("Hello, 世界"), 10, utf8.RuneLen('界')},
		{[]byte("Hello, 世界!"), 13, utf8.RuneLen('!')},
	}

	for _, c := range cases {
		actI, actS := LastValidRune(c.data)
		if actI != c.expI || actS != c.expS {
			test.Errorf("Data: %[1]v, %[1]q; expected: %d, %d; actual: %d, %d", c.data, c.expI, c.expS, actI, actS)
		}
	}
}

func TestBuilder(test *testing.T) {
	builder := Builder{}
	if builder.Len() != 0 {
		test.Error("Invalid string length just after init", builder.Len())
	}
	if s := builder.Flush(); s != "" {
		test.Error("Invalid string has built just after init", s)
	}
	content := []byte("Hello Builder!")
	builder.Write(content)
	cpoint := []byte{226, 140, 152} // ⌘
	// write invalid unicode sequence
	builder.Write(cpoint[:2])
	if s := builder.Flush(); s != string(content) {
		test.Error("Expected Flush() result:", string(content), "actual:", s)
	}
	// write invalid unicode sequence again,
	// but Builder remembers previous bytes
	builder.Write(cpoint[2:])
	if s := builder.Flush(); s != string(cpoint) {
		test.Error("Expected Flush() result:", string(cpoint), "actual:", s)
	}
}
