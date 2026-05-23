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
	http.HandleFunc("/post", handler.ShowPost)

	// Routes - Admin
	http.HandleFunc("/admin", requireAuth(handler.AdminIndex))
	http.HandleFunc("/admin/login", handler.Login)
	http.HandleFunc("/admin/logout", handler.Logout)

	// Admin CRUD routes
	http.HandleFunc("/admin/posts/add", requireAuth(handler.NewPostForm))
	http.HandleFunc("/admin/posts/create", requireAuth(handler.CreatePostForm))
	http.HandleFunc("/admin/posts/edit", requireAuth(handler.EditPostForm))
	http.HandleFunc("/admin/posts/update", requireAuth(handler.UpdatePostForm))
	http.HandleFunc("/admin/posts/delete", requireAuth(handler.DeletePostForm))

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

func (h *Handler) ShowPost(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Post ID required", http.StatusBadRequest)
		return
	}

	posts, err := h.postStore.LoadPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var foundPost *Post

	for _, post := range posts {
		if post.ID == id && post.Published {
			foundPost = &post
			break
		}
	}

	if foundPost == nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	data := struct {
		Title string
		Post  Post
	}{
		Title: foundPost.Title,
		Post:  *foundPost,
	}

	err = h.templates["guest/post"].ExecuteTemplate(w, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

	tmpl["guest/post"] = template.Must(template.ParseFiles(
		"templates/layout/base.html",
		"templates/guest/post.html",
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

	tmpl["admin/post"] = template.Must(template.ParseFiles(
		"templates/layout/base.html",
		"templates/admin/posts/add.html",
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

func (h *Handler) NewPostForm(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title  string
		Post   Post
		Errors map[string]string
	}{
		Title:  "Create New Post",
		Post:   Post{},
		Errors: nil,
	}

	err := h.templates["admin/post"].ExecuteTemplate(w, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) CreatePostForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate form
	title := r.FormValue("title")
	content := r.FormValue("content")
	published := r.FormValue("published") == "on"

	// Validation
	errors := make(map[string]string)
	if len(title) < 3 {
		errors["title"] = "Title must be at least 3 characters"
	}
	if len(title) > 200 {
		errors["title"] = "Title must be less than 200 characters"
	}
	if len(content) < 10 {
		errors["content"] = "Content must be at least 10 characters"
	}
	if len(content) > 50000 {
		errors["content"] = "Content must be less than 50000 characters"
	}

	if len(errors) > 0 {
		// Return to form with errors
		data := struct {
			Title  string
			Post   Post
			Errors map[string]string
		}{
			Title: "Create New Post",
			Post: Post{
				Title:     title,
				Content:   content,
				Published: published,
			},
			Errors: errors,
		}

		err := h.templates["admin/post"].ExecuteTemplate(w, "base.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Create post
	post := Post{
		ID:        generateID(),
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Published: published,
	}

	if err := h.postStore.CreatePost(post); err != nil {
		http.Error(w, "Failed to create post: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) EditPostForm(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Post ID required", http.StatusBadRequest)
		return
	}

	posts, err := h.postStore.LoadPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var foundPost *Post
	for _, post := range posts {
		if post.ID == id {
			foundPost = &post
			break
		}
	}

	if foundPost == nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	data := struct {
		Title  string
		Post   Post
		Errors map[string]string
	}{
		Title:  "Edit Post",
		Post:   *foundPost,
		Errors: nil,
	}

	err = h.templates["admin/post"].ExecuteTemplate(w, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) UpdatePostForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "Post ID required", http.StatusBadRequest)
		return
	}

	// Validate form
	title := r.FormValue("title")
	content := r.FormValue("content")
	published := r.FormValue("published") == "on"

	// Validation
	errors := make(map[string]string)
	if len(title) < 3 {
		errors["title"] = "Title must be at least 3 characters"
	}
	if len(title) > 200 {
		errors["title"] = "Title must be less than 200 characters"
	}
	if len(content) < 10 {
		errors["content"] = "Content must be at least 10 characters"
	}
	if len(content) > 50000 {
		errors["content"] = "Content must be less than 50000 characters"
	}

	if len(errors) > 0 {
		// Return to form with errors
		data := struct {
			Title  string
			Post   Post
			Errors map[string]string
		}{
			Title: "Edit Post",
			Post: Post{
				Title:     title,
				Content:   content,
				Published: published,
			},
			Errors: errors,
		}

		err := h.templates["admin/post"].ExecuteTemplate(w, "base.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Load existing post to preserve creation data
	posts, err := h.postStore.LoadPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var existingPost *Post
	for _, post := range posts {
		if post.ID == id {
			existingPost = &post
			break
		}
	}

	if existingPost == nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	// Update post
	updatedPost := Post{
		ID:        id,
		Title:     title,
		Content:   content,
		CreatedAt: existingPost.CreatedAt,
		UpdatedAt: time.Now(),
		Published: published,
	}

	if err := h.postStore.UpdatePost(updatedPost); err != nil {
		http.Error(w, "Failed to update post: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handler) DeletePostForm(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Post ID required", http.StatusBadRequest)
		return
	}

	// Add CSRF protection for DELETE
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.postStore.DeletePost(id); err != nil {
		http.Error(w, "Failed to delete post: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
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

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
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
