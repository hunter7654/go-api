package database

import (
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/mattn/go-oci8"
	//_ "gopkg.in/rana/ora.v4"
	//_ "gopkg.in/goracle.v2"
	"database/sql"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
)

type DataSource struct {
	Connection       *sql.DB
	Driver           string
	ConnectionString string
}
type Key int

const JsonKey = "RandomKeyHere"

const MyKey Key = 0

// JWT schema of the data it will store.
type JwtData struct {
	Username interface{} `json:"username"`
	jwt.StandardClaims
}

type ErrorResponse struct {
	Error       string
	StackTrace  string
	ErrorObject error
}
//this is where the database connections are declared
var DatabaseConn = &DataSource{nil, "oci8", "username/password@ipAddress:port/databaseName"}

//this function initialised the connections when the server starts
func InitDB(source *DataSource) error {
	var err error
	source.Connection, err = sql.Open(source.Driver, source.ConnectionString)
	if err != nil {
		return err
	}
	source.Connection.SetMaxIdleConns(10)
	if err = source.Connection.Ping(); err != nil {
		return err
	}
	return nil
}

//this function runs a sql select statement to the passed data source and returns the response as a json array
func RunGet(sqlCommand string, source *DataSource, params ...interface{}) string {
	tableData := GetQueryAsArray(sqlCommand, source, params...)
	jsonData, err := json.Marshal(tableData)
	if err != nil {
		panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
	}
	return string(jsonData)
}

//this function runs an insert/update/delete/etc statement to the passed data source
func RunDataChange(sqlCommand string, source *sql.Tx, values ...interface{}) (sql.Result) {
	stmt, err := source.Prepare(sqlCommand)
	if err != nil {
		panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
	}
	res, err := stmt.Exec(values...)
	if err != nil {
		panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
	}
	stmt.Close()
	return res
}

func GetPostData(r *http.Request, database *DataSource) (tx *sql.Tx, jwtData JwtData, postData map[string]interface{}, params []interface{}) {
	jwtData, _ = r.Context().Value(MyKey).(JwtData)
	var err error
	if database.Connection == nil {
		if err = InitDB(database); err != nil {
			panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
		}
	}
	if err = database.Connection.Ping(); err != nil {
		if err = InitDB(database); err != nil {
			panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
		}
	}
	if tx, err = database.Connection.Begin(); err != nil {
		panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
	}
	if err = json.NewDecoder(r.Body).Decode(&postData); err != nil {
		if err.Error() != "EOF" {
			panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
		}
	}
	return
}

func GetParameters(r *http.Request) (data map[string]string) {
	data = mux.Vars(r)
	for k := range data {
		data[k], _ = url.QueryUnescape(data[k])
		data[k] = strings.Replace(data[k], "encodedslash", "/", -1)
	}
	return
}

func GetQueryAsArray(sqlCommand string, source *DataSource, params ...interface{}) []map[string]interface{} {
	if source.Connection == nil {
		if err := InitDB(source); err != nil {
			panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
		}
	}
	if err := source.Connection.Ping(); err != nil {
		if err := InitDB(source); err != nil {
			panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
		}
	}
	tx, err := source.Connection.Begin()
	if err != nil {
		panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(sqlCommand)
	if err != nil {
		panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
	}
	rows, err := stmt.Query(params...)
	if err != nil {
		panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		panic(ErrorResponse{err.Error(), string(debug.Stack()), err})
	}
	count := len(columns)
	tableData := make([]map[string]interface{}, 0)
	values := make([]interface{}, count)
	valuePtrs := make([]interface{}, count)
	for rows.Next() {
		for i := 0; i < count; i++ {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
		entry := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = b
			} else {
				v = val
			}
			entry[col] = v
		}
		tableData = append(tableData, entry)
	}
	stmt.Close()
	tx.Commit()
	return tableData
}
