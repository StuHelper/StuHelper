package academics

import (
	"context"
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

func upsertTermsTx(ctx context.Context, tx pgx.Tx, jobID, sourceID int64, terms []ImportTerm) (map[string]int64, error) {
	mapping := make(map[string]int64, len(terms))
	for _, item := range terms {
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO academic_terms (source_id, import_job_id, external_id, code, name, start_date, end_date, is_current)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (source_id, external_id)
			DO UPDATE SET import_job_id = EXCLUDED.import_job_id, code = EXCLUDED.code, name = EXCLUDED.name,
			              start_date = EXCLUDED.start_date, end_date = EXCLUDED.end_date, is_current = EXCLUDED.is_current
			RETURNING id
		`, sourceID, jobID, item.ExternalID, item.Code, item.Name, item.StartDate, item.EndDate, item.IsCurrent).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert academic term %q: %w", item.ExternalID, err)
		}
		mapping[item.ExternalID] = id
	}
	return mapping, nil
}

func upsertCoursesTx(ctx context.Context, tx pgx.Tx, jobID, sourceID int64, courses []ImportCourse) (map[string]int64, error) {
	mapping := make(map[string]int64, len(courses))
	for _, item := range courses {
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO academic_courses (source_id, import_job_id, external_id, code, name, department_code, department_name, credit)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (source_id, external_id)
			DO UPDATE SET import_job_id = EXCLUDED.import_job_id, code = EXCLUDED.code, name = EXCLUDED.name,
			              department_code = EXCLUDED.department_code, department_name = EXCLUDED.department_name, credit = EXCLUDED.credit
			RETURNING id
		`, sourceID, jobID, item.ExternalID, item.Code, item.Name, item.DepartmentCode, item.DepartmentName, item.Credit).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert academic course %q: %w", item.ExternalID, err)
		}
		mapping[item.ExternalID] = id
	}
	return mapping, nil
}

func upsertTeachersTx(ctx context.Context, tx pgx.Tx, jobID, sourceID int64, teachers []ImportTeacher) (map[string]int64, error) {
	mapping := make(map[string]int64, len(teachers))
	for _, item := range teachers {
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO academic_teachers (source_id, import_job_id, external_id, name, department_name)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (source_id, external_id)
			DO UPDATE SET import_job_id = EXCLUDED.import_job_id, name = EXCLUDED.name, department_name = EXCLUDED.department_name
			RETURNING id
		`, sourceID, jobID, item.ExternalID, item.Name, item.DepartmentName).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert academic teacher %q: %w", item.ExternalID, err)
		}
		mapping[item.ExternalID] = id
	}
	return mapping, nil
}

func upsertOfferingsTx(
	ctx context.Context,
	tx pgx.Tx,
	jobID, sourceID int64,
	offerings []ImportOffering,
	termIDs map[string]int64,
	courseIDs map[string]int64,
) (map[string]int64, error) {
	mapping := make(map[string]int64, len(offerings))
	for _, item := range offerings {
		termID, ok := termIDs[item.TermExternalID]
		if !ok {
			return nil, fmt.Errorf("unknown academic term external id %q", item.TermExternalID)
		}
		courseID, ok := courseIDs[item.CourseExternalID]
		if !ok {
			return nil, fmt.Errorf("unknown academic course external id %q", item.CourseExternalID)
		}
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO academic_offerings (source_id, import_job_id, external_id, term_id, course_id, section_code, school_name, department_name, campus, enrollment_limit)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (source_id, external_id)
			DO UPDATE SET import_job_id = EXCLUDED.import_job_id, term_id = EXCLUDED.term_id, course_id = EXCLUDED.course_id,
			              section_code = EXCLUDED.section_code, school_name = EXCLUDED.school_name,
			              department_name = EXCLUDED.department_name, campus = EXCLUDED.campus, enrollment_limit = EXCLUDED.enrollment_limit
			RETURNING id
		`, sourceID, jobID, item.ExternalID, termID, courseID, item.SectionCode, item.SchoolName, item.DepartmentName, item.Campus, item.EnrollmentLimit).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("upsert academic offering %q: %w", item.ExternalID, err)
		}
		mapping[item.ExternalID] = id
	}
	return mapping, nil
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
	for _, item := range offerings {
		offeringID := offeringIDs[item.ExternalID]
		for _, teacherExternalID := range item.TeacherExternalIDs {
			teacherID, ok := teacherIDs[teacherExternalID]
			if !ok {
				return fmt.Errorf("unknown academic teacher external id %q", teacherExternalID)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO academic_offering_teachers (offering_id, teacher_id)
				VALUES ($1, $2)
			`, offeringID, teacherID); err != nil {
				return fmt.Errorf("insert academic offering teacher relation: %w", err)
			}
		}
		for _, schedule := range item.Schedules {
			if _, err := tx.Exec(ctx, `
				INSERT INTO academic_schedules (offering_id, weekday, start_period, end_period, location, building, weeks_text)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, offeringID, schedule.Weekday, schedule.StartPeriod, schedule.EndPeriod, schedule.Location, schedule.Building, schedule.WeeksText); err != nil {
				return fmt.Errorf("insert academic schedule: %w", err)
			}
		}
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
		offeringID, ok := offeringIDs[item.OfferingExternalID]
		if !ok {
			return fmt.Errorf("unknown academic membership offering external id %q", item.OfferingExternalID)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO academic_memberships (source_id, import_job_id, offering_id, external_user_id, student_id, role)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, sourceID, jobID, offeringID, item.ExternalUserID, item.StudentID, item.Role); err != nil {
			return fmt.Errorf("insert academic membership: %w", err)
		}
	}
	return nil
}
