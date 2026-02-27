import 'package:flutter_bloc/flutter_bloc.dart';

import '../services/api_service.dart';
import '../models/driver.dart';

// Events
sealed class DriverEvent {}

class DriverLoadRequested extends DriverEvent {
  final String driverId;
  DriverLoadRequested(this.driverId);
}

class DriverUpdateAvailability extends DriverEvent {
  final bool isAvailable;
  DriverUpdateAvailability(this.isAvailable);
}

class DriverUpdateLocation extends DriverEvent {
  final double latitude;
  final double longitude;
  DriverUpdateLocation(this.latitude, this.longitude);
}

// States
sealed class DriverState {}

class DriverInitial extends DriverState {}

class DriverLoading extends DriverState {}

class DriverLoaded extends DriverState {
  final Driver driver;
  DriverLoaded(this.driver);
}

class DriverError extends DriverState {
  final String message;
  DriverError(this.message);
}

// Bloc
class DriverBloc extends Bloc<DriverEvent, DriverState> {
  final ApiService _apiService = ApiService();

  DriverBloc() : super(DriverInitial());

  @override
  Stream<DriverState> mapEventToState(DriverEvent event) async* {
    if (event is DriverLoadRequested) {
      yield DriverLoading();
      try {
        final response = await _apiService.get('/drivers/${event.driverId}');
        final driver = Driver.fromJson(response.data);
        yield DriverLoaded(driver);
      } catch (e) {
        yield DriverError(e.toString());
      }
    }

    if (event is DriverUpdateAvailability) {
      try {
        await _apiService.put('/drivers/availability', data: {
          'isAvailable': event.isAvailable,
        });
        if (state is DriverLoaded) {
          final current = (state as DriverLoaded).driver;
          yield DriverLoaded(current.copyWith(isAvailable: event.isAvailable));
        }
      } catch (e) {
        yield DriverError(e.toString());
      }
    }

    if (event is DriverUpdateLocation) {
      if (state is DriverLoaded) {
        final current = (state as DriverLoaded).driver;
        yield DriverLoaded(current.copyWith(
          currentLat: event.latitude,
          currentLng: event.longitude,
        ));
      }
    }
  }
}
