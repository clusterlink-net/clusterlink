#!/bin/bash

# Copyright 2024 The ClusterLink Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# gen-doc-version is used by the "make docs-version" target in Makefile.
# It generates a new versioned documentation directory under website/content/en/docs
# using the following process:
#   1. Create a new directory for `$NEW_DOCS_VERSION`
#   2. Copy and git commit the contents of the last released docs directory
#      (`$PREVIOUS_DOCS_VERSION``) into the new directory, to establish a baseline
#      for documentation comparison.
#   3. Delete and replaces the contents of the new docs directory with the
#      contents of the 'main' docs directory. 
#   4. Update and version and/or revision specific value in the documentation.
#
# The unstaged changes in the working directory can now easily be diff'ed 
# using 'git diff' to review all docs changes made since the previous
# released version. Once the unstaged changes are ready, they can be added
# and committed.
#
# To run gen-doc-version: "NEW_DOCS_VERSION=v0.2.0 PREVIOUS_DOCS_VERSION=v0.1.0 make docs-version"
# Note: if PREVIOUS_DOCS_VERSION is not set, the script will guess it from the directory listing
#

set -o errexit
set -o nounset
set -o pipefail

DOCS_DIRECTORY=website/content/en/docs
CONFIG_FILE=website/config.toml
MAIN_BRANCH=main
INDEX_MD=_index.md
RELEASES_LATEST=releases/latest
DOWNLOADS_LATEST=releases/latest/download

# NEW_DOCS_VERSION must be defined
if [[ -z "${NEW_DOCS_VERSION:-}" ]]; then
    echo "ERROR: \$NEW_DOCS_VERSION environment variable must be defined"
    exit 1
fi 

# don't run if there's already a directory for the target docs version
if [[ -d ${DOCS_DIRECTORY}/${NEW_DOCS_VERSION} ]]; then
    echo "ERROR: $DOCS_DIRECTORY/$NEW_DOCS_VERSION already exists"
    exit 1
fi

# get the alphabetically last item in $DOCS_DIRECTORY to use as PREVIOUS_DOCS_VERSION
# if not explicitly specified by the user
if [[ -z "${PREVIOUS_DOCS_VERSION:-}" ]]; then
    echo "PREVIOUS_DOCS_VERSION was not specified, getting the latest version"
    PREVIOUS_DOCS_VERSION=$(ls -1 $DOCS_DIRECTORY/ | tail -n 1)
fi

# make a copy of the previous versioned docs dir
git checkout -b ${NEW_DOCS_VERSION}
echo "Creating copy of docs directory ${DOCS_DIRECTORY}/${PREVIOUS_DOCS_VERSION} in ${DOCS_DIRECTORY}/${NEW_DOCS_VERSION}"
cp -r ${DOCS_DIRECTORY}/${PREVIOUS_DOCS_VERSION}/ ${DOCS_DIRECTORY}/${NEW_DOCS_VERSION}/

# Copy the previous version's docs as-is so we get a useful diff when we copy the $MAIN_BRANCH docs in
echo "Running 'git add' for previous version's doc contents to use as a base for diff"
git add -f ${DOCS_DIRECTORY}/${NEW_DOCS_VERSION}

# Delete all and then copy the contents of $DOCS_DIRECTORY/$MAIN_BRANCH into
# the same directory so we can get git diff of what changed since previous version
echo "Copying ${DOCS_DIRECTORY}/${MAIN_BRANCH}/ to ${DOCS_DIRECTORY}/${NEW_DOCS_VERSION}/"
rm -rf ${DOCS_DIRECTORY}/${NEW_DOCS_VERSION}/ && cp -r ${DOCS_DIRECTORY}/${MAIN_BRANCH}/ ${DOCS_DIRECTORY}/${NEW_DOCS_VERSION}/

# replace known version-specific links
echo "Updating config file"
sed -i'' "s/latest_stable_version = ${PREVIOUS_DOCS_VERSION}/latest_stable_version = ${NEW_DOCS_VERSION}/" $CONFIG_FILE
echo "Updating release to use git_version_tag"
# note that order of pattern replacements matters: 
# 1. releases/latest/downlad to releases/download/$GIT_TAG (so it no longer matches /releases/latest in next match)
# 2. releases/latest to releases/tag/$GIT_TAG
for f in $(grep -rl ${DOWNLOADS_LATEST} ${DOCS_DIRECTORY}/${NEW_DOCS_VERSION})
do
    sed -i'' "s|latest/download|download/{{% param git_version_tag %}}|g" ${f}
done
for f in $(grep -rl ${RELEASES_LATEST} ${DOCS_DIRECTORY}/${NEW_DOCS_VERSION})
do
    sed -i'' "s|latest|tag/{{% param git_version_tag %}}|g" ${f}
done

# TODO: automate some of the below
echo "Done - $DOCS_DIRECTORY/$NEW_DOCS_VERSION has been created"
echo ""
echo "1. Run a 'git status' / 'git diff' to review all changes made to the docs since the previous version."
echo "2. Make any manual changes/corrections necessary. For example:"
echo "   - Remove docs directories of deprecated/EOL versions, if any."
echo "   - Revert latest_stable_version in $CONFIG_FILE for non-GA releases."
echo "   - Add version ${NEW_DOCS_VERSION} to [params.versions] list in ${CONFIG_FILE} (after main, above ${PREVIOUS_DOCS_VERSION})."
echo "   - Fix frontmatter in $DOCS_DIRECTORY/${NEW_DOCS_VERSION}/${INDEX_MD}."
echo "3. Run 'git add' to stage all unstaged changes, then 'git commit'."
