import 'package:flutter_bloc/flutter_bloc.dart';

import '../services/api_service.dart';
import '../models/earnings.dart';

// Events
sealed class EarningsEvent {}

class EarningsLoadRequested extends EarningsEvent {
  final String driverId;
  final String? period;
  EarningsLoadRequested(this.driverId, {this.period});
}

class EarningsLoadHistory extends EarningsEvent {
  final String driverId;
  final DateTime startDate;
  final DateTime endDate;
  EarningsLoadHistory(this.driverId, this.startDate, this.endDate);
}

// States
sealed class EarningsState {}

class EarningsInitial extends EarningsState {}

class EarningsLoading extends EarningsState {}

class EarningsLoaded extends EarningsState {
  final EarningsSummary summary;
  final List<EarningsDetail>? history;
  EarningsLoaded(this.summary, {this.history});
}

class EarningsError extends EarningsState {
  final String message;
  EarningsError(this.message);
}

// Bloc
class EarningsBloc extends Bloc<EarningsEvent, EarningsState> {
  final ApiService _apiService = ApiService();

  EarningsBloc() : super(EarningsInitial());

  @override
  Stream<EarningsState> mapEventToState(EarningsEvent event) async* {
    if (event is EarningsLoadRequested) {
      yield EarningsLoading();
      try {
        final response = await _apiService.get(
          '/drivers/${event.driverId}/earnings',
          queryParameters: event.period != null ? {'period': event.period} : null,
        );
        final summary = EarningsSummary.fromJson(response.data);
        yield EarningsLoaded(summary);
      } catch (e) {
        yield EarningsError(e.toString());
      }
    }

    if (event is EarningsLoadHistory) {
      yield EarningsLoading();
      try {
        final response = await _apiService.get(
          '/drivers/${event.driverId}/earnings/history',
          queryParameters: {
            'startDate': event.startDate.toIso.8601String(),
            'endDate': event.endDate.toIso.8601String(),
          },
        );
        final history = (response.data as List)
            .map((e) => EarningsDetail.fromJson(e))
            .toList();
        final summary = state is EarningsLoaded
            ? (state as EarningsLoaded).summary
            : EarningsSummary(today: 0, week: 0, month: 0, total: 0);
        yield EarningsLoaded(summary, history: history);
      } catch (e) {
        yield EarningsError(e.toString());
      }
    }
  }
}
