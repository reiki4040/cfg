package cfg

import (
	"os"
	"strings"
)

const (
	DefaultStage     = "dev"
	StageEnvVar      = "CFG_STAGE"
	StagePlaceholder = "{stage}"
)

type StageResolver struct {
	stage string
}

func NewStageResolver(stage string) *StageResolver {
	return &StageResolver{
		stage: resolveStageValue(stage),
	}
}

func (s *StageResolver) ResolvePath(path string) string {
	return strings.ReplaceAll(path, StagePlaceholder, s.stage)
}

func (s *StageResolver) ResolveString(input string) string {
	return strings.ReplaceAll(input, StagePlaceholder, s.stage)
}

func (s *StageResolver) GetStage() string {
	return s.stage
}

func resolveStageValue(providedStage string) string {
	// Priority: CLI option > environment variable > default
	if providedStage != "" {
		return providedStage
	}

	if envStage := os.Getenv(StageEnvVar); envStage != "" {
		return envStage
	}

	return DefaultStage
}