package v1

type StartSynthesisRequest struct {
	SourceLeafIDs []string `json:"source_leaf_ids"`
}

type CompleteSynthesisRequest struct {
	ResultLeafID string `json:"result_leaf_id"`
}

type FailSynthesisRequest struct {
	Reason string `json:"reason"`
}
