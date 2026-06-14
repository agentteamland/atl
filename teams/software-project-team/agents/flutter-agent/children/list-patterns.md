---
knowledge-base-summary: "Infinite scroll with cursor-based pagination (matching API pattern). Pull-to-refresh. Empty state widget. Loading shimmer/skeleton. Error state with retry. Search with debounce. Filter chips. SliverList for performance."
---
# List Patterns

## Philosophy

Flutter is a BRIDGE. Lists display paginated data from the API. The API uses cursor-based pagination; Flutter mirrors it. The app fetches a page, displays it, detects "near bottom," fetches the next page. No client-side sorting or filtering of data -- that is the API's job.

## Dependencies

```yaml
dependencies:
  riverpod: ^2.5.0
  shimmer: ^3.0.0
```

## Pagination State

```dart
class PaginatedState<T> {
  final List<T> items;
  final String? cursor; // null = no more pages
  final bool isLoadingMore;
  final bool isRefreshing;
  final Object? error;

  const PaginatedState({
    this.items = const [],
    this.cursor,
    this.isLoadingMore = false,
    this.isRefreshing = false,
    this.error,
  });

  bool get hasMore => cursor != null;
  bool get isEmpty => items.isEmpty && !isLoadingMore && error == null;
  bool get hasError => error != null;

  PaginatedState<T> copyWith({
    List<T>? items,
    String? Function()? cursor,
    bool? isLoadingMore,
    bool? isRefreshing,
    Object? Function()? error,
  }) {
    return PaginatedState(
      items: items ?? this.items,
      cursor: cursor != null ? cursor() : this.cursor,
      isLoadingMore: isLoadingMore ?? this.isLoadingMore,
      isRefreshing: isRefreshing ?? this.isRefreshing,
      error: error != null ? error() : this.error,
    );
  }
}
```

## Pagination Notifier

```dart
class PaginatedNotifier<T> extends StateNotifier<PaginatedState<T>> {
  final Future<PaginatedResponse<T>> Function(String? cursor, int pageSize) _fetcher;
  final int _pageSize;

  PaginatedNotifier({
    required Future<PaginatedResponse<T>> Function(String? cursor, int pageSize) fetcher,
    int pageSize = 20,
  })  : _fetcher = fetcher,
        _pageSize = pageSize,
        super(const PaginatedState()) {
    loadFirstPage();
  }

  /// Load the first page. Called on init and pull-to-refresh.
  Future<void> loadFirstPage() async {
    state = state.copyWith(
      isRefreshing: () => true,
      error: () => null,
    );

    try {
      final response = await _fetcher(null, _pageSize);
      state = PaginatedState(
        items: response.items,
        cursor: response.nextCursor,
      );
    } catch (e) {
      state = state.copyWith(
        isRefreshing: () => false,
        error: () => e,
      );
    }
  }

  /// Load the next page. Called when user scrolls near bottom.
  Future<void> loadNextPage() async {
    if (!state.hasMore || state.isLoadingMore) return;

    state = state.copyWith(isLoadingMore: () => true);

    try {
      final response = await _fetcher(state.cursor, _pageSize);
      state = state.copyWith(
        items: [...state.items, ...response.items],
        cursor: () => response.nextCursor,
        isLoadingMore: () => false,
      );
    } catch (e) {
      state = state.copyWith(
        isLoadingMore: () => false,
        error: () => e,
      );
    }
  }

  /// Refresh: reload from scratch.
  Future<void> refresh() async {
    await loadFirstPage();
  }
}

class PaginatedResponse<T> {
  final List<T> items;
  final String? nextCursor; // null means no more pages

  const PaginatedResponse({
    required this.items,
    this.nextCursor,
  });
}
```

## Riverpod Provider

```dart
// Repository method
class TaskRepository {
  final ApiClient _api;

  TaskRepository(this._api);

  Future<PaginatedResponse<Task>> getTasks({
    String? cursor,
    int pageSize = 20,
    String? searchQuery,
  }) async {
    final response = await _api.get('/tasks', queryParameters: {
      'pageSize': pageSize,
      if (cursor != null) 'cursor': cursor,
      if (searchQuery != null && searchQuery.isNotEmpty) 'q': searchQuery,
    });

    final data = response.data as Map<String, dynamic>;
    final items = (data['items'] as List)
        .map((e) => Task.fromJson(e as Map<String, dynamic>))
        .toList();

    return PaginatedResponse(
      items: items,
      nextCursor: data['nextCursor'] as String?,
    );
  }
}

// Provider
final tasksNotifierProvider =
    StateNotifierProvider.autoDispose<PaginatedNotifier<Task>, PaginatedState<Task>>(
  (ref) {
    final repository = ref.watch(taskRepositoryProvider);
    return PaginatedNotifier(
      fetcher: (cursor, pageSize) => repository.getTasks(
        cursor: cursor,
        pageSize: pageSize,
      ),
    );
  },
);
```

## Complete Paginated List Screen

