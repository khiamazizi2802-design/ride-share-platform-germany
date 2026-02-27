import 'package:flutter/foundation.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../services/auth_service.dart';

// Events
sealed class AuthEvent {}

class AuthLoginRequested extends AuthEvent {
  final String email;
  final String password;
  AuthLoginRequested(this.email, this.password);
}

class AuthLogoutRequested extends AuthEvent {}

class AuthCheckRequested extends AuthEvent {}

// States
sealed class AuthState {}

class AuthInitial extends AuthState {}

class AuthLoading extends AuthState {}

class AuthAuthenticated extends AuthState {
  final Map<String, dynamic>? user;
  AuthAuthenticated({this.user});
}

class AuthUnauthenticated extends AuthState {
  final String? message;
  AuthUnauthenticated({this.message});
}

class AuthError extends AuthState {
  final String message;
  AuthError(this.message);
}

// Bloc
class AuthBloc extends Bloc<AuthEvent, AuthState> {
  final AuthService _authService = AuthService();

  AuthBloc() : super(AuthInitial());

  @override
  Stream<AuthState> mapEventToState(AuthEvent event) async* {
    if (event is AuthLoginRequested) {
      yield AuthLoading();
      try {
        final result = await _authService.login(event.email, event.password);
        if (result['success'] == true) {
          yield AuthAuthenticated(user: result['user']);
        } else {
          yield AuthUnauthenticated(message: result['message']);
        }
      } catch (e) {
        yield AuthError(e.toString());
      }
    }

    if (event is AuthLogoutRequested) {
      yield AuthLoading();
      await _authService.logout();
      yield AuthUnauthenticated();
    }

    if (event is AuthCheckRequested) {
      yield AuthLoading();
      final isLoggedIn = await _authService.isLoggedIn();
      if (isLoggedIn) {
        yield AuthAuthenticated();
      } else {
        yield AuthUnauthenticated();
      }
    }
  }
}
