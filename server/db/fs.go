package db

import (
	"fmt"
	"os"
	"path/filepath"
)

var BaseDir string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("Error getting user home dir: %v", err))
	}

	BaseDir = filepath.Join(home, "plandex-server")
}

func InitPlan(orgId, planId string) error {
	dir := getPlanDir(orgId, planId)
	err := os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		return fmt.Errorf("error creating plan dir: %v", err)
	}

	for _, subdirFn := range [](func(orgId, planId string) string){
		getPlanContextDir,
		getPlanConversationDir,
		getPlanResultsDir} {
		err = os.MkdirAll(subdirFn(orgId, planId), os.ModePerm)

		if err != nil {
			return fmt.Errorf("error creating plan subdir: %v", err)
		}
	}

	err = InitGitRepo(orgId, planId)

	if err != nil {
		return fmt.Errorf("error initializing git repo: %v", err)
	}

	return nil
}

func DeletePlanDir(orgId, planId string) error {
	dir := getPlanDir(orgId, planId)
	err := os.RemoveAll(dir)

	if err != nil {
		return fmt.Errorf("error deleting plan dir: %v", err)
	}

	return nil
}

func getPlanDir(orgId, planId string) string {
	return filepath.Join(BaseDir, "orgs", orgId, "plans", planId)
}

func getPlanContextDir(orgId, planId string) string {
	return filepath.Join(getPlanDir(orgId, planId), "context")
}

func getPlanConversationDir(orgId, planId string) string {
	return filepath.Join(getPlanDir(orgId, planId), "conversation")
}

func getPlanResultsDir(orgId, planId string) string {
	return filepath.Join(getPlanDir(orgId, planId), "results")
}