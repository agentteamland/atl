---
knowledge-base-summary: "Standard AsyncValue pattern: loading → data → error. Loading skeleton (shimmer effect). Error widget with retry button. Empty state widget with illustration + message. Consistent across all screens. Reusable LoadingView, ErrorView, EmptyView widgets."
---
# Error & Loading States

## Philosophy

Every async operation has three states: loading, success, error. Riverpod's `AsyncValue` makes this explicit. Every screen must handle all three. Reusable widgets ensure consistency across the app.

## AsyncValue.when() Pattern

The core pattern for displaying async data:

```dart
// Standard usage in any screen or widget
final tasks = ref.watch(tasksProvider);

tasks.when(
  data: (taskList) => TaskListView(tasks: taskList),
  loading: () => const LoadingView(),
  error: (error, stack) => ErrorView(
    message: _errorMessage(error),
    onRetry: () => ref.invalidate(tasksProvider),
  ),
);
```

**Rule:** Never use `tasks.value` directly. Always use `.when()` to handle all states explicitly. Using `.value` silently ignores loading and error states.

## Loading View (Shimmer Skeleton)

Shimmer loading shows placeholder shapes that match the actual content layout. This gives users a sense of what is coming and feels faster than a spinner.

### Shimmer Widget

```dart
// lib/core/widgets/shimmer.dart

import 'package:flutter/material.dart';

class Shimmer extends StatefulWidget {
  final Widget child;

  const Shimmer({super.key, required this.child});

  @override
  State<Shimmer> createState() => _ShimmerState();
}

class _ShimmerState extends State<Shimmer>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _animation;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1500),
    )..repeat();
    _animation = Tween<double>(begin: -1.0, end: 2.0).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeInOutSine),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _animation,
      builder: (context, child) {
        return ShaderMask(
          shaderCallback: (bounds) {
            return LinearGradient(
              begin: Alignment.centerLeft,
              end: Alignment.centerRight,
              colors: [
                Theme.of(context).colorScheme.surfaceContainerHighest,
                Theme.of(context).colorScheme.surfaceContainerHighest
                    .withAlpha(130),
                Theme.of(context).colorScheme.surfaceContainerHighest,
              ],
              stops: [
                _animation.value - 0.3,
                _animation.value,
                _animation.value + 0.3,
              ].map((s) => s.clamp(0.0, 1.0)).toList(),
            ).createShader(bounds);
          },
          blendMode: BlendMode.srcATop,
          child: child,
        );
      },
      child: widget.child,
    );
  }
}
```

### Shimmer Placeholder

```dart
// lib/core/widgets/shimmer_placeholder.dart

class ShimmerPlaceholder extends StatelessWidget {
  final double width;
  final double height;
  final double borderRadius;

  const ShimmerPlaceholder({
    super.key,
    this.width = double.infinity,
    required this.height,
    this.borderRadius = 8,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: width,
      height: height,
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(borderRadius),
      ),
    );
  }
}
```

### Loading View (Screen Level)

```dart
// lib/core/widgets/loading_view.dart

class LoadingView extends StatelessWidget {
  final int itemCount;

  const LoadingView({super.key, this.itemCount = 5});

  @override
  Widget build(BuildContext context) {
    return Shimmer(
      child: ListView.builder(
        physics: const NeverScrollableScrollPhysics(),
        itemCount: itemCount,
        padding: const EdgeInsets.all(16),
        itemBuilder: (context, index) => const _LoadingCard(),
      ),
    );
  }
}

class _LoadingCard extends StatelessWidget {
  const _LoadingCard();

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Row(
        children: [
          // Avatar placeholder
          const ShimmerPlaceholder(
            width: 48,
            height: 48,
            borderRadius: 24,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Title placeholder
                ShimmerPlaceholder(
                  height: 16,
                  width: MediaQuery.of(context).size.width * 0.5,
                ),
                const SizedBox(height: 8),
                // Subtitle placeholder
                const ShimmerPlaceholder(height: 12),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
```

### Feature-Specific Skeleton

Each feature can have its own skeleton that matches the content layout exactly:

