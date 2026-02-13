package v1

type UpdateSubscriptionRequest struct {
	Channels        []string `json:"channels"`
	DigestFrequency string   `json:"digest_frequency"`
}