```dart
class TasksScreen extends ConsumerStatefulWidget {
  const TasksScreen({super.key});

  @override
  ConsumerState<TasksScreen> createState() => _TasksScreenState();
}

class _TasksScreenState extends ConsumerState<TasksScreen> {
  final _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    // Trigger load when 200px from the bottom
    if (_scrollController.position.pixels >=
        _scrollController.position.maxScrollExtent - 200) {
      ref.read(tasksNotifierProvider.notifier).loadNextPage();
    }
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(tasksNotifierProvider);

    return Scaffold(
      appBar: AppBar(title: Text(context.l10n.myTasks)),
      body: _buildBody(state),
    );
  }

  Widget _buildBody(PaginatedState<Task> state) {
    // Initial loading
    if (state.items.isEmpty && state.isRefreshing) {
      return const TaskListSkeleton();
    }

    // Error on first load
    if (state.items.isEmpty && state.hasError) {
      return AppErrorView(
        message: state.error.toString(),
        onRetry: () => ref.read(tasksNotifierProvider.notifier).refresh(),
      );
    }

    // Empty state
    if (state.isEmpty) {
      return AppEmptyState(
        icon: Icons.check_circle_outline,
        title: context.l10n.noTasksYet,
        subtitle: context.l10n.startYourFirstTask,
        actionLabel: context.l10n.startTasking,
        onAction: () => context.push('/tasks/new'),
      );
    }

    // Data loaded
    return RefreshIndicator(
      onRefresh: () => ref.read(tasksNotifierProvider.notifier).refresh(),
      child: CustomScrollView(
        controller: _scrollController,
        slivers: [
          // Collapsible header (optional)
          SliverAppBar(
            expandedHeight: 120,
            floating: true,
            snap: true,
            flexibleSpace: FlexibleSpaceBar(
              title: Text(
                '${state.items.length} tasks',
                style: const TextStyle(fontSize: 14),
              ),
            ),
          ),

          // List items
          SliverPadding(
            padding: const EdgeInsets.all(16),
            sliver: SliverList.builder(
              itemCount: state.items.length + (state.hasMore ? 1 : 0),
              itemBuilder: (context, index) {
                // Loading indicator at bottom
                if (index == state.items.length) {
                  return const Padding(
                    padding: EdgeInsets.symmetric(vertical: 16),
                    child: Center(child: CircularProgressIndicator()),
                  );
                }

                final task = state.items[index];
                return Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: TaskCard(
                    task: task,
                    onTap: () => context.push('/tasks/${task.id}'),
                  ),
                );
              },
            ),
          ),

          // Error while loading more
          if (state.hasError && state.items.isNotEmpty)
            SliverToBoxAdapter(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: AppButton(
                  label: context.l10n.retry,
                  onPressed: () {
                    ref.read(tasksNotifierProvider.notifier).loadNextPage();
                  },
                  variant: AppButtonVariant.outline,
                ),
              ),
            ),
        ],
      ),
    );
  }
}
```

## Empty State Widget

```dart
class AppEmptyState extends StatelessWidget {
  const AppEmptyState({
    super.key,
    required this.icon,
    required this.title,
    required this.subtitle,
    this.actionLabel,
    this.onAction,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              icon,
              size: 80,
              color: theme.colorScheme.onSurfaceVariant.withOpacity(0.5),
            ),
            const SizedBox(height: 16),
            Text(
              title,
              style: theme.textTheme.titleLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              subtitle,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            if (actionLabel != null && onAction != null) ...[
              const SizedBox(height: 24),
              AppButton(
                label: actionLabel!,
                onPressed: onAction,
                icon: Icons.add,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
```

## Error State Widget

```dart
class AppErrorView extends StatelessWidget {
  const AppErrorView({
    super.key,
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.error_outline,
              size: 64,
              color: theme.colorScheme.error,
            ),
            const SizedBox(height: 16),
            Text(
              'Something went wrong',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              message,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 24),
            AppButton(
              label: 'Retry',
              onPressed: onRetry,
              icon: Icons.refresh,
              variant: AppButtonVariant.outline,
            ),
          ],
        ),
      ),
    );
  }
}
```

## Shimmer Skeleton Loading

Show placeholder shapes while data is loading. Better UX than a spinner.

```dart
class TaskListSkeleton extends StatelessWidget {
  const TaskListSkeleton({super.key, this.itemCount = 5});

  final int itemCount;

  @override
  Widget build(BuildContext context) {
    return Shimmer.fromColors(
      baseColor: Theme.of(context).colorScheme.surfaceContainerHighest,
      highlightColor: Theme.of(context).colorScheme.surface,
      child: ListView.builder(
        padding: const EdgeInsets.all(16),
        physics: const NeverScrollableScrollPhysics(),
        itemCount: itemCount,
        itemBuilder: (context, index) => const _TaskCardSkeleton(),
      ),
    );
  }
}

class _TaskCardSkeleton extends StatelessWidget {
  const _TaskCardSkeleton();

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Container(
        height: 100,
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              // Avatar placeholder
              Container(
                width: 48,
                height: 48,
                decoration: const BoxDecoration(
                  color: Colors.white,
                  shape: BoxShape.circle,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    // Title placeholder
                    Container(
                      width: double.infinity,
                      height: 14,
                      decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(4),
                      ),
                    ),
                    const SizedBox(height: 8),
                    // Subtitle placeholder
                    Container(
                      width: 150,
                      height: 12,
                      decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(4),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
```

