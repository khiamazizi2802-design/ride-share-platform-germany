class ApiEndpoints {
  // Auth
  static const String login = '/auth/login';
  static const String register = '/auth/register';
  static const String refreshToken = '/auth/refresh';
  static const String logout = '/auth/logout';
  
  // User
  static const String userProfile = '/users/me';
  static const String updateProfile = '/users/me';
  static const String deleteAccount = '/users/me';
  
  // Rides
  static const String requestRide = '/rides';
  static const String getRide = '/rides/';
  static const String cancelRide = '/rides/';
  static const String rideHistory = '/rides/history';
  
  // Booking
  static const String estimateFare = '/pricing/estimate';
  static const String getNearbyDrivers = '/drivers/nearby';
  
  // Payments
  static const String paymentMethods = '/payments/methods';
  static const String addPaymentMethod = '/payments/methods';
  static const String removePaymentMethod = '/payments/methods/';
  static const String processPayment = '/payments/process';
  
  // Voice Assistant
  static const String voiceCommand = '/voice/command';
  
  // Compliance
  static const String dataRequest = '/compliance/data-requests';
  static const String consentWithdrawal = '/compliance/consent';
}