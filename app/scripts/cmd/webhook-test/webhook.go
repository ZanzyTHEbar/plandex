package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"plandex-server/types"
)

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// webhookHandler handles the incoming POST request
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	//var data map[string]interface{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}

	var payload RequestPayload

	err = json.Unmarshal(body, &payload)
	if err != nil {

		err = fmt.Errorf("Error unmarshalling JSON: %v", err)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	var args types.ChangesWithLineNums
	err = json.Unmarshal([]byte(payload.Output[0].Function.Arguments), &args)
	if err != nil {
		http.Error(w, "Failed to parse arguments field", http.StatusInternalServerError)
		return
	}

	// Get the `preBuildState` from context.vars
	preBuildState := payload.Context.Vars.PreBuildState

	// Get the `postBuildState` from context.vars
	postBuildState := payload.Context.Vars.PostBuildState

	// Get the filePath from context.vars
	filePath := payload.Context.Vars.FilePath

	// Perform the diff and evaluation
	pass, score, err := compareAppliedDiffs(preBuildState, postBuildState, filePath, args.Changes)

	if err != nil {
		log.Printf("Error comparing applied diffs: %v\n", err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create and send the response payload
	response := ResponsePayload{
		Pass:  pass,
		Score: score,
	}

	//fmt.Printf("Received JSON: %v\n", data)

	//write data to a file
	//file, err := os.Create("data.json")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//defer file.Close()
	//
	//encoder := json.NewEncoder(file)
	//
	//err = encoder.Encode(data)
	//
	//if err != nil {
	//	log.Fatal(err)
	//}

	// Process the dynamic JSON with the provided logic
	//response, err := processDynamicJSON(data)
	//if err != nil {
	//	fmt.Printf("Error processing dynamic JSON: %v\n", err)
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	http.HandleFunc("/health", healthCheckHandler)
	fmt.Println("Starting server on port 9797...")
	log.Fatal(http.ListenAndServe(":9797", nil))
}
