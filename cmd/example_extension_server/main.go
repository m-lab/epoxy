// Copyright 2016 ePoxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//////////////////////////////////////////////////////////////////////////////

// The example_extension_server demonstrates how a simple HTTP server
// can receive and respond to requests from the ePoxy server's extension API.
//
// The ePoxy server must have an extension registered that maps an operation name
// to this server, e.g. "operation" -> "http://localhost:8001/operation"
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/m-lab/epoxy/extension"
)

// Definition of return message. This may have any structure.
type returnMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// operationHandler is an http.HandlerFunc for responding to an epoxy extension
// Request.
func operationHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: verify this is a POST request.
	// TODO: verify this is from a trusted source.
	var result *returnMessage

	// Decode the request.
	ext := &extension.Request{}
	err := ext.Decode(r.Body)
	// Prepare and respond to caller.
	if err != nil {
		result = &returnMessage{
			Status:  "error",
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		fmt.Println("Request:", ext.Encode())
		result = &returnMessage{
			Status:  "success",
			Message: time.Now().String(),
		}
		w.WriteHeader(http.StatusOK)
	}
	raw, _ := json.MarshalIndent(result, "", "    ")
	w.Write(raw)

	// Log data sent to caller.
	fmt.Println("Response:", string(raw))
}

func main() {
	http.HandleFunc("/operation", operationHandler)
	log.Fatal(http.ListenAndServe(":8001", nil))
}
