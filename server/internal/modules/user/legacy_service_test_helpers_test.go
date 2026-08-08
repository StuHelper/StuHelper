package user

import "context"

// These helpers support unit tests for the sealed legacy implementation. The
// corresponding HTTP routes are intentionally not registered after the
// student-verification reset cutover.
func ptr[T any](value T) *T {
	return &value
}

type fakeIdentityPhotoStore struct {
	uploadErr     error
	presignErr    error
	presignURL    string
	uploadedKey   string
	uploadedType  string
	uploadedData  []byte
	presignedKey  string
	presignedKeys []string
}

type fakeVerificationStatusGateway struct {
	student CurrentStudentStatus
	phone   CurrentPhoneStatus
	err     error
}

func (f fakeVerificationStatusGateway) GetCurrentStudentStatus(context.Context, int64) (CurrentStudentStatus, error) {
	return f.student, f.err
}

func (f fakeVerificationStatusGateway) GetCurrentPhoneStatus(context.Context, int64) (CurrentPhoneStatus, error) {
	return f.phone, f.err
}

func (f *fakeIdentityPhotoStore) Upload(_ context.Context, key string, content []byte, contentType string) error {
	f.uploadedKey = key
	f.uploadedType = contentType
	f.uploadedData = append([]byte(nil), content...)
	return f.uploadErr
}

func (f *fakeIdentityPhotoStore) PresignGetURL(_ context.Context, key string) (string, error) {
	f.presignedKey = key
	f.presignedKeys = append(f.presignedKeys, key)
	if f.presignErr != nil {
		return "", f.presignErr
	}
	return f.presignURL, nil
}
