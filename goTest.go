package main

// Created with help from https://go.dev/doc/articles/wiki/
// Added on to the original code to allow for links to other pages and the webroot to redirect to the FrontPage

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var templates = template.Must(template.ParseFiles("edit.html", "view.html"))
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

type Page struct {
	Title         string
	Body          []byte
	UnEscapedBody template.HTML
}

// Handles the saving of webpages, by saving a page struct to a file
func (p *Page) save() error {
	filename := p.Title + ".txt"
	return os.WriteFile(filename, p.Body, 0600)
}

// Renders the template to a page with the given page struct
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Reads a file and returns a page struct with the contents of the file
func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

// Handles the editing of a page, by loading the page and rendering the edit template, which places the loaded body into a fillable text area
func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

// Replaces all instances of [text] with <a href="/view/text">text</a>
func deepReplacement(b []byte) []byte {
	body := string(b)
	for i := 0; i < len(body); i++ {
		if body[i] == '[' && (i-1 > 0 && body[i-1] != '\\') {
			for j := i + 1; j < len(body); j++ {
				if body[j] == ']' {
					fmt.Println(body[i+1 : j])
					body = strings.Replace(body, body[i:j+1], "<a href=\"/view/"+body[i+1:j]+"\">"+body[i+1:j]+"</a>", 1)
					break
				}
			}
		}
	}
	return []byte(body)
}

// Handles the user viewing a page, by loading the page, then rendering it with the view template
func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	p.UnEscapedBody = template.HTML(deepReplacement(p.Body))
	renderTemplate(w, "view", p)
}

// Handles saves from the edit page, by saving the body of the page to a file, then redirecting to the view page
func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	p.save()
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

// Redirects the webroot to the FrontPage
func frontPageRedirect(w http.ResponseWriter, r *http.Request, title string) {
	http.Redirect(w, r, "/view/FrontPage", http.StatusFound)
}

// Handler for the handlers, which determines which handler to use based on the URL and protects against invalid URLs
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			fn(w, r, "FrontPage")
			return
		}
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func main() {
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/", makeHandler(frontPageRedirect))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
