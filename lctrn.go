package lctrn

import (
	"context"
	"embed"
	_ "embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/chromedp/chromedp"
)

type App struct {
	WindowWidth     int  // default 1200
	WindowHeight    int  // default 800
	StartFullscreen bool // default false

	ctx       context.Context
	cancel    context.CancelCauseFunc
	Mux       *http.ServeMux
	popupFunc func(title, msg string) error
}

func New() *App {
	ctx, cancel := context.WithCancelCause(context.Background())

	return &App{
		WindowWidth:  1200,
		WindowHeight: 800,

		ctx:    ctx,
		cancel: cancel,
		Mux:    http.NewServeMux(),
	}
}

func (a *App) ServeEmbeddedFolderAtRoot(embeddedFolder embed.FS) *App {
	folder := getTopLevel(embeddedFolder)
	a.Mux.Handle("/", http.FileServer(http.FS(folder)))
	return a
}

func (a *App) ServeBytesAtRoot(root []byte) *App {
	a.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(root)
	})
	return a
}

func (a *App) Handle(pattern string, handler http.Handler) *App {
	a.Mux.Handle(pattern, handler)
	return a
}

func (a *App) HandleFunc(pattern string, handler func(w http.ResponseWriter, r *http.Request)) *App {
	a.Mux.HandleFunc(pattern, handler)
	return a
}

func (a *App) UseMux(mux *http.ServeMux) *App {
	a.Mux = mux
	return a
}

// If root contains exactly one top-level dir and nothing else, substitute it..
func getTopLevel(root fs.FS) fs.FS {
	entries, err := fs.ReadDir(root, ".")

	if err != nil {
		return root
	}

	if len(entries) != 1 {
		return root
	}

	entry := entries[0]

	if !entry.IsDir() {
		return root
	}

	newRoot, err := fs.Sub(root, entry.Name())

	if err != nil {
		return root
	}

	return newRoot
}

// Runs the app. Any errors will log, as well as showing a platform-specific popup
func (a *App) Launch() {
	if err := a.Run(); err != nil {
		a.Popup("Error", err.Error())
		log.Fatalln(err)
	}
}

func (a *App) Run() error {
	errc := make(chan error, 1)

	listener, err := net.Listen("tcp", ":0") // random port
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		errc <- a.startServer(listener)
	}()

	if err := a.startChrome(port); err != nil {
		a.Close()
		<-errc
		return err
	}

	return <-errc
}

func (a *App) startServer(ln net.Listener) error {
	log.Println("starting server...")

	srv := &http.Server{
		Handler: a.Mux,
	}

	go func() {
		<-a.ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	if err := srv.Serve(ln); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (a *App) startChrome(port int) error {
	log.Println("starting chrome...")

	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("app", fmt.Sprintf("http://localhost:%v", port)),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.WindowSize(a.WindowWidth, a.WindowHeight),
	}

	if a.StartFullscreen {
		opts = append(opts, chromedp.Flag("start-fullscreen", true))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(a.ctx, opts...)
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))

	err := chromedp.Run(browserCtx,
		chromedp.Navigate(fmt.Sprintf("http://localhost:%v", port)),
		chromedp.WaitReady("body"),
	)

	if err != nil {
		cancelBrowser()
		cancelAlloc()
		return fmt.Errorf("chrome launch failed: %w", err)
	}

	log.Println("chrome opened")

	go func() {
		<-browserCtx.Done()
		cancelBrowser()
		cancelAlloc()
		a.Close()
	}()

	return nil
}

func (a *App) Close() {
	a.cancel(fmt.Errorf("app closed"))
}
