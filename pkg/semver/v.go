package semver

import (
	"strconv"
	"strings"
)

type (
	// V is structured semantic version representation
	V struct {
		Major, Minor, Patch uint
		PreRelease          string
		BuildMetadata       []string
	}
)

func (v V) String() string {
	buf := strings.Builder{}
	buf.WriteString(strconv.FormatUint(uint64(v.Major), 10))
	buf.WriteByte('.')
	buf.WriteString(strconv.FormatUint(uint64(v.Minor), 10))
	buf.WriteByte('.')
	buf.WriteString(strconv.FormatUint(uint64(v.Patch), 10))
	if v.PreRelease != "" {
		buf.WriteByte('-')
		buf.WriteString(v.PreRelease)
	}
	if len(v.BuildMetadata) > 0 {
		buf.WriteByte('+')
		buf.WriteString(strings.Join(v.BuildMetadata, "."))
	}

	return buf.String()
}
