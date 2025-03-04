package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	// STEP 5-1: uncomment this line
	// _ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")

type Item struct {
	ID   int    `db:"id" json:"-"`
	Name string `db:"name" json:"name"`
	Category string `db:"category" json:"category"`
	Image string `db:"image" json:"image"`
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
	LoadItems(ctx context.Context) ([]*Item, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	fileName string
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository() ItemRepository {
	return &itemRepository{fileName: "items.json"}
}
func (i *itemRepository) LoadItems(ctx context.Context) ([]*Item, error) {
	file, err := os.OpenFile(i.fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var wrapper ItemsWrapper
	err = json.NewDecoder(file).Decode(&wrapper)
	if err == io.EOF {
		return []*Item{}, nil // 空の配列を返す
	} else if err != nil {
		return nil, err
	}
	// []Item → []*Item に変換
	items := make([]*Item, len(wrapper.Items))
	for idx, item := range wrapper.Items {
		items[idx] = &item
	}

	return items, nil
}

func (i *itemRepository) storeItems(items []*Item) error {
	file, err := os.Create(i.fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(items)
}
// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	// STEP 4-1: add an implementation to store an item
	items, err := i.LoadItems(ctx)
	if err != nil {
		return err
	}
	items = append(items, item)
	return i.storeItems(items)
}

// StoreImage stores an image and returns an error if any.
// This package doesn't have a related interface for simplicity.
func StoreImage(fileName string, image []byte) error {
	// STEP 4-4: add an implementation to store an image
	return os.WriteFile(fileName, image, 0644)
}
