---
knowledge-base-summary: "Provider types: Provider, StateNotifierProvider, FutureProvider, StreamProvider. AsyncValue pattern for API data (loading/data/error). Provider scoping, family providers for parameterized queries. When to use which provider type."
---
# State Management (Riverpod)

All async data flows through Riverpod. No raw `setState` for API data. `setState` is only acceptable for local UI state (animation toggles, form field visibility).

---

## Provider Types

### 1. Provider (Synchronous, Read-Only)

For computed values, configuration, and dependency injection.

```dart
// Dependency injection: expose a repository instance
final taskRepositoryProvider = Provider<TaskRepository>((ref) {
  final dio = ref.watch(dioProvider);
  return TaskRepositoryImpl(dio);
});

// Computed value from another provider
final completedTasksCountProvider = Provider<int>((ref) {
  final tasks = ref.watch(tasksProvider).valueOrNull ?? [];
  return tasks.where((w) => w.isCompleted).length;
});
```

### 2. FutureProvider (Async, Read-Only)

For one-time async data fetches. The most common provider for API data.

```dart
// Simple fetch
final tasksProvider = FutureProvider<List<Task>>((ref) async {
  final repository = ref.watch(taskRepositoryProvider);
  return repository.getTasks();
});

// Auto-dispose: free resources when no widget is listening
final tasksProvider = FutureProvider.autoDispose<List<Task>>((ref) async {
  final repository = ref.watch(taskRepositoryProvider);
  return repository.getTasks();
});
```

### 3. FutureProvider.family (Parameterized Async)

For fetching data that depends on a parameter (ID, filter, page).

```dart
// Single parameter
final taskDetailProvider = FutureProvider.family<Task, String>((ref, taskId) async {
  final repository = ref.watch(taskRepositoryProvider);
  return repository.getTaskById(taskId);
});

// Usage in widget
final taskAsync = ref.watch(taskDetailProvider(taskId));

// Multiple parameters: use a record or custom class
final tasksByFilterProvider = FutureProvider.family<List<Task>, ({String status, int page})>(
  (ref, params) async {
    final repository = ref.watch(taskRepositoryProvider);
    return repository.getTasks(status: params.status, page: params.page);
  },
);

// Usage
final tasks = ref.watch(tasksByFilterProvider((status: 'active', page: 1)));
```

### 4. StateNotifierProvider (Mutable State)

For complex local state that changes over time. Forms, filters, multi-step flows.

```dart
// State class (immutable)
@immutable
class TaskFilterState {
  const TaskFilterState({
    this.status = TaskStatus.all,
    this.dateRange,
    this.searchQuery = '',
    this.sortBy = TaskSortBy.newest,
  });

  final TaskStatus status;
  final DateTimeRange? dateRange;
  final String searchQuery;
  final TaskSortBy sortBy;

  TaskFilterState copyWith({
    TaskStatus? status,
    DateTimeRange? dateRange,
    String? searchQuery,
    TaskSortBy? sortBy,
  }) {
    return TaskFilterState(
      status: status ?? this.status,
      dateRange: dateRange ?? this.dateRange,
      searchQuery: searchQuery ?? this.searchQuery,
      sortBy: sortBy ?? this.sortBy,
    );
  }
}

// Notifier
class TaskFilterNotifier extends StateNotifier<TaskFilterState> {
  TaskFilterNotifier() : super(const TaskFilterState());

  void setStatus(TaskStatus status) {
    state = state.copyWith(status: status);
  }

  void setSearchQuery(String query) {
    state = state.copyWith(searchQuery: query);
  }

  void setDateRange(DateTimeRange? range) {
    state = state.copyWith(dateRange: range);
  }

  void setSortBy(TaskSortBy sortBy) {
    state = state.copyWith(sortBy: sortBy);
  }

  void reset() {
    state = const TaskFilterState();
  }
}

// Provider
final taskFilterProvider =
    StateNotifierProvider<TaskFilterNotifier, TaskFilterState>((ref) {
  return TaskFilterNotifier();
});

// Filtered tasks provider that reacts to filter changes
final filteredTasksProvider = FutureProvider<List<Task>>((ref) async {
  final filter = ref.watch(taskFilterProvider);
  final repository = ref.watch(taskRepositoryProvider);
  return repository.getTasks(
    status: filter.status,
    dateRange: filter.dateRange,
    search: filter.searchQuery,
    sortBy: filter.sortBy,
  );
});
```

