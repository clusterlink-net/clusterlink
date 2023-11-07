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

var (
	// SemVer will be the version specified in version.txt file
	SemVer = strings.TrimSpace(semver)
	//go:embed version.txt
	semver string

	// Version will be the version tag if the binary is built with "go install url/tool@version".
	// If the binary is built some other way, it will be "(devel)".
	Version = "unknown"
	// Revision is taken from the vcs.revision tag in Go 1.18+.
	Revision = "unknown"
	// LastCommit is taken from the vcs.time tag in Go 1.18+.
	LastCommit time.Time
	// DirtyBuild is taken from the vcs.modified tag in Go 1.18+.
	DirtyBuild = true
)
