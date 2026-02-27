package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Event types
type RideCompletionEvent struct {
	RideID        string    `json:"ride_id"`
	DriverID      string    `json:"driver_id"`
	RiderID       string    `json:"rider_id"`
	City          string    `json:"city"`
	Distance      float64   `json:"distance"`
	Fare          float64   `json:"fare"`
	Duration      int       `json:"duration"`
	StartLat      float64   `json:"start_lat"`
	StartLng      float64   `json:"start_lng"`
	EndLat        float64   `json:"end_lat"`
	EndLng        float64   `json:"end_lng"`
	CompletedAt   time.Time `json:"completed_at"`
}

type RideCancellationEvent struct {
	RideID       string    `json:"ride_id"`
	DriverID     string    `json:"driver_id"`
	RiderID      string    `json:"rider_id"`
	City         string    `json:"city"`
	CancelledBy  string    `json:"cancelled_by"`
	Reason       string    `json:"reason"`
	CancelledAt  time.Time `json:"cancelled_at"`
}

// Metric result types
type DashboardMetrics struct {
	TotalRidesToday        int     `json:"total_rides_today"`
	TotalRevenueToday      float64 `json:"total_revenue_today"`
	ActiveDrivers          int     `json:"active_drivers"`
	ActiveRiders           int     `json:"active_riders"`
	AverageRating          float64 `json:"average_rating"`
	CancellationRate       float64 `json:"cancellation_rate"`
	AverageWaitTime        float64 `json:"average_wait_time"`
	CompletionRate         float64 `json:"completion_rate"`
}

type RideMetrics struct {
	TotalRides         int     `json:"total_rides"`
	CompletedRides     int     `json:"completed_rides"`
	CancelledRides     int     `json:"cancelled_rides"`
	AverageDistance    float64 `json:"average_distance"`
	AverageDuration    float64 `json:"average_duration"`
	AverageFare        float64 `json:"average_fare"`
	TotalDistance      float64 `json:"total_distance"`
	CancellationRate   float64 `json:"cancellation_rate"`
}

type RevenueMetrics struct {
	Period        string  `json:"period"`
	TotalRevenue  float64 `json:"total_revenue"`
	AverageFare   float64 `json:"average_fare"`
	TotalRides    int     `json:"total_rides"`
	GrowthRate    float64 `json:"growth_rate"`
}

type DriverMetrics struct {
	TotalDrivers      int     `json:"total_drivers"`
	ActiveDrivers     int     `json:"active_drivers"`
	AverageRating     float64 `json:"average_rating"`
	AverageRidesPerDay float64 `json:"average_rides_per_day"`
	TopPerformers     []DriverPerformance `json:"top_performers"`
}

type RiderMetrics struct {
	TotalRiders        int     `json:"total_riders"`
	ActiveRiders       int     `json:"active_riders"`
	AverageRidesPerRider float64 `json:"average_rides_per_rider"`
	RetentionRate      float64 `json:"retention_rate"`
	NewRidersThisMonth int     `json:"new_riders_this_month"`
}

type HeatmapData struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Intensity int     `json:"intensity"`
}

type TrendData struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label"`
}

type PeakHourData struct {
	Hour        int     `json:"hour"`
	RideCount   int     `json:"ride_count"`
	DemandScore float64 `json:"demand_score"`
}

type PopularRoute struct {
	StartZone   string  `json:"start_zone"`
	EndZone     string  `json:"end_zone"`
	RideCount   int     `json:"ride_count"`
	AverageFare float64 `json:"average_fare"`
}

type DriverPerformance struct {
	DriverID       string  `json:"driver_id"`
	TotalRides     int     `json:"total_rides"`
	TotalRevenue   float64 `json:"total_revenue"`
	AverageRating  float64 `json:"average_rating"`
	CancellationRate float64 `json:"cancellation_rate"`
	OnlineHours    float64 `json:"online_hours"`
}

type RiderBehavior struct {
	RiderID           string    `json:"rider_id"`
	TotalRides        int       `json:"total_rides"`
	TotalSpent        float64   `json:"total_spent"`
	FavoriteCity      string    `json:"favorite_city"`
	AverageRideFare   float64   `json:"average_ride_fare"`
	PreferredHours    []int     `json:"preferred_hours"`
	CancellationRate  float64   `json:"cancellation_rate"`
	FirstRideAt       time.Time `json:"first_ride_at"`
	LastRideAt        time.Time `json:"last_ride_at"`
}

// Report types
type Report struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	City        string                 `json:"city"`
	StartDate   time.Time              `json:"start_date"`
	EndDate     time.Time              `json:"end_date"`
	Parameters  map[string]interface{} `json:"parameters"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"`
	Status      string                 `json:"status"`
}

type ReportFilters struct {
	Type      string
	City      string
	CreatedBy string
	StartDate *time.Time
	EndDate   *time.Time
	Status    string
	Limit     int
	Offset    int
}

// MetricsRepository handles metrics data storage and retrieval
type MetricsRepository struct {
	db     *sql.DB
	redis  *redis.Client
	logger *zap.Logger
}

// NewMetricsRepository creates a new MetricsRepository
func NewMetricsRepository(db *sql.DB, redisClient *redis.Client, logger *zap.Logger) *MetricsRepository {
	return &MetricsRepository{
		db:     db,
		redis:  redisClient,
		logger: logger,
	}
}