```dart
// lib/features/home/widgets/task_card_skeleton.dart

class TaskCardSkeleton extends StatelessWidget {
  const TaskCardSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Route map placeholder
            const ShimmerPlaceholder(height: 120),
            const SizedBox(height: 12),
            // Title
            const ShimmerPlaceholder(height: 18, width: 180),
            const SizedBox(height: 8),
            // Stats row
            Row(
              children: const [
                ShimmerPlaceholder(height: 14, width: 60),
                SizedBox(width: 16),
                ShimmerPlaceholder(height: 14, width: 60),
                SizedBox(width: 16),
                ShimmerPlaceholder(height: 14, width: 80),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

// Usage in tasks loading
class TasksLoadingView extends StatelessWidget {
  const TasksLoadingView({super.key});

  @override
  Widget build(BuildContext context) {
    return Shimmer(
      child: ListView.builder(
        physics: const NeverScrollableScrollPhysics(),
        itemCount: 4,
        padding: const EdgeInsets.all(16),
        itemBuilder: (context, index) => const Padding(
          padding: EdgeInsets.only(bottom: 12),
          child: TaskCardSkeleton(),
        ),
      ),
    );
  }
}
```

## Error View

A reusable error widget with a message and retry button. Supports both full-screen and inline usage.

```dart
// lib/core/widgets/error_view.dart

class ErrorView extends StatelessWidget {
  final String message;
  final VoidCallback? onRetry;
  final IconData icon;

  const ErrorView({
    super.key,
    required this.message,
    this.onRetry,
    this.icon = Icons.error_outline,
  });

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              icon,
              size: 64,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 16),
            Text(
              message,
              style: Theme.of(context).textTheme.bodyLarge,
              textAlign: TextAlign.center,
            ),
            if (onRetry != null) ...[
              const SizedBox(height: 24),
              FilledButton.icon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh),
                label: Text(context.l10n.retry),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
```

## Error Categorization

Different errors deserve different messages and icons:

```dart
// lib/core/utils/error_mapper.dart

class ErrorMapper {
  static String message(BuildContext context, Object error) {
    if (error is DioException) {
      return _dioErrorMessage(context, error);
    }
    if (error is FormatException) {
      return context.l10n.unexpectedDataFormat;
    }
    return context.l10n.somethingWentWrong;
  }

  static String _dioErrorMessage(BuildContext context, DioException error) {
    switch (error.type) {
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.sendTimeout:
      case DioExceptionType.receiveTimeout:
        return context.l10n.connectionTimeout;

      case DioExceptionType.connectionError:
        return context.l10n.noInternetConnection;

      case DioExceptionType.badResponse:
        final statusCode = error.response?.statusCode ?? 0;
        if (statusCode == 404) return context.l10n.notFound;
        if (statusCode == 403) return context.l10n.accessDenied;
        if (statusCode == 422) return _validationMessage(context, error);
        if (statusCode >= 500) return context.l10n.serverError;
        return context.l10n.somethingWentWrong;

      case DioExceptionType.cancel:
        return context.l10n.requestCancelled;

      default:
        return context.l10n.somethingWentWrong;
    }
  }

  static String _validationMessage(
    BuildContext context,
    DioException error,
  ) {
    // Try to extract validation errors from API response
    final data = error.response?.data;
    if (data is Map<String, dynamic> && data.containsKey('errors')) {
      final errors = data['errors'] as Map<String, dynamic>;
      return errors.values.first.toString();
    }
    return context.l10n.validationError;
  }

  static IconData icon(Object error) {
    if (error is DioException) {
      switch (error.type) {
        case DioExceptionType.connectionTimeout:
        case DioExceptionType.connectionError:
          return Icons.wifi_off;
        case DioExceptionType.badResponse:
          final code = error.response?.statusCode ?? 0;
          if (code == 404) return Icons.search_off;
          if (code == 403) return Icons.lock;
          if (code >= 500) return Icons.cloud_off;
          return Icons.error_outline;
        default:
          return Icons.error_outline;
      }
    }
    return Icons.error_outline;
  }
}
```

## Empty View

Displayed when the API returns successfully but with no data:

```dart
// lib/core/widgets/empty_view.dart

class EmptyView extends StatelessWidget {
  final String message;
  final String? actionLabel;
  final VoidCallback? onAction;
  final IconData icon;

  const EmptyView({
    super.key,
    required this.message,
    this.actionLabel,
    this.onAction,
    this.icon = Icons.inbox_outlined,
  });

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              icon,
              size: 80,
              color: Theme.of(context).colorScheme.outline,
            ),
            const SizedBox(height: 16),
            Text(
              message,
              style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                    color: Theme.of(context).colorScheme.onSurfaceVariant,
                  ),
              textAlign: TextAlign.center,
            ),
            if (actionLabel != null && onAction != null) ...[
              const SizedBox(height: 24),
              FilledButton(
                onPressed: onAction,
                child: Text(actionLabel!),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
```

