import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'api_service.dart';

class AuthService {
  static final AuthService _instance = AuthService._internal();
  factory AuthService() => _instance;
  AuthService._internal();

  final ApiService _apiService = ApiService();
  late FlutterSecureStorage _storage;

  Future<void> initialize() async {
    _storage = FlutterSecureStorage();
  }

  Future<Map<String, dynamic>> login(String email, String password) async {
    try {
      final response = await _apiService.post('/auth/login', data: {
        'email': email,
        'password': password,
        'role': 'DRIVER',
      });

      final data = jsonDecode(response.data);
      final token = data['token'];
      final user = data['user'];

      // Store token securely
      await _storage.write(key: 'auth_token', value: token);
      await _storage.write(key: 'user_id', value: user['id'].toString());
      await _storage.write(key: 'user_email', value: user['email']);
      await _storage.write(key: 'user_name', value: user['firstName'] + ' ' + user['lastName']);

      return {
        'success': true,
        'user': user,
        'token': token,
      };
    } catch (e) {
      if (e is ApiException) {
        return {
          'success': false,
          'message': e.message,
        };
      }
      return {
        'success': false,
        'message': 'Login failed. Please try again.',
      };
    }
  }

  Future<Map<String, dynamic>> register({
    required String firstName,
    required String lastName,
    required String email,
    required String password,
    required String phone,
    required String licenseNumber,
    required String vehicleRegistration,
    required String vehicleType,
  }) async {
    try {
      final response = await _apiService.post('/auth/register', data: {
        'firstName': firstName,
        'lastName': lastName,
        'email': email,
        'password': password,
        'phone': phone,
        'licenseNumber': licenseNumber,
        'vehicleRegistration': vehicleRegistration,
        'vehicleType': vehicleType,
        'role': 'DRIVER',
      });

      final data = jsonDecode(response.data);
      return {
        'success': true,
        'message': data['message'] ?? 'Registration successful',
        'userId': data['userId'],
      };
    } catch (e) {
      if (e is ApiException) {
        return {
          'success': false,
          'message': e.message,
        };
      }
      return {
        'success': false,
        'message': 'Registration failed. Please try again.',
      };
    }
  }

  Future<void> logout() async {
    await _storage.delete(key: 'auth_token');
    await _storage.delete(key: 'user_id');
    await _storage.delete(key: 'user_email');
    await _storage.delete(key: 'user_name');
  }

  Future<bool> isLoggedIn() async {
    final token = await _storage.read(key: 'auth_token');
    return token != null;
  }

  Future<String?> getToken() async {
    return await _storage.read(key: 'auth_token');
  }

  Future<String?> getUserId() async {
    return await _storage.read(key: 'user_id');
  }

  Future<String?> getUserEmail() async {
    return await _storage.read(key: 'user_email');
  }
}
