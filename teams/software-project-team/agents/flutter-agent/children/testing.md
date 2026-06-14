---
knowledge-base-summary: "Widget testing with WidgetTester. Provider mocking with ProviderScope overrides. Integration testing. Golden tests for visual regression. Test naming conventions. What to test: user interactions, state changes, navigation. What NOT to test: implementation details."
---
# Testing

## Philosophy

Test what the user sees and does, not how the widget is built internally. Flutter is a bridge -- the real logic is in the API. Testing in Flutter means: "When the user taps X, does Y happen on screen?" and "When the API returns Z, does the UI show it correctly?"

## Test File Structure

Mirror `lib/` under `test/`:

```
lib/
  features/
    home/
      screens/home_screen.dart
      widgets/task_card.dart
    auth/
      screens/login_screen.dart
test/
  features/
    home/
      screens/home_screen_test.dart
      widgets/task_card_test.dart
    auth/
      screens/login_screen_test.dart
  helpers/
    pump_app.dart            # Test wrapper with providers, theme, i18n
    mock_repositories.dart   # Shared mock implementations
integration_test/
  app_test.dart
```

## Test Naming Convention

Format: `should [expected behavior] when [condition]`

```dart
// Good
'should show loading indicator when tasks are being fetched'
'should display error message when API returns failure'
'should navigate to detail screen when task card is tapped'
'should disable submit button when form is invalid'

// Bad
'test1'
'loading test'
'task card works'
```

## What to Test

| Test | Example |
|------|---------|
| User interactions | Tap button -> action happens |
| State changes | Loading -> data displayed, error -> retry button visible |
| Navigation | Tap card -> navigated to detail screen |
| Error states | API failure -> error message with retry |
| Empty states | No data -> empty state widget shown |
| Form validation | Empty email -> "Email required" shown |
| Conditional rendering | Admin user -> edit button visible, regular user -> no edit button |

## What NOT to Test

- Widget implementation details (internal state variables, private methods)
- Third-party package internals (Riverpod, go_router, Dio behavior)
- Pixel-perfect layout (use golden tests for visual regression instead)
- Platform-specific behavior (test on real device with integration tests)
- API business logic (that belongs in API tests)

## Test Wrapper (pumpApp)

Every widget test needs providers, theme, localization, and a router. Create a shared wrapper:

```dart
// test/helpers/pump_app.dart

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:example_app/l10n/app_localizations.dart';

extension PumpApp on WidgetTester {
  Future<void> pumpApp(
    Widget widget, {
    List<Override> overrides = const [],
    GoRouter? router,
  }) async {
    await pumpWidget(
      ProviderScope(
        overrides: overrides,
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('en'),
          home: widget,
        ),
      ),
    );
    await pump(); // Allow one frame for providers to initialize
  }
}
```

## Widget Testing

### Basic Widget Test

```dart
// test/features/home/widgets/task_card_test.dart

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:example_app/features/home/widgets/task_card.dart';
import '../../../helpers/pump_app.dart';

void main() {
  group('TaskCard', () {
    final task = Task(
      id: '1',
      title: 'Morning Task',
      distance: 2.5,
      duration: Duration(minutes: 30),
      date: DateTime(2026, 4, 14),
    );

    testWidgets(
      'should display task title and distance',
      (tester) async {
        await tester.pumpApp(TaskCard(task: task));

        expect(find.text('Morning Task'), findsOneWidget);
        expect(find.text('2.5 km'), findsOneWidget);
      },
    );

    testWidgets(
      'should call onTap when card is tapped',
      (tester) async {
        var tapped = false;

        await tester.pumpApp(
          TaskCard(task: task, onTap: () => tapped = true),
        );

        await tester.tap(find.byType(TaskCard));
        await tester.pump();

        expect(tapped, isTrue);
      },
    );

    testWidgets(
      'should show duration in minutes',
      (tester) async {
        await tester.pumpApp(TaskCard(task: task));

        expect(find.text('30 min'), findsOneWidget);
      },
    );
  });
}
```

### Finding Widgets

```dart
// By type
find.byType(TaskCard)
find.byType(CircularProgressIndicator)
find.byType(ElevatedButton)

// By text
find.text('Morning Task')
find.textContaining('km')

// By key (for widgets without unique text)
find.byKey(const Key('submit_button'))
find.byKey(const ValueKey('task_card_1'))

// By icon
find.byIcon(Icons.delete)

// By widget predicate
find.byWidgetPredicate(
  (widget) => widget is Text && widget.data?.contains('Error') == true,
)

// Descendant: find a Text inside a specific card
find.descendant(
  of: find.byType(TaskCard),
  matching: find.text('Morning Task'),
)
```

