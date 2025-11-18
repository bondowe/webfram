---
layout: default
title: Server-Sent Events
nav_order: 11
description: "Real-time server-to-client streaming with SSE"
---

# Server-Sent Events (SSE)

WebFram provides built-in support for Server-Sent Events, enabling real-time server-to-client communication over HTTP.

## Overview

SSE is perfect for:

- Push notifications
- Live updates
- Streaming data
- Real-time dashboards
- Log streaming

## Creating an SSE Endpoint

Use the `app.SSE()` function:

```go
mux.Handle("GET /events", app.SSE(
    payloadFunc,      // Function that generates SSE payload
    disconnectFunc,   // Function called when client disconnects
    errorFunc,        // Function called on errors
    interval,         // Time interval between messages
    headers,          // Optional custom headers
))
```

## SSE Payload Structure

```go
type SSEPayload struct {
    Id       string        // Event ID (optional)
    Event    string        // Event type/name (optional)
    Comments []string      // Comments (optional, for debugging)
    Data     any           // Data payload (required)
    Retry    time.Duration // Retry interval (optional)
}
```

## Basic Example

```go
mux.Handle("GET /time", app.SSE(
    // Payload function
    func() app.SSEPayload {
        return app.SSEPayload{
            Data: fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC3339)),
        }
    },
    // Disconnect function
    func() {
        log.Println("Client disconnected")
    },
    // Error function
    func(err error) {
        log.Printf("SSE error: %v\n", err)
    },
    // Interval
    2*time.Second,
    // Custom headers
    nil,
))
```

## Advanced Example

```go
mux.Handle("GET /notifications", app.SSE(
    func() app.SSEPayload {
        notification := getNextNotification()
        
        return app.SSEPayload{
            Id:       uuid.New().String(),
            Event:    notification.Type,
            Comments: []string{"Notification event"},
            Data:     notification,
            Retry:    5 * time.Second,
        }
    },
    func() {
        log.Println("Client stopped listening")
    },
    func(err error) {
        log.Printf("Error: %v\n", err)
    },
    1*time.Second,
    nil,
))
```

## Client-Side Usage

**Vanilla JavaScript:**

```javascript
const eventSource = new EventSource('http://localhost:8080/events');

// Listen for messages
eventSource.onmessage = function(event) {
    console.log('Received:', event.data);
};

// Listen for specific event types
eventSource.addEventListener('TIME_UPDATE', function(event) {
    console.log('Time update:', event.data);
});

// Handle errors
eventSource.onerror = function(error) {
    console.error('EventSource error:', error);
};

// Close connection
eventSource.close();
```

**React Example:**

```javascript
import { useEffect, useState } from 'react';

function Dashboard() {
    const [data, setData] = useState(null);
    
    useEffect(() => {
        const eventSource = new EventSource('http://localhost:8080/events');
        
        eventSource.onmessage = (event) => {
            setData(JSON.parse(event.data));
        };
        
        return () => eventSource.close();
    }, []);
    
    return <div>{JSON.stringify(data)}</div>;
}
```

## Configuration

### Required Parameters

- **`payloadFunc`**: Function returning SSEPayload
- **`interval`**: Must be > 0

### Optional Parameters

- **`disconnectFunc`**: Called on client disconnect (default: no-op)
- **`errorFunc`**: Called on errors (default: prints errors)
- **`headers`**: Custom HTTP headers (can be nil)

## Use Cases

### Real-Time Dashboard

```go
mux.Handle("GET /dashboard", app.SSE(
    func() app.SSEPayload {
        return app.SSEPayload{
            Data: map[string]interface{}{
                "activeUsers":  getActiveUsers(),
                "requests":     getRequestCount(),
                "errors":       getErrorCount(),
            },
        }
    },
    nil,
    nil,
    1*time.Second,
    nil,
))
```

### Live Notifications

