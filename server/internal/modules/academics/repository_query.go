package academics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
)

func (r *Repository) ListTerms(ctx context.Context) ([]Term, error) {
	ctx = withDBTable(ctx, "academic_terms")
	rows, err := r.db.Query(ctx, `
		SELECT id, code, name, start_date, end_date, is_current
		FROM academic_terms
		ORDER BY is_current DESC, code DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list academic terms: %w", err)
	}
	defer rows.Close()
	var terms []Term
	for rows.Next() {
		var item Term
		var startDate *time.Time
		var endDate *time.Time
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &startDate, &endDate, &item.IsCurrent); err != nil {
			return nil, fmt.Errorf("scan academic term: %w", err)
		}
		item.StartDate = formatDatePtr(startDate)
		item.EndDate = formatDatePtr(endDate)
		terms = append(terms, item)
	}
	return terms, rows.Err()
}

func (r *Repository) ListOfferings(ctx context.Context, filters OfferingFilters) ([]Offering, int, error) {
	ctx = withDBTable(ctx, "academic_offerings")
	offset := httputil.SafeOffset(filters.Page, filters.PageSize)
	pattern := "%" + httputil.EscapeLikePattern(filters.CourseQuery) + "%"
	teacherPattern := "%" + httputil.EscapeLikePattern(filters.TeacherQuery) + "%"
	schoolPattern := "%" + httputil.EscapeLikePattern(filters.SchoolName) + "%"
	departmentPattern := "%" + httputil.EscapeLikePattern(filters.DepartmentName) + "%"
	rows, err := r.db.Query(ctx, `
		SELECT o.id, t.code, t.name, c.code, c.name, o.section_code, o.school_name, o.department_name, o.campus,
		       COALESCE(teachers.teacher_names, ''), COUNT(*) OVER()
		FROM academic_offerings o
		JOIN academic_terms t ON t.id = o.term_id
		JOIN academic_courses c ON c.id = o.course_id
		LEFT JOIN LATERAL (
			SELECT string_agg(at.name, ', ' ORDER BY at.name) AS teacher_names
			FROM academic_offering_teachers aot
			JOIN academic_teachers at ON at.id = aot.teacher_id
			WHERE aot.offering_id = o.id
		) teachers ON TRUE
		WHERE ($1 = '' OR t.code = $1)
		  AND ($2 = '%%' OR COALESCE(o.school_name, '') ILIKE $2 ESCAPE '\')
		  AND ($3 = '%%' OR COALESCE(o.department_name, '') ILIKE $3 ESCAPE '\')
		  AND ($4 = '%%' OR c.name ILIKE $4 ESCAPE '\' OR c.code ILIKE $4 ESCAPE '\')
		  AND ($5 = '%%' OR EXISTS (
			SELECT 1
			FROM academic_offering_teachers aot
			JOIN academic_teachers at ON at.id = aot.teacher_id
			WHERE aot.offering_id = o.id AND at.name ILIKE $5 ESCAPE '\'
		  ))
		ORDER BY t.code DESC, c.name ASC, o.section_code ASC, o.id ASC
		LIMIT $6 OFFSET $7
	`, filters.TermCode, schoolPattern, departmentPattern, pattern, teacherPattern, filters.PageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list academic offerings: %w", err)
	}
	defer rows.Close()
	return scanOfferings(rows)
}

func (r *Repository) GetOfferingByID(ctx context.Context, offeringID int64) (*Offering, error) {
	ctx = withDBTable(ctx, "academic_offerings")
	rows, err := r.db.Query(ctx, `
		SELECT o.id, t.code, t.name, c.code, c.name, o.section_code, o.school_name, o.department_name, o.campus,
		       COALESCE(teachers.teacher_names, ''), COUNT(*) OVER()
		FROM academic_offerings o
		JOIN academic_terms t ON t.id = o.term_id
		JOIN academic_courses c ON c.id = o.course_id
		LEFT JOIN LATERAL (
			SELECT string_agg(at.name, ', ' ORDER BY at.name) AS teacher_names
			FROM academic_offering_teachers aot
			JOIN academic_teachers at ON at.id = aot.teacher_id
			WHERE aot.offering_id = o.id
		) teachers ON TRUE
		WHERE o.id = $1
	`, offeringID)
	if err != nil {
		return nil, fmt.Errorf("get academic offering: %w", err)
	}
	defer rows.Close()
	list, _, err := scanOfferings(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrOfferingNotFound
	}
	schedules, err := r.listSchedulesByOfferingID(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	list[0].Schedules = schedules
	return &list[0], nil
}

func (r *Repository) ListMyCourses(ctx context.Context, externalUserID, termCode string) ([]Offering, error) {
	ctx = withDBTable(ctx, "academic_memberships")
	rows, err := r.db.Query(ctx, `
		SELECT o.id, t.code, t.name, c.code, c.name, o.section_code, o.school_name, o.department_name, o.campus,
		       COALESCE(teachers.teacher_names, ''), COUNT(*) OVER()
		FROM academic_memberships m
		JOIN academic_offerings o ON o.id = m.offering_id
		JOIN academic_terms t ON t.id = o.term_id
		JOIN academic_courses c ON c.id = o.course_id
		LEFT JOIN LATERAL (
			SELECT string_agg(at.name, ', ' ORDER BY at.name) AS teacher_names
			FROM academic_offering_teachers aot
			JOIN academic_teachers at ON at.id = aot.teacher_id
			WHERE aot.offering_id = o.id
		) teachers ON TRUE
		WHERE m.external_user_id = $1 AND m.role = 'student'
		  AND ($2 = '' OR t.code = $2)
		ORDER BY t.code DESC, c.name ASC, o.section_code ASC, o.id ASC
	`, externalUserID, termCode)
	if err != nil {
		return nil, fmt.Errorf("list my academic courses: %w", err)
	}
	defer rows.Close()
	list, _, err := scanOfferings(rows)
	return list, err
}

func (r *Repository) ListMySchedule(ctx context.Context, externalUserID, termCode string) ([]Offering, error) {
	courses, err := r.ListMyCourses(ctx, externalUserID, termCode)
	if err != nil {
		return nil, err
	}
	for i := range courses {
		schedules, err := r.listSchedulesByOfferingID(ctx, courses[i].ID)
		if err != nil {
			return nil, err
		}
		courses[i].Schedules = schedules
	}
	return courses, nil
}

func (r *Repository) listSchedulesByOfferingID(ctx context.Context, offeringID int64) ([]ScheduleSlot, error) {
	ctx = withDBTable(ctx, "academic_schedules")
	rows, err := r.db.Query(ctx, `
		SELECT weekday, start_period, end_period, location, building, weeks_text
		FROM academic_schedules
		WHERE offering_id = $1
		ORDER BY weekday ASC, start_period ASC, end_period ASC, id ASC
	`, offeringID)
	if err != nil {
		return nil, fmt.Errorf("list academic schedules: %w", err)
	}
	defer rows.Close()
	var schedules []ScheduleSlot
	for rows.Next() {
		var item ScheduleSlot
		if err := rows.Scan(&item.Weekday, &item.StartPeriod, &item.EndPeriod, &item.Location, &item.Building, &item.WeeksText); err != nil {
			return nil, fmt.Errorf("scan academic schedule: %w", err)
		}
		schedules = append(schedules, item)
	}
	return schedules, rows.Err()
}

func scanOfferings(rows pgx.Rows) ([]Offering, int, error) {
	list := make([]Offering, 0)
	total := 0
	for rows.Next() {
		var item Offering
		var teacherNames string
		if err := rows.Scan(
			&item.ID, &item.TermCode, &item.TermName, &item.CourseCode, &item.CourseName,
			&item.SectionCode, &item.SchoolName, &item.DepartmentName, &item.Campus, &teacherNames, &total,
		); err != nil {
			return nil, 0, fmt.Errorf("scan academic offering: %w", err)
		}
		if teacherNames != "" {
			item.TeacherNames = strings.Split(teacherNames, ", ")
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

func formatDatePtr(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format("2006-01-02")
	return &formatted
}
