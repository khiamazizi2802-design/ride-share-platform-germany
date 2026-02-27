class AppConstants {
  // API
  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'https://api.gruenfahrt.de/v1',
  );
  
  // Google Maps
  static const String googleMapsApiKey = String.fromEnvironment(
    'GOOGLE_MAPS_API_KEY',
    defaultValue: '',
  );
  
  // Stripe
  static const String stripePublishableKey = String.fromEnvironment(
    'STRIPE_PUBLISHABLE_KEY',
    defaultValue: '',
  );
  
  // App
  static const String appName = 'GruenFahrt';
  static const String appVersion = '1.0.0';
  
  // Defaults
  static const double defaultMapZoom = 15.0;
  static const Duration locationUpdateInterval = Duration(seconds: 5);
  static const Duration tripUpdateInterval = Duration(seconds: 3);
  
  // German regulations
  static const double maxWaitingTimeMinutes = 5.0;
  static const double baseFare = 3.50;
  static const double perKmRate = 1.80;
  static const double perMinuteRate = 0.35;
}