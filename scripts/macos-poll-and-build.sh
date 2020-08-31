#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$(cd $(dirname "$0") && pwd)
REPO_ROOT="$SCRIPT_DIR/.."

function build_for_mac() {
    local -r artifact
    artifact=$1
    make macos
    mv sprig-mac.tar.gz $artifact
}

function build_for_tag() {
    local -r tag
    tag="$1"
    artifact="sprig-$tag-macOS.tar.gz"
    # check if we are on a new tagged commit
    if ! [ -e "$artifact" ]; then
        echo "building tag $tag"
        if ! build_for_mac "$artifact"; then
            return 1
        fi
        if ! curl --http1.2 -H "Authorization: token $SRHT_TOKEN" \
        	-F "file=@$artifact" "https://git.sr.ht/api/repos/sprig/artifacts/$tag" ; then
            echo "upload failed"
            return 2
        fi
    fi
}

# poll indefinitely
while true; do
    cd "$REPO_ROOT"
    # update our repo
    git fetch --tags

    # check if we're on a tag
    for tag in $(git tag); do
        git checkout "$tag"
        if ! build_for_tag "$tag"; then echo "failed building for tag $tag"; fi
    done

    # sleep 15 minutes
    sleep 900
done
