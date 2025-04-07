#!/usr/bin/env bash

set -euo pipefail

# This script prints a nice-ish version based on an available tag and commit id. It deals with all these cases
#
# 1. It only uses tags that match the $pattern. This takes care of cases where a commit has more than one tag
# 2. If there are no tags associated with the current HEAD, it returns the current head with a "dev" suffix
# 3. If there's more than one tag on the current head, it returns the "highest" version as per `sort --version-sort`

pattern="$1"
commit_id="$( git rev-parse --short HEAD )"
tag="$(
    git tag --points-at "$commit_id" | \
    awk -F/ "/$pattern/{ print \$2 }" | \
    sort --version-sort | \
    tail --lines 1
)"

if [ -n "$tag" ]; then
        echo "$tag"
else
        echo "$commit_id-dev"
fi
