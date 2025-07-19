package treehandler

import (
	"net/http"
)

func Handleopening(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/opening.html")
}
