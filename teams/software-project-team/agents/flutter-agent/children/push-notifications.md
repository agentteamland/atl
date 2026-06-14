---
knowledge-base-summary: "Firebase Cloud Messaging (FCM) for Android, APNs for iOS. Token registration with API. Notification handling: foreground, background, terminated. Local notifications for scheduled alerts. Permission request flow. Deep linking from notification tap."
---
# Push Notifications

## Philosophy

Flutter is a BRIDGE. Notifications originate from the API. The Flutter app registers its FCM token with the API, receives notifications, and navigates the user to the relevant screen. No notification logic lives in Flutter -- the API decides who gets notified, when, and with what content.

## Dependencies

```yaml
dependencies:
  firebase_core: ^3.6.0
  firebase_messaging: ^15.1.0
  flutter_local_notifications: ^18.0.0
```

## Firebase Project Setup

### Android

1. Add `google-services.json` to `android/app/`.
2. In `android/build.gradle`, add Google services plugin.
3. In `android/app/build.gradle`, apply the plugin.

### iOS

1. Add `GoogleService-Info.plist` to iOS project via Xcode.
2. Enable Push Notifications capability in Xcode.
3. Upload APNs authentication key to Firebase Console.
4. Enable Background Modes: Remote notifications.

## Initialization

```dart
Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await Firebase.initializeApp();

  // Request permission early (iOS shows a prompt)
  await NotificationService.instance.init();

  runApp(const ProviderScope(child: App()));
}
```

## Notification Service

Central service that handles all notification lifecycle: permission, token, foreground, background, terminated.

