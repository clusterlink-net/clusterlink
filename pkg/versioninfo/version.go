// Copyright (c) The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package versioninfo

import (
	"runtime/debug"
	"strings"
	"time"
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			Revision = kv.Value
		case "vcs.time":
			LastCommit, _ = time.Parse(time.RFC3339, kv.Value) //nolint:errcheck // ignore last commit not being a valid time
		case "vcs.modified":
			DirtyBuild = kv.Value == "true"
		}
	}
}

// Short provides a short string summarizing available version information.
// The format is <SemVer>-<GIT SHA>[-<dirty>].
func Short() string {
	parts := make([]string, 0, 4)
	if GitTag != "" {
		parts = append(parts, GitTag)
	}
	if Version != "unknown" && Version != "(devel)" {
		parts = append(parts, Version)
	}
	if Revision != "unknown" && Revision != "" {
		parts = append(parts, "rev")
		commit := Revision
		if len(commit) > 7 {
			commit = commit[:7]
		}
		parts = append(parts, commit)
		if DirtyBuild {
			parts = append(parts, "dirty")
		}
	}

	if len(parts) == 0 {
		return "devel"
	}
	return strings.Join(parts, "-")
}
