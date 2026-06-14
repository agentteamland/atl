---
knowledge-base-summary: "Foreground/background state changes via WidgetsBindingObserver. Background → token might expire → check and refresh on resume. Socket reconnection on resume. Data staleness check. Battery/memory considerations for background work. App termination cleanup."
---
# App Lifecycle

## Overview

When the app moves to background, timers should stop, sockets should disconnect, and resources should be released. When the app resumes, tokens may have expired, data may be stale, and connections need re-establishing. `WidgetsBindingObserver` is the mechanism.

## AppLifecycleState Values

| State | Meaning | Triggered When |
|-------|---------|----------------|
| `resumed` | App is visible and responding to user input | User opens app, returns from background, switches back |
| `inactive` | App is visible but not responding to input | Incoming phone call, system dialog overlay, app switcher |
| `paused` | App is not visible | User pressed home, switched to another app |
| `detached` | App is still hosted but detached from any views | Engine is detached (rare, mostly Android) |
| `hidden` | App is hidden (transitional state) | Between inactive and paused on some platforms |

## Lifecycle Transition Order

```
App start:     → resumed
Go background: resumed → inactive → hidden → paused
Come back:     paused → hidden → inactive → resumed
App killed:    paused → detached (not always called)
```

## AppLifecycleManager

A centralized mixin that handles all lifecycle events. Applied to the root app widget or a dedicated lifecycle widget.

```dart
// lib/core/lifecycle/app_lifecycle_manager.dart

import 'dart:async';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

mixin AppLifecycleManager<T extends ConsumerStatefulWidget>
    on ConsumerState<T>, WidgetsBindingObserver {
  Timer? _stalenessTimer;
  DateTime? _pausedAt;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _stalenessTimer?.cancel();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    switch (state) {
      case AppLifecycleState.resumed:
        _onResumed();
        break;
      case AppLifecycleState.paused:
        _onPaused();
        break;
      case AppLifecycleState.inactive:
        _onInactive();
        break;
      case AppLifecycleState.detached:
        _onDetached();
        break;
      case AppLifecycleState.hidden:
        // Transitional state, usually no action needed
        break;
    }
  }

  /// App came to foreground.
  void _onResumed() {
    _checkTokenExpiry();
    _checkDataStaleness();
    _reconnectSocket();
    _restartTimers();
  }

  /// App went to background.
  void _onPaused() {
    _pausedAt = DateTime.now();
    _disconnectSocket();
    _cancelTimers();
    _saveDraftState();
  }

  /// App is visible but not interactive (e.g., phone call overlay).
  void _onInactive() {
    // Pause animations, stop video playback
  }

  /// App is being terminated.
  void _onDetached() {
    // Final cleanup. Not always called -- do not rely on this.
  }

  // --- Token Management ---

  void _checkTokenExpiry() {
    final authNotifier = ref.read(authProvider.notifier);

    // If token has expired while in background, refresh it
    if (authNotifier.isTokenExpired()) {
      authNotifier.refreshToken().catchError((error) {
        // Refresh failed -- force logout
        authNotifier.logout();
      });
    }
  }

  // --- Data Staleness ---

  void _checkDataStaleness() {
    if (_pausedAt == null) return;

    final elapsed = DateTime.now().difference(_pausedAt!);

    // If app was in background for more than 5 minutes, refresh data
    if (elapsed.inMinutes >= 5) {
      ref.invalidate(tasksProvider);
      ref.invalidate(notificationsProvider);
      ref.invalidate(profileProvider);
    }

    _pausedAt = null;
  }

  // --- Socket Management ---

  void _reconnectSocket() {
    final socketService = ref.read(socketServiceProvider);
    if (!socketService.isConnected) {
      socketService.connect();
    }
  }

  void _disconnectSocket() {
    final socketService = ref.read(socketServiceProvider);
    socketService.disconnect();
  }

  // --- Timer Management ---

  void _restartTimers() {
    _stalenessTimer?.cancel();
    _stalenessTimer = Timer.periodic(
      const Duration(minutes: 5),
      (_) {
        // Periodic data freshness check while app is active
        ref.invalidate(notificationsProvider);
      },
    );
  }

  void _cancelTimers() {
    _stalenessTimer?.cancel();
    _stalenessTimer = null;
  }

  // --- Draft State ---

  void _saveDraftState() {
    // Save any in-progress form data to local storage
    // so it survives if the OS kills the app
    final draftService = ref.read(draftServiceProvider);
    draftService.saveCurrentDraft();
  }
}
```

## Usage in Root Widget

Apply the mixin to the root app widget so lifecycle events are handled globally:

