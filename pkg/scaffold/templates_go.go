package scaffold

import "fmt"

// GetGoTemplates returns the map of relative file path -> file content for Go projects
func GetGoTemplates(projectName, framework, goVersion string, includeCRUD bool) map[string]string {
	files := make(map[string]string)

	if goVersion == "" {
		goVersion = "1.22"
	}

	files[".gitignore"] = `bin/
dist/
*.exe
*.test
*.out
.env
.DS_Store
vendor/
`
	files[".env.example"] = "PORT=8080\nENV=development\n"
	files[".env"] = "PORT=8080\nENV=development\n"

	files["Makefile"] = fmt.Sprintf(`.PHONY: build run test clean

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v ./...

clean:
	rm -rf bin/
`)

	switch framework {
	case "chi":
		getGoChiTemplates(files, projectName, goVersion, includeCRUD)
	case "fiber":
		getGoFiberTemplates(files, projectName, goVersion, includeCRUD)
	default:
		// Default: Gin
		getGoGinTemplates(files, projectName, goVersion, includeCRUD)
	}

	return files
}

func getGoGinTemplates(files map[string]string, projectName, goVersion string, includeCRUD bool) {
	files["go.mod"] = fmt.Sprintf(`module %s

go %s

require (
	github.com/gin-gonic/gin v1.9.1
)
`, projectName, goVersion)

	files["internal/models/item.go"] = `package models

import "time"

type Item struct {
	ID          string    ` + "`json:\"id\"`" + `
	Name        string    ` + "`json:\"name\" binding:\"required\"`" + `
	Description string    ` + "`json:\"description\"`" + `
	CreatedAt   time.Time ` + "`json:\"createdAt\"`" + `
}

type CreateItemRequest struct {
	Name        string ` + "`json:\"name\" binding:\"required\"`" + `
	Description string ` + "`json:\"description\"`" + `
}

type UpdateItemRequest struct {
	Name        string ` + "`json:\"name\"`" + `
	Description string ` + "`json:\"description\"`" + `
}
`

	files["internal/repository/item_repository.go"] = `package repository

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"` + projectName + `/internal/models"
)

var ErrNotFound = errors.New("item not found")

type ItemRepository interface {
	GetAll() []models.Item
	GetByID(id string) (*models.Item, error)
	Create(req models.CreateItemRequest) models.Item
	Update(id string, req models.UpdateItemRequest) (*models.Item, error)
	Delete(id string) error
}

type inMemoryItemRepository struct {
	mu    sync.RWMutex
	items map[string]models.Item
}

func NewItemRepository() ItemRepository {
	return &inMemoryItemRepository{
		items: make(map[string]models.Item),
	}
}

func (r *inMemoryItemRepository) GetAll() []models.Item {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]models.Item, 0, len(r.items))
	for _, it := range r.items {
		res = append(res, it)
	}
	return res
}

func (r *inMemoryItemRepository) GetByID(id string) (*models.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, exists := r.items[id]
	if !exists {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (r *inMemoryItemRepository) Create(req models.CreateItemRequest) models.Item {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	item := models.Item{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}
	r.items[id] = item
	return item
}

func (r *inMemoryItemRepository) Update(id string, req models.UpdateItemRequest) (*models.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, exists := r.items[id]
	if !exists {
		return nil, ErrNotFound
	}

	if req.Name != "" {
		item.Name = req.Name
	}
	if req.Description != "" {
		item.Description = req.Description
	}

	r.items[id] = item
	return &item, nil
}

func (r *inMemoryItemRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[id]; !exists {
		return ErrNotFound
	}
	delete(r.items, id)
	return nil
}
`

	files["internal/handlers/item_handler.go"] = `package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"` + projectName + `/internal/models"
	"` + projectName + `/internal/repository"
)

type ItemHandler struct {
	repo repository.ItemRepository
}

func NewItemHandler(repo repository.ItemRepository) *ItemHandler {
	return &ItemHandler{repo: repo}
}

func (h *ItemHandler) GetAll(c *gin.Context) {
	items := h.repo.GetAll()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func (h *ItemHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	item, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *ItemHandler) Create(c *gin.Context) {
	var req models.CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	created := h.repo.Create(req)
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": created})
}

func (h *ItemHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	updated, err := h.repo.Update(id, req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": updated})
}

func (h *ItemHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Item deleted successfully"})
}
`

	files["internal/routes/routes.go"] = `package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"` + projectName + `/internal/handlers"
	"` + projectName + `/internal/repository"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	repo := repository.NewItemRepository()
	itemHandler := handlers.NewItemHandler(repo)

	api := r.Group("/api/items")
	{
		api.GET("", itemHandler.GetAll)
		api.GET("/:id", itemHandler.GetByID)
		api.POST("", itemHandler.Create)
		api.PUT("/:id", itemHandler.Update)
		api.DELETE("/:id", itemHandler.Delete)
	}

	return r
}
`

	files["cmd/server/main.go"] = `package main

import (
	"log"
	"os"

	"` + projectName + `/internal/routes"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := routes.SetupRouter()
	log.Printf("🚀 Gin Server listening on http://localhost:%s", port)
	log.Printf("👉 Health check: http://localhost:%s/health", port)
	log.Printf("👉 CRUD API:    http://localhost:%s/api/items", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
`

	files["README.md"] = fmt.Sprintf(`# %s (Go + Gin)

Clean Architecture REST API built with **Go** and **Gin**, scaffolded by **UVM**.

## 🚀 Quick Start

1. **Install Dependencies**:
   `+"```bash\n   go mod tidy\n   ```"+`
2. **Run Server**:
   `+"```bash\n   go run ./cmd/server\n   ```"+`
3. **Build Binary**:
   `+"```bash\n   make build\n   ./bin/server\n   ```"+`

## 📡 API Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| **GET** | `+"`/health`"+` | Health check |
| **GET** | `+"`/api/items`"+` | List all items |
| **GET** | `+"`/api/items/:id`"+` | Get item by ID |
| **POST** | `+"`/api/items`"+` | Create new item |
| **PUT** | `+"`/api/items/:id`"+` | Update item |
| **DELETE** | `+"`/api/items/:id`"+` | Delete item |
`, projectName)
}

