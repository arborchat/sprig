## sprig

Sprig is the [Arbor](https://arbor.chat) reference chat client. 


![sprig screenshot](https://git.sr.ht/~whereswaldon/sprig/blob/main/img/screenshot.png)

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
