package academics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) ReplaceSnapshot(ctx context.Context, jobID, sourceID int64, snapshot Snapshot) error {
	return r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		termIDs, err := upsertTermsTx(ctx, tx, jobID, sourceID, snapshot.Terms)
		if err != nil {
			return err
		}
		courseIDs, err := upsertCoursesTx(ctx, tx, jobID, sourceID, snapshot.Courses)
		if err != nil {
			return err
		}
		teacherIDs, err := upsertTeachersTx(ctx, tx, jobID, sourceID, snapshot.Teachers)
		if err != nil {
			return err
		}
		offeringIDs, err := upsertOfferingsTx(ctx, tx, jobID, sourceID, snapshot.Offerings, termIDs, courseIDs)
		if err != nil {
			return err
		}
		if err := replaceOfferingRelationsTx(ctx, tx, sourceID, snapshot.Offerings, teacherIDs, offeringIDs); err != nil {
			return err
		}
		if err := replaceMembershipsTx(ctx, tx, jobID, sourceID, snapshot.Memberships, offeringIDs); err != nil {
			return err
		}
		return pruneStaleSnapshotRowsTx(ctx, tx, sourceID, snapshot)
	})
}

func pruneStaleSnapshotRowsTx(ctx context.Context, tx pgx.Tx, sourceID int64, snapshot Snapshot) error {
	if err := deleteStaleRowsByExternalIDTx(ctx, tx, "academic_offerings", sourceID, offeringExternalIDs(snapshot.Offerings)); err != nil {
		return fmt.Errorf("delete stale academic offerings: %w", err)
	}
	if err := deleteStaleRowsByExternalIDTx(ctx, tx, "academic_terms", sourceID, termExternalIDs(snapshot.Terms)); err != nil {
		return fmt.Errorf("delete stale academic terms: %w", err)
	}
	if err := deleteStaleRowsByExternalIDTx(ctx, tx, "academic_courses", sourceID, courseExternalIDs(snapshot.Courses)); err != nil {
		return fmt.Errorf("delete stale academic courses: %w", err)
	}
	if err := deleteStaleRowsByExternalIDTx(ctx, tx, "academic_teachers", sourceID, teacherExternalIDs(snapshot.Teachers)); err != nil {
		return fmt.Errorf("delete stale academic teachers: %w", err)
	}
	return nil
}

func deleteStaleRowsByExternalIDTx(ctx context.Context, tx pgx.Tx, table string, sourceID int64, currentExternalIDs []string) error {
	if len(currentExternalIDs) == 0 {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE source_id = $1`, table), sourceID); err != nil {
			return err
		}
		return nil
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE source_id = $1
		  AND NOT (external_id = ANY($2::text[]))
	`, table), sourceID, currentExternalIDs); err != nil {
		return err
	}
	return nil
}

func termExternalIDs(items []ImportTerm) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item.ExternalID == "" {
			continue
		}
		ids = append(ids, item.ExternalID)
	}
	return ids
}

func courseExternalIDs(items []ImportCourse) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item.ExternalID == "" {
			continue
		}
		ids = append(ids, item.ExternalID)
	}
	return ids
}

func teacherExternalIDs(items []ImportTeacher) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item.ExternalID == "" {
			continue
		}
		ids = append(ids, item.ExternalID)
	}
	return ids
}

func offeringExternalIDs(items []ImportOffering) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item.ExternalID == "" {
			continue
		}
		ids = append(ids, item.ExternalID)
	}
	return ids
}

func marshalAcademicImportRows(label string, items any) ([]byte, error) {
	payload, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal academic %s import rows: %w", label, err)
	}
	if string(payload) == "null" {
		return []byte("[]"), nil
	}
	return payload, nil
}

func scanAcademicExternalIDMapping(
	rows pgx.Rows,
	label string,
	expected int,
) (map[string]int64, error) {
	defer rows.Close()
	mapping := make(map[string]int64, expected)
	for rows.Next() {
		var externalID string
		var id int64
		if err := rows.Scan(&externalID, &id); err != nil {
			return nil, fmt.Errorf("scan academic %s mapping: %w", label, err)
		}
		mapping[externalID] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan academic %s mapping: %w", label, err)
	}
	if len(mapping) != expected {
		return nil, fmt.Errorf("academic %s import returned %d mappings for %d rows", label, len(mapping), expected)
	}
	return mapping, nil
}

