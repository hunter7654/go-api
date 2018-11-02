package webservices

import (
	"github.com/hunter7654/go-api/database"
	"github.com/hunter7654/go-api/router"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

func init() {
	router.AddAuth(router.Route{Method: "GET", Pattern: "/webservices/database/{schema_name}/{table_name}/{json}", HandlerFunc: Get})
	router.AddAuth(router.Route{Method: "POST", Pattern: "/webservices/database/{schema_name}/{table_name}", HandlerFunc: Insert})
	router.AddAuth(router.Route{Method: "PUT", Pattern: "/webservices/database/{schema_name}/{table_name}", HandlerFunc: Update})
}

/*
Webservices Guide.

Get:
	Takes a schema and a table name and returns all data in that table.
	E.g. webservices/database/schema/table would return all data in the SCHEMA.TABLE table.

	This webservice is not much use unless its for lookup tables for select fields and
	other things where you require all data.

Post:
	Takes a schema name, table name and a json array of columns and values
	and inserts them in to the selected table. It then returns the id of the inserted value.

	This requires the table to contain the following columns:
	ID(Optional if not using a sequence), CREATED_DATE, CREATED_BY

	It is also recommended to have a sequence named the same as the table with
	SEQ_ before the name so for example SCHEMA.SEQ_TABLE. This will allow the
	function to add an ID automatically and not require you to pass one in.

	The post data should be passed as a json array with the keys being the names of
	the columns that the data is to be inserted to.
	E.g. {COL1:"Val1", COL2:"Val2", COL3:"Val3"}

Put:
	This works almost the same way as the post method.
	It takes a schema name, table name and json array of values exactly the same as the post method.

	The differences are that it requires the table to have UPDATED_BY and UPDATED_DATE columns and
	that you must also pass a where parameter to it. This is done by adding the columns that you
	would like to search by to the posted json array. To specify that a column is part of the
	where clause the column name needs to be followed by a colon and then the type of
	comparison that needs to be done. They are as follows:
	=       	-- Column equals value
	>=      	-- Column equal to or greater than value
	<=      	-- Column less than or equal to
	!=      	-- Column does not equal
	null    	-- Column is null
	notnull 	-- Column is not null
	in 		  	-- Column is in array
						(To use this the passed parameter
						needs to be a string of values
						separated by a comma) E.g. "COL1:in":"Val1,Val2,Val3"
	Example json array:
	{
	COL1 : Val1,
	"COL2:=" : "Val2",
	"COL3:>=" : 0,
	"COL4:<=" : 1,
	"COL5:!=" : 2,
	"COL6:null" : "",
	"COL7:notnull" : "",
	"COL8:in" : "Val1,Val2,Val3",
	}

 */
func Get(w http.ResponseWriter, r *http.Request) {
	var postData map[string]interface{}
	var params []interface{}
	data := database.GetParameters(r)
	if err := json.Unmarshal([]byte(data["json"]), &postData); err != nil {
		panic(err)
	}
	CheckValidParameters(data, postData)
	whereData := make(map[string]interface{}, 0)
	for columnName, value := range postData {
		if len(strings.Split(columnName, ":")) > 1 {
			whereData[columnName] = value
			delete(postData, columnName)
		}
	}
	sql := `SELECT * FROM ` + data["schema_name"] + `.` + data["table_name"] + ` WHERE `
	for columnName, data := range whereData {
		data = ConvertToOracleDate(data)
		split := strings.Split(columnName, ":")
		switch split[1] {
		case "=":
			sql += split[0] + " = :v AND "
			params = append(params, data)
			break
		case ">=":
			sql += split[0] + " >= :v AND "
			params = append(params, data)
			break
		case "<=":
			sql += split[0] + " <= :v AND "
			params = append(params, data)
			break
		case "!=":
			sql += split[0] + " != :v AND "
			params = append(params, data)
			break
		case "null":
			sql += split[0] + " IS NULL AND "
			break
		case "notnull":
			sql += split[0] + " IS NOT NULL AND "
			break
		case "in":
			sql += split[0] + " IN("
			for temp := range strings.Split(data.(string), ",") {
				sql += ":v, "
				params = append(params, temp)
			}
			sql = sql[0 : len(sql)-2]
			sql += ") AND "
			break
		default:
			panic(database.ErrorResponse{Error: "Unknown comparator in where clause", StackTrace: string(debug.Stack())})
		}

	}
	sql = sql[0 : len(sql)-5]
	fmt.Fprintln(w, database.RunGet(sql, database.DatabaseConn /*, params...*/))
}

func Insert(w http.ResponseWriter, r *http.Request) {
	tx, jwtData, postData, params := database.GetPostData(r, database.DatabaseConn)
	defer tx.Rollback()
	data := database.GetParameters(r)
	CheckValidParameters(data, postData)
	tableData := make([]map[string]interface{}, 0)
	sql := `SELECT DISTINCT OBJECT_NAME FROM DBA_OBJECTS WHERE OBJECT_TYPE = 'SEQUENCE' AND OWNER = UPPER(:v)`
	err := json.Unmarshal([]byte(database.RunGet(sql, database.DatabaseConn, data[`schema_name`])), &tableData)
	if err != nil {
		panic(database.ErrorResponse{Error: err.Error(), StackTrace: string(debug.Stack()), ErrorObject: err})
	}
	insertId := 0
	if stringInSlice(strings.ToUpper("SEQ_"+data["table_name"]), tableData) {
		sql = `select ` + data["schema_name"] + `.SEQ_` + data["table_name"] + `.nextval id from dual`
		err := json.Unmarshal([]byte(database.RunGet(sql, database.DatabaseConn)), &tableData)
		if err != nil {
			panic(database.ErrorResponse{Error: err.Error(), StackTrace: string(debug.Stack()), ErrorObject: err})
		}
		switch tableData[0]["ID"].(type) {
		case float64:
			insertId = int(tableData[0]["ID"].(float64))
			if err != nil {
				panic(database.ErrorResponse{Error: err.Error(), StackTrace: string(debug.Stack()), ErrorObject: err})
			}
		case string:
			insertId, err = strconv.Atoi(tableData[0]["ID"].(string))
			if err != nil {
				panic(database.ErrorResponse{Error: err.Error(), StackTrace: string(debug.Stack()), ErrorObject: err})
			}
		default:
			panic(database.ErrorResponse{Error: "ID type not recognised", StackTrace: string(debug.Stack()), ErrorObject: err})
		}

	}
	sql = `INSERT INTO ` + data["schema_name"] + `.` + data["table_name"] + `(CREATED_DATE, CREATED_BY, `
	params = append(params, jwtData.Username)
	if insertId > 0 {
		sql += "id, "
		params = append(params, insertId)
	}
	for columnName, data := range postData {
		data = ConvertToOracleDate(data)
		sql += columnName + `, `
		params = append(params, data)
	}
	sql = sql[0 : len(sql)-2]
	sql += `) values (SYSDATE, :v, `
	if insertId > 0 {
		sql += `:v, `
	}
	for range postData {
		sql += `:v, `
	}
	sql = sql[0 : len(sql)-2]
	sql += `)`
	database.RunDataChange(sql, tx, params...)
	fmt.Fprintln(w, insertId)
	tx.Commit()
}

func Update(w http.ResponseWriter, r *http.Request) {
	tx, jwtData, postData, params := database.GetPostData(r, database.DatabaseConn)
	defer tx.Rollback()
	data := database.GetParameters(r)
	CheckValidParameters(data, postData)
	whereData := make(map[string]interface{}, 0)
	for columnName, value := range postData {
		if len(strings.Split(columnName, ":")) > 1 {
			whereData[columnName] = value
			delete(postData, columnName)
		}
	}
	sql := `UPDATE ` + data["schema_name"] + `.` + data["table_name"] + ` SET UPDATED_DATE = SYSDATE ,UPDATED_BY = :v, `
	params = append(params, jwtData.Username)
	for columnName, data := range postData {
		data = ConvertToOracleDate(data)
		sql += columnName + ` = :v, `
		params = append(params, data)
	}
	sql = sql[0 : len(sql)-2]
	sql += ` WHERE `
	for columnName, data := range whereData {
		data = ConvertToOracleDate(data)
		split := strings.Split(columnName, ":")
		switch split[1] {
		case "=":
			sql += split[0] + " = :v AND "
			params = append(params, data)
			break
		case ">=":
			sql += split[0] + " >= :v AND "
			params = append(params, data)
			break
		case "<=":
			sql += split[0] + " <= :v AND "
			params = append(params, data)
			break
		case "!=":
			sql += split[0] + " != :v AND "
			params = append(params, data)
			break
		case "null":
			sql += split[0] + " IS NULL AND "
			break
		case "notnull":
			sql += split[0] + " IS NOT NULL AND "
			break
		case "me":
			sql += split[0] + " = :v AND "
			params = append(params, jwtData.Username)
			break
		case "in":
			sql += split[0] + " IN("
			for temp := range strings.Split(data.(string), ",") {
				sql += ":v, "
				params = append(params, temp)
			}
			sql = sql[0 : len(sql)-2]
			sql += ") AND "
			break
		default:
			panic(database.ErrorResponse{Error: "Unknown comparator in where clause", StackTrace: string(debug.Stack())})
		}

	}
	sql = sql[0 : len(sql)-5]
	database.RunDataChange(sql, tx, params...)
	fmt.Fprintln(w, "Record successfully updated")
	tx.Commit()
}

func CheckValidParameters(data map[string]string, postData map[string]interface{}) {
	tableData := make([]map[string]interface{}, 0)
	sql := `select DISTINCT username from dba_users`
	err := json.Unmarshal([]byte(database.RunGet(sql, database.DatabaseConn)), &tableData)
	if err != nil {
		panic(database.ErrorResponse{Error: err.Error(), StackTrace: string(debug.Stack()), ErrorObject: err})
	}
	if !stringInSlice(strings.ToUpper(data["schema_name"]), tableData) {
		panic(database.ErrorResponse{Error: "schema name not recognised", StackTrace: string(debug.Stack())})
	}
	sql = `SELECT DISTINCT OBJECT_NAME FROM DBA_OBJECTS WHERE OBJECT_TYPE = 'TABLE' AND OWNER = UPPER(:v)`
	json.Unmarshal([]byte(database.RunGet(sql, database.DatabaseConn, data[`schema_name`])), &tableData)
	if !stringInSlice(strings.ToUpper(data["table_name"]), tableData) {
		panic(database.ErrorResponse{Error: "table name not recognised", StackTrace: string(debug.Stack())})
	}
	if postData != nil {
		sql = `SELECT column_name FROM all_tab_cols WHERE owner = UPPER(:v) AND table_name = UPPER(:v)`
		json.Unmarshal([]byte(database.RunGet(sql, database.DatabaseConn, data[`schema_name`], data[`table_name`])), &tableData)
		for columnName, value := range postData {
			if value == nil && len(strings.Split(columnName, ":")) < 1 {
				delete(postData, columnName)
			}
			if !stringInSlice(strings.ToUpper(strings.Split(columnName, ":")[0]), tableData) {
				panic(database.ErrorResponse{Error: "column name not recognised : " + columnName, StackTrace: string(debug.Stack())})
			}
		}
	}
}

func ConvertToOracleDate(data interface{}) (interface{}) {
	if stringData, ok := data.(string); ok {
		if stringData != "" {
			if t, err := time.Parse("02-01-2006", stringData); err == nil {
				data = t
				return data
			}
			if t, err := time.Parse("2-1-2006", stringData); err == nil {
				data = t
				return data
			}
			if t, err := time.Parse("02/01/2006", stringData); err == nil {
				data = t
				return data
			}
			if t, err := time.Parse("2/1/2006", stringData); err == nil {
				data = t
				return data
			}
			if t, err := time.Parse("02-01-2006 15:04", stringData); err == nil {
				data = t
				return data
			}
			if t, err := time.Parse("2-1-2006 15:4", stringData); err == nil {
				data = t
				return data
			}
			if t, err := time.Parse("02/01/2006 15:04", stringData); err == nil {
				data = t
				return data
			}
			if t, err := time.Parse("2/1/2006 15:4", stringData); err == nil {
				data = t
				return data
			}
			if t, err := time.Parse("2006-01-02T15:04:05.000Z", stringData); err == nil {
				data = t
				return data
			}
			if t, err := time.Parse("2006-01-02T15:04:05-07:00", stringData); err == nil {
				data = t
				return data
			}
		}
	}
	return data
}

func stringInSlice(a string, list []map[string]interface{}) bool {
	for _, data := range list {
		for _, b := range data {
			if b == a {
				return true
			}
		}
	}
	return false
}
