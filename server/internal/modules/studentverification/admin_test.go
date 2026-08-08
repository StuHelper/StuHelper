package studentverification

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAdminEmailDomains(t *testing.T) {
	domains, ok := normalizeAdminEmailDomains([]string{
		" BUAA.EDU.CN ",
		"mail.example.edu.cn",
		"buaa.edu.cn",
	})
	require.True(t, ok)
	require.Equal(t, []string{"buaa.edu.cn", "mail.example.edu.cn"}, domains)

	for _, invalid := range [][]string{
		{"localhost"},
		{"127.0.0.1"},
		{"*.example.edu.cn"},
		{"example.edu.cn/path"},
	} {
		_, ok = normalizeAdminEmailDomains(invalid)
		require.False(t, ok, "domain list should fail closed: %#v", invalid)
	}
}

func TestValidateAdminSchoolProfile(t *testing.T) {
	policy := map[string]any{
		"rosterKnownEligibilityCodes": []any{"eligible", "ineligible"},
		"rosterEligibleCodes":         []any{"eligible"},
		"rosterMinimumRows":           float64(1),
		"rosterMaximumRowDeltaRatio":  float64(0.25),
		"rosterRequireCurrentMarker":  true,
	}
	raw, err := json.Marshal(policy)
	require.NoError(t, err)
	profile := &AdminVerificationSchoolConfig{
		SchoolCode:                  BUAASchoolCode,
		AdapterID:                   BUAAAdapterID,
		AdapterVersion:              "1",
		EmailDomains:                []string{"buaa.edu.cn"},
		StudentIDPolicy:             map[string]any{"strategy": "adapter"},
		NameMatchPolicy:             map[string]any{"strategy": "adapter"},
		EnrollmentPolicy:            policy,
		SnapshotSyncIntervalSeconds: 21_600,
		SnapshotWarningAfterSeconds: 43_200,
		SnapshotHardExpirySeconds:   172_800,
		SnapshotGraceSeconds:        0,
		enrollmentPolicyRaw:         raw,
	}

	valid, code := validateAdminSchoolProfile(profile)
	require.True(t, valid)
	require.Empty(t, code)

	profile.AdapterID = "declarative"
	valid, code = validateAdminSchoolProfile(profile)
	require.False(t, valid)
	require.Equal(t, "school_adapter_mismatch", code)
}

func TestAdminStudentIDPolicyIsSchoolSpecific(t *testing.T) {
	require.True(t, validAdminStudentIDPolicy(
		BUAASchoolCode,
		BUAAAdapterID,
		map[string]any{"strategy": "adapter"},
	))
	require.False(t, validAdminStudentIDPolicy(
		"4111019999",
		"declarative",
		map[string]any{"strategy": "adapter"},
	))
	require.True(t, validAdminStudentIDPolicy(
		"4111019999",
		"declarative",
		map[string]any{"strategy": "regex", "pattern": "^[A-Z][0-9]{7}$", "transform": "uppercase"},
	))
	require.False(t, validAdminStudentIDPolicy(
		"4111019999",
		"declarative",
		map[string]any{"strategy": "regex", "pattern": "[0-9]+"},
	))
}

func TestValidPrivacyNoticeRequiresACompleteDisclosure(t *testing.T) {
	version := "2026-08-05"
	require.True(t, validPrivacyNotice(&version, map[string]any{
		"title":            "学校账号校验说明",
		"summary":          "仅用于本次学生身份校验。",
		"retentionSummary": "学校密码不会持久化。",
		"dataCategories":   []string{"学号", "一次性学校密码"},
	}))
	require.False(t, validPrivacyNotice(&version, map[string]any{
		"title":   "不完整说明",
		"summary": "缺少保留期限与数据类型",
	}))
}

func TestMethodDraftSupportsExplicitCreateRevision(t *testing.T) {
	require.True(t, validAdminActorReason(42, "新增学校人工审核方法"))
	require.False(t, validAdminActorAndReason(42, 0, "更新既有方法"))
	require.True(t, validRosterDependency("conditional", map[string]any{"type": "adapter_assertion"}))
	require.False(t, validRosterDependency("conditional", map[string]any{}))
}
