#!/bin/bash

set -euo pipefail

BASEDIR=$(dirname "$(realpath "$0")")

PREFIX=${PREFIX:-${XDG_DATA_HOME:-$HOME/.local/share}}

cp -v "$BASEDIR/desktop-assets/sprig.desktop" "$PREFIX/applications/"
cp -v "$BASEDIR/appicon.png" "$PREFIX/icons/sprig.png"
