package project

import "regexp"

var projectIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9-]*)+$`)
var draftIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)
var workIDPattern = regexp.MustCompile(`^work\.[A-Za-z0-9-]+$`)
var claimIDPattern = regexp.MustCompile(`^claim\.[A-Za-z0-9-]+$`)

// ValidWorkID reports whether a work identity is safe for both schemas and paths.
func ValidWorkID(value string) bool { return workIDPattern.MatchString(value) }

func validClaimID(value string) bool { return claimIDPattern.MatchString(value) }