// RecordRideCompletion records a ride completion event in the database
func (r *MetricsRepository) RecordRideCompletion(ctx context.Context, event RideCompletionEvent) error {
	query := `
		INSERT INTO ride_events (
			ride_id, driver_id, rider_id, city, event_type,
			distance, fare, duration,
			start_lat, start_lng, end_lat, end_lng, occurred_at
		) VALUES ($1, $2, $3, $4, 'completed', $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		event.RideID,
		event.DriverID,
		event.RiderID,
		event.City,
		event.Distance,
		event.Fare,
		event.Duration,
		event.StartLat,
		event.StartLng,
		event.EndLat,
		event.EndLng,
		event.CompletedAt,
	)
	if err != nil {
		r.logger.Error("failed to record ride completion", zap.String("ride_id", event.RideID), zap.Error(err))
		return fmt.Errorf("failed to record ride completion: %w", err)
	}

	// Invalidate dashboard cache for the city
	cacheKey := fmt.Sprintf("dashboard:metrics:%s", event.City)
	if err := r.redis.Del(ctx, cacheKey).Err(); err != nil {
		r.logger.Warn("failed to invalidate dashboard cache", zap.String("key", cacheKey), zap.Error(err))
	}

	r.logger.Info("recorded ride completion", zap.String("ride_id", event.RideID), zap.String("city", event.City))
	return nil
}

// RecordRideCancellation records a ride cancellation event in the database
func (r *MetricsRepository) RecordRideCancellation(ctx context.Context, event RideCancellationEvent) error {
	query := `
		INSERT INTO ride_events (
			ride_id, driver_id, rider_id, city, event_type,
			cancelled_by, cancellation_reason, occurred_at
		) VALUES ($1, $2, $3, $4, 'cancelled', $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		event.RideID,
		event.DriverID,
		event.RiderID,
		event.City,
		event.CancelledBy,
		event.Reason,
		event.CancelledAt,
	)
	if err != nil {
		r.logger.Error("failed to record ride cancellation", zap.String("ride_id", event.RideID), zap.Error(err))
		return fmt.Errorf("failed to record ride cancellation: %w", err)
	}

	// Invalidate dashboard cache for the city
	cacheKey := fmt.Sprintf("dashboard:metrics:%s", event.City)
	if err := r.redis.Del(ctx, cacheKey).Err(); err != nil {
		r.logger.Warn("failed to invalidate dashboard cache", zap.String("key", cacheKey), zap.Error(err))
	}

	r.logger.Info("recorded ride cancellation", zap.String("ride_id", event.RideID), zap.String("city", event.City))
	return nil
}

// UpdateDriverStatus updates the current status of a driver
func (r *MetricsRepository) UpdateDriverStatus(ctx context.Context, driverID, status, city string) error {
	query := `
		INSERT INTO driver_status (driver_id, status, city, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (driver_id)
		DO UPDATE SET status = EXCLUDED.status, city = EXCLUDED.city, updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, driverID, status, city)
	if err != nil {
		r.logger.Error("failed to update driver status", zap.String("driver_id", driverID), zap.Error(err))
		return fmt.Errorf("failed to update driver status: %w", err)
	}

	// Update driver status in Redis for fast lookups
	statusKey := fmt.Sprintf("driver:status:%s", driverID)
	statusData := map[string]interface{}{
		"status":     status,
		"city":       city,
		"updated_at": time.Now().Unix(),
	}
	statusJSON, err := json.Marshal(statusData)
	if err != nil {
		r.logger.Warn("failed to marshal driver status for cache", zap.Error(err))
	} else {
		if err := r.redis.Set(ctx, statusKey, statusJSON, 24*time.Hour).Err(); err != nil {
			r.logger.Warn("failed to cache driver status", zap.String("driver_id", driverID), zap.Error(err))
		}
	}

	r.logger.Info("updated driver status", zap.String("driver_id", driverID), zap.String("status", status))
	return nil
}

// GetDashboardMetrics retrieves aggregated dashboard metrics for a city
func (r *MetricsRepository) GetDashboardMetrics(ctx context.Context, city string) (*DashboardMetrics, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("dashboard:metrics:%s", city)
	cachedData, err := r.redis.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var metrics DashboardMetrics
		if jsonErr := json.Unmarshal(cachedData, &metrics); jsonErr == nil {
			r.logger.Debug("dashboard metrics cache hit", zap.String("city", city))
			return &metrics, nil
		}
	}

	cityFilter := ""
	args := []interface{}{}
	argIdx := 1

	if city != "" {
		cityFilter = fmt.Sprintf("AND city = $%d", argIdx)
		args = append(args, city)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) FILTER (WHERE event_type = 'completed' AND occurred_at >= NOW() - INTERVAL '1 day') AS total_rides_today,
			COALESCE(SUM(fare) FILTER (WHERE event_type = 'completed' AND occurred_at >= NOW() - INTERVAL '1 day'), 0) AS total_revenue_today,
			COUNT(*) FILTER (WHERE event_type = 'cancelled' AND occurred_at >= NOW() - INTERVAL '1 day') AS cancelled_rides_today,
			COALESCE(AVG(fare) FILTER (WHERE event_type = 'completed'), 0) AS average_fare
		FROM ride_events
		WHERE 1=1 %s
	`, cityFilter)

	row := r.db.QueryRowContext(ctx, query, args...)

	var totalRidesToday int
	var totalRevenueToday float64
	var cancelledRidesToday int
	var averageFare float64

	if err := row.Scan(&totalRidesToday, &totalRevenueToday, &cancelledRidesToday, &averageFare); err != nil {
		r.logger.Error("failed to query dashboard metrics", zap.String("city", city), zap.Error(err))
		return nil, fmt.Errorf("failed to query dashboard metrics: %w", err)
	}

	// Get active driver count
	driverQuery := `SELECT COUNT(*) FROM driver_status WHERE status = 'active'`
	driverArgs := []interface{}{}
	if city != "" {
		driverQuery += " AND city = $1"
		driverArgs = append(driverArgs, city)
	}

	var activeDrivers int
	if err := r.db.QueryRowContext(ctx, driverQuery, driverArgs...).Scan(&activeDrivers); err != nil {
		r.logger.Warn("failed to get active driver count", zap.Error(err))
	}

	cancellationRate := 0.0
	totalRides := totalRidesToday + cancelledRidesToday
	if totalRides > 0 {
		cancellationRate = float64(cancelledRidesToday) / float64(totalRides) * 100
	}

	completionRate := 0.0
	if totalRides > 0 {
		completionRate = float64(totalRidesToday) / float64(totalRides) * 100
	}

	metrics := &DashboardMetrics{
		TotalRidesToday:   totalRidesToday,
		TotalRevenueToday: totalRevenueToday,
		ActiveDrivers:     activeDrivers,
		CancellationRate:  cancellationRate,
		CompletionRate:    completionRate,
	}

	// Cache the result for 5 minutes
	metricsJSON, err := json.Marshal(metrics)
	if err == nil {
		if err := r.redis.Set(ctx, cacheKey, metricsJSON, 5*time.Minute).Err(); err != nil {
			r.logger.Warn("failed to cache dashboard metrics", zap.String("city", city), zap.Error(err))
		}
	}

	return metrics, nil
}

