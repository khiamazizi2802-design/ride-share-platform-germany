import 'package:freezed_annotation/freezed_annotation.dart';

part 'trip_request.freezed.dart';
part 'trip_request.g.dart';

@freezed
class TripRequest with _$TripRequest {
  const factory TripRequest({
    required String id,
    required String riderId,
    String? riderName,
    String? riderPhotoUrl,
    double? riderRating,
    required PickupLocation pickup,
    required DropoffLocation dropoff,
    required double estimatedDistance,
    required double estimatedDuration,
    required double estimatedPrice,
    String? vehicleType,
    List<String>? paymentMethods,
    String? notes,
    required DateTime requestedAt,
    DateTime? expiresAt,
    String? status,
    double? driverEarnings,
    double? platformFee,
  }) = _TripRequest;

  factory TripRequest.fromJson(Map<String, dynamic> json) =>
      _$TripRequestFromJson(json);
}

@freezed
class PickupLocation with _$PickupLocation {
  const factory PickupLocation({
    required double latitude,
    required double longitude,
    required String address,
    String? placeName,
    String? additionalInfo,
  }) = _PickupLocation;

  factory PickupLocation.fromJson(Map<String, dynamic> json) =>
      _$PickupLocationFromJson(json);
}

@Freezed
class DropoffLocation with _$DropoffLocation {
  const factory DropoffLocation({
    required double latitude,
    required double longitude,
    required String address,
    String? placeName,
    String? additionalInfo,
  }) = _DropoffLocation;

  factory DropoffLocation.fromJson(Map<String, dynamic> json) =>
      _$DropoffLocationFromJson(json);
}

enum TripRequestStatus {
  pending,
  accepted,
  declined,
  expired,
  cancelled,
}

extension TripRequestStatusExtension on TripRequestStatus {
  String get value {
    switch (this) {
      case TripRequestStatus.pending:
        return 'PENDING';
      case TripRequestStatus.accepted:
        return 'ACCEPTED';
      case TripRequestStatus.declined:
        return 'DECLINED';
      case TripRequestStatus.expired:
        return 'EXPIRED';
      case TripRequestStatus.cancelled:
        return 'CANCELLED';
    }
  }

  String get displayName {
    switch (this) {
      case TripRequestStatus.pending:
        return 'Pending';
      case TripRequestStatus.accepted:
        return 'Accepted';
      case TripRequestStatus.declined:
        return 'Declined';
      case TripRequestStatus.expired:
        return 'Expired';
      case TripRequestStatus.cancelled:
        return 'Cancelled';
    }
  }
}

@freezed
class TripAcceptanceResponse with _$TripAcceptanceResponse {
  const factory TripAcceptanceResponse({
    required bool success,
    required String tripId,
    String? message,
    TripRequest? tripRequest,
  }) = _TripAcceptanceResponse;

  factory TripAcceptanceResponse.fromJson(Map<String, dynamic> json) =>
      _$TripAcceptanceResponseFromJson(json);
}
