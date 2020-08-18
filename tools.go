//+build tools

package main

import _ "gioui.org/cmd/gogio"

/*
This file locks gogio as a dependency so that its version will
stay in sync with the version of gio that we use in our go.mod.
*/
