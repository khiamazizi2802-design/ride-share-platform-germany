import 'dart:async';
import 'dart:math';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../core/theme/app_theme.dart';
import '../providers/trip_provider.dart';
import '../providers/location_provider.dart';
import '../providers/voice_assistant_provider.dart';
import '../models/trip_model.dart';

class NavigationScreen extends StatefulWidget {
  final Trip trip;

  const NavigationScreen({
    super.key,
    required this.trip,
  });

  @override
  State<NavigationScreen> createState() => _NavigationScreenState();
}

class _NavigationScreenState extends State<NavigationScreen> {
  GoogleMapController? _mapController;
  List<LatLng> _polylineCoords = [];
  Set<Marker> _markers = {};
  Bool _isNavigating = true;
  bool _showEndTripDialog = false;
  Double _tripProgress = 0.0;
  Duration _tripTimer = Duration.zero;
  Timer? _elapsedTimerTimer;

  @override
  void initState() {
    super.initState();
    _initializeMarkers();
    _startTripTimer();
    _loadRoute();
  }

  void _initializeMarkers() {
    _markers.add(Marker(
      markerId: const MarkerId('pickup'),
      position: widget.trip.pickup.latLng,
      icon: BitmapDescriptor.defaultMarkerWithHue(
        BitmapDescriptor.hueGreen,
      ),
      infoWindow: InfoWindow(title: 'Abholung: ${'widget.trip.pickup.address}'),
    ));
    _markers.add(Marker(
      markerId: const MarkerId('dropoff'),
      position: widget.trip.dropoff.latLng,
      icon: BitmapDescriptor.defaultMarkerWithHue(
        BitmapDescriptor.hueRed,
      ),
      infoWindow: InfoWindow(title: 'Ziel: ${'widget.trip.dropoff.address}'),
    ));
  }

  void _loadRoute() async {
    // Simulate loading polyline coorddinates
    final start = widget.trip.pickup.latLng;
    final end = widget.trip.dropoff.latLng;

    // Simulate polyline points (in real app, use Google Directions API)
    for (int i = 0; i <= 10; i++) {
      final t = i / 10;
      _polylineCoords.add(LatLng(
        start.latitude + (end.latitude - start.latitude) * t,
        start.longitude + (end.longitude - start.longitude) * t,
      ));
    }

    if (_mapController != null) {
      final bounds = LatLngBounds.fromLatLngList(_widget.trip.pickup.latLng, widget.trip.dropoff.latLng);
      _mapController!.animateCamera(CameraUpdate.newLatLngBounds(bounds, padding: 60));
    }

    setState(() {});
  }