## Full-Screen vs Inline

| Situation | Widget | Example |
|-----------|--------|---------|
| Initial page load | Full-screen `LoadingView` | First time opening Home screen |
| List pagination error | Inline error at bottom of list | "Failed to load more. Tap to retry." |
| Form submission error | Snackbar | "Could not save task. Try again." |
| Section within a page | Inline loading/error within section | Stats card loading while tasks are visible |
| Pull-to-refresh error | Snackbar (list stays visible) | Network error during refresh |

### Inline Error for Pagination

```dart
// lib/core/widgets/inline_error.dart

class InlineError extends StatelessWidget {
  final String message;
  final VoidCallback? onRetry;

  const InlineError({
    super.key,
    required this.message,
    this.onRetry,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.error_outline,
            size: 16,
            color: Theme.of(context).colorScheme.error,
          ),
          const SizedBox(width: 8),
          Flexible(
            child: Text(
              message,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Theme.of(context).colorScheme.error,
                  ),
            ),
          ),
          if (onRetry != null) ...[
            const SizedBox(width: 8),
            TextButton(
              onPressed: onRetry,
              child: Text(context.l10n.retry),
            ),
          ],
        ],
      ),
    );
  }
}
```

### Snackbar for Transient Errors

```dart
// lib/core/utils/snackbar_utils.dart

void showErrorSnackbar(BuildContext context, String message) {
  ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(
      content: Text(message),
      behavior: SnackBarBehavior.floating,
      backgroundColor: Theme.of(context).colorScheme.error,
      action: SnackBarAction(
        label: context.l10n.dismiss,
        textColor: Theme.of(context).colorScheme.onError,
        onPressed: () {
          ScaffoldMessenger.of(context).hideCurrentSnackBar();
        },
      ),
    ),
  );
}

void showSuccessSnackbar(BuildContext context, String message) {
  ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(
      content: Text(message),
      behavior: SnackBarBehavior.floating,
      backgroundColor: Colors.green.shade700,
    ),
  );
}
```

## Complete Screen Example

Putting it all together in a real screen:

```dart
// lib/features/home/screens/home_screen.dart

class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final tasks = ref.watch(tasksProvider);

    return Scaffold(
      appBar: AppBar(title: Text(context.l10n.home)),
      floatingActionButton: FloatingActionButton(
        onPressed: () => context.push('/home/task/new'),
        child: const Icon(Icons.add),
      ),
      body: RefreshIndicator(
        onRefresh: () async {
          ref.invalidate(tasksProvider);
          // Wait for the new data to load
          await ref.read(tasksProvider.future);
        },
        child: tasks.when(
          // Success: show list or empty state
          data: (taskList) {
            if (taskList.isEmpty) {
              return EmptyView(
                message: context.l10n.noTasksYet,
                icon: Icons.check_circle_outline,
                actionLabel: context.l10n.startFirstTask,
                onAction: () => context.push('/home/task/new'),
              );
            }

            return ListView.builder(
              padding: const EdgeInsets.all(16),
              itemCount: taskList.length,
              itemBuilder: (context, index) => Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: TaskCard(
                  task: taskList[index],
                  onTap: () => context.push('/home/task/${taskList[index].id}'),
                ),
              ),
            );
          },

          // Loading: shimmer skeleton
          loading: () => const TasksLoadingView(),

          // Error: error view with retry
          error: (error, stack) => ErrorView(
            message: ErrorMapper.message(context, error),
            icon: ErrorMapper.icon(error),
            onRetry: () => ref.invalidate(tasksProvider),
          ),
        ),
      ),
    );
  }
}
```

## Retry Pattern

Retry always works by invalidating the provider. The provider re-fetches automatically:

```dart
// Invalidate causes the provider to re-execute its build/future
onRetry: () => ref.invalidate(tasksProvider),

// For parameterized providers (family)
onRetry: () => ref.invalidate(taskDetailProvider(taskId)),
```

Do NOT manually call API methods on retry. Let the provider handle it -- that is the single source of truth.
