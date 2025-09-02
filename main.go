package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"go.etcd.io/bbolt"
)

type Store struct {
	db *bbolt.DB
}

func NewStore(path string) *Store {
	db, _ := bbolt.Open(path, 0600, nil)
	db.Update(func(tx *bbolt.Tx) error {
		_, _ = tx.CreateBucketIfNotExists([]byte("seen"))
		return nil
	})
	return &Store{db}
}

func (store *Store) IsNew(id string) bool {
	var known bool
	store.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("seen"))
		v := b.Get([]byte(id))
		known = v != nil
		return nil
	})
	return !known
}

func (store *Store) Mark(id string) {
	store.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("seen"))
		return b.Put([]byte(id), []byte("1"))
	})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	store := NewStore("scrape.db")
	defer store.db.Close()

	searchURL := os.Getenv("SEARCH_URL")
	if searchURL == "" {
		fmt.Println("SEARCH_URL environment variable is required")
		return
	}
	resp, err := http.Get(os.Getenv("SEARCH_URL"))
	if err != nil {
		log.Fatalf("Error fetching URL: %v", err)
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}
	doc.Find("a").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if strings.Contains(href, os.Getenv("SEARCH_TERM_1")) ||
			strings.Contains(href, os.Getenv("SEARCH_TERM_2")) &&
				!strings.Contains(href, os.Getenv("EXCLUDE_TERM_1")) {
			id := href

			if store.IsNew(id) {
				fmt.Printf("NEW LISTING: %s (ID: %s)\n", href, id)
				store.Mark(id)
			}
		}
	})

}