// GetRideMetrics retrieves ride metrics for a given date range and city
func (r *MetricsRepository) GetRideMetrics(ctx context.Context, startDate, endDate time.Time, city string) (*RideMetrics, error) {
	args := []interface{}{startDate, endDate}
	cityFilter := ""
	argIdx := 3

	if city != "" {
		cityFilter = fmt.Sprintf("AND city = $%d", argIdx)
		args = append(args, city)
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total_rides,
			COUNT(*) FILTER (WHERE event_type = 'completed') AS completed_rides,
			COUNT(*) FILTER (WHERE event_type = 'cancelled') AS cancelled_rides,
			COALESCE(AVG(distance) FILTER (WHERE event_type = 'completed'), 0) AS avg_distance,
			COALESCE(AVG(duration) FILTER (WHERE event_type = 'completed'), 0) AS avg_duration,
			COALESCE(AVG(fare) FILTER (WHERE event_type = 'completed'), 0) AS avg_fare,
			COALESCE(SUM(distance) FILTER (WHERE event_type = 'completed'), 0) AS total_distance
		FROM ride_events
		WHERE occurred_at BETWEEN $1 AND $2 %s
	`, cityFilter)

	row := r.db.QueryRowContext(ctx, query, args...)

	var metrics RideMetrics
	if err := row.Scan(
		&metrics.TotalRides,
		&metrics.CompletedRides,
		&metrics.CancelledRides,
		&metrics.AverageDistance,
		&metrics.AverageDuration,
		&metrics.AverageFare,
		&metrics.TotalDistance,
	); err != nil {
		r.logger.Error("failed to get ride metrics", zap.Error(err))
		return nil, fmt.Errorf("failed to get ride metrics: %w", err)
	}

	if metrics.TotalRides > 0 {
		metrics.CancellationRate = float64(metrics.CancelledRides) / float64(metrics.TotalRides) * 100
	}

	return &metrics, nil
}

// GetRevenueMetrics retrieves revenue metrics grouped by period
func (r *MetricsRepository) GetRevenueMetrics(ctx context.Context, startDate, endDate time.Time, city, groupBy string) ([]RevenueMetrics, error) {
	var truncUnit string
	switch groupBy {
	case "hour":
		truncUnit = "hour"
	case "day":
		truncUnit = "day"
	case "week":
		truncUnit = "week"
	case "month":
		truncUnit = "month"
	default:
		truncUnit = "day"
	}

	args := []interface{}{startDate, endDate}
	cityFilter := ""
	argIdx := 3

	if city != "" {
		cityFilter = fmt.Sprintf("AND city = $%d", argIdx)
		args = append(args, city)
	}

	query := fmt.Sprintf(`
		SELECT
			DATE_TRUNC('%s', occurred_at) AS period,
			COALESCE(SUM(fare), 0) AS total_revenue,
			COALESCE(AVG(fare), 0) AS average_fare,
			COUNT(*) AS total_rides
		FROM ride_events
		WHERE event_type = 'completed'
			AND occurred_at BETWEEN $1 AND $2 %s
		GROUP BY DATE_TRUNC('%s', occurred_at)
		ORDER BY period ASC
	`, truncUnit, cityFilter, truncUnit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to get revenue metrics", zap.Error(err))
		return nil, fmt.Errorf("failed to get revenue metrics: %w", err)
	}
	defer rows.Close()

	var results []RevenueMetrics
	for rows.Next() {
		var rm RevenueMetrics
		var periodTime time.Time
		if err := rows.Scan(&periodTime, &rm.TotalRevenue, &rm.AverageFare, &rm.TotalRides); err != nil {
			r.logger.Error("failed to scan revenue metrics row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan revenue metrics: %w", err)
		}
		rm.Period = periodTime.Format(time.RFC3339)
		results = append(results, rm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating revenue metrics: %w", err)
	}

	// Calculate growth rates
	for i := 1; i < len(results); i++ {
		if results[i-1].TotalRevenue > 0 {
			results[i].GrowthRate = (results[i].TotalRevenue - results[i-1].TotalRevenue) / results[i-1].TotalRevenue * 100
		}
	}

	return results, nil
}

// GetDriverMetrics retrieves aggregated driver metrics for a city
func (r *MetricsRepository) GetDriverMetrics(ctx context.Context, city string) (*DriverMetrics, error) {
	args := []interface{}{}
	cityFilter := ""
	argIdx := 1

	if city != "" {
		cityFilter = fmt.Sprintf("WHERE city = $%d", argIdx)
		args = append(args, city)
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total_drivers,
			COUNT(*) FILTER (WHERE status = 'active') AS active_drivers
		FROM driver_status
		%s
	`, cityFilter)

	row := r.db.QueryRowContext(ctx, query, args...)
	var metrics DriverMetrics
	if err := row.Scan(&metrics.TotalDrivers, &metrics.ActiveDrivers); err != nil {
		r.logger.Error("failed to get driver metrics", zap.String("city", city), zap.Error(err))
		return nil, fmt.Errorf("failed to get driver metrics: %w", err)
	}

	// Get top performers
	topPerformers, err := r.getTopPerformingDrivers(ctx, city, 5)
	if err != nil {
		r.logger.Warn("failed to get top performers", zap.Error(err))
	} else {
		metrics.TopPerformers = topPerformers
	}

	return &metrics, nil
}

