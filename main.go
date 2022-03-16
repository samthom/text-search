package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/RediSearch/redisearch-go/redisearch"
	"github.com/google/go-tika/tika"
)

func main() {
	filePath := "/Users/samthomas/Downloads/test.pdf"
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fmt.Println("file read")

	client := tika.NewClient(nil, "http://localhost:9998")
	body, err := client.Parse(context.Background(), f)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Print(body)

	// redis
	c := redisearch.NewClient("localhost:6379", "demo")

	// schema
	sc := redisearch.NewSchema(redisearch.DefaultOptions).
		AddField(redisearch.NewTextField("body")).
		AddField(redisearch.NewTextFieldOptions("title", redisearch.TextFieldOptions{Weight: 5.0, Sortable: true})).
		AddField(redisearch.NewNumericField("data"))

	c.Drop()

	if err := c.CreateIndex(sc); err != nil {
		log.Fatal(err)
	}

	doc := redisearch.NewDocument("doc1", 1.0)
	doc.Set("title", filePath).
		Set("body", body).
		Set("date", time.Now().Unix())

	if err := c.Index([]redisearch.Document{doc}...); err != nil {
		log.Fatal(err)
	}

	docs, total, err := c.Search(redisearch.NewQuery("@body:560001").
		Limit(0, 2).
		SetReturnFields("title"))

	fmt.Println(docs[0].Id, docs[0].Properties["title"], total, err)

	os.Exit(0)
}