```go
mux.Handle("GET /notifications", app.SSE(
    func() app.SSEPayload {
        notif := getLatestNotification()
        return app.SSEPayload{
            Id:    notif.ID,
            Event: "notification",
            Data:  notif,
        }
    },
    nil,
    nil,
    2*time.Second,
    nil,
))
```

### Stock Price Updates

```go
mux.Handle("GET /stocks/{symbol}", app.SSE(
    func() app.SSEPayload {
        symbol := getSymbolFromContext()
        price := getStockPrice(symbol)
        
        return app.SSEPayload{
            Event: "price_update",
            Data: map[string]interface{}{
                "symbol": symbol,
                "price":  price,
                "time":   time.Now(),
            },
        }
    },
    nil,
    nil,
    1*time.Second,
    nil,
))
```

### Log Streaming

```go
mux.Handle("GET /logs", app.SSE(
    func() app.SSEPayload {
        logs := tailLogs(10)
        return app.SSEPayload{
            Event: "log_entry",
            Data:  logs,
        }
    },
    nil,
    nil,
    500*time.Millisecond,
    nil,
))
```

## Custom Headers

Add custom headers to SSE responses:

```go
headers := map[string]string{
    "X-Custom-Header": "value",
    "Cache-Control":   "no-cache",
}

mux.Handle("GET /events", app.SSE(
    payloadFunc,
    nil,
    nil,
    1*time.Second,
    headers,
))
```

## Error Handling

Handle errors gracefully:

```go
mux.Handle("GET /events", app.SSE(
    func() app.SSEPayload {
        data, err := fetchData()
        if err != nil {
            return app.SSEPayload{
                Event: "error",
                Data:  map[string]string{"error": err.Error()},
            }
        }
        return app.SSEPayload{Data: data}
    },
    func() {
        log.Println("Client disconnected")
    },
    func(err error) {
        log.Printf("SSE error: %v\n", err)
        // Send alert, metric, etc.
    },
    1*time.Second,
    nil,
))
```

## Best Practices

1. **Keep intervals reasonable** - Don't flood clients with data
2. **Handle disconnects** - Clean up resources
3. **Error logging** - Monitor SSE errors
4. **Use event types** - Categorize different message types
5. **Set retry intervals** - Help clients reconnect
6. **Add event IDs** - Enable clients to resume from last event
7. **Use HTTPS** in production

## OpenAPI Documentation

When generating OpenAPI documentation for SSE endpoints, **do not set a TypeHint**. The framework automatically uses the `SSEPayload` type for `text/event-stream` media types.

```go
// ✅ Correct - no TypeHint needed for SSE
mux.Handle("GET /events", app.SSE(...)).WithOperationConfig(&app.OperationConfig{
    OperationID: "streamEvents",
    Summary:     "Stream server events",
    Responses: map[string]app.Response{
        "200": {
            Description: "SSE stream of events",
            Content: map[string]app.TypeInfo{
                "text/event-stream": {}, // SSEPayload is automatically used
            },
        },
    },
})

// ❌ Incorrect - don't override TypeHint for SSE
mux.Handle("GET /events", app.SSE(...)).WithOperationConfig(&app.OperationConfig{
    OperationID: "streamEvents",
    Summary:     "Stream server events",
    Responses: map[string]app.Response{
        "200": {
            Description: "SSE stream of events",
            Content: map[string]app.TypeInfo{
                "text/event-stream": {TypeHint: &MyCustomType{}}, // Will be ignored
            },
        },
    },
})
```

The `SSEPayload.Data` field accepts `any` type, allowing flexible payloads while maintaining proper OpenAPI schema generation.

See the [OpenAPI documentation](openapi.md) for more details on TypeHint usage with different media types.

## Browser Support

SSE is supported in all modern browsers:

- Chrome
- Firefox
- Safari
- Edge
- Opera

For older browsers, consider:

- Polyfills
- Fallback to WebSockets
- Long polling

## See Also

- [Request & Response](request-response.md)
- [Routing](routing.md)
- [Middleware](middleware.md)
