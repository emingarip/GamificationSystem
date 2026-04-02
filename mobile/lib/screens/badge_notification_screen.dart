import 'dart:async';

import 'package:confetti/confetti.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/user_provider.dart';
import '../widgets/confetti_animation.dart';

/// Full-screen badge notification overlay with confetti celebration
class BadgeNotificationScreen extends StatefulWidget {
  final String badgeId;
  final String badgeName;
  final String badgeDescription;
  final VoidCallback onDismiss;
  final Duration displayDuration;

  const BadgeNotificationScreen({
    super.key,
    required this.badgeId,
    required this.badgeName,
    required this.badgeDescription,
    required this.onDismiss,
    this.displayDuration = const Duration(seconds: 5),
  });

  @override
  State<BadgeNotificationScreen> createState() => _BadgeNotificationScreenState();
}

class _BadgeNotificationScreenState extends State<BadgeNotificationScreen>
    with SingleTickerProviderStateMixin {
  late final ConfettiController _confettiController;
  late final AnimationController _scaleController;
  late final Animation<double> _scaleAnimation;
  Timer? _dismissTimer;
  bool _isDismissed = false;

  @override
  void initState() {
    super.initState();
    
    // Initialize confetti controller
    _confettiController = ConfettiController(
      duration: const Duration(seconds: 3),
    );
    
    // Initialize scale animation for entrance effect
    _scaleController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 500),
    );
    
    _scaleAnimation = CurvedAnimation(
      parent: _scaleController,
      curve: Curves.elasticOut,
    );
    
    // Start animations
    _startAnimations();
  }

  void _startAnimations() async {
    // Play confetti if enabled in user preferences
    final userProvider = context.read<UserProvider>();
    if (userProvider.confettiEnabled) {
      await Future.delayed(const Duration(milliseconds: 300));
      _confettiController.play();
    }
    
    // Animate in the badge card
    _scaleController.forward();
    
    // Set up auto-dismiss timer
    _dismissTimer = Timer(widget.displayDuration, () {
      if (!_isDismissed) {
        _dismiss();
      }
    });
  }

  void _dismiss() async {
    if (_isDismissed) return;
    _isDismissed = true;
    
    // Cancel timer
    _dismissTimer?.cancel();
    
    // Animate out
    await _scaleController.reverse();
    
    // Call onDismiss callback
    widget.onDismiss();
  }

  void _onTap() {
    _dismiss();
  }

  @override
  void dispose() {
    _confettiController.dispose();
    _scaleController.dispose();
    _dismissTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final userProvider = context.watch<UserProvider>();
    
    return Material(
      color: Colors.black54,
      child: Stack(
        children: [
          // Confetti animation
          if (userProvider.confettiEnabled)
            Align(
              alignment: Alignment.topCenter,
              child: ConfettiAnimation(
                controller: _confettiController,
              ),
            ),
          
          // Badge notification card
          Center(
            child: ScaleTransition(
              scale: _scaleAnimation,
              child: GestureDetector(
                onTap: _onTap,
                child: Container(
                  margin: const EdgeInsets.all(32),
                  padding: const EdgeInsets.all(24),
                  decoration: BoxDecoration(
                    gradient: LinearGradient(
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                      colors: [
                        theme.colorScheme.primaryContainer,
                        theme.colorScheme.secondaryContainer,
                      ],
                    ),
                    borderRadius: BorderRadius.circular(24),
                    boxShadow: [
                      BoxShadow(
                        color: theme.colorScheme.primary.withOpacity(0.3),
                        blurRadius: 20,
                        offset: const Offset(0, 10),
                      ),
                    ],
                  ),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      // Badge icon
                      Container(
                        padding: const EdgeInsets.all(16),
                        decoration: BoxDecoration(
                          color: Colors.amber.withOpacity(0.2),
                          shape: BoxShape.circle,
                        ),
                        child: const Icon(
                          Icons.emoji_events,
                          size: 64,
                          color: Colors.amber,
                        ),
                      ),
                      
                      const SizedBox(height: 24),
                      
                      // Celebration text
                      Text(
                        '🎉 Badge Earned! 🎉',
                        style: theme.textTheme.headlineSmall?.copyWith(
                          fontWeight: FontWeight.bold,
                          color: theme.colorScheme.onPrimaryContainer,
                        ),
                      ),
                      
                      const SizedBox(height: 16),
                      
                      // Badge name
                      Text(
                        widget.badgeName,
                        style: theme.textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.bold,
                          color: theme.colorScheme.onPrimaryContainer,
                        ),
                        textAlign: TextAlign.center,
                      ),
                      
                      const SizedBox(height: 8),
                      
                      // Badge description
                      if (widget.badgeDescription.isNotEmpty)
                        Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 16),
                          child: Text(
                            widget.badgeDescription,
                            style: theme.textTheme.bodyMedium?.copyWith(
                              color: theme.colorScheme.onPrimaryContainer.withOpacity(0.8),
                            ),
                            textAlign: TextAlign.center,
                          ),
                        ),
                      
                      const SizedBox(height: 24),
                      
                      // Tap to dismiss hint
                      Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 16,
                          vertical: 8,
                        ),
                        decoration: BoxDecoration(
                          color: theme.colorScheme.onPrimaryContainer.withOpacity(0.1),
                          borderRadius: BorderRadius.circular(20),
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Icon(
                              Icons.touch_app,
                              size: 16,
                              color: theme.colorScheme.onPrimaryContainer.withOpacity(0.6),
                            ),
                            const SizedBox(width: 8),
                            Text(
                              'Tap anywhere to continue',
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.onPrimaryContainer.withOpacity(0.6),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// Mini badge notification for in-app display (non-fullscreen)
class BadgeNotificationCard extends StatelessWidget {
  final String badgeId;
  final String badgeName;
  final String badgeDescription;
  final VoidCallback? onTap;
  final bool showConfetti;

  const BadgeNotificationCard({
    super.key,
    required this.badgeId,
    required this.badgeName,
    required this.badgeDescription,
    this.onTap,
    this.showConfetti = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    
    return Card(
      elevation: 4,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Container(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              // Badge icon
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: Colors.amber.withOpacity(0.2),
                  shape: BoxShape.circle,
                ),
                child: const Icon(
                  Icons.emoji_events,
                  color: Colors.amber,
                  size: 32,
                ),
              ),
              
              const SizedBox(width: 16),
              
              // Badge info
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      badgeName,
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    if (badgeDescription.isNotEmpty)
                      Text(
                        badgeDescription,
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurface.withOpacity(0.6),
                        ),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                  ],
                ),
              ),
              
              // Arrow icon
              Icon(
                Icons.chevron_right,
                color: theme.colorScheme.onSurface.withOpacity(0.4),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// List tile for displaying badge in a list
class BadgeListTile extends StatelessWidget {
  final String badgeId;
  final String badgeName;
  final String badgeDescription;
  final DateTime? earnedAt;
  final VoidCallback? onTap;

  const BadgeListTile({
    super.key,
    required this.badgeId,
    required this.badgeName,
    required this.badgeDescription,
    this.earnedAt,
    this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    
    return ListTile(
      onTap: onTap,
      leading: CircleAvatar(
        backgroundColor: Colors.amber.withOpacity(0.2),
        child: const Icon(
          Icons.emoji_events,
          color: Colors.amber,
        ),
      ),
      title: Text(badgeName),
      subtitle: Text(
        badgeDescription,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      trailing: earnedAt != null
          ? Text(
              _formatDate(earnedAt!),
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurface.withOpacity(0.5),
              ),
            )
          : null,
    );
  }

  String _formatDate(DateTime date) {
    final now = DateTime.now();
    final difference = now.difference(date);
    
    if (difference.inDays == 0) {
      return 'Today';
    } else if (difference.inDays == 1) {
      return 'Yesterday';
    } else if (difference.inDays < 7) {
      return '${difference.inDays} days ago';
    } else {
      return '${date.day}/${date.month}/${date.year}';
    }
  }
}