# API Gateway Service

A modern, SOLID-compliant API Gateway for the ride-sharing microservices platform. This service routes HTTP requests and WebSocket connections to backend microservices using gRPC.

## Architecture

The API Gateway follows clean architecture principles with proper separation of concerns:

```
api-gateway/
|-- cmd/
|   +-- main.go                     # Application entry point with dependency injection
+-- internal/
    |-- clients/                     # gRPC client abstractions
    |   |-- interfaces.go            # Client interface definitions
    |   |-- driver_client.go         # Driver service gRPC client
    |   +-- trip_client.go           # Trip service gRPC client
    |-- dto/                         # Data Transfer Objects
    |   +-- requests.go              # HTTP request/response types
    |-- handlers/
    |   |-- http/                    # HTTP request handlers
    |   |   |-- trip_handler.go      # Trip-related endpoints
    |   |   |-- response.go          # JSON response utility
    |   |   +-- middleware.go        # CORS and other middleware
    |   +-- websocket/               # WebSocket connection handlers
    |       |-- handler.go           # Base WebSocket handler
    |       |-- validator.go         # Request validation helpers
    |       |-- rider_handler.go     # Rider WebSocket connections
    |       +-- driver_handler.go    # Driver WebSocket connections
    +-- websocket/                   # WebSocket infrastructure
        |-- connection_manager.go    # Connection registry
        +-- upgrader.go              # HTTP to WebSocket upgrade
```

## SOLID Principles Applied

### Single Responsibility Principle (SRP)
- **Handlers**: Each handler has a single purpose (HTTP or WebSocket)
- **Clients**: Separate client wrappers for each backend service
- **Validators**: Isolated request validation logic
- **Infrastructure**: Connection management separate from upgrade logic

### Open/Closed Principle (OCP)
- **Interface-based design**: Extend functionality via new implementations
- **Middleware pattern**: Add cross-cutting concerns without modifying handlers
- **Decorator pattern**: Wrap clients with logging, metrics, or caching

### Liskov Substitution Principle (LSP)
- **Interfaces**: Any implementation can be substituted without breaking behavior
- **Consistent contracts**: All implementations honor interface contracts

### Interface Segregation Principle (ISP)
- **Focused interfaces**: Clients expose only needed methods
- **No fat interfaces**: Each interface serves a specific purpose

### Dependency Inversion Principle (DIP)
- **Depend on abstractions**: Handlers depend on client interfaces, not concrete types
- **Dependency injection**: All dependencies injected via constructors
- **No global state**: Zero global variables

## API Endpoints

### HTTP Endpoints

#### Preview Trip
```http
POST /trip/preview
Content-Type: application/json

{
  "userID": "user123",
  "pickup": {
    "latitude": 37.7749,
    "longitude": -122.4194
  },
  "destination": {
    "latitude": 37.8044,
    "longitude": -122.2712
  }
}
```

**Response:**
```json
{
  "data": {
    "route": {...},
    "rideFares": [...]
  }
}
```

#### Start Trip
```http
POST /trip/start
Content-Type: application/json

{
  "userID": "user123",
  "rideFareID": "fare456"
}
```

**Response:**
```json
{
  "data": {
    "tripID": "trip789"
  }
}
```

### WebSocket Endpoints

#### Rider Connection
```
ws://localhost:8081/ws/riders?userID=user123
```

**Purpose**: Real-time updates for riders (trip status, driver location, etc.)

#### Driver Connection
```
ws://localhost:8081/ws/drivers?userID=driver456&packageSlug=standard
```

**Purpose**: Real-time updates for drivers (trip requests, navigation, etc.)

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_ADDR` | HTTP server address | `:8081` |
| `TRIP_SERVICE_URL` | Trip service gRPC endpoint | `trip-service:9093` |
| `DRIVER_SERVICE_URL` | Driver service gRPC endpoint | `driver-service:9092` |

## Building & Running

### Local Development

```bash
# Build the service
go build -o api-gateway ./cmd/main.go

# Run the service
./api-gateway

# Or build and run in one command
go run ./cmd/main.go
```

### Docker

```bash
# Build Docker image
docker build -t ride-sharing/api-gateway .

# Run container
docker run -p 8081:8081 \
  -e TRIP_SERVICE_URL=trip-service:9093 \
  -e DRIVER_SERVICE_URL=driver-service:9092 \
  ride-sharing/api-gateway
```

### Kubernetes

```bash
# Deploy to Kubernetes
kubectl apply -f ../../infra/development/k8s/api-gateway-deployment.yaml

# Check deployment status
kubectl get pods -l app=api-gateway

