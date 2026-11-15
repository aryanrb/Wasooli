package types

import "encoding/json"

// ProposedTerms is the contract spec under negotiation, stored on an invitation's
// `proposed` jsonb and applied to the contract when the link is finalized.
type ProposedTerms struct {
	TenantName string      `json:"tenant_name"`
	StartDate  string      `json:"start_date"` // YYYY-MM-DD
	Cycle      string      `json:"cycle"`
	Anchor     int32       `json:"anchor"`
	Components  []Component `json:"components"`
}

func (p ProposedTerms) Marshal() []byte {
	b, _ := json.Marshal(p)
	return b
}

func ParseProposed(b []byte) ProposedTerms {
	var p ProposedTerms
	if len(b) > 0 {
		_ = json.Unmarshal(b, &p)
	}
	return p
}

// ParseComponents decodes a stored jsonb breakdown back into components.
func ParseComponents(b []byte) []Component {
	var cs []Component
	if len(b) > 0 {
		_ = json.Unmarshal(b, &cs)
	}
	return cs
}
