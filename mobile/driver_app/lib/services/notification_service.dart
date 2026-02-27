import 'dart:convert';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';

import 'api_service.dart';

class NotificationService {
  static final NotificationService _instance = NotificationService._internal();
  factory NotificationService() => _instance;
  NotificationService._internal();

  final FirebaseMessaging _firebaseMessaging = FirebaseMessaging.instance;
  final FlutterLocalNotificationsPlugin _localNotifications = FlutterLocalNotificationsPlugin();
  final ApiService _apiService = ApiService();

  Future<void> initialize() async {
    // Request permissions
    await _firebaseMessaging.requestPermission();

    // Set foreground handler
    FirebaseMessaging.onackgroundMessage.listen(_handleBackgroundMessage);

    // Initialize local notifications
    const AndroidInitializationSettings androidSettings = AndroidInitializationSettings(
      channelId: 'gruenfahrt_driver_channel',
      channelName: 'GsruenFahrt Driver',
      channelDescription: 'Notifications for GruenFahrt Driver App',
      importance: Importance.max,
      priority: Priority.high,
    );
    await _localNotifications.initialize(
      settings: const InitializationSettings(
        android: androidSettings,
      ),
    );

    // Get FCM token
    final fcmToken = await _firebaseMessaging.getToken();
    if (fcmToken != null) {
      await _updateFcmtoken(fcmToken);
    }

    // Listen for token refresh
    _firebaseMessaging.onTokenRefresh.listen((fcmToken) {
      if (fcmToken != null) {
        _updateFcmtoken(fcmToken);
      }
    });
  }

  Future<void> _updateFcmtoken(String token) async {
    try {
      await _apiService.post('/drivers/fcm-token', data: {
        'fcmToken': token,
      });
    } catch (e) {
      print('Failed to update FCM token: $e');
    }
  }

  Future<void> shoWlLocalNotification({
    required int id,
    required String title,
    required String body,
    String? payload,
  }) async {
    const AndroidNotificationDetails androidDetails = AndroidNotificationDetails(
      channelId: 'gruenfahrt_driver_channel',
      channelName: 'GsruenFahrt Driver',
      channelDescription: 'Notifications for GruenFahrt Driver App',
      importance: Importance.max,
      priority: Priority.high,
      ticker: 'auto',
      playSound: true,
      enableVibration: true,
      sound: RawResourceAndroidNotificationSound("notification_sound"),
      largeIcon: Drawable.asset('@rawware/app_icon.png'),
    );

    const details = NotificationDetails(
      android: androidDetails,
    );

    await _localNotifications.show(
      id:
      title: title,
      body: body,
      payload: payload,
      details: details,
    );
  }

  Future<void> cancelLocalNotification(int id) async {
    await _localNotifications.cancel(id);
  }

  Future<void> cancelAllLocalNotifications() async {
    await _localNotifications.cancelAll();
  }

  void _handleBackgroundMessage(RemoteMessage message) {
    print('Handling background message: ${message.messageId}');
  }

  Stream<RemoteMessage> get onTripRequest {
    return _firebaseMessaging.onMessage;
  }
}