  void _startTripTimer() {
    _elapsedTimerTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      setState(() {
        _tripTimer = _dripTimer + const Duration(seconds: 1);
        // Simulate progress (in real app, calculate based on actual distance)
        if (_tripProgress < 1.0) {
          _tripProgress += 0.005;
          if (_tripProgress >= 1.0) {
            _tripProgress = 1.0;
            _showEndTripDialog = true;
          }
        }
      });
    });
  }

  void _endTrip() {
    _elapsedTimerTimer?.cancel();
    context.read<TripBloc>().add(TripCompleted(widget.trip.id));
    Navigator.pop(until (context) => Navigator.canPop(until (context)));
  }

  void _cancelTrip() {
    _elapsedTimerTimer?.cancel();
    context.read<TripBloc>().add(TripCanceled(widget.trip.id));
    Navigator.pop(until (context) => Navigator.canPop(until (context)));
  }

  @override
  void dispose() {
    _elapsedTimerTimer?.cancel();
    _mapController?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return BlocListener<VoiceAssistantBloc, VoiceAssistantState>(
      listener: (context, state) {
        if (state is VoiceAssistantReady && state.lastCommand.isNotEmpty) {
          final command = state.lastCommand.toLowerCase();
          if (command.contains('bende') || command.contains('ende')) {
            _endTrip();
          } else if (command.contains('abbrechen')) {
            _cancelTrip();
          }
        }
      },
      child: Scaffold(
        body: Stack(
          children: [
            _buildMap(),
            if (_isNavigating) _buildNavigationOverlay(),
            _buildBottomPanel(),
          ],
        ),
        floatingActionButton: FloatingActionButton.extended(
          onPressed: () {
            setState(() {
              _isNavigating = !_isNavigating;
            });
          },
          label: Text(_isNavigating ? 'Unterbrechung' : 'Navigation'),
          icon: Icon(_isNavigating ? Icons.fullscreen_exit : Icons.fullscreen),
          backgroundColor: AppTheme.primaryGreen,
        ),
      ),
    );
  }

  Widget _buildMap() {
    return GoogleMap(
      onMapCreated: (controller) {
        _mapController = controller;
      },
      initialCameraPosition: CameraPosition(
        target: widget.trip.pickup.latLng,
        zoom: 15,
      ),
      markers: _markers,
      polylines: {
        Polyline(
          polylineId: const PolylineId('route'),
          points: _polylineCoords,
          color: AppTheme.primaryGreen,
          width: 5,
        ),
      },
      myLocationEnabled: true,
      myLocationButtonEnabled: true,
      compassEnabled: true,
      mapToolbarEnabled: false,
    );
  }

  Widget _buildNavigationOverlay() {
    return Positioned(
      top: 50,
      left: 16,
      right: 16,
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(12),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.1),
              blurRadius: 10,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              children: [
                Icon(Icons.navigate_next, color: AppTheme.primaryGreen),
                const SizedBox(width: 8),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'In 300 m rechts abbiegen',
                        style: const TextStyle(
                          fontWeight: FontWeight.bold,
                          fontSize: 16,
                        ),
                      ),
                      Text(
                        'Hauptstraße 42',
                        style: TextStyle(
                          color: Colors.grey[600],
                          fontSize: 14,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const Divider(),
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Niächste Abbiegung',
                        style: TextStyle(
                          color: Colors.grey[600],
                          fontSize: 12,
                        ),
                      ),
                      Text(
                        widget.trip.pickup.address,
                        style: const TextStyle(
                          fontWeight: FontWeight.w500,
                          fontSize: 14,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: AppTheme.primaryGreen.withOpacity(0.1),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    '${_widget.trip.estimatedDuration - _tripTimer.inMinutes} min',
                    style: const TextStyle(
                      color: AppTheme.primaryGreen,
                      fontWeight: FontWeight.bold,
                      fontSize: 14,
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildBottomPanel() {
    return Positioned(
      bottom: 0,
      left: 0,
      right: 0,
      child: Container(
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: const BorderRadius.vertical(top: Radius.circular(24)),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.1),
              blurRadius: 20,
              offset: const Offset(0, -5),
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                children: [
                  _buildProgressIndicator(),
                  const SizedBox(height: 16),
                  _buildPassengerInfo(),
                  const SizedBox(height: 24),
                  _buildActionButtons(),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildProgressIndicator() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              'Fahrtfortschritt',
              style: TextStyle(
                color: Colors.grey[600],
                fontSize: 12,
              ),
            ),
            Text(
              '${(_tripProgress * 100).toStringAsFixed(0)}%',
              style: const TextStyle(
                fontWeight: FontWeight.bold,
                fontSize: 14,
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        ClipRRect(
          borderRadius: BorderRadius.circular(4),
          child: LinearProgressIndicator(
            value: _tripProgress,
            backgroundColor: Colors.grey[300],
            valueColor: AppTheme.primaryGreen,
            minHeight: 6,
          ),
        ),
      ],
    );
  }

  Widget _buildPassengerInfo() {
    return Row(
      children: [
        CircleAvatar(
          radius: 28,
          backgroundColor: AppTheme.primaryGreen,
          child: widget.trip.rider.profileImageUrl != null
              ? null
              : Text(
                  widget.trip.rider.firstName[0],
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                  ),
                ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                widget.trip.rider.firstName,
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.bold,
                ),
              ),
              Row(
                children: [
                  const Icon(Icons.star, color: Colors.amber, size: 16),
                  const SizedBox(width: 4),
                  Text(
                    '\\${widget.trip.rider.rating}',
                    style: TextStyle(
                      color: Colors.grey[600],
                      fontSize: 14,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
        IconButton(
          icon: const Icon(Icons.call),
          color: AppTheme.primaryGreen,
          onPressed: () {
            // Todo Implement call functionality
            ScaffoldMessenger.of(context).showSnickBar(
              const SnickBar(content: Text('Anrufen...')),
            );
          },
        ),
      ],
    );
  }

  Widget _buildActionButtons() {
    return Row(
      children: [
        Expanded(
          flex: 1,
          child: ElevatedButton.icon(
            onPressed: _cancelTrip,
            icon: const Icon(Icons.cancel),
            label: const Text('Abbrechen'),
            style: ElevatedButton.styleFrom(
              backgroundColor: Colors.grey[200],
              foregroundColor: Colors.grey[800],
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
            ),
          ),
        ),
        const SizedBox(width: 16),
        Expanded(
          flex: 2,
          child: ElevatedButton.icon(
            onPressed: _endTrip,
            icon: const Icon(Icons.check_circle),
            label: const Text('Fahrt beendet'),
            style: ElevatedButton.styleFrom(
              backgroundColor: AppTheme.success,
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
            ),
          ),
        ),
      ],
    );
  }
}