```dart
class NotificationService {
  NotificationService._();
  static final instance = NotificationService._();

  final FirebaseMessaging _fcm = FirebaseMessaging.instance;
  final FlutterLocalNotificationsPlugin _local =
      FlutterLocalNotificationsPlugin();

  String? _fcmToken;
  String? get fcmToken => _fcmToken;

  /// Called once at app start.
  Future<void> init() async {
    // 1. Request permission
    await _requestPermission();

    // 2. Initialize local notifications (for foreground display)
    await _initLocalNotifications();

    // 3. Get FCM token
    _fcmToken = await _fcm.getToken();

    // 4. Listen for token refresh
    _fcm.onTokenRefresh.listen(_onTokenRefresh);

    // 5. Handle foreground messages
    FirebaseMessaging.onMessage.listen(_onForegroundMessage);

    // 6. Handle background tap (app was in background)
    FirebaseMessaging.onMessageOpenedApp.listen(_onNotificationTap);

    // 7. Check if app was opened from a terminated state via notification
    final initialMessage = await _fcm.getInitialMessage();
    if (initialMessage != null) {
      _onNotificationTap(initialMessage);
    }
  }

  // ──────────────────────────────────────────────
  // Permission
  // ──────────────────────────────────────────────

  Future<void> _requestPermission() async {
    final settings = await _fcm.requestPermission(
      alert: true,
      badge: true,
      sound: true,
      provisional: false, // Set true for quiet notifications on iOS
    );

    if (settings.authorizationStatus == AuthorizationStatus.denied) {
      // User denied -- app should work without notifications.
      // Do NOT re-ask immediately. Show in settings screen later.
    }
  }

  // ──────────────────────────────────────────────
  // Token Registration
  // ──────────────────────────────────────────────

  /// Send FCM token to API so backend can target this device.
  Future<void> registerToken(ApiClient api) async {
    if (_fcmToken == null) return;

    await api.post('/notifications/register', data: {
      'token': _fcmToken,
      'platform': Platform.isIOS ? 'ios' : 'android',
    });
  }

  void _onTokenRefresh(String newToken) {
    _fcmToken = newToken;
    // Re-register with API -- caller should provide a way to do this.
    // Typically done via a provider that watches auth state.
  }

  // ──────────────────────────────────────────────
  // Local Notifications Setup
  // ──────────────────────────────────────────────

  Future<void> _initLocalNotifications() async {
    const androidSettings = AndroidInitializationSettings(
      '@mipmap/ic_notification', // Custom notification icon
    );

    const iosSettings = DarwinInitializationSettings(
      requestAlertPermission: false, // Already handled by FCM
      requestBadgePermission: false,
      requestSoundPermission: false,
    );

    const initSettings = InitializationSettings(
      android: androidSettings,
      iOS: iosSettings,
    );

    await _local.initialize(
      initSettings,
      onDidReceiveNotificationResponse: _onLocalNotificationTap,
    );

    // Create Android notification channel
    const channel = AndroidNotificationChannel(
      'default_channel',
      'Default',
      description: 'Default notification channel',
      importance: Importance.high,
    );

    await _local
        .resolvePlatformSpecificImplementation<
            AndroidFlutterLocalNotificationsPlugin>()
        ?.createNotificationChannel(channel);
  }

  // ──────────────────────────────────────────────
  // Foreground Message (app is open)
  // ──────────────────────────────────────────────

  /// When app is in foreground, FCM does NOT show a system notification.
  /// We show it manually via flutter_local_notifications.
  void _onForegroundMessage(RemoteMessage message) {
    final notification = message.notification;
    if (notification == null) return;

    _local.show(
      message.hashCode,
      notification.title,
      notification.body,
      NotificationDetails(
        android: const AndroidNotificationDetails(
          'default_channel',
          'Default',
          importance: Importance.high,
          priority: Priority.high,
          icon: '@mipmap/ic_notification',
        ),
        iOS: const DarwinNotificationDetails(
          presentAlert: true,
          presentBadge: true,
          presentSound: true,
        ),
      ),
      payload: jsonEncode(message.data),
    );
  }

  // ──────────────────────────────────────────────
  // Notification Tap Handlers
  // ──────────────────────────────────────────────

  /// User tapped a notification (from background or terminated state).
  /// Navigate to the relevant screen based on notification data.
  void _onNotificationTap(RemoteMessage message) {
    _handleNavigation(message.data);
  }

  /// User tapped a local notification (foreground).
  void _onLocalNotificationTap(NotificationResponse response) {
    if (response.payload == null) return;
    final data = jsonDecode(response.payload!) as Map<String, dynamic>;
    _handleNavigation(data);
  }

  /// Central navigation handler. Routes to correct screen based on data.
  void _handleNavigation(Map<String, dynamic> data) {
    final type = data['type'] as String?;
    final id = data['id'] as String?;

    if (type == null) return;

    // Use the global navigator key or go_router to navigate.
    // This is called from outside the widget tree, so we need a global ref.
    switch (type) {
      case 'task_completed':
        _navigateTo('/tasks/$id');
      case 'achievement':
        _navigateTo('/achievements/$id');
      case 'friend_request':
        _navigateTo('/friends/requests');
      default:
        _navigateTo('/notifications');
    }
  }

  /// Navigate using go_router's global key.
  void _navigateTo(String path) {
    // This requires a global GoRouter reference.
    // Typically stored as a static or provided via GetIt.
    AppRouter.router.go(path);
  }
}
```

## Three Notification States

### 1. Foreground (App is open)

FCM delivers the message as a data payload. No system notification appears by default. We show it manually using `flutter_local_notifications` as an in-app banner or a local notification.

### 2. Background (App is in background)

FCM shows the system notification automatically (using the `notification` payload). When user taps it, `onMessageOpenedApp` fires.

### 3. Terminated (App is killed)

FCM shows the system notification. When user taps it and the app cold-starts, `getInitialMessage()` returns the message.

```
Foreground:  FCM.onMessage → show via local notification → user taps → navigate
Background:  System tray auto-shows → user taps → onMessageOpenedApp → navigate
Terminated:  System tray auto-shows → user taps → app starts → getInitialMessage → navigate
```

## Token Registration with Riverpod

Register the FCM token with the API after the user logs in.

```dart
final fcmTokenRegistrationProvider = FutureProvider.autoDispose((ref) async {
  final authState = ref.watch(authProvider);

  // Only register when logged in
  if (authState is! AuthAuthenticated) return;

  final api = ref.read(apiClientProvider);
  await NotificationService.instance.registerToken(api);
});
```

In the main app widget:

