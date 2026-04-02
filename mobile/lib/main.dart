import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'providers/user_provider.dart';
import 'services/websocket_service.dart';
import 'screens/badge_notification_screen.dart';

void main() {
  runApp(const GamificationApp());
}

class GamificationApp extends StatelessWidget {
  const GamificationApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => UserProvider()),
        Provider<WebSocketService>(
          create: (context) => WebSocketService(
            userProvider: context.read<UserProvider>(),
          ),
          dispose: (_, service) => service.dispose(),
        ),
      ],
      child: MaterialApp(
        title: 'Gamification Platform',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          useMaterial3: true,
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF6366F1),
            brightness: Brightness.light,
          ),
        ),
        darkTheme: ThemeData(
          useMaterial3: true,
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF6366F1),
            brightness: Brightness.dark,
          ),
        ),
        themeMode: ThemeMode.system,
        home: const HomeScreen(),
      ),
    );
  }
}

/// Main home screen that displays the app and handles badge notifications
class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  late final WebSocketService _webSocketService;
  bool _showBadgeNotification = false;
  String _currentBadgeId = '';
  String _currentBadgeName = '';
  String _currentBadgeDescription = '';

  @override
  void initState() {
    super.initState();
    // Initialize WebSocket service after build
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _webSocketService = context.read<WebSocketService>();
      _webSocketService.connect();
      _webSocketService.onBadgeEarned = _onBadgeEarned;
    });
  }

  void _onBadgeEarned(String badgeId, String badgeName, String badgeDescription) {
    setState(() {
      _showBadgeNotification = true;
      _currentBadgeId = badgeId;
      _currentBadgeName = badgeName;
      _currentBadgeDescription = badgeDescription;
    });
  }

  void _dismissBadgeNotification() {
    setState(() {
      _showBadgeNotification = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    final userProvider = context.watch<UserProvider>();
    
    return Scaffold(
      appBar: AppBar(
        title: const Text('Gamification Platform'),
        centerTitle: true,
        elevation: 0,
        actions: [
          IconButton(
            icon: const Icon(Icons.settings),
            onPressed: _showSettings,
          ),
        ],
      ),
      body: Stack(
        children: [
          // Main content
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // User profile card
                _buildProfileCard(userProvider),
                const SizedBox(height: 24),
                
                // Stats section
                _buildStatsSection(userProvider),
                const SizedBox(height: 24),
                
                // Recent badges
                _buildRecentBadges(userProvider),
                const SizedBox(height: 24),
                
                // Connection status
                _buildConnectionStatus(),
              ],
            ),
          ),
          
          // Badge notification overlay
          if (_showBadgeNotification)
            BadgeNotificationScreen(
              badgeId: _currentBadgeId,
              badgeName: _currentBadgeName,
              badgeDescription: _currentBadgeDescription,
              onDismiss: _dismissBadgeNotification,
            ),
        ],
      ),
    );
  }

  Widget _buildProfileCard(UserProvider userProvider) {
    return Card(
      elevation: 2,
      child: Padding(
        padding: const EdgeInsets.all(16.0),
        child: Row(
          children: [
            CircleAvatar(
              radius: 30,
              backgroundColor: Theme.of(context).colorScheme.primaryContainer,
              child: Icon(
                Icons.person,
                size: 30,
                color: Theme.of(context).colorScheme.onPrimaryContainer,
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    userProvider.userName ?? 'Guest User',
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                  Text(
                    'Level ${userProvider.level}',
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      color: Theme.of(context).colorScheme.primary,
                    ),
                  ),
                ],
              ),
            ),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(
                  '${userProvider.totalPoints}',
                  style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.bold,
                    color: Theme.of(context).colorScheme.primary,
                  ),
                ),
                Text(
                  'points',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildStatsSection(UserProvider userProvider) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Your Progress',
          style: Theme.of(context).textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: _buildStatCard(
                icon: Icons.emoji_events,
                label: 'Badges',
                value: '${userProvider.earnedBadges.length}',
                color: Colors.amber,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: _buildStatCard(
                icon: Icons.local_fire_department,
                label: 'Streak',
                value: '${userProvider.currentStreak} days',
                color: Colors.orange,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: _buildStatCard(
                icon: Icons.trending_up,
                label: 'Rank',
                value: '#${userProvider.rank}',
                color: Colors.green,
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildStatCard({
    required IconData icon,
    required String label,
    required String value,
    required Color color,
  }) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12.0),
        child: Column(
          children: [
            Icon(icon, color: color, size: 28),
            const SizedBox(height: 8),
            Text(
              value,
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
              ),
            ),
            Text(
              label,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildRecentBadges(UserProvider userProvider) {
    final badges = userProvider.earnedBadges.take(5).toList();
    
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Recent Badges',
          style: Theme.of(context).textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
        ),
        const SizedBox(height: 12),
        if (badges.isEmpty)
          Card(
            child: Padding(
              padding: const EdgeInsets.all(24.0),
              child: Center(
                child: Column(
                  children: [
                    Icon(
                      Icons.emoji_events_outlined,
                      size: 48,
                      color: Theme.of(context).colorScheme.outline,
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'No badges earned yet',
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        color: Theme.of(context).colorScheme.outline,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          )
        else
          SizedBox(
            height: 100,
            child: ListView.builder(
              scrollDirection: Axis.horizontal,
              itemCount: badges.length,
              itemBuilder: (context, index) {
                final badge = badges[index];
                return Padding(
                  padding: const EdgeInsets.only(right: 12),
                  child: Column(
                    children: [
                      CircleAvatar(
                        radius: 30,
                        backgroundColor: Colors.amber.withOpacity(0.2),
                        child: const Icon(
                          Icons.emoji_events,
                          color: Colors.amber,
                          size: 28,
                        ),
                      ),
                      const SizedBox(height: 4),
                      SizedBox(
                        width: 60,
                        child: Text(
                          badge,
                          textAlign: TextAlign.center,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: Theme.of(context).textTheme.bodySmall,
                        ),
                      ),
                    ],
                  ),
                );
              },
            ),
          ),
      ],
    );
  }

  Widget _buildConnectionStatus() {
    return Consumer<WebSocketService>(
      builder: (context, wsService, child) {
        final isConnected = wsService.isConnected;
        return Card(
          color: isConnected
              ? Colors.green.withOpacity(0.1)
              : Colors.red.withOpacity(0.1),
          child: Padding(
            padding: const EdgeInsets.all(12.0),
            child: Row(
              children: [
                Icon(
                  isConnected ? Icons.cloud_done : Icons.cloud_off,
                  color: isConnected ? Colors.green : Colors.red,
                ),
                const SizedBox(width: 12),
                Text(
                  isConnected
                      ? 'Connected to server'
                      : 'Disconnected - Reconnecting...',
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
                const Spacer(),
                if (!isConnected)
                  SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: Theme.of(context).colorScheme.primary,
                    ),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }

  void _showSettings() {
    showModalBottomSheet(
      context: context,
      builder: (context) => const SettingsSheet(),
    );
  }
}

/// Settings bottom sheet for user preferences
class SettingsSheet extends StatelessWidget {
  const SettingsSheet({super.key});

  @override
  Widget build(BuildContext context) {
    final userProvider = context.watch<UserProvider>();
    
    return Container(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Settings',
            style: Theme.of(context).textTheme.headlineSmall,
          ),
          const SizedBox(height: 24),
          SwitchListTile(
            title: const Text('Badge Notifications'),
            subtitle: const Text('Show notifications when badges are earned'),
            value: userProvider.notificationsEnabled,
            onChanged: (value) {
              userProvider.setNotificationsEnabled(value);
            },
          ),
          SwitchListTile(
            title: const Text('Sound Effects'),
            subtitle: const Text('Play sounds on badge earned'),
            value: userProvider.soundEnabled,
            onChanged: (value) {
              userProvider.setSoundEnabled(value);
            },
          ),
          SwitchListTile(
            title: const Text('Confetti Animation'),
            subtitle: const Text('Show confetti celebration'),
            value: userProvider.confettiEnabled,
            onChanged: (value) {
              userProvider.setConfettiEnabled(value);
            },
          ),
          const SizedBox(height: 24),
          ListTile(
            leading: const Icon(Icons.refresh),
            title: const Text('Reconnect WebSocket'),
            onTap: () {
              context.read<WebSocketService>().reconnect();
            },
          ),
          ListTile(
            leading: const Icon(Icons.info_outline),
            title: const Text('About'),
            onTap: () {
              showAboutDialog(
                context: context,
                applicationName: 'Gamification Platform',
                applicationVersion: '1.0.0',
                applicationLegalese: '© 2024 Gamification System',
              );
            },
          ),
        ],
      ),
    );
  }
}