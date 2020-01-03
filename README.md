## sprig

Sprig is a highly experimental [Arbor](https://arbor.chat) chat client. It currently is readonly and missing most of the features you would expect from a client. It has been tested on Linux and Android. In theory, it should work across mainstream OSes and on iOS, but I actually can't test that.

### Try it

To give it a shot on desktop, install [go 1.13+](https://golang.org/dl).

Then make sure you have the [gio dependencies](https://gioui.org/doc/install#linux) for your current OS.

Finally, run:

```
env GO111MODULE=on go run git.sr.ht/~whereswaldon/sprig
```

To run on android, clone this repo and run:

```
make android_install
```

You'll need a functional android development toolchain for that to work.
