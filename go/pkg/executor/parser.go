package executor

// NOTE: Parsing and CoT extraction will be implemented in Phase 3.
// This file intentionally contains placeholders to anchor future work without impacting build stability.

// parseFullDecisionResponse will parse a raw assistant response into a FullDecision structure.
func parseFullDecisionResponse(raw string) (*FullDecision, error) {
	// TODO: implement robust parsing (json repair, CoT extraction)
	return &FullDecision{CoTTrace: "", Decisions: nil}, nil
}
