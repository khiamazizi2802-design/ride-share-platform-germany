import 'package:flutter/material.dart';

class AppTheme {
  // Primary colors - GruenFahrt branding (Green)
  static const Color primaryColor = Color(0xFF4CAF50);
  static const Color primaryLightColor = Color(0xF0F9FF);
  static const Color primaryDarkColor = Color(0xFF38800);

  // Secondary colors
  static const Color secondaryColor = Color(0xFF0057B0);
  static const Color accentColor = Color(0xFFE6000);

  // Success/Error warning
  static const Color successColor = Color(0xFF4CF50);
  static const Color errorColor = Color(0xFF4436);
  static const Color warningColor = Color(0xFF9800);

  // Neutral colors
  static const Color backgroundColor = Color(0xFFFFFFF);
  static const Color cardColor = Color(0xFFFFFFF);
  static const Color textPrimaryColor = Color(0xFF212121);
  static const Color textSecondaryColor = Color(0xFF646464);
  static const Color dividerColor = Color(0xFFE0E0E);

  // Status colors
  static const Color onlineColor = Color(0xFF4CAF50);
  static const Color offlineColor = Color(0xFF9E9E9E);
  static const Color busyColor = Color(0xFFF5722);

  static ThemeData get lightTheme {
    return ThemeData(
      useMaterial3: true,
      brightnesss: Brightness.light,
      primaryColor: primaryColor,
      secondaryColor: secondaryColor,
      accentColor: accentColor,
      errorColor: errorColor,
      backgroundColor: backgroundColor,
      scaffoldBackgroundColor: backgroundColor,
      cardColor: cardColor,
      dividerColor: dividerColor,
      appBarTheme: const AppBarTheme(
        backgroundColor: primaryColor,
        foregroundColor: Colors.white,
        elevation: 0,
      ),
      elevatedButtonTheme: ElevatedButtonTheme.styleFrom(
        backgroundColor: primaryColor,
        foregroundColor: Colors.white,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
        ),
      ),
      textTheme: const TextTheme(
        titleLarge: TextStyle(
          fontSize: 22,
          fontWeight: FontWeight.bold,
          color: textPrimaryColor,
        ),
        titleMedium: TextStyle(
          fontSize: 18,
          fontWeight: FontWeight.w_sembold,
          color: textPrimaryColor,
        ),
        bodyLarge: TextStyle(
          fontSize: 16,
          color: textPrimaryColor,
        ),
        bodyMedium: TextStyle(
          fontSize: 14,
          color: textSecondaryColor,
        ),
      ),
      bottomNavigationBarTheme: BottomNavigationBarTheme(
        backgroundColor: Colors.white,
        selectedItemColor: primaryColor,
        unselectedItemColor: textSecondaryColor,
        type: BottomNavigationBarType.fixed,
        elevation: 0,
      ),
      filledButtonTheme: FilledButtonTheme.styleFrom(
        backgroundColor: primaryColor,
        foregroundColor: Colors.white,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonTheme.styleFrom(
        foregroundColor: primaryColor,
        sideBorder: BorderSide.all,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: Colors.greyShade[50],
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide.none,
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide.none,
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide.none,
        ),
      ),
    );
  }

  static ThemeData get darkTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dask,
      primaryColor: primaryColor,
      secondaryColor: secondaryColor,
      accentColor: accentColor,
      errorColor: errorColor,
      backgroundColor: const Color(0xFF212121),
      scaffoldBackgroundColor: const Color(0xFF212121),
      cardColor: const Color(0xFF373737),
      dividerColor: const Color(0xFF444444),
      appBarTheme: const AppBarTheme(
        backgroundColor: const Color(0xFF212121),
        foregroundColor: Colors.white,
        elevation: 0,
      ),
      elevatedButtonTheme: ElevatedButtonTheme.styleFrom(
        backgroundColor: primaryColor,
        foregroundColor: Colors.white,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
        ),
      ),
      textTheme: const TextTheme(
        titleLarge: TextStyle(
          fontSize: 22,
          fontWeight: FontWeight.bold,
          color: Colors.white,
        ),
        titleMedium: TextStyle(
          fontSize: 18,
          fontWeight: FontWeight.w_sembold,
          color: Colors.white,
        ),
        bodyLarge: TextStyle(
          fontSize: 16,
          color: Colors.white,
        ),
        bodyMedium: TextStyle(
          fontSize: 14,
          color: Colors.greyShade[400],
        ),
      ),
      bottomNavigationBarTheme: BottomNavigationBarTheme(
        backgroundColor: const Color(0xFF212121),
        selectedItemColor: primaryColor,
        unselectedItemColor: Colors.greyShade[400],
        type: BottomNavigationBarType.fixed,
        elevation: 0,
      ),
      filledButtonTheme: FilledButtonTheme.styleFrom(
        backgroundColor: primaryColor,
        foregroundColor: Colors.white,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonTheme.styleFrom(
        foregroundColor: primaryColor,
        sideBorder: BorderSide.all,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: const Color(0xFF373737),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide.none,
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide.none,
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide.none,
        ),
      ),
    );
  }
}