### Interaction Testing

```dart
// Tap
await tester.tap(find.byType(ElevatedButton));
await tester.pump(); // Process the tap

// Enter text
await tester.enterText(find.byType(TextField), 'hello@test.com');
await tester.pump();

// Scroll
await tester.drag(find.byType(ListView), const Offset(0, -300));
await tester.pumpAndSettle(); // Wait for scroll animation

// Long press
await tester.longPress(find.byType(TaskCard));
await tester.pump();

// Swipe to dismiss
await tester.drag(find.byType(Dismissible), const Offset(-500, 0));
await tester.pumpAndSettle();

// Pull to refresh
await tester.fling(find.byType(RefreshIndicator), const Offset(0, 300), 1000);
await tester.pumpAndSettle();
```

## Provider Mocking

Override providers in tests to control data flow. This is the primary way to mock API behavior in Flutter tests.

### Mock Repository

```dart
// test/helpers/mock_repositories.dart

class MockTaskRepository implements TaskRepository {
  List<Task> tasksToReturn = [];
  Exception? errorToThrow;
  Task? taskToReturn;

  @override
  Future<List<Task>> getTasks() async {
    if (errorToThrow != null) throw errorToThrow!;
    return tasksToReturn;
  }

  @override
  Future<Task> getTaskById(String id) async {
    if (errorToThrow != null) throw errorToThrow!;
    return taskToReturn ?? tasksToReturn.first;
  }

  @override
  Future<void> deleteTask(String id) async {
    if (errorToThrow != null) throw errorToThrow!;
    tasksToReturn.removeWhere((w) => w.id == id);
  }
}
```

### Testing with Provider Overrides

```dart
// test/features/home/screens/home_screen_test.dart

void main() {
  late MockTaskRepository mockTaskRepo;

  setUp(() {
    mockTaskRepo = MockTaskRepository();
  });

  group('HomeScreen', () {
    testWidgets(
      'should show loading indicator when tasks are being fetched',
      (tester) async {
        // FutureProvider will be in loading state initially
        await tester.pumpApp(
          const HomeScreen(),
          overrides: [
            taskRepositoryProvider.overrideWithValue(mockTaskRepo),
          ],
        );

        // On first frame, FutureProvider is still loading
        expect(find.byType(CircularProgressIndicator), findsOneWidget);
      },
    );

    testWidgets(
      'should display task list when data is loaded',
      (tester) async {
        mockTaskRepo.tasksToReturn = [
          Task(id: '1', title: 'Morning Task', distance: 2.5,
               duration: Duration(minutes: 30), date: DateTime(2026, 4, 14)),
          Task(id: '2', title: 'Evening Task', distance: 3.0,
               duration: Duration(minutes: 45), date: DateTime(2026, 4, 14)),
        ];

        await tester.pumpApp(
          const HomeScreen(),
          overrides: [
            taskRepositoryProvider.overrideWithValue(mockTaskRepo),
          ],
        );

        await tester.pumpAndSettle(); // Wait for future to complete

        expect(find.text('Morning Task'), findsOneWidget);
        expect(find.text('Evening Task'), findsOneWidget);
      },
    );

    testWidgets(
      'should show error message when API fails',
      (tester) async {
        mockTaskRepo.errorToThrow = Exception('Network error');

        await tester.pumpApp(
          const HomeScreen(),
          overrides: [
            taskRepositoryProvider.overrideWithValue(mockTaskRepo),
          ],
        );

        await tester.pumpAndSettle();

        expect(find.byType(ErrorView), findsOneWidget);
        expect(find.text('Something went wrong'), findsOneWidget);
      },
    );

    testWidgets(
      'should show empty state when no tasks exist',
      (tester) async {
        mockTaskRepo.tasksToReturn = [];

        await tester.pumpApp(
          const HomeScreen(),
          overrides: [
            taskRepositoryProvider.overrideWithValue(mockTaskRepo),
          ],
        );

        await tester.pumpAndSettle();

        expect(find.byType(EmptyView), findsOneWidget);
      },
    );

    testWidgets(
      'should refresh tasks when pulled down',
      (tester) async {
        mockTaskRepo.tasksToReturn = [
          Task(id: '1', title: 'Morning Task', distance: 2.5,
               duration: Duration(minutes: 30), date: DateTime(2026, 4, 14)),
        ];

        await tester.pumpApp(
          const HomeScreen(),
          overrides: [
            taskRepositoryProvider.overrideWithValue(mockTaskRepo),
          ],
        );

        await tester.pumpAndSettle();

        // Add a new task to the mock
        mockTaskRepo.tasksToReturn.add(
          Task(id: '3', title: 'New Task', distance: 1.0,
               duration: Duration(minutes: 15), date: DateTime(2026, 4, 14)),
        );

        // Pull to refresh
        await tester.fling(
          find.byType(RefreshIndicator),
          const Offset(0, 300),
          1000,
        );
        await tester.pumpAndSettle();

        expect(find.text('New Task'), findsOneWidget);
      },
    );
  });
}
```

