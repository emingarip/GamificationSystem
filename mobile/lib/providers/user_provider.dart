import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// User data model
class User {
  final String id;
  final String name;
  final String? email;
  final String? avatarUrl;
  final int level;
  final int totalPoints;
  final int currentStreak;
  final int rank;
  final List<String> earnedBadges;
  final DateTime? lastActive;

  const User({
    required this.id,
    required this.name,
    this.email,
    this.avatarUrl,
    this.level = 1,
    this.totalPoints = 0,
    this.currentStreak = 0,
    this.rank = 0,
    this.earnedBadges = const [],
    this.lastActive,
  });

  User copyWith({
    String? id,
    String? name,
    String? email,
    String? avatarUrl,
    int? level,
    int? totalPoints,
    int? currentStreak,
    int? rank,
    List<String>? earnedBadges,
    DateTime? lastActive,
  }) {
    return User(
      id: id ?? this.id,
      name: name ?? this.name,
      email: email ?? this.email,
      avatarUrl: avatarUrl ?? this.avatarUrl,
      level: level ?? this.level,
      totalPoints: totalPoints ?? this.totalPoints,
      currentStreak: currentStreak ?? this.currentStreak,
      rank: rank ?? this.rank,
      earnedBadges: earnedBadges ?? this.earnedBadges,
      lastActive: lastActive ?? this.lastActive,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'email': email,
      'avatar_url': avatarUrl,
      'level': level,
      'total_points': totalPoints,
      'current_streak': currentStreak,
      'rank': rank,
      'earned_badges': earnedBadges,
      'last_active': lastActive?.toIso8601String(),
    };
  }

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: json['id'] ?? '',
      name: json['name'] ?? 'Guest',
      email: json['email'],
      avatarUrl: json['avatar_url'],
      level: json['level'] ?? 1,
      totalPoints: json['total_points'] ?? 0,
      currentStreak: json['current_streak'] ?? 0,
      rank: json['rank'] ?? 0,
      earnedBadges: List<String>.from(json['earned_badges'] ?? []),
      lastActive: json['last_active'] != null
          ? DateTime.parse(json['last_active'])
          : null,
    );
  }
}

/// User provider for state management
/// Handles user data, badges, points, and preferences
class UserProvider extends ChangeNotifier {
  // User data
  String _userId = '';
  String _userName = 'Guest User';
  String? _email;
  String? _avatarUrl;
  int _level = 1;
  int _totalPoints = 0;
  int _currentStreak = 0;
  int _rank = 0;
  List<String> _earnedBadges = [];
  DateTime? _lastActive;
  String? _authToken;

  // User preferences (stored locally)
  bool _notificationsEnabled = true;
  bool _soundEnabled = true;
  bool _confettiEnabled = true;
  String _serverUrl = 'ws://localhost:8080/ws';

  // Shared preferences instance
  SharedPreferences? _prefs;

  UserProvider() {
    _initPreferences();
  }

  // Getters
  String get userId => _userId;
  String get userName => _userName;
  String? get email => _email;
  String? get avatarUrl => _avatarUrl;
  int get level => _level;
  int get totalPoints => _totalPoints;
  int get currentStreak => _currentStreak;
  int get rank => _rank;
  List<String> get earnedBadges => List.unmodifiable(_earnedBadges);
  DateTime? get lastActive => _lastActive;
  String? get authToken => _authToken;
  bool get notificationsEnabled => _notificationsEnabled;
  bool get soundEnabled => _soundEnabled;
  bool get confettiEnabled => _confettiEnabled;
  String get serverUrl => _serverUrl;

  /// Initialize shared preferences
  Future<void> _initPreferences() async {
    _prefs = await SharedPreferences.getInstance();
    _loadPreferences();
  }

