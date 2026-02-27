import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:gruenfahrt_rider/core/theme.dart';
import 'package:gruenfahrt_rider/core/router.dart';
import 'package:gruenfahrt_rider/blocs/auth/auth_bloc.dart';
import 'package:gruenfahrt_rider/core/constants.dart';
import 'package:gruenfahrt_rider/screens/splash_screen.dart';
import 'package:gruenfahrt_rider/screens/login_screen.dart';
import 'package:gruenfahrt_rider/screens/home_screen.dart';

class GruenFahrtApp extends StatefulWidget {
  const GruenFahrtApp({super.key});

  @override
  State<GruenFahrtApp> createState() => _GruenFahrtAppState();
}

class _GruenFahrtAppState extends State<GruenFahrtApp> {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'GrünFahrt',
      debugShowCheckedModeBanner: false,
      
      // Thema
      theme: GruenFahrtTheme.lightTheme,
      darkTheme: GruenFahrtTheme.darkTheme,
      themeMode: ThemeMode.system,
      
      // Lokalisierung
      locale: const Locale('de', 'DE'),
      supportedLocales: const [
        Locale('de', 'DE'),
        Locale('en', 'US'),
      ],
      localizationsDelegates: const [
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      
      home: BlocBuilder<AuthBloc, AuthState>(
        builder: (context, state) {
          if (state is AuthAuthenticated) {
            return const HomeScreen();
          }
          return const SplashScreen();
        },
      ),
    );
  }
}
