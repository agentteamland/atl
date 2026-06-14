---
knowledge-base-summary: "What goes through WebSocket vs REST. Rule: Socket carries notifications and lightweight state signals. File uploads, large data, CRUD operations → always REST. Socket says \"file is ready\", client fetches via REST."
---
# Data Boundary: WebSocket vs REST

## The Fundamental Rule

**WebSocket carries signals. REST carries data.**

If the payload is a lightweight notification saying "something happened" -- use WebSocket.
If the payload is the actual data (a full object, a list, a file) -- use REST.

Socket says **what** happened. REST provides the **details**.

> **i18n note:** socket events that include user-facing text (a toast string, a notification title) MUST use the `messageKey + placeholders + fallback` envelope defined in `api-agent/children/user-facing-strings.md`. Never embed raw English/Turkish sentences in event payloads — UI localizes.

## What Goes Through WebSocket

WebSocket is for real-time, fire-and-forget signals that are:
- Small (< 1KB typically)
- Ephemeral (no response expected)
- Time-sensitive (must arrive NOW or lose value)

### Belongs on WebSocket

| Category               | Examples                                                    |
|------------------------|-------------------------------------------------------------|
| **Notifications**      | "You have a new friend request", "Task goal achieved"       |
| **Status changes**     | "Order confirmed", "Payment processed", "Task started"      |
| **Typing indicators**  | "Alice is typing..."                                        |
| **Presence signals**   | "User came online", "User went idle"                        |
| **Real-time counters** | "Step count: 4521", "3 users viewing this page"             |
| **Sync triggers**      | "Your profile was updated on another device"                |
| **Read receipts**      | "Message was read by Ali"                                   |
| **Progress updates**   | "File upload 45% complete", "Task 3.2km / 5km"             |

### WebSocket Event Example

```csharp
// The Socket event is SMALL -- just a signal with minimal context
await Clients.User(userId).SendAsync("notification-received", new
{
    EventId = Guid.NewGuid().ToString("N"),
    NotificationId = notificationId,      // ID only, not the full object
    Type = "friend-request",
    Title = "New Friend Request",
    Body = "Ali wants to task with you",
    Timestamp = DateTimeOffset.UtcNow
});
```

The client receives this signal and either:
1. Shows a toast/banner directly (enough info in the event).
2. Fetches full details via REST if needed: `GET /api/notifications/{notificationId}`.

## What Goes Through REST

REST is for operations that are:
- Data-heavy (responses > 1KB)
- Request-response (client needs a result)
- Paginated or filterable
- Idempotent or transactional
- File-based

### Belongs on REST

| Category               | Examples                                                    |
|------------------------|-------------------------------------------------------------|
| **CRUD operations**    | Create task, update profile, delete notification            |
| **File uploads**       | Profile photo, task route GPX file                          |
| **Paginated lists**    | Task history (page 2 of 10), notification inbox             |
| **Search**             | Find nearby taskers, search task routes                     |
| **Reports**            | Weekly task summary, monthly stats                          |
| **Authentication**     | Login, register, refresh token                              |
| **Full object fetch**  | GET /api/tasks/{id} with all details                        |
| **Bulk operations**    | Mark all notifications as read, batch update                |

### REST Endpoint Example

```
GET /api/tasks/history?page=1&pageSize=20
Authorization: Bearer {jwt}

Response (2.5KB):
{
    "items": [
        {
            "id": "abc-123",
            "distanceMeters": 5420,
            "durationSeconds": 3600,
            "stepCount": 7800,
            "startedAt": "2026-04-14T08:00:00Z",
            "completedAt": "2026-04-14T09:00:00Z",
            "route": { ... }
        },
        ...
    ],
    "page": 1,
    "pageSize": 20,
    "totalPages": 5,
    "totalCount": 93
}
```

This response is too large, too structured, and too query-dependent for WebSocket.

## The Decision Rules

### Rule 1: If payload > 1KB, use REST

WebSocket messages should be small signals. If you are sending a full object graph, a list of items, or any payload that regularly exceeds 1KB, it belongs on REST.

### Rule 2: If the client needs a response, use REST

WebSocket is fire-and-forget. The server sends an event; the client may or may not receive it. There is no built-in acknowledgment mechanism for server-to-client events.

If the client needs confirmation ("was my task saved?"), use REST. The HTTP response gives a definitive answer.

### Rule 3: If the data needs pagination, filtering, or sorting, use REST

WebSocket has no query string, no pagination headers, no content negotiation. REST was built for this.

### Rule 4: If it involves file transfer, use REST

Never upload or download files through WebSocket. Binary frames over WebSocket are fragile, non-resumable, and bypass all HTTP middleware (compression, CDN, caching).

### Rule 5: If it is a CRUD operation, use REST

