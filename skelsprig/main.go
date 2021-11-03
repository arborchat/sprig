package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"gioui.org/app"
	"git.sr.ht/~gioverse/skel/scheduler"
	"git.sr.ht/~gioverse/skel/window"
	"git.sr.ht/~whereswaldon/sprig/skelsprig/pages"
	"git.sr.ht/~whereswaldon/sprig/skelsprig/settings"
)

// Suffix is joined to the path for convenience.
func getDataDir(suffix string) (string, error) {
	d, err := app.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, suffix), nil
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	var (
		dataDir    string
		invalidate bool
	)

	dataDir, err := getDataDir("sprig")
	if err != nil {
		log.Printf("finding application data dir: %v", err)
	}

	flag.BoolVar(&invalidate, "invalidate", false, "invalidate every single frame, only useful for profiling")
	flag.StringVar(&dataDir, "data-dir", dataDir, "application state directory")
	flag.Parse()

	// handle ctrl+c to shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	bus := scheduler.NewWorkerPool()

	_, err = settings.New(dataDir, bus.Connect())
	if err != nil {
		log.Fatalf("Failed initializing settings: %v", err)
	}

	w := window.NewWindower(bus)
	go func() {
		// Launch a goroutine to supervise window lifecycles.
		w.Run()
		os.Exit(0)
	}()
	go func() {
		// Launch a goroutine that will shut down the application when
		// we get SIGINT.
		<-sigs
		w.Stop()
	}()

	// Create a new application window.
	w.Wait()
	window.NewWindow(bus, pages.Window, app.Title("Sprig"))
	app.Main()
}
