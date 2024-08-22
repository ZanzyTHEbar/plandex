package types

import "github.com/plandex/plandex/shared"

type OverlapStrategy int

const (
	OverlapStrategySkip OverlapStrategy = iota
	OverlapStrategyError
)

type PlanResultParams struct {
	OrgId               string
	PlanId              string
	PlanBuildId         string
	ConvoMessageId      string
	FilePath            string
	PreBuildState       string
	OverlapStrategy     OverlapStrategy
	ChangesWithLineNums []*shared.StreamedChangeWithLineNums

	CheckSyntax bool

	IsFix       bool
	IsSyntaxFix bool
	IsOtherFix  bool
	FixEpoch    int
}
