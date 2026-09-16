## lctrn

*lctrn* is a way to build lightweight cross-platform desktop apps, by fully relying on installed Chrome instances. It uses the [Chrome DevTools Protocol](https://github.com/chromedp/chromedp) to handle the interop.

If Chrome is not installed, *lctrn* will attempt to show an error popup using whatever tools are available, falling back on writing a .txt file and opening that.

### Getting started

```go
package main

import "github.com/Angus-Warman/lctrn"

func main() {
	lctrn.New().ServeBytesAtRoot([]byte("hello world")).Launch()
}
```

See the [demo](/demo/main.go) for a more involved example. 

For existing web-apps, *lctrn* accepts an HTTP mux.