### 5. StreamProvider (Real-Time Data)

For WebSocket streams, real-time updates, connectivity status.

```dart
// Connectivity stream
final connectivityProvider = StreamProvider<ConnectivityResult>((ref) {
  return Connectivity().onConnectivityChanged;
});

// Usage
final connectivity = ref.watch(connectivityProvider);
connectivity.when(
  data: (result) => result == ConnectivityResult.none
      ? const OfflineBanner()
      : const SizedBox.shrink(),
  loading: () => const SizedBox.shrink(),
  error: (_, __) => const SizedBox.shrink(),
);
```

### 6. StateProvider (Simple Mutable Value)

For a single value that changes. Tab index, toggle, selected item.

```dart
final selectedTabProvider = StateProvider<int>((ref) => 0);
final isDarkModeProvider = StateProvider<bool>((ref) => false);

// Usage
final tabIndex = ref.watch(selectedTabProvider);
ref.read(selectedTabProvider.notifier).state = 1;
```

---

## AsyncValue Pattern

The core pattern for handling async data in widgets. Every FutureProvider and StreamProvider returns an `AsyncValue`.

```dart
@override
Widget build(BuildContext context, WidgetRef ref) {
  final tasksAsync = ref.watch(tasksProvider);

  return tasksAsync.when(
    data: (tasks) => TasksList(tasks: tasks),
    loading: () => const LoadingView(),
    error: (error, stack) => ErrorView(
      message: error.toString(),
      onRetry: () => ref.invalidate(tasksProvider),
    ),
  );
}
```

### Preserving Previous Data During Refresh

When refreshing, `when` defaults to showing loading again. To keep old data visible during refresh:

```dart
// Option 1: skipLoadingOnRefresh (built-in)
tasksAsync.when(
  skipLoadingOnRefresh: true, // keeps showing old data during refresh
  data: (tasks) => TasksList(tasks: tasks),
  loading: () => const LoadingView(),
  error: (error, stack) => ErrorView(...),
);

// Option 2: use hasValue + isRefreshing
if (tasksAsync.isRefreshing) {
  // show a subtle indicator (e.g., linear progress bar at top)
}
final tasks = tasksAsync.valueOrNull;
if (tasks != null) {
  // render data even during refresh
}
```

### Combining Multiple Providers

When a screen needs data from multiple providers:

```dart
@override
Widget build(BuildContext context, WidgetRef ref) {
  final tasksAsync = ref.watch(tasksProvider);
  final profileAsync = ref.watch(profileProvider);

  // Both must be loaded
  if (tasksAsync.isLoading || profileAsync.isLoading) {
    return const LoadingView();
  }

  if (tasksAsync.hasError) {
    return ErrorView(message: tasksAsync.error.toString());
  }

  final tasks = tasksAsync.requireValue;
  final profile = profileAsync.requireValue;

  return DashboardContent(tasks: tasks, profile: profile);
}
```

---

## ref.watch vs ref.read

### ref.watch -- Reactive, Rebuild on Change

Use inside `build` method. Widget rebuilds when the provider value changes.

```dart
@override
Widget build(BuildContext context, WidgetRef ref) {
  // CORRECT: watch in build -- rebuilds when data changes
  final tasks = ref.watch(tasksProvider);
  final filter = ref.watch(taskFilterProvider);

  return ...;
}
```

### ref.read -- One-Time Read, No Rebuild

Use inside callbacks, event handlers, `onPressed`. Never in `build`.

```dart
@override
Widget build(BuildContext context, WidgetRef ref) {
  return ElevatedButton(
    onPressed: () {
      // CORRECT: read in callback
      ref.read(taskFilterProvider.notifier).reset();
    },
    child: Text('Reset'),
  );
}
```

### Common Mistake