func upsertTermsTx(ctx context.Context, tx pgx.Tx, jobID, sourceID int64, terms []ImportTerm) (map[string]int64, error) {
	payload, err := marshalAcademicImportRows("term", terms)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		WITH input AS (
			SELECT *
			FROM jsonb_to_recordset($3::jsonb) AS item(
				"externalId" text,
				code text,
				name text,
				"startDate" date,
				"endDate" date,
				"isCurrent" boolean
			)
		), upserted AS (
			INSERT INTO academic_terms (
				source_id, import_job_id, external_id, code, name, start_date, end_date, is_current
			)
			SELECT $1, $2, "externalId", code, name, "startDate", "endDate", "isCurrent"
			FROM input
			ON CONFLICT (source_id, external_id)
			DO UPDATE SET import_job_id = EXCLUDED.import_job_id,
			              code = EXCLUDED.code,
			              name = EXCLUDED.name,
			              start_date = EXCLUDED.start_date,
			              end_date = EXCLUDED.end_date,
			              is_current = EXCLUDED.is_current
			RETURNING external_id, id
		)
		SELECT external_id, id FROM upserted
	`, sourceID, jobID, payload)
	if err != nil {
		return nil, fmt.Errorf("bulk upsert academic terms: %w", err)
	}
	return scanAcademicExternalIDMapping(rows, "term", len(terms))
}

func upsertCoursesTx(ctx context.Context, tx pgx.Tx, jobID, sourceID int64, courses []ImportCourse) (map[string]int64, error) {
	payload, err := marshalAcademicImportRows("course", courses)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		WITH input AS (
			SELECT *
			FROM jsonb_to_recordset($3::jsonb) AS item(
				"externalId" text,
				code text,
				name text,
				"departmentCode" text,
				"departmentName" text,
				credit numeric
			)
		), upserted AS (
			INSERT INTO academic_courses (
				source_id, import_job_id, external_id, code, name, department_code, department_name, credit
			)
			SELECT $1, $2, "externalId", code, name, "departmentCode", "departmentName", credit
			FROM input
			ON CONFLICT (source_id, external_id)
			DO UPDATE SET import_job_id = EXCLUDED.import_job_id,
			              code = EXCLUDED.code,
			              name = EXCLUDED.name,
			              department_code = EXCLUDED.department_code,
			              department_name = EXCLUDED.department_name,
			              credit = EXCLUDED.credit
			RETURNING external_id, id
		)
		SELECT external_id, id FROM upserted
	`, sourceID, jobID, payload)
	if err != nil {
		return nil, fmt.Errorf("bulk upsert academic courses: %w", err)
	}
	return scanAcademicExternalIDMapping(rows, "course", len(courses))
}

