package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//go:embed templates static
var assets embed.FS

type NavItem struct {
	Key, Label, Href string
}

var nav = []NavItem{
	{"overview", "Home", "/"},
	{"photos", "Photos", "/photos"},
	{"documents", "Documents", "/documents"},
	{"media", "Media", "/media"},
	{"network", "Network", "/network"},
	{"services", "Apps", "/services"},
	{"backups", "Backups", "/backups"},
	{"users", "Family", "/users"},
}

type view struct {
	Page string
	OOB  bool
	Q    string
	Tag  string
	Snapshot
}

// Dot picks the status dot shown on a nav tab; "ok" tabs show none.
func (v view) Dot(key string) string {
	switch key {
	case "overview":
		for _, d := range v.Disks {
			if d.Status != "ok" {
				return "warn"
			}
		}
	case "network":
		if !v.DNS.Enabled {
			return "off"
		}
	case "services":
		if v.ServicesUp() < len(v.Services) {
			return "warn"
		}
	case "backups":
		for _, b := range v.Backups {
			if b.Status == "failed" {
				return "bad"
			}
		}
	case "documents":
		if v.Inbox > 0 {
			return "info"
		}
	}
	return ""
}

type server struct {
	store *Store
	pages map[string]*template.Template
	parts *template.Template
}

func main() {
	s := &server{store: NewStore(), pages: map[string]*template.Template{}}
	s.parts = template.Must(template.New("").Funcs(funcs).ParseFS(assets, "templates/layout.html", "templates/components.html"))
	for _, n := range nav {
		s.pages[n.Key] = template.Must(template.Must(s.parts.Clone()).ParseFS(assets, "templates/pages/"+n.Key+".html"))
	}

	static, _ := fs.Sub(assets, "static")
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	for _, n := range nav {
		path := n.Href
		if path == "/" {
			path = "/{$}"
		}
		mux.HandleFunc("GET "+path, s.page(n.Key))
	}
	mux.HandleFunc("GET /partials/{name}", s.partial)
	mux.HandleFunc("GET /services/{name}", s.serviceRow)
	mux.HandleFunc("POST /services/{name}/{action}", s.serviceAction)
	mux.HandleFunc("GET /backups/{id}", s.backupRow)
	mux.HandleFunc("POST /backups/{id}/run", s.runBackup)
	mux.HandleFunc("POST /network/blocking", s.toggleBlocking)
	mux.HandleFunc("POST /users", s.addUser)
	mux.HandleFunc("DELETE /users/{id}", s.deleteUser)
	mux.HandleFunc("POST /users/{id}/access/{app}", s.toggleAccess)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("hearth dashboard on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (s *server) view(r *http.Request, page string) view {
	q, tag := r.URL.Query().Get("q"), r.URL.Query().Get("tag")
	return view{Page: page, Q: q, Tag: tag, Snapshot: s.store.Snapshot(q, tag)}
}

func (s *server) page(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := s.view(r, name)
		tmpl := "layout"
		// Boosted nav clicks only need the new <main> plus an out-of-band nav refresh.
		if r.Header.Get("HX-Target") == "main" && r.Header.Get("HX-History-Restore-Request") != "true" {
			tmpl, v.OOB = "swap", true
		}
		w.Header().Set("Vary", "HX-Request")
		render(w, s.pages[name], tmpl, v)
	}
}

var partials = map[string]bool{"vitals": true, "jobs": true, "dns-stats": true, "doc-results": true, "pihole-toggle": true}

func (s *server) partial(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !partials[name] {
		http.NotFound(w, r)
		return
	}
	render(w, s.parts, name, s.view(r, ""))
}

func (s *server) serviceRow(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.store.Service(r.PathValue("name"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	render(w, s.parts, "service-row", svc)
}

func (s *server) serviceAction(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.store.ServiceAction(r.PathValue("name"), r.PathValue("action"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	render(w, s.parts, "service-row", svc)
}

func (s *server) backupRow(w http.ResponseWriter, r *http.Request) {
	b, ok := s.store.Backup(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	render(w, s.parts, "backup-row", b)
}

func (s *server) runBackup(w http.ResponseWriter, r *http.Request) {
	b, ok := s.store.RunBackup(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	render(w, s.parts, "backup-row", b)
}

func (s *server) toggleBlocking(w http.ResponseWriter, r *http.Request) {
	s.store.ToggleBlocking()
	v := s.view(r, "network")
	v.OOB = true
	render(w, s.parts, "pihole-toggle", v)
	render(w, s.parts, "nav", v)
}

func (s *server) addUser(w http.ResponseWriter, r *http.Request) {
	name, role := strings.TrimSpace(r.FormValue("name")), r.FormValue("role")
	if name == "" || roleAccess[role] == nil {
		http.Error(w, "name and a valid role are required", http.StatusUnprocessableEntity)
		return
	}
	render(w, s.parts, "user-row", s.store.AddUser(name, role))
}

func (s *server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	s.store.DeleteUser(id)
}

func (s *server) toggleAccess(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	chip, ok := s.store.ToggleAccess(id, r.PathValue("app"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	render(w, s.parts, "access-chip", chip)
}

func render(w http.ResponseWriter, t *template.Template, name string, data any) {
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

var funcs = template.FuncMap{
	"nav": func() []NavItem { return nav },
	"greeting": func() string {
		switch h := time.Now().Hour(); {
		case h < 5:
			return "Up late"
		case h < 12:
			return "Good morning"
		case h < 17:
			return "Good afternoon"
		}
		return "Good evening"
	},
	"today": func() string { return time.Now().Format("Monday, January 2") },
	"tb": func(tb float64) string {
		if tb < 1 {
			return fmt.Sprintf("%.0f GB", tb*1000)
		}
		return fmt.Sprintf("%.2f TB", tb)
	},
	"gb": func(gb float64) string {
		if gb >= 1000 {
			return fmt.Sprintf("%.2f TB", gb/1000)
		}
		return fmt.Sprintf("%.0f GB", gb)
	},
	"pct": func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b * 100
	},
	"f0":    func(v float64) string { return fmt.Sprintf("%.0f", v) },
	"f1":    func(v float64) string { return fmt.Sprintf("%.1f", v) },
	"comma": comma,
	"ago":   ago,
	"uptime": func(d time.Duration) string {
		return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24)
	},
	"spark": spark,
	"barh": func(n int, bars []Bar) float64 {
		top := 1
		for _, b := range bars {
			top = max(top, b.Q)
		}
		return float64(n) / float64(top) * 100
	},
	"share": func(n int, all []Count) float64 { return float64(n) / float64(all[0].N) * 100 },
}

func comma(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func ago(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 60*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("Jan 2006")
}

func spark(vals []float64) string {
	var b strings.Builder
	step := 200.0 / float64(len(vals)-1)
	for i, v := range vals {
		fmt.Fprintf(&b, "%.1f,%.1f ", float64(i)*step, 46-v/100*44)
	}
	return b.String()
}
