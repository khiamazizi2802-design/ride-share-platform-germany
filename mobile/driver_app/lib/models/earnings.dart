import 'package:freezed_annotation/freezed_annotation.dart';

part 'earnings.freezed.dart';

enum EarningsPeriod {
  today,
  week,
  month,
  year,
}

@freezed
class EarningsSummary with _$EarningsSummary {
  const factory EarningsSummary() = _$EarningsSummary;

  String get driverId;
  double get totalEarnings;
  int get totalTrips;
  double get averageFare;
  double get totalDistanceKm;
  double get totalTimeMins;
  double get averageRating;
  double get todayEarnings;
  int get todayTrips;
  double get weekearnings;
  double get monthEarnings;
  double get pendingBalance;
  double get availableBalance;
  DateTime get lastUpdated;
}

enum PaymentStatus {
  pending,
  processing,
  completed,
  failed,
  refunded,
}

enum PaymentMethod {
  bankTransfer,
  paypal,
  stripe,
  cash,
}

@freezed
class Earnings item with _$EarningsItem {
  const factory EarningsItem() = _$EarningsItem;

  String get id;
  String get driverId;
  String get tripId;
  double get baseFare;
  double get timeFare;
  double get serviceFee;
  double get surgeFare;
  double get discount;
  double get totalFare;
  double get driverPayout;
  double get platformFee;
  double get taxFee;
  String get currency;
  String get status;
  DateTime get tripDate;
  DateTime get processedAt;
  DateTime get createdAt;
}

@freezed
class Payment with _$Payment {
  const factory Payment() = _$Payment;

  String get id;
  String get driverId;
  double get amount;
  String get currency;
  PaymentMethod get paymentMethod;
  PaymentStatus get status;
  String? get transactionId;
  String? payoutDates;
  DateTime? get processedAt;
  DateTime get createdAt;
}

enum WithdrawalStatus {
  pending,
  processing,
  completed,
  rejected,
  cancelled,
}

enum RejectionReason {
  insufficientFunds,
  invalidAccount,
  suspiciousActivity,
  accountVerificationRequired,
  other,
}

class Withdrawal Request {
  final String id;
  final String driverId;
  final double amount;
  final String currency;
  final WithdrawalStatus status;
  final PaymentMethod paymentMethod;
  final String? rejectionReason;
  final DateTime? processedAt;
  final DateTime createdAt;

  WithdrawalRequest({
    required this.id,
    required this.driverId,
    required this.amount,
    required this.currency,
    required this.status,
    required this.paymentMethod,
    this.rejectionReason,
    this.processedAt,
    required this.createdAt,
  });
}
