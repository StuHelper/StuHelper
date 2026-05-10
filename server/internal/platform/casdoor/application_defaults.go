package casdoor

import "github.com/casdoor/casdoor-go-sdk/casdoorsdk"

const (
	defaultApplicationCertificate = "cert-built-in"
	casdoorRuleAll                = "All"
	casdoorRuleNone               = "None"
	casdoorRuleNormal             = "Normal"
	casdoorRuleRandom             = "Random"
)

func defaultProviderItems() []*casdoorsdk.ProviderItem {
	return []*casdoorsdk.ProviderItem{}
}

func defaultSigninMethods(interactive bool) []*casdoorsdk.SigninMethod {
	if !interactive {
		return []*casdoorsdk.SigninMethod{}
	}
	return []*casdoorsdk.SigninMethod{
		{Name: "Password", DisplayName: "Password", Rule: casdoorRuleAll},
		{Name: "Verification code", DisplayName: "Verification code", Rule: casdoorRuleNone},
		{Name: "WebAuthn", DisplayName: "WebAuthn", Rule: casdoorRuleNone},
		{Name: "Face ID", DisplayName: "Face ID", Rule: casdoorRuleNone},
	}
}

func defaultSignupItems(interactive bool) []*casdoorsdk.SignupItem {
	if !interactive {
		return []*casdoorsdk.SignupItem{}
	}
	return []*casdoorsdk.SignupItem{
		signupItem("ID", false, true, casdoorRuleRandom),
		signupItem("Username", true, true, casdoorRuleNone),
		signupItem("Display name", true, true, casdoorRuleNone),
		signupItem("Password", true, true, casdoorRuleNone),
		signupItem("Confirm password", true, true, casdoorRuleNone),
		signupItem("Email", true, false, casdoorRuleNormal),
		signupItem("Phone", true, false, casdoorRuleNone),
		signupItem("Agreement", true, true, casdoorRuleNone),
	}
}

func signupItem(name string, visible bool, required bool, rule string) *casdoorsdk.SignupItem {
	return &casdoorsdk.SignupItem{
		Name:     name,
		Visible:  visible,
		Required: required,
		Options:  []string{},
		Rule:     rule,
	}
}

func defaultSigninItems(interactive bool) []*casdoorsdk.SigninItem {
	if !interactive {
		return []*casdoorsdk.SigninItem{}
	}
	return []*casdoorsdk.SigninItem{
		signinItem("Back button", ".back-button { top: 65px; left: 15px; position: absolute; }"),
		signinItem("Languages", ".login-languages { top: 55px; right: 5px; position: absolute; }"),
		signinItem("Logo", ".login-logo-box {}"),
		signinItem("Signin methods", ".signin-methods {}"),
		signinItem("Username", ".login-username {} .login-username-input {}"),
		signinItem("Password", ".login-password {} .login-password-input {}"),
		signinItem("Login button", ".login-button-box { margin-bottom: 5px; } .login-button { width: 100%; }"),
		signinItem("Signup link", ".login-signup-link { margin-bottom: 24px; display: flex; justify-content: end; }"),
		signinItem("Providers", ".provider-img { width: 30px; margin: 5px; }"),
	}
}

func signinItem(name string, customCSS string) *casdoorsdk.SigninItem {
	return &casdoorsdk.SigninItem{
		Name:      name,
		Visible:   true,
		CustomCss: customCSS,
		Rule:      casdoorRuleNone,
	}
}