```dart
class App extends ConsumerWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Trigger token registration when auth changes
    ref.watch(fcmTokenRegistrationProvider);

    return MaterialApp.router(
      routerConfig: AppRouter.router,
      // ...
    );
  }
}
```

## Local Notifications (Scheduled)

For reminders or alerts that don't come from the server.

```dart
extension LocalNotifications on NotificationService {
  /// Schedule a task reminder.
  Future<void> scheduleTaskReminder({
    required int id,
    required String title,
    required String body,
    required DateTime scheduledTime,
  }) async {
    await _local.zonedSchedule(
      id,
      title,
      body,
      tz.TZDateTime.from(scheduledTime, tz.local),
      const NotificationDetails(
        android: AndroidNotificationDetails(
          'reminders_channel',
          'Reminders',
          importance: Importance.high,
        ),
        iOS: DarwinNotificationDetails(),
      ),
      androidScheduleMode: AndroidScheduleMode.exactAllowWhileIdle,
      matchDateTimeComponents: null, // One-time, not recurring
    );
  }

  /// Cancel a scheduled reminder.
  Future<void> cancelReminder(int id) async {
    await _local.cancel(id);
  }

  /// Cancel all scheduled reminders.
  Future<void> cancelAllReminders() async {
    await _local.cancelAll();
  }
}
```

## iOS Permission Request Best Practice

iOS only shows the permission dialog ONCE. If the user denies, the only way to re-enable is through Settings. Strategy:

1. **Do NOT request on first launch.** Show a pre-permission screen explaining the value.
2. **Request after a meaningful action** (e.g., after completing first task).
3. **If denied, show a "Enable in Settings" button** in the notification settings screen.

```dart
class NotificationPermissionScreen extends StatelessWidget {
  const NotificationPermissionScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.notifications_active, size: 80),
              const SizedBox(height: 24),
              Text(
                'Stay on track',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 12),
              Text(
                'Get reminders for your daily tasks and celebrate achievements.',
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.bodyLarge,
              ),
              const SizedBox(height: 32),
              AppButton(
                label: 'Enable Notifications',
                onPressed: () async {
                  await NotificationService.instance.requestPermission();
                  if (context.mounted) context.go('/home');
                },
                fullWidth: true,
              ),
              const SizedBox(height: 12),
              AppButton(
                label: 'Not now',
                onPressed: () => context.go('/home'),
                variant: AppButtonVariant.text,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
```

## Notification Data Payload Structure

The API sends notifications with a `data` payload for deep linking:

```json
{
  "notification": {
    "title": "Task completed!",
    "body": "you completed 5.2 km today. Great job!"
  },
  "data": {
    "type": "task_completed",
    "id": "abc-123"
  }
}
```

- `notification` -- displayed to user (title + body).
- `data` -- used by the app for navigation. NOT displayed.

## Background Message Handler

For data-only messages that need processing even when the app is terminated. Must be a top-level function (not a class method).

```dart
// This MUST be a top-level function.
@pragma('vm:entry-point')
Future<void> _firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  await Firebase.initializeApp();
  // Process silently. Do NOT update UI here -- there is no UI.
  // Example: update local cache, increment badge count.
}

// Register in main():
Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await Firebase.initializeApp();
  FirebaseMessaging.onBackgroundMessage(_firebaseMessagingBackgroundHandler);
  // ...
}
```

## Checklist

- [ ] Firebase project created with Android + iOS apps
- [ ] google-services.json in android/app/
- [ ] GoogleService-Info.plist added via Xcode
- [ ] APNs key uploaded to Firebase Console
- [ ] iOS Push Notifications capability enabled
- [ ] iOS Background Modes: Remote notifications enabled
- [ ] Notification channel created for Android
- [ ] Custom notification icon at android/app/src/main/res/mipmap/ic_notification
- [ ] FCM token sent to API after login
- [ ] Token refresh handler re-registers with API
- [ ] Foreground messages shown via local notification
- [ ] Background tap navigates to correct screen
- [ ] Terminated cold-start navigates to correct screen
- [ ] Permission requested at appropriate time (not on first launch)
