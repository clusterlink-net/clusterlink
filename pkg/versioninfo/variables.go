// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package versioninfo uses Go's runtime/debug to set executable revision information.
package versioninfo

import (
	_ "embed"
	"strings"
	"time"
)

// @TODO
// The placement of version.txt inside this package is not ideal.
// Unfortunately Go does not allow ".." in embed paths. A possible workaround is to
// use "go:generate" comment to copy the file from the root to this location (and adding
// the local version to .gitignore...)

// An alternative to using a fixed file, would be to use a go:generate comment with
// "git describe" (git describe output is something like v1.1-19-g175c485, where v1.1
// is the most recent annotated tag, 19 is the number of commits since that tag, g stands
// for "git," and 175c485 is the short commit hash.
// Build date can also be added to the version with "go:generate date +%F"
var (
	//go:embed version.txt
	semver string
	// SemVer will be the version specified in version.txt file
	SemVer = strings.TrimSpace(semver)
)

var (
	// Version will be the version tag if the binary is built or "(devel)".
	Version = "unknown"
	// Revision is taken from the vcs.revision tag.
	Revision = "unknown"
	// LastCommit is taken from the vcs.time tag.
	LastCommit time.Time
	// DirtyBuild is taken from the vcs.modified tag.
	DirtyBuild = true
)