func (r *MetricsRepository) getTopPerformingDrivers(ctx context.Context, city string, limit int) ([]DriverPerformance, error) {
	args := []interface{}{limit}
	cityFilter := ""
	argIdx := 2

	if city != "" {
		cityFilter = fmt.Sprintf("AND city = $%d", argIdx)
		args = append(args, city)
		args[0] = limit
	}

	query := fmt.Sprintf(`
		SELECT
			driver_id,
			COUNT(*) AS total_rides,
			COALESCE(SUM(fare), 0) AS total_revenue,
			COALESCE(AVG(rating), 0) AS average_rating
		FROM ride_events
		WHERE event_type = 'completed'
			AND occurred_at >= NOW() - INTERVAL '30 days' %s
		GROUP BY driver_id
		ORDER BY total_rides DESC
		LIMIT $1
	`, cityFilter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top performing drivers: %w", err)
	}
	defer rows.Close()

	var performers []DriverPerformance
	for rows.Next() {
		var dp DriverPerformance
		if err := rows.Scan(&dp.DriverID, &dp.TotalRides, &dp.TotalRevenue, &dp.AverageRating); err != nil {
			return nil, fmt.Errorf("failed to scan driver performance: %w", err)
		}
		performers = append(performers, dp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating driver performance rows: %w", err)
	}

	return performers, nil
}

// GetRiderMetrics retrieves aggregated rider metrics for a city
func (r *MetricsRepository) GetRiderMetrics(ctx context.Context, city string) (*RiderMetrics, error) {
	args := []interface{}{}
	cityFilter := ""
	argIdx := 1

	if city != "" {
		cityFilter = fmt.Sprintf("AND city = $%d", argIdx)
		args = append(args, city)
		argIdx++
	}
	_ = argIdx

	query := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT rider_id) AS total_riders,
			COUNT(DISTINCT rider_id) FILTER (WHERE occurred_at >= NOW() - INTERVAL '30 days') AS active_riders,
			COUNT(DISTINCT rider_id) FILTER (WHERE occurred_at >= DATE_TRUNC('month', NOW())) AS new_riders_this_month
		FROM ride_events
		WHERE event_type = 'completed' %s
	`, cityFilter)

	row := r.db.QueryRowContext(ctx, query, args...)
	var metrics RiderMetrics
	if err := row.Scan(&metrics.TotalRiders, &metrics.ActiveRiders, &metrics.NewRidersThisMonth); err != nil {
		r.logger.Error("failed to get rider metrics", zap.String("city", city), zap.Error(err))
		return nil, fmt.Errorf("failed to get rider metrics: %w", err)
	}

	if metrics.TotalRiders > 0 {
		metrics.RetentionRate = float64(metrics.ActiveRiders) / float64(metrics.TotalRiders) * 100
	}

	return &metrics, nil
}

// GetHeatmapData retrieves geographic heatmap data for a city at a specific hour
func (r *MetricsRepository) GetHeatmapData(ctx context.Context, city string, hour int) ([]HeatmapData, error) {
	cacheKey := fmt.Sprintf("heatmap:%s:%d", city, hour)
	cachedData, err := r.redis.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var heatmap []HeatmapData
		if jsonErr := json.Unmarshal(cachedData, &heatmap); jsonErr == nil {
			return heatmap, nil
		}
	}

	args := []interface{}{hour}
	cityFilter := ""
	argIdx := 2

	if city != "" {
		cityFilter = fmt.Sprintf("AND city = $%d", argIdx)
		args = append(args, city)
	}

	query := fmt.Sprintf(`
		SELECT
			ROUND(start_lat::numeric, 3) AS lat,
			ROUND(start_lng::numeric, 3) AS lng,
			COUNT(*) AS intensity
		FROM ride_events
		WHERE EXTRACT(HOUR FROM occurred_at) = $1
			AND event_type = 'completed' %s
		GROUP BY ROUND(start_lat::numeric, 3), ROUND(start_lng::numeric, 3)
		ORDER BY intensity DESC
		LIMIT 1000
	`, cityFilter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to get heatmap data", zap.String("city", city), zap.Int("hour", hour), zap.Error(err))
		return nil, fmt.Errorf("failed to get heatmap data: %w", err)
	}
	defer rows.Close()

	var heatmap []HeatmapData
	for rows.Next() {
		var hd HeatmapData
		if err := rows.Scan(&hd.Lat, &hd.Lng, &hd.Intensity); err != nil {
			r.logger.Error("failed to scan heatmap row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan heatmap data: %w", err)
		}
		heatmap = append(heatmap, hd)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating heatmap rows: %w", err)
	}

	// Cache for 1 hour
	if heatmapJSON, err := json.Marshal(heatmap); err == nil {
		if err := r.redis.Set(ctx, cacheKey, heatmapJSON, time.Hour).Err(); err != nil {
			r.logger.Warn("failed to cache heatmap data", zap.Error(err))
		}
	}

	return heatmap, nil
}

// GetTrends retrieves trend data for a given metric over a period
func (r *MetricsRepository) GetTrends(ctx context.Context, metric, period, city string) ([]TrendData, error) {
	var interval, truncUnit string
	switch period {
	case "7d":
		interval = "7 days"
		truncUnit = "day"
	case "30d":
		interval = "30 days"
		truncUnit = "day"
	case "90d":
		interval = "90 days"
		truncUnit = "week"
	case "1y":
		interval = "1 year"
		truncUnit = "month"
	default:
		interval = "30 days"
		truncUnit = "day"
	}

	var valueExpr string
	switch metric {
	case "rides":
		valueExpr = "COUNT(*)"
	case "revenue":
		valueExpr = "COALESCE(SUM(fare), 0)"
	case "avg_fare":
		valueExpr = "COALESCE(AVG(fare), 0)"
	case "distance":
		valueExpr = "COALESCE(SUM(distance), 0)"
	default:
		valueExpr = "COUNT(*)"
	}

	args := []interface{}{interval}
	cityFilter := ""
	argIdx := 2

	if city != "" {
		cityFilter = fmt.Sprintf("AND city = $%d", argIdx)
		args = append(args, city)
	}

	query := fmt.Sprintf(`
		SELECT
			DATE_TRUNC('%s', occurred_at) AS ts,
			%s AS value
		FROM ride_events
		WHERE event_type = 'completed'
			AND occurred_at >= NOW() - $1::interval %s
		GROUP BY DATE_TRUNC('%s', occurred_at)
		ORDER BY ts ASC
	`, truncUnit, valueExpr, cityFilter, truncUnit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to get trends", zap.String("metric", metric), zap.String("period", period), zap.Error(err))
		return nil, fmt.Errorf("failed to get trends: %w", err)
	}
	defer rows.Close()

	var trends []TrendData
	for rows.Next() {
		var td TrendData
		if err := rows.Scan(&td.Timestamp, &td.Value); err != nil {
			r.logger.Error("failed to scan trend row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan trend data: %w", err)
		}
		td.Label = td.Timestamp.Format("2006-01-02")
		trends = append(trends, td)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trend rows: %w", err)
	}

	return trends, nil
}

// GetPeakHours retrieves peak hour data for a city on a specific day of week
func (r *MetricsRepository) GetPeakHours(ctx context.Context, city string, dayOfWeek int) ([]PeakHourData, error) {
	cacheKey := fmt.Sprintf("peak_hours:%s:%d", city, dayOfWeek)
	cachedData, err := r.redis.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var peaks []PeakHourData
		if jsonErr := json.Unmarshal(cachedData, &peaks); jsonErr == nil {
			return peaks, nil
		}
	}

	args := []interface{}{}
	filters := ""
	argIdx := 1

	if dayOfWeek >= 0 && dayOfWeek <= 6 {
		filters = fmt.Sprintf("AND EXTRACT(DOW FROM occurred_at) = $%d", argIdx)
		args = append(args, dayOfWeek)
		argIdx++
	}

	if city != "" {
		filters += fmt.Sprintf(" AND city = $%d", argIdx)
		args = append(args, city)
	}

	query := fmt.Sprintf(`
		SELECT
			EXTRACT(HOUR FROM occurred_at)::int AS hour,
			COUNT(*) AS ride_count
		FROM ride_events
		WHERE event_type = 'completed'
			AND occurred_at >= NOW() - INTERVAL '90 days' %s
		GROUP BY EXTRACT(HOUR FROM occurred_at)
		ORDER BY hour ASC
	`, filters)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to get peak hours", zap.String("city", city), zap.Error(err))
		return nil, fmt.Errorf("failed to get peak hours: %w", err)
	}
	defer rows.Close()

	var peaks []PeakHourData
	maxCount := 0
	for rows.Next() {
		var ph PeakHourData
		if err := rows.Scan(&ph.Hour, &ph.RideCount); err != nil {
			return nil, fmt.Errorf("failed to scan peak hour row: %w", err)
		}
		if ph.RideCount > maxCount {
			maxCount = ph.RideCount
		}
		peaks = append(peaks, ph)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating peak hour rows: %w", err)
	}

	// Compute demand score
	for i := range peaks {
		if maxCount > 0 {
			peaks[i].DemandScore = float64(peaks[i].RideCount) / float64(maxCount) * 100
		}
	}

	// Cache for 6 hours
	if peaksJSON, err := json.Marshal(peaks); err == nil {
		if err := r.redis.Set(ctx, cacheKey, peaksJSON, 6*time.Hour).Err(); err != nil {
			r.logger.Warn("failed to cache peak hours", zap.Error(err))
		}
	}

	return peaks, nil
}

// GetPopularRoutes retrieves the most popular routes in a city
func (r *MetricsRepository) GetPopularRoutes(ctx context.Context, city string, limit int) ([]PopularRoute, error) {
	if limit <= 0 {
		limit = 10
	}

	cacheKey := fmt.Sprintf("popular_routes:%s:%d", city, limit)
	cachedData, err := r.redis.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var routes []PopularRoute
		if jsonErr := json.Unmarshal(cachedData, &routes); jsonErr == nil {
			return routes, nil
		}
	}

	args := []interface{}{limit}
	cityFilter := ""
	argIdx := 2

	if city != "" {
		cityFilter = fmt.Sprintf("AND city = $%d", argIdx)
		args = append(args, city)
	}

	query := fmt.Sprintf(`
		SELECT
			start_zone,
			end_zone,
			COUNT(*) AS ride_count,
			COALESCE(AVG(fare), 0) AS average_fare
		FROM ride_events
		WHERE event_type = 'completed'
			AND occurred_at >= NOW() - INTERVAL '30 days'
			AND start_zone IS NOT NULL
			AND end_zone IS NOT NULL %s
		GROUP BY start_zone, end_zone
		ORDER BY ride_count DESC
		LIMIT $1
	`, cityFilter)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to get popular routes", zap.String("city", city), zap.Error(err))
		return nil, fmt.Errorf("failed to get popular routes: %w", err)
	}
	defer rows.Close()

	var routes []PopularRoute
	for rows.Next() {
		var pr PopularRoute
		if err := rows.Scan(&pr.StartZone, &pr.EndZone, &pr.RideCount, &pr.AverageFare); err != nil {
			r.logger.Error("failed to scan popular route row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan popular route: %w", err)
		}
		routes = append(routes, pr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating popular route rows: %w", err)
	}

	// Cache for 1 hour
	if routesJSON, err := json.Marshal(routes); err == nil {
		if err := r.redis.Set(ctx, cacheKey, routesJSON, time.Hour).Err(); err != nil {
			r.logger.Warn("failed to cache popular routes", zap.Error(err))
		}
	}

	return routes, nil
}

// GetDriverPerformance retrieves performance metrics for a specific driver
func (r *MetricsRepository) GetDriverPerformance(ctx context.Context, driverID string, startDate, endDate time.Time) (*DriverPerformance, error) {
	query := `
		SELECT
			driver_id,
			COUNT(*) FILTER (WHERE event_type = 'completed') AS total_rides,
			COALESCE(SUM(fare) FILTER (WHERE event_type = 'completed'), 0) AS total_revenue,
			COALESCE(AVG(rating) FILTER (WHERE event_type = 'completed'), 0) AS average_rating,
			CASE
				WHEN COUNT(*) > 0
				THEN COUNT(*) FILTER (WHERE event_type = 'cancelled')::float / COUNT(*)::float * 100
				ELSE 0
			END AS cancellation_rate
		FROM ride_events
		WHERE driver_id = $1
			AND occurred_at BETWEEN $2 AND $3
		GROUP BY driver_id
	`

	row := r.db.QueryRowContext(ctx, query, driverID, startDate, endDate)
	var dp DriverPerformance
	if err := row.Scan(
		&dp.DriverID,
		&dp.TotalRides,
		&dp.TotalRevenue,
		&dp.AverageRating,
		&dp.CancellationRate,
	); err != nil {
		if err == sql.ErrNoRows {
			return &DriverPerformance{DriverID: driverID}, nil
		}
		r.logger.Error("failed to get driver performance", zap.String("driver_id", driverID), zap.Error(err))
		return nil, fmt.Errorf("failed to get driver performance: %w", err)
	}

	return &dp, nil
}

// GetRiderBehavior retrieves behavioral analytics for a specific rider
func (r *MetricsRepository) GetRiderBehavior(ctx context.Context, riderID string, startDate, endDate time.Time) (*RiderBehavior, error) {
	query := `
		SELECT
			rider_id,
			COUNT(*) FILTER (WHERE event_type = 'completed') AS total_rides,
			COALESCE(SUM(fare) FILTER (WHERE event_type = 'completed'), 0) AS total_spent,
			COALESCE(AVG(fare) FILTER (WHERE event_type = 'completed'), 0) AS average_ride_fare,
			CASE
				WHEN COUNT(*) > 0
				THEN COUNT(*) FILTER (WHERE event_type = 'cancelled')::float / COUNT(*)::float * 100
				ELSE 0
			END AS cancellation_rate,
			MIN(occurred_at) AS first_ride_at,
			MAX(occurred_at) FILTER (WHERE event_type = 'completed') AS last_ride_at
		FROM ride_events
		WHERE rider_id = $1
			AND occurred_at BETWEEN $2 AND $3
		GROUP BY rider_id
	`

	row := r.db.QueryRowContext(ctx, query, riderID, startDate, endDate)
	var rb RiderBehavior
	var lastRideAt sql.NullTime

	if err := row.Scan(
		&rb.RiderID,
		&rb.TotalRides,
		&rb.TotalSpent,
		&rb.AverageRideFare,
		&rb.CancellationRate,
		&rb.FirstRideAt,
		&lastRideAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return &RiderBehavior{RiderID: riderID}, nil
		}
		r.logger.Error("failed to get rider behavior", zap.String("rider_id", riderID), zap.Error(err))
		return nil, fmt.Errorf("failed to get rider behavior: %w", err)
	}

	if lastRideAt.Valid {
		rb.LastRideAt = lastRideAt.Time
	}

	// Get preferred hours
	preferredHours, err := r.getRiderPreferredHours(ctx, riderID, startDate, endDate)
	if err != nil {
		r.logger.Warn("failed to get rider preferred hours", zap.String("rider_id", riderID), zap.Error(err))
	} else {
		rb.PreferredHours = preferredHours
	}

	// Get favorite city
	favCity, err := r.getRiderFavoriteCity(ctx, riderID, startDate, endDate)
	if err != nil {
		r.logger.Warn("failed to get rider favorite city", zap.String("rider_id", riderID), zap.Error(err))
	} else {
		rb.FavoriteCity = favCity
	}

	return &rb, nil
}

func (r *MetricsRepository) getRiderPreferredHours(ctx context.Context, riderID string, startDate, endDate time.Time) ([]int, error) {
	query := `
		SELECT EXTRACT(HOUR FROM occurred_at)::int AS hour
		FROM ride_events
		WHERE rider_id = $1
			AND event_type = 'completed'
			AND occurred_at BETWEEN $2 AND $3
		GROUP BY hour
		ORDER BY COUNT(*) DESC
		LIMIT 3
	`

	rows, err := r.db.QueryContext(ctx, query, riderID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query preferred hours: %w", err)
	}
	defer rows.Close()

	var hours []int
	for rows.Next() {
		var h int
		if err := rows.Scan(&h); err != nil {
			return nil, fmt.Errorf("failed to scan preferred hour: %w", err)
		}
		hours = append(hours, h)
	}

	return hours, rows.Err()
}

func (r *MetricsRepository) getRiderFavoriteCity(ctx context.Context, riderID string, startDate, endDate time.Time) (string, error) {
	query := `
		SELECT city
		FROM ride_events
		WHERE rider_id = $1
			AND event_type = 'completed'
			AND occurred_at BETWEEN $2 AND $3
		GROUP BY city
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`

	var city string
	if err := r.db.QueryRowContext(ctx, query, riderID, startDate, endDate).Scan(&city); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to query favorite city: %w", err)
	}

	return city, nil
}

// AggregateHourlyMetrics aggregates raw events into hourly summary metrics
func (r *MetricsRepository) AggregateHourlyMetrics(ctx context.Context) error {
	query := `
		INSERT INTO hourly_metrics (
			hour_bucket, city,
			total_rides, completed_rides, cancelled_rides,
			total_revenue, avg_fare, avg_distance, avg_duration
		)
		SELECT
			DATE_TRUNC('hour', occurred_at) AS hour_bucket,
			city,
			COUNT(*) AS total_rides,
			COUNT(*) FILTER (WHERE event_type = 'completed') AS completed_rides,
			COUNT(*) FILTER (WHERE event_type = 'cancelled') AS cancelled_rides,
			COALESCE(SUM(fare) FILTER (WHERE event_type = 'completed'), 0) AS total_revenue,
			COALESCE(AVG(fare) FILTER (WHERE event_type = 'completed'), 0) AS avg_fare,
			COALESCE(AVG(distance) FILTER (WHERE event_type = 'completed'), 0) AS avg_distance,
			COALESCE(AVG(duration) FILTER (WHERE event_type = 'completed'), 0) AS avg_duration
		FROM ride_events
		WHERE occurred_at >= DATE_TRUNC('hour', NOW() - INTERVAL '2 hours')
			AND occurred_at < DATE_TRUNC('hour', NOW())
		GROUP BY DATE_TRUNC('hour', occurred_at), city
		ON CONFLICT (hour_bucket, city)
		DO UPDATE SET
			total_rides = EXCLUDED.total_rides,
			completed_rides = EXCLUDED.completed_rides,
			cancelled_rides = EXCLUDED.cancelled_rides,
			total_revenue = EXCLUDED.total_revenue,
			avg_fare = EXCLUDED.avg_fare,
			avg_distance = EXCLUDED.avg_distance,
			avg_duration = EXCLUDED.avg_duration,
			updated_at = NOW()
	`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		r.logger.Error("failed to aggregate hourly metrics", zap.Error(err))
		return fmt.Errorf("failed to aggregate hourly metrics: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.Info("aggregated hourly metrics", zap.Int64("rows_affected", rowsAffected))
	return nil
}

// AggregateDailyMetrics aggregates hourly metrics into daily summary metrics
func (r *MetricsRepository) AggregateDailyMetrics(ctx context.Context) error {
	query := `
		INSERT INTO daily_metrics (
			day_bucket, city,
			total_rides, completed_rides, cancelled_rides,
			total_revenue, avg_fare, avg_distance, avg_duration,
			active_drivers, active_riders
		)
		SELECT
			DATE_TRUNC('day', hour_bucket) AS day_bucket,
			city,
			SUM(total_rides) AS total_rides,
			SUM(completed_rides) AS completed_rides,
			SUM(cancelled_rides) AS cancelled_rides,
			SUM(total_revenue) AS total_revenue,
			AVG(avg_fare) AS avg_fare,
			AVG(avg_distance) AS avg_distance,
			AVG(avg_duration) AS avg_duration,
			0 AS active_drivers,
			0 AS active_riders
		FROM hourly_metrics
		WHERE hour_bucket >= DATE_TRUNC('day', NOW() - INTERVAL '2 days')
			AND hour_bucket < DATE_TRUNC('day', NOW())
		GROUP BY DATE_TRUNC('day', hour_bucket), city
		ON CONFLICT (day_bucket, city)
		DO UPDATE SET
			total_rides = EXCLUDED.total_rides,
			completed_rides = EXCLUDED.completed_rides,
			cancelled_rides = EXCLUDED.cancelled_rides,
			total_revenue = EXCLUDED.total_revenue,
			avg_fare = EXCLUDED.avg_fare,
			avg_distance = EXCLUDED.avg_distance,
			avg_duration = EXCLUDED.avg_duration,
			updated_at = NOW()
	`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		r.logger.Error("failed to aggregate daily metrics", zap.Error(err))
		return fmt.Errorf("failed to aggregate daily metrics: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.Info("aggregated daily metrics", zap.Int64("rows_affected", rowsAffected))
	return nil
}

// ReportRepository handles report storage and retrieval
type ReportRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewReportRepository creates a new ReportRepository
func NewReportRepository(db *sql.DB, logger *zap.Logger) *ReportRepository {
	return &ReportRepository{
		db:     db,
		logger: logger,
	}
}

// GetReport retrieves a report by its ID
func (r *ReportRepository) GetReport(ctx context.Context, reportID string) (*Report, error) {
	query := `
		SELECT
			id, type, title, city,
			start_date, end_date,
			parameters, data,
			created_at, created_by, status
		FROM reports
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, reportID)

	var report Report
	var parametersJSON, dataJSON []byte

	if err := row.Scan(
		&report.ID,
		&report.Type,
		&report.Title,
		&report.City,
		&report.StartDate,
		&report.EndDate,
		&parametersJSON,
		&dataJSON,
		&report.CreatedAt,
		&report.CreatedBy,
		&report.Status,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("report not found: %s", reportID)
		}
		r.logger.Error("failed to get report", zap.String("report_id", reportID), zap.Error(err))
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	if len(parametersJSON) > 0 {
		if err := json.Unmarshal(parametersJSON, &report.Parameters); err != nil {
			r.logger.Warn("failed to unmarshal report parameters", zap.String("report_id", reportID), zap.Error(err))
		}
	}

	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &report.Data); err != nil {
			r.logger.Warn("failed to unmarshal report data", zap.String("report_id", reportID), zap.Error(err))
		}
	}

	return &report, nil
}

