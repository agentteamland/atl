---
knowledge-base-summary: "Universal links (iOS) and app links (Android) setup. Handling incoming URLs. Email verification links, password reset links, shared content links. go_router integration for URL-to-route mapping. Platform-specific configuration (apple-app-site-association, assetlinks.json)."
---
# Deep Linking

## Overview

Deep linking allows external URLs to open the app directly at a specific screen. Email verification links, password reset links, shared content, invite links -- all route to the correct screen. go_router already defines URL patterns; deep linking maps incoming platform URLs to those routes.

## Architecture

```
External URL (https://example-app.com/verify-email?token=abc)
    ↓
Platform (iOS: Universal Link / Android: App Link)
    ↓
Flutter app receives URL
    ↓
go_router matches route pattern
    ↓
Correct screen opens with parameters
```

## Use Cases

| URL Pattern | Screen | Purpose |
|-------------|--------|---------|
| `/verify-email?token=xxx` | EmailVerificationScreen | Confirm email after registration |
| `/reset-password?token=xxx` | ResetPasswordScreen | Set new password from email link |
| `/task/:id` | TaskDetailScreen | Shared task link |
| `/invite/:code` | InviteScreen | Accept group invite |

## iOS Setup: Universal Links

### 1. apple-app-site-association (Server Side)

Host this JSON file at `https://example-app.com/.well-known/apple-app-site-association` (no file extension, `Content-Type: application/json`):

```json
{
  "applinks": {
    "apps": [],
    "details": [
      {
        "appID": "TEAM_ID.com.example_app.app",
        "paths": [
          "/verify-email",
          "/reset-password",
          "/task/*",
          "/invite/*"
        ]
      }
    ]
  }
}
```

- `TEAM_ID` is the Apple Developer Team ID.
- Must be served over HTTPS with a valid certificate.
- No redirects allowed on this URL.

### 2. Runner.entitlements (iOS Side)

Add the Associated Domains entitlement in Xcode:

```
ios/Runner/Runner.entitlements
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
 "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.developer.associated-domains</key>
    <array>
        <string>applinks:example-app.com</string>
    </array>
</dict>
</plist>
```

Also enable "Associated Domains" capability in Xcode project settings under Signing & Capabilities.

### 3. Info.plist (URL Scheme Fallback)

For older iOS versions or custom scheme links:

```xml
<!-- ios/Runner/Info.plist -->
<key>CFBundleURLTypes</key>
<array>
    <dict>
        <key>CFBundleURLSchemes</key>
        <array>
            <string>example-app</string>
        </array>
    </dict>
</array>
```

## Android Setup: App Links

### 1. assetlinks.json (Server Side)

Host at `https://example-app.com/.well-known/assetlinks.json`:

```json
[
  {
    "relation": ["delegate_permission/common.handle_all_urls"],
    "target": {
      "namespace": "android_app",
      "package_name": "com.example_app.app",
      "sha256_cert_fingerprints": [
        "AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99"
      ]
    }
  }
]
```

Get SHA-256 fingerprint:

```bash
# Debug key
keytool -list -v -keystore ~/.android/debug.keystore -alias androiddebugkey -storepass android

# Release key
keytool -list -v -keystore your-release-key.keystore -alias your-alias
```

### 2. AndroidManifest.xml (Android Side)

```xml
<!-- android/app/src/main/AndroidManifest.xml -->
<activity
    android:name=".MainActivity"
    android:exported="true"
    android:launchMode="singleTask">

    <!-- Deep link intent filter -->
    <intent-filter android:autoVerify="true">
        <action android:name="android.intent.action.VIEW" />
        <category android:name="android.intent.category.DEFAULT" />
        <category android:name="android.intent.category.BROWSABLE" />
        <data
            android:scheme="https"
            android:host="example-app.com"
            android:pathPrefix="/verify-email" />
        <data
            android:scheme="https"
            android:host="example-app.com"
            android:pathPrefix="/reset-password" />
        <data
            android:scheme="https"
            android:host="example-app.com"
            android:pathPrefix="/task" />
        <data
            android:scheme="https"
            android:host="example-app.com"
            android:pathPrefix="/invite" />
    </intent-filter>

    <!-- Custom scheme fallback -->
    <intent-filter>
        <action android:name="android.intent.action.VIEW" />
        <category android:name="android.intent.category.DEFAULT" />
        <category android:name="android.intent.category.BROWSABLE" />
        <data android:scheme="example-app" />
    </intent-filter>
</activity>
```

**Important:** `android:autoVerify="true"` tells Android to verify the `assetlinks.json` file at install time.

## go_router Integration

go_router supports deep linking out of the box. The routes already define URL patterns. The key configuration is telling `GoRouter` to handle incoming URLs:

