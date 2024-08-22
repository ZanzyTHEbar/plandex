package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"

	"plandex-server/model/plan"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
)

func compareAppliedDiffs(preBuildState, postBuildState, filePath string, changes []*shared.StreamedChangeWithLineNums) (bool, float64, error) {
	sorted, err := shared.SortStreamedChanges(changes)

	if err != nil {
		return false, 0.0, fmt.Errorf("error sorting streamed changes: %v", err)
	}

	var overlapStrategy types.OverlapStrategy = types.OverlapStrategyError

	_, updatedFile, allSucceeded, err := plan.GetPlanResult(
		context.Background(),
		types.PlanResultParams{
			OrgId:               "orgId",
			PlanId:              "planId",
			PlanBuildId:         "planBuildId",
			ConvoMessageId:      "convoMessageId",
			FilePath:            filePath,
			PreBuildState:       preBuildState,
			ChangesWithLineNums: sorted,
			OverlapStrategy:     overlapStrategy,
			CheckSyntax:         false,
		},
	)

	if err != nil {
		return false, 0.0, fmt.Errorf("error getting plan result: %v", err)
	}

	if !allSucceeded {
		return false, 0.0, fmt.Errorf("not all changes succeeded")
	}

	updatedFile = shared.RemoveLineNums(updatedFile)

	score := cosineSimilarity(postBuildState, updatedFile)
	valid := score > 0.99 // leave a little wiggle room for spaces and other minor differences - but effectively we just want an equality check

	if !valid {
		log.Printf("compareAppliedDiffs - postBuildState does not match updatedFile\n")
		log.Printf("compareAppliedDiffs - score: %f\n", score)
		log.Printf("compareAppliedDiffs - postBuildState: %s\n", postBuildState)
		log.Printf("compareAppliedDiffs - updatedFile: %s\n", updatedFile)

		return false, score, nil
	}

	return true, score, nil
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

// NOTE: THIS IS NOT IN USE, only kept for posterity incase we need it. processDynamicJSON processes the dynamic JSON data and implements the provided logic
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
