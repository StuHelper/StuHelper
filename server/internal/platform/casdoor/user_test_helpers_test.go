package casdoor

import "github.com/casdoor/casdoor-go-sdk/casdoorsdk"

type fakeUserAPI struct {
	user       *casdoorsdk.User
	gotSubject string
	updated    *casdoorsdk.User
	columns    []string
}

func (f *fakeUserAPI) GetUserByUserId(subject string) (*casdoorsdk.User, error) {
	f.gotSubject = subject
	return f.user, nil
}

func (f *fakeUserAPI) UpdateUserForColumns(user *casdoorsdk.User, columns []string) (bool, error) {
	f.updated = user
	f.columns = append([]string(nil), columns...)
	return true, nil
}
