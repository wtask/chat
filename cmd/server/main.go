package main

import "github.com/wtask/chat/pkg/semver"

var (
	// Version - application version fingerprint
	Version = semver.V{Minor: 1, PreRelease: "prototype"}.String()
)

func main() {

}
