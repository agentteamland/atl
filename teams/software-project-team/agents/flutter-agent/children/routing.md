---
knowledge-base-summary: "Route definition, GoRoute, ShellRoute for nested navigation. Auth guard (redirect if not logged in). Deep linking support. Route naming convention. Passing parameters. Bottom navigation with nested routes."
---
# Routing (go_router)

All navigation uses `go_router`. No `Navigator.push` anywhere. Routes are defined centrally, navigation is declarative, and auth guards protect authenticated routes.

---

## Router Configuration

The router lives in `lib/core/routing/app_router.dart` and is exposed as a Riverpod provider:

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:example_app/core/routing/routes.dart';
import 'package:example_app/features/auth/presentation/providers/auth_provider.dart';

final routerProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authStateProvider);

  return GoRouter(
    initialLocation: '/home',
    debugLogDiagnostics: true,
    refreshListenable: authState,
    redirect: (context, state) {
      final isLoggedIn = authState.isAuthenticated;
      final isAuthRoute = state.matchedLocation.startsWith('/auth');

      // Not logged in and not on auth page -> redirect to login
      if (!isLoggedIn && !isAuthRoute) {
        return '/auth/login';
      }

      // Logged in and on auth page -> redirect to home
      if (isLoggedIn && isAuthRoute) {
        return '/home';
      }

      // No redirect needed
      return null;
    },
    routes: $appRoutes,
    errorBuilder: (context, state) => NotFoundScreen(
      path: state.uri.toString(),
    ),
  );
});
```

### Using the Router in MaterialApp

```dart
class App extends ConsumerWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);

    return MaterialApp.router(
      routerConfig: router,
      title: 'ExampleApp',
      theme: AppTheme.light(),
      darkTheme: AppTheme.dark(),
      themeMode: ref.watch(themeModeProvider),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      locale: ref.watch(localeProvider),
    );
  }
}
```

---

## Route Definitions

### Simple Routes (GoRoute)

```dart
GoRoute(
  path: '/home',
  name: 'home',
  builder: (context, state) => const HomeScreen(),
),
GoRoute(
  path: '/profile',
  name: 'profile',
  builder: (context, state) => const ProfileScreen(),
),
```

### Nested Routes

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
      routes: [
        GoRoute(
          path: 'edit',
          name: 'tasks-edit',
          builder: (context, state) {
            final taskId = state.pathParameters['taskId']!;
            return EditTaskScreen(taskId: taskId);
          },
        ),
      ],
    ),
  ],
),
```

### Auth Routes (Public)

```dart
GoRoute(
  path: '/auth',
  name: 'auth',
  redirect: (context, state) {
    // /auth alone redirects to /auth/login
    if (state.matchedLocation == '/auth') return '/auth/login';
    return null;
  },
  routes: [
    GoRoute(
      path: 'login',
      name: 'auth-login',
      builder: (context, state) => const LoginScreen(),
    ),
    GoRoute(
      path: 'register',
      name: 'auth-register',
      builder: (context, state) => const RegisterScreen(),
    ),
    GoRoute(
      path: 'forgot-password',
      name: 'auth-forgot-password',
      builder: (context, state) => const ForgotPasswordScreen(),
    ),
    GoRoute(
      path: 'verify-email',
      name: 'auth-verify-email',
      builder: (context, state) {
        final token = state.uri.queryParameters['token'];
        return VerifyEmailScreen(token: token);
      },
    ),
  ],
),
```

---

## Bottom Navigation with StatefulShellRoute

For main app navigation with persistent tabs:

```dart
StatefulShellRoute.indexedStack(
  builder: (context, state, navigationShell) {
    return MainShell(navigationShell: navigationShell);
  },
  branches: [
    // Home tab
    StatefulShellBranch(
      routes: [
        GoRoute(
          path: '/home',
          name: 'home',
          builder: (context, state) => const HomeScreen(),
        ),
      ],
    ),
    // Tasks tab
    StatefulShellBranch(
      routes: [
        GoRoute(
          path: '/tasks',
          name: 'tasks',
          builder: (context, state) => const TasksScreen(),
          routes: [
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
      ],
    ),
    // Stats tab
    StatefulShellBranch(
      routes: [
        GoRoute(
          path: '/stats',
          name: 'stats',
          builder: (context, state) => const StatsScreen(),
        ),
      ],
    ),
    // Profile tab
    StatefulShellBranch(
      routes: [
        GoRoute(
          path: '/profile',
          name: 'profile',
          builder: (context, state) => const ProfileScreen(),
        ),
      ],
    ),
  ],
),
```

### MainShell Widget

```dart
class MainShell extends StatelessWidget {
  const MainShell({super.key, required this.navigationShell});
  final StatefulNavigationShell navigationShell;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: navigationShell,
      bottomNavigationBar: NavigationBar(
        selectedIndex: navigationShell.currentIndex,
        onDestinationSelected: (index) {
          navigationShell.goBranch(
            index,
            initialLocation: index == navigationShell.currentIndex,
          );
        },
        destinations: [
          NavigationDestination(
            icon: const Icon(Icons.home_outlined),
            selectedIcon: const Icon(Icons.home),
            label: context.l10n.nav_home,
          ),
          NavigationDestination(
            icon: const Icon(Icons.directions_task_outlined),
            selectedIcon: const Icon(Icons.check_circle_outline),
            label: context.l10n.nav_tasks,
          ),
          NavigationDestination(
            icon: const Icon(Icons.bar_chart_outlined),
            selectedIcon: const Icon(Icons.bar_chart),
            label: context.l10n.nav_stats,
          ),
          NavigationDestination(
            icon: const Icon(Icons.person_outline),
            selectedIcon: const Icon(Icons.person),
            label: context.l10n.nav_profile,
          ),
        ],
      ),
    );
  }
}
```

