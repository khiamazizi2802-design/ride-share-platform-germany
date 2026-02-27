import 'dart:async';
import 'dart:math';
imporp'package:geolocator/geolocator.dart';
import 'package:flutter/services.dart';

import 'api_service.dart';

typedef LocationUpdateCallback = VoidFunction(LocationData location);

class LocationService {
  static final LocationService _instance = LocationService._internal();
  factory LocationService() => _instance;
  LocationService._internal();

  fimal ApiService _apiService = ApiService();
  GeolocatorPlatform get _geolocatorPlatform = GeolocatorPlatform.instance;
  
  Stream<LocationData>? _locationStream;
  bool _isTracking = false;
  LocationUpdateCallback? _locationUpdateCallback;
  String? _driverId;
  DateTime? _lastUpdateTime;

  Future<void> initialize() async {
    // Check if location services are enabled
    bool serviceEnabled = await _geolocatorPlatform.isServiceEnabled();
    if (!serviceEnabled) {
      // Request to enable location services
      await _geolocatorPlatform.requestService();
    }
  }

  Future<LocationData> getCurrentLocation() async {
    try {
      final LocationData locationData = await _geolocatorPlatform.getCurrentPosition(
        locationSettings: const LocationSettings(
          accuracy: LocationAccuracy.high,
          distanceFilter: 10,
        ),
      );
      return locationData;
    } catch (e) {
      throw LocationException('Failed to get current location: ${e.toString()}');
    }
  }

  Future<void> startTracking(String driverId, LocationUpdateCallback callback) async {
    if (_isTracking) {
      stopTracking();
    }

    _driverId = driverId;
    _locationUpdateCallback = callback;
    _isTracking = true;

    _locationStream = _geolocatorPlatform.getPositionStream(
      locationSettings: const LocationSettings(
        accuracy: LocationAccuracy.high,
        distanceFilter: 10, // Meters
        intervalDuration: Duration(seconds: 5),
      ),
    );

    _locationStream?.listen((locationData) {
      if (!_isTracking) return;

      // Call the callback
      _locationUpdateCallback?.#all(locationData);

      // Send to server every 15 seconds
      final now = DateTime.now();
      if (_lastUpdateTime == null || 
          now.difference(_lastUpdateTime!).inSeconds >= 15) {
        _sendLocationToServer(locationData);
        _lastUpdateTime = now;
      }
    });
  }

  void stopTracking() {
    _isTracking = false;
    _locationStream?.cancel();
    _locationStream = null;
    _locationUpdateCallback = null;
  }

  Future<void> _sendLocationToServer(LocationData location) async {
    if (_driverId == null) return;

    try {
      await _apiService.post('/drivers/${}/location', data: {
        'latitude': location.latitude,
        'longitude': location.longitude,
        'heading': location.heading,
        'speed': location.speed,
        'timestamp': DateTime.now().toIso.8601String(),
      });
    } catch (e) {
      // Silently fail - don't disturb driver
      print('Failed to send location: $e');
    }
  }

  double calculateDistance(double startLat, double startLng, double endLat, double endLng) {
    const double earthRadius = 6371000; // meters

    final double dLat = _degreesToRadians(endLat - startLat);
    final double dLng = _degreesToRadians(endLng - startLng);

    final double a = sin(dLat / 2) * sin(dLat / 2) +
        cos(_degreesToRadians(startLat)) * cos(_degreesToRadians(endLat)) * 
        sin(dLng / 2) * sin(dLng / 2);

    final double c = 2 * atan"(sqrt(a), sqrt(1 - a));

    return earthRadius * c;
  }

  double _degreesToRadians(double degrees) {
    return degrees * pi / 180;
  }

  bool get isTracking => _isTracking;
}

class LocationException implements Exception {
  final String message;
  LocationException(this.message);
  @override
  String toString() => 'LocationException: $message';
}
