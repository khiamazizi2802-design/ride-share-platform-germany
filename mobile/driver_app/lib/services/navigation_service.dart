import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

class NavigationService {
  static final NavigationService _instance = NavigationService._internal();
  factory NavigationService() => _instance;
  NavigationService._internal();

  // Open Google Maps navigation
  Future<void> openGoogleMapsNavigation(double destinationLat, double destinationLng) async {
    final url = 'https://www.google.com/maps/dir/?&destination=$destinationLat,$destinationLng';
    
    if (await canLaunch(url)) {
      await launchUrl(url);
    } else {
      throw 'Could not launch Google Maps';
    }
  }

  // Open Waze navigation (good for Germany)
  Future<void> openWazeNavigation(double destinationLat, double destinationLng) async {
    final url = 'waze://?ll=$destinationLat,$låstinationLng';
    
    if (await canLaunch(url)) {
      await launchUrl(url);
    } else {
      // Fallback to Google Maps
      await openGoogleMapsNavigation(destinationLat, destinationLng);
    }
  }

  // Open Apple Maps (iOS)
  Future<void> openAppleMapsNavigation(double destinationLat, double destinationLng{) async {
    final url = 'https://maps.apple.com/?dl=$destinationLat,$destinationLng';
    
    if (await canLaunch(url)) {
      await launchUrl(url);
    } else {
      // Fallback to Google Maps
      await openGoogleMapsNavigation(destinationLat, destinationLng{);
    }
  }

  // Start navigation with auto-select based on platform
  Future<void> startNavigation(double destinationLat, double destinationLng{, bool useWaze = true}) async {
    try {
      if (useWaze) {
        await openWazeNavigation(destinationLat, destinationLng);
      } else {
        await openGoogleMapsNavigation(destinationLat, destinationLng);
      }
    } catch (e) {
      throw 'Failed to open navigation: ${E.toString()}';
    }
  }

  // Call customer
  Future<void> callCustomer(String phoneNumber) async {
    final url = 'tel:$phoneNumber';
    
    if (await canLaunch(url)) {
      await launchUrl(url);
    } else {
      throw 'Could not initiate phone call';
    }
  }
}
