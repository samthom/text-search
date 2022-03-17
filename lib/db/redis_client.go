package db

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/RediSearch/redisearch-go/redisearch"
)

// DB interface with required methods for our service
// @TODO: need to add remove method for deleting
type DB interface {
	Add(doc []redisearch.Document) (err error)
	Get(key string) (result *redisearch.Document, err error)
	Search(search string, offset int, limit int, selectors ...string) (docs []redisearch.Document, total int, err error)
	CreateSchema(sc *redisearch.Schema) (ok bool, err error)
}

// dbClient struct is the concrete type implementing the DB interface
type dbClient struct {
	Client *redisearch.Client
	Index  string
}

// NewDB function to create new client with index
// use nil as url for default url. This will not create an index. Use CreateSchema to create index if not exist
func NewDB(url *string, index string) DB {
	if url == nil {
		return &dbClient{redisearch.NewClient("localhost:6379", index), index}
	}
	return &dbClient{redisearch.NewClient(*url, index), index}
}

// Add method takes slice of redisearch Doc type
// Method adds new doc to the index
func (db *dbClient) Add(docs []redisearch.Document) error {
	return db.Client.Index(docs...)
}

// Search method takes search query in "@field:string" format, offset starting position to paginate
// takes limit and selector as slice of string
// returns the document list, total count and error
func (db *dbClient) Search(search string, offset int, limit int, selectors ...string) ([]redisearch.Document, int, error) {
	return db.Client.Search(redisearch.NewQuery(search).
		Limit(offset, limit).
		SetReturnFields(selectors...))
}

func (db *dbClient) Get(key string) (*redisearch.Document, error) {
	return db.Client.Get(key)
}

// CreateSchema method checks if the exact same index exists
// if not then creates new index with specified schema and returns bool
func (db *dbClient) CreateSchema(sc *redisearch.Schema) (bool, error) {
	idx, err := db.Client.Info()
	if err != nil && err.Error() == "Unknown Index name" {
		if err = db.Client.CreateIndex(sc); err != nil {
			return false, err
		}
		return true, nil
	} else if idx != nil && idx.Name == db.Index {
		return true, nil
	}
	return false, nil
}

type Index interface {
	// Create idx method to create a index. Called initially to check and create index if not exist.
	// This will create schema if not exist
	Create() (ok bool, err error)
	Save(f File) (err error)
	Get(key string) (file File, err error)
	Find(key string, limit int, fields ...string) (files []File, err error)
}

type idx struct {
	DB
}

// NewIndex function takes DB type and creates the schema
func NewIndex(db DB) (Index, error) {
	i := &idx{db}
	ok, err := i.Create()
	if err != nil || !ok {
		if err == nil && !ok {
			return nil, errors.New("unable to create index in DB")
		}
		return nil, err
	}
	return i, nil
}

// fileSchema - File data model to be stored inside the redis cache for indexing
var fileSchema = redisearch.NewSchema(redisearch.DefaultOptions).
	AddField(redisearch.NewTextField("body")).
	AddField(redisearch.NewTextField("url")).
	AddField(redisearch.NewTextField("key")).
	AddField(redisearch.NewNumericField("created_at")).
	AddField(redisearch.NewNumericFieldOptions("size", redisearch.NumericFieldOptions{NoIndex: false, Sortable: true}))

// create schema and index
// returns false if not created
func (i *idx) Create() (bool, error) {
	return i.CreateSchema(fileSchema)
}

// Save method for saving a file into the cache/index
func (i *idx) Save(f File) error {
	// create redis doc
	doc := redisearch.NewDocument(f.GetETag(), 1.0)
	doc.Set("key", f.GetKey()).
		Set("url", f.GetURL()).
		Set("body", f.GetBody()).
		Set("size", f.GetSize()).
		Set("created_at", time.Now().Unix())
	return i.Add([]redisearch.Document{doc})
}

// Find method searches the index and return selected fields
// @BUG There might be bug with the way we unmarshalling the data from the response
func (i *idx) Find(key string, limit int, fields ...string) ([]File, error) {
	d, total, err := i.Search(key, 0, limit, fields...)
	if err != nil {
		return nil, err
	}
	f := []File{}
	if total == 0 {
		return f, nil
	}
	for _, v := range d {
		sizeStr := v.Properties["size"].(string)
		size, _ := strconv.Atoi(sizeStr)
		file := &fileInfo{
			ETag: v.Id,
			Key:  v.Properties["key"].(string),
			URL:  v.Properties["url"].(string),
			Size: int64(size),
		}
		f = append(f, file)
	}
	return f, nil
}

// Get method to get a single doc using the key used to save
func (i *idx) Get(key string) (File, error) {
	d, err := i.DB.Get(key)
	if err != nil {
		return nil, err
	}
	if d != nil {
		sizeStr := d.Properties["size"].(string)
		size, _ := strconv.Atoi(sizeStr)
		file := &fileInfo{
			ETag: d.Id,
			URL:  d.Properties["url"].(string),
			Key:  d.Properties["key"].(string),
			Size: int64(size),
		}
		return file, nil
	}
	return nil, nil
}

// File interface
// For basic operations with concorete type
type File interface {
	Marshal() ([]byte, error)
	UnMarshal(data []byte) error
	GetETag() string
	GetURL() string
	GetKey() string
	GetBody() string
	GetSize() int64
}

// fileInfo struct, concrete type that implements File interface
type fileInfo struct {
	ETag      string `json:"ETag,omitempty"`
	Key       string `json:"key"`
	URL       string `json:"url"`
	Body      string `json:"body,omitempty"`
	Size      int64  `json:"size,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
}

func NewFile(ETag string, URL string, Key string, Body *string, size int64) File {
	return &fileInfo{ETag, Key, URL, *Body, size, 0}
}

// Marshal method to return json byte slice for sending back server
func (f *fileInfo) Marshal() ([]byte, error) {
	return json.Marshal(f)
}

func (f *fileInfo) UnMarshal(data []byte) error {
	return json.Unmarshal(data, f)
}

func (f *fileInfo) GetETag() string {
	return f.ETag
}

func (f *fileInfo) GetURL() string {
	return f.URL
}

func (f *fileInfo) GetKey() string {
	return f.Key
}

func (f *fileInfo) GetBody() string {
	return f.Body
}

func (f *fileInfo) GetSize() int64 {
	return f.Size
}
