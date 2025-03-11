package app

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func TestParseAddItemRequest(t *testing.T) {
	t.Parallel()
	cwd, err:= os.Getwd()
	if err!= nil{
		t.Fatalf("failed to get current directory: %v", err)
	}
	t.Logf("Current working directory: %s", cwd)

	type wants struct {
		req *AddItemRequest
		err bool
	}
	ImageBytes, err := os.ReadFile("../images/default.jpg")
	if err != nil {
		t.Fatalf("Failed to read image file: %v", err)
	}

	// STEP 6-1: define test cases
	cases := map[string]struct {
		args map[string]string
		ImageData []byte
		wants
	}{
		"ok: valid request": {
			args: map[string]string{
				"name":     "jacket", // fill here
				"category": "fashion", // fill here
				"image": "default.jpg",
			},
			ImageData: ImageBytes,
			wants: wants{
				req: &AddItemRequest{
					Name: "jacket", // fill here
					Category: "fashion", // fill here
					Image: ImageBytes,
				},
				err: false,
			},
		},
		"ng: empty request": {
			args: map[string]string{},
			ImageData:	nil,
			wants: wants{
				req: nil,
				err: true,
			},
		},
		"ng: empty name": {
			args:		map[string]string{
				"category": "fashion",
				"image":    "default.jpg",
			},
			ImageData:	ImageBytes,
			wants: wants{
				req: nil,
				err: true,
			},
		},

		"ng: empty category": {
			args:		map[string]string{
				"name":		"jacket",
				"image":	"default.jpg",
			},
			ImageData:	ImageBytes,
			wants: wants{
				req: nil,
				err: true,
			},
		},

		"ng: empty image": {
			args:		map[string]string{
				"name":     "jacket",
				"category": "fashion",
			},
			ImageData:	nil,
			wants: wants{
				req: &AddItemRequest{
                    Name:     "jacket",
                    Category: "fashion",
                },
                err: false,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// prepare request body
			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)

			for key, val := range tt.args {
    			_ = writer.WriteField(key, val)
			}

			if tt.ImageData != nil {
    			part, _ := writer.CreateFormFile("image", "default.jpg")
    			part.Write(tt.ImageData)
			}

			writer.Close()

			req := httptest.NewRequest("POST", "/items", &buf)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			// execute test target
			got, err := parseAddItemRequest(req)

			// confirm the result
			if err != nil {
				if !tt.err {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if diff := cmp.Diff(tt.wants.req, got); diff != "" {
				t.Errorf("unexpected request (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelloHandler(t *testing.T) {
	t.Parallel()

	// Please comment out for STEP 6-2
	// predefine what we want
	type wants struct {
		code int               // desired HTTP status code
		body map[string]string // desired body
	}
	want := wants{
		code: http.StatusOK,
		body: map[string]string{"message": "Hello, world!"},
	}

	// set up test
	req := httptest.NewRequest("GET", "/hello", nil)
	res := httptest.NewRecorder()

	h := &Handlers{}
	h.Hello(res, req)

	// STEP 6-2: confirm the status code
	if res.Code != want.code {
		t.Errorf("expected status code %d, got %d", want.code, res.Code)
	}

	// STEP 6-2: confirm response body
	expectedRes := HelloResponse{Message: want.body["message"]}

	var response HelloResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if diff := cmp.Diff(expectedRes, response); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
}

func TestAddItem(t *testing.T) {
	t.Parallel()

	type wants struct {
		code int
		body string
	}
	cases := map[string]struct {
		args     map[string]string
		injector func(m *MockItemRepository)
		wants
	}{
		"ok: correctly inserted": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
			},
			injector: func(m *MockItemRepository) {
				// STEP 6-3: define mock expectation
				// succeeded to insert
				m.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
			},
			wants: wants{
				code: http.StatusOK,
				body: `{"name":"used iPhone 16e","category":"phone","image":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.jpg"}` + "\n",
			},
		},
		"ng: failed to insert": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
			},
			injector: func(m *MockItemRepository) {
				// STEP 6-3: define mock expectation
				// failed to insert
				m.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(errors.New("failed to insert"))
			},
			wants: wants{
				code: http.StatusInternalServerError,
				body: "failed to insert\n",
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockIR := NewMockItemRepository(ctrl)
			tt.injector(mockIR)
			h := &Handlers{itemRepo: mockIR}

			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)

			for key, val := range tt.args {
    			_ = writer.WriteField(key, val)
			}

			if tt.args["image"] != "" {
    			ImageBytes, err := os.ReadFile("../go/images/default.jpg")
    			if err != nil {
        			t.Fatalf("failed to read image file: %v", err)
    			}
    			part, err := writer.CreateFormFile("image", "default.jpg")
    			if err != nil {
        			t.Fatalf("failed to create form file: %v", err)
    			}
    			part.Write(ImageBytes)
			}

			writer.Close()
			req := httptest.NewRequest("POST", "/items", &buf)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			rr := httptest.NewRecorder()

			h.AddItem(rr, req)

			if tt.wants.code != rr.Code {
				t.Errorf("expected status code %d, got %d", tt.wants.code, rr.Code)
			}
			if tt.wants.code >= 400 {
				return
			}

			for _, v := range tt.args {
				if !strings.Contains(rr.Body.String(), v) {
					t.Errorf("response body does not contain %s, got: %s", v, rr.Body.String())
				}
			}
			if !bytes.Equal(rr.Body.Bytes(), []byte(tt.wants.body)) {
				t.Errorf("expected response body %s, got %s", tt.wants.body, rr.Body.String())
			}
		})
	}
}

// STEP 6-4: uncomment this test
func TestAddItemE2e(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	db, closers, err := setupDB(t)
	if err != nil {
		t.Fatalf("failed to set up database: %v", err)
	}
	t.Cleanup(func() {
		for _, c := range closers {
			c()
		}
	})

	type wants struct {
		code int
	}
	cases := map[string]struct {
		args map[string]string
		wants
	}{
		"ok: correctly inserted": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
			},
			wants: wants{
				code: http.StatusOK,
			},
		},
		"ng: failed to insert": {
			args: map[string]string{
				"name":     "",
				"category": "phone",
			},
			wants: wants{
				code: http.StatusBadRequest,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			h := &Handlers{itemRepo: &itemRepository{db: db}}

			var buf bytes.Buffer
            writer := multipart.NewWriter(&buf)

            for key, val := range tt.args {
                _ = writer.WriteField(key, val)
            }

            if tt.args["image"] != "" {
                ImageBytes, err := os.ReadFile("../go/images/default.jpg")
                if err != nil {
                    t.Fatalf("failed to read image file: %v", err)
                }
                part, err := writer.CreateFormFile("image", "default.jpg")
                if err != nil {
                    t.Fatalf("failed to create form file: %v", err)
                }
                part.Write(ImageBytes)
            }
			writer.Close()
            req := httptest.NewRequest("POST", "/items", &buf)
            req.Header.Set("Content-Type", writer.FormDataContentType())

			rr := httptest.NewRecorder()
			h.AddItem(rr, req)

			// check response
			if tt.wants.code != rr.Code {
				t.Errorf("expected status code %d, got %d", tt.wants.code, rr.Code)
			}
			if tt.wants.code >= 400 {
				return
			}
			for _, v := range tt.args {
				if !strings.Contains(rr.Body.String(), v) {
					t.Errorf("response body does not contain %s, got: %s", v, rr.Body.String())
				}
			}

			// STEP 6-4: check inserted data
		})
	}
}

func setupDB(t *testing.T) (db *sql.DB, closers []func(), e error) {
	t.Helper()

	defer func() {
		if e != nil {
			for _, c := range closers {
				c()
			}
		}
	}()

	// create a temporary file for e2e testing
	f, err := os.CreateTemp("", "*.sqlite3")
	if err != nil {
		return nil, nil, err
	}
	closers = append(closers, func() {
		f.Close()
		os.Remove(f.Name())
	})

	// set up tables
	db, err = sql.Open("sqlite3", f.Name())
	if err != nil {
		return nil, nil, err
	}
	closers = append(closers, func() {
		db.Close()
	})

	// TODO: replace it with real SQL statements.
	cmd := `CREATE TABLE IF NOT EXISTS items (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name VARCHAR(255),
        category_id INTEGER,
        image VARCHAR(255),
        FOREIGN KEY (category_id) REFERENCES categories(id)
    )`
	_, err = db.Exec(cmd)
	if err != nil {
		return nil, nil, err
	}
	// categoriesテーブルの作成
    cmd = `CREATE TABLE IF NOT EXISTS categories (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name VARCHAR(255) UNIQUE
    )`
    _, err = db.Exec(cmd)
    if err != nil {
        return nil, nil, err
    }

	return db, closers, nil
}
