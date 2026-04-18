package notification

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestNewServicePanicsOnNilDependencies(t *testing.T) {
	tests := []struct {
		name string
		repo *Repository
		hub  *Hub
		rdb  *redis.Client
	}{
		{name: "nil repo", hub: &Hub{}, rdb: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})},
		{name: "nil hub", repo: &Repository{}, rdb: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})},
		{name: "nil redis", repo: &Repository{}, hub: &Hub{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic")
				}
			}()
			_ = NewService(tt.repo, tt.hub, tt.rdb)
		})
	}
}
