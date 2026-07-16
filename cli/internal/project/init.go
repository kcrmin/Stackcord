package project

import (
	"fmt"
	"os"

	"fullstack-orchestrator/cli/internal/operation"
)

// InitRequest is framework-neutral project metadata.
type InitRequest struct{ Root, ProjectID, Name, Locale, DraftRoot string }

// PlanInit builds an exact project harness without choosing implementation technologies.
func PlanInit(request InitRequest) (operation.Plan, error) {
	if request.Root == "" || request.ProjectID == "" || (request.Locale != "en" && request.Locale != "ko") {
		return operation.Plan{}, fmt.Errorf("root, stable project ID, and locale en|ko are required")
	}
	if entries, err := os.ReadDir(request.Root); err == nil && len(entries) > 0 {
		return operation.Plan{}, fmt.Errorf("target is not empty; use adopt")
	} else if err != nil && !os.IsNotExist(err) {
		return operation.Plan{}, err
	}
	plan := operation.Plan{ID: "init-" + request.ProjectID, Root: request.Root, Files: render(request)}
	fingerprint, err := operation.StateFingerprint(plan)
	plan.InitialStateFingerprint = fingerprint
	return plan, err
}
