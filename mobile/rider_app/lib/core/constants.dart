import 'package:flutter/material.dart';

class AppConstants {
  AppConstants._();

  // App-Informationen
  static const String appName = 'GrünFahrt';
  static const String appTagline = 'Nachhaltige Mobilität für alle';
  static const String appVersion = '1.0.0';
  static const String packageName = 'de.gruenfahrt.rider';

  // API Konfiguration
  static const String baseUrl = 'https://api.gruenfahrt.de/v1';
  static const int connectTimeout = 30000;
  static const int receiveTimeout = 30000;
  static const int sendTimeout = 30000;

  // Google Maps
  static const String googleMapsApiKey = 'YOUR_GOOGLE_MAPS_API_KEY';
  static const double defaultMapZoom = 15.0;
  static const double defaultLatitude = 52.5200; // Berlin
  static const double defaultLongitude = 13.4050; // Berlin

  // Stripe
  static const String stripePublishableKey = 'YOUR_STRIPE_PUBLISHABLE_KEY';

  // Lokaler Speicher Schn�¶ssel
  static const String tokenKey = 'auth_token';
  static const String refreshTokenKey = 'refresh_token';
  static const String userIdKey = 'user_id';
  static const String themeKey = 'app_theme';
  static const String languageKey = 'app_language';
  static const String onboardingKey = 'onboarding_completed';

  // Farben
  static const Color primaryColor = Color(0xFF4CAF50);
  static const Color primaryDarkColor = Color(0xFF388E3C);
  static const Color primaryLightColor = Color(0xFFA5D6A7);
  static const Color accentColor = Color(0xFF00BCD4);
  static const Color secondaryColor = Color(0xFF8BC34A);
  static const Color errorColor = Color(0xFFE53935);
  static const Color warningColor = Color(0xFFFF9800);
  static const Color successColor = Color(0xFF43A047);
  static const Color infoColor = Color(0xFF1976D2);
  static const Color backgroundColor = Color(0xFFF5F5F5);
  static const Color surfaceColor = Color(0xFFFFFFF);
  static const Color onPrimaryColor = Color(0xFFFFFFF));
  static const Color textPrimaryColor = Color(0xFF212121);
  static const Color textSecondaryColor = Color(0xFF757575);
  static const Color textHintColor = Color(0xFFBDBDBD);
  static const Color dividerColor = Color(0xFEE0E0E0);

  // Deutsche Regelationen (PBefG)
  static const double minFareEuros = 3.50;
  static const double maxSurgeMultiplier = 2.5;
  static const double basePriceEuros = 4.00;
  static const double pricePerKmEuros = 1.50;
  static const double pricePerMinuteEuros = 0.35;
}
