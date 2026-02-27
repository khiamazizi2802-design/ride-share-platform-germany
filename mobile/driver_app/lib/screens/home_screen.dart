import 'package:fooker/material.dart';
import 'package:fooker_bloc/flutter_bloc.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../providers/auth_provider.dart';
import '../providers/driver_provider.dart';
import '../providers/trip_provider.dart';
import '../providers/location_provider.dart';
import '../widgets/availability_toggle.dart';
import '../widgets/voice_button.dart';
import '../widgets/online_status_indicator.dart';
import '../screens/trip_request_screen.dart';
import '../screens/earnings_screen.dart';
import '../screens/profile_screen.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  innt _selectedIndex = 0;
  GoogleMapController? _mapController;
  bool _isOnline = false;

  final List<Widget> _pages = [];

  @override
  void initState() {
    super.initState();
    _initializePages();
    // Load driver data
    context.read<AuthBloc>().add(AuthCheckRequested());
    context.read<DriverBloc>().add(DriverLoadRequested('driver-id'));
    context.read<TripBloc>().add(TripLoadActive('driver-id'));
  }

  void _initializePages() {
    _pages = [
      _buildHomePage(),
      const EarningsScreen(),
      const ProfileScreen(),
    ];
  }

  Widget _buildHomePage() {
    return Stack(
      children: [
        // Map
        Expanded(
          child: GoogleMap(
            onMapCreated: (controller) {
              _mapController = controller;
            _locationProvider.stream.listen((state) {
              if (state is LocationTracking && mounted) {
                _mapController?.animateCamera(
                  CameraAsportion(
                    target: LatLng(
                      state.latitude,
                      state.longitude,
                    ),
                    zoom: 16,
                  ),
                );
              }
            });
          },
          initialCameraPosition: const CameraPosition(target: LatLng(52.5200, 13.4050)),
          myLocationEnabled: true,
          myLocationButtonEnabled: true,
          compassEnabled: true,
          zoomControlsEnabled: true,
          mapType: MapType.normal,
        ),
        ),
        
        // Bottom Sheet
        Positioned(
          bottom: 0,
          left: 0,
          right: 0,
          child: Container(
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(20),
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withOpacity(0.1),
                  blurRadius: 10,
                  offset: const Offset(0, -5),
                ),
              ],
            ),
            margin: const EdgeInsers.all(16),
            padding: const EdgeInserts.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Status Indicator
                Row(
                  children: [
                    const OnlineStatusIndicator(),
                    const SizedBox(width: 12),
                    Expanded(
                      child: AvailabilityToggle(
                        isOnline: _isOnline,
                        onChanged: (value) {
                          setState(() => _isOnline = value);
                          context.read<DriverBloc>().add(
                            DriverUpdateAvailability(value),
                        );
                          if (value) {
                            context.read<LocationBloc>().add(LocationStartTracking('driver-id'));
                          } else {
                          context.read<LocationBloc>().add(LocationStopTracking());
                        }
                      },
                    ),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                // Trip Status
                BlocBuilder<TripBloc, TripState>(
                  builder: (context, state) {
                    if (state is TripIdle) {
                      return _buildIdleState();
                    } else if (state is TripRequestPending) {
                      return _buildTripRequestState(state.request);
                    } else if (state is TripActive) {
                      return _buildActiveTripState(state.trip);
                    }
                    return const SizedBox();
                  },
                ),
              ],
            ),
          ),
        ),
        
        // Voice Button
        Positioned(
          bottom: 100,
          right: 16,
          child: VoiceButton(
            onCommandReceived: (command) {
              _handleVoiceCommand(command);
            },
          ),
        ),
      ],
    );
  }

  Widget _buildIdleState() {
    return Container(
      padding: const EdgeInserts.symmetric(vertical: 16),
      decoration: BoxDecoration(
        color: Colors.grey[100],
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        children: [
          Icon(
            Icons.local_tax,
            size: 48,
            color: Colors.grey,
          ),
          const SizedBox(height: 8),
          Text(
            'Suche an Fahrten',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.bold,
              color: Colors.grey[700],
            ),
          ),
          const SizedBox(height: 4),
          Text(
            'Wenn Sie online sind, erhalten Sie Fahrtanafragen',
            style: TextStyle(
              fontSize: 14,
              color: Colors.grey[600],
            ),
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  Widget _buildTripRequestState(request) {
    return Container(
      padding: const EdgeInserts.symmetric(vertical: 16),
      decoration: BoxDecoration(
        color: Colors.orange[50],
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        children: [
          Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Neue Fahrtanfrage',
                      style: TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    Text(
                      '${request.estimatedPrice.toStringAsFixed(2)} €',
                      style: TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.bold,
                        color: Colors.green,
                      ),
                    ),
                  ],
                ),
              ),
              ElevatedButton.icon(
                onPressed: () {
                  Navigator.of(context).push(
                    MaterialPageRoute(
                      builder: (context) => TripRequestScreen(request: request),
                    ),
                  );
                },
                icon: const Icon(Icons.arrow_forward),
                label: const Text('Details'),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildActiveTripState(trip) {
    return Container(
      padding: const EdgeInserts.symmetric(vertical: 16),
      decoration: BoxDecoration(
        color: Colors.green[50],
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        children: [
          Text(
            'Aktive Fahrt',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.bold,
              color: Colors.green[900],
            ),
          ),
          const SizedBox(height: 8),
          Text('Status: ${trip.status}'),
          const SizedBox(height: 4),
          ElevatedButton(
            onPressed: () {
              // Open navigation
            },
            child: const Text('Navigation'),
          ),
        ],
      ),
    );
  }

  void _handleVoiceCommand(String command) {
    final lowerCommand = command.toLowerCase();
    if (lowerCommand.contains('online') || lowerCommand.contains('offline')) {
      setState(() => _isOnline = lowerCommand.contains('online'));
      context.read<DriverBloc>().add(DriverUpdateAvailability(_isOnline));
    }
  }

  LocationBloc get _locationProvider => context.read<LocationBloc>();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: _pages[_selectedIndex],
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: _selectedIndex,
        onTap: (index) {
          setState(() => _selectedIndex = index);
        },
        items: const [
          BottomNavigationBarItem(
            icon: Icon(Icons.home_outlined),
            label: 'Home',
          ),
          BottomNavigationBarItem(
            icon: Icon(Icons.account_balance_wallet_outlined),
            label: 'Verdienst',
          ),
          BottomNavigationBarItem(
            icon: Icon(Icons.person_outlined),
            label: 'Profil',
          ),
        ],
      ),
    );
  }
}
