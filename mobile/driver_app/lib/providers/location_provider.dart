import 'package:flutter_bloc/flutter_bloc.dart';

import '../services/location_service.dart';

// Events
sealed class LocationEvent {}

class LocationStartTracking extends LocationEvent {
  final String driverId;
  LocationStartTracking(this.driverId);
}

class LocationStopTracking extends LocationEvent {}

class LocationUpdated extends LocationEvent {
  final double latitude;
  final double longitude;
  final double? heading;
  final double? speed;
  LocationUpdated(this.latitude, this.longitude, {this.heading, this.speed});
}

// States
sealed class LocationState {}

class LocationInitial extends LocationState {}

class LocationTracking extends LocationState {
  final double latitude;
  final double longitude;
  final double? heading;
  final double? speed;
  LocationTracking({
    required this.latitude,
    required this.longitude,
    this.heading,
    this.speed,
  });
}

class LocationError extends LocationState {
  final String message;
  LocationError(this.message);
}

// Bloc
class LocationBloc extends Bloc<LocationEvent, LocationState> {
  final LocationService _locationService = LocationService();

  LocationBloc() : super(LocationInitial());

  @override
  Stream<LocationState> mapEventToState(LocationEvent event) async* {
    if (event is LocationStartTracking) {
      try {
        await _locationService.startTracking(
          event.driverId,
          (location) {
            add(LocationUpdated(
              location.latitude,
              location.longitude,
              heading: location.heading,
              speed: location.speed,
            ));
          },
        );
      } catch (e) {
        yield LocationError(e.toString());
      }
    }

    if (event is LocationStopTracking) {
      _locationService.stopTracking();
      yield LocationInitial();
    }

    if (event is LocationUpdated) {
      yield LocationTracking(
        latitude: event.latitude,
        longitude: event.longitude,
        heading: event.heading,
        speed: event.speed,
      );
    }
  }
}
