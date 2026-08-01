package casdoor

import (
	"errors"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

type fakeUserAPI struct {
	user       *casdoorsdk.User
	gotSubject string
	updated    *casdoorsdk.User
	columns    []string
	err        error
}

func (f *fakeUserAPI) GetUserByUserId(subject string) (*casdoorsdk.User, error) {
	f.gotSubject = subject
	return f.user, f.err
}

func (f *fakeUserAPI) UpdateUserForColumns(user *casdoorsdk.User, columns []string) (bool, error) {
	f.updated = user
	f.columns = append([]string(nil), columns...)
	return true, nil
}

var errFakeUserAPIUnavailable = errors.New("fake Casdoor user API unavailable")
