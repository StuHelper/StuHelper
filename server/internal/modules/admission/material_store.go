package admission

import "context"

type AdmissionMaterialStore interface {
	PutAdmissionMaterial(ctx context.Context, objectKey string, content []byte, contentType string) error
	DeleteAdmissionMaterial(ctx context.Context, objectKey string) error
	GetAdmissionMaterialURL(ctx context.Context, objectKey string) (string, error)
}
