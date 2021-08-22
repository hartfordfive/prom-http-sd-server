package version

import (
	"fmt"
	"strconv"
)

var (
	Version        string
	BuildDate      string
	CommitHash     string
	Author         = "Alain Lefebvre <hartfordfive@gmail.com>"
	symbolsEnabled string
)

// PrintVersion returns the current version information
func PrintVersion() {
	fmt.Printf("Version %s, Date: %s, Commit: %s\nAuthor: %s\n", Version, BuildDate, CommitHash, Author)
	if b, _ := strconv.ParseBool(symbolsEnabled); b {
		fmt.Printf("Debug symbols enabled: %v\n", b)
	}
}
