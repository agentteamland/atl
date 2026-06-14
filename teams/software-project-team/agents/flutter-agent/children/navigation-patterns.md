---
knowledge-base-summary: "Bottom navigation bar, drawer, tab bar, modal bottom sheet, dialog. When to use which pattern. Nested navigation (tabs with independent stacks). AppBar actions. Back button handling. Navigation state preservation."
---
# Navigation Patterns

## Overview

Flutter navigation beyond routing. This file covers UI navigation components: bottom bars, drawers, tabs, sheets, dialogs. For route definition and go_router setup, see `routing.md`.

## Decision Table: When to Use Which

| Pattern | Use Case | Example |
|---------|----------|---------|
| Bottom Navigation Bar | Primary app sections (3-5 tabs) | Home, Search, Profile |
| Drawer | Secondary navigation, settings, less-frequent screens | Settings, Help, Account |
| Tab Bar | Sub-sections within a screen | Activity: Daily / Weekly / Monthly |
| Modal Bottom Sheet | Quick actions, filters, pickers | Sort options, share menu |
| Dialog | Confirmations, destructive actions, critical info | "Delete task?" confirmation |

**Rule of thumb:** If the user needs it constantly → bottom bar. If they need it sometimes → drawer. If it is contextual to the current screen → tab or sheet. If it requires a yes/no decision → dialog.

## Bottom Navigation Bar with go_router

Bottom navigation uses `StatefulShellRoute` from go_router to maintain independent navigation stacks per tab. Each tab has its own `Navigator` — navigating within one tab does not reset others.

```dart
// lib/core/router/app_router.dart

final appRouter = GoRouter(
  initialLocation: '/home',
  routes: [
    StatefulShellRoute.indexedStack(
      builder: (context, state, navigationShell) {
        return AppScaffold(navigationShell: navigationShell);
      },
      branches: [
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/home',
              builder: (context, state) => const HomeScreen(),
              routes: [
                GoRoute(
                  path: 'task/:id',
                  builder: (context, state) => TaskDetailScreen(
                    id: state.pathParameters['id']!,
                  ),
                ),
              ],
            ),
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/search',
              builder: (context, state) => const SearchScreen(),
            ),
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/stats',
              builder: (context, state) => const StatsScreen(),
            ),
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/profile',
              builder: (context, state) => const ProfileScreen(),
            ),
          ],
        ),
      ],
    ),
  ],
);
```

### AppScaffold with BottomNavigationBar

```dart
// lib/core/widgets/app_scaffold.dart

class AppScaffold extends StatelessWidget {
  final StatefulNavigationShell navigationShell;

  const AppScaffold({super.key, required this.navigationShell});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: navigationShell,
      bottomNavigationBar: NavigationBar(
        selectedIndex: navigationShell.currentIndex,
        onDestinationSelected: (index) {
          // goBranch navigates to the branch and preserves its state.
          // initialLocation: true resets the branch to its root.
          navigationShell.goBranch(
            index,
            initialLocation: index == navigationShell.currentIndex,
          );
        },
        destinations: [
          NavigationDestination(
            icon: const Icon(Icons.home_outlined),
            selectedIcon: const Icon(Icons.home),
            label: context.l10n.navHome,
          ),
          NavigationDestination(
            icon: const Icon(Icons.search_outlined),
            selectedIcon: const Icon(Icons.search),
            label: context.l10n.navSearch,
          ),
          NavigationDestination(
            icon: const Icon(Icons.bar_chart_outlined),
            selectedIcon: const Icon(Icons.bar_chart),
            label: context.l10n.navStats,
          ),
          NavigationDestination(
            icon: const Icon(Icons.person_outlined),
            selectedIcon: const Icon(Icons.person),
            label: context.l10n.navProfile,
          ),
        ],
      ),
    );
  }
}
```

**Key behavior:** Tapping the current tab again resets it to root (`initialLocation: true`). This is standard iOS/Android behavior.

## Drawer (Side Menu)

Drawer is for secondary navigation: settings, help, about, account management. It is NOT for primary sections that belong in the bottom bar.

```dart
// lib/core/widgets/app_drawer.dart

class AppDrawer extends StatelessWidget {
  const AppDrawer({super.key});

  @override
  Widget build(BuildContext context) {
    return Drawer(
      child: SafeArea(
        child: Column(
          children: [
            // Header with user info
            const _DrawerHeader(),
            const Divider(),

            // Navigation items
            _DrawerItem(
              icon: Icons.settings,
              label: context.l10n.settings,
              onTap: () {
                Navigator.of(context).pop(); // Close drawer first
                context.push('/settings');
              },
            ),
            _DrawerItem(
              icon: Icons.help_outline,
              label: context.l10n.help,
              onTap: () {
                Navigator.of(context).pop();
                context.push('/help');
              },
            ),
            _DrawerItem(
              icon: Icons.info_outline,
              label: context.l10n.about,
              onTap: () {
                Navigator.of(context).pop();
                context.push('/about');
              },
            ),

            const Spacer(),
            const Divider(),

            // Logout at bottom
            _DrawerItem(
              icon: Icons.logout,
              label: context.l10n.logout,
              color: Theme.of(context).colorScheme.error,
              onTap: () {
                Navigator.of(context).pop();
                _showLogoutConfirmation(context);
              },
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }
}

class _DrawerItem extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final Color? color;

  const _DrawerItem({
    required this.icon,
    required this.label,
    required this.onTap,
    this.color,
  });

  @override
  Widget build(BuildContext context) {
    final effectiveColor = color ?? Theme.of(context).colorScheme.onSurface;
    return ListTile(
      leading: Icon(icon, color: effectiveColor),
      title: Text(label, style: TextStyle(color: effectiveColor)),
      onTap: onTap,
    );
  }
}
```

