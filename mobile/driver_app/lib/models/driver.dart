import 'package:freezed_annotation/freezed_annotation.dart';

part 'driver.freezed.dart';

enum DriverStatus {
  offline,
  online,
  busy,
  offDuty,
}

@freezed
class Driver with _$Driver {
  const factory Driver() = _$Driver;

  String get id;
  String get userId;
  String get firstName;
  String get lastName;
  String get email;
  String? get phone;
  String? get profileImageUrl;
  DriverStatus get status;
  bool get isAvailable;
  bool get isVerified;
  bool get isDocumentsVerified;
  bool get isBackgroundChecked;
  String? get licenseNumber;
  DateTime? get licenseExpiryDate;
  String? get vehicleMake;
  String? get vehicleModel;
  String? get vehicleLicensePlate;
  String? get vehicleColor;
  int? get vehicleYear;
  double? get rating;
  int? get totalTrips;
  double? get earningsToDate;
  DateTime? get createdAt;
  DateTime? get updatedAt;

  String get fullName => '$firstName $lastName';

  bool get canReceiveRequests =>
      status == DriverStatus.online && isAvailable && isVerified;
}

@freezed
class DriverLocation with _$DriverLocation {
  const factory DriverLocation() = _$DriverLocation;

  String get driverId;
  double get latitude;
  double get longitude;
  double? get heading;
  double? get speed;
  DateTime get timestamp;
  bool get isAvailable;
}

@freezed
class DriverSettings with _$DriverSettings {
  const factory DriverSettings() = _$DriverSettings;

  String get driverId;
  bool get notificationsEnabled;
  bool get soundEnabled;
  bool get vibrationEnabled;
  bool get autoAcceptRequests;
  double get maxDistanceKm;
  double? get minimumFare;
  list<String> get preferredAreas;
  list<String> get preferredVehicleTypes;
  String? get preferredPaymentMethod;
}

enum DocumentType {
  license,
  vehicleRegistration,
  insurance,
  backgroundCheck,
  medicalCertificate,
}

enum DocumentStatus {
  pending,
  verified,
  rejected,
  expired,
}

@freezed
class DriverDocument with _$DriverDocument {
  const factory DriverDocument() = _$DriverDocument;

  String get id;
  String get driverId;
  DocumentType get type;
  String get documentNumber;
  DateTime? get expiryDate;
  String get documentImageUrl;
  DocumentStatus get status;
  String? get rejectionReason;
  DateTime get createdAt;
  DateTime get updatedAt;
}