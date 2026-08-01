package review

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
)

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	assert.PanicsWithValue(t, "review.NewRepository: database must not be nil", func() {
		NewRepository(nil)
	})
}

func TestNewRepositoryConfiguresTeacherStatsRefreshTimeout(t *testing.T) {
	database := &db.DB{}

	defaultRepository := NewRepository(database)
	assert.Equal(t, defaultTeacherStatsRefreshTimeout, defaultRepository.teacherStatsRefreshTimeout)

	configuredRepository := NewRepository(database, WithTeacherStatsRefreshTimeout(75*time.Second))
	assert.Equal(t, 75*time.Second, configuredRepository.teacherStatsRefreshTimeout)

	fallbackRepository := NewRepository(database, WithTeacherStatsRefreshTimeout(0))
	assert.Equal(t, defaultTeacherStatsRefreshTimeout, fallbackRepository.teacherStatsRefreshTimeout)
}
