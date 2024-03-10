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
	"time"
)

var (
	// GitTag will hold the Git tag information.
	GitTag string
	// Version will be the version tag if the binary is built or "(devel)".
	Version = "unknown"
	// Revision is taken from the vcs.revision tag.
	Revision = "unknown"
	// LastCommit is taken from the vcs.time tag.
	LastCommit time.Time
	// DirtyBuild is taken from the vcs.modified tag.
	DirtyBuild = false
)
