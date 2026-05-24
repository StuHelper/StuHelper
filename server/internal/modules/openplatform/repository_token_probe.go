package openplatform

import (
	"context"
	"encoding/json"
	"fmt"
)

func (r *Repository) RecordTokenProbeEvidence(ctx context.Context, evidence TokenProbeEvidence) error {
	ctx = withDBTable(ctx, "open_platform_token_probe_evidence")
	if evidence.InspectedClaims == nil {
		evidence.InspectedClaims = []string{}
	}
	if evidence.BusinessClaims == nil {
		evidence.BusinessClaims = []string{}
	}
	if evidence.TokenClaims == nil {
		evidence.TokenClaims = map[string][]string{}
	}
	if evidence.Metadata == nil {
		evidence.Metadata = map[string]any{}
	}
	inspectedClaims, err := json.Marshal(evidence.InspectedClaims)
	if err != nil {
		return fmt.Errorf("RecordTokenProbeEvidence inspected claims: %w", err)
	}
	businessClaims, err := json.Marshal(evidence.BusinessClaims)
	if err != nil {
		return fmt.Errorf("RecordTokenProbeEvidence business claims: %w", err)
	}
	tokenClaims, err := json.Marshal(evidence.TokenClaims)
	if err != nil {
		return fmt.Errorf("RecordTokenProbeEvidence token claims: %w", err)
	}
	metadata, err := json.Marshal(evidence.Metadata)
	if err != nil {
		return fmt.Errorf("RecordTokenProbeEvidence metadata: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO open_platform_token_probe_evidence (
			app_id,
			reviewer_user_id,
			request_id,
			casdoor_application_name,
			client_id,
			redirect_uri,
			probe_method,
			result,
			inspected_claims,
			business_claims,
			token_claims,
			metadata,
			error
		)
		VALUES ($1, NULLIF($2, 0), NULLIF($3, ''), $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, evidence.AppID,
		evidence.ReviewerUserID,
		evidence.RequestID,
		evidence.CasdoorApplicationName,
		evidence.ClientID,
		evidence.RedirectURI,
		evidence.ProbeMethod,
		evidence.Result,
		inspectedClaims,
		businessClaims,
		tokenClaims,
		metadata,
		evidence.Error,
	)
	if err != nil {
		return fmt.Errorf("RecordTokenProbeEvidence insert: %w", err)
	}
	return nil
}

type tokenProbeEvidenceListFilter struct {
	AppID          int64
	ReviewerUserID int64
	Result         string
	ClientID       string
	Limit          int
	Offset         int
}

func (r *Repository) ListTokenProbeEvidence(
	ctx context.Context,
	filter tokenProbeEvidenceListFilter,
) (TokenProbeEvidenceListResult, error) {
	ctx = withDBTable(ctx, "open_platform_token_probe_evidence")
	whereSQL := `
		WHERE ($1::bigint = 0 OR app_id = $1)
		  AND ($2::bigint = 0 OR reviewer_user_id = $2)
		  AND ($3::text = '' OR result = $3)
		  AND ($4::text = '' OR client_id = $4)
	`

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM open_platform_token_probe_evidence
		`+whereSQL+`
	`, filter.AppID, filter.ReviewerUserID, filter.Result, filter.ClientID).Scan(&total); err != nil {
		return TokenProbeEvidenceListResult{}, fmt.Errorf("ListTokenProbeEvidence count: %w", err)
	}
	if total == 0 {
		return TokenProbeEvidenceListResult{List: []TokenProbeEvidenceRecord{}, Total: 0}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			app_id,
			reviewer_user_id,
			request_id,
			casdoor_application_name,
			client_id,
			redirect_uri,
			probe_method,
			result,
			inspected_claims,
			business_claims,
			token_claims,
			metadata,
			error,
			created_at
		FROM open_platform_token_probe_evidence
		`+whereSQL+`
		ORDER BY created_at DESC, id DESC
		LIMIT $5 OFFSET $6
	`, filter.AppID, filter.ReviewerUserID, filter.Result, filter.ClientID, filter.Limit, filter.Offset)
	if err != nil {
		return TokenProbeEvidenceListResult{}, fmt.Errorf("ListTokenProbeEvidence query: %w", err)
	}
	defer rows.Close()

	list := make([]TokenProbeEvidenceRecord, 0, filter.Limit)
	for rows.Next() {
		record, err := scanTokenProbeEvidence(rows)
		if err != nil {
			return TokenProbeEvidenceListResult{}, fmt.Errorf("ListTokenProbeEvidence scan: %w", err)
		}
		list = append(list, record)
	}
	if err := rows.Err(); err != nil {
		return TokenProbeEvidenceListResult{}, fmt.Errorf("ListTokenProbeEvidence rows: %w", err)
	}
	return TokenProbeEvidenceListResult{List: list, Total: total}, nil
}

func scanTokenProbeEvidence(row rowScanner) (TokenProbeEvidenceRecord, error) {
	var record TokenProbeEvidenceRecord
	var inspectedClaims []byte
	var businessClaims []byte
	var tokenClaims []byte
	var metadata []byte
	if err := row.Scan(
		&record.ID,
		&record.AppID,
		&record.ReviewerUserID,
		&record.RequestID,
		&record.CasdoorApplicationName,
		&record.ClientID,
		&record.RedirectURI,
		&record.ProbeMethod,
		&record.Result,
		&inspectedClaims,
		&businessClaims,
		&tokenClaims,
		&metadata,
		&record.Error,
		&record.CreatedAt,
	); err != nil {
		return TokenProbeEvidenceRecord{}, err
	}
	if err := json.Unmarshal(inspectedClaims, &record.InspectedClaims); err != nil {
		return TokenProbeEvidenceRecord{}, fmt.Errorf("unmarshal inspected claims: %w", err)
	}
	if err := json.Unmarshal(businessClaims, &record.BusinessClaims); err != nil {
		return TokenProbeEvidenceRecord{}, fmt.Errorf("unmarshal business claims: %w", err)
	}
	if err := json.Unmarshal(tokenClaims, &record.TokenClaims); err != nil {
		return TokenProbeEvidenceRecord{}, fmt.Errorf("unmarshal token claims: %w", err)
	}
	if err := json.Unmarshal(metadata, &record.Metadata); err != nil {
		return TokenProbeEvidenceRecord{}, fmt.Errorf("unmarshal metadata: %w", err)
	}
	if record.InspectedClaims == nil {
		record.InspectedClaims = []string{}
	}
	if record.BusinessClaims == nil {
		record.BusinessClaims = []string{}
	}
	if record.TokenClaims == nil {
		record.TokenClaims = map[string][]string{}
	}
	if record.Metadata == nil {
		record.Metadata = map[string]any{}
	}
	return record, nil
}
