package cfg

import (
	"os"
	"strings"
)

const (
	DefaultStage           = "dev"
	StageEnvVar            = "CFG_STAGE"
	StagePlaceholder       = "{stage}"
	StagePrefixPlaceholder = "{stage-prefix}"
)

type StageResolver struct {
	stage             string
	stagePrefixBlanks map[string]bool
}

func NewStageResolver(stage string) *StageResolver {
	return NewStageResolverWithPrefixBlanks(stage, nil)
}

func NewStageResolverWithPrefixBlanks(stage string, stagePrefixBlanks []string) *StageResolver {
	blankMap := make(map[string]bool)
	for _, s := range stagePrefixBlanks {
		blankMap[s] = true
	}

	return &StageResolver{
		stage:             resolveStageValue(stage),
		stagePrefixBlanks: blankMap,
	}
}

func (s *StageResolver) ResolvePath(path string) string {
	result := strings.ReplaceAll(path, StagePlaceholder, s.stage)
	result = s.resolveStagePrefixPlaceholder(result)
	return result
}

func (s *StageResolver) ResolveString(input string) string {
	result := strings.ReplaceAll(input, StagePlaceholder, s.stage)
	result = s.resolveStagePrefixPlaceholder(result)
	return result
}

func (s *StageResolver) resolveStagePrefixPlaceholder(input string) string {
	if !strings.Contains(input, StagePrefixPlaceholder) {
		return input
	}

	var stagePrefix string
	if s.stagePrefixBlanks[s.stage] {
		// This stage should be blank (no prefix)
		stagePrefix = ""
	} else {
		// Add dash suffix to stage
		stagePrefix = s.stage + "-"
	}

	return strings.ReplaceAll(input, StagePrefixPlaceholder, stagePrefix)
}

func (s *StageResolver) GetStage() string {
	return s.stage
}

func resolveStageValue(providedStage string) string {
	// Priority: environment variable > CLI option > default
	if envStage := os.Getenv(StageEnvVar); envStage != "" {
		return envStage
	}

	if providedStage != "" {
		return providedStage
	}

	return DefaultStage
}
