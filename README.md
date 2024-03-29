## sprig

Sprig is the [Arbor](https://arbor.chat) reference chat client.

![sprig screenshot](https://git.sr.ht/~whereswaldon/sprig/blob/main/img/screenshot.png)

### Try it

To give it a shot on desktop, install [go 1.18+](https://golang.org/dl).

Then make sure you have the
[gio dependencies](https://gioui.org/doc/install#linux) for your current OS.

Run:

```
# install a build system tool
go install github.com/magefile/mage@latest
# clone the source code
git clone https://git.sr.ht/~whereswaldon/sprig
# enter the source code directory
cd sprig
```

Then issue a build for the platform you're targeting by executing one of these:

- `windows`: `make windows`
- `macos`: `make macos` (only works from a macOS computer)
- `linux`: `make linux`
- `android`: `make android` (requires android development environment)
- `ios`: `make ios` (only works from a macOS computer)

After running `make`, there should be an archive file containing a build for the
target platform in your current working directory.

For android in particular, you can automatically install it on a plugged-in
device (in developer mode) with:

```
make android_install
```

You'll need a functional android development toolchain for that to work.
