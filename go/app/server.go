package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Server struct {
	// Port is the port number to listen on.
	Port string
	// ImageDirPath is the path to the directory storing images.
	ImageDirPath string
}

// Run is a method to start the server.
// This method returns 0 if the server started successfully, and 1 otherwise.
func (s Server) Run() int {
	// set up logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug, // step4-6で変更
	}))
	slog.SetDefault(logger)
	// STEP 4-6: set the log level to DEBUG
	slog.SetLogLoggerLevel(slog.LevelInfo)

	// set up CORS settings
	frontURL, found := os.LookupEnv("FRONT_URL")
	if !found {
		frontURL = "http://localhost:3000"
	}

	// STEP 5-1: set up the database connection

	// set up handlers
	itemRepo := NewItemRepository()
	h := &Handlers{imgDirPath: s.ImageDirPath, itemRepo: itemRepo}

	// set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Hello)
	mux.HandleFunc("POST /items", h.AddItem)
	mux.HandleFunc("GET /items", h.GetItems)
	mux.HandleFunc("GET /items/{item_id}", h.GetItem)
	mux.HandleFunc("GET /images/{filename}", h.GetImage)
	mux.HandleFunc("GET /search", h.Search)

	// start the server
	slog.Info("http server started on", "port", s.Port)
	err := http.ListenAndServe(":"+s.Port, simpleCORSMiddleware(simpleLoggerMiddleware(mux), frontURL, []string{"GET", "HEAD", "POST", "OPTIONS"}))
	if err != nil {
		slog.Error("failed to start server: ", "error", err)
		return 1
	}

	return 0
}

type Handlers struct {
	// imgDirPath is the path to the directory storing images.
	imgDirPath string
	itemRepo   ItemRepository
}

type HelloResponse struct {
	Message string `json:"message"`
}

// Hello is a handler to return a Hello, world! message for GET / .
func (s *Handlers) Hello(w http.ResponseWriter, r *http.Request) {
	resp := HelloResponse{Message: "Hello, world!"}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
type ItemsWrapper struct {
    Items []Item `json:"items"`
}

type AddItemRequest struct {
	Name string `form:"name"`
	// Category string `form:"category"` // STEP 4-2: add a category field
	Category string `form:"category"`
	Image []byte `form:"image"` // STEP 4-4: add an image field
}

type AddItemResponse struct {
	Message string `json:"message"`
}

// parseAddItemRequest parses and validates the request to add an item.
func parseAddItemRequest(r *http.Request) (*AddItemRequest, error) {
	req := &AddItemRequest{
		Name: r.FormValue("name"),
		// STEP 4-2: add a category field
		Category: r.FormValue("category"),
	}
	// STEP 4-4: add an image field
	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, errors.New("image is required")
	}
	defer file.Close()

	// validate the request
	if req.Name == "" {
		return nil, errors.New("name is required")
	}

	// STEP 4-2: validate the category field
	if req.Category == "" {
		return nil, errors.New("category is required")
	}
	// STEP 4-4: validate the image field
	imageData, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.New("failed to read image data")
	}
	req.Image = imageData
	return req, nil
}

