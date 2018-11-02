package routes

import (
	"github.com/hunter7654/go-api/database"
	"github.com/hunter7654/go-api/router"
	"encoding/json"
	"github.com/jtblin/go-ldap-client"
	"net/http"
	"runtime/debug"
)

type loginStruct struct {
	Username string
	Password string
}

func init() {
	router.AddDef(router.Route{Method: "POST", Pattern: "/login", HandlerFunc: Login})
	router.AddAuth(router.Route{Method: "GET", Pattern: "/tknrefresh", HandlerFunc: RefreshToken})
}

//this route refreshes the json web token to keep the user logged in between page refreshes
func RefreshToken(w http.ResponseWriter, r *http.Request) {
	jwtData, _ := r.Context().Value(database.MyKey).(database.JwtData)
	token := router.SetToken(jwtData.Username, []byte(database.JsonKey))
	response, _ := json.Marshal(token)
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

//this route handles logins
func Login(w http.ResponseWriter, r *http.Request) {
	var loginParams loginStruct
	err := json.NewDecoder(r.Body).Decode(&loginParams)
	if err != nil {
		panic(database.ErrorResponse{Error: err.Error(), StackTrace: string(debug.Stack()), ErrorObject: err})
	}
	ok, username := Authenticate(loginParams.Username, loginParams.Password)
	if ok {
		token := router.SetToken(username, []byte(database.JsonKey))
		response, _ := json.Marshal(token)
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	} else {
		http.Error(w, "Incorrect Username/password", http.StatusUnauthorized)
	}
}

// LDAP auth
func Authenticate(username string, password string) (bool, string) {
	client := &ldap.LDAPClient{
		Base:         "",
		Host:         "",
		Port:         0,
		UseSSL:       false,
		BindDN:       "",
		BindPassword: "",
		UserFilter:   "",
		Attributes:   []string{""},
	}
	defer client.Close()
	ok, user, err := client.Authenticate(username, password)
	if err != nil {
		return false, ""
	}
	if !ok {
		return false, ""
	}
	return true, user[""]
}
