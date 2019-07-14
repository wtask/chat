package message

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Builder - implements io.Writer interface to build message body from byte parts.
type Builder struct {
	remainder bytes.Buffer
	mu        sync.RWMutex // protects both of builders
	draft,
	message strings.Builder
}

func (b *Builder) Write(p []byte) (n int, err error) {
	remainder := []byte{} // invalid unicode sequence at the end of p (if exist)
	defer func() {
		b.remainder.Write(remainder)
	}()

	i, s := LastValidRune(p)
	if i > -1 {
		remainder = p[i+s:]
	} else {
		// all bytes in p represemt invalid unicode sequence
		remainder = p
	}
	b.remainder.Write(p)

	var prev rune
	for n, err = 0, nil; ; {
		r, size, err := b.remainder.ReadRune()
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
			b.mu.Lock()
			b.draft.WriteRune(r)
			b.mu.Unlock()
		case r == '\n':
			if prev == r {
				// drop continuos EOLs
				break
			}
			b.mu.Lock()
			if b.draft.Len() > 0 {
				// complete message
				if b.message.Len() > 0 {
					b.message.WriteRune(r)
				}
				b.message.WriteString(b.draft.String())
				b.draft.Reset()
			}
			b.mu.Unlock()
		case unicode.IsControl(r):
			// drop
		case unicode.IsSpace(r):
			b.mu.Lock()
			b.draft.WriteByte(' ')
			b.mu.Unlock()
		}
		prev = r
	}
}

// Len - returns length (in bytes) of whole filtered string, excluding unfiltered remainder part.
func (b *Builder) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.message.Len() + b.draft.Len()
}

// MessageLen - returns length of filtered message. Values is always less or equals than whole length.
func (b *Builder) MessageLen() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.message.Len()
}

// Flush - immediately returns the whole filtered string and resets all internal buffers.
func (b *Builder) Flush() string {
	b.mu.Lock()
	defer func() {
		b.draft.Reset()
		b.message.Reset()
		b.mu.Unlock()
	}()
	if b.message.Len() > 0 && b.draft.Len() > 0 {
		b.message.WriteByte('\n')
	}
	b.message.WriteString(b.draft.String())
	return b.message.String()
}

// FlushMessage - returns filtered message without of the buffered remainder and resets message buffer.
// The buffered string is became a message if builder meets Unix EOL in the unfiltered source.
func (b *Builder) FlushMessage() string {
	b.mu.Lock()
	defer func() {
		b.message.Reset()
		b.mu.Unlock()
	}()
	return b.message.String()
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
