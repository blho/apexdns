// Package version used to specify whole repo version. related version variable are inject via ldflags
package version

import (
	"fmt"
	"runtime"
)

// Those variables are mostly set via ldflags while building
var (
	// semantic version for matrix
	version string
	// sha1 from git, output of $(git rev-parse HEAD)
	gitCommit string
	// build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
	buildDate = "1970-01-01T00:00:00Z"
)

// Info of version
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"GoVersion"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

// String formatted
func (i Info) String() string {
	return fmt.Sprintf("version: %s, gitCommit: %s, buildData: %s, GoVersion: %s, compiler: %s, platform: %s",
		i.Version, i.GitCommit, i.BuildDate, i.GoVersion, i.Compiler, i.Platform)
}

// Get version info
func Get() Info {
	return Info{
		Version:   version,
		GitCommit: gitCommit,
		BuildDate: buildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
