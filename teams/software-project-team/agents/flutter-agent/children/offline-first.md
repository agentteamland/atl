---
knowledge-base-summary: "Two patterns: cache-first (read) and offline queue (write). Local storage with shared_preferences or Hive. Connectivity monitoring. Show cached data when offline with staleness indicator. Queue user actions, sync when connection returns. Conflict resolution strategy."
---
# Offline First

## Philosophy

Flutter is a BRIDGE, but a resilient one. The app must handle no-internet gracefully. Two patterns cover all scenarios:

1. **Cache-first (read)** -- show cached data when offline, refresh when online.
2. **Offline queue (write)** -- queue user actions locally, flush to API when connection returns.

No business logic in either pattern. The app stores raw API responses and replays raw user actions. The API decides what is valid.

## Dependencies

```yaml
dependencies:
  connectivity_plus: ^6.0.0      # Network state monitoring
  shared_preferences: ^2.3.0     # Simple key-value cache
  hive_flutter: ^1.1.0           # Complex structured cache
  hive: ^2.2.3
```

## Pattern 1: Cache-First (Read)

API response flows: API -> local storage -> UI. On next load, show local first, then refresh from API.

### Cache Service

```dart
abstract class CacheService {
  Future<String?> get(String key);
  Future<void> set(String key, String value, {Duration? ttl});
  Future<void> remove(String key);
  Future<void> clear();
  Future<bool> isExpired(String key);
}

class SharedPrefsCacheService implements CacheService {
  final SharedPreferences _prefs;

  SharedPrefsCacheService(this._prefs);

  static const _ttlSuffix = '_ttl';

  @override
  Future<String?> get(String key) async {
    final expired = await isExpired(key);
    if (expired) {
      // Data is stale but still return it -- caller decides what to do
    }
    return _prefs.getString(key);
  }

  @override
  Future<void> set(String key, String value, {Duration? ttl}) async {
    await _prefs.setString(key, value);
    if (ttl != null) {
      final expiresAt = DateTime.now().add(ttl).millisecondsSinceEpoch;
      await _prefs.setInt('$key$_ttlSuffix', expiresAt);
    }
  }

  @override
  Future<void> remove(String key) async {
    await _prefs.remove(key);
    await _prefs.remove('$key$_ttlSuffix');
  }

  @override
  Future<void> clear() async {
    await _prefs.clear();
  }

  @override
  Future<bool> isExpired(String key) async {
    final expiresAt = _prefs.getInt('$key$_ttlSuffix');
    if (expiresAt == null) return false; // No TTL set -> never expires
    return DateTime.now().millisecondsSinceEpoch > expiresAt;
  }
}
```

### Cache-First Repository Pattern

```dart
class TaskRepository {
  final ApiClient _api;
  final CacheService _cache;

  TaskRepository(this._api, this._cache);

  static const _tasksKey = 'tasks_list';
  static const _cacheTtl = Duration(minutes: 5);

  /// Returns cached data first (if available), then refreshes from API.
  /// The provider rebuilds on refresh.
  Future<List<Task>> getTasks({bool forceRefresh = false}) async {
    // 1. Try cache first (unless forced refresh)
    if (!forceRefresh) {
      final cached = await _cache.get(_tasksKey);
      if (cached != null) {
        final tasks = _parseTasks(cached);
        // Fire-and-forget background refresh if stale
        if (await _cache.isExpired(_tasksKey)) {
          _refreshInBackground();
        }
        return tasks;
      }
    }

    // 2. Fetch from API
    try {
      final response = await _api.get('/tasks');
      final json = jsonEncode(response.data);
      await _cache.set(_tasksKey, json, ttl: _cacheTtl);
      return _parseTasks(json);
    } on DioException catch (e) {
      // 3. Network error -> fall back to cache (even if expired)
      final cached = await _cache.get(_tasksKey);
      if (cached != null) return _parseTasks(cached);
      rethrow; // No cache, no network -> error
    }
  }

  void _refreshInBackground() async {
    try {
      final response = await _api.get('/tasks');
      final json = jsonEncode(response.data);
      await _cache.set(_tasksKey, json, ttl: _cacheTtl);
    } catch (_) {
      // Silent fail -- stale cache is still being shown
    }
  }

  List<Task> _parseTasks(String json) {
    final list = jsonDecode(json) as List;
    return list.map((e) => Task.fromJson(e as Map<String, dynamic>)).toList();
  }
}
```

### Riverpod Provider with Cache

