# Go Facades Library

A simple and elegant facade library for common services, inspired by Laravel facades.

## Installation

Install the library via Go modules:

```bash
go get github.com/nanaaikinson/gofacades
```

### Usage Example: Redis Facade

Seamlessly integrate with Redis using a simple facade:

#### Import and Initialize

```go
import (
    "context"
    "log"
    "time"

    redisFacade "github.com/nanaaikinson/gofacades/redis"
)

func main() {
    // Configure and create a new Redis client
    redisClient, err := redisFacade.New(redisFacade.Config{
        Host:     "localhost",
        Port:     6379,
        Password: "",
        DB:       0,
    })
    if err != nil {
        log.Fatalf("Failed to initialize Redis client: %v", err)
    }
    defer redisClient.Close()

    // Set a value in Redis
    err = redisClient.Set(context.Background(), "key", "value", time.Hour)
    if err != nil {
        log.Fatalf("Failed to set value in Redis: %v", err)
    }

    log.Println("Value successfully set in Redis")
}

```

<!-- ## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details. -->
