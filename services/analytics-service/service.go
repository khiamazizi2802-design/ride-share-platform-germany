package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// MetricsRepository defines the interface for metrics data access
type MetricsRepository interface {
	GetDashboardOverview(ctx context.Context, city string) (*DashboardMetrics, error)
	GetRideStats(ctx context.Context, startDate, endDate time.Time, city string) (*RideMetrics, error)
	GetRevenueStats(ctx context.Context, startDate, endDate time.Time, city, groupBy string) (*RevenueMetrics, error)
	GetDriverStats(ctx context.Context, city string) (*DriverMetrics, error)
	GetRiderStats(ctx context.Context, city string) (*RiderMetrics, error)
	GetHeatmap(ctx context.Context, city string, hour int) ([]*HeatmapPoint, error)
	GetTrendData(ctx context.Context, metric, period, city string) (*TrendData, error)
	GetPeakHourData(ctx context.Context, city string, dayOfWeek int) ([]*PeakHourData, error)
	GetTopRoutes(ctx context.Context, city string, limit int) ([]*RouteData, error)
	GetDriverPerformanceData(ctx context.Context, driverID string, startDate, endDate time.Time) (*DriverPerformance, error)
	GetRiderBehaviorData(ctx context.Context, riderID string, startDate, endDate time.Time) (*RiderBehavior, error)
	AggregateHourly(ctx context.Context, hour time.Time) error
	AggregateDaily(ctx context.Context, day time.Time) error
}

// ReportRepository defines the interface for report data access
type ReportRepository interface {
	SaveReport(ctx context.Context, report *Report) error
	GetReport(ctx context.Context, reportID string) (*Report, error)
	GetReportsByType(ctx context.Context, reportType string) ([]*Report, error)
	UpdateReportStatus(ctx context.Context, reportID, status string) error
	GetExportData(ctx context.Context, request *ExportRequest) ([]map[string]interface{}, error)
}

// DashboardMetrics represents an overview of dashboard statistics
type DashboardMetrics struct {
	TotalRidesToday       int64     `json:"total_rides_today"`
	TotalRevenueToday     float64   `json:"total_revenue_today"`
	ActiveDrivers         int64     `json:"active_drivers"`
	ActiveRiders          int64     `json:"active_riders"`
	AverageRating         float64   `json:"average_rating"`
	CompletionRate        float64   `json:"completion_rate"`
	AverageWaitTime       float64   `json:"average_wait_time_minutes"`
	CancellationRate      float64   `json:"cancellation_rate"`
	City                  string    `json:"city"`
	GeneratedAt           time.Time `json:"generated_at"`
}

// RideMetrics represents ride statistics over a period
type RideMetrics struct {
	TotalRides          int64     `json:"total_rides"`
	CompletedRides      int64     `json:"completed_rides"`
	CancelledRides      int64     `json:"cancelled_rides"`
	AverageDuration     float64   `json:"average_duration_minutes"`
	AverageDistance     float64   `json:"average_distance_km"`
	PeakHour            int       `json:"peak_hour"`
	CompletionRate      float64   `json:"completion_rate"`
	City                string    `json:"city"`
	StartDate           time.Time `json:"start_date"`
	EndDate             time.Time `json:"end_date"`
}

// RevenueMetrics represents revenue data over a period
type RevenueMetrics struct {
	TotalRevenue        float64            `json:"total_revenue"`
	AveragePerRide      float64            `json:"average_per_ride"`
	GroupedRevenue      map[string]float64 `json:"grouped_revenue"`
	GrowthRate          float64            `json:"growth_rate_percent"`
	City                string             `json:"city"`
	GroupBy             string             `json:"group_by"`
	StartDate           time.Time          `json:"start_date"`
	EndDate             time.Time          `json:"end_date"`
}

// DriverMetrics represents driver statistics
type DriverMetrics struct {
	TotalDrivers        int64   `json:"total_drivers"`
	ActiveDrivers       int64   `json:"active_drivers"`
	NewDriversThisMonth int64   `json:"new_drivers_this_month"`
	AverageRating       float64 `json:"average_rating"`
	AverageRidesPerDay  float64 `json:"average_rides_per_day"`
	AverageEarnings     float64 `json:"average_earnings"`
	RetentionRate       float64 `json:"retention_rate_percent"`
	City                string  `json:"city"`
}

