package casdoor

import "github.com/casdoor/casdoor-go-sdk/casdoorsdk"

const (
	casdoorViewAdmin       = "Admin"
	casdoorViewPublic      = "Public"
	casdoorViewSelf        = "Self"
	casdoorModifyAdmin     = "Admin"
	casdoorModifyImmutable = "Immutable"
	casdoorModifySelf      = "Self"
)

func newBootstrapOrganization(spec OrganizationSpec) *casdoorsdk.Organization {
	return &casdoorsdk.Organization{
		Owner:              "admin",
		Name:               spec.Name,
		DisplayName:        spec.DisplayName,
		DefaultApplication: spec.DefaultApplication,
		PasswordType:       "bcrypt",
		PasswordOptions:    []string{"AtLeast6"},
		CountryCodes:       []string{"CN"},
		UserTypes:          []string{},
		Tags:               []string{},
		Languages:          []string{"zh", "en"},
		NavItems:           []string{},
		UserNavItems:       []string{},
		WidgetItems:        []string{},
		MfaItems:           []*casdoorsdk.MfaItem{},
		AccountItems:       defaultAccountItems(),
		ThemeData:          &casdoorsdk.ThemeData{},
	}
}

func defaultAccountItems() []*casdoorsdk.AccountItem {
	return []*casdoorsdk.AccountItem{
		accountItem("Organization", casdoorViewPublic, casdoorModifyAdmin),
		accountItem("ID", casdoorViewPublic, casdoorModifyImmutable),
		accountItem("Name", casdoorViewPublic, casdoorModifyAdmin),
		accountItem("Display name", casdoorViewPublic, casdoorModifySelf),
		accountItem("Avatar", casdoorViewPublic, casdoorModifySelf),
		accountItem("Password", casdoorViewSelf, casdoorModifySelf),
		accountItem("Email", casdoorViewPublic, casdoorModifySelf),
		accountItem("Phone", casdoorViewPublic, casdoorModifySelf),
		accountItem("Roles", casdoorViewPublic, casdoorModifyImmutable),
		accountItem("Is admin", casdoorViewAdmin, casdoorModifyAdmin),
	}
}

func accountItem(name string, viewRule string, modifyRule string) *casdoorsdk.AccountItem {
	return &casdoorsdk.AccountItem{
		Name:       name,
		Visible:    true,
		ViewRule:   viewRule,
		ModifyRule: modifyRule,
	}
}