```dart
// lib/core/router/app_router.dart

final appRouter = GoRouter(
  initialLocation: '/home',

  // Deep link routes
  routes: [
    // Auth routes (no shell, no bottom bar)
    GoRoute(
      path: '/verify-email',
      builder: (context, state) => EmailVerificationScreen(
        token: state.uri.queryParameters['token'] ?? '',
      ),
    ),
    GoRoute(
      path: '/reset-password',
      builder: (context, state) => ResetPasswordScreen(
        token: state.uri.queryParameters['token'] ?? '',
      ),
    ),
    GoRoute(
      path: '/invite/:code',
      builder: (context, state) => InviteScreen(
        code: state.pathParameters['code']!,
      ),
    ),

    // Main app shell with bottom nav
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
        // ...other branches
      ],
    ),
  ],

  // Auth guard
  redirect: (context, state) {
    final isLoggedIn = /* check auth state */;
    final isAuthRoute = state.matchedLocation.startsWith('/login') ||
        state.matchedLocation.startsWith('/register') ||
        state.matchedLocation.startsWith('/verify-email') ||
        state.matchedLocation.startsWith('/reset-password');

    if (!isLoggedIn && !isAuthRoute) return '/login';
    if (isLoggedIn && state.matchedLocation == '/login') return '/home';
    return null; // No redirect
  },
);
```

## Handling Incoming URLs

### Email Verification Flow

When user taps verification link in email:

```dart
// lib/features/auth/screens/email_verification_screen.dart

class EmailVerificationScreen extends ConsumerStatefulWidget {
  final String token;

  const EmailVerificationScreen({super.key, required this.token});

  @override
  ConsumerState<EmailVerificationScreen> createState() =>
      _EmailVerificationScreenState();
}

class _EmailVerificationScreenState
    extends ConsumerState<EmailVerificationScreen> {
  @override
  void initState() {
    super.initState();
    // Verify token on screen open
    if (widget.token.isNotEmpty) {
      _verifyEmail();
    }
  }

  Future<void> _verifyEmail() async {
    try {
      await ref.read(authRepositoryProvider).verifyEmail(widget.token);

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(context.l10n.emailVerified)),
        );
        context.go('/home');
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(context.l10n.verificationFailed)),
        );
        context.go('/login');
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (widget.token.isEmpty) {
      return Scaffold(
        body: Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.error_outline, size: 64),
              const SizedBox(height: 16),
              Text(context.l10n.invalidVerificationLink),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: () => context.go('/login'),
                child: Text(context.l10n.goToLogin),
              ),
            ],
          ),
        ),
      );
    }

    return const Scaffold(
      body: Center(child: CircularProgressIndicator()),
    );
  }
}
```

### Password Reset Flow

```dart
// lib/features/auth/screens/reset_password_screen.dart

class ResetPasswordScreen extends StatelessWidget {
  final String token;

  const ResetPasswordScreen({super.key, required this.token});

  @override
  Widget build(BuildContext context) {
    if (token.isEmpty) {
      return Scaffold(
        body: Center(
          child: Text(context.l10n.invalidResetLink),
        ),
      );
    }

    return Scaffold(
      appBar: AppBar(title: Text(context.l10n.resetPassword)),
      body: ResetPasswordForm(token: token),
    );
  }
}
```

## App State Handling

When the app opens via deep link, it might be in one of three states:

```dart
// 1. App not running (cold start)
//    → Flutter starts, go_router receives initial URL, routes directly.

// 2. App in background (warm start)
//    → App resumes, go_router receives URL via platform channel, navigates.

// 3. App in foreground
//    → URL received while app is active, go_router navigates.
```

go_router handles all three cases automatically. No additional setup needed for basic deep linking.

## Testing Deep Links

### Android Testing

```bash
# Test deep link (app must be installed)
adb shell am start -a android.intent.action.VIEW \
  -d "https://example-app.com/verify-email?token=test123" \
  com.example_app.app

# Test custom scheme
adb shell am start -a android.intent.action.VIEW \
  -d "example-app://task/abc-123" \
  com.example_app.app

# Verify app links
adb shell pm get-app-links com.example_app.app
```

### iOS Testing

```bash
# Test universal link (on simulator)
xcrun simctl openurl booted "https://example-app.com/verify-email?token=test123"

# Test custom scheme
xcrun simctl openurl booted "example-app://task/abc-123"
```

### Debug Logging

Add logging to verify deep link reception during development:

```dart
final appRouter = GoRouter(
  debugLogDiagnostics: true, // Logs all routing decisions
  initialLocation: '/home',
  routes: [...],
);
```

## Checklist for New Deep Link

1. Define the route in go_router (`GoRoute` with path and parameters)
2. Add path to `apple-app-site-association` on server
3. Add path to `assetlinks.json` on server
4. Add `<data>` entry in `AndroidManifest.xml`
5. If route needs auth, add it to the redirect exclusion list
6. Create the target screen that reads parameters from `state`
7. Handle empty/invalid parameters gracefully (show error, redirect)
8. Test with `adb shell` (Android) and `xcrun simctl openurl` (iOS)
