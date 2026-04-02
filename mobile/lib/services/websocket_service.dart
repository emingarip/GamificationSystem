import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../providers/user_provider.dart';

/// WebSocket connection configuration
class WebSocketConfig {
  /// Default WebSocket URL - configurable via environment
  /// In production, this would come from environment variables or config file
  static const String defaultUrl = 'ws://localhost:8080/ws';
  
  /// Connection timeout in seconds
  static const int connectionTimeout = 10;
  
  /// Maximum reconnection attempts before giving up
  static const int maxReconnectAttempts = 5;
  
  /// Initial reconnection delay in seconds
  static const int initialReconnectDelay = 2;
  
  /// Maximum reconnection delay in seconds
  static const int maxReconnectDelay = 30;
}

/// WebSocket message types
enum WebSocketMessageType {
  badgeEarned,
  pointsUpdated,
  userStatsUpdated,
  streakUpdated,
  unknown,
}

/// Badge earned event data
class BadgeEarnedEvent {
  final String badgeId;
  final String badgeName;
  final String badgeDescription;
  final String badgeIcon;
  final DateTime earnedAt;
  final int pointsAwarded;

  BadgeEarnedEvent({
    required this.badgeId,
    required this.badgeName,
    required this.badgeDescription,
    this.badgeIcon = 'emoji_events',
    required this.earnedAt,
    required this.pointsAwarded,
  });

  factory BadgeEarnedEvent.fromJson(Map<String, dynamic> json) {
    return BadgeEarnedEvent(
      badgeId: json['badge_id'] ?? json['id'] ?? '',
      badgeName: json['badge_name'] ?? json['name'] ?? 'Unknown Badge',
      badgeDescription: json['badge_description'] ?? json['description'] ?? '',
      badgeIcon: json['badge_icon'] ?? 'emoji_events',
      earnedAt: json['earned_at'] != null
          ? DateTime.parse(json['earned_at'])
          : DateTime.now(),
      pointsAwarded: json['points_awarded'] ?? json['points'] ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'badge_id': badgeId,
      'badge_name': badgeName,
      'badge_description': badgeDescription,
      'badge_icon': badgeIcon,
      'earned_at': earnedAt.toIso8601String(),
      'points_awarded': pointsAwarded,
    };
  }
}

/// WebSocket service for real-time communication with the Go backend
/// Handles connection management, auto-reconnect, and message parsing
class WebSocketService extends ChangeNotifier {
  WebSocketChannel? _channel;
  final UserProvider userProvider;
  
  // Connection state
  bool _isConnected = false;
  bool _isConnecting = false;
  int _reconnectAttempts = 0;
  Timer? _reconnectTimer;
  Timer? _heartbeatTimer;
  StreamSubscription? _subscription;
  
  // Message handlers
  Function(String badgeId, String badgeName, String badgeDescription)? onBadgeEarned;
  Function(int newPoints)? onPointsUpdated;
  Function(Map<String, dynamic> userStats)? onUserStatsUpdated;
  Function(int streakDays)? onStreakUpdated;
  
  // Configuration
  String _serverUrl = WebSocketConfig.defaultUrl;

  WebSocketService({
    required this.userProvider,
    String? serverUrl,
  }) {
    if (serverUrl != null) {
      _serverUrl = serverUrl;
    }
  }

  /// Whether the WebSocket is currently connected
  bool get isConnected => _isConnected;
  
  /// Whether the WebSocket is currently connecting
  bool get isConnecting => _isConnecting;
  
  /// Current server URL
  String get serverUrl => _serverUrl;

  /// Set a custom server URL
  void setServerUrl(String url) {
    _serverUrl = url;
    // If connected, disconnect and reconnect with new URL
    if (_isConnected) {
      disconnect();
      connect();
    }
  }

  /// Connect to the WebSocket server
  Future<void> connect() async {
    if (_isConnected || _isConnecting) {
      return;
    }

    _isConnecting = true;
    notifyListeners();

    try {
      debugPrint('WebSocket: Connecting to $_serverUrl...');
      
      _channel = WebSocketChannel.connect(
        Uri.parse(_serverUrl),
        protocols: ['gamification-v1'],
      );

      // Set connection timeout
      final completer = Completer<void>();
      
      _subscription = _channel!.stream.listen(
        _onMessage,
        onError: _onError,
        onDone: _onDone,
      );

      // Wait for connection to be established
      await Future.delayed(const Duration(milliseconds: 500));
      
      _isConnected = true;
      _isConnecting = false;
      _reconnectAttempts = 0;
      
      debugPrint('WebSocket: Connected successfully');
      
      // Start heartbeat to keep connection alive
      _startHeartbeat();
      
      // Send initial connection message
      _sendMessage({
        'type': 'connect',
        'user_id': userProvider.userId,
        'token': userProvider.authToken,
      });
      
      notifyListeners();
    } catch (e) {
      debugPrint('WebSocket: Connection failed - $e');
      _isConnecting = false;
      _scheduleReconnect();
      notifyListeners();
    }
  }

  /// Disconnect from the WebSocket server
  Future<void> disconnect() async {
    _stopTimers();
    await _channel?.sink.close();
    _channel = null;
    _isConnected = false;
    _reconnectAttempts = 0;
    notifyListeners();
    debugPrint('WebSocket: Disconnected');
  }

