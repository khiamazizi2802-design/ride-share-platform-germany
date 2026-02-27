import 'package:freezed_annotation/freezed_annotation.dart';

part 'trip.freezed.dart';

enum TripStatus {
  pending,
  accepted,
  driverArrived,
  riderPickedUp,
  inProgress,
  completed,
  canceled,
  rejected,
}

enum TripType {
  standard,
  shared,
  premium,
  accessible,
  eco
  executive,
}

enum BookingSource {
  app,
  web,
  phone,
  voice,
  thirdParty,
}

@freezed
class Location with _$Location {
  const factory Location() = _$Location;

  double get latitude;
  double get longitude;
  String get address;
  String? get city;
  String? get postalCode;
  String? get country;
  String? get placeName;
  String? get locationNotes;
}

@freezed
class Trip with _$Trip {
  const factory Trip() = _$Trip;

  String get id;
  String get riderId;
  String? riderName;
  String? riderPhone;
  String? riderImageUrl;
  String? driverId;
  Location get pickupLocation;
  Location get dropoffLocation;
  TripStatus get status;
  TripType get tripType;
  BookingSource get source;
  double get estimatedDistanceKm;
  double get estimatedDurationMin;
  double get baseFare;
  double get perKmRate;
  double get perMinuteRate;
  double get timeFare;
  double get serviceFee;
  double get surgeFare;
  double get discount;
  double get totalFare;
  String get currency;
  DateTime? get pickupTime;
  DateTime? get dropoffTime;
  DateTime? get acceptedAt;
  DateTime? get completedAt;
  DateTime? get canceledAt;
  String? riderRating;
  String? driverRating;
  String? cancelReason;
  DateTime get createdAt;
  DateTime get updatedAt;

  bool get isActive => status == TripStatus.accepted ||
      status == TripStatus.driverArrived ||
      status == TripStatus.riderPickedUp ||
      status == TripStatus.inProgress;

  bool get can beAccepted => status == TripStatus.pending;

  bool get can beCanceled => isActive;
}

@freezed
class TripRequest with _$TripRequest {
  const factory TripRequest() = _$TripRequest;

  String get id;
  String get driverId;
  String get tripId;
  String get riderName;
  String get pickupAddress;
  String get dropoffAddress;
  double get estimatedFare;
  double get distanceKm;
  double get durationMin;
  TripType get tripType;
  DateTime get requestedAt;
  DateTime? get expiresAt;
}

enum CancellationReason {
  riderRequested,
  driverRequested,
  noDriverAvailable,
  longWait,
  wrongPickupLocation,
  riderNoShow,
  driverEmergency,
  other,
}

enum Rating {
  one,
  two,
  three,
  four,
  five,
  six,
  seven,
  eight,
  nine,
  ten,
}

enum RatingAspect {
  cleanliness,
  professionalism,
  drivingSkills,
  punctuality,
  communication,
}