// SaveReport saves or updates a report in the database
func (r *ReportRepository) SaveReport(ctx context.Context, report *Report) error {
	parametersJSON, err := json.Marshal(report.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal report parameters: %w", err)
	}

	dataJSON, err := json.Marshal(report.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal report data: %w", err)
	}

	query := `
		INSERT INTO reports (
			id, type, title, city,
			start_date, end_date,
			parameters, data,
			created_at, created_by, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id)
		DO UPDATE SET
			type = EXCLUDED.type,
			title = EXCLUDED.title,
			city = EXCLUDED.city,
			start_date = EXCLUDED.start_date,
			end_date = EXCLUDED.end_date,
			parameters = EXCLUDED.parameters,
			data = EXCLUDED.data,
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	_, err = r.db.ExecContext(ctx, query,
		report.ID,
		report.Type,
		report.Title,
		report.City,
		report.StartDate,
		report.EndDate,
		parametersJSON,
		dataJSON,
		report.CreatedAt,
		report.CreatedBy,
		report.Status,
	)
	if err != nil {
		r.logger.Error("failed to save report", zap.String("report_id", report.ID), zap.Error(err))
		return fmt.Errorf("failed to save report: %w", err)
	}

	r.logger.Info("saved report", zap.String("report_id", report.ID), zap.String("type", report.Type))
	return nil
}

// ListReports retrieves a list of reports matching the given filters
func (r *ReportRepository) ListReports(ctx context.Context, filters ReportFilters) ([]Report, error) {
	args := []interface{}{}
	conditions := []string{"1=1"}
	argIdx := 1

	if filters.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, filters.Type)
		argIdx++
	}

	if filters.City != "" {
		conditions = append(conditions, fmt.Sprintf("city = $%d", argIdx))
		args = append(args, filters.City)
		argIdx++
	}

	if filters.CreatedBy != "" {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argIdx))
		args = append(args, filters.CreatedBy)
		argIdx++
	}

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}

	if filters.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filters.StartDate)
		argIdx++
	}

	if filters.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filters.EndDate)
		argIdx++
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}

	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	wherePart := ""
	for i, cond := range conditions {
		if i == 0 {
			wherePart = "WHERE " + cond
		} else {
			wherePart += " AND " + cond
		}
	}

	query := fmt.Sprintf(`
		SELECT
			id, type, title, city,
			start_date, end_date,
			parameters, data,
			created_at, created_by, status
		FROM reports
		%s
		ORDER BY created_at DESC
		LIMIT %d OFFSET %d
	`, wherePart, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to list reports", zap.Error(err))
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var report Report
		var parametersJSON, dataJSON []byte

		if err := rows.Scan(
			&report.ID,
			&report.Type,
			&report.Title,
			&report.City,
			&report.StartDate,
			&report.EndDate,
			&parametersJSON,
			&dataJSON,
			&report.CreatedAt,
			&report.CreatedBy,
			&report.Status,
		); err != nil {
			r.logger.Error("failed to scan report row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan report: %w", err)
		}

		if len(parametersJSON) > 0 {
			if err := json.Unmarshal(parametersJSON, &report.Parameters); err != nil {
				r.logger.Warn("failed to unmarshal report parameters", zap.String("report_id", report.ID), zap.Error(err))
			}
		}

		if len(dataJSON) > 0 {
			if err := json.Unmarshal(dataJSON, &report.Data); err != nil {
				r.logger.Warn("failed to unmarshal report data", zap.String("report_id", report.ID), zap.Error(err))
			}
		}

		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating report rows: %w", err)
	}

	return reports, nil
}
