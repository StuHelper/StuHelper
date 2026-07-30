package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestListFavorites_PreservesNullableCourseMetadata(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	var courseID int64
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		INSERT INTO courses (school_id, name, category)
		VALUES ($1, $2, $3)
		RETURNING id
	`, int64(4111010006), "收藏 NULL 元数据课程", "待分类").Scan(&courseID))
	_, err := fixture.Pool.Exec(ctx, `
		INSERT INTO course_favorites (id, user_hash, course_id)
		VALUES ($1, $2, $3)
	`, "33333333-3333-4333-8333-333333333333", "nullable-course-favorite-user", courseID)
	require.NoError(t, err)

	favorites, total, err := repo.ListFavorites(ctx, "nullable-course-favorite-user", 10, 0)

	require.NoError(t, err)
	require.Len(t, favorites, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, courseID, favorites[0].ID)
	assert.Nil(t, favorites[0].Code)
	assert.Nil(t, favorites[0].Credits)
	assert.Nil(t, favorites[0].DepartmentID)
	assert.Nil(t, favorites[0].DepartmentName)
}
