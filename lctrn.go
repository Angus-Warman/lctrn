package lctrn

import (
	"context"
	"embed"
	_ "embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

type App struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	mux    *http.ServeMux
}

func (a *App) Close() {
	a.cancel(fmt.Errorf("app closed"))
}

func FromMux(mux *http.ServeMux) *App {
	ctx, cancel := context.WithCancelCause(context.Background())

	return &App{
		ctx:    ctx,
		cancel: cancel,
		mux:    mux,
	}
}

func FromEmbeddedFolder(embeddedFolder embed.FS) *App {
	folder := getTopLevel(embeddedFolder)
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(folder)))
	return FromMux(mux)
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

func (a *App) Run() error {
	errc := make(chan error, 1)

	port := os.Getenv("LCTRN_PORT")
	if port == "" {
		port = "8778"
	}

	go func() {
		errc <- a.startServer(port)
	}()

	if err := a.startChrome(port); err != nil {
		a.Close()
		<-errc
		return err
	}

	return <-errc
}

func (a *App) startServer(port string) error {
	log.Println("starting server...")

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: a.mux,
	}

	go func() {
		<-a.ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (a *App) startChrome(port string) error {
	log.Println("starting chrome...")

	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("app", fmt.Sprintf("http://localhost:%v", port)),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.WindowSize(1200, 800),
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
