## sprig

Sprig is an [Arbor](https://arbor.chat) chat client focused on mobile devices. 

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
