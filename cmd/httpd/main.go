package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/samthom/text-search/cmd/handlers"
	"github.com/samthom/text-search/lib/db"
	"github.com/samthom/text-search/lib/file"
	"github.com/samthom/text-search/lib/storage"
	log "github.com/sirupsen/logrus"
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("content-type", "application/json"))
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	tpl := template.Must(template.ParseFiles("./index.html"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "text/html")
		err := tpl.Execute(w, nil)
		if err != nil {
			log.Error(err)
		}
		// _, e := w.Write([]byte(`{ "message" : "welcome to Searchr." }`))
		// if e != nil {
		// 	log.Error("index controller failed.")
		// }
	})

	// Create storage object
	storageOpts := &minio.Options{
		// Creds:  credentials.NewStaticV4("searchr", "searchrpwd", ""),
		Creds:  credentials.NewStaticV4("ROOT", "PASSWORD", ""),
		Secure: false,
	}
	st, err := storage.NewStorage("localhost:9000", "searchr", storageOpts)
	if err != nil {
		log.Fatal(err)
	}

	// Create Parse object
	parser := file.NewParser(nil)

	// create index
	d := db.NewDB(nil, "searchr")
	idx, err := db.NewIndex(d)
	if err != nil {
		log.Fatal(err)
	}

	// Create SearchrHandler object
	h := handlers.NewSearchrHandler(st, parser, idx)

	// Routes
	r.Post("/upload", h.Upload())
	r.Get("/search", h.Search())

	// Server code
	s := &http.Server{
		Addr:         ":2112",
		Handler:      r,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
	}

	go func() {
		err := s.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)

	sig := <-sigChan
	log.Warning("Recieved terminate, graceful shutdown ", sig)
	tc, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer func() {
		cancel()
	}()
	err = s.Shutdown(tc)
	if err != nil {
		panic("Shutdown Panic")
	}

}