```dart
// BAD: watch in callback -- causes unnecessary rebuilds
onPressed: () {
  ref.watch(taskFilterProvider.notifier).reset(); // WRONG
}

// BAD: read in build -- won't rebuild when data changes
final tasks = ref.read(tasksProvider); // WRONG in build method
```

---

## Invalidation and Refresh

### ref.invalidate -- Discard and Refetch

Marks the provider as dirty. Next time a widget reads it, it will refetch.

```dart
// After creating a new task, invalidate the list to refetch
Future<void> _createTask(WidgetRef ref, Task task) async {
  await ref.read(taskRepositoryProvider).createTask(task);
  ref.invalidate(tasksProvider); // will refetch on next read
}
```

### ref.refresh -- Invalidate + Immediately Read

Forces an immediate refetch and returns the new Future.

```dart
// Pull-to-refresh
RefreshIndicator(
  onRefresh: () => ref.refresh(tasksProvider.future),
  child: ...,
);
```

### When to Use Which

| Scenario | Use |
|----------|-----|
| After a mutation (create/update/delete) | `ref.invalidate` |
| Pull-to-refresh (need to await completion) | `ref.refresh(provider.future)` |
| User taps retry button | `ref.invalidate` |

---

## Provider Lifecycle and Scoping

### autoDispose

Providers are kept alive by default. Use `autoDispose` to free resources when no widget listens:

```dart
// Disposed when the screen is popped
final taskDetailProvider = FutureProvider.autoDispose.family<Task, String>(
  (ref, taskId) async {
    final repository = ref.watch(taskRepositoryProvider);
    return repository.getTaskById(taskId);
  },
);

// Keep alive for a duration even after listeners are gone
final tasksProvider = FutureProvider.autoDispose<List<Task>>((ref) async {
  // Keeps data cached for 30 seconds after last listener
  ref.keepAlive();
  final link = ref.keepAlive();
  Timer(const Duration(seconds: 30), link.close);

  final repository = ref.watch(taskRepositoryProvider);
  return repository.getTasks();
});
```

### Provider Scoping with ProviderScope

Override providers for different parts of the widget tree (useful for testing):

```dart
// In tests: override repository with a mock
ProviderScope(
  overrides: [
    taskRepositoryProvider.overrideWithValue(MockTaskRepository()),
  ],
  child: const TasksScreen(),
);
```

---

## Provider Organization

### File structure

```
lib/features/tasks/
  presentation/
    providers/
      tasks_provider.dart          # FutureProvider for task list
      task_detail_provider.dart    # FutureProvider.family for single task
      task_filter_provider.dart    # StateNotifierProvider for filters
```

### Naming Convention

| Provider Type | Naming Pattern | Example |
|--------------|----------------|---------|
| FutureProvider (list) | `{feature}Provider` | `tasksProvider` |
| FutureProvider.family (detail) | `{feature}DetailProvider` | `taskDetailProvider` |
| StateNotifierProvider | `{feature}{Purpose}Provider` | `taskFilterProvider` |
| StateProvider | `{purpose}Provider` | `selectedTabProvider` |
| StreamProvider | `{feature}StreamProvider` | `taskUpdatesStreamProvider` |

---

## Anti-Patterns

### 1. Provider doing business logic
```dart
// BAD: calculating in the provider
final bmiProvider = Provider<double>((ref) {
  final profile = ref.watch(profileProvider).valueOrNull;
  return profile!.weight / (profile.height * profile.height);
});

// GOOD: API returns calculated data
final profileProvider = FutureProvider<Profile>((ref) async {
  final repo = ref.watch(profileRepositoryProvider);
  return repo.getProfile(); // API returns bmi field
});
```

### 2. Deeply nested provider chains
```dart
// BAD: 5 providers chained together
// GOOD: flatten the chain, max 2-3 levels
```

### 3. Forgetting autoDispose on detail screens
```dart
// BAD: detail data stays in memory after navigating away
final taskDetailProvider = FutureProvider.family<Task, String>(...);

// GOOD: dispose when the detail screen is popped
final taskDetailProvider = FutureProvider.autoDispose.family<Task, String>(...);
```