Create, Read, Update, Delete are request-response by nature. The client needs to know if the operation succeeded. WebSocket cannot provide this guarantee.

## The Coordination Pattern

The most common pattern: REST does the work, Socket announces the result.

### Example: User Completes a Task

```
1. Client (Flutter) -----> API (REST)
   POST /api/tasks/{id}/complete
   Response: 200 OK { taskId, stats, badges }

2. API -----> Socket (Internal Broadcast)
   POST /api/internal/broadcast
   {
       targetType: "user",
       targetId: "{userId}",
       eventName: "task-completed",
       payload: { taskId, distanceMeters: 5420, stepCount: 7800 }
   }

3. Socket -----> All User Devices (WebSocket)
   Event: "task-completed"
   Payload: { taskId, distanceMeters: 5420, stepCount: 7800 }

4. Other Devices (Flutter) -----> API (REST)
   GET /api/tasks/{taskId}
   Response: 200 OK { full task details including route, badges, etc. }
```

- Step 1: The initiating device gets the full response via REST.
- Step 2-3: Other devices get a lightweight signal via Socket.
- Step 4: Other devices fetch full details if needed via REST.

### Example: File Upload

```
1. Client -----> API (REST)
   POST /api/tasks/{id}/photo
   Content-Type: multipart/form-data
   [binary image data]
   Response: 200 OK { photoUrl }

2. API -----> Socket (Internal Broadcast)
   Event: "task-photo-uploaded"
   Payload: { taskId, photoUrl, thumbnailUrl }   // URL only, NOT the file

3. Other Devices -----> CDN / API (REST)
   GET {photoUrl}   // Download the actual image
```

The Socket event says "a photo was uploaded" and includes the URL. The actual file is fetched via REST/CDN. Never push binary data through WebSocket.

## Decision Table

| Question                                        | WebSocket | REST |
|-------------------------------------------------|-----------|------|
| Is the payload < 1KB?                           | Yes       | --   |
| Is it a fire-and-forget signal?                 | Yes       | --   |
| Does the client need a response?                | --        | Yes  |
| Does it involve file transfer?                  | --        | Yes  |
| Is it a CRUD operation?                         | --        | Yes  |
| Does it need pagination or filtering?           | --        | Yes  |
| Is it time-sensitive (must arrive in < 1 sec)?  | Yes       | --   |
| Is it ephemeral (typing, presence)?             | Yes       | --   |
| Is it a status change notification?             | Yes       | --   |
| Does it trigger other devices to sync?          | Yes       | --   |

## Anti-Patterns

### Anti-Pattern 1: Sending full objects through WebSocket

```csharp
// WRONG -- sending the entire task object (2KB+) via Socket
await Clients.User(userId).SendAsync("task-completed", new
{
    TaskId = task.Id,
    Distance = task.Distance,
    Duration = task.Duration,
    Steps = task.Steps,
    Route = task.Route,           // Large GeoJSON
    Badges = task.EarnedBadges,   // Array of badge objects
    Stats = task.DailyStats,      // Aggregate stats
    // ... 20 more fields
});

// CORRECT -- send a lightweight signal, client fetches full object via REST
await Clients.User(userId).SendAsync("task-completed", new
{
    EventId = Guid.NewGuid().ToString("N"),
    TaskId = task.Id,
    DistanceMeters = task.Distance,   // Enough for a toast notification
    StepCount = task.Steps,
    Timestamp = DateTimeOffset.UtcNow
});
```

### Anti-Pattern 2: File upload through WebSocket

```csharp
// NEVER DO THIS
public async Task UploadPhoto(byte[] imageData)
{
    // Binary over WebSocket is fragile, non-resumable, and
    // bypasses all HTTP middleware (compression, CDN, auth, etc.)
}
```

### Anti-Pattern 3: Using WebSocket for CRUD

```csharp
// NEVER DO THIS
public async Task<TaskDto> CreateTask(CreateTaskRequest request)
{
    var result = await _apiClient.PostAsync("/api/tasks", request);
    return result; // Hub methods should not return complex data
}
```

Hub methods that call API and return results are tempting but wrong. The client should call REST directly for CRUD. Socket is not a REST proxy.

### Anti-Pattern 4: Fetching paginated data through WebSocket

```csharp
// NEVER DO THIS
public async Task GetTaskHistory(int page, int pageSize)
{
    var result = await _apiClient.GetAsync($"/api/tasks?page={page}&pageSize={pageSize}");
    await Clients.Caller.SendAsync("task-history", result);
}
```

## Socket's Role Summary

Socket is a **notification channel**, not a data channel.

```
Socket tells you:  "Hey, something happened."
REST tells you:    "Here are the details of what happened."
```

Keep the Socket thin. Keep it fast. Keep it small.
