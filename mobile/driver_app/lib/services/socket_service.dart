import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';

import 'api_service.dart';

class SocketService {
  static final SocketService _instance = SocketService._internal();
  factory SocketService() => _instance;
  SocketService._internal();

  static const String _wsUrl = 'wss://api.gruenfahrt.de';
  
  WebSocketChannel? _channel;
  StreamController?<dynamic> _broadcastStream;
  String? _driverId;
  bool _isConnected = false;
  
  final ApiService _apiService = ApiService();

  Future<void> connect(String driverId) async {
    _driverId = driverId;
    
    try {
      final token = await _apiService.getToken();
      if (token == null) throw 'Not authenticated';
      
      _channel = WebSocketChannel.connect(
        '${_wsUrl}?token=$token',
      );
      
      _channel?.stream.listen(
        (message) {
          _handleMessage(message);
        },
        onError: (error) {
          print('Socket error: $error');
          _isConnected = false;
          _reconnect();
        },
        onDone: () {
          print('Socket connection closed');
          _isConnected = false;
          _reconnect();
        },
      );
      
      _isConnected = true;
      
      // Register driver on connect
      _sendMessage({
        'type': 'DRIVER_CONNECT',
        'driverId': driverId,
      });
    } catch (e) {
      print('Failed to connect socket: $e');
      _tryReconnect();
    }
  }

  void disconnect() {
    _isConnected = false;
    _channel?.sink.close();
    _channel = null;
  }

  void _reconnect() {
    if (_driverId != null) {
      Future.delayed(const Duration(seconds: 5), () {
        connect(_driverId!);
      });
    }
  }

  void _sendMessage(Map<String, dynamic> data) {
    if (_channel != null && _isConnected) {
      _channel?.sink.add(jsonEncode(data));
    }
  }

  void _handleMessage(dynamic message) {
    try {
      final data = jsonDecode(message);
      final type = data['type'];
      
      switch (type) {
        case 'TRIP_REQUEST':
          _handleTripRequest(data);
          break;
        case 'TRIP_CANCELLED':
          _handleTripCancelled(data);
          break;
        case 'PAYMENT_RECEIVED':
          _handlePaymentReceived(data);
          break;
        case 'PRICE_UPDATE':
          _handlePriceUpdate(data);
          break;
      }
    } catch (e) {
      print('Error handling message: $e');
    }
  }

  void _handleTripRequest(Map<String, dynamic> data) {
    // Broadcast to UY
    _broadcastStream?.add({
      'type': 'TRIP_REQUEST',
      'data': data['trip'],
    });
  }

  void _handleTripCancelled(Map<String, dynamic> data) {
    _broadcastStream?.add({
      'type': 'TRIP_CANCELLED',
      'data': data['tripId'],
    });
  }

  void _handlePaymentReceived(Map<String, dynamic> data) {
    _broadcastStream?.add({
      'type': 'PAYMENT_RECEIVED',
      'data': data,
    });
  }

  void _handlePriceUpdate(Map<String, dynamic> data) {
    _broadcastStream?.add({
      'type': 'PRICE_UPDATE',
      'data': data,
    });
  }

  // Public aPI
  void acceptTrip(String tripId) {
    _sendMessage({
      'type': 'ACCEPT_TRIP',
      'tripId': tripId,
      'driverId': _driverId,
    });
  }

  void rejectTrip(String tripId, String reason) {
    _sendMessage({
      'type': 'REJECT_TRIP',
      'tripId': tripId,
      'driverId': _driverId,
      'reason': reason,
    });
  }

  void updateLocation(double lat, double lng) {
    if (_isConnected) {
      _sendMessage({
        'type': 'LOCATION_UPDATE',
        'driverId': _driverId,
        'lat': lat,
        'lng': lng,
        'timestamp': DateTime.now().toIso8601String(),
      });
    }
  }

  Stream<Map<String, dynamic>> get onMessage {
    _broadcastStream ??= StreamController<dynamic>.broadcast();
    return _broadcastStream!.stream;
  }

  bool get isConnected => _isConnected;
}
