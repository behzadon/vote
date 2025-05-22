# Vote - Interactive Polling Platform

A massively scalable, interactive polling platform built with Go that provides a vertical feed of polls for mobile and web applications. The platform supports real-time voting, polling statistics, and user engagement features.

## Product Overview

The platform provides a vertical feed of polls, where each poll has:
- A text title
- Multiple-choice options
- One or more tags (e.g., sports, news, entertainment, etc.)

### Core User Interactions
- Vote on a poll by selecting exactly one of the options
- Skip a poll if it's not interesting
- Each user never sees the same poll twice (once voted or skipped, it's removed from their feed)
- Filter the feed by tag or search criteria
- Daily Vote Limit: A user can only vote on up to 100 polls per day (skips are unlimited)

### System Scale & Performance
- Large user base with high read/write concurrency
- Rapidly growing dataset of polls, votes, and skips
- Fast feed loading (read operations)
- Instant vote/skip feedback (write operations)

## Features

### Core Functionality
- **Interactive Poll Feed**: Vertical feed of polls with real-time updates
- **Voting System**: 
  - Single-choice voting
  - Daily vote limits (100 votes per user)
  - Skip functionality
  - Never see the same poll twice
- **Poll Management**:
  - Create polls with multiple options
  - Tag-based categorization
  - Poll statistics and analytics
- **User Management**:
  - User registration and authentication
  - JWT-based authentication
  - User profile management

### Technical Features
- **Performance & Scale**:
  - High read/write concurrency support
  - Redis caching layer for fast access
  - RabbitMQ event streaming for real-time updates
  - Horizontal scalability
- **Rate Limiting**:
  - Per-user rate limits (100 requests per minute)
  - Burst protection (200 requests per second)
  - Redis-based rate limiting
- **Monitoring & Observability**:
  - Prometheus metrics integration
  - Grafana dashboards
  - Structured logging with Zap
  - Health checks for all services

## Architecture

### System Components
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   API Layer │────▶│  Service    │────▶│ Repository  │
│   (Gin)     │     │  Layer      │     │  Layer      │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Redis     │     │  RabbitMQ   │     │ PostgreSQL  │
│  (Cache)    │     │  (Events)   │     │  (Storage)  │
└─────────────┘     └─────────────┘     └─────────────┘
```

### Technology Stack
- **Backend**: Go 1.21
- **Web Framework**: Gin
- **Database**: PostgreSQL 15
- **Cache**: Redis 7
- **Message Queue**: RabbitMQ 3
- **Monitoring**: Prometheus & Grafana
- **Containerization**: Docker & Docker Compose

## Getting Started

### Prerequisites
- Go 1.21 or later
- Docker and Docker Compose
- Make (optional, for using Makefile commands)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/vote.git
   cd vote
   ```

2. Start the services using Docker Compose:
   ```bash
   docker-compose up -d
   ```
   This will start:
   - PostgreSQL database
   - Redis cache
   - RabbitMQ message broker
   - Prometheus metrics
   - Grafana dashboards
   - The Vote application

3. Run database migrations:
   ```bash
   go run cmd/migrate/main.go
   ```

4. Start the application:
   ```bash
   go run cmd/server/main.go
   ```

The service will be available at `http://localhost:8080`

### Configuration

The application can be configured using environment variables or the `config/config.yaml` file. Key configuration options include:

```yaml
server:
  port: 8080
  env: development

postgres:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: vote

redis:
  host: localhost
  port: 6379

rabbitmq:
  host: localhost
  port: 5672
  user: guest
  password: guest

jwt:
  secret_key: "your-secret-key"
  token_duration: 24h
```

## Monitoring & Observability

### Prometheus Metrics

The application exposes detailed metrics for every API endpoint using Prometheus. Metrics are automatically collected for all HTTP endpoints, including request counts, durations, status codes, and business operations (polls, votes, users, cache).

- **Metrics endpoint:**
  - `GET /metrics` — Exposes all Prometheus metrics in the standard format.
- **What is tracked:**
  - Total HTTP requests per endpoint, method, and status code
  - Request duration histograms per endpoint
  - Active requests in progress
  - Business operations (poll creation, voting, user registration, etc.)
  - Cache hit/miss rates

### Prometheus Setup

Prometheus is pre-configured to scrape metrics from the application. See `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'vote'
    static_configs:
      - targets: ['app:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s
```

You can view metrics in Prometheus or connect Grafana dashboards for visualization.

## API Documentation

### Authentication

#### Register User
```http
POST /api/auth/register
Content-Type: application/json

{
    "username": "user123",
    "email": "user@example.com",
    "password": "password123"
}
```

#### Login
```http
POST /api/auth/login
Content-Type: application/json

{
    "email": "user@example.com",
    "password": "password123"
}
```

### Polls

#### Create Poll
```http
POST /api/polls
Authorization: Bearer <token>
Content-Type: application/json

{
    "title": "Your favorite programming language?",
    "options": ["Go", "Python", "Rust"],
    "tags": ["programming", "favorites"]
}
```

#### Get Poll Feed
```http
GET /api/polls?tag=programming&page=1&limit=10&userId=123
Authorization: Bearer <token>
```

#### Vote on Poll
```http
POST /api/polls/{id}/vote
Authorization: Bearer <token>
Content-Type: application/json

{
    "userId": "123",
    "optionIndex": 1
}
```

#### Skip Poll
```http
POST /api/polls/{id}/skip
Authorization: Bearer <token>
Content-Type: application/json

{
    "userId": "123"
}
```

#### Get Poll Statistics
```http
GET /api/polls/{id}/stats
```

### Metrics

- `GET /metrics` — Prometheus metrics endpoint for all API and business operations.

### Rate Limiting

The API implements rate limiting using Redis:
- **Per-User Rate Limit**: 100 requests per minute
- **Burst Protection**: 200 requests per second
- **Rate Limit Headers**:
  - `X-RateLimit-Limit`: Maximum requests per window
  - `X-RateLimit-Remaining`: Remaining requests in current window
  - `X-RateLimit-Reset`: Time when the rate limit resets

## Technical Implementation

### Database Schema

```sql
-- Polls table
CREATE TABLE polls (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    options JSONB NOT NULL,
    tags TEXT[] NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Votes table
CREATE TABLE votes (
    id SERIAL PRIMARY KEY,
    poll_id INTEGER REFERENCES polls(id),
    user_id INTEGER NOT NULL,
    option_index INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(poll_id, user_id)
);

-- Skips table
CREATE TABLE skips (
    id SERIAL PRIMARY KEY,
    poll_id INTEGER REFERENCES polls(id),
    user_id INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(poll_id, user_id)
);

-- Daily vote counts
CREATE TABLE daily_votes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    vote_date DATE NOT NULL,
    vote_count INTEGER DEFAULT 0,
    UNIQUE(user_id, vote_date)
);
```

### Caching Strategy

1. **Poll Feed Caching**:
   - Redis cache for poll feed with 5-minute TTL
   - Cache invalidation on new votes/skips
   - User-specific cache keys to handle voted/skipped polls

2. **Poll Statistics Caching**:
   - Redis cache for poll statistics
   - Incremental updates for vote counts
   - 1-hour TTL with background refresh

3. **Rate Limiting**:
   - Redis-based rate limiting
   - Sliding window implementation
   - Separate counters for votes and API requests

### Concurrency Model

1. **Vote Processing**:
   - Optimistic locking for vote updates
   - Distributed locks for daily vote limits
   - Event-driven architecture for real-time updates

2. **Feed Generation**:
   - Read replicas for poll queries
   - Materialized views for popular polls
   - Background job for feed pre-computation

### Performance Testing

The project includes a `loadtest` directory with k6 scripts for performance testing:

```bash
# Run load tests
k6 run loadtest/poll-feed.js
k6 run loadtest/voting.js
```

Performance metrics are collected and visualized in Grafana:
- Response time percentiles (p50, p90, p99)
- Request rate and error rates
- Database and cache performance
- System resource utilization

### Future Growth Considerations

1. **10x Scale**:
   - Implement database sharding
   - Add read replicas
   - Introduce CDN for static content
   - Implement circuit breakers

2. **100x Scale**:
   - Microservices architecture
   - Global database replication
   - Edge caching
   - Queue-based processing

3. **Feature Evolution**:
   - Multi-stage polls
   - Real-time leaderboards
   - Scheduled polls
   - Poll templates
   - A/B testing framework

### Technical Debt & Trade-offs

1. **Current Trade-offs**:
   - Eventual consistency for vote counts
   - Cache staleness vs. performance
   - Batch processing vs. real-time updates

2. **Areas for Improvement**:
   - Implement database partitioning
   - Add more sophisticated caching strategies
   - Enhance monitoring and alerting
   - Implement feature flags
   - Add more comprehensive testing

## Infrastructure Components

### Database (PostgreSQL)

#### Configuration
```yaml
postgres:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: vote
  max_connections: 100
  idle_connections: 10
  connection_lifetime: 1h
```

#### Performance Optimizations
1. **Indexes**:
   ```sql
   -- Poll feed optimization
   CREATE INDEX idx_polls_created_at ON polls(created_at DESC);
   CREATE INDEX idx_polls_tags ON polls USING GIN(tags);
   
   -- Vote tracking optimization
   CREATE INDEX idx_votes_user_poll ON votes(user_id, poll_id);
   CREATE INDEX idx_skips_user_poll ON skips(user_id, poll_id);
   
   -- Daily vote limit optimization
   CREATE INDEX idx_daily_votes_user_date ON daily_votes(user_id, vote_date);
   ```

2. **Partitioning Strategy**:
   - Votes table partitioned by date range
   - Daily vote counts partitioned by user_id ranges
   - Enables efficient cleanup of old data

3. **Connection Pooling**:
   - PgBouncer for connection pooling
   - Max 100 connections per instance
   - Connection timeout: 30 seconds
   - Idle timeout: 10 minutes

### Caching (Redis)

#### Configuration
```yaml
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 100
  min_idle_conns: 10
  max_retries: 3
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
```

#### Cache Keys Structure
1. **Poll Feed Cache**:
   ```
   poll:feed:{userId}:{page} -> JSON array of polls
   poll:feed:tags:{tag}:{page} -> JSON array of poll IDs
   poll:feed:user:{userId}:voted -> Set of voted poll IDs
   poll:feed:user:{userId}:skipped -> Set of skipped poll IDs
   ```

2. **Poll Statistics Cache**:
   ```
   poll:stats:{pollId} -> JSON of vote counts
   poll:stats:hot -> Set of hot poll IDs
   ```

3. **Rate Limiting Cache**:
   ```
   rate:user:{userId}:votes -> Counter with expiry
   rate:user:{userId}:api -> Counter with expiry
   ```

#### Cache Policies
1. **TTL Settings**:
   - Poll feed: 5 minutes
   - Poll statistics: 1 hour
   - Rate limit counters: 1 minute
   - User session data: 24 hours

2. **Eviction Policy**:
   - Max memory: 2GB
   - Eviction policy: allkeys-lru
   - No persistence (cache-only)

### Message Queue (RabbitMQ)

#### Configuration
```yaml
rabbitmq:
  host: localhost
  port: 5672
  user: guest
  password: guest
  vhost: /
  prefetch_count: 10
  reconnect_interval: 5s
  max_retries: 3
```

#### Queue Structure
1. **Vote Processing**:
   ```
   Exchange: vote.events
   Queues:
   - vote.processing (durable)
   - vote.notifications (durable)
   - vote.analytics (durable)
   ```

2. **Feed Updates**:
   ```
   Exchange: feed.events
   Queues:
   - feed.updates (durable)
   - feed.cache.invalidation (durable)
   ```

3. **System Events**:
   ```
   Exchange: system.events
   Queues:
   - system.metrics (durable)
   - system.alerts (durable)
   ```

#### Message Patterns
1. **Vote Processing Flow**:
   ```
   Producer -> vote.events -> vote.processing
   vote.processing -> vote.notifications (fanout)
   vote.processing -> vote.analytics (fanout)
   ```

2. **Feed Update Flow**:
   ```
   Producer -> feed.events -> feed.updates
   feed.updates -> feed.cache.invalidation (fanout)
   ```

#### Queue Policies
1. **Durability**:
   - All queues are durable
   - Messages persisted to disk
   - HA mode enabled for queues

2. **Message TTL**:
   - Vote processing: 5 minutes
   - Feed updates: 1 minute
   - System events: 24 hours

3. **Dead Letter Handling**:
   - Dead letter exchange: dlx.events
   - Retry policy: 3 attempts
   - Error queue for failed processing

### Infrastructure Monitoring

#### Database Metrics
- Connection pool utilization
- Query performance (slow queries)
- Index usage statistics
- Table sizes and growth
- Replication lag (if applicable)

#### Cache Metrics
- Memory usage
- Hit/miss ratios
- Key eviction rates
- Command latency
- Connection pool status

#### Message Queue Metrics
- Queue depths
- Message rates
- Consumer lag
- Channel status
- Connection status

### High Availability Setup

1. **Database**:
   - Primary-Replica setup
   - Automatic failover
   - Connection pooling with PgBouncer
   - Regular backups

2. **Cache**:
   - Redis Sentinel for HA
   - Automatic failover
   - No persistence (cache-only)
   - Memory monitoring

3. **Message Queue**:
   - RabbitMQ cluster
   - Queue mirroring
   - Automatic failover
   - Message persistence

## Development

### Testing

#### Current Test Coverage
1. **Unit Tests**:
   ```bash
   # Run all unit tests
   go test ./...
   ```
   - Service layer tests (`internal/service/service_test.go`)
   - Domain layer tests (`internal/domain/domain_test.go`)
   - API handler tests (`internal/api/handlers_test.go`)
   - Repository layer tests (`internal/domain/repository_test.go`)
   - Auth handler tests (`internal/api/auth_handler_test.go`)

2. **Load Tests**:
   ```bash
   # Run performance tests using k6
   k6 run loadtest/*.js
   ```

#### Needed Integration Tests
The following integration tests should be implemented to ensure proper component interaction:

1. **Database Integration Tests**:
   - Test actual database operations with a test database
   - Verify transaction handling
   - Test concurrent operations
   - Validate data consistency

2. **Cache Integration Tests**:
   - Test Redis cache operations
   - Verify cache invalidation
   - Test cache consistency with database
   - Validate cache performance

3. **Message Queue Integration Tests**:
   - Test RabbitMQ message publishing/consuming
   - Verify event handling
   - Test message persistence
   - Validate queue behavior under load

4. **End-to-End Tests**:
   - Test complete user flows
   - Verify system behavior with all components
   - Test error handling and recovery
   - Validate performance in integrated environment

To implement these tests, we need to:
1. Create a test environment with all dependencies
2. Set up test databases and caches
3. Implement test fixtures and helpers
4. Add integration test tags and CI/CD pipeline support

### Docker Setup

```bash
docker-compose up --build -d

docker-compose logs -f

docker-compose run --rm app go test ./...
```

### Monitoring

Access monitoring tools at:
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000
- API Metrics: http://localhost:8080/metrics

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [Go Redis](https://github.com/go-redis/redis)
- [RabbitMQ Go Client](https://github.com/rabbitmq/amqp091-go)
- [Prometheus Go Client](https://github.com/prometheus/client_golang) 