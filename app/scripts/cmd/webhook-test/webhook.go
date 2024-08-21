package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
)

type ContextVars struct {
	Changes        string `json:"changes"`
	FilePath       string `json:"filePath"`
	PostBuildState string `json:"postBuildState"`
	PreBuildState  string `json:"preBuildState"`
}

type Context struct {
	Prompt string      `json:"prompt"`
	Vars   ContextVars `json:"vars"`
}

type Function struct {
	Arguments string `json:"arguments"`
	Name      string `json:"name"`
}

type Output struct {
	Function Function `json:"function"`
	ID       string   `json:"id"`
	Type     string   `json:"type"`
}

type RequestPayload struct {
	Context Context  `json:"context"`
	Output  []Output `json:"output"`
}

type ArgumentsChanges struct {
	Summary   string `json:"summary"`
	HasChange bool   `json:"hasChange"`
	Old       struct {
		StartLineString string `json:"startLineString"`
		EndLineString   string `json:"endLineString"`
	} `json:"old"`
	StartLineIncludedReasoning string `json:"startLineIncludedReasoning"`
	StartLineIncluded          bool   `json:"startLineIncluded"`
	EndLineIncludedReasoning   string `json:"endLineIncludedReasoning"`
	EndLineIncluded            bool   `json:"endLineIncluded"`
	New                        string `json:"new"`
}

type Arguments struct {
	FilePath string             `json:"filePath"`
	Changes  []ArgumentsChanges `json:"changes"`
	Comments []struct {
		Txt       string `json:"txt"`
		Reference bool   `json:"reference"`
	} `json:"comments"`
}

// ResponsePayload represents the response that the webhook will send back
type ResponsePayload struct {
	Pass   bool    `json:"pass"`
	Score  float64 `json:"score,omitempty"`  // Score is optional
	Reason string  `json:"reason,omitempty"` // Reason is optional
}

// processDynamicJSON processes the dynamic JSON data and implements the provided logic
func processDynamicJSON(data map[string]interface{}) (ResponsePayload, error) {
	var postBuildState, newChanges string
	var pass bool
	var score float64
	var reason string

	responseBody := ResponsePayload{}

	// Navigate through the dynamic JSON structure
	context, ok := data["context"].(map[string]interface{})

	if !ok {
		return responseBody, fmt.Errorf("missing 'context' key in JSON")
	}

	vars, ok := context["vars"].(map[string]interface{})

	if !ok {
		return responseBody, fmt.Errorf("missing 'vars' key in JSON")
	}

	// Extract the postBuildState
	pbs, ok := vars["postBuildState"].(string)

	if !ok {
		return responseBody, fmt.Errorf("missing 'postBuildState' key in JSON")
	}

	postBuildState = pbs

	// Look for the changes key within the function arguments
	output, ok := data["output"].(map[string]interface{})

	if !ok {
		return responseBody, fmt.Errorf("missing 'output' key in JSON")
	}

	function, ok := output["function"].(map[string]interface{})

	if !ok {
		return responseBody, fmt.Errorf("missing 'function' key in JSON")
	}

	// check for arguments key
	args, ok := function["arguments"].(string)

	if !ok {
		return responseBody, fmt.Errorf("missing 'arguments' key in JSON")
	}

	var parsedArgs map[string]interface{}
	err := json.Unmarshal([]byte(args), &parsedArgs)
	if err != nil {
		return responseBody, fmt.Errorf("error parsing 'arguments' JSON: %v", err)
	}

	changes, ok := parsedArgs["changes"].([]interface{})

	if !ok || len(changes) == 0 {
		return responseBody, fmt.Errorf("missing 'changes' key in JSON")
	}

	change, ok := changes[0].(map[string]interface{})

	if !ok {
		return responseBody, fmt.Errorf("missing 'changes' key in JSON")
	}

	newVal, ok := change["new"].(string)

	if !ok {
		return responseBody, fmt.Errorf("missing 'new' key in JSON")
	}

	newChanges = newVal

	fmt.Printf("postBuildState: %s\n", postBuildState)
	fmt.Printf("newChanges: %s\n", newChanges)

	// Apply the evaluation logic
	pass, score, reason = diffAndEvaluate(newChanges, postBuildState)

	responseBody.Pass = pass
	responseBody.Score = score
	responseBody.Reason = reason

	return responseBody, nil
}

// cosineSimilarity calculates the cosine similarity between two strings - used for similarity comparison across large and varied text
func cosineSimilarity(a, b string) float64 {
	vecA := getWordFrequency(a)
	vecB := getWordFrequency(b)

	dotProduct := 0.0
	magA := 0.0
	magB := 0.0

	for word, freqA := range vecA {
		freqB := vecB[word]
		dotProduct += float64(freqA * freqB)
		magA += float64(freqA * freqA)
	}

	for _, freqB := range vecB {
		magB += float64(freqB * freqB)
	}

	return dotProduct / (math.Sqrt(magA) * math.Sqrt(magB))
}

// getWordFrequency creates a word frequency map for a given string
func getWordFrequency(text string) map[string]int {
	words := strings.Fields(text)
	frequency := make(map[string]int)
	for _, word := range words {
		frequency[word]++
	}
	return frequency
}

// diffAndEvaluate calculates the similarity and determines pass/fail
func diffAndEvaluate(newData, postBuildState string) (bool, float64, string) {

	if newData == "" || postBuildState == "" {
		return false, 0.0, "Incomplete data for comparison."
	}

	if newData == postBuildState {
		return true, 1.0, "Changes are consistent with the postBuildState."
	}

	similarity := cosineSimilarity(newData, postBuildState)
	passThreshold := 0.8 // Set our desired threshold here - i think 0.8 is fair.

	if similarity >= passThreshold {
		return true, similarity, "The newData meets the custom validation criteria."
	}
	return false, similarity, "The newData does not meet the custom validation criteria."
}

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

	var args Arguments
	err = json.Unmarshal([]byte(payload.Output[0].Function.Arguments), &args)
	if err != nil {
		http.Error(w, "Failed to parse arguments field", http.StatusInternalServerError)
		return
	}

	// Extract the `new` value from `arguments.changes[0].new`
	newChange := args.Changes[0].New

	// Get the `postBuildState` from context.vars
	postBuildState := payload.Context.Vars.PostBuildState

	// Perform the diff and evaluation
	pass, score, reason := diffAndEvaluate(newChange, postBuildState)

	// Create and send the response payload
	response := ResponsePayload{
		Pass:   pass,
		Score:  score,
		Reason: reason,
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