**Usage in AppScaffold:**

```dart
Scaffold(
  drawer: const AppDrawer(),
  appBar: AppBar(
    // Hamburger icon is added automatically when drawer is present
    title: Text(context.l10n.appTitle),
  ),
  body: navigationShell,
  bottomNavigationBar: ...,
);
```

## Tab Bar

Tab bars are for sub-sections within a single screen. They use `DefaultTabController` or a `TabController` for more control.

```dart
// lib/features/stats/screens/stats_screen.dart

class StatsScreen extends StatelessWidget {
  const StatsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 3,
      child: Scaffold(
        appBar: AppBar(
          title: Text(context.l10n.stats),
          bottom: TabBar(
            tabs: [
              Tab(text: context.l10n.daily),
              Tab(text: context.l10n.weekly),
              Tab(text: context.l10n.monthly),
            ],
          ),
        ),
        body: const TabBarView(
          children: [
            DailyStatsTab(),
            WeeklyStatsTab(),
            MonthlyStatsTab(),
          ],
        ),
      ),
    );
  }
}
```

### TabController for Programmatic Control

When you need to change tabs programmatically (e.g., after an action), use a `TabController`:

```dart
class StatsScreen extends ConsumerStatefulWidget {
  const StatsScreen({super.key});

  @override
  ConsumerState<StatsScreen> createState() => _StatsScreenState();
}

class _StatsScreenState extends ConsumerState<StatsScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this);
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  void _jumpToWeekly() {
    _tabController.animateTo(1); // Index 1 = weekly tab
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(context.l10n.stats),
        bottom: TabBar(
          controller: _tabController,
          tabs: [
            Tab(text: context.l10n.daily),
            Tab(text: context.l10n.weekly),
            Tab(text: context.l10n.monthly),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: const [
          DailyStatsTab(),
          WeeklyStatsTab(),
          MonthlyStatsTab(),
        ],
      ),
    );
  }
}
```

## Modal Bottom Sheet

Bottom sheets are for contextual actions: sort, filter, quick settings, share options. They slide up from the bottom and can be dismissed by swiping down.

```dart
// lib/core/widgets/sheets/sort_sheet.dart

class SortSheet extends StatelessWidget {
  final SortOption currentSort;
  final ValueChanged<SortOption> onSortChanged;

  const SortSheet({
    super.key,
    required this.currentSort,
    required this.onSortChanged,
  });

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Drag handle
          Center(
            child: Container(
              margin: const EdgeInsets.only(top: 12, bottom: 8),
              width: 40,
              height: 4,
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.outline,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
          ),

          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Text(
              context.l10n.sortBy,
              style: Theme.of(context).textTheme.titleMedium,
            ),
          ),

          for (final option in SortOption.values)
            RadioListTile<SortOption>(
              title: Text(option.label(context)),
              value: option,
              groupValue: currentSort,
              onChanged: (value) {
                if (value != null) {
                  onSortChanged(value);
                  Navigator.of(context).pop();
                }
              },
            ),

          const SizedBox(height: 16),
        ],
      ),
    );
  }
}
```

### Showing a Bottom Sheet

```dart
// From any screen or widget:
void _showSortSheet(BuildContext context) {
  showModalBottomSheet(
    context: context,
    // Rounded top corners
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
    ),
    // Allow sheet to be taller than half screen
    isScrollControlled: true,
    // Prevent dismissing by tapping outside (optional)
    isDismissible: true,
    builder: (context) => SortSheet(
      currentSort: currentSort,
      onSortChanged: (sort) {
        setState(() => currentSort = sort);
        ref.invalidate(tasksProvider); // Re-fetch with new sort
      },
    ),
  );
}
```

### Scrollable Bottom Sheet (Full Content)

For sheets with long content (filter lists, pickers):

```dart
showModalBottomSheet(
  context: context,
  isScrollControlled: true,
  builder: (context) => DraggableScrollableSheet(
    initialChildSize: 0.6,
    minChildSize: 0.3,
    maxChildSize: 0.9,
    expand: false,
    builder: (context, scrollController) => FilterSheet(
      scrollController: scrollController,
    ),
  ),
);
```

## Dialogs

Dialogs are for confirmations, critical information, and decisions that block the user flow. Use sparingly.

### Confirmation Dialog (Destructive Action)

