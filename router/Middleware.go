package router

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/getsentry/raven-go"
	"github.com/hunter7654/go-api/database"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

//initialises the json web token for the current user
func SetToken(Username interface{}, Signingkey []byte) string {
	expireToken := time.Now().Add(time.Hour * 1).Unix()
	jwtData := database.JwtData{
		Username: Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireToken,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtData)

	signedToken, _ := token.SignedString(Signingkey)

	return signedToken
}

// makes sure that the incoming request has a valid json web token and either approves or denies the access
func Validate(page http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		var tokenJson string
		tokenHeader, ok := req.Header["Authorization"]
		if ok && len(tokenHeader) >= 1 {
			tokenJson = strings.TrimPrefix(tokenHeader[0], "Bearer ")
		}

		if tokenJson == "" {
			http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		var tokenArray map[string]string
		if err := json.Unmarshal([]byte(tokenJson), &tokenArray); err != nil {
			panic(database.ErrorResponse{Error: err.Error(), StackTrace: string(debug.Stack())})
		}

		parsedToken, err := jwt.ParseWithClaims(tokenArray["token"], &database.JwtData{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(database.JsonKey), nil
		})
		if err != nil {
			http.Error(res, err.Error(), http.StatusUnauthorized)
			return
		}
		if jwtData, ok := parsedToken.Claims.(*database.JwtData); ok && parsedToken.Valid {
			ctx := context.WithValue(req.Context(), database.MyKey, *jwtData)
			page(res, req.WithContext(ctx))
		} else {
			http.Error(res, err.Error(), http.StatusUnauthorized)
			return
		}
	})
}

//this function handles any errors and stops the program from crashing when it encounters them.
func HandleError(page http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				if response, ok := r.(database.ErrorResponse); ok {
					if response.ErrorObject != nil {
						raven.CaptureError(response.ErrorObject, nil)
					}
					data, _ := json.Marshal(response)
					http.Error(res, string(data), http.StatusInternalServerError)
					return
				}

			}
		}()
		page(res, req)
	})
}

func LogTime(page http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()
		page(res, req)
		t := time.Now()
		elapsed := t.Sub(start)
		if elapsed.Seconds() > 5 {
			log.Println("URL:	" + req.URL.String() + "	Time taken:	" + fmt.Sprint(elapsed.Seconds()))
		}
	})
}