## Search with Debounce

Search input that debounces API calls to avoid spamming the server.

```dart
class SearchableTasksScreen extends ConsumerStatefulWidget {
  const SearchableTasksScreen({super.key});

  @override
  ConsumerState<SearchableTasksScreen> createState() =>
      _SearchableTasksScreenState();
}

class _SearchableTasksScreenState
    extends ConsumerState<SearchableTasksScreen> {
  final _searchController = TextEditingController();
  Timer? _debounce;

  @override
  void dispose() {
    _debounce?.cancel();
    _searchController.dispose();
    super.dispose();
  }

  void _onSearchChanged(String query) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 400), () {
      ref.read(taskSearchQueryProvider.notifier).state = query;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: TextField(
          controller: _searchController,
          onChanged: _onSearchChanged,
          decoration: InputDecoration(
            hintText: context.l10n.searchTasks,
            border: InputBorder.none,
            prefixIcon: const Icon(Icons.search),
            suffixIcon: _searchController.text.isNotEmpty
                ? IconButton(
                    icon: const Icon(Icons.clear),
                    onPressed: () {
                      _searchController.clear();
                      _onSearchChanged('');
                    },
                  )
                : null,
          ),
        ),
      ),
      body: const TaskSearchResults(),
    );
  }
}

// Search query provider
final taskSearchQueryProvider = StateProvider<String>((ref) => '');

// Filtered tasks provider (re-fetches when query changes)
final searchedTasksProvider = StateNotifierProvider.autoDispose<
    PaginatedNotifier<Task>, PaginatedState<Task>>((ref) {
  final repository = ref.watch(taskRepositoryProvider);
  final query = ref.watch(taskSearchQueryProvider);

  return PaginatedNotifier(
    fetcher: (cursor, pageSize) => repository.getTasks(
      cursor: cursor,
      pageSize: pageSize,
      searchQuery: query,
    ),
  );
});
```

## SliverList + SliverAppBar (Collapsible Header)

For screens with a collapsible header that hides on scroll.

```dart
CustomScrollView(
  controller: _scrollController,
  slivers: [
    SliverAppBar(
      expandedHeight: 200,
      pinned: true,
      flexibleSpace: FlexibleSpaceBar(
        title: Text(context.l10n.myTasks),
        background: Image.asset(
          AppAssets.tasksHeader,
          fit: BoxFit.cover,
        ),
      ),
    ),
    SliverPadding(
      padding: const EdgeInsets.all(16),
      sliver: SliverList.builder(
        itemCount: state.items.length,
        itemBuilder: (context, index) {
          return TaskCard(task: state.items[index]);
        },
      ),
    ),
  ],
)
```

## Pull-to-Refresh

Wrap the scroll view with `RefreshIndicator`. Calls the notifier's refresh method.

```dart
RefreshIndicator(
  onRefresh: () => ref.read(tasksNotifierProvider.notifier).refresh(),
  child: ListView.builder(
    controller: _scrollController,
    itemCount: state.items.length,
    itemBuilder: (context, index) {
      return TaskCard(task: state.items[index]);
    },
  ),
)
```

**Important:** `RefreshIndicator` only works with scrollable widgets. If the list is empty, wrap the empty state in a `SingleChildScrollView` with `AlwaysScrollableScrollPhysics` so pull-to-refresh still works.

```dart
if (state.isEmpty) {
  return RefreshIndicator(
    onRefresh: () => ref.read(tasksNotifierProvider.notifier).refresh(),
    child: SingleChildScrollView(
      physics: const AlwaysScrollableScrollPhysics(),
      child: SizedBox(
        height: MediaQuery.of(context).size.height * 0.7,
        child: AppEmptyState(
          icon: Icons.check_circle_outline,
          title: context.l10n.noTasksYet,
          subtitle: context.l10n.startYourFirstTask,
        ),
      ),
    ),
  );
}
```

## Key Rules

1. **Cursor-based pagination** -- matches API pattern. No offset/page numbers.
2. **Load next page at 200px from bottom** -- smooth infinite scroll experience.
3. **Guard against duplicate loads** -- check `isLoadingMore` and `hasMore` before fetching.
4. **Shimmer skeleton** -- always show on first load. Never a plain spinner.
5. **Empty, error, loading** -- every list must handle all three states.
6. **Pull-to-refresh** -- always available, even on empty state.
7. **Debounce search** -- 400ms delay before sending search query to API.
8. **No client-side sorting** -- API sorts. Flutter displays.
9. **Dispose scroll controller** -- prevent memory leaks.
10. **SliverList for performance** -- lazy rendering for large lists.