```dart
// lib/app.dart

class App extends ConsumerStatefulWidget {
  const App({super.key});

  @override
  ConsumerState<App> createState() => _AppState();
}

class _AppState extends ConsumerState<App>
    with WidgetsBindingObserver, AppLifecycleManager {
  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      routerConfig: ref.watch(routerProvider),
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      themeMode: ref.watch(themeModeProvider),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
    );
  }
}
```

## Screen-Level Lifecycle

Some screens need their own lifecycle handling. For example, a map screen should stop GPS tracking when the screen is not visible:

```dart
// lib/features/task/screens/task_tracking_screen.dart

class TaskTrackingScreen extends ConsumerStatefulWidget {
  const TaskTrackingScreen({super.key});

  @override
  ConsumerState<TaskTrackingScreen> createState() =>
      _TaskTrackingScreenState();
}

class _TaskTrackingScreenState extends ConsumerState<TaskTrackingScreen>
    with WidgetsBindingObserver {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    switch (state) {
      case AppLifecycleState.resumed:
        // Resume GPS tracking, restart location stream
        ref.read(locationTrackingProvider.notifier).resume();
        break;
      case AppLifecycleState.paused:
        // Reduce GPS frequency in background to save battery
        ref.read(locationTrackingProvider.notifier).reduceFrequency();
        break;
      default:
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: const TaskTrackingView(),
    );
  }
}
```

## Battery Considerations

When the app is in background or resuming after a long pause:

```dart
// Reduce polling frequency when battery is low
void _adjustForBattery() {
  // Use shorter intervals when plugged in, longer when on battery
  final refreshInterval = _isCharging
      ? const Duration(minutes: 1)
      : const Duration(minutes: 10);

  _stalenessTimer?.cancel();
  _stalenessTimer = Timer.periodic(refreshInterval, (_) {
    ref.invalidate(notificationsProvider);
  });
}
```

## Data Staleness Strategy

| Background Duration | Action |
|---------------------|--------|
| < 1 minute | No action, data is fresh |
| 1-5 minutes | Refresh notifications only |
| 5-30 minutes | Refresh main data (tasks, profile, notifications) |
| > 30 minutes | Full refresh of all visible data |

```dart
void _checkDataStaleness() {
  if (_pausedAt == null) return;

  final elapsed = DateTime.now().difference(_pausedAt!);

  if (elapsed.inMinutes >= 30) {
    // Full refresh
    ref.invalidate(tasksProvider);
    ref.invalidate(profileProvider);
    ref.invalidate(notificationsProvider);
    ref.invalidate(statsProvider);
  } else if (elapsed.inMinutes >= 5) {
    // Partial refresh
    ref.invalidate(tasksProvider);
    ref.invalidate(notificationsProvider);
  } else if (elapsed.inMinutes >= 1) {
    // Light refresh
    ref.invalidate(notificationsProvider);
  }

  _pausedAt = null;
}
```

## AppLifecycleListener (Alternative API)

Flutter 3.13+ provides `AppLifecycleListener` as an alternative to `WidgetsBindingObserver`. It uses callbacks instead of override methods:

```dart
class _AppState extends ConsumerState<App> {
  late final AppLifecycleListener _lifecycleListener;

  @override
  void initState() {
    super.initState();
    _lifecycleListener = AppLifecycleListener(
      onResume: _onResumed,
      onPause: _onPaused,
      onInactive: _onInactive,
      onDetach: _onDetached,
      onHide: _onHidden,
      onShow: _onShown,
      // Called for any state transition
      onStateChange: _onStateChanged,
    );
  }

  @override
  void dispose() {
    _lifecycleListener.dispose();
    super.dispose();
  }

  void _onResumed() { /* ... */ }
  void _onPaused() { /* ... */ }
  void _onInactive() { /* ... */ }
  void _onDetached() { /* ... */ }
  void _onHidden() { /* ... */ }
  void _onShown() { /* ... */ }
  void _onStateChanged(AppLifecycleState state) { /* ... */ }

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(...);
  }
}
```

Both approaches are valid. `WidgetsBindingObserver` with the mixin pattern is more established and works well with Riverpod's `ConsumerState`. `AppLifecycleListener` is cleaner for simple cases.

## Key Rules

1. **Always cancel timers on pause.** Leaked timers drain battery and cause unexpected behavior.
2. **Always check token on resume.** The token may have expired. Refresh before any API call.
3. **Socket disconnect on pause, reconnect on resume.** Do not keep WebSocket alive in background.
4. **Save draft state on pause.** The OS may kill the app while in background.
5. **Do not rely on detached.** It is not always called. Do critical cleanup in paused.
6. **Stale data thresholds are configurable.** 5 minutes is a starting point; adjust per feature.
