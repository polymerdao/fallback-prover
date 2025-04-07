package fallback_prover

import (
	"fmt"
	"runtime/debug"
	"strings"
)

// Version is the current version of the native-proof.
var Version = "v0.0.1"

// Meta contains extra metadata to be shown in the version info.
var Meta = "dev"

// FormattedVersion formats the version string with git commit and date.
func FormattedVersion(gitCommit string, gitDate string) string {
	var commitMetadata string
	if gitCommit != "" {
		commitMetadata = fmt.Sprintf("-commit%s", gitCommit)
	}
	if gitDate != "" {
		if commitMetadata != "" {
			commitMetadata += "-"
		}
		commitMetadata += fmt.Sprintf("date%s", gitDate)
	}
	if Meta != "" {
		if commitMetadata != "" {
			commitMetadata += "-"
		}
		commitMetadata += Meta
	}
	if commitMetadata != "" {
		commitMetadata = "+" + commitMetadata
	}

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		if strings.TrimSpace(commitMetadata) == "" {
			for _, setting := range buildInfo.Settings {
				if strings.HasPrefix(setting.Key, "vcs") {
					commitMetadata = "+" + strings.TrimPrefix(setting.Key, "vcs") + setting.Value
				}
			}
		}
	}
	return Version + commitMetadata
}