// RiderMetrics represents rider statistics
type RiderMetrics struct {
	TotalRiders         int64   `json:"total_riders"`
	ActiveRiders        int64   `json:"active_riders"`
	NewRidersThisMonth  int64   `json:"new_riders_this_month"`
	AverageRidesPerWeek float64 `json:"average_rides_per_week"`
	AverageSpend        float64 `json:"average_spend"`
	ChurnRate           float64 `json:"churn_rate_percent"`
	City                string  `json:"city"`
}

// HeatmapPoint represents a geographic point with intensity
type HeatmapPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Intensity float64 `json:"intensity"`
	Count     int64   `json:"count"`
}

// TrendData represents trend analysis over time
type TrendData struct {
	Metric      string           `json:"metric"`
	Period      string           `json:"period"`
	City        string           `json:"city"`
	DataPoints  []*TrendPoint    `json:"data_points"`
	TrendSlope  float64          `json:"trend_slope"`
	GrowthRate  float64          `json:"growth_rate_percent"`
}

// TrendPoint represents a single data point in a trend
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label"`
}

// PeakHourData represents activity data for a specific hour
type PeakHourData struct {
	Hour        int     `json:"hour"`
	RideCount   int64   `json:"ride_count"`
	Revenue     float64 `json:"revenue"`
	IsPeak      bool    `json:"is_peak"`
	DayOfWeek   int     `json:"day_of_week"`
}

// RouteData represents a popular route
type RouteData struct {
	Origin          string  `json:"origin"`
	Destination     string  `json:"destination"`
	TripCount       int64   `json:"trip_count"`
	Averagefare     float64 `json:"average_fare"`
	AverageDuration float64 `json:"average_duration_minutes"`
	Rank            int     `json:"rank"`
}

// DriverPerformance represents individual driver performance data
type DriverPerformance struct {
	DriverID            string    `json:"driver_id"`
	TotalRides          int64     `json:"total_rides"`
	CompletedRides      int64     `json:"completed_rides"`
	CancelledRides      int64     `json:"cancelled_rides"`
	TotalEarnings       float64   `json:"total_earnings"`
	AverageRating       float64   `json:"average_rating"`
	OnlineHours         float64   `json:"online_hours"`
	UtilizationRate     float64   `json:"utilization_rate_percent"`
	AverageResponseTime float64   `json:"average_response_time_seconds"`
	StartDate           time.Time `json:"start_date"`
	EndDate             time.Time `json:"end_date"`
}

// RiderBehavior represents individual rider behavior data
type RiderBehavior struct {
	RiderID             string    `json:"rider_id"`
	TotalRides          int64     `json:"total_rides"`
	TotalSpend          float64   `json:"total_spend"`
	AverageRideDistance float64   `json:"average_ride_distance_km"`
	FavoritePickupArea  string    `json:"favorite_pickup_area"`
	MostActiveHour      int       `json:"most_active_hour"`
	CancellationRate    float64   `json:"cancellation_rate_percent"`
	LoyaltyScore        float64   `json:"loyalty_score"`
	StartDate           time.Time `json:"start_date"`
	EndDate             time.Time `json:"end_date"`
}

// Report represents a generated report
type Report struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Data        map[string]interface{} `json:"data"`
	GeneratedAt time.Time              `json:"generated_at"`
	PeriodStart time.Time              `json:"period_start"`
	PeriodEnd   time.Time              `json:"period_end"`
	City        string                 `json:"city,omitempty"`
	Format      string                 `json:"format,omitempty"`
}

// CustomReportRequest represents a request for a custom report
type CustomReportRequest struct {
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	City       string    `json:"city"`
	Metrics    []string  `json:"metrics"`
	GroupBy    string    `json:"group_by"`
	Format     string    `json:"format"`
}

// ExportRequest represents a GDPR-compliant data export request
type ExportRequest struct {
	UserID    string    `json:"user_id"`
	UserType  string    `json:"user_type"` // "driver" or "rider"
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Format    string    `json:"format"` // "json" or "csv"
}

