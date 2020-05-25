#!/bin/bash

set -euo pipefail

# poll indefinitely
while true; do
    # update our repo
    git pull && git pull --tags

    # check if we're on a tag
    if git describe --tags --exact-match HEAD; then
        tag=$(git describe --exact-match HEAD)
        artifact="sprig-$tag"
        # check if we are on a new tagged commit
        if ! [ -e "$artifact" ]; then
            echo "building tag $tag"
            env GOOS=darwin go build -o "$artifact" .
            if ! curl -H "Authorization: token $SRHT_TOKEN" -F "file=@$artifact" "https://git.sr.ht/api/repos/sprig/artifacts/$tag" ; then
                echo "upload failed"
            fi
        fi
    fi

    # sleep 15 minutes
    sleep 900
done
