package dto

type CreateRegistrationCodesReq struct {
	Count int `json:"count"` // default 1
}

// CreatedRegistrationCode is returned once, at creation: the access code holds
// the plain registration code, which the platform does not keep.
type CreatedRegistrationCode struct {
	ID         int64  `json:"id"`
	AccessCode string `json:"accessCode"`
	ExpiresAt  int64  `json:"expiresAt"`
}

// DeviceAccessResp is where devices reach the mTLS device port.
type DeviceAccessResp struct {
	Addr           string `json:"addr"` // host:port put into access codes; "" = not configured
	Port           string `json:"port"`
	OutdatedBundle bool   `json:"outdatedBundle"` // the compose file does not publish the port
}

type CreateRegistrationCodesResp struct {
	Items []CreatedRegistrationCode `json:"items"`
}

type RegistrationCode struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"` // unused | used | expired | revoked
	CreatedAt int64  `json:"createdAt"`
	ExpiresAt int64  `json:"expiresAt"`
	UsedAt    int64  `json:"usedAt,omitempty"`
	UsedMac   string `json:"usedMac,omitempty"`
	Devid     string `json:"devid,omitempty"`
	RevokedAt int64  `json:"revokedAt,omitempty"`
}

type ListRegistrationCodesResp struct {
	Items []RegistrationCode `json:"items"`
}
