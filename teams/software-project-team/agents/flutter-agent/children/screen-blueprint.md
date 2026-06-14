---
knowledge-base-summary: "The primary production unit of this agent. Template + checklist for creating new screens. Covers: file structure, Riverpod consumer widget, responsive layout, loading/error/empty states, navigation registration, i18n strings."
---
# Screen Blueprint (Primary Production Unit)

This is the primary production unit of the Flutter Agent. Every new screen follows this blueprint exactly.

## File Structure

```
lib/features/{feature}/
  data/
    models/{feature}_model.dart
    repositories/{feature}_repository.dart
  presentation/
    providers/{feature}_provider.dart
    screens/{screen}_screen.dart
    widgets/{widget}_widget.dart
```

Every feature is self-contained. No cross-feature imports at the presentation layer.

---

## Screen Template

Every screen is a `ConsumerWidget` (or `ConsumerStatefulWidget` if local animation/controller state is needed). The screen fetches data through a Riverpod provider and handles all three async states.

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:example_app/core/extensions/context_extensions.dart';
import 'package:example_app/core/widgets/loading_view.dart';
import 'package:example_app/core/widgets/error_view.dart';
import 'package:example_app/core/widgets/empty_view.dart';
import 'package:example_app/core/widgets/responsive_layout.dart';
import 'package:example_app/features/tasks/presentation/providers/tasks_provider.dart';
import 'package:example_app/features/tasks/presentation/widgets/task_list_tile.dart';
import 'package:example_app/features/tasks/presentation/widgets/task_grid_card.dart';

class TasksScreen extends ConsumerWidget {
  const TasksScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final tasksAsync = ref.watch(tasksProvider);

    return Scaffold(
      appBar: AppBar(
        title: Text(context.l10n.tasks_title),
      ),
      body: tasksAsync.when(
        data: (tasks) {
          if (tasks.isEmpty) {
            return EmptyView(
              icon: Icons.check_circle_outline,
              title: context.l10n.tasks_empty_title,
              subtitle: context.l10n.tasks_empty_subtitle,
            );
          }

          return RefreshIndicator(
            onRefresh: () => ref.refresh(tasksProvider.future),
            child: ResponsiveLayout(
              mobile: _MobileLayout(tasks: tasks),
              tablet: _TabletLayout(tasks: tasks),
            ),
          );
        },
        loading: () => const LoadingView(),
        error: (error, stack) => ErrorView(
          message: error.toString(),
          onRetry: () => ref.invalidate(tasksProvider),
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () => context.push('/tasks/new'),
        child: const Icon(Icons.add),
      ),
    );
  }
}

class _MobileLayout extends StatelessWidget {
  const _MobileLayout({required this.tasks});
  final List<Task> tasks;

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: tasks.length,
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (context, index) => TaskListTile(task: tasks[index]),
    );
  }
}

class _TabletLayout extends StatelessWidget {
  const _TabletLayout({required this.tasks});
  final List<Task> tasks;

  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      padding: const EdgeInsets.all(24),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        crossAxisSpacing: 16,
        mainAxisSpacing: 16,
        childAspectRatio: 1.5,
      ),
      itemCount: tasks.length,
      itemBuilder: (context, index) => TaskGridCard(task: tasks[index]),
    );
  }
}
```

---

## ConsumerStatefulWidget (When Needed)

Use `ConsumerStatefulWidget` ONLY when you need:
- `TextEditingController`, `ScrollController`, `AnimationController`
- `TickerProviderStateMixin` for animations
- `initState`/`dispose` lifecycle

```dart
class TaskDetailScreen extends ConsumerStatefulWidget {
  const TaskDetailScreen({super.key, required this.taskId});
  final String taskId;

  @override
  ConsumerState<TaskDetailScreen> createState() => _TaskDetailScreenState();
}

class _TaskDetailScreenState extends ConsumerState<TaskDetailScreen> {
  late final ScrollController _scrollController;

  @override
  void initState() {
    super.initState();
    _scrollController = ScrollController();
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final taskAsync = ref.watch(taskDetailProvider(widget.taskId));

    return Scaffold(
      body: taskAsync.when(
        data: (task) => CustomScrollView(
          controller: _scrollController,
          slivers: [
            SliverAppBar(
              expandedHeight: 200,
              flexibleSpace: FlexibleSpaceBar(
                title: Text(task.title),
              ),
            ),
            SliverPadding(
              padding: const EdgeInsets.all(16),
              sliver: SliverToBoxAdapter(
                child: _TaskContent(task: task),
              ),
            ),
          ],
        ),
        loading: () => const LoadingView(),
        error: (error, stack) => ErrorView(
          message: error.toString(),
          onRetry: () => ref.invalidate(taskDetailProvider(widget.taskId)),
        ),
      ),
    );
  }
}
```

---

## Route Registration

Every screen must be registered in the router. Add the route to the appropriate section in `lib/core/routing/app_router.dart`:

```dart
GoRoute(
  path: '/tasks',
  name: 'tasks',
  builder: (context, state) => const TasksScreen(),
  routes: [
    GoRoute(
      path: 'new',
      name: 'tasks-new',
      builder: (context, state) => const NewTaskScreen(),
    ),
    GoRoute(
      path: ':taskId',
      name: 'tasks-detail',
      builder: (context, state) {
        final taskId = state.pathParameters['taskId']!;
        return TaskDetailScreen(taskId: taskId);
      },
    ),
  ],
),
```

---

## i18n Strings

Every user-facing string must be added to both ARB files before the screen is complete.

**`lib/l10n/app_en.arb`:**
```json
{
  "tasks_title": "My Tasks",
  "tasks_empty_title": "No tasks yet",
  "tasks_empty_subtitle": "Start your first task by tapping the + button",
  "tasks_error_load": "Could not load tasks"
}
```

**`lib/l10n/app_<locale>.arb`** (any non-default locale): mirror the same keys with translated values. Code generation fails if any key is missing.

---

## Provider for the Screen

Every screen has a corresponding provider that fetches data from the repository:

```dart
// lib/features/tasks/presentation/providers/tasks_provider.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:example_app/features/tasks/data/repositories/task_repository.dart';
import 'package:example_app/features/tasks/data/models/task_model.dart';

