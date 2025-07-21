package routes

import (
	"geotagging/treehandler"
	"net/http"

	"github.com/gin-gonic/gin"
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

	/* ────── Head/Admin Dashboard ────── */
	r.HandleFunc("/head_dashboard", treehandler.HeadDashboard).Methods("GET")

	/* ────── Verified Volunteers ────── */
	r.HandleFunc("/head/head_verified", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/head_verified.html")
	}).Methods("GET")

	r.HandleFunc("/head/verified_list", func(w http.ResponseWriter, r *http.Request) {
		ginContext, _ := gin.CreateTestContext(w)
		treehandler.GetVerifiedVolunteers(ginContext)
	}).Methods("GET")

	r.HandleFunc("/update_volunteer", func(w http.ResponseWriter, r *http.Request) {
		ginContext, _ := gin.CreateTestContext(w)
		treehandler.UpdateVolunteerPermission(ginContext)
	}).Methods("POST")

	r.HandleFunc("/revoke_volunteer", func(w http.ResponseWriter, r *http.Request) {
		ginContext, _ := gin.CreateTestContext(w)
		treehandler.RevokeVolunteer(ginContext)
	}).Methods("GET")

	/* ────── Pending Volunteers ────── */
	r.HandleFunc("/head/pending", treehandler.HeadPending).Methods("GET")
	r.HandleFunc("/head/approve", treehandler.HeadApprove).Methods("POST")
	r.HandleFunc("/head/reject", treehandler.HeadReject).Methods("POST")
	r.HandleFunc("/validator", treehandler.LoginHandlervalidator).Methods("GET", "POST")
	r.HandleFunc("/admin", treehandler.LoginHandleradmin).Methods("GET", "POST")

	// Admin Dashboard route
	r.HandleFunc("/admin/dashboard", treehandler.AdminDashboardHandler)

	r.HandleFunc("/admin/edit/{id}", treehandler.EditTreeHandler)
	r.HandleFunc("/admin/delete/{id}", treehandler.DeleteTreeHandleradmin)

	// main.go  (add just after your dashboard route)
	r.HandleFunc("/admin/edit/{id}", treehandler.EditTreeHandler).Methods("GET", "POST")

	// API endpoints
	r.HandleFunc("/api/volunteers", treehandler.GetAllUsersHandler).Methods("GET")
	r.HandleFunc("/api/updateRole", treehandler.UpdateUserRoleHandler).Methods("POST")

	// HTML page
	r.HandleFunc("/volunteers", treehandler.ServeVolunteersPage).Methods("GET")

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	return r
}
