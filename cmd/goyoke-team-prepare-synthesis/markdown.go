package main

import (
	"fmt"
	"strings"
)

// generateMarkdown creates the pre-synthesis.md content from extracted sections.
//
// Integration contract (FIX #16): This markdown is a SUPPLEMENTAL context document placed in the
// team directory. goyoke-team-run reads this file and incorporates its content into
// Beethoven's JSON stdin (via the "pre_synthesis" field). Beethoven does NOT read this
// file directly — goyoke-team-run is the intermediary.
func generateMarkdown(einstein EinsteinSections, staffArch StaffArchSections) string {
	var sb strings.Builder

	sb.WriteString("# Pre-Synthesis Input for Beethoven\n\n")
	sb.WriteString("This document contains extracted insights from Einstein (theoretical analysis) and Staff-Architect (critical review) for Beethoven to synthesize.\n\n")

	// FIX #11: Status warning banner
	if einstein.Status != "complete" || staffArch.Status != "complete" {
		sb.WriteString("**WARNING: One or more analyses reported non-complete status. Review findings carefully.**\n\n")
		if einstein.Status != "" && einstein.Status != "complete" {
			sb.WriteString(fmt.Sprintf("- Einstein status: %s\n", einstein.Status))
		}
		if staffArch.Status != "" && staffArch.Status != "complete" {
			sb.WriteString(fmt.Sprintf("- Staff-Architect status: %s\n", staffArch.Status))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")

	// Einstein sections
	sb.WriteString("## Einstein: Theoretical Analysis\n\n")

	sb.WriteString("### Executive Summary\n\n")
	sb.WriteString(einstein.ExecutiveSummary)
	sb.WriteString("\n\n")

	sb.WriteString("### Root Cause Analysis\n\n")
	if len(einstein.RootCauses) > 0 {
		for _, cause := range einstein.RootCauses {
			sb.WriteString(fmt.Sprintf("- %s\n\n", cause))
		}
	} else {
		sb.WriteString("(No root causes identified)\n\n")
	}

	sb.WriteString("### Conceptual Frameworks\n\n")
	if len(einstein.Frameworks) > 0 {
		for _, framework := range einstein.Frameworks {
			sb.WriteString(framework)
			sb.WriteString("\n\n")
		}
	} else {
		sb.WriteString("(No frameworks provided)\n\n")
	}

	// FIX #2: First Principles Analysis section
	sb.WriteString("### First Principles Analysis\n\n")
	if len(einstein.FirstPrinciples) > 0 {
		for _, fp := range einstein.FirstPrinciples {
			sb.WriteString(fmt.Sprintf("- %s\n\n", fp))
		}
	} else {
		sb.WriteString("(None identified)\n\n")
	}

	sb.WriteString("### Novel Approaches\n\n")
	if len(einstein.NovelApproaches) > 0 {
		for i, approach := range einstein.NovelApproaches {
			sb.WriteString(fmt.Sprintf("%d. %s\n\n", i+1, approach))
		}
	} else {
		sb.WriteString("(No novel approaches provided)\n\n")
	}

	// FIX #2: Theoretical Tradeoffs section
	sb.WriteString("### Theoretical Tradeoffs\n\n")
	if len(einstein.TheoreticalTradeoffs) > 0 {
		for _, tradeoff := range einstein.TheoreticalTradeoffs {
			sb.WriteString(fmt.Sprintf("%s\n\n", tradeoff))
		}
	} else {
		sb.WriteString("(None identified)\n\n")
	}

	// FIX #2: Assumptions Surfaced section
	sb.WriteString("### Assumptions Surfaced\n\n")
	if len(einstein.AssumptionsSurfaced) > 0 {
		for _, assumption := range einstein.AssumptionsSurfaced {
			sb.WriteString(fmt.Sprintf("- %s\n\n", assumption))
		}
	} else {
		sb.WriteString("(None identified)\n\n")
	}

	sb.WriteString("### Open Questions\n\n")
	if len(einstein.OpenQuestions) > 0 {
		for _, question := range einstein.OpenQuestions {
			sb.WriteString(fmt.Sprintf("- %s\n\n", question))
		}
	} else {
		sb.WriteString("(No open questions)\n\n")
	}

	sb.WriteString("### Handoff Notes\n\n")
	sb.WriteString(einstein.HandoffNotes)
	sb.WriteString("\n\n")

	sb.WriteString("---\n\n")

	// Staff-Architect sections
	sb.WriteString("## Staff-Architect: Critical Review\n\n")

	sb.WriteString("### Executive Assessment\n\n")
	sb.WriteString(staffArch.ExecutiveVerdict)
	sb.WriteString("\n\n")

	sb.WriteString("### Critical Issues\n\n")
	if len(staffArch.CriticalIssues) > 0 {
		for _, issue := range staffArch.CriticalIssues {
			sb.WriteString(fmt.Sprintf("%s\n\n", issue))
		}
	} else {
		sb.WriteString("(No critical issues)\n\n")
	}

	sb.WriteString("### Major Issues\n\n")
	if len(staffArch.MajorIssues) > 0 {
		for _, issue := range staffArch.MajorIssues {
			sb.WriteString(fmt.Sprintf("%s\n\n", issue))
		}
	} else {
		sb.WriteString("(No major issues)\n\n")
	}

	sb.WriteString("### Minor Issues\n\n")
	if len(staffArch.MinorIssues) > 0 {
		for _, issue := range staffArch.MinorIssues {
			sb.WriteString(fmt.Sprintf("%s\n\n", issue))
		}
	} else {
		sb.WriteString("(No minor issues)\n\n")
	}

	// FIX #5: Commendations section
	sb.WriteString("### Commendations\n\n")
	if len(staffArch.Commendations) > 0 {
		for _, commendation := range staffArch.Commendations {
			sb.WriteString(fmt.Sprintf("- %s\n", commendation))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("(None identified)\n\n")
	}

	// FIX #5: Failure Mode Analysis section
	sb.WriteString("### Failure Mode Analysis\n\n")
	if len(staffArch.FailureModes) > 0 {
		for _, fm := range staffArch.FailureModes {
			sb.WriteString(fmt.Sprintf("%s\n\n", fm))
		}
	} else {
		sb.WriteString("(None identified)\n\n")
	}

	sb.WriteString("### Recommendations\n\n")
	if len(staffArch.Recommendations) > 0 {
		for _, rec := range staffArch.Recommendations {
			sb.WriteString(fmt.Sprintf("%s\n\n", rec))
		}
	} else {
		sb.WriteString("(No recommendations)\n\n")
	}

	sb.WriteString("### Sign-Off Conditions\n\n")
	if len(staffArch.SignOffConditions) > 0 {
		for _, condition := range staffArch.SignOffConditions {
			sb.WriteString(fmt.Sprintf("- %s\n", condition))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("(No conditions)\n\n")
	}

	sb.WriteString("### Handoff Notes\n\n")
	sb.WriteString(staffArch.HandoffNotes)
	sb.WriteString("\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("**End of Pre-Synthesis Document**\n")

	return sb.String()
}
