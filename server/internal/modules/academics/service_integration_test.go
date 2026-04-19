package academics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
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
