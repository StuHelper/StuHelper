package academics

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestTriggerImportAndQueryReadModel(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, NewRegistry())
	ctx := context.Background()

	job, err := svc.TriggerImport(ctx, "buaa-fixture", "oidc-user-1")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "succeeded", job.Status)

	terms, err := svc.ListTerms(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, terms)
	assert.Equal(t, "2026-SPRING", terms[0].Code)

	offerings, total, err := svc.ListOfferings(ctx, OfferingFilters{
		TermCode: "2026-SPRING",
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.NotEmpty(t, offerings)
	assert.GreaterOrEqual(t, total, 1)

	offerings, total, err = svc.ListOfferings(ctx, OfferingFilters{
		TermCode: "2026-SPRING",
		Page:     0,
		PageSize: 0,
	})
	require.NoError(t, err)
	require.NotEmpty(t, offerings)
	assert.GreaterOrEqual(t, total, 1)

	detail, err := svc.GetOffering(ctx, offerings[0].ID)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.NotEmpty(t, detail.Schedules)

	myCourses, err := svc.ListMyCourses(ctx, "oidc-user-1", "2026-SPRING")
	require.NoError(t, err)
	require.Len(t, myCourses, 2)

	mySchedule, err := svc.ListMySchedule(ctx, "oidc-user-1", "2026-SPRING")
	require.NoError(t, err)
	require.Len(t, mySchedule, 2)
	assert.NotEmpty(t, mySchedule[0].Schedules)
}

func TestTriggerImportRejectsDisabledSource(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, NewRegistry())
	ctx := context.Background()

	_, err := fixture.DB.Exec(ctx, `UPDATE academic_sources SET enabled = FALSE WHERE key = $1`, "buaa-fixture")
	require.NoError(t, err)

	_, err = svc.TriggerImport(ctx, "buaa-fixture", "oidc-user-1")
	require.ErrorIs(t, err, ErrSourceDisabled)
}

