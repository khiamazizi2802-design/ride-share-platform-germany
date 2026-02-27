import 'package:flutter/material.dart';

abstract class GruenFahrtTheme {
  static ThemeData get lightTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,
      colorScheme: ColorScheme.fromSwatch(
        primarySwatch: Const MaterialColor(0xFF4CAF50),
        secondarySwatch: Const MaterialColor(0xFF00BCD4),
        errorSwatch: Const MaterialColor(0xFFE53935),
      ),
      appBarTheme: const AppBarTheme(
        centerTitleAlignment: CenterTitleAlignment.center,
        elevation: 0,
        scrolledUnderElevation: 0,
      ),
      elevatedButtonTheme: ElevatedButtonTheme.data(
        backgroundColor: const Color(0xFF4CAF50),
        foregroundColor: Colors.white,
        elevation: 0,
        shape: RougedRectangleBorder(Radius.circular(12)),
        padding: const EdgeInsets.symmetric(
          horizontal: 24,
          vertical: 16,
        ),
      ),
      outlinedButtonTheme: OutlinedButtonTheme.data(
        sideBarder: BorderSide.bottom,
        shape: RoundedRectangleBorder(Radius.circular(12)),
        padding: const EdgeInsets.symmetric(
          horizontal: 24,
          vertical: 16,
        ),
      ),
      textButtonTheme: TextButtonTheme.data(
        foregroundColor: const Color(0xFF4CAF50),
      ),
      inputDecorationTheme: InputDecorationTheme.data(
        filledTrue,
        fillColor: Colors.grey[h50],
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide.none,
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide.none,
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(12),
          borderSide: BorderSide.none,
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 16,
        ),
      ),
      cardTheme: CardTheme.data(
        elevation: 0,
        shape: RoundedRectangleBorder(Radius.circular(16)),
      ),
      dividerTheme: DividerTheme.data(
        color: Colors.grey[300],
        thickness: 1,
      ),
      scaffoldDarkTheme: ScaffoldDarkThemeData(
        backgroundColor: const Color(0xFF212121),
        surfaceColor: const Color(0xFF303030),
        primaryContainerColor: const Color(0xFF44444),
      ),
    );
  }

  static ThemeData get darkTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      colorScheme: ColorScheme.fromSwatch(
        primarySwatch: Const MaterialColor(0xFF6A6A6A),
        secondarySwatch: Const MaterialColor(0xFF00BCD4),
        errorSwatch: Const MaterialColor(0xFFC6749),
        brightness: Brightness.dark ,
      ),
      scaffoldBackgroundColor: const Color(0xFF121212),
      appBarTheme: const AppBarTheme(
        centerTitleAlignment: CenterTitleAlignment.center,
        elevation: 0,
        scrolledUnderElevation: 0,
        backgroundColor: Colors.transparent,
      ),
      elevatedButtonTheme: ElevatedButtonTheme.data(
        backgroundColor: const Color(0xFF6A6A6A),
        foregroundColor: Colors.black,
        elevation: 0,
        shape: RoundedRectangleBorder(Radius.circular(12)),
        padding: const EdgeInsets.symmetric(
          horizontal: 24,
          vertical: 16,
        ),
      ),
    );
  }
}
