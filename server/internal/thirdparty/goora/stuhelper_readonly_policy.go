package go_ora

import (
	"errors"
	"strings"
)

var errStuHelperSelectOnly = errors.New("StuHelper Oracle driver permits fixed SELECT queries only")

func validateStuHelperSelectOnly(query string) error {
	normalized := strings.ToUpper(strings.Join(strings.Fields(query), " "))
	if !strings.HasPrefix(normalized, "SELECT ") {
		return errStuHelperSelectOnly
	}
	for _, forbidden := range []string{
		";", "--", "/*", "*/", " FOR UPDATE", ".NEXTVAL", ".CURRVAL",
		" DBMS_", " UTL_", " EXECUTE ", " CALL ",
	} {
		if strings.Contains(normalized, forbidden) {
			return errStuHelperSelectOnly
		}
	}
	return nil
}
