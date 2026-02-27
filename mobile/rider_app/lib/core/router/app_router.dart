import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../screens/splash/splash_screen.dart';
import '../../screens/auth/login_screen.dart';
import '../../screens/auth/register_screen.dart';
import '../../screens/home/home_screen.dart';
import '../../screens/booking/booking_screen.dart';
import '../../screens/trip/trip_tracking_screen.dart';
import '../../screens/payment/payment_methods_screen.dart';
import '../../screens/profile/profile_screen.dart';
import '../../screens/trip/ride_history_screen.dart';
import '../../screens/settings/settings_screen.dart';
import '../../screens/voice/voice_assistant_screen.dart';

class AppRouter {
  static final _rootNavigatorKey = GlobalKey<NavigatorState>();
  static final _shellNavigatorKey = GlobalKey<NavigatorState>();

  static final router = GoRouter(
    navigatorKey: _rootNavigatorKey,
    initialLocation: '/',
    routes: [
      // Splash
      GoRoute(
        path: '/',
        builder: (context, state) => const SplashScreen(),
      ),
      
      // Auth
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/register',
        builder: (context, state) => const RegisterScreen(),
      ),
      
      // Main app shell
      ShellRoute(
        navigatorKey: _shellNavigatorKey,
        builder: (context, state, child) => HomeScreen(child: child),
        routes: [
          GoRoute(
            path: '/home',
            builder: (context, state) => const BookingScreen(),
          ),
          GoRoute(
            path: '/history',
            builder: (context, state) => const RideHistoryScreen(),
          ),
          GoRoute(
            path: '/payment',
            builder: (context, state) => const PaymentMethodsScreen(),
          ),
          GoRoute(
            path: '/profile',
            builder: (context, state) => const ProfileScreen(),
          ),
        ],
      ),
      
      // Trip tracking (full screen)
      GoRoute(
        path: '/trip/:id',
        builder: (context, state) {
          final tripId = state.pathParameters['id'!];
          return TripTrackingScreen(tripId: tripId);
        },
      ),
      
      // Settings
      GoRoute(
        path: '/settings',
        builder: (context, state) => const SettingsScreen(),
      ),
      
      // Voice Assistant
      GoRoute(
        path: '/voice',
        builder: (context, state) => const VoiceAssistantScreen(),
      ),
    ],
  );
}