// ExportResult represents the result of a data export
type ExportResult struct {
	UserID      string    `json:"user_id"`
	Format      string    `json:"format"`
	Data        []byte    `json:"data"`
	GeneratedAt time.Time `json:"generated_at"`
	RecordCount int       `json:"record_count"`
}

// AnalyticsService handles analytics business logic
type AnalyticsService struct {
	metricsRepo MetricsRepository
	logger      *zap.Logger
}

// NewAnalyticsService creates a new AnalyticsService
func NewAnalyticsService(metricsRepo MetricsRepository, logger *zap.Logger) *AnalyticsService {
	return &AnalyticsService{
		metricsRepo: metricsRepo,
		logger:      logger,
	}
}

// GetDashboardMetrics returns an overview of dashboard statistics for a city
func (s *AnalyticsService) GetDashboardMetrics(ctx context.Context, city string) (*DashboardMetrics, error) {
	s.logger.Info("fetching dashboard metrics", zap.String("city", city))

	metrics, err := s.metricsRepo.GetDashboardOverview(ctx, city)
	if err != nil {
		s.logger.Error("failed to fetch dashboard metrics",
			zap.String("city", city),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch dashboard metrics for city %s: %w", city, err)
	}

	metrics.GeneratedAt = time.Now().UTC()
	return metrics, nil
}

// GetRideMetrics returns ride statistics for the given date range and city
func (s *AnalyticsService) GetRideMetrics(ctx context.Context, startDate, endDate time.Time, city string) (*RideMetrics, error) {
	s.logger.Info("fetching ride metrics",
		zap.String("city", city),
		zap.Time("start_date", startDate),
		zap.Time("end_date", endDate),
	)

	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end_date must be after start_date")
	}

	metrics, err := s.metricsRepo.GetRideStats(ctx, startDate, endDate, city)
	if err != nil {
		s.logger.Error("failed to fetch ride metrics",
			zap.String("city", city),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch ride metrics: %w", err)
	}

	if metrics.TotalRides > 0 {
		metrics.CompletionRate = float64(metrics.CompletedRides) / float64(metrics.TotalRides) * 100
	}

	return metrics, nil
}

// GetRevenueMetrics returns revenue data for the given date range, city, and grouping
func (s *AnalyticsService) GetRevenueMetrics(ctx context.Context, startDate, endDate time.Time, city, groupBy string) (*RevenueMetrics, error) {
	s.logger.Info("fetching revenue metrics",
		zap.String("city", city),
		zap.String("group_by", groupBy),
		zap.Time("start_date", startDate),
		zap.Time("end_date", endDate),
	)

	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end_date must be after start_date")
	}

	validGroupBy := map[string]bool{"hour": true, "day": true, "week": true, "month": true}
	if groupBy != "" && !validGroupBy[groupBy] {
		return nil, fmt.Errorf("invalid group_by value: %s, must be one of hour, day, week, month", groupBy)
	}
	if groupBy == "" {
		groupBy = "day"
	}

	metrics, err := s.metricsRepo.GetRevenueStats(ctx, startDate, endDate, city, groupBy)
	if err != nil {
		s.logger.Error("failed to fetch revenue metrics",
			zap.String("city", city),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch revenue metrics: %w", err)
	}

	return metrics, nil
}

// GetDriverMetrics returns driver statistics for the given city
func (s *AnalyticsService) GetDriverMetrics(ctx context.Context, city string) (*DriverMetrics, error) {
	s.logger.Info("fetching driver metrics", zap.String("city", city))

	metrics, err := s.metricsRepo.GetDriverStats(ctx, city)
	if err != nil {
		s.logger.Error("failed to fetch driver metrics",
			zap.String("city", city),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch driver metrics for city %s: %w", city, err)
	}

	return metrics, nil
}

// GetRiderMetrics returns rider statistics for the given city
func (s *AnalyticsService) GetRiderMetrics(ctx context.Context, city string) (*RiderMetrics, error) {
	s.logger.Info("fetching rider metrics", zap.String("city", city))

	metrics, err := s.metricsRepo.GetRiderStats(ctx, city)
	if err != nil {
		s.logger.Error("failed to fetch rider metrics",
			zap.String("city", city),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch rider metrics for city %s: %w", city, err)
	}

	return metrics, nil
}

// GetHeatmapData returns geographic heatmap data for the given city and hour
func (s *AnalyticsService) GetHeatmapData(ctx context.Context, city string, hour int) ([]*HeatmapPoint, error) {
	s.logger.Info("fetching heatmap data",
		zap.String("city", city),
		zap.Int("hour", hour),
	)

	if hour < 0 || hour > 23 {
		return nil, fmt.Errorf("hour must be between 0 and 23, got %d", hour)
	}

	points, err := s.metricsRepo.GetHeatmap(ctx, city, hour)
	if err != nil {
		s.logger.Error("failed to fetch heatmap data",
			zap.String("city", city),
			zap.Int("hour", hour),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch heatmap data: %w", err)
	}

	return points, nil
}

// GetTrends returns trend analysis for the given metric, period, and city
func (s *AnalyticsService) GetTrends(ctx context.Context, metric, period, city string) (*TrendData, error) {
	s.logger.Info("fetching trend data",
		zap.String("metric", metric),
		zap.String("period", period),
		zap.String("city", city),
	)

	validMetrics := map[string]bool{"rides": true, "revenue": true, "drivers": true, "riders": true, "rating": true}
	if !validMetrics[metric] {
		return nil, fmt.Errorf("invalid metric: %s", metric)
	}

	validPeriods := map[string]bool{"daily": true, "weekly": true, "monthly": true, "yearly": true}
	if !validPeriods[period] {
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	trendData, err := s.metricsRepo.GetTrendData(ctx, metric, period, city)
	if err != nil {
		s.logger.Error("failed to fetch trend data",
			zap.String("metric", metric),
			zap.String("period", period),
			zap.String("city", city),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch trend data: %w", err)
	}

	return trendData, nil
}

// GetPeakHours returns peak hour analysis for the given city and day of week
func (s *AnalyticsService) GetPeakHours(ctx context.Context, city string, dayOfWeek int) ([]*PeakHourData, error) {
	s.logger.Info("fetching peak hours",
		zap.String("city", city),
		zap.Int("day_of_week", dayOfWeek),
	)

	if dayOfWeek < 0 || dayOfWeek > 6 {
		return nil, fmt.Errorf("day_of_week must be between 0 (Sunday) and 6 (Saturday), got %d", dayOfWeek)
	}

	peakHours, err := s.metricsRepo.GetPeakHourData(ctx, city, dayOfWeek)
	if err != nil {
		s.logger.Error("failed to fetch peak hours",
			zap.String("city", city),
			zap.Int("day_of_week", dayOfWeek),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch peak hours: %w", err)
	}

	return peakHours, nil
}

// GetPopularRoutes returns popular routes for the given city
func (s *AnalyticsService) GetPopularRoutes(ctx context.Context, city string, limit int) ([]*RouteData, error) {
	s.logger.Info("fetching popular routes",
		zap.String("city", city),
		zap.Int("limit", limit),
	)

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	routes, err := s.metricsRepo.GetTopRoutes(ctx, city, limit)
	if err != nil {
		s.logger.Error("failed to fetch popular routes",
			zap.String("city", city),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch popular routes: %w", err)
	}

	for i, route := range routes {
		route.Rank = i + 1
	}

	return routes, nil
}

// GetDriverPerformance returns performance data for a specific driver
func (s *AnalyticsService) GetDriverPerformance(ctx context.Context, driverID string, startDate, endDate time.Time) (*DriverPerformance, error) {
	s.logger.Info("fetching driver performance",
		zap.String("driver_id", driverID),
		zap.Time("start_date", startDate),
		zap.Time("end_date", endDate),
	)

	if driverID == "" {
		return nil, fmt.Errorf("driver_id is required")
	}
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end_date must be after start_date")
	}

	performance, err := s.metricsRepo.GetDriverPerformanceData(ctx, driverID, startDate, endDate)
	if err != nil {
		s.logger.Error("failed to fetch driver performance",
			zap.String("driver_id", driverID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch driver performance for driver %s: %w", driverID, err)
	}

	if performance.TotalRides > 0 {
		performance.UtilizationRate = float64(performance.CompletedRides) / float64(performance.TotalRides) * 100
	}

	return performance, nil
}

// GetRiderBehavior returns behavior data for a specific rider
func (s *AnalyticsService) GetRiderBehavior(ctx context.Context, riderID string, startDate, endDate time.Time) (*RiderBehavior, error) {
	s.logger.Info("fetching rider behavior",
		zap.String("rider_id", riderID),
		zap.Time("start_date", startDate),
		zap.Time("end_date", endDate),
	)

	if riderID == "" {
		return nil, fmt.Errorf("rider_id is required")
	}
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end_date must be after start_date")
	}

	behavior, err := s.metricsRepo.GetRiderBehaviorData(ctx, riderID, startDate, endDate)
	if err != nil {
		s.logger.Error("failed to fetch rider behavior",
			zap.String("rider_id", riderID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch rider behavior for rider %s: %w", riderID, err)
	}

	return behavior, nil
}

// AggregateHourlyMetrics aggregates metrics for the current hour
func (s *AnalyticsService) AggregateHourlyMetrics(ctx context.Context) error {
	now := time.Now().UTC()
	hour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)

	s.logger.Info("aggregating hourly metrics", zap.Time("hour", hour))

	if err := s.metricsRepo.AggregateHourly(ctx, hour); err != nil {
		s.logger.Error("failed to aggregate hourly metrics",
			zap.Time("hour", hour),
			zap.Error(err),
		)
		return fmt.Errorf("failed to aggregate hourly metrics for %s: %w", hour.Format(time.RFC3339), err)
	}

	s.logger.Info("hourly metrics aggregation completed", zap.Time("hour", hour))
	return nil
}

// AggregateDailyMetrics aggregates metrics for the current day
func (s *AnalyticsService) AggregateDailyMetrics(ctx context.Context) error {
	now := time.Now().UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	s.logger.Info("aggregating daily metrics", zap.Time("day", day))

	if err := s.metricsRepo.AggregateDaily(ctx, day); err != nil {
		s.logger.Error("failed to aggregate daily metrics",
			zap.Time("day", day),
			zap.Error(err),
		)
		return fmt.Errorf("failed to aggregate daily metrics for %s: %w", day.Format("2006-01-02"), err)
	}

	s.logger.Info("daily metrics aggregation completed", zap.Time("day", day))
	return nil
}

// ReportService handles report generation business logic
type ReportService struct {
	reportRepo  ReportRepository
	metricsRepo MetricsRepository
	logger      *zap.Logger
}

// NewReportService creates a new ReportService
func NewReportService(reportRepo ReportRepository, metricsRepo MetricsRepository, logger *zap.Logger) *ReportService {
	return &ReportService{
		reportRepo:  reportRepo,
		metricsRepo: metricsRepo,
		logger:      logger,
	}
}

// GenerateDailyReport generates a comprehensive daily report for the given date
func (s *ReportService) GenerateDailyReport(ctx context.Context, date time.Time) (*Report, error) {
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endDate := date.Add(24 * time.Hour)

	s.logger.Info("generating daily report", zap.Time("date", date))

	rideMetrics, err := s.metricsRepo.GetRideStats(ctx, date, endDate, "")
	if err != nil {
		s.logger.Error("failed to get ride stats for daily report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate daily report: %w", err)
	}

	revenueMetrics, err := s.metricsRepo.GetRevenueStats(ctx, date, endDate, "", "hour")
	if err != nil {
		s.logger.Error("failed to get revenue stats for daily report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate daily report: %w", err)
	}

	driverMetrics, err := s.metricsRepo.GetDriverStats(ctx, "")
	if err != nil {
		s.logger.Error("failed to get driver stats for daily report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate daily report: %w", err)
	}

	report := &Report{
		ID:          fmt.Sprintf("daily-%s", date.Format("2006-01-02")),
		Type:        "daily",
		Status:      "completed",
		PeriodStart: date,
		PeriodEnd:   endDate,
		GeneratedAt: time.Now().UTC(),
		Data: map[string]interface{}{
			"rides":   rideMetrics,
			"revenue": revenueMetrics,
			"drivers": driverMetrics,
		},
	}

	if err := s.reportRepo.SaveReport(ctx, report); err != nil {
		s.logger.Error("failed to save daily report", zap.Error(err))
		return nil, fmt.Errorf("failed to save daily report: %w", err)
	}

	s.logger.Info("daily report generated", zap.String("report_id", report.ID))
	return report, nil
}

// GenerateWeeklyReport generates a weekly report starting from the given date
func (s *ReportService) GenerateWeeklyReport(ctx context.Context, weekStart time.Time) (*Report, error) {
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.UTC)
	weekEnd := weekStart.Add(7 * 24 * time.Hour)

	s.logger.Info("generating weekly report", zap.Time("week_start", weekStart))

	rideMetrics, err := s.metricsRepo.GetRideStats(ctx, weekStart, weekEnd, "")
	if err != nil {
		s.logger.Error("failed to get ride stats for weekly report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate weekly report: %w", err)
	}

	revenueMetrics, err := s.metricsRepo.GetRevenueStats(ctx, weekStart, weekEnd, "", "day")
	if err != nil {
		s.logger.Error("failed to get revenue stats for weekly report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate weekly report: %w", err)
	}

	riderMetrics, err := s.metricsRepo.GetRiderStats(ctx, "")
	if err != nil {
		s.logger.Error("failed to get rider stats for weekly report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate weekly report: %w", err)
	}

	topRoutes, err := s.metricsRepo.GetTopRoutes(ctx, "", 10)
	if err != nil {
		s.logger.Error("failed to get top routes for weekly report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate weekly report: %w", err)
	}

	report := &Report{
		ID:          fmt.Sprintf("weekly-%s", weekStart.Format("2006-01-02")),
		Type:        "weekly",
		Status:      "completed",
		PeriodStart: weekStart,
		PeriodEnd:   weekEnd,
		GeneratedAt: time.Now().UTC(),
		Data: map[string]interface{}{
			"rides":      rideMetrics,
			"revenue":    revenueMetrics,
			"riders":     riderMetrics,
			"top_routes": topRoutes,
		},
	}

	if err := s.reportRepo.SaveReport(ctx, report); err != nil {
		s.logger.Error("failed to save weekly report", zap.Error(err))
		return nil, fmt.Errorf("failed to save weekly report: %w", err)
	}

	s.logger.Info("weekly report generated", zap.String("report_id", report.ID))
	return report, nil
}

// GenerateMonthlyReport generates a monthly report for the given month
func (s *ReportService) GenerateMonthlyReport(ctx context.Context, month time.Time) (*Report, error) {
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	s.logger.Info("generating monthly report", zap.Time("month", monthStart))

	rideMetrics, err := s.metricsRepo.GetRideStats(ctx, monthStart, monthEnd, "")
	if err != nil {
		s.logger.Error("failed to get ride stats for monthly report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate monthly report: %w", err)
	}

	revenueMetrics, err := s.metricsRepo.GetRevenueStats(ctx, monthStart, monthEnd, "", "week")
	if err != nil {
		s.logger.Error("failed to get revenue stats for monthly report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate monthly report: %w", err)
	}

	driverMetrics, err := s.metricsRepo.GetDriverStats(ctx, "")
	if err != nil {
		s.logger.Error("failed to get driver stats for monthly report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate monthly report: %w", err)
	}

	riderMetrics, err := s.metricsRepo.GetRiderStats(ctx, "")
	if err != nil {
		s.logger.Error("failed to get rider stats for monthly report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate monthly report: %w", err)
	}

	trendData, err := s.metricsRepo.GetTrendData(ctx, "rides", "daily", "")
	if err != nil {
		s.logger.Error("failed to get trend data for monthly report", zap.Error(err))
		return nil, fmt.Errorf("failed to generate monthly report: %w", err)
	}

	report := &Report{
		ID:          fmt.Sprintf("monthly-%s", monthStart.Format("2006-01")),
		Type:        "monthly",
		Status:      "completed",
		PeriodStart: monthStart,
		PeriodEnd:   monthEnd,
		GeneratedAt: time.Now().UTC(),
		Data: map[string]interface{}{
			"rides":   rideMetrics,
			"revenue": revenueMetrics,
			"drivers": driverMetrics,
			"riders":  riderMetrics,
			"trends":  trendData,
		},
	}

	if err := s.reportRepo.SaveReport(ctx, report); err != nil {
		s.logger.Error("failed to save monthly report", zap.Error(err))
		return nil, fmt.Errorf("failed to save monthly report: %w", err)
	}

	s.logger.Info("monthly report generated", zap.String("report_id", report.ID))
	return report, nil
}

// GenerateCustomReport generates a custom report based on the provided request
func (s *ReportService) GenerateCustomReport(ctx context.Context, request *CustomReportRequest) (*Report, error) {
	s.logger.Info("generating custom report",
		zap.String("city", request.City),
		zap.Strings("metrics", request.Metrics),
		zap.Time("start_date", request.StartDate),
		zap.Time("end_date", request.EndDate),
	)

	if request == nil {
		return nil, fmt.Errorf("custom report request cannot be nil")
	}
	if request.EndDate.Before(request.StartDate) {
		return nil, fmt.Errorf("end_date must be after start_date")
	}
	if len(request.Metrics) == 0 {
		return nil, fmt.Errorf("at least one metric must be specified")
	}

	data := make(map[string]interface{})

	for _, metric := range request.Metrics {
		switch metric {
		case "rides":
			rideMetrics, err := s.metricsRepo.GetRideStats(ctx, request.StartDate, request.EndDate, request.City)
			if err != nil {
				s.logger.Error("failed to get ride stats for custom report", zap.Error(err))
				return nil, fmt.Errorf("failed to fetch rides metric: %w", err)
			}
			data["rides"] = rideMetrics
		case "revenue":
			groupBy := request.GroupBy
			if groupBy == "" {
				groupBy = "day"
			}
			revenueMetrics, err := s.metricsRepo.GetRevenueStats(ctx, request.StartDate, request.EndDate, request.City, groupBy)
			if err != nil {
				s.logger.Error("failed to get revenue stats for custom report", zap.Error(err))
				return nil, fmt.Errorf("failed to fetch revenue metric: %w", err)
			}
			data["revenue"] = revenueMetrics
		case "drivers":
			driverMetrics, err := s.metricsRepo.GetDriverStats(ctx, request.City)
			if err != nil {
				s.logger.Error("failed to get driver stats for custom report", zap.Error(err))
				return nil, fmt.Errorf("failed to fetch drivers metric: %w", err)
			}
			data["drivers"] = driverMetrics
		case "riders":
			riderMetrics, err := s.metricsRepo.GetRiderStats(ctx, request.City)
			if err != nil {
				s.logger.Error("failed to get rider stats for custom report", zap.Error(err))
				return nil, fmt.Errorf("failed to fetch riders metric: %w", err)
			}
			data["riders"] = riderMetrics
		case "routes":
			routes, err := s.metricsRepo.GetTopRoutes(ctx, request.City, 20)
			if err != nil {
				s.logger.Error("failed to get top routes for custom report", zap.Error(err))
				return nil, fmt.Errorf("failed to fetch routes metric: %w", err)
			}
			data["routes"] = routes
		default:
			s.logger.Warn("unknown metric requested in custom report", zap.String("metric", metric))
		}
	}

	reportID := fmt.Sprintf("custom-%d", time.Now().UnixNano())
	report := &Report{
		ID:          reportID,
		Type:        "custom",
		Status:      "completed",
		City:        request.City,
		PeriodStart: request.StartDate,
		PeriodEnd:   request.EndDate,
		GeneratedAt: time.Now().UTC(),
		Format:      request.Format,
		Data:        data,
	}

	if err := s.reportRepo.SaveReport(ctx, report); err != nil {
		s.logger.Error("failed to save custom report", zap.Error(err))
		return nil, fmt.Errorf("failed to save custom report: %w", err)
	}

	s.logger.Info("custom report generated", zap.String("report_id", report.ID))
	return report, nil
}

// DownloadReport retrieves a report and serializes it in the specified format
func (s *ReportService) DownloadReport(ctx context.Context, reportID, format string) ([]byte, error) {
	s.logger.Info("downloading report",
		zap.String("report_id", reportID),
		zap.String("format", format),
	)

	if reportID == "" {
		return nil, fmt.Errorf("report_id is required")
	}

	report, err := s.reportRepo.GetReport(ctx, reportID)
	if err != nil {
		s.logger.Error("failed to fetch report",
			zap.String("report_id", reportID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch report %s: %w", reportID, err)
	}

	switch format {
	case "json", "":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			s.logger.Error("failed to marshal report to JSON", zap.Error(err))
			return nil, fmt.Errorf("failed to serialize report to JSON: %w", err)
		}
		return data, nil
	case "csv":
		data, err := s.serializeReportToCSV(report)
		if err != nil {
			s.logger.Error("failed to serialize report to CSV", zap.Error(err))
			return nil, fmt.Errorf("failed to serialize report to CSV: %w", err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s, must be json or csv", format)
	}
}

// serializeReportToCSV converts a Report to CSV bytes
func (s *ReportService) serializeReportToCSV(report *Report) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	header := []string{"report_id", "type", "status", "period_start", "period_end", "generated_at", "city", "data_key", "data_value"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	for key, value := range report.Data {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data value for key %s: %w", key, err)
		}
		row := []string{
			report.ID,
			report.Type,
			report.Status,
			report.PeriodStart.Format(time.RFC3339),
			report.PeriodEnd.Format(time.RFC3339),
			report.GeneratedAt.Format(time.RFC3339),
			report.City,
			key,
			string(valueJSON),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportData performs a GDPR-compliant data export for a user
func (s *ReportService) ExportData(ctx context.Context, request *ExportRequest) (*ExportResult, error) {
	s.logger.Info("exporting user data",
		zap.String("user_id", request.UserID),
		zap.String("user_type", request.UserType),
		zap.String("format", request.Format),
	)

	if request == nil {
		return nil, fmt.Errorf("export request cannot be nil")
	}
	if request.UserID == "" {
		return nil, fmt.Errorf("user_id is required for data export")
	}
	if request.UserType != "driver" && request.UserType != "rider" {
		return nil, fmt.Errorf("user_type must be 'driver' or 'rider', got '%s'", request.UserType)
	}
	if request.Format == "" {
		request.Format = "json"
	}

	records, err := s.reportRepo.GetExportData(ctx, request)
	if err != nil {
		s.logger.Error("failed to retrieve export data",
			zap.String("user_id", request.UserID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to retrieve export data for user %s: %w", request.UserID, err)
	}

	var exportedData []byte
	switch request.Format {
	case "json":
		exportPayload := map[string]interface{}{
			"user_id":      request.UserID,
			"user_type":    request.UserType,
			"export_date":  time.Now().UTC().Format(time.RFC3339),
			"period_start": request.StartDate.Format(time.RFC3339),
			"period_end":   request.EndDate.Format(time.RFC3339),
			"records":      records,
		}
		exportedData, err = json.MarshalIndent(exportPayload, "", "  ")
		if err != nil {
			s.logger.Error("failed to marshal export data to JSON", zap.Error(err))
			return nil, fmt.Errorf("failed to serialize export data to JSON: %w", err)
		}
	case "csv":
		exportedData, err = s.serializeExportToCSV(records)
		if err != nil {
			s.logger.Error("failed to serialize export data to CSV", zap.Error(err))
			return nil, fmt.Errorf("failed to serialize export data to CSV: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported export format: %s, must be json or csv", request.Format)
	}

	result := &ExportResult{
		UserID:      request.UserID,
		Format:      request.Format,
		Data:        exportedData,
		GeneratedAt: time.Now().UTC(),
		RecordCount: len(records),
	}

	s.logger.Info("data export completed",
		zap.String("user_id", request.UserID),
		zap.Int("record_count", result.RecordCount),
	)

	return result, nil
}

// serializeExportToCSV converts a slice of records to CSV bytes
func (s *ReportService) serializeExportToCSV(records []map[string]interface{}) ([]byte, error) {
	if len(records) == 0 {
		return []byte{}, nil
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := make([]string, 0, len(records[0]))
	for key := range records[0] {
		headers = append(headers, key)
	}

	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, record := range records {
		row := make([]string, 0, len(headers))
		for _, header := range headers {
			val := record[header]
			row = append(row, fmt.Sprintf("%v", val))
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}
