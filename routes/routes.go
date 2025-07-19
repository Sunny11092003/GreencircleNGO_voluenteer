package routes

import (
	"geotagging/treehandler"
	"net/http"

	"github.com/gorilla/mux"
)

// routes/routes.go
func SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Home route
	r.HandleFunc("/", treehandler.Handleopening)

	// Other API routes
	r.HandleFunc("/home", treehandler.HandleHome).Methods("GET")
	r.HandleFunc("/api/treecount", treehandler.HandleTreeCount).Methods("GET")
	r.HandleFunc("/append-image", treehandler.AppendImageHandler)
	r.HandleFunc("/signup", treehandler.SignupHandler).Methods("GET", "POST")
	r.HandleFunc("/signin", treehandler.LoginHandler).Methods("GET", "POST")
	r.HandleFunc("/generate", treehandler.GenerateTreeHandler).Methods("GET", "POST")
	r.HandleFunc("/publishsave/{uid}", treehandler.PublishHandler1).Methods("POST")
	r.HandleFunc("/qr", treehandler.ServeQRHandler).Methods("GET")
	r.HandleFunc("/download-pdf", treehandler.DownloadPDFHandler)
	r.HandleFunc("/qr-display", treehandler.ServeQRHandlertree).Methods("GET", "POST")
	r.HandleFunc("/download-pdf-tree", treehandler.DownloadPDFHandlertree).Methods("GET")
	r.HandleFunc("/delete", treehandler.DeleteTreeHandler).Methods("POST")
	r.HandleFunc("/data_entry", treehandler.DataEntryHandler).Methods("GET", "POST")
	r.HandleFunc("/location", treehandler.LocationHandler).Methods("GET", "POST")
	r.HandleFunc("/image", treehandler.UploadImagesHandler).Methods("GET", "POST")
	r.HandleFunc("/complete", treehandler.CompleteHandler)
	r.HandleFunc("/publish", treehandler.PublishHandler)
	r.HandleFunc("/saveai/{uid}", treehandler.SaveAIHandler).Methods("POST")
	r.HandleFunc("/drafts", treehandler.ServeQRHandlertreedrafts).Methods("GET", "POST")
	r.HandleFunc("/generate-direct/{uid}", treehandler.GenerateDirectQR)
	//r.HandleFunc("/download-pdf-tree-drafts", treehandler.DownloadPDFHandlertreedrafts).Methods("GET")
	r.HandleFunc("/delete-drafts", treehandler.DeleteTreeHandlerdrafts).Methods("POST")
	r.HandleFunc("/classification", treehandler.ClassificationHandler).Methods("GET", "POST")
	r.HandleFunc("/library", treehandler.ServelibraryHandlertree).Methods("GET")
	r.HandleFunc("/api/treecount", treehandler.HandleTreeCount).Methods("GET")
	r.HandleFunc("/list", treehandler.ListTreesHandler).Methods("GET")
	r.HandleFunc("/genera", treehandler.ListGeneraTreesHandler)
	r.HandleFunc("/species", treehandler.ListSpeciesTreesHandler)
	r.HandleFunc("/list/{category}", treehandler.ListTreesHandler).Methods("GET")
	r.HandleFunc("/report", treehandler.ReportHandler).Methods("POST")
	r.HandleFunc("/identify", treehandler.IdentifyHandler).Methods("POST")
	r.HandleFunc("/getdetails", treehandler.GetTreeDetailsHandler).Methods("POST")
	r.HandleFunc("/delete-tree", treehandler.DeleteTreeHandleridentify)
	r.HandleFunc("/get-event", treehandler.GetEventHandler).Methods("GET")
	r.HandleFunc("/change-password", treehandler.ChangePasswordHandler).Methods("POST")
	r.HandleFunc("/settings", treehandler.RenderSettingPage).Methods("GET")
	r.HandleFunc("/delete-image", treehandler.DeleteImageHandler).Methods("POST")

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	return r
}
