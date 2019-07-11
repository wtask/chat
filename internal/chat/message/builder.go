package message

import (
	"bytes"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Builder - implements io.Writer interface to build message body from byte parts
type Builder struct {
	reminder bytes.Buffer
	str      strings.Builder
}

func (b *Builder) Write(p []byte) (n int, err error) {
	reminder := []byte{} // invalid unicode sequence at the end of p (if exist)
	defer func() {
		b.reminder.Write(reminder)
	}()

	i, s := LastValidRune(p)
	if i > -1 {
		reminder = p[i+s:]
	} else {
		// all bytes in p represemt invalid unicode sequence
		reminder = p
	}
	b.reminder.Write(p)

	var prev rune
	for n, err = 0, nil; ; {
		r, size, err := b.reminder.ReadRune()
		if err != nil {
			if err == io.EOF {
				return n, nil
			}
			return n, err
		}
		n += size
		if r == utf8.RuneError {
			// drop
			continue
		}
		switch {
		default:
			b.str.WriteRune(r)
		case r == '\n':
			// replace continuous EOL with single space
			if prev != r {
				b.str.WriteByte(' ')
			}
		case unicode.IsSpace(r):
			b.str.WriteByte(' ')
		case unicode.IsControl(r):
			// drop
		}
		prev = r
	}
}

// Len - returns length (in bytes) of ready string.
func (b *Builder) Len() int {
	return b.str.Len()
}

// Total - return total size in bytes of underlying data.
// Total value may be grater than length of ready string.
func (b *Builder) Total() int {
	return b.Len() + b.reminder.Len()
}

// Flush - returns built string and resets internal builder
func (b *Builder) Flush() string {
	defer b.str.Reset()
	return b.str.String()
}

// LastValidRune - return index and size in bytes of last well-encoded rune in given slice.
// Returns (-1, 0) if source does not contain valide unicode code points.
func LastValidRune(s []byte) (i, size int) {
	if len(s) == 0 {
		return 0, 0
	}
	return bytes.LastIndexFunc(s, func(r rune) bool {
			valid := r != utf8.RuneError && utf8.ValidRune(r)
			if valid {
				size = utf8.RuneLen(r)
			}
			return valid
		}),
		size
}
