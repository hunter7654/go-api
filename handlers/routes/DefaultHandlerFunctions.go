package routes

import (
	"fmt"
	"github.com/hunter7654/go-api/database"
	"github.com/hunter7654/go-api/router"
	"net/http"
)

func init() {
	router.AddAuth(router.Route{Method: "POST", Pattern: "/examplepost", HandlerFunc: ExamplePost})
	router.AddDef(router.Route{Method: "GET", Pattern: "/exampleget/{id}/{test}", HandlerFunc: ExampleGet})
}

type exampleStruct struct {
	ExampleID   string
	ExampleName string
	ExampleData string
}

//this is an example route to show how to handle a get request
func ExampleGet(w http.ResponseWriter, r *http.Request) {
	jwtData, _ := r.Context().Value(database.MyKey).(database.JwtData)
	sql := `Enter select statement here`
	data := database.GetParameters(r)
	fmt.Fprintln(w, database.RunGet(sql, database.DatabaseConn, data["id"], data["test"], jwtData.Username))
}

//this is an example route to show how to handle a post request
func ExamplePost(w http.ResponseWriter, r *http.Request) {
	tx, jwtData, postData, params := database.GetPostData(r, database.DatabaseConn)
	defer tx.Rollback()
	sql := `Enter Insert Statement Here`
	params = append(params, "Attach Params Here", jwtData.Username, postData["data"])
	fmt.Fprintln(w, database.RunDataChange(sql, tx, params...))
	tx.Commit()
}
