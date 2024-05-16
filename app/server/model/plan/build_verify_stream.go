package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"plandex-server/model"
	"plandex-server/types"
	"strings"
	"time"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (fileState *activeBuildStreamFileState) listenStreamVerifyOutput(stream *openai.ChatCompletionStream) {

	filePath := fileState.filePath
	planId := fileState.plan.Id
	branch := fileState.branch

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("listenStreamVerifyOutput - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	defer stream.Close()

	// Create a timer that will trigger if no chunk is received within the specified duration
	timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
	defer timer.Stop()

	for {
		select {
		case <-activePlan.Ctx.Done():
			// The main context was canceled (not the timer)
			return
		case <-timer.C:
			// Timer triggered because no new chunk was received in time
			fileState.verifyRetryOrAbort(fmt.Errorf("listenStreamVerifyOutput - stream timeout due to inactivity for file '%s'", filePath))
			return
		default:
			response, err := stream.Recv()

			if err == nil {
				// Successfully received a chunk, reset the timer
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(model.OPENAI_STREAM_CHUNK_TIMEOUT)
			} else {
				log.Printf("listenStreamVerifyOutput - File %s: Error receiving stream chunk: %v\n", filePath, err)

				if err == context.Canceled {
					log.Printf("listenStreamVerifyOutput - File %s: Stream canceled\n", filePath)
					return
				}

				fileState.verifyRetryOrAbort(fmt.Errorf("listenStreamVerifyOutput - stream error: %v", err))
				return
			}

			if len(response.Choices) == 0 {
				fileState.verifyRetryOrAbort(fmt.Errorf("listenStreamVerifyOutput - stream error: no choices"))
				return
			}

			choice := response.Choices[0]
			var content string
			delta := response.Choices[0].Delta

			if len(delta.ToolCalls) > 0 {
				content = delta.ToolCalls[0].Function.Arguments

				trimmed := strings.TrimSpace(content)
				if trimmed == "{%invalidjson%}" || trimmed == "``(no output)``````" {
					log.Println("File", filePath+":", "%invalidjson%} token in streamed chunk")
					fileState.verifyRetryOrAbort(fmt.Errorf("invalid JSON in streamed chunk for file '%s'", filePath))
					return
				}

				buildInfo := &shared.BuildInfo{
					Path:      filePath,
					NumTokens: 1,
					Finished:  false,
				}

				// log.Printf("%s: %s", filePath, content)
				activePlan.Stream(shared.StreamMessage{
					Type:      shared.StreamMessageBuildInfo,
					BuildInfo: buildInfo,
				})

				fileState.activeBuild.VerifyBuffer += content
				fileState.activeBuild.VerifyBufferTokens++
			}

			var streamed types.StreamedVerifyResult
			err = json.Unmarshal([]byte(fileState.activeBuild.VerifyBuffer), &streamed)

			if err == nil {
				log.Printf("listenStreamVerifyOutput - File %s: Parsed streamed verify result\n", filePath)
				// spew.Dump(streamed)

				if streamed.IsCorrect() {
					buildInfo := &shared.BuildInfo{
						Path:      filePath,
						NumTokens: 0,
						Finished:  true,
					}
					activePlan.Stream(shared.StreamMessage{
						Type:      shared.StreamMessageBuildInfo,
						BuildInfo: buildInfo,
					})
					log.Println("build verify - streamed.IsCorrect")

					time.Sleep(50 * time.Millisecond)

					fileState.onFinishBuildFile(nil, "")
				} else {

					log.Printf("listenStreamVerifyOutput - File %s: Streamed verify result is incorrect\n", filePath)

					fileState.verificationErrors = streamed.GetReasoning()
					fileState.isFixingOther = true

					// log.Println("Verification errors:")
					// log.Println(fileState.verificationErrors)

					select {
					case <-activePlan.Ctx.Done():
						log.Println("listenStreamVerifyOutput - Context canceled. Exiting.")
						return
					case <-time.After(time.Duration(rand.Intn(1001)) * time.Millisecond):
						break
					}
					fileState.fixFileLineNums()
				}
				return
			} else if len(delta.ToolCalls) == 0 {
				log.Println("listenStreamVerifyOutput - Stream chunk missing function call.")
				// log.Println(spew.Sdump(response))

				fileState.verifyRetryOrAbort(fmt.Errorf("listenStreamVerifyOutput - stream chunk missing function call. Reason: %s, File: %s", choice.FinishReason, filePath))
			}
		}
	}

}

func (fileState *activeBuildStreamFileState) verifyRetryOrAbort(err error) {
	if fileState.verifyFileNumRetry < MaxBuildStreamErrorRetries {
		fileState.verifyFileNumRetry++
		fileState.activeBuild.VerifyBuffer = ""
		fileState.activeBuild.VerifyBufferTokens = 0
		log.Printf("Retrying verify file '%s' due to error: %v\n", fileState.filePath, err)

		// Exponential backoff
		time.Sleep(time.Duration((fileState.verifyFileNumRetry*fileState.verifyFileNumRetry)/2)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond)

		fileState.verifyFileBuild()
	} else {
		log.Printf("Aborting verify file '%s' due to error: %v\n", fileState.filePath, err)

		fileState.onFinishBuildFile(nil, "")
	}
}