// AddItem is a handler to add a new item for POST /items .
func (s *Handlers) AddItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req, err := parseAddItemRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// STEP 4-4: uncomment on adding an implementation to store an image
	fileName, err := s.storeImage(req.Image)
	slog.Info("Stored image", "fileName", fileName)
	if err != nil {
		slog.Error("failed to store image: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	items := &Item{
		Name: req.Name,
		// STEP 4-2: add a category field
		Category: req.Category,
		// STEP 4-4: add an image field
		Image: fileName,
	}
	// データベースに保存
	err = s.itemRepo.Insert(ctx, items)
	if err != nil {
    	slog.Error("failed to store item", "error", err)
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
	}

	// レスポンスを送信
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(items)
	if err != nil {
    	slog.Error("failed to encode response", "error", err)
    	http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
// GetItems is a handler to return a list of items for GET /items.
func (s *Handlers) GetItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	items, err := s.itemRepo.LoadItems(ctx)
	if err != nil {
		slog.Error("failed to load items: ", "error", err)
		http.Error(w, "Failed to load items", http.StatusInternalServerError)
		return
	}
	// JSONレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Handlers) GetItem(w http.ResponseWriter, r *http.Request) {
	// 1. "item_id" を取得
    itemIDStr := r.PathValue("item_id")
    itemID, err := strconv.Atoi(itemIDStr)//整数に変換
    if err != nil {
        http.Error(w, "Invalid item ID", http.StatusBadRequest)
        return
    }
    slog.Info("Received item_id:", "item_id", itemID)

	//2. itemを取得
	ctx := r.Context()
	items, err := s.itemRepo.LoadItems(ctx)
	if err != nil {
		slog.Error("failed to load items: ", "error", err)
		http.Error(w, "Failed to load items", http.StatusInternalServerError)
		return
	}
	if itemID <= 0 || itemID > len(items) {//配列の範囲外チェック
        http.Error(w, "Item not found", http.StatusNotFound)
        return
    }
	// leni := len(items)
	// slog.Info("items_length", "length", leni)
	item := items[itemID-1]
	// JSONレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(item)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

// storeImage stores an image and returns the file path and an error if any.
// this method calculates the hash sum of the image as a file name to avoid the duplication of a same file
// and stores it in the image directory.
func (s *Handlers) storeImage(image []byte) (filePath string, err error) {
	// STEP 4-4: add an implementation to store an image
	// - calc hash sum
	hasher := sha256.New()
	hasher.Write(image) // image をハッシュに書き込む
	hashSum := hex.EncodeToString(hasher.Sum(nil)) // ハッシュ値を16進文字列に変換
	// - build image file path
	fileName := fmt.Sprintf("%s.jpg", hashSum)
	filePath = filepath.Join(s.imgDirPath, fileName)
	// - check if the image already exists
	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}
	// - store image
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(image)
	if err != nil {
		return "", fmt.Errorf("failed to write image file: %w", err)
	}
	// - return the image file path
	return fileName, nil
}

type GetImageRequest struct {
	FileName string // path value
}

// parseGetImageRequest parses and validates the request to get an image.
func parseGetImageRequest(r *http.Request) (*GetImageRequest, error) {
	req := &GetImageRequest{
		FileName: r.PathValue("filename"), // from path parameter
	}

	// validate the request
	if req.FileName == "" {
		return nil, errors.New("filename is required")
	}

	return req, nil
}

// GetImage is a handler to return an image for GET /images/{filename} .
// If the specified image is not found, it returns the default image.
func (s *Handlers) GetImage(w http.ResponseWriter, r *http.Request) {
	req, err := parseGetImageRequest(r)
	if err != nil {
		slog.Warn("failed to parse get image request: ", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imgPath, err := s.buildImagePath(req.FileName)
	if err != nil {
		if !errors.Is(err, errImageNotFound) {
			slog.Warn("failed to build image path: ", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// when the image is not found, it returns the default image without an error.
		slog.Debug("image not found", "filename", imgPath)
		imgPath = filepath.Join(s.imgDirPath, "default.jpg")
	}

	slog.Info("returned image", "path", imgPath)
	http.ServeFile(w, r, imgPath)
}

// buildImagePath builds the image path and validates it.
func (s *Handlers) buildImagePath(imageFileName string) (string, error) {
	imgPath := filepath.Join(s.imgDirPath, filepath.Clean(imageFileName))

	// to prevent directory traversal attacks
	rel, err := filepath.Rel(s.imgDirPath, imgPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid image path: %s", imgPath)
	}

	// validate the image suffix
	if !strings.HasSuffix(imgPath, ".jpg") && !strings.HasSuffix(imgPath, ".jpeg") {
		return "", fmt.Errorf("image path does not end with .jpg or .jpeg: %s", imgPath)
	}

	// check if the image exists
	_, err = os.Stat(imgPath)
	if err != nil {
		return imgPath, errImageNotFound
	}

	return imgPath, nil
}

// 指定されたキーワードを含む商品を検索するエンドポイント
func (s *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	// クエリパラメータから keyword を取得
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		http.Error(w, "keyword is required", http.StatusBadRequest)
		return
	}

	// データベースで商品を検索
	items, err := s.itemRepo.SearchItems(r.Context(), keyword)
	if err != nil {
		http.Error(w, "failed to search items", http.StatusInternalServerError)
		return
	}

	// JSONレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
}