  /// Load preferences from local storage
  void _loadPreferences() {
    if (_prefs == null) return;

    _notificationsEnabled = _prefs!.getBool('notifications_enabled') ?? true;
    _soundEnabled = _prefs!.getBool('sound_enabled') ?? true;
    _confettiEnabled = _prefs!.getBool('confetti_enabled') ?? true;
    _serverUrl = _prefs!.getString('server_url') ?? 'ws://localhost:8080/ws';
    
    // Load user data
    _userId = _prefs!.getString('user_id') ?? '';
    _userName = _prefs!.getString('user_name') ?? 'Guest User';
    _level = _prefs!.getInt('level') ?? 1;
    _totalPoints = _prefs!.getInt('total_points') ?? 0;
    _currentStreak = _prefs!.getInt('current_streak') ?? 0;
    _rank = _prefs!.getInt('rank') ?? 0;
    _earnedBadges = _prefs!.getStringList('earned_badges') ?? [];
    _authToken = _prefs!.getString('auth_token');
    
    notifyListeners();
  }

  /// Set user ID
  void setUserId(String id) {
    _userId = id;
    _prefs?.setString('user_id', id);
    notifyListeners();
  }

  /// Set user name
  void setUserName(String name) {
    _userName = name;
    _prefs?.setString('user_name', name);
    notifyListeners();
  }

  /// Set email
  void setEmail(String? email) {
    _email = email;
    if (email != null) {
      _prefs?.setString('email', email);
    }
    notifyListeners();
  }

  /// Set avatar URL
  void setAvatarUrl(String? url) {
    _avatarUrl = url;
    if (url != null) {
      _prefs?.setString('avatar_url', url);
    }
    notifyListeners();
  }

  /// Set authentication token
  void setAuthToken(String? token) {
    _authToken = token;
    if (token != null) {
      _prefs?.setString('auth_token', token);
    } else {
      _prefs?.remove('auth_token');
    }
    notifyListeners();
  }

  /// Set user level
  void setLevel(int level) {
    _level = level;
    _prefs?.setInt('level', level);
    notifyListeners();
  }

  /// Set total points
  void setTotalPoints(int points) {
    _totalPoints = points;
    _prefs?.setInt('total_points', points);
    
    // Update level based on points
    _updateLevelFromPoints(points);
    notifyListeners();
  }

  /// Add points to total
  void addPoints(int points) {
    setTotalPoints(_totalPoints + points);
  }

  /// Update level based on points
  void _updateLevelFromPoints(int points) {
    // Simple level calculation: level = sqrt(points / 100) + 1
    final newLevel = (points > 0) ? ((points / 100).sqrt().floor() + 1) : 1;
    if (newLevel != _level) {
      _level = newLevel;
      _prefs?.setInt('level', _level);
    }
  }

  /// Set current streak
  void setCurrentStreak(int streak) {
    _currentStreak = streak;
    _prefs?.setInt('current_streak', streak);
    notifyListeners();
  }

  /// Set user rank
  void setRank(int rank) {
    _rank = rank;
    _prefs?.setInt('rank', rank);
    notifyListeners();
  }

  /// Add a badge to user's collection
  void addBadge(String badgeId, String badgeName) {
    if (!_earnedBadges.contains(badgeId)) {
      _earnedBadges.add(badgeId);
      _earnedBadges.add(badgeName);
      _prefs?.setStringList('earned_badges', _earnedBadges);
      notifyListeners();
    }
  }

  /// Remove a badge from user's collection
  void removeBadge(String badgeId) {
    _earnedBadges.remove(badgeId);
    _prefs?.setStringList('earned_badges', _earnedBadges);
    notifyListeners();
  }

  /// Check if user has a specific badge
  bool hasBadge(String badgeId) {
    return _earnedBadges.contains(badgeId);
  }

  /// Set last active time
  void setLastActive(DateTime time) {
    _lastActive = time;
    notifyListeners();
  }

  /// Update notification preference
  void setNotificationsEnabled(bool enabled) {
    _notificationsEnabled = enabled;
    _prefs?.setBool('notifications_enabled', enabled);
    notifyListeners();
  }

  /// Update sound preference
  void setSoundEnabled(bool enabled) {
    _soundEnabled = enabled;
    _prefs?.setBool('sound_enabled', enabled);
    notifyListeners();
  }

