package main

import (
	"flag"
	"github.com/getsentry/raven-go"
	"github.com/gorilla/handlers"
	"github.com/hunter7654/go-api/automatic"
	"github.com/hunter7654/go-api/database"
	_ "github.com/hunter7654/go-api/handlers/routes"
	"github.com/hunter7654/go-api/router"
	"net/http"
	"os"
	"runtime"
)

func init() {
	//sentry path
	raven.SetDSN("")
}
func main() {

	portPtr := flag.String("port", "25566", "The port to run the server on.")
	flag.Parse()

	//initialises DatabaseConn connection
	if err := database.InitDB(database.DatabaseConn); err != nil {
		raven.CaptureError(err, nil)
	}
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "origin", "content-type", "Authorization", "authorization"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"})
	go automatic.Start()
	go http.ListenAndServe(":"+*portPtr, handlers.LoggingHandler(os.Stdout, handlers.CORS(originsOk, headersOk, methodsOk)(router.NewRouter())))
	runtime.Goexit()
}