```dart
// lib/core/widgets/dialogs/confirm_dialog.dart

class ConfirmDialog extends StatelessWidget {
  final String title;
  final String message;
  final String confirmLabel;
  final String cancelLabel;
  final bool isDestructive;

  const ConfirmDialog({
    super.key,
    required this.title,
    required this.message,
    required this.confirmLabel,
    this.cancelLabel = 'Cancel',
    this.isDestructive = false,
  });

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(title),
      content: Text(message),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(false),
          child: Text(cancelLabel),
        ),
        TextButton(
          onPressed: () => Navigator.of(context).pop(true),
          style: isDestructive
              ? TextButton.styleFrom(
                  foregroundColor: Theme.of(context).colorScheme.error,
                )
              : null,
          child: Text(confirmLabel),
        ),
      ],
    );
  }
}
```

### Usage with Result

```dart
Future<void> _deleteTask(BuildContext context, String taskId) async {
  final confirmed = await showDialog<bool>(
    context: context,
    builder: (context) => ConfirmDialog(
      title: context.l10n.deleteTaskTitle,
      message: context.l10n.deleteTaskMessage,
      confirmLabel: context.l10n.delete,
      isDestructive: true,
    ),
  );

  if (confirmed == true && context.mounted) {
    ref.read(deleteTaskProvider(taskId));
  }
}
```

### Custom Dialog (Complex Content)

For dialogs with forms or custom layouts:

```dart
class EditNicknameDialog extends StatefulWidget {
  final String currentNickname;

  const EditNicknameDialog({super.key, required this.currentNickname});

  @override
  State<EditNicknameDialog> createState() => _EditNicknameDialogState();
}

class _EditNicknameDialogState extends State<EditNicknameDialog> {
  late final TextEditingController _controller;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController(text: widget.currentNickname);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(context.l10n.editNickname),
      content: TextField(
        controller: _controller,
        autofocus: true,
        maxLength: 30,
        decoration: InputDecoration(
          hintText: context.l10n.nicknameHint,
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(context.l10n.cancel),
        ),
        FilledButton(
          onPressed: () {
            final value = _controller.text.trim();
            if (value.isNotEmpty) {
              Navigator.of(context).pop(value);
            }
          },
          child: Text(context.l10n.save),
        ),
      ],
    );
  }
}
```

## Back Button Handling

### PopScope (replaces deprecated WillPopScope)

`PopScope` controls what happens when the user presses the system back button or swipes back.

```dart
// Prevent back navigation (e.g., during form submission)
PopScope(
  canPop: !_isSubmitting,
  onPopInvokedWithResult: (didPop, result) {
    if (!didPop) {
      // User tried to go back but canPop is false.
      // Show a warning or ignore.
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(context.l10n.waitForSubmission)),
      );
    }
  },
  child: Scaffold(...),
);
```

### Confirm Before Leaving (Unsaved Changes)

```dart
PopScope(
  canPop: !_hasUnsavedChanges,
  onPopInvokedWithResult: (didPop, result) async {
    if (didPop) return; // Already popped, nothing to do.

    final shouldLeave = await showDialog<bool>(
      context: context,
      builder: (context) => ConfirmDialog(
        title: context.l10n.unsavedChangesTitle,
        message: context.l10n.unsavedChangesMessage,
        confirmLabel: context.l10n.discard,
        isDestructive: true,
      ),
    );

    if (shouldLeave == true && context.mounted) {
      Navigator.of(context).pop();
    }
  },
  child: Scaffold(...),
);
```

## Navigation State Preservation

Bottom navigation already preserves state via `StatefulShellRoute.indexedStack`. Each branch maintains its own navigation stack. When the user switches tabs and returns, the previous tab state is intact.

For preserving scroll position within a tab:

```dart
class TaskListTab extends ConsumerStatefulWidget {
  const TaskListTab({super.key});

  @override
  ConsumerState<TaskListTab> createState() => _TaskListTabState();
}

class _TaskListTabState extends ConsumerState<TaskListTab>
    with AutomaticKeepAliveClientMixin {
  // This prevents the tab content from being disposed when switching tabs
  @override
  bool get wantKeepAlive => true;

  @override
  Widget build(BuildContext context) {
    super.build(context); // Required by AutomaticKeepAliveClientMixin
    return ListView.builder(...);
  }
}
```

## Nested Navigation Pattern Summary

```
AppScaffold (BottomNavigationBar)
├── Home Branch (StatefulShellBranch)
│   ├── HomeScreen
│   └── TaskDetailScreen (pushed within this branch)
├── Search Branch
│   ├── SearchScreen
│   └── SearchResultDetailScreen
├── Stats Branch
│   └── StatsScreen (with TabBar: Daily/Weekly/Monthly)
└── Profile Branch
    ├── ProfileScreen
    └── EditProfileScreen
```

Each branch is an independent navigation stack. Going from `HomeScreen → TaskDetailScreen` does not affect the Search branch. Back button within a branch pops within that branch. Back button at root of a branch exits the app (or goes to previous branch, configurable).
