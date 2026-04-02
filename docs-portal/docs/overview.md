---
sidebar_position: 1
slug: /
---

# Overview

Welcome to the Gamification API documentation. This platform provides comprehensive documentation for the AI-Native Gamification System with Knowledge Graph integration.

## What is Gamification API?

The Gamification API is a powerful system that enables you to:

- **Manage Users**: Track and manage user profiles, points, and levels
- **Create Rules**: Define gamification rules that automatically award points and badges
- **Process Events**: Handle game events (goals, cards, fouls, etc.) and trigger rule evaluations
- **Award Badges**: Automatically or manually assign badges to users
- **Analytics**: Get comprehensive analytics on user activities and system performance
- **Leaderboards**: Track top performers with customizable leaderboards

## Architecture

The system is built on modern technologies:

- **Go** - High-performance backend API
- **Redis** - Fast caching and leaderboard storage
- **Neo4j** - Knowledge graph for complex user relationships
- **Kafka** - Event streaming for real-time processing

## API Version

All API endpoints are prefixed with `/api/v1`. The current version is **1.0.0**.

## Getting Started

1. [Quick Start Guide](./quick-start) - Get up and running in 5 minutes
2. [Authentication](./authentication) - Learn about JWT authentication
3. [Workflows](./workflows) - Common API workflows
4. [API Reference](/api-reference) - Full OpenAPI specification

## Base URL

```
http://localhost:3000/api/v1
```

## Response Format

All API responses follow a consistent JSON format:

### Success Response
```json
{
  "data": { ... },
  "message": "Operation successful"
}
```

### Error Response
```json
{
  "error": "Error message description"
}
```

## Rate Limits

Currently, no rate limiting is enforced. Future versions will include rate limiting for API protection.

## Support

For questions or issues:
- Check the [Error Handling](./error-handling) guide
- Review common [Workflows](./workflows)
- Contact the development team