---

## Navigation Methods

### context.go -- Replace the Entire Stack

Replaces the navigation stack. Use for top-level navigation (switching tabs, going to a completely different section).

```dart
// Go to home (clears stack)
context.go('/home');

// After logout, go to login (clears authenticated routes from stack)
context.go('/auth/login');
```

### context.push -- Push on Top of Stack

Adds a route on top. Use for detail screens, modals, and drill-down navigation.

```dart
// Push detail screen (back button returns to list)
context.push('/tasks/${task.id}');

// Push with query parameters
context.push('/tasks?status=active');
```

### context.pop -- Go Back

Pops the current route. Same as back button.

```dart
// Go back
context.pop();

// Pop with result
context.pop(selectedTask);
```

### context.pushNamed -- Named Route Navigation

```dart
context.pushNamed(
  'tasks-detail',
  pathParameters: {'taskId': task.id},
);

context.goNamed(
  'auth-verify-email',
  queryParameters: {'token': verificationToken},
);
```

---

## Path Parameters and Query Parameters

### Path Parameters

For required identifiers (IDs, slugs):

```dart
// Definition
GoRoute(
  path: ':taskId',
  builder: (context, state) {
    final taskId = state.pathParameters['taskId']!;
    return TaskDetailScreen(taskId: taskId);
  },
),

// Navigation
context.push('/tasks/abc-123');
```

### Query Parameters

For optional filters, search terms, tokens:

```dart
// Definition
GoRoute(
  path: '/tasks',
  builder: (context, state) {
    final status = state.uri.queryParameters['status'];
    final page = int.tryParse(state.uri.queryParameters['page'] ?? '');
    return TasksScreen(initialStatus: status, initialPage: page);
  },
),

// Navigation
context.push('/tasks?status=active&page=2');
```

---

## Route Guard (Auth Redirect)

The global redirect in `GoRouter` handles authentication. It fires on every navigation:

```dart
redirect: (context, state) {
  final isLoggedIn = authState.isAuthenticated;
  final isAuthRoute = state.matchedLocation.startsWith('/auth');
  final isOnboarding = state.matchedLocation.startsWith('/onboarding');

  // Public routes that don't require auth
  final isPublicRoute = isAuthRoute || isOnboarding;

  if (!isLoggedIn && !isPublicRoute) {
    // Save the intended destination for post-login redirect
    final from = state.matchedLocation;
    return '/auth/login?from=$from';
  }

  if (isLoggedIn && isAuthRoute) {
    return '/home';
  }

  return null;
},
```

### Post-Login Redirect

After successful login, redirect to the originally intended page:

```dart
// In login screen, after successful login:
final from = GoRouterState.of(context).uri.queryParameters['from'];
if (from != null) {
  context.go(from);
} else {
  context.go('/home');
}
```

---

## Deep Linking

go_router handles deep links automatically when routes are defined. The URL structure maps directly to routes:

```
example-app://app/tasks/abc-123       -> TaskDetailScreen(taskId: 'abc-123')
example-app://app/auth/verify-email?token=xyz -> VerifyEmailScreen(token: 'xyz')
```

Platform configuration (iOS `apple-app-site-association`, Android `assetlinks.json`) is separate from routing code.

---

## Route Naming Convention

| Pattern | Example | When |
|---------|---------|------|
| `/{feature}` | `/tasks` | List screen |
| `/{feature}/new` | `/tasks/new` | Create screen |
| `/{feature}/:id` | `/tasks/:taskId` | Detail screen |
| `/{feature}/:id/edit` | `/tasks/:taskId/edit` | Edit screen |
| `/auth/{action}` | `/auth/login` | Auth flow |
| `/settings` | `/settings` | Settings root |
| `/settings/{section}` | `/settings/notifications` | Settings section |

Route names use kebab-case: `tasks-detail`, `auth-login`, `tasks-new`.

---

## Page Transitions

Custom transitions per route:

```dart
GoRoute(
  path: '/tasks/:taskId',
  name: 'tasks-detail',
  pageBuilder: (context, state) {
    final taskId = state.pathParameters['taskId']!;
    return CustomTransitionPage(
      key: state.pageKey,
      child: TaskDetailScreen(taskId: taskId),
      transitionsBuilder: (context, animation, secondaryAnimation, child) {
        return SlideTransition(
          position: animation.drive(
            Tween(begin: const Offset(1, 0), end: Offset.zero)
                .chain(CurveTween(curve: Curves.easeInOut)),
          ),
          child: child,
        );
      },
    );
  },
),
```

---

## Anti-Patterns

### 1. Navigator.push
```dart
// BAD
Navigator.push(context, MaterialPageRoute(builder: (_) => DetailScreen()));
// GOOD
context.push('/tasks/${task.id}');
```

### 2. Routes scattered across files
```dart
// BAD: routes defined in each feature
// GOOD: all routes in app_router.dart
```

### 3. Hardcoded route strings everywhere
```dart
// BAD: same string in multiple places
context.push('/tasks/${task.id}');
context.push('/tasks/$id');

// GOOD: use named routes or route constants
abstract class AppRoutes {
  static const home = '/home';
  static const tasks = '/tasks';
  static String taskDetail(String id) => '/tasks/$id';
}
context.push(AppRoutes.taskDetail(task.id));
```
