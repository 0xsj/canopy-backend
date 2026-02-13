package v1

type CreateDraftRequest struct {
	Format        string   `json:"format"`
	Content       string   `json:"content"`
	SourceLeafIDs []string `json:"source_leaf_ids"`
}

type UpdateDeliverableRequest struct {
	Content string `json:"content"`
}