  /// Update confetti preference
  void setConfettiEnabled(bool enabled) {
    _confettiEnabled = enabled;
    _prefs?.setBool('confetti_enabled', enabled);
    notifyListeners();
  }

  /// Set server URL
  void setServerUrl(String url) {
    _serverUrl = url;
    _prefs?.setString('server_url', url);
    notifyListeners();
  }

  /// Update user from JSON data
  void updateFromJson(Map<String, dynamic> json) {
    if (json.containsKey('level')) setLevel(json['level']);
    if (json.containsKey('total_points')) setTotalPoints(json['total_points']);
    if (json.containsKey('current_streak')) setCurrentStreak(json['current_streak']);
    if (json.containsKey('rank')) setRank(json['rank']);
    if (json.containsKey('name')) setUserName(json['name']);
    if (json.containsKey('email')) setEmail(json['email']);
    if (json.containsKey('avatar_url')) setAvatarUrl(json['avatar_url']);
    if (json.containsKey('badges')) {
      final badges = json['badges'] as List;
      for (final badge in badges) {
        if (badge is Map) {
          addBadge(badge['id'] ?? '', badge['name'] ?? '');
        }
      }
    }
    notifyListeners();
  }

  /// Load user from storage
  Future<void> loadUser() async {
    await _initPreferences();
  }

  /// Clear all user data (logout)
  Future<void> clearUser() async {
    _userId = '';
    _userName = 'Guest User';
    _email = null;
    _avatarUrl = null;
    _level = 1;
    _totalPoints = 0;
    _currentStreak = 0;
    _rank = 0;
    _earnedBadges = [];
    _lastActive = null;
    _authToken = null;

    // Clear preferences
    await _prefs?.remove('user_id');
    await _prefs?.remove('user_name');
    await _prefs?.remove('email');
    await _prefs?.remove('avatar_url');
    await _prefs?.remove('level');
    await _prefs?.remove('total_points');
    await _prefs?.remove('current_streak');
    await _prefs?.remove('rank');
    await _prefs?.remove('earned_badges');
    await _prefs?.remove('auth_token');

    notifyListeners();
  }

  /// Get current user object
  User getUser() {
    return User(
      id: _userId,
      name: _userName,
      email: _email,
      avatarUrl: _avatarUrl,
      level: _level,
      totalPoints: _totalPoints,
      currentStreak: _currentStreak,
      rank: _rank,
      earnedBadges: _earnedBadges,
      lastActive: _lastActive,
    );
  }

  /// Set user from User object
  void setUser(User user) {
    _userId = user.id;
    _userName = user.name;
    _email = user.email;
    _avatarUrl = user.avatarUrl;
    _level = user.level;
    _totalPoints = user.totalPoints;
    _currentStreak = user.currentStreak;
    _rank = user.rank;
    _earnedBadges = List.from(user.earnedBadges);
    _lastActive = user.lastActive;

    // Save to preferences
    _prefs?.setString('user_id', _userId);
    _prefs?.setString('user_name', _userName);
    if (_email != null) _prefs?.setString('email', _email!);
    if (_avatarUrl != null) _prefs?.setString('avatar_url', _avatarUrl!);
    _prefs?.setInt('level', _level);
    _prefs?.setInt('total_points', _totalPoints);
    _prefs?.setInt('current_streak', _currentStreak);
    _prefs?.setInt('rank', _rank);
    _prefs?.setStringList('earned_badges', _earnedBadges);

    notifyListeners();
  }
}

// Extension for sqrt calculation
extension on num {
  double sqrt() {
    if (this < 0) return 0;
    return this.toDouble().sqrt();
  }
}

extension on double {
  double sqrt() {
    if (this < 0) return 0;
    if (this == 0) return 0;
    double x = this;
    double y = (x + 1) / 2;
    while (y < x) {
      x = y;
      y = (x + this / x) / 2;
    }
    return x;
  }
}