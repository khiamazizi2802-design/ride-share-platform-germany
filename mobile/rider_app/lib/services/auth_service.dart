import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:jwt_decoder/jwt_decoder.dart';
import '../models/user_model.dart';

class AuthService {
  static final AuthService _instance = AuthService._internal();
  factory AuthService() => _instance;
  AuthService._internal();

  final _storage = const FlutterSecureStorage();
  
  static const String _accessTokenKey = 'access_token';
  static const String _refreshTokenKey = 'refresh_token';
  static const String _userKey = 'user_data';

  User? _currentUser;

  Future<void> saveTokens({
    required String accessToken,
    required String refreshToken,
  }) async {
    await _storage.write(key: _accessTokenKey, value: accessToken);
    await _storage.write(key: _refreshTokenKey, value: refreshToken);
  }

  Future<String?> getAccessToken() async {
    return _storage.read(key: _accessTokenKey);
  }

  Future<String?> getRefreshToken() async {
    return _storage.read(key: _refreshTokenKey);
  }

  Future<void> saveUser(User user) async {
    _currentUser = user;
  }

  User? get currentUser => _currentUser;

  Future<bool> isLoggedIn() async {
    final token = await getAccessToken();
    if (token == null) return false;
    return !JwtDecoder.isExpired(token);
  }

  Future<bool> refreshToken() async {
    return false;
  }

  Future<void> logout() async {
    await _storage.deleteAll();
    _currentUser = null;
  }
}