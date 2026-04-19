package storage

type CapabilitySet struct {
	Put               bool `json:"put"`
	Delete            bool `json:"delete"`
	Stat              bool `json:"stat"`
	PresignedDownload bool `json:"presignedDownload"`
}

type Mount struct {
	ID                  int64         `json:"id"`
	Key                 string        `json:"key"`
	Name                string        `json:"name"`
	Driver              string        `json:"driver"`
	Bucket              *string       `json:"bucket,omitempty"`
	BasePath            string        `json:"basePath"`
	CredentialSource    string        `json:"credentialSource"`
	Enabled             bool          `json:"enabled"`
	LastHealthStatus    *string       `json:"lastHealthStatus,omitempty"`
	LastHealthError     *string       `json:"lastHealthError,omitempty"`
	LastHealthCheckedAt *string       `json:"lastHealthCheckedAt,omitempty"`
	Capabilities        CapabilitySet `json:"capabilities"`
}

type CreateMountRequest struct {
	Key      string  `json:"key"`
	Name     string  `json:"name"`
	Driver   string  `json:"driver"`
	Bucket   *string `json:"bucket,omitempty"`
	BasePath string  `json:"basePath"`
	Enabled  bool    `json:"enabled"`
}

type StoredObject struct {
	ObjectKey   string `json:"objectKey"`
	SizeBytes   int64  `json:"sizeBytes"`
	ContentType string `json:"contentType"`
}