func getGoChiTemplates(files map[string]string, projectName, goVersion string, includeCRUD bool) {
	files["go.mod"] = fmt.Sprintf(`module %s

go %s

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.1
)
`, projectName, goVersion)

	files["internal/models/item.go"] = `package models

import "time"

type Item struct {
	ID          string    ` + "`json:\"id\"`" + `
	Name        string    ` + "`json:\"name\"`" + `
	Description string    ` + "`json:\"description\"`" + `
	CreatedAt   time.Time ` + "`json:\"createdAt\"`" + `
}

type CreateItemRequest struct {
	Name        string ` + "`json:\"name\"`" + `
	Description string ` + "`json:\"description\"`" + `
}
`

	files["internal/repository/item_repository.go"] = `package repository

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"` + projectName + `/internal/models"
)

var ErrNotFound = errors.New("item not found")

type ItemRepository struct {
	mu    sync.RWMutex
	items map[string]models.Item
}

func NewItemRepository() *ItemRepository {
	return &ItemRepository{items: make(map[string]models.Item)}
}

func (r *ItemRepository) GetAll() []models.Item {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]models.Item, 0, len(r.items))
	for _, it := range r.items {
		res = append(res, it)
	}
	return res
}

func (r *ItemRepository) GetByID(id string) (*models.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	it, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &it, nil
}

func (r *ItemRepository) Create(req models.CreateItemRequest) models.Item {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	it := models.Item{ID: id, Name: req.Name, Description: req.Description, CreatedAt: time.Now()}
	r.items[id] = it
	return it
}

func (r *ItemRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return ErrNotFound
	}
	delete(r.items, id)
	return nil
}
`

	files["internal/handlers/item_handler.go"] = `package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"` + projectName + `/internal/models"
	"` + projectName + `/internal/repository"
)

type ItemHandler struct {
	repo *repository.ItemRepository
}

func NewItemHandler(repo *repository.ItemRepository) *ItemHandler {
	return &ItemHandler{repo: repo}
}

func (h *ItemHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	items := h.repo.GetAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *ItemHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	item, err := h.repo.GetByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created := h.repo.Create(req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
`

	files["internal/routes/routes.go"] = `package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"` + projectName + `/internal/handlers"
	"` + projectName + `/internal/repository"
)

func SetupRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"status\":\"ok\"}"))
	})

	repo := repository.NewItemRepository()
	itemHandler := handlers.NewItemHandler(repo)

	r.Route("/api/items", func(r chi.Router) {
		r.Get("/", itemHandler.GetAll)
		r.Get("/{id}", itemHandler.GetByID)
		r.Post("/", itemHandler.Create)
		r.Delete("/{id}", itemHandler.Delete)
	})

	return r
}
`

	files["cmd/server/main.go"] = `package main

import (
	"log"
	"net/http"
	"os"

	"` + projectName + `/internal/routes"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := routes.SetupRouter()
	log.Printf("🚀 Chi Server listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
`

	files["README.md"] = fmt.Sprintf(`# %s (Go + Chi)

Idiomatic lightweight REST API built with **Go** and **Chi**, scaffolded by **UVM**.

## 🚀 Quick Start
`+"```bash\ngo mod tidy\ngo run ./cmd/server\n```", projectName)
}

