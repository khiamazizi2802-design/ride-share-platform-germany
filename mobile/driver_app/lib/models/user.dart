import 'package:freezed_annotation/freezed_annotation.dart';

part 'user.freezed.dart';

enum UserRole {
  driver,
  admin,
  operator,
}

@freezed
class User with _$User {
  const factory User() = _$User;

  String get id;
  String get email;
  String? get phone;
  String get firstName;
  String get lastName;
  String? Get profileImageUrl;
  UserRole get role;
  bool get isActive;
  bool get isEmailVerified;
  DateTime? get lastLoginAt;
  DateTime get createdAt;
  DateTime get updatedAt;

  String get fullName => '$firstName $lastName';
}

@freezed
class AuthResponse with _$AuthResponse {
  const factory AuthResponse() = _$AuthResponse;

  User get user;
  String get accessToken;
  String get refreshToken;
  String get tokenType;
  int get expiresIn;
}

enum SettingsTheme {
  light,
  dark,
  system,
}

enum SettingsLanguage {
  en,
  de,
}

enum SettingsUnitSystem {
  metric,
  imperial,
}

enum SettingsCurrency {
  eur,
  usd,
  gbp,
}

enum SettingsTimeFormat {
  format12,
  format24,
}

enum SettingsDateFormat {
  dmy,
  mdy,
  ymd,
}
