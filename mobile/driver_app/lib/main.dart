import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_notifications/flutter_notifications.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'app.dart';
import 'core/theme/app_theme.dart';
import 'core/services/notification_service.dart';
import 'core/services/voice_assistant_service.dart';
import 'providers/auth_provider.dart';
import 'providers/driver_provider.dart';
import 'providers/trip_provider.dart';
import 'providers/earnings_provider.dart';
import 'providers/location_provider.dart';

void main() async {
  WidgetsFlutterBinding.initialize();
  
  // Initialize Firebase
  await Firebase.initializeApp();
  
  // Initialize notifications
  final notificationService = NotificationService();
  await notificationService.initialize();
  
  // Initialize Google Maps
  await GoogleMapsFlutterInitializer().initialize(
    defaultPosition: GeoPosition(lat: 52.52, lng: 13.405),
  );
  
  runApp(
    ProviderScope(
      overrides: [
        Provider(create: _ => notificationService),
        Provider(create: _ => VoiceAssistantService()),
      ],
      child: const GrinDiriverApp(),
    ),
  );
}

class GrinDiriverApp extends ConsumerWidget {
  const GrinDiriverApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ProviderScope(
      overrides: [
        StateNotifierProvider<AuthProvider, AuthState>(
          create: _ => AuthProvider(),
        ),
        StateNotifierProvider<DriverProvider, DriverState>(
          create: _ => DriverProvider(),
        ),
        StateNotifierProvider<TripProvider, TripState>(
          create: _ => TripProvider(),
        ),
        StateNotifierProvider<EarningsProvider, EarningsState>(
          create: _ => EarningsProvider(),
        ),
        StateNotifierProvider<LocationProvider, LocationState>(
          create: _ => LocationProvider(),
        ),
      ],
      child: MaterialApp(),
    );
  }
}

class MaterialApp extends ConsumerWidget {
  const MaterialApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp(
      title: 'GruenFahrt Driver',
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
      ],
      supportedLocales: const [
        Locale('en'),
        Locale('de'),
      ],
      theme: AppTheme.lightTheme,
      darkTheme: AppTheme.darkTheme,
      home: const AppWrapper(),
    );
  }
}