# Driver Service

A microservice responsible for managing driver registration, availability, and trip assignment in the ride-sharing platform.

## Architecture

The service follows Clean Architecture principles with clear separation of concerns:

```
driver-service/
|-- cmd/
|   +-- main.go                    # Application entry point
|-- internal/
|   |-- domain/                    # Business logic layer
|   |   |-- interfaces.go          # Domain interfaces
|   |   |-- routes.go              # Predefined driver routes
|   |   +-- service.go             # Core business logic
|   |-- infrastructure/            # External dependencies
|   |   |-- grpc/
|   |   |   +-- handler.go         # gRPC API handlers
|   |   +-- messaging/
|   |       +-- trip_consumer.go   # RabbitMQ event consumer
|   +-- util/
|       +-- plate_generator.go     # Utility functions
+-- README.md
```

### Layer Responsibilities

**Domain Layer** (`internal/domain/`)
- Core business logic and rules
- Domain interfaces (contracts)
- Independent of external frameworks
- No infrastructure dependencies

**Infrastructure Layer** (`internal/infrastructure/`)
- gRPC handlers for external API
- RabbitMQ event consumers
- Protocol translation (protobuf <-> domain)
- Depends on domain interfaces

**Utilities** (`internal/util/`)
- Shared helper functions
- Pure functions with no dependencies

## SOLID Principles

### Single Responsibility Principle (SRP)
Each component has ONE clear responsibility:
- `service.go`: Driver management business logic
- `handler.go`: gRPC protocol handling
- `trip_consumer.go`: RabbitMQ event processing
- `plate_generator.go`: License plate generation

### Open/Closed Principle (OCP)
- Extensible via interfaces without modifying existing code
- New driver assignment strategies can be added without changing service.go
- New event types can be handled without changing consumer structure

### Liskov Substitution Principle (LSP)
- All implementations honor their interface contracts
- Consistent error handling patterns
- Idempotent operations (e.g., unregister returns success even if driver not found)

### Interface Segregation Principle (ISP)
- Focused interfaces: `DriverService`, `TripEventConsumer`
- Clients only depend on methods they use
- No fat interfaces with unused methods

### Dependency Inversion Principle (DIP)
- High-level modules (handlers) depend on abstractions (interfaces)
- Infrastructure depends on domain interfaces, not vice versa
- Dependency injection in cmd/main.go

## API

### gRPC Endpoints

**RegisterDriver**
```protobuf
rpc RegisterDriver(RegisterDriverRequest) returns (RegisterDriverResponse)
```
Registers a new driver with the system.

**UnregisterDriver**
```protobuf
rpc UnregisterDriver(RegisterDriverRequest) returns (RegisterDriverResponse)
```
Removes a driver from the available pool (idempotent).

### RabbitMQ Events

**Consumes:**
- `trip.event.created` - New trip requests
- `trip.event.driver_not_interested` - Driver rejection events

**Publishes:**
- `driver.cmd.register` - Driver assignment confirmation
- `trip.event.no_drivers_found` - No available drivers

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RABBITMQ_URI` | RabbitMQ connection string | (required) |
| `GRPC_ADDR` | gRPC server address | `:9092` |

## Building and Running

### Local Development

```bash
# Build
go build -o build/driver-service ./services/driver-service/cmd/main.go

# Run
RABBITMQ_URI=amqp://guest:guest@localhost:5672/ ./build/driver-service
```

### Docker (via Tilt)

```bash
# Start all services including driver-service
tilt up

# View logs
tilt logs driver-service

# Rebuild
tilt trigger driver-service-compile
```

### Testing

```bash
# Run tests
go test ./services/driver-service/...

# Run tests with coverage
go test -cover ./services/driver-service/...

# Run specific package tests
go test ./services/driver-service/internal/domain
```

## Design Decisions

### In-Memory Storage
Currently uses in-memory slice for driver storage with mutex synchronization:
- **Pros**: Fast, simple, no external dependencies
- **Cons**: Data lost on restart, not scalable across instances
- **Future**: Replace with Redis/database for production

### Thread Safety
All driver operations protected by `sync.Mutex`:
- Prevents race conditions
- Safe for concurrent gRPC/RabbitMQ access
- Uses defer for guaranteed unlock

### Idempotent Operations
`UnregisterDriver` is idempotent:
- Returns success even if driver not found
- Prevents errors on duplicate unregister calls
- Simplifies client retry logic

### Error Handling
- Wrapped errors with context: `fmt.Errorf("%w: %s", ErrDriverNotFound, driverID)`
- gRPC status codes for protocol errors
- Structured logging for debugging

## Driver Assignment Logic

1. **Filter by Package Type**: Only match drivers with requested package type
2. **Return First Match**: Simple first-available assignment
3. **Future Enhancements**:
   - Distance-based matching using geohash
   - Driver rating consideration
   - Load balancing across drivers

## Dependencies

- **gRPC**: Inter-service communication
- **RabbitMQ**: Event-driven messaging
- **Protocol Buffers**: API contracts in `/shared/proto/driver`
- **Geohash**: Location encoding for future proximity matching

## Troubleshooting

### gRPC Server Won't Start
```
Error: failed to listen: address already in use
```
Solution: Check if port 9092 is available or set `GRPC_ADDR` to different port

### RabbitMQ Connection Failed
```
Error: RABBITMQ_URI environment variable is required
```
Solution: Set `RABBITMQ_URI=amqp://guest:guest@localhost:5672/`

### No Drivers Found
- Ensure drivers are registered via gRPC `RegisterDriver`
- Check package type matches trip request
- Verify drivers weren't unregistered

### Build Errors After Refactoring
- Run `go mod tidy` to update dependencies
- Clear build cache: `go clean -cache`
- Verify import paths match new structure

## Migration from Old Structure

This service was refactored from a monolithic main package structure to Clean Architecture:

**Before:**
- All code in root package `main`
- Global variables and tight coupling
- Difficult to test and extend

**After:**
- Layered architecture with clear boundaries
- Interface-based dependency injection
- Testable, maintainable, extensible

See commit history for detailed migration steps.
