import 'package:firebase_core/firebase_core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:gruenfahrt_rider/app.dart';
import 'package:gruenfahrt_rider/core/constants.dart';
import 'package:gruenfahrt_rider/core/di/service_locator.dart';
import 'package:gruenfahrt_rider/blocs/auth/auth_bloc.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  
  // Firebase initialisieren
  await Firebase.initializeApp();
  
  // Dependency Injection setup
  await setupServiceLocator();
  
  // System Chrome Styling
  SystemChrome.setPreferredOrientations([DeviceOrientation.portraitUp]);
  SystemChrome.setEnabledSystemUIOverlayStyle(
    const SystemUiOverlayStyle(
      statusBarColor: Colors.transparent,
      statusBarIconBrightness: Brightness.dark
    ),
  );

  runApp(
    BlocProvider(
      create: (context) => serviceLocator<AuthBloc>(),
      child: const GruenFahrtApp(),
    ),
  );
}
