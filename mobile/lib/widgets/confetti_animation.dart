import 'dart:math';

import 'package:confetti/confetti.dart';
import 'package:flutter/material.dart';

/// Confetti animation widget for celebration effects
/// Uses the confetti package for particle-based celebrations
class ConfettiAnimation extends StatelessWidget {
  final ConfettiController controller;

  const ConfettiAnimation({
    super.key,
    required this.controller,
  });

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.topCenter,
      child: ConfettiWidget(
        confettiController: controller,
        blastDirectionality: BlastDirectionality.explosive,
        particleDrag: 0.05,
        emissionFrequency: 0.05,
        numberOfParticles: 20,
        gravity: 0.1,
        shouldLoop: false,
        colors: const [
          Colors.red,
          Colors.blue,
          Colors.green,
          Colors.yellow,
          Colors.pink,
          Colors.orange,
          Colors.purple,
          Colors.amber,
          Colors.teal,
          Colors.cyan,
        ],
        strokeWidth: 1,
        strokeColor: Colors.white,
      ),
    );
  }
}

/// Celebration confetti with customizable colors for badge events
class BadgeCelebrationConfetti extends StatelessWidget {
  final ConfettiController controller;
  final List<Color>? customColors;

  const BadgeCelebrationConfetti({
    super.key,
    required this.controller,
    this.customColors,
  });

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.topCenter,
      child: ConfettiWidget(
        confettiController: controller,
        blastDirectionality: BlastDirectionality.explosive,
        particleDrag: 0.05,
        emissionFrequency: 0.02,
        numberOfParticles: 30,
        gravity: 0.05,
        shouldLoop: false,
        colors: customColors ?? const [
          Colors.amber,
          Colors.orange,
          Colors.yellow,
          Colors.pink,
          Colors.purple,
        ],
        minimumSize: const Size(8, 8),
        maximumSize: const Size(15, 15),
        strokeWidth: 1,
        strokeColor: Colors.white,
      ),
    );
  }
}

/// Custom confetti blast for special celebrations
class SpecialCelebrationConfetti extends StatelessWidget {
  final ConfettiController controller;

  const SpecialCelebrationConfetti({
    super.key,
    required this.controller,
  });

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        // Main confetti blast
        Align(
          alignment: Alignment.topCenter,
          child: ConfettiWidget(
            confettiController: controller,
            blastDirectionality: BlastDirectionality.explosive,
            particleDrag: 0.02,
            emissionFrequency: 0.01,
            numberOfParticles: 50,
            gravity: 0.02,
            shouldLoop: false,
            colors: const [
              Colors.amber,
              Colors.yellow,
              Colors.orange,
            ],
            minimumSize: const Size(5, 10),
            maximumSize: const Size(10, 20),
            strokeWidth: 0,
          ),
        ),
        
        // Star confetti overlay
        Align(
          alignment: Alignment.topCenter,
          child: ConfettiWidget(
            confettiController: controller,
            blastDirectionality: BlastDirectionality.explosive,
            particleDrag: 0.05,
            emissionFrequency: 0.03,
            numberOfParticles: 20,
            gravity: 0.08,
            shouldLoop: false,
            colors: const [
              Colors.pink,
              Colors.purple,
              Colors.blue,
            ],
            minimumSize: const Size(8, 8),
            maximumSize: const Size(12, 12),
            strokeWidth: 1,
            strokeColor: Colors.white,
          ),
        ),
      ],
    );
  }
}

/// Animated badge icon with pulsing glow effect
class AnimatedBadgeIcon extends StatefulWidget {
  final IconData icon;
  final Color color;
  final double size;
  final Duration animationDuration;

  const AnimatedBadgeIcon({
    super.key,
    this.icon = Icons.emoji_events,
    this.color = Colors.amber,
    this.size = 64,
    this.animationDuration = const Duration(milliseconds: 1500),
  });

  @override
  State<AnimatedBadgeIcon> createState() => _AnimatedBadgeIconState();
}

class _AnimatedBadgeIconState extends State<AnimatedBadgeIcon>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _scaleAnimation;
  late final Animation<double> _glowAnimation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: widget.animationDuration,
    )..repeat(reverse: true);

    _scaleAnimation = Tween<double>(begin: 1.0, end: 1.1).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeInOut),
    );

    _glowAnimation = Tween<double>(begin: 0.3, end: 0.8).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeInOut),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, child) {
        return Transform.scale(
          scale: _scaleAnimation.value,
          child: Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: widget.color.withOpacity(_glowAnimation.value),
              shape: BoxShape.circle,
              boxShadow: [
                BoxShadow(
                  color: widget.color.withOpacity(_glowAnimation.value),
                  blurRadius: 20,
                  spreadRadius: 5,
                ),
              ],
            ),
            child: Icon(
              widget.icon,
              size: widget.size,
              color: Colors.white,
            ),
          ),
        );
      },
    );
  }
}

/// Simple particle burst effect without external dependency
class SimpleParticleBurst extends StatefulWidget {
  final bool isPlaying;
  final Widget child;

  const SimpleParticleBurst({
    super.key,
    required this.isPlaying,
    required this.child,
  });

  @override
  State<SimpleParticleBurst> createState() => _SimpleParticleBurstState();
}

class _SimpleParticleBurstState extends State<SimpleParticleBurst>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  final List<_Particle> _particles = [];
  final Random _random = Random();

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 2000),
    );

    if (widget.isPlaying) {
      _startParticles();
    }
  }

  @override
  void didUpdateWidget(SimpleParticleBurst oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.isPlaying && !oldWidget.isPlaying) {
      _startParticles();
    }
  }

  void _startParticles() {
    _particles.clear();
    for (int i = 0; i < 20; i++) {
      _particles.add(_Particle(
        angle: _random.nextDouble() * 2 * pi,
        speed: _random.nextDouble() * 200 + 100,
        color: _getRandomColor(),
        size: _random.nextDouble() * 8 + 4,
      ));
    }
    _controller.forward(from: 0);
  }

  Color _getRandomColor() {
    final colors = [
      Colors.red,
      Colors.blue,
      Colors.green,
      Colors.yellow,
      Colors.pink,
      Colors.orange,
      Colors.purple,
      Colors.amber,
    ];
    return colors[_random.nextInt(colors.length)];
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        widget.child,
        if (widget.isPlaying)
          AnimatedBuilder(
            animation: _controller,
            builder: (context, _) {
              return CustomPaint(
                painter: _ParticlePainter(
                  particles: _particles,
                  progress: _controller.value,
                ),
                size: Size.infinite,
              );
            },
          ),
      ],
    );
  }
}

class _Particle {
  final double angle;
  final double speed;
  final Color color;
  final double size;

  _Particle({
    required this.angle,
    required this.speed,
    required this.color,
    required this.size,
  });
}

class _ParticlePainter extends CustomPainter {
  final List<_Particle> particles;
  final double progress;

  _ParticlePainter({
    required this.particles,
    required this.progress,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    
    for (final particle in particles) {
      final distance = particle.speed * progress;
      final x = center.dx + cos(particle.angle) * distance;
      final y = center.dy + sin(particle.angle) * distance;
      
      final opacity = (1 - progress).clamp(0.0, 1.0);
      final paint = Paint()
        ..color = particle.color.withOpacity(opacity)
        ..style = PaintingStyle.fill;
      
      canvas.drawCircle(
        Offset(x, y),
        particle.size * (1 - progress * 0.5),
        paint,
      );
    }
  }

  @override
  bool shouldRepaint(covariant _ParticlePainter oldDelegate) {
    return oldDelegate.progress != progress;
  }
}