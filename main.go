package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"
)

type Post struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Published bool      `json:"published"`
}

type PostStore struct {
	filename string
}

type Handler struct {
	postStore *PostStore
	templates map[string]*template.Template
}

func main() {
	startServer()
}

func startServer() {
	// Initialize data store
	postStore := newPostStore("data/posts.json")

	// Static Files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Initialize handlers
	handler := newHandler(postStore)

	// Routes - Guest
	http.HandleFunc("/", handler.Index)

	// Routes - Admin
	http.HandleFunc("/admin/login", handler.Login)
	http.HandleFunc("/admin/logout", handler.Logout)
	http.HandleFunc("/admin", requireAuth(handler.AdminIndex))

	fmt.Println("Server starting on http://localhost:3000")
	http.ListenAndServe(":3000", nil)
}

func newPostStore(filename string) *PostStore {
	return &PostStore{filename: filename}
}

func newHandler(postStore *PostStore) *Handler {
	return &Handler{
		postStore: postStore,
		templates: loadTemplates(),
	}
}

func (s *PostStore) LoadPosts() ([]Post, error) {
	data, err := os.ReadFile(s.filename)
	if err != nil {
		return []Post{}, nil // return empty if file doesn't exists
	}

	var posts []Post
	err = json.Unmarshal(data, &posts)
	return posts, err
}

func (s *PostStore) SavePosts(posts []Post) error {
	data, err := json.MarshalIndent(posts, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filename, data, 0644)
}

func (s *PostStore) CreatePost(post Post) error {
	posts, err := s.LoadPosts()
	if err != nil {
		return err
	}

	posts = append(posts, post)

	return s.SavePosts(posts)
}

func (s *PostStore) UpdatePost(updated Post) error {
	posts, err := s.LoadPosts()
	if err != nil {
		return err
	}

	for i, post := range posts {
		if post.ID == updated.ID {
			posts[i] = updated
			break
		}
	}

	return s.SavePosts(posts)
}

func (s *PostStore) DeletePost(id string) error {
	posts, err := s.LoadPosts()
	if err != nil {
		return err
	}

	var filtered []Post

	for _, post := range posts {
		if post.ID != id {
			filtered = append(filtered, post)
		}
	}

	return s.SavePosts(filtered)
}

func loadTemplates() map[string]*template.Template {
	tmpl := make(map[string]*template.Template)

	// Guest
	tmpl["guest/index"] = template.Must(template.ParseFiles(
		"templates/layout/base.html",
		"templates/guest/index.html",
	))

	// Admin
	tmpl["admin/index"] = template.Must(template.ParseFiles(
		"templates/layout/base.html",
		"templates/admin/index.html",
	))

	tmpl["admin/login"] = template.Must(template.ParseFiles(
		"templates/layout/base.html",
		"templates/admin/login.html",
	))

	tmpl["admin/logout"] = template.Must(template.ParseFiles(
		"templates/layout/base.html",
		"templates/admin/logout.html",
	))

	return tmpl
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	posts, err := h.postStore.LoadPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter published posts
	var publishedPosts []Post
	for _, post := range posts {
		if post.Published {
			publishedPosts = append(publishedPosts, post)
		}
	}

	data := struct {
		Title string
		Posts []Post
	}{
		Title: "My Blog",
		Posts: publishedPosts,
	}

	h.templates["guest/index"].ExecuteTemplate(w, "base.html", data)
}

func (h *Handler) AdminIndex(w http.ResponseWriter, r *http.Request) {
	posts, err := h.postStore.LoadPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter published posts
	var publishedPosts []Post
	for _, post := range posts {
		if post.Published {
			publishedPosts = append(publishedPosts, post)
		}
	}

	data := struct {
		Title string
		Posts []Post
	}{
		Title: "My Blog",
		Posts: publishedPosts,
	}

	h.templates["admin/index"].ExecuteTemplate(w, "base.html", data)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.templates["admin/login"].ExecuteTemplate(w, "base.html", nil)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	adminUser := os.Getenv("ADMIN_USERNAME")
	// For testing purposes only admin user
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	// For testing purposes only admin password
	if adminPassword == "" {
		adminPassword = "password"
	}

	if username == adminUser && password == adminPassword {
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "authenticated",
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Now().Add(24 * time.Hour),
		})

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	http.Error(w, "Invalid credentials", http.StatusUnauthorized)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(-1 * time.Hour),
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}

	return cookie.Value == "authenticated"
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated(r) {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}
