package handlers

import (
	"context"
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

	// Watch for changes (Job)
	Watch()
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
			http.Error(rw, "Unable to parse file from request", http.StatusBadRequest)
			return
		}
		defer f.Close()

		// Upload file
		_, err = h.storage.Add(r.Context(), handler.Filename, f, handler.Size, minio.PutObjectOptions{
			ContentType: handler.Header.Get("Content-Type"),
		})
		if err != nil {
			log.Error(err)
			http.Error(rw, "Unable to upload file", http.StatusInternalServerError)
			return
		}

		// // parse file
		// body, err := h.parser.ParseFile(r.Context(), f)
		// // body, err := h.parser.ParseFileReader(r.Context(), f)
		// if err != nil {
		// 	log.Error(err)
		// 	http.Error(rw, "Unable to parse content from file", http.StatusInternalServerError)
		// 	return
		// }

		// URL := h.storage.GetURL(handler.Filename)
		// // add index
		// fl := db.NewFile(ETag, URL, handler.Filename, &body, handler.Size)
		// err = h.index.Save(fl)
		// if err != nil {
		// 	log.Error(err)
		// 	http.Error(rw, "Unable to index the file", http.StatusInternalServerError)
		// 	return
		// }
		rw.WriteHeader(http.StatusCreated)
	}
}

func (h *searchrHandlerstr) Search() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		search := r.URL.Query().Get("file")
		search = "@body:" + search

		docs, err := h.index.Find(search, 2, "key", "size", "url")
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

func (h *searchrHandlerstr) Watch() {
	log.Info("Watching for uploads in storage.")
	ch := h.storage.Watch([]string{"s3:ObjectCreated:*"})

	for info := range ch {
		for _, event := range info.Records {
			objectInfo := event.S3.Object
			file, err := h.storage.Get(context.TODO(), objectInfo.Key)
			if err != nil {
				continue
			}
			body, err := h.parser.ParseFile(context.TODO(), file)
			if err != nil {
				continue
			}
			URL := h.storage.GetURL(objectInfo.Key)
			// add index
			fl := db.NewFile(objectInfo.ETag, URL, objectInfo.Key, &body, objectInfo.Size)
			err = h.index.Save(fl)
			if err != nil {
				continue
			}
			log.Info(objectInfo.Key, " - indexed")
		}
	}
}
