package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepositoryCacheStatsUsesDailyAggregates(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery("FROM usage_dashboard_daily").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"key",
			"label",
			"requests",
			"input_tokens",
			"output_tokens",
			"cache_creation_tokens",
			"cache_read_tokens",
			"cost",
			"actual_cost",
			"account_cost",
		}).AddRow(
			"summary",
			"Summary",
			int64(3),
			int64(22),
			int64(33),
			int64(2),
			int64(1),
			float64(2.2),
			float64(2.1),
			float64(2.0),
		))

	mock.ExpectQuery("FROM usage_dashboard_daily").
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"key",
			"label",
			"requests",
			"input_tokens",
			"output_tokens",
			"cache_creation_tokens",
			"cache_read_tokens",
			"cost",
			"actual_cost",
			"account_cost",
		}).AddRow(
			"2026-05-09",
			"2026-05-09",
			int64(3),
			int64(22),
			int64(33),
			int64(2),
			int64(1),
			float64(2.2),
			float64(2.1),
			float64(2.0),
		))

	resp, err := repo.GetCacheStatsWithFilters(context.Background(), usagestats.CacheStatsQuery{
		StartTime: start,
		EndTime:   end,
		Dimension: usagestats.CacheStatsDimensionDay,
		Timezone:  "UTC",
	})
	require.NoError(t, err)
	require.Equal(t, usagestats.CacheStatsDimensionDay, resp.Dimension)
	require.Equal(t, int64(3), resp.Summary.Requests)
	require.Equal(t, int64(2), resp.Summary.CacheCreationTokens)
	require.Equal(t, int64(1), resp.Summary.CacheReadTokens)
	require.Len(t, resp.Items, 1)
	require.Equal(t, "2026-05-09", resp.Items[0].Key)
	require.Equal(t, float64(3)/float64(58), resp.Summary.CacheTokenRate)
	require.Equal(t, float64(1)/float64(23), resp.Summary.CacheReadRate)
	require.NoError(t, mock.ExpectationsWereMet())
}