# View logs
kubectl logs -f deployment/api-gateway
```

## Testing

### Unit Tests

The architecture is designed for easy unit testing with interfaces:

```go
// Example: Testing TripHandler with mock client
func TestHandleTripPreview(t *testing.T) {
    mockClient := &MockTripServiceClient{
        PreviewTripFunc: func(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {
            return &pb.PreviewTripResponse{...}, nil
        },
    }

    handler := NewTripHandler(mockClient)
    // Test handler with mock...
}
```

### Integration Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

## Key Design Decisions

### Why Interface-Based Design?

1. **Testability**: Easy to mock dependencies for unit tests
2. **Flexibility**: Swap implementations without changing consumers
3. **Decoupling**: Handlers don't depend on concrete gRPC implementations
4. **Maintainability**: Clear contracts between components

### Why Dependency Injection?

1. **No Global State**: Makes code predictable and thread-safe
2. **Lifecycle Control**: Manage connection pooling and cleanup centrally
3. **Configuration**: Easy to configure different behaviors per environment
4. **Testing**: Inject mocks instead of real dependencies

### Why Separate ConnectionManager and Upgrader?

1. **Single Responsibility**: Connection storage vs protocol upgrade are separate concerns
2. **Testability**: Test connection logic without HTTP dependencies
3. **Reusability**: ConnectionManager works with any WebSocket source
4. **Flexibility**: Customize upgrade behavior (CORS, compression) independently

## Error Handling

### HTTP Errors

- `400 Bad Request`: Invalid request body or missing required fields
- `500 Internal Server Error`: Backend service failures

### WebSocket Errors

- Connection closed on invalid parameters (userID, packageSlug)
- Graceful cleanup on disconnect
- Automatic unregistration on error

## Security Considerations

### Current Implementation

- **CORS**: Allows all origins (`Access-Control-Allow-Origin: *`)
- **WebSocket Origin**: Accepts all origins (`CheckOrigin: return true`)

### Production Recommendations

1. **Restrict CORS**: Whitelist specific frontend domains
2. **WebSocket Origin Validation**: Implement origin validator with whitelist
3. **Authentication**: Add JWT validation middleware
4. **Rate Limiting**: Implement rate limiting per user/IP
5. **TLS**: Enable HTTPS/WSS in production

## Performance Optimizations

### Connection Pooling

gRPC clients maintain connection pools automatically. Client instances are reused across requests.

### Graceful Shutdown

The service properly closes all gRPC connections during shutdown:

```go
// Cleanup on SIGTERM/SIGINT
tripClient.Close()
driverClient.Close()
server.Shutdown(ctx)
```

### Concurrent Connection Handling

- WebSocket connections handled in separate goroutines
- Thread-safe connection registry with RWMutex
- Per-connection mutex for write operations

## Monitoring & Observability

### Logging

All handlers log:
- WebSocket upgrade failures
- Message read/write errors
- Driver registration/unregistration
- Backend service errors

### Future Enhancements

- [ ] Prometheus metrics (request count, latency, errors)
- [ ] Distributed tracing with OpenTelemetry
- [ ] Health check endpoint (`/health`)
- [ ] Readiness probe (`/ready`)

## Troubleshooting

### Service won't start

**Issue**: `Failed to create trip client: connection refused`

**Solution**: Ensure trip-service is running and accessible at configured URL

### WebSocket connection fails

**Issue**: `WebSocket upgrade failed: 403 Forbidden`

**Solution**: Check CORS configuration and origin validation

### High memory usage

**Issue**: Memory grows over time

**Solution**: Check for WebSocket connection leaks, ensure proper cleanup in defer statements

## Migration History

This service was refactored from a monolithic `main` package to clean architecture:

### Before Refactoring
- All code in `main` package (untestable)
- Global variables (`connManager`)
- Direct gRPC client creation in handlers
- Mixed concerns (HTTP, WebSocket, connection management)

### After Refactoring
- Clean layered architecture (`cmd`, `internal/`)
- Zero global variables
- Dependency injection throughout
- SOLID principles applied
- Fully testable with interfaces

## Contributing

### Code Style

- Follow Go best practices and idioms
- Use `gofmt` for formatting
- Run `go vet` before committing
- Write unit tests for new functionality

### Adding New Endpoints

1. Define DTO in `internal/dto/`
2. Create handler method in appropriate handler file
3. Wire route in `cmd/main.go`
4. Add tests

### Adding New Backend Service Clients

1. Define interface in `internal/clients/interfaces.go`
2. Implement wrapper in `internal/clients/{service}_client.go`
3. Inject into handlers that need it
4. Add to `cmd/main.go` initialization

## License

Copyright (c) 2025 Ride Sharing Platform

## Related Services

- **Trip Service**: Handles trip creation, fare calculation, and routing
- **Driver Service**: Manages driver registration, availability, and matching
- **Web Frontend**: User-facing web application (connects via this gateway)

## References

- [SOLID Principles](https://en.wikipedia.org/wiki/SOLID)
- [Dependency Injection in Go](https://blog.drewolson.org/dependency-injection-in-go)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [API Gateway Pattern](https://microservices.io/patterns/apigateway.html)
