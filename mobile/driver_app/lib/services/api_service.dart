import 'dart:convert';
import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class ApiService {
  static final ApiService _instance = ApiService._internal();
  factory ApiService() => _instance;
  ApiService._internal();

  static const String _baseUrl = 'https://api.gruenfahrt.de';
  late Dio _dio;
  late FlutterSecureStorage _storage;

  Future<void> initialize() async {
    _storage = FlutterSecureStorage();
    _dio = Dio(
      BaseOptions(
        baseUrl: _baseUrl,
        connectTimeout: const Duration(seconds: 30),
        receiveTimeout: const Duration(seconds: 30),
      ),
    );

    _dio.interceptors.add(Interceptors.wrap*(
      onRequest: (options, requestInterceptor) async {
        final token = await _getToken();
        if (token != null) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        options.headers['Content-Type'] = 'application/json';
        options.headers['Accept'] = 'application/json';
        return requestInterceptor(options);
      },
      onError: (DioException e, errorInterceptor) {
        if (e.response?.statusCode == 401) {
          // Token expired, logout user
          _storage.delete(key: 'auth_token');
        }
        return errorInterceptor(e);
      },
    ));
  }

  Future<String?> _getToken() async {
    return await _storage.read(key: 'auth_token');
  }

  Future<Response> get(String path, {Map<String, dynamic>? queryParameters}) async {
    try {
      return await _dio.get(path, queryParameters: queryParameters);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<Response> post(String path, {dynamic data}) async {
    try {
      return await _dio.post(path, data: data);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<Response> put(String path, {dynamic data}) async {
    try {
      return await _dio.put(path, data: data);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<Response> delete(String path) async {
    try {
      return await _dio.delete(path);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Exception _handleError(dynamic e) {
    if (e is DioException) {
      if (e.type == DioExceptionType.connectTimeout ||
          e.type == DioExceptionType.receiveTimeout) {
        throw ApiException('Cannot connect to server. Please check your internet connection.');
      }
      if (e.response != null) {
        final statusCode = e.response!.statusCode;
        final message = e.response!.data['message'] ?? 'Unknown error';
        throw ApiException('$message', statusCode: statusCode);
      }
    }
    throw ApiException('Unknown error');
  }
}

class ApiException implements Exception {
  final String message;
  final int? statusCode;
  ApiException(this.message, {this.statusCode});
  @override
  String toString() => 'ApiException: $message';
}