  /// Attempt to reconnect to the server
  Future<void> reconnect() async {
    await disconnect();
    _reconnectAttempts = 0;
    await connect();
  }

  /// Handle incoming WebSocket messages
  void _onMessage(dynamic message) {
    try {
      final data = jsonDecode(message as String) as Map<String, dynamic>;
      final messageType = _parseMessageType(data['type'] ?? '');
      
      debugPrint('WebSocket: Received message type: ${data['type']}');
      
      switch (messageType) {
        case WebSocketMessageType.badgeEarned:
          _handleBadgeEarned(data);
          break;
        case WebSocketMessageType.pointsUpdated:
          _handlePointsUpdated(data);
          break;
        case WebSocketMessageType.userStatsUpdated:
          _handleUserStatsUpdated(data);
          break;
        case WebSocketMessageType.streakUpdated:
          _handleStreakUpdated(data);
          break;
        default:
          debugPrint('WebSocket: Unknown message type: ${data['type']}');
      }
    } catch (e) {
      debugPrint('WebSocket: Error parsing message - $e');
    }
  }

  /// Handle badge earned event
  void _handleBadgeEarned(Map<String, dynamic> data) {
    final event = BadgeEarnedEvent.fromJson(data['payload'] ?? data);
    
    debugPrint('WebSocket: Badge earned - ${event.badgeName}');
    
    // Update user provider
    userProvider.addBadge(event.badgeId, event.badgeName);
    userProvider.addPoints(event.pointsAwarded);
    
    // Trigger callback
    onBadgeEarned?.call(
      event.badgeId,
      event.badgeName,
      event.badgeDescription,
    );
  }

  /// Handle points update
  void _handlePointsUpdated(Map<String, dynamic> data) {
    final points = data['points'] as int? ?? 0;
    userProvider.setTotalPoints(points);
    onPointsUpdated?.call(points);
  }

  /// Handle user stats update
  void _handleUserStatsUpdated(Map<String, dynamic> data) {
    final stats = data['stats'] as Map<String, dynamic>? ?? {};
    userProvider.updateFromJson(stats);
    onUserStatsUpdated?.call(stats);
  }

  /// Handle streak update
  void _handleStreakUpdated(Map<String, dynamic> data) {
    final streak = data['streak'] as int? ?? 0;
    userProvider.setCurrentStreak(streak);
    onStreakUpdated?.call(streak);
  }

  /// Handle errors
  void _onError(dynamic error) {
    debugPrint('WebSocket: Error - $error');
    _isConnected = false;
    notifyListeners();
    _scheduleReconnect();
  }

  /// Handle connection closed
  void _onDone() {
    debugPrint('WebSocket: Connection closed');
    _isConnected = false;
    notifyListeners();
    _scheduleReconnect();
  }

  /// Schedule a reconnection attempt with exponential backoff
  void _scheduleReconnect() {
    if (_reconnectAttempts >= WebSocketConfig.maxReconnectAttempts) {
      debugPrint('WebSocket: Max reconnect attempts reached');
      return;
    }
    
    // Calculate delay with exponential backoff
    final delay = WebSocketConfig.initialReconnectDelay *
        (1 << _reconnectAttempts); // 2^n
    final actualDelay = delay.clamp(
      WebSocketConfig.initialReconnectDelay,
      WebSocketConfig.maxReconnectDelay,
    );
    
    debugPrint('WebSocket: Scheduling reconnect in $actualDelay seconds');
    _reconnectAttempts++;
    
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(Duration(seconds: actualDelay), () {
      connect();
    });
  }

  /// Send a message to the server
  void _sendMessage(Map<String, dynamic> message) {
    if (_channel != null && _isConnected) {
      _channel!.sink.add(jsonEncode(message));
    }
  }

  /// Send a raw message to the server
  void sendRawMessage(String message) {
    if (_channel != null && _isConnected) {
      _channel!.sink.add(message);
    }
  }

  /// Start heartbeat to keep connection alive
  void _startHeartbeat() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = Timer.periodic(
      const Duration(seconds: 30),
      (_) => _sendMessage({'type': 'ping'}),
    );
  }

  /// Stop all timers
  void _stopTimers() {
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    _heartbeatTimer?.cancel();
    _heartbeatTimer = null;
  }

  /// Parse message type from string
  WebSocketMessageType _parseMessageType(String type) {
    switch (type.toLowerCase()) {
      case 'badge_earned':
      case 'badgeearnedevent':
        return WebSocketMessageType.badgeEarned;
      case 'points_updated':
      case 'pointsupdatedevent':
        return WebSocketMessageType.pointsUpdated;
      case 'user_stats_updated':
      case 'userstatsupdatedevent':
        return WebSocketMessageType.userStatsUpdated;
      case 'streak_updated':
      case 'streakupdatedevent':
        return WebSocketMessageType.streakUpdated;
      default:
        return WebSocketMessageType.unknown;
    }
  }

  @override
  void dispose() {
    _stopTimers();
    _subscription?.cancel();
    _channel?.sink.close();
    super.dispose();
    debugPrint('WebSocket: Service disposed');
  }
}