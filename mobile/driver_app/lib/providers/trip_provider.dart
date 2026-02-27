import 'package:flutter_bloc/flutter_bloc.dart';

import '../services/api_service.dart';
import '../services/socket_service.dart';
import '../models/trip.dart';
import '../models/trip_request.dart';

// Events
sealed class TripEvent {}

class TripLoadActive extends TripEvent {
  final String driverId;
  TripLoadActive(this.driverId);
}

class TripAcceptRequested extends TripEvent {
  final String tripId;
  TripAcceptRequested(this.tripId);
}

class TripRejectRequested extends TripEvent {
  final String tripId;
  final String reason;
  TripRejectRequested(this.tripId, this.reason);
}

class TripUpdateStatus extends TripEvent {
  final String tripId;
  final String status;
  TripUpdateStatus(this.tripId, this.status);
}

class TripReceived extends TripEvent {
  final TripRequest request;
  TripReceived(this.request);
}

// States
sealed class TripState {}

class TripInitial extends TripState {}

class TripLoading extends TripState {}

class TripIdle extends TripState {}

class TripRequestPending extends TripState {
  final TripRequest request;
  TripRequestPending(this.request);
}

class TripActive extends TripState {
  final Trip trip;
  TripActive(this.trip);
}

class TripError extends TripState {
  final String message;
  TripError(this.message);
}

// Bloc
class TripBloc extends Bloc<TripEvent, TripState> {
  final ApiService _apiService = ApiService();
  final SocketService _socketService = SocketService();

  TripBloc() : super(TripInitial()) {
    // Listen to socket messages
    _socketService.onMessage.listen((message) {
      if (message['type'] == 'TRIP_REQUEST') {
        final request = TripRequest.fromJson(message['data']);
        add(TripReceived(request));
      }
    });
  }

  @override
  Stream<TripState> mapEventToState(TripEvent event) async* {
    if (event is TripLoadActive) {
      yield TripLoading();
      try {
        final response = await _apiService.get('/drivers/${event.driverId}/current-trip');
        if (response.data != null) {
          final trip = Trip.fromJson(response.data);
          yield TripActive(trip);
        } else {
          yield TripIdle();
        }
      } catch (e) {
        yield TripIdle();
      }
    }

    if (event is TripReceived) {
      yield TripRequestPending(event.request);
    }

    if (event is TripAcceptRequested) {
      yield TripLoading();
      try {
        _socketService.acceptTrip(event.tripId);
        // Wait for trip confirmation
        await Future.delayed(const Duration(seconds: 2));
        final response = await _apiService.get('/trips/${event.tripId}');
        final trip = Trip.fromJson(response.data);
        yield TripActive(trip);
      } catch (e) {
        yield TripError(e.toString());
      }
    }

    if (event is TripRejectRequested) {
      try {
        _socketService.rejectTrip(event.tripId, event.reason);
        yield TripIdle();
      } catch (e) {
        yield TripError(e.toString());
      }
    }

    if (event is TripUpdateStatus) {
      try {
        await _apiService.put('/trips/${event.tripId}/status', data: {
          'status': event.status,
        });
        if (state is TripActive) {
          final current = (state as TripActive).trip;
          if (event.status == 'COMPLETED' || event.status == 'CANCELLED') {
            yield TripIdle();
          } else {
            yield TripActive(current.copyWith(status: event.status));
          }
        }
      } catch (e) {
        yield TripError(e.toString());
      }
    }
  }
}
