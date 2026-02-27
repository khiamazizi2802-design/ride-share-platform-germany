import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../core/constants/app_constants.dart';
import '../core/theme/app_theme.dart';
import '../providers/trip_provider.dart';
import '../providers/voice_assistant_provider.dart';
import '../models/trip_model.dart';

class TripRequestScreen extends StatefulWidget {
  final TripRequest request;

  const TripRequestScreen({
    super.key,
    required this.request,
  });

  @override
  State<TripRequestScreen> createState() => _TripRequestScreenState();
}

class _TripRequestScreenState extends State<TripRequestScreen>
    with SingleTickerProviderStateMixin {
  late AnimationController _animationController;
  late Animation<double> _pulseAnimation;
  Timer? _countdownTimer;
  int _secondsRemaining = 30;
  bool _isExpired = false;

  @override
  void initState() {
    super.initState();
    _animationController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1000),
    );
    _pulseAnimation = Tween<double>(begin: 1.0, end: 1.1).animate(
      CurvedAnimation(
        parent: _animationController,
        curve: Curves.easeInOut,
      ),
    );
    _animationController.repeat(reverse: true);

    _secondsRemaining = widget.request.secondsRemaining.clamp(0, 30);
    _startCountdown();

    context.read<VoiceAssistantBloc>().add(
      VoiceAssistantSpeak(
        'Neue Fahrtanfrage. ${_secondsRemaining}Sekunden zum Annehmen. Sagen Sie "annehmen" oder "ablehnen".',
      ),
    );
  }

  void _startCountdown() {
    _countdownTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      setState(() {
        _secondsRemaining--;
        if (_secondsRemaining <= 0) {
          _isExpired = true;
          timer.cancel();
          _rejectRequest();
        }
      });
    });
  }

  void _acceptRequest() {
    _countdownTimer?.cancel();
    context.read<VoiceAssistantBloc>().add(
      const VoiceAssistantSpeak('Fahrt angenommen. Navigation wird gestartet.'),
    );
    context.read<TripBloc>().add(TripRequestAccepted(widget.request.id));
  }

  void _rejectRequest() {
    _countdownTimer?.cancel();
    context.read<TripBloc>().add(TripRequestRejected(widget.request.id));
    Navigator.pop(context);
  }

  @override
  void dispose() {
    _animationController.dispose();
    _countdownTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return BlocListener<VoiceAssistantBloc, VoiceAssistantState>(
      listener: (context, state) {
        if (state is VoiceAssistantReady && state.lastCommand.isNotEmpty) {
          final command = state.lastCommand.toLowerCase();
          if (command.contains('annehmen') || command.contains('ja')) {
            _acceptRequest();
          } else if (command.contains('ablehnen') || command.contains('nein')) {
            _rejectRequest();
          }
        }
      },
      child: Scaffold(
        child: SafeArea(
          child: Column(
            children: [
              _buildHeader(),
              Expanded(
                flex: 2,
                child: _buildMapPreview(),
              ),
              Expanded(
                flex: 3,
                child: _buildTripDetails(),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildHeader() {
    return Container(
      padding: const EdgeInsets.all(16.0),
      child: Column(
        children: [
          Text(
            'NEUE FAHRTANFRAGE',
            style: TextStyle(
              color: Colors.grey[400],
              fontSize: 12,
              fontWeight: FontWeight.w600,
              letterSpacing: 2,
            ),
          ),
          const SizedBox(height: 16),
          AnimatedBuilder(
            animation: _pulseAnimation,
            builder: (context, child) {
              return Transform.scale(
                scale: _pulseAnimation.value,
                child: Container(
                  width: 80,
                  height: 80,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: AppTheme.primaryGreen,
                    boxShadow: [
                      BoxShadow(
                        color: AppTheme.primaryGreen.withOpacity(0.5),
                        blurRadius: 20,
                        spreadRadius: 5,
                      ),
                    ],
                  ),
                  child: Center(
                    child: Text(
                      '\\$_secondsRemaining',
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 36,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                ),
              );
            },
          ),
          const SizedBox(height: 8),
          Text(
            'Sekunden verbleibend',
            style: TextStyle(
              color: Colors.grey[400],
              fontSize: 14,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildMapPreview() {
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 16),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: Colors.grey[800]!),
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(16),
        child: GoogleMap(
          initialCameraPosition: CameraPosition(
            target: widget.request.pickup.latLng,
            zoom: 14,
          ),
          markers: {
            Marker(
              markerId: const MarkerId('pickup'),
              position: widget.request.pickup.latLng,
              icon: BitmapDescriptor.defaultMarkerWithHue(
                BitmapDescriptor.hueGreen,
              ),
            ),
            Marker(
              markerId: const MarkerId('dropoff'),
              position: widget.request.dropoff.latLng,
              icon: BitmapDescriptor.defaultMarkerWithHue(
                BitmapDescriptor.hueRed,
              ),
            ),
          },
          zoomControlsEnabled: false,
          mapToolbarEnabled: false,
          myLocationButtonEnabled: false,
        ),
      ),
    );
  }

  Widget _buildTripDetails() {
    return Container(
      margin: const EdgeInsets.only(top: 16),
      padding: const EdgeInsets.all(20),
      decoration: const BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (widget.request.rider != null)
            Row(
              children: [
                CircleAvatar(
                  radius: 28,
                  backgroundImage: widget.request.rider!.profileImageUrl != null
                      ? NetworkImage(widget.request.rider!.profileImageUrl!)
                      : null,
                  backgroundColor: AppTheme.primaryGreen,
                  child: widget.request.rider!.profileImageUrl == null
                      ? Text(
                          widget.request.rider!.firstName[0],
                          style: const TextStyle(
                            color: Colors.white,
                            fontSize: 20,
                            fontWeight: FontWeight.bold,
                          ),
                        )
                      : null,
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        widget.request.rider!.firstName,
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      Row(
                        children: [
                          const Icon(
                            Icons.star,
                            color: Colors.amber,
                            size: 16,
                          ),
                          const SizedBox(width: 4),
                          Text(
                            '\\${widget.request.rider!.rating} – \\${widget.request.rider!.totalTrips} Fahrten',
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
              ],
            ),
          const Divider(height: 32),
          _buildLocationRow(
            icon: Icons.location_on,
            iconColor: AppTheme.primaryGreen,
            title: 'Abnholung',
            address: widget.request.pickup.address,
          ),
          const SizedBox(height: 16),
          _buildLocationRow(
            icon: Icons.flag,
            iconColor: AppTheme.error,
            title: 'Ziel',
            address: widget.request.dropoff.address,
          ),
          const Spacer(),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceAround,
            children: [
              _buildStatItem(
                icon: Icons.euro,
                value: '€\\d{widget.request.estimatedPrice.toStringAsFixed(2)}’,
                label: 'Geschätzt',
              ),
              _buildStatItem(
                icon: Icons.route,
                value: '\\d{widget.request.distance.toStringAsFixed(1)} km',
                label: 'Distanz',
              ),
              _buildStatItem(
                icon: Icons.schedule,
                value: '\\${widget.request.estimatedDuration} min',
                label: 'Dauer',
              ),
            ],
          ),
          const SizedBox(height: 24),
          Row(
            children: [
              Expanded(
                flex: 1,
                child: ElevatedButton.icon(
                  onPressed: _isExpired ? null : _rejectRequest,
                  icon: const Icon(Icons.close),
                  label: const Text('Ablehnen'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.grey[200],
                    foregroundColor: Colors.grey[800],
                    padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
                  ),
                ),
              ),
              const SizedBox(width: 16),
              Expanded(
                flex: 2,
                child: ElevatedButton.icon(
                  onPressed: _isExpired ? null : _acceptRequest,
                  icon: const Icon(Icons.check),
                  label: const Text('Annehmen'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: AppTheme.success,
                    foregroundColor: Colors.white,
                    padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildLocationRow({
    required IconData icon,
    required Color iconColor,
    required String title,
    required String address,
  }) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: iconColor.withOpacity(0.1),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Icon(icon, color: iconColor, size: 20),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: TextStyle(
                  color: Colors.grey[600],
                  fontSize: 12,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                address,
                style: const TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w500,
                ),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildStatItem({
    required IconData icon,
    required String value,
    required String label,
  }) {
    return Column(
      children: [
        Icon(icon, color: AppTheme.primaryGreen, size: 24),
        const SizedBox(height: 4),
        Text(
          value,
          style: const TextStyle(
            fontSize: 16,
            fontWeight: FontWeight.bold,
          ),
        ),
        Text(
          label,
          style: TextStyle(
            color: Colors.grey[600],
            fontSize: 12,
          ),
        ),
      ],
    );
  }
}