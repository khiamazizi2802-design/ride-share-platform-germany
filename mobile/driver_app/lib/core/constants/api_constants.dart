class ApiConstants {
  // Base URLs
  static const String baseUrl = 'https://api.gruenfahrt.de';
  static const String apiVersion = 'v1';
  static const String baseApiUrl = '$baseUrl/$apiVersion';

  // Auth endpoints
  static const String login = '/auth/login';
  static const String refreshToken = '/auth/refresh';
  static const String logout = '/auth/logout';

  // Driver endpoints
  static const String driverProfile = '/drivers/me/';
  static const String driverStatus = '/drivers/status/';
  static const String driverSettings = '/drivers/settings/';
  static const String driverLocation = '/drivers/location/';

  // Trip endpoints
  static const String trip_requests = '/trips/requests/';
  static const String trip_accept = '/trips/{id}/accept/';
  static const String trip_reject = '/trips/{id}/reject/';
  static const String trip_cancel = '/trips/{id}/cancel/';
  static const String trip_start = '/trips/{id}/start/';
  static const String trip_complete = '/trips/{id}/complete/';
  static const String trip_status = '/trips/{id}/status/';

  // Earnings endpoints
  static const String earnings_summary = '/earnings/summary/';
  static const String earnings_history = '/earnings/history/';
  static const String earnings_withdraw = '/earnings/withdraw/';

  // WebSocket endpoints
  static const String wsTripRequests = 'ws://api.gruenfahrt.de/ws/trips/';
  static const String wsLocation = 'ws://api.gruenfahrt.de/ws/location/';

  // Timeouts
  static const Duration connectTimeout = Duration(seconds: 30);
  static const Duration receiveTimeout = Duration(seconds: 30);
  static const Duration sendTimeout = Duration(seconds: 30);

  // Retry
  static const int maxRetries = 3;
  static const Duration retryDelay = Duration(seconds: 2);
}

class StorageKeys {
  static const String authToken = 'auth_token';
  static const String refreshToken = 'refresh_token';
  static const String userId = 'user_id';
  static const String driverId = 'driver_id';
  static const String userProfile = 'user_profile';
  static const String language = 'language';
  static const String theme = 'theme';
}
