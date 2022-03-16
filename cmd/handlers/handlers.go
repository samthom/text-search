package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/samthom/text-search/lib/db"
	"github.com/samthom/text-search/lib/file"
	"github.com/samthom/text-search/lib/storage"
	log "github.com/sirupsen/logrus"
)

type SearchrHandlers interface {
	// Upload
	Upload() http.HandlerFunc
	// Search
	Search() http.HandlerFunc
}

type searchrHandlerstr struct {
	storage storage.Storage
	parser  file.Parser
	index   db.Index
}

func NewSearchrHandler(storage storage.Storage, parser file.Parser, index db.Index) SearchrHandlers {
	return &searchrHandlerstr{storage, parser, index}
}

func (h *searchrHandlerstr) Upload() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// parse our request as multipart form, size limiter 10MB
		r.ParseMultipartForm(10 << 20)

		f, handler, err := r.FormFile("file")
		if err != nil {
			log.Error(err)
			http.Error(rw, "Unable to parse file from request", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		// Upload file
		ETag, err := h.storage.Add(r.Context(), handler.Filename, f, handler.Size, minio.PutObjectOptions{
			ContentType: handler.Header.Get("Content-Type"),
		})
		if err != nil {
			log.Error(err)
			http.Error(rw, "Unable to upload file", http.StatusInternalServerError)
			return
		}

		// parse file
		body, err := h.parser.ParseFile(r.Context(), f)
		// body, err := h.parser.ParseFileReader(r.Context(), f)
		if err != nil {
			log.Error(err)
			http.Error(rw, "Unable to parse content from file", http.StatusInternalServerError)
			return
		}

		// add index
		fl := db.NewFile(ETag, handler.Filename, &body, handler.Size)
		err = h.index.Save(fl)
		if err != nil {
			log.Error(err)
			http.Error(rw, "Unable to index the file", http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusCreated)
	}
}

func (h *searchrHandlerstr) Search() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		search := r.URL.Query().Get("file")

		docs, err := h.index.Find(search, 2, "key", "size")
		if err != nil {
			log.Error(err)
			http.Error(rw, "Unable search for file", http.StatusInternalServerError)
			return
		}
		if len(docs) == 0 {
			rw.WriteHeader(http.StatusNoContent)
		}
		json.NewEncoder(rw).Encode(docs)
	}
}
