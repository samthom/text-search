package file

import (
	"context"
	"io"

	"github.com/google/go-tika/tika"
)

// Parser interface with parser function
type Parser interface {
	ParseFile(ctx context.Context, r io.Reader) (res string, err error)
	ParseFileReader(ctx context.Context, r io.Reader) (res io.ReadCloser, err error)
}

// fileParser concrete type that implements Parser interface
// tika client encapsulated
type fileParser struct {
	*tika.Client
}

func NewParser(url *string) Parser {
	if url != nil {
		client := tika.NewClient(nil, *url)
		return &fileParser{client}
	}
	return &fileParser{tika.NewClient(nil, "http://localhost:9998")}
}

// ParseFile method parses the file and return the value as a string
func (p *fileParser) ParseFile(ctx context.Context, r io.Reader) (string, error) {
	return p.Parse(ctx, r)
}

// ParseFileReader method parses the file and return the value as a readable (ReadCloser)
func (p *fileParser) ParseFileReader(ctx context.Context, r io.Reader) (io.ReadCloser, error) {
	return p.ParseReader(ctx, r)
}