### Overriding AsyncNotifierProvider

For providers that use `AsyncNotifier`:

```dart
testWidgets(
  'should show tasks from overridden notifier',
  (tester) async {
    await tester.pumpApp(
      const HomeScreen(),
      overrides: [
        tasksProvider.overrideWith(() => MockTasksNotifier()),
      ],
    );

    await tester.pumpAndSettle();
    // assertions...
  },
);

class MockTasksNotifier extends TasksNotifier {
  @override
  Future<List<Task>> build() async {
    return [
      Task(id: '1', title: 'Test Task', ...),
    ];
  }
}
```

## Integration Testing

Integration tests run on a real device or emulator. They test the full app flow.

```dart
// integration_test/app_test.dart

import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:example_app/main.dart' as app;

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  group('end-to-end test', () {
    testWidgets(
      'should complete login and see home screen',
      (tester) async {
        app.main();
        await tester.pumpAndSettle();

        // Login screen should be visible
        expect(find.text('Login'), findsOneWidget);

        // Enter credentials
        await tester.enterText(
          find.byKey(const Key('email_field')),
          'test@example.com',
        );
        await tester.enterText(
          find.byKey(const Key('password_field')),
          'password123',
        );

        // Tap login
        await tester.tap(find.byKey(const Key('login_button')));
        await tester.pumpAndSettle();

        // Home screen should be visible
        expect(find.text('Home'), findsOneWidget);
      },
    );
  });
}
```

Run integration tests:

```bash
flutter test integration_test/app_test.dart
```

## Golden Tests

Golden tests capture a screenshot of a widget and compare it against a reference image. Useful for visual regression testing.

```dart
// test/features/home/widgets/task_card_golden_test.dart

void main() {
  testWidgets(
    'TaskCard matches golden file',
    (tester) async {
      await tester.pumpApp(
        Center(
          child: SizedBox(
            width: 350,
            child: TaskCard(
              task: Task(
                id: '1',
                title: 'Morning Task',
                distance: 2.5,
                duration: Duration(minutes: 30),
                date: DateTime(2026, 4, 14),
              ),
            ),
          ),
        ),
      );

      await expectLater(
        find.byType(TaskCard),
        matchesGoldenFile('goldens/task_card.png'),
      );
    },
  );
}
```

Generate golden files:

```bash
flutter test --update-goldens test/features/home/widgets/task_card_golden_test.dart
```

Run golden comparison:

```bash
flutter test test/features/home/widgets/task_card_golden_test.dart
```

**Warning:** Golden tests are sensitive to platform (macOS vs Linux renders fonts differently). Run golden generation and comparison on the same platform (CI should match local).

## Testing Async Operations

```dart
testWidgets(
  'should show success snackbar after deleting task',
  (tester) async {
    mockTaskRepo.tasksToReturn = [
      Task(id: '1', title: 'Morning Task', ...),
    ];

    await tester.pumpApp(
      const HomeScreen(),
      overrides: [
        taskRepositoryProvider.overrideWithValue(mockTaskRepo),
      ],
    );

    await tester.pumpAndSettle();

    // Long press to show delete option
    await tester.longPress(find.text('Morning Task'));
    await tester.pumpAndSettle();

    // Tap delete
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();

    // Confirm in dialog
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();

    // Verify snackbar
    expect(find.text('Task deleted'), findsOneWidget);
  },
);
```

## Running Tests

```bash
# All tests
flutter test

# Specific file
flutter test test/features/home/screens/home_screen_test.dart

# With coverage
flutter test --coverage

# Only tests matching a pattern
flutter test --name "should show loading"

# Verbose output
flutter test --reporter expanded
```