```dart
final tasksProvider = FutureProvider.autoDispose<List<Task>>((ref) async {
  final repository = ref.watch(taskRepositoryProvider);
  return repository.getTasks();
});

// Force refresh (pull-to-refresh)
ref.refresh(tasksProvider);

// Force refresh bypassing cache
final tasksForceProvider = FutureProvider.family<List<Task>, bool>(
  (ref, forceRefresh) async {
    final repository = ref.watch(taskRepositoryProvider);
    return repository.getTasks(forceRefresh: forceRefresh);
  },
);
```

## Pattern 2: Offline Queue (Write)

User actions are stored in a local queue when offline. When connectivity returns, the queue is flushed to the API in order.

### Offline Action Model

```dart
@HiveType(typeId: 0)
class OfflineAction {
  @HiveField(0)
  final String id;

  @HiveField(1)
  final String endpoint;

  @HiveField(2)
  final String method; // POST, PUT, DELETE

  @HiveField(3)
  final String? body; // JSON string

  @HiveField(4)
  final DateTime createdAt;

  @HiveField(5)
  final int retryCount;

  OfflineAction({
    String? id,
    required this.endpoint,
    required this.method,
    this.body,
    DateTime? createdAt,
    this.retryCount = 0,
  })  : id = id ?? const Uuid().v4(),
        createdAt = createdAt ?? DateTime.now();

  OfflineAction copyWithRetry() {
    return OfflineAction(
      id: id,
      endpoint: endpoint,
      method: method,
      body: body,
      createdAt: createdAt,
      retryCount: retryCount + 1,
    );
  }
}
```

### Offline Queue Service

```dart
class OfflineQueueService {
  static const _boxName = 'offline_queue';
  static const _maxRetries = 3;

  late final Box<OfflineAction> _box;
  final ApiClient _api;

  OfflineQueueService(this._api);

  Future<void> init() async {
    Hive.registerAdapter(OfflineActionAdapter());
    _box = await Hive.openBox<OfflineAction>(_boxName);
  }

  /// Add an action to the queue (called when offline).
  Future<void> enqueue(OfflineAction action) async {
    await _box.put(action.id, action);
  }

  /// Get pending action count for UI indicator.
  int get pendingCount => _box.length;

  /// Flush all queued actions to API (called when connectivity returns).
  Future<FlushResult> flush() async {
    final actions = _box.values.toList()
      ..sort((a, b) => a.createdAt.compareTo(b.createdAt));

    int succeeded = 0;
    int failed = 0;

    for (final action in actions) {
      try {
        await _executeAction(action);
        await _box.delete(action.id);
        succeeded++;
      } catch (e) {
        if (action.retryCount >= _maxRetries) {
          // Give up after max retries -- remove from queue
          await _box.delete(action.id);
          failed++;
        } else {
          // Increment retry count and keep in queue
          await _box.put(action.id, action.copyWithRetry());
          failed++;
        }
      }
    }

    return FlushResult(succeeded: succeeded, failed: failed);
  }

  Future<void> _executeAction(OfflineAction action) async {
    final data = action.body != null ? jsonDecode(action.body!) : null;

    switch (action.method) {
      case 'POST':
        await _api.post(action.endpoint, data: data);
      case 'PUT':
        await _api.put(action.endpoint, data: data);
      case 'DELETE':
        await _api.delete(action.endpoint);
    }
  }
}

class FlushResult {
  final int succeeded;
  final int failed;
  FlushResult({required this.succeeded, required this.failed});
}
```

### Usage in Repository

```dart
class TaskRepository {
  final ApiClient _api;
  final OfflineQueueService _queue;
  final ConnectivityService _connectivity;

  TaskRepository(this._api, this._queue, this._connectivity);

  Future<void> completeTask(Task task) async {
    if (await _connectivity.isOnline) {
      // Online: send directly
      await _api.post('/tasks', data: task.toJson());
    } else {
      // Offline: queue for later
      await _queue.enqueue(OfflineAction(
        endpoint: '/tasks',
        method: 'POST',
        body: jsonEncode(task.toJson()),
      ));
    }
  }
}
```

## Connectivity Monitoring

