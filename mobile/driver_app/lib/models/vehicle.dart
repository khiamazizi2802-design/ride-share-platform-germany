import 'package:freezed_annotation/freezed_annotation.dart';

part 'vehicle.freezed.dart';
part 'vehicle.g.dart';

@freezed
class Vehicle with _$Vehicle {
  const factory Vehicle({
    required String id,
    required String driverId,
    required String licensePlate,
    required String make,
    required String model,
    required int year,
    required String color,
    required String vehicleType,
    required int seatingCapacity,
    String? vin,
    DateTime? kbaVerificationDate,
    DateTime? tuvExpiryDate,
    DateTime? insuranceExpiryDate,
    required bool isVerified,
    required bool isActive,
    List<String>? documentUrls,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) = _Vehicle;

  factory Vehicle.fromJson(Map<String, dynamic> json) =>
      _$VehicleFromJson(json);
}

@freezed
class VehicleDocument with _$VehicleDocument {
  const factory VehicleDocument({
    required String id,
    required String vehicleId,
    required String documentType,
    required String documentUrl,
    required String status,
    DateTime? uploadedAt,
    DateTime? verifiedAt,
    String? verifiedBy,
    String? rejectionReason,
  }) = _VehicleDocument;

  factory VehicleDocument.fromJson(Map<String, dynamic> json) =>
      _$VehicleDocumentFromJson(json);
}

enum VehicleType {
  standard,
  comfort,
  premium,
  van,
  electric,
}

enum VehicleDocumentType {
  registrationCertificate,
  insurancePolicy,
  tuvCertificate,
  kbaApproval,
  vehiclePhoto,
}

extension VehicleTypeExtension on VehicleType {
  String get displayName {
    switch (this) {
      case VehicleType.standard:
        return 'Standard';
      case VehicleType.comfort:
        return 'Comfort';
      case VehicleType.premium:
        return 'Premium';
      case VehicleType.van:
        return 'Van';
      case VehicleType.electric:
        return 'Electric';
    }
  }

  String get description {
    switch (this) {
      case VehicleType.standard:
        return 'Economy class vehicle';
      case VehicleType.comfort:
        return 'Mid-range vehicle with extra comfort';
      case VehicleType.premium:
        return 'Luxury vehicle for premium rides';
      case VehicleType.van:
        return 'Spacious vehicle for groups up to 8';
      case VehicleType.electric:
        return 'Environmentally friendly electric vehicle';
    }
  }
}