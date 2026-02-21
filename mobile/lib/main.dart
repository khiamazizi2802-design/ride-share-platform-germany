import 'package:flutter/material.dart';
import 'screens/onboarding_screen.dart';
import 'screens/verification_screen.dart';

void main() {
  runApp(const RideShareApp());
}

class RideShareApp extends StatelessWidget {
  const RideShareApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'RideShare Germany',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.green),
        useMaterial3: true,
      ),
      initialRoute: '/',
      routes: {
        '/': (context) => const OnboardingScreen(),
        '/verify': (context) => const VerificationScreen(),
      },
    );
  }
}
