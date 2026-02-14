package v1

type RecordSignalRequest struct {
	SignalType string `json:"signal_type"`
	Annotation string `json:"annotation,omitempty"`
}

type CreateCheckpointRequest struct {
	LeafIDs []string `json:"leaf_ids"`
}

type RecordConsensusPositionRequest struct {
	Position    string `json:"position"`
	Explanation string `json:"explanation,omitempty"`
}