func TestReplaceSnapshotRemovesStaleReadModelRows(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	source, err := repo.GetSourceByKey(ctx, "buaa-fixture")
	require.NoError(t, err)

	connector, err := newFixtureConnector(*source)
	require.NoError(t, err)
	initialSnapshot, err := loadSnapshot(ctx, connector)
	require.NoError(t, err)

	job1, err := repo.CreateImportJob(ctx, *source, "oidc-user-1")
	require.NoError(t, err)
	require.NoError(t, repo.ReplaceSnapshot(ctx, job1, source.ID, initialSnapshot))

	initialOfferings, total, err := repo.ListOfferings(ctx, OfferingFilters{
		TermCode: "2026-SPRING",
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Len(t, initialOfferings, 2)
	assert.Equal(t, 2, total)
	var removedOfferingID int64
	err = fixture.DB.QueryRow(ctx, `
		SELECT id
		FROM academic_offerings
		WHERE source_id = $1 AND external_id = $2
	`, source.ID, initialSnapshot.Offerings[1].ExternalID).Scan(&removedOfferingID)
	require.NoError(t, err)

	prunedSnapshot := Snapshot{
		Terms:       append([]ImportTerm(nil), initialSnapshot.Terms...),
		Courses:     append([]ImportCourse(nil), initialSnapshot.Courses[:1]...),
		Teachers:    append([]ImportTeacher(nil), initialSnapshot.Teachers[:1]...),
		Offerings:   append([]ImportOffering(nil), initialSnapshot.Offerings[:1]...),
		Memberships: []ImportMembership{initialSnapshot.Memberships[0]},
	}

	job2, err := repo.CreateImportJob(ctx, *source, "oidc-user-1")
	require.NoError(t, err)
	require.NoError(t, repo.ReplaceSnapshot(ctx, job2, source.ID, prunedSnapshot))

	currentOfferings, total, err := repo.ListOfferings(ctx, OfferingFilters{
		TermCode: "2026-SPRING",
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Len(t, currentOfferings, 1)
	assert.Equal(t, 1, total)
	assert.NotEqual(t, removedOfferingID, currentOfferings[0].ID)

	_, err = repo.GetOfferingByID(ctx, removedOfferingID)
	require.ErrorIs(t, err, ErrOfferingNotFound)

	myCourses, err := repo.ListMyCourses(ctx, "oidc-user-1", "2026-SPRING")
	require.NoError(t, err)
	assert.Len(t, myCourses, 1)
}

func TestReplaceSnapshotLoadsLargeSnapshotSetWise(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	source, err := repo.GetSourceByKey(ctx, "buaa-fixture")
	require.NoError(t, err)
	jobID, err := repo.CreateImportJob(ctx, *source, "oidc-user-bulk")
	require.NoError(t, err)

	const itemCount = 1000
	snapshot := Snapshot{
		Terms: []ImportTerm{{
			ExternalID: "term-bulk",
			Code:       "2099-SPRING",
			Name:       "批量导入学期",
			IsCurrent:  true,
		}},
		Courses:     make([]ImportCourse, 0, itemCount),
		Teachers:    make([]ImportTeacher, 0, itemCount),
		Offerings:   make([]ImportOffering, 0, itemCount),
		Memberships: make([]ImportMembership, 0, itemCount),
	}
	for index := range itemCount {
		externalID := fmt.Sprintf("bulk-%04d", index)
		courseID := "course-" + externalID
		teacherID := "teacher-" + externalID
		offeringID := "offering-" + externalID
		snapshot.Courses = append(snapshot.Courses, ImportCourse{
			ExternalID: courseID,
			Code:       fmt.Sprintf("C%04d", index),
			Name:       "批量课程 " + externalID,
		})
		snapshot.Teachers = append(snapshot.Teachers, ImportTeacher{
			ExternalID: teacherID,
			Name:       "批量教师 " + externalID,
		})
		snapshot.Offerings = append(snapshot.Offerings, ImportOffering{
			ExternalID:         offeringID,
			TermExternalID:     "term-bulk",
			CourseExternalID:   courseID,
			SectionCode:        fmt.Sprintf("S%04d", index),
			TeacherExternalIDs: []string{teacherID},
			Schedules: []ScheduleSlot{{
				Weekday:     int16(index%7 + 1),
				StartPeriod: 1,
				EndPeriod:   2,
				Location:    "批量教室",
				WeeksText:   "1-16",
			}},
		})
		snapshot.Memberships = append(snapshot.Memberships, ImportMembership{
			OfferingExternalID: offeringID,
			ExternalUserID:     fmt.Sprintf("bulk-user-%04d", index),
			Role:               "student",
		})
	}

	require.NoError(t, repo.ReplaceSnapshot(ctx, jobID, source.ID, snapshot))

	var termCount, courseCount, teacherCount, offeringCount int
	var offeringTeacherCount, scheduleCount, membershipCount int
	err = fixture.DB.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM academic_terms WHERE source_id = $1),
			(SELECT COUNT(*) FROM academic_courses WHERE source_id = $1),
			(SELECT COUNT(*) FROM academic_teachers WHERE source_id = $1),
			(SELECT COUNT(*) FROM academic_offerings WHERE source_id = $1),
			(SELECT COUNT(*) FROM academic_offering_teachers relation
			 JOIN academic_offerings offering ON offering.id = relation.offering_id
			 WHERE offering.source_id = $1),
			(SELECT COUNT(*) FROM academic_schedules schedule
			 JOIN academic_offerings offering ON offering.id = schedule.offering_id
			 WHERE offering.source_id = $1),
			(SELECT COUNT(*) FROM academic_memberships WHERE source_id = $1)
	`, source.ID).Scan(
		&termCount,
		&courseCount,
		&teacherCount,
		&offeringCount,
		&offeringTeacherCount,
		&scheduleCount,
		&membershipCount,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, termCount)
	assert.Equal(t, itemCount, courseCount)
	assert.Equal(t, itemCount, teacherCount)
	assert.Equal(t, itemCount, offeringCount)
	assert.Equal(t, itemCount, offeringTeacherCount)
	assert.Equal(t, itemCount, scheduleCount)
	assert.Equal(t, itemCount, membershipCount)
}
