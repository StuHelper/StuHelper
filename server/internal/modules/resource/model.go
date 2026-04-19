package resource

type Binding struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Version struct {
	ID          int64  `json:"id"`
	VersionNo   int    `json:"versionNo"`
	MountID     int64  `json:"mountID"`
	ObjectKey   string `json:"objectKey"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	CreatedAt   string `json:"createdAt"`
}

type Item struct {
	ID            int64     `json:"id"`
	OwnerUserID   string    `json:"ownerUserID,omitempty"`
	Title         string    `json:"title"`
	Description   *string   `json:"description,omitempty"`
	Category      *string   `json:"category,omitempty"`
	Visibility    string    `json:"visibility"`
	Tags          []string  `json:"tags"`
	Bindings      []Binding `json:"bindings"`
	LatestVersion Version   `json:"latestVersion"`
	CreatedAt     string    `json:"createdAt"`
	UpdatedAt     string    `json:"updatedAt"`
}

type CreateRequest struct {
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	Category    *string   `json:"category,omitempty"`
	Visibility  string    `json:"visibility"`
	Tags        []string  `json:"tags"`
	Bindings    []Binding `json:"bindings"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	DataBase64  string    `json:"dataBase64"`
	MountKey    string    `json:"mountKey,omitempty"`
}

type UpdateRequest struct {
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	Category    *string   `json:"category,omitempty"`
	Visibility  string    `json:"visibility"`
	Tags        []string  `json:"tags"`
	Bindings    []Binding `json:"bindings"`
}

type ListFilters struct {
	Query        string
	Tag          string
	BindingType  string
	BindingValue string
	Page         int
	PageSize     int
}