func upsertTeachersTx(ctx context.Context, tx pgx.Tx, jobID, sourceID int64, teachers []ImportTeacher) (map[string]int64, error) {
	payload, err := marshalAcademicImportRows("teacher", teachers)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		WITH input AS (
			SELECT *
			FROM jsonb_to_recordset($3::jsonb) AS item(
				"externalId" text,
				name text,
				"departmentName" text
			)
		), upserted AS (
			INSERT INTO academic_teachers (source_id, import_job_id, external_id, name, department_name)
			SELECT $1, $2, "externalId", name, "departmentName"
			FROM input
			ON CONFLICT (source_id, external_id)
			DO UPDATE SET import_job_id = EXCLUDED.import_job_id,
			              name = EXCLUDED.name,
			              department_name = EXCLUDED.department_name
			RETURNING external_id, id
		)
		SELECT external_id, id FROM upserted
	`, sourceID, jobID, payload)
	if err != nil {
		return nil, fmt.Errorf("bulk upsert academic teachers: %w", err)
	}
	return scanAcademicExternalIDMapping(rows, "teacher", len(teachers))
}

func upsertOfferingsTx(
	ctx context.Context,
	tx pgx.Tx,
	jobID, sourceID int64,
	offerings []ImportOffering,
	termIDs map[string]int64,
	courseIDs map[string]int64,
) (map[string]int64, error) {
	for _, item := range offerings {
		if _, ok := termIDs[item.TermExternalID]; !ok {
			return nil, fmt.Errorf("unknown academic term external id %q", item.TermExternalID)
		}
		if _, ok := courseIDs[item.CourseExternalID]; !ok {
			return nil, fmt.Errorf("unknown academic course external id %q", item.CourseExternalID)
		}
	}
	payload, err := marshalAcademicImportRows("offering", offerings)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		WITH input AS (
			SELECT *
			FROM jsonb_to_recordset($3::jsonb) AS item(
				"externalId" text,
				"termExternalId" text,
				"courseExternalId" text,
				"sectionCode" text,
				"schoolName" text,
				"departmentName" text,
				campus text,
				"enrollmentLimit" integer
			)
		), upserted AS (
			INSERT INTO academic_offerings (
				source_id, import_job_id, external_id, term_id, course_id, section_code,
				school_name, department_name, campus, enrollment_limit
			)
			SELECT $1, $2, item."externalId", term.id, course.id, item."sectionCode",
			       item."schoolName", item."departmentName", item.campus, item."enrollmentLimit"
			FROM input item
			JOIN academic_terms term
			  ON term.source_id = $1 AND term.external_id = item."termExternalId"
			JOIN academic_courses course
			  ON course.source_id = $1 AND course.external_id = item."courseExternalId"
			ON CONFLICT (source_id, external_id)
			DO UPDATE SET import_job_id = EXCLUDED.import_job_id,
			              term_id = EXCLUDED.term_id,
			              course_id = EXCLUDED.course_id,
			              section_code = EXCLUDED.section_code,
			              school_name = EXCLUDED.school_name,
			              department_name = EXCLUDED.department_name,
			              campus = EXCLUDED.campus,
			              enrollment_limit = EXCLUDED.enrollment_limit
			RETURNING external_id, id
		)
		SELECT external_id, id FROM upserted
	`, sourceID, jobID, payload)
	if err != nil {
		return nil, fmt.Errorf("bulk upsert academic offerings: %w", err)
	}
	return scanAcademicExternalIDMapping(rows, "offering", len(offerings))
}

type academicOfferingTeacherImportRow struct {
	OfferingExternalID string `json:"offeringExternalId"`
	TeacherExternalID  string `json:"teacherExternalId"`
}

type academicScheduleImportRow struct {
	OfferingExternalID string  `json:"offeringExternalId"`
	Weekday            int16   `json:"weekday"`
	StartPeriod        int16   `json:"startPeriod"`
	EndPeriod          int16   `json:"endPeriod"`
	Location           string  `json:"location"`
	Building           *string `json:"building,omitempty"`
	WeeksText          string  `json:"weeksText"`
}