func getGoFiberTemplates(files map[string]string, projectName, goVersion string, includeCRUD bool) {
	files["go.mod"] = fmt.Sprintf(`module %s

go %s

require (
	github.com/gofiber/fiber/v2 v2.52.1
)
`, projectName, goVersion)

	files["internal/models/item.go"] = `package models

import "time"

type Item struct {
	ID          string    ` + "`json:\"id\"`" + `
	Name        string    ` + "`json:\"name\"`" + `
	Description string    ` + "`json:\"description\"`" + `
	CreatedAt   time.Time ` + "`json:\"createdAt\"`" + `
}

type CreateItemRequest struct {
	Name        string ` + "`json:\"name\"`" + `
	Description string ` + "`json:\"description\"`" + `
}
`

	files["internal/repository/item_repository.go"] = `package repository

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"` + projectName + `/internal/models"
)

var ErrNotFound = errors.New("item not found")

type ItemRepository struct {
	mu    sync.RWMutex
	items map[string]models.Item
}

func NewItemRepository() *ItemRepository {
	return &ItemRepository{items: make(map[string]models.Item)}
}

func (r *ItemRepository) GetAll() []models.Item {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]models.Item, 0, len(r.items))
	for _, it := range r.items {
		res = append(res, it)
	}
	return res
}

func (r *ItemRepository) GetByID(id string) (*models.Item, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	it, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &it, nil
}

func (r *ItemRepository) Create(req models.CreateItemRequest) models.Item {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	it := models.Item{ID: id, Name: req.Name, Description: req.Description, CreatedAt: time.Now()}
	r.items[id] = it
	return it
}

func (r *ItemRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return ErrNotFound
	}
	delete(r.items, id)
	return nil
}
`

	files["internal/handlers/item_handler.go"] = `package handlers

import (
	"github.com/gofiber/fiber/v2"
	"` + projectName + `/internal/models"
	"` + projectName + `/internal/repository"
)

type ItemHandler struct {
	repo *repository.ItemRepository
}

func NewItemHandler(repo *repository.ItemRepository) *ItemHandler {
	return &ItemHandler{repo: repo}
}

func (h *ItemHandler) GetAll(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "data": h.repo.GetAll()})
}

func (h *ItemHandler) GetByID(c *fiber.Ctx) error {
	item, err := h.repo.GetByID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": item})
}

func (h *ItemHandler) Create(c *fiber.Ctx) error {
	var req models.CreateItemRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	created := h.repo.Create(req)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": created})
}

func (h *ItemHandler) Delete(c *fiber.Ctx) error {
	if err := h.repo.Delete(c.Params("id")); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Item deleted successfully"})
}
`

	files["internal/routes/routes.go"] = `package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"` + projectName + `/internal/handlers"
	"` + projectName + `/internal/repository"
)

func SetupRouter(app *fiber.App) {
	app.Use(logger.New())
	app.Use(recover.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	repo := repository.NewItemRepository()
	itemHandler := handlers.NewItemHandler(repo)

	api := app.Group("/api/items")
	api.Get("/", itemHandler.GetAll)
	api.Get("/:id", itemHandler.GetByID)
	api.Post("/", itemHandler.Create)
	api.Delete("/:id", itemHandler.Delete)
}
`

	files["cmd/server/main.go"] = `package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"` + projectName + `/internal/routes"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app := fiber.New()
	routes.SetupRouter(app)

	log.Printf("🚀 Fiber Server listening on http://localhost:%s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
`

	files["README.md"] = fmt.Sprintf(`# %s (Go + Fiber)

Express-inspired high-performance REST API built with **Go** and **Fiber**, scaffolded by **UVM**.

## 🚀 Quick Start
`+"```bash\ngo mod tidy\ngo run ./cmd/server\n```", projectName)
}
