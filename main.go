package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

var templatesPath = "static/templates/"

var pornImagesPath = "static/images/"

var postsPath = "static/templates/posts/"
var tmpl = template.Must(template.ParseFiles(
	templatesPath+"index.html",
	templatesPath+"index_search.html",

	templatesPath+"pronlist.html",
	templatesPath+"onephoto.html"))

var header, _ = ioutil.ReadFile("static/templates/post_header.html")
var footer, _ = ioutil.ReadFile("static/templates/post_footer.html")

//  porn  ///////////////////////////////////////////
type Image struct {
	URL    string
	PrevId int
	Id     int
	NextId int
}

type Images struct {
	Images []Image
	PrevId int
	Id     int
	NextId int
}

func pornImages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	files, _ := ioutil.ReadDir("static/pron/")
	varid := vars["id"]
	id, _ := strconv.Atoi(varid)
	prevId := id - 1
	if prevId < 1 {
		prevId = 1
	}
	nextId := id + 1
	if nextId > len(files) {
		nextId = id
	}

	im := Image{"static/pron/img" + varid, prevId, id, nextId}
	tmpl.ExecuteTemplate(w, "onephoto.html", im)
}
func pornImagesMain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	files, _ := ioutil.ReadDir("static/pron/")
	PicsCount := 12
	firstId, _ := strconv.Atoi(vars["id"])

	diff := len(files) - firstId

	prevId := firstId - PicsCount
	if prevId < 0 {
		prevId = 0
	}

	if diff >= 0 {
		if diff < PicsCount {
			PicsCount = diff
		}
	} else {
		PicsCount = 0
	}

	nextId := firstId + PicsCount
	if nextId > len(files) {
		nextId = firstId
	}
	imgs := Images{make([]Image, PicsCount), prevId, firstId, firstId + PicsCount}
	for i := firstId; i < firstId+PicsCount; i++ {
		curI := i + 1
		imgs.Images[i-firstId] = Image{"static/pron/img" + strconv.Itoa(curI), 0, curI, 0}
	}
	tmpl.ExecuteTemplate(w, "pronlist.html", imgs)
}

////////////////////////////////////////////////////

func index(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "index.html", nil)
}

type Post struct {
	Name string
	Id   int
}

func indexSearch(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")

	query = strings.ToLower(query)
	words := strings.Split(query, " ")

	files, _ := ioutil.ReadDir(postsPath)
	SearchedFiles := make([]Post, 0, len(files))
	for idy, file := range files {
		for _, word := range words {
			fileName := file.Name()
			if strings.Contains(fileName, word) && !file.IsDir() && len(strings.TrimSpace(word)) > 0 {
				SearchedFiles = append(SearchedFiles, Post{fileName, idy})
			}
		}
	}
	tmpl.ExecuteTemplate(w, "index_search.html", SearchedFiles)
}
func showpost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	varid := vars["id"]

	input, err := ioutil.ReadFile(postsPath + varid)
	if err != nil {
		panic(err)
	}
	unsafe := blackfriday.MarkdownCommon(input)
	html := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	fmt.Fprint(w, string(header)+string(html)+string(footer))
}
func neuter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") || r.URL.Path == "" {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func main() {

	if false {
		fmt.Println("Preparing images")
		cmd := exec.Command("python", "rename.py")
		cmd.Dir = pornImagesPath
		out, err := cmd.Output()
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(string(out))
	}

	fmt.Println("Set up routers")
	rtr := mux.NewRouter()

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", neuter(fs)))

	rtr.HandleFunc("/pron{id:[0-9]+}", pornImagesMain).Methods("GET")
	rtr.HandleFunc("/pronf{id:[0-9]+}", pornImages).Methods("GET")
	rtr.HandleFunc("/pron", pornImagesMain).Methods("GET")

	rtr.HandleFunc("/", index).Methods("GET")
	rtr.HandleFunc("/search", indexSearch).Methods("POST")
	rtr.HandleFunc("/post/{id}", showpost).Methods("GET")
	http.Handle("/", rtr)

	fmt.Println("Starting henkServer")
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		panic(err)
	}
}
