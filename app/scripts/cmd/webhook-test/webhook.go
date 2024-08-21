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

// Context represents the context within the request
type Context struct {
	Prompt string            `json:"prompt"`
	Vars   map[string]string `json:"vars"`
}

// RequestPayload represents the incoming POST request payload
type RequestPayload struct {
	Output  string  `json:"output"`
	Context Context `json:"context"`
}

// ResponsePayload represents the response that the webhook will send back
type ResponsePayload struct {
	Pass   bool    `json:"pass"`
	Score  float64 `json:"score,omitempty"`  // Score is optional
	Reason string  `json:"reason,omitempty"` // Reason is optional
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
func diffAndEvaluate(output, postBuildState string) (bool, float64, string) {
	similarity := cosineSimilarity(output, postBuildState)
	passThreshold := 0.8 // Set our desired threshold here - i think 0.8 is fair.

	if similarity >= passThreshold {
		return true, similarity, "The output meets the custom validation criteria."
	}
	return false, similarity, "The output does not meet the custom validation criteria."
}

// webhookHandler handles the incoming POST request
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}

	var payload RequestPayload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Here, we'll use the context variables if needed; otherwise, they are just part of the payload structure
	postBuildState := payload.Context.Vars["postBuildState"]

	pass, score, reason := diffAndEvaluate(payload.Output, postBuildState)

	response := ResponsePayload{
		Pass:   pass,
		Score:  score,
		Reason: reason,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)
	fmt.Println("Starting server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
