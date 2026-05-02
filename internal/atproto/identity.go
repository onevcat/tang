package atproto

type Identity struct {
	DID    string `json:"did"`
	Handle string `json:"handle"`
	PDS    string `json:"pds"`
}
