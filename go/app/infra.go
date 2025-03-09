package app

import (
	"context"
	// "encoding/json"
	"database/sql"
	"errors"
	"log"
	"os"

	// "net/http"

	// STEP 5-1: uncomment this line
	_ "github.com/mattn/go-sqlite3"
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
	SearchItems(ctx context.Context, keyword string) ([]*Item, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	// fileName string
	db *sql.DB
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository() ItemRepository {
	db, err := sql.Open("sqlite3", "db/mercari.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	return &itemRepository{db: db}
}
func (i *itemRepository) LoadItems(ctx context.Context) ([]*Item, error) {
	query := `
        SELECT items.id, items.name, categories.name, items.image_name 
        FROM items 
        JOIN categories ON items.category_id = categories.id
    `
	rows, err := i.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}

	return items, nil
}
// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	var categoryID int
	err := i.db.QueryRowContext(ctx, "SELECT id FROM categories WHERE name = ?", item.Category).Scan(&categoryID)
	if err == sql.ErrNoRows {
		// カテゴリが存在しない場合 -> 新規作成
		res, err := i.db.ExecContext(ctx, "INSERT INTO categories (name) VALUES (?)", item.Category)
		if err != nil {
			return err
		}
		lastID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		categoryID = int(lastID)
	} else if err != nil {
		return err
	}

	// items テーブルに新しいデータを挿入
	_, err = i.db.ExecContext(ctx, "INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)", item.Name, categoryID, item.Image)
	return err
}

// StoreImage stores an image and returns an error if any.
// This package doesn't have a related interface for simplicity.
func StoreImage(fileName string, image []byte) error {
	// STEP 4-4: add an implementation to store an image
	return os.WriteFile(fileName, image, 0644)
}

func (i *itemRepository) SearchItems(ctx context.Context, keyword string) ([]*Item, error) {
	query := `
        SELECT items.id, items.name, categories.name, items.image_name 
        FROM items 
        JOIN categories ON items.category_id = categories.id
        WHERE items.name LIKE ?
    `
	rows, err := i.db.QueryContext(ctx, query, "%"+keyword+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, nil
}