```dart
class ConnectivityService {
  final Connectivity _connectivity = Connectivity();
  late final StreamSubscription<List<ConnectivityResult>> _subscription;
  final OfflineQueueService _queue;

  final _controller = StreamController<bool>.broadcast();
  Stream<bool> get onConnectivityChanged => _controller.stream;

  bool _isOnline = true;
  bool get isOnline => _isOnline;

  ConnectivityService(this._queue);

  Future<void> init() async {
    final result = await _connectivity.checkConnectivity();
    _isOnline = !result.contains(ConnectivityResult.none);

    _subscription = _connectivity.onConnectivityChanged.listen((results) {
      final wasOffline = !_isOnline;
      _isOnline = !results.contains(ConnectivityResult.none);
      _controller.add(_isOnline);

      // Came back online -> flush the queue
      if (wasOffline && _isOnline) {
        _queue.flush();
      }
    });
  }

  void dispose() {
    _subscription.cancel();
    _controller.close();
  }
}
```

### Riverpod Provider

```dart
final connectivityProvider = StreamProvider<bool>((ref) {
  final service = ref.watch(connectivityServiceProvider);
  return service.onConnectivityChanged;
});

final isOnlineProvider = Provider<bool>((ref) {
  final connectivity = ref.watch(connectivityProvider);
  return connectivity.maybeWhen(data: (val) => val, orElse: () => true);
});
```

## Staleness Indicator

Show users whether they are seeing fresh or cached data.

```dart
class StalenessIndicator extends StatelessWidget {
  const StalenessIndicator({
    super.key,
    required this.lastUpdated,
    required this.isOnline,
  });

  final DateTime? lastUpdated;
  final bool isOnline;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    if (!isOnline) {
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: theme.colorScheme.errorContainer,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.cloud_off,
              size: 14,
              color: theme.colorScheme.onErrorContainer,
            ),
            const SizedBox(width: 6),
            Text(
              'Offline -- showing cached data',
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onErrorContainer,
              ),
            ),
          ],
        ),
      );
    }

    if (lastUpdated == null) return const SizedBox.shrink();

    final ago = DateTime.now().difference(lastUpdated!);
    final text = _formatDuration(ago);

    return Text(
      'Updated $text ago',
      style: theme.textTheme.labelSmall?.copyWith(
        color: theme.colorScheme.onSurfaceVariant,
      ),
    );
  }

  String _formatDuration(Duration duration) {
    if (duration.inMinutes < 1) return 'just now';
    if (duration.inMinutes < 60) return '${duration.inMinutes}m';
    if (duration.inHours < 24) return '${duration.inHours}h';
    return '${duration.inDays}d';
  }
}
```

**Usage in screen:**

```dart
@override
Widget build(BuildContext context) {
  final isOnline = ref.watch(isOnlineProvider);
  final tasks = ref.watch(tasksProvider);

  return Column(
    children: [
      StalenessIndicator(
        lastUpdated: tasks.valueOrNull != null ? _lastFetch : null,
        isOnline: isOnline,
      ),
      Expanded(
        child: tasks.when(
          data: (data) => TaskList(tasks: data),
          loading: () => const TaskListSkeleton(),
          error: (e, _) => AppErrorView(
            message: e.toString(),
            onRetry: () => ref.refresh(tasksProvider),
          ),
        ),
      ),
    ],
  );
}
```

## Pending Actions Indicator

Show user that queued actions exist.

```dart
class PendingActionsIndicator extends StatelessWidget {
  const PendingActionsIndicator({super.key, required this.count});
  final int count;

  @override
  Widget build(BuildContext context) {
    if (count == 0) return const SizedBox.shrink();

    return AppBadge(
      label: '$count pending',
      variant: AppBadgeVariant.warning,
      icon: Icons.sync,
    );
  }
}
```

## Conflict Resolution

Two strategies:

### Last-Write-Wins (Simple)

The offline action replays with a timestamp. The API accepts the latest timestamp. No conflict detection on client side.

### Server-Decides (Safe)

The offline action includes the `lastModified` timestamp of the data it was based on. The API compares and rejects if stale, returning a conflict error. The client shows the conflict to the user and lets them choose.

```dart
// In the API response handler after queue flush:
if (response.statusCode == 409) {
  // Conflict -- server has newer data
  // Show user a dialog to choose: keep server version or retry with local
}
```

**Default strategy:** Server-decides. The API always has the final say. Flutter is a bridge -- it does not resolve business conflicts.

## Storage Decision Guide

| Data Type | Storage | Example |
|-----------|---------|---------|
| Simple key-value | SharedPreferences | Auth token, theme pref, last sync time |
| Single JSON object | SharedPreferences | User profile cache |
| List of objects | Hive | Task history, offline queue |
| Large binary | File system | Cached images (handled by CachedNetworkImage) |
| Sensitive data | flutter_secure_storage | Tokens, passwords |