func replaceOfferingRelationsTx(
	ctx context.Context,
	tx pgx.Tx,
	sourceID int64,
	offerings []ImportOffering,
	teacherIDs map[string]int64,
	offeringIDs map[string]int64,
) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM academic_offering_teachers
		WHERE offering_id IN (SELECT id FROM academic_offerings WHERE source_id = $1)
	`, sourceID); err != nil {
		return fmt.Errorf("clear academic offering teachers: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM academic_schedules
		WHERE offering_id IN (SELECT id FROM academic_offerings WHERE source_id = $1)
	`, sourceID); err != nil {
		return fmt.Errorf("clear academic schedules: %w", err)
	}
	teacherRows := make([]academicOfferingTeacherImportRow, 0)
	scheduleRows := make([]academicScheduleImportRow, 0)
	for _, item := range offerings {
		if _, ok := offeringIDs[item.ExternalID]; !ok {
			return fmt.Errorf("unknown academic offering external id %q", item.ExternalID)
		}
		for _, teacherExternalID := range item.TeacherExternalIDs {
			if _, ok := teacherIDs[teacherExternalID]; !ok {
				return fmt.Errorf("unknown academic teacher external id %q", teacherExternalID)
			}
			teacherRows = append(teacherRows, academicOfferingTeacherImportRow{
				OfferingExternalID: item.ExternalID,
				TeacherExternalID:  teacherExternalID,
			})
		}
		for _, schedule := range item.Schedules {
			scheduleRows = append(scheduleRows, academicScheduleImportRow{
				OfferingExternalID: item.ExternalID,
				Weekday:            schedule.Weekday,
				StartPeriod:        schedule.StartPeriod,
				EndPeriod:          schedule.EndPeriod,
				Location:           schedule.Location,
				Building:           schedule.Building,
				WeeksText:          schedule.WeeksText,
			})
		}
	}
	teacherPayload, err := marshalAcademicImportRows("offering teacher", teacherRows)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		WITH input AS (
			SELECT *
			FROM jsonb_to_recordset($2::jsonb) AS item(
				"offeringExternalId" text,
				"teacherExternalId" text
			)
		)
		INSERT INTO academic_offering_teachers (offering_id, teacher_id)
		SELECT DISTINCT offering.id, teacher.id
		FROM input item
		JOIN academic_offerings offering
		  ON offering.source_id = $1 AND offering.external_id = item."offeringExternalId"
		JOIN academic_teachers teacher
		  ON teacher.source_id = $1 AND teacher.external_id = item."teacherExternalId"
	`, sourceID, teacherPayload); err != nil {
		return fmt.Errorf("bulk insert academic offering teacher relations: %w", err)
	}
	schedulePayload, err := marshalAcademicImportRows("schedule", scheduleRows)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		WITH input AS (
			SELECT *
			FROM jsonb_to_recordset($2::jsonb) AS item(
				"offeringExternalId" text,
				weekday smallint,
				"startPeriod" smallint,
				"endPeriod" smallint,
				location text,
				building text,
				"weeksText" text
			)
		)
		INSERT INTO academic_schedules (
			offering_id, weekday, start_period, end_period, location, building, weeks_text
		)
		SELECT offering.id, item.weekday, item."startPeriod", item."endPeriod",
		       item.location, item.building, item."weeksText"
		FROM input item
		JOIN academic_offerings offering
		  ON offering.source_id = $1 AND offering.external_id = item."offeringExternalId"
	`, sourceID, schedulePayload)
	if err != nil {
		return fmt.Errorf("bulk insert academic schedules: %w", err)
	}
	if tag.RowsAffected() != int64(len(scheduleRows)) {
		return fmt.Errorf("bulk insert academic schedules affected %d rows for %d inputs", tag.RowsAffected(), len(scheduleRows))
	}
	return nil
}

func replaceMembershipsTx(
	ctx context.Context,
	tx pgx.Tx,
	jobID, sourceID int64,
	memberships []ImportMembership,
	offeringIDs map[string]int64,
) error {
	if _, err := tx.Exec(ctx, `DELETE FROM academic_memberships WHERE source_id = $1`, sourceID); err != nil {
		return fmt.Errorf("clear academic memberships: %w", err)
	}
	for _, item := range memberships {
		if _, ok := offeringIDs[item.OfferingExternalID]; !ok {
			return fmt.Errorf("unknown academic membership offering external id %q", item.OfferingExternalID)
		}
	}
	payload, err := marshalAcademicImportRows("membership", memberships)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		WITH input AS (
			SELECT *
			FROM jsonb_to_recordset($3::jsonb) AS item(
				"offeringExternalId" text,
				"externalUserId" text,
				"studentId" text,
				role text
			)
		)
		INSERT INTO academic_memberships (
			source_id, import_job_id, offering_id, external_user_id, student_id, role
		)
		SELECT $1, $2, offering.id, item."externalUserId", item."studentId", item.role
		FROM input item
		JOIN academic_offerings offering
		  ON offering.source_id = $1 AND offering.external_id = item."offeringExternalId"
	`, sourceID, jobID, payload)
	if err != nil {
		return fmt.Errorf("bulk insert academic memberships: %w", err)
	}
	if tag.RowsAffected() != int64(len(memberships)) {
		return fmt.Errorf("bulk insert academic memberships affected %d rows for %d inputs", tag.RowsAffected(), len(memberships))
	}
	return nil
}
