package project

import (
	"fmt"
	"os"

	"github.com/kcrmin/Stackcord/cli/internal/operation"
)

// InitRequest is framework-neutral project metadata.
type InitRequest struct{ Root, ProjectID, Name, Locale, DraftRoot string }

// PlanInit builds an exact project harness without choosing implementation technologies.
func PlanInit(request InitRequest) (operation.Plan, error) {
	if request.Root == "" || request.ProjectID == "" || (request.Locale != "en" && request.Locale != "ko") {
		return operation.Plan{}, fmt.Errorf("root, stable project ID, and locale en|ko are required")
	}
	if !projectIDPattern.MatchString(request.ProjectID) {
		return operation.Plan{}, fmt.Errorf("project ID must be a lowercase dot-separated stable ID")
	}
	if entries, err := os.ReadDir(request.Root); err == nil && len(entries) > 0 {
		return operation.Plan{}, fmt.Errorf("target is not empty; use adopt")
	} else if err != nil && !os.IsNotExist(err) {
		return operation.Plan{}, err
	}
	files, err := render(request)
	if err != nil {
		return operation.Plan{}, err
	}
	plan := operation.Plan{ID: "init-" + request.ProjectID, Root: request.Root, Files: files}
	fingerprint, err := operation.StateFingerprint(plan)
	plan.InitialStateFingerprint = fingerprint
	return plan, err
}
