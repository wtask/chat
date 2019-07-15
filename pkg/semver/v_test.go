package semver

import "testing"

func TestV_String(test *testing.T) {
	cases := []struct {
		v        V
		expected string
	}{
		{V{}, "0.0.0"},
		{V{Major: 1}, "1.0.0"},
		{V{Major: 1, Minor: 2}, "1.2.0"},
		{V{Major: 1, Minor: 2, Patch: 3}, "1.2.3"},
		{V{PreRelease: "alfa"}, "0.0.0-alfa"},
		{V{BuildMetadata: []string{"tag1", "tag2"}}, "0.0.0+tag1.tag2"},
		{V{Major: 1, Minor: 2, Patch: 3, PreRelease: "beta", BuildMetadata: []string{"x64"}}, "1.2.3-beta+x64"},
	}

	for _, c := range cases {
		test.Logf("%#v", c.v)
		actual := c.v.String()
		if actual != c.expected {
			test.Errorf("Error: expected %q, actual %q", c.expected, actual)
		}
	}
}