final tasksProvider = FutureProvider<List<Task>>((ref) async {
  final repository = ref.watch(taskRepositoryProvider);
  return repository.getTasks();
});

final taskDetailProvider = FutureProvider.family<Task, String>((ref, taskId) async {
  final repository = ref.watch(taskRepositoryProvider);
  return repository.getTaskById(taskId);
});
```

---

## Loading / Error / Empty State Widgets

These are shared across all screens. They live in `lib/core/widgets/`:

```dart
// lib/core/widgets/loading_view.dart
class LoadingView extends StatelessWidget {
  const LoadingView({super.key});

  @override
  Widget build(BuildContext context) {
    return const Center(
      child: CircularProgressIndicator(),
    );
  }
}

// lib/core/widgets/error_view.dart
class ErrorView extends StatelessWidget {
  const ErrorView({
    super.key,
    required this.message,
    this.onRetry,
  });

  final String message;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 64,
              color: context.colors.error,
            ),
            const SizedBox(height: 16),
            Text(
              message,
              textAlign: TextAlign.center,
              style: context.text.bodyLarge,
            ),
            if (onRetry != null) ...[
              const SizedBox(height: 24),
              FilledButton.icon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh),
                label: Text(context.l10n.common_retry),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

// lib/core/widgets/empty_view.dart
class EmptyView extends StatelessWidget {
  const EmptyView({
    super.key,
    required this.icon,
    required this.title,
    this.subtitle,
    this.action,
  });

  final IconData icon;
  final String title;
  final String? subtitle;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 80, color: context.colors.outline),
            const SizedBox(height: 16),
            Text(
              title,
              style: context.text.titleMedium,
              textAlign: TextAlign.center,
            ),
            if (subtitle != null) ...[
              const SizedBox(height: 8),
              Text(
                subtitle!,
                style: context.text.bodyMedium?.copyWith(
                  color: context.colors.onSurfaceVariant,
                ),
                textAlign: TextAlign.center,
              ),
            ],
            if (action != null) ...[
              const SizedBox(height: 24),
              action!,
            ],
          ],
        ),
      ),
    );
  }
}
```

---

## Checklist

Before a screen is considered complete, every item must be checked:

| # | Item | Details |
|---|------|---------|
| 1 | **File location** | `lib/features/{feature}/presentation/screens/{screen}_screen.dart` |
| 2 | **ConsumerWidget** | Uses `ConsumerWidget` (or `ConsumerStatefulWidget` only if controllers needed) |
| 3 | **Provider** | Screen has a corresponding provider in `providers/` |
| 4 | **AsyncValue** | `.when(data:, loading:, error:)` pattern used |
| 5 | **Loading state** | Shows `LoadingView` or shimmer skeleton |
| 6 | **Error state** | Shows `ErrorView` with retry button |
| 7 | **Empty state** | Shows `EmptyView` with icon, title, subtitle |
| 8 | **Responsive** | `ResponsiveLayout` with separate mobile and tablet builders |
| 9 | **i18n** | All strings use `context.l10n.key`. Both EN and TR ARB files updated |
| 10 | **Route** | Registered in `app_router.dart` with name and path |
| 11 | **Theme** | Colors from `Theme.of(context)`, no hardcoded colors |
| 12 | **RefreshIndicator** | Pull-to-refresh on list screens |
| 13 | **Navigation** | `context.push`/`context.go` for navigation, not `Navigator.push` |
| 14 | **SafeArea** | Handled (Scaffold does it by default, check custom layouts) |
| 15 | **Tests** | Widget test file created at `test/features/{feature}/presentation/screens/{screen}_screen_test.dart` |

---

## Anti-Patterns (Never Do These)

### 1. Business logic in widget
```dart
// BAD: calculating in the widget
final bmi = weight / (height * height);
// GOOD: API returns the calculated value, widget only displays it
Text(task.bmiFormatted)
```

### 2. Raw setState for API data
```dart
// BAD: fetching data with setState
void initState() {
  super.initState();
  setState(() => _isLoading = true);
  api.getTasks().then((data) {
    setState(() { _tasks = data; _isLoading = false; });
  });
}
// GOOD: use a Riverpod provider
final tasksAsync = ref.watch(tasksProvider);
```

### 3. Hardcoded strings
```dart
// BAD
Text('My Tasks')
// GOOD
Text(context.l10n.tasks_title)
```

### 4. Hardcoded colors
```dart
// BAD
Container(color: Color(0xFF2196F3))
// GOOD
Container(color: context.colors.primary)
```

### 5. Navigator.push instead of go_router
```dart
// BAD
Navigator.push(context, MaterialPageRoute(builder: (_) => TaskDetailScreen()));
// GOOD
context.push('/tasks/${task.id}');
```

### 6. Ignoring error/empty states
```dart
// BAD: only handles data
body: ListView.builder(itemCount: tasks.length, ...)
// GOOD: handles all three states
body: tasksAsync.when(data: ..., loading: ..., error: ...)
```

### 7. One giant widget
```dart
// BAD: 500-line build method
// GOOD: extract private widgets or separate widget files
class _TaskHeader extends StatelessWidget { ... }
class _TaskStats extends StatelessWidget { ... }
```
