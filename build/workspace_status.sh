#!/bin/sh

# Copyright 2020 Codelogia
#
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

set -o errexit -o nounset -o pipefail

noversionstr="0.0.0"
tag=$(git describe --tags --abbrev=0 2> /dev/null || echo "${noversionstr}")
version="${tag}"

if [ "${version}" != "${noversionstr}" ]; then
  # If the tag doesn't point to HEAD, it's a pre-release.
  if [ -z "$(git tag --points-at HEAD 2> /dev/null)" ]; then
    # The commit timestamp should be in the format yyyymmddHHMMSS in UTC.
    git_commit_timestamp=$(
      git show --no-patch --format="%at" HEAD \
      | awk '{ print strftime("%Y%m%d%H%M%S", $0) }')

    # The number of commits since last tag that points to a commits in the
    # branch.
    git_number_commits=$(git rev-list --count ${version}..HEAD)

    # Add `g` to the short hash to match git describe.
    git_commit_short_hash="g$(git rev-parse --short=8 HEAD)"

    # The version gets assembled with the pre-release part.
    version="${version}-${git_commit_timestamp}.${git_number_commits}.${git_commit_short_hash}"
  fi
fi

# If there's a change in the source tree that didn't get committed, append
# `-dirty` to the version string.
if [ -n "$(git status --short 2> /dev/null)" ]; then
  version="${version}-dirty"
fi

echo "STABLE_VERSION ${version}"
