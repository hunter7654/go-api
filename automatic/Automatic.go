package automatic

import (
	"github.com/hunter7654/go-api/database"
	"encoding/json"
	"fmt"
	"github.com/getsentry/raven-go"
	"time"
)

func Start() {
	go doEvery(60*time.Minute, TestFunc)
}

func doEvery(d time.Duration, f func()) {
	defer handleError()
	for range time.Tick(d) {
		f()
	}
}

func TestFunc() {
	//do something here
}

func handleError() {
	if r := recover(); r != nil {
		if response, ok := r.(database.ErrorResponse); ok {
			if response.ErrorObject != nil {
				raven.CaptureError(response.ErrorObject, nil)
			}
			data, _ := json.Marshal(response)
			fmt.Println(string(data))
			return
		}
	}
}
