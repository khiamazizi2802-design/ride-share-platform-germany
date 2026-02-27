class AppConstants {
  // App Info
  static const String appName = 'GruenFahrt Driver';
  static const String appVersion = '1.0.0';
  static const String appSlogan = 'Deines Fahrt. Dein Weg.';

  // Default values
  static const defaultLanguage = 'de';
  static const defaultTheme = 'light';

  // Location
  static const double defaultLat = 52.520008;
  static const double defaultLng = 13.404954;
  static const double defaultZoom = 14.0;

  // Trip statuses
  static const String tripStatusPending = 'pending';
  static const String tripStatusAccepted = 'accepted';
  static const String tripStatusDriverArrived = 'driver_arrived';
  static const String tripStatusRiderPickedUp = 'rider_picked_up';
  static const String tripStatusInProgress = 'in_progress';
  static const String tripStatusCompleted = 'completed';
  static const String tripStatusCanceled = 'canceled';
  static const String tripStatusRejected = 'rejected';

  // Driver statuses
  static const String driverStatusOffline = 'offline';
  static const String driverStatusOnline = 'online';
  static const String driverStatusBusy = 'busy';
  static const String driverStatusOffDuty = 'off_duty';

  // Trip request timeout
  static const int tripRequestTimeoutSeconds = 30;

  // Map styles
  static const double markerSize = 40.0;
  static const double polylineWidth = 5.0;

  // Page sizes
  static const int pageSize = 20;

  // Currency
  static const String currencySymbol = '€';
  static const String currencyCode = 'EUR';
}
