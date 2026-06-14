---
knowledge-base-summary: "Camera, photo gallery, location, notifications, microphone. Permission request flow: check → request → handle (granted/denied/permanently denied). Platform differences (iOS one-time dialog vs Android). Graceful degradation when denied. permission_handler package."
---
# Permissions

## Overview

Platform permissions gate access to device features: camera, location, photos, notifications. The `permission_handler` package provides a unified API for both iOS and Android. The request flow is always: check status, explain why, request, handle result.

## Dependency

```yaml
# pubspec.yaml
dependencies:
  permission_handler: ^11.0.0
```

### iOS Configuration (Info.plist)

Every permission requires a usage description string in `ios/Runner/Info.plist`. If the string is missing, the app crashes when requesting that permission:

```xml
<!-- ios/Runner/Info.plist -->
<key>NSCameraUsageDescription</key>
<string>We need camera access to take photos of your work activities.</string>

<key>NSPhotoLibraryUsageDescription</key>
<string>We need photo access to let you choose a profile picture.</string>

<key>NSLocationWhenInUseUsageDescription</key>
<string>We need your location to track your work activities.</string>

<key>NSLocationAlwaysAndWhenInUseUsageDescription</key>
<string>We need background location to track your task even when the app is minimized.</string>

<key>NSMicrophoneUsageDescription</key>
<string>We need microphone access for voice notes during tasks.</string>

<key>NSContactsUsageDescription</key>
<string>We need contacts access to invite friends to task with you.</string>
```

### Android Configuration (AndroidManifest.xml)

```xml
<!-- android/app/src/main/AndroidManifest.xml -->
<uses-permission android:name="android.permission.CAMERA" />
<uses-permission android:name="android.permission.READ_MEDIA_IMAGES" />
<uses-permission android:name="android.permission.ACCESS_FINE_LOCATION" />
<uses-permission android:name="android.permission.ACCESS_COARSE_LOCATION" />
<uses-permission android:name="android.permission.ACCESS_BACKGROUND_LOCATION" />
<uses-permission android:name="android.permission.RECORD_AUDIO" />
<uses-permission android:name="android.permission.READ_CONTACTS" />
<uses-permission android:name="android.permission.POST_NOTIFICATIONS" />
```

## Permission Status Values

```dart
// PermissionStatus enum
PermissionStatus.granted          // User said yes
PermissionStatus.denied           // User said no (can ask again on Android)
PermissionStatus.permanentlyDenied // User said "Don't ask again" (Android) or denied on iOS
PermissionStatus.restricted       // iOS only: parental controls or MDM restriction
PermissionStatus.limited          // iOS only: limited photo access (selected photos only)
PermissionStatus.provisional      // iOS only: provisional notification permission
```

## Platform Differences

| Behavior | iOS | Android |
|----------|-----|---------|
| First request | System dialog appears | System dialog appears |
| After denial | Cannot show system dialog again. Must go to Settings. | Can show system dialog again (unless "Don't ask again") |
| "Don't ask again" | Implicit after first denial | Explicit checkbox in dialog |
| Permanent denial recovery | `openAppSettings()` | `openAppSettings()` |
| Location "While Using" vs "Always" | Two-step: first "While Using", later can upgrade to "Always" | Similar two-step since Android 11 |

## Permission Request Flow

### Standard Flow

```
Check status
    ↓
granted? → Use the feature
    ↓ (no)
Show explanation screen ("We need camera to...")
    ↓
User taps "Allow"
    ↓
Request permission (system dialog)
    ↓
granted? → Use the feature
denied? → Show graceful degradation
permanentlyDenied? → Show "Go to Settings" button
```

### PermissionService

```dart
// lib/core/services/permission_service.dart

import 'package:permission_handler/permission_handler.dart';

class PermissionService {
  /// Request a single permission with full flow handling.
  /// Returns true if granted, false otherwise.
  Future<bool> request(Permission permission) async {
    final status = await permission.status;

    if (status.isGranted) return true;
    if (status.isRestricted) return false; // iOS: cannot change this

    if (status.isPermanentlyDenied) {
      // User previously denied with "Don't ask again".
      // Only option is to open app settings.
      return false;
    }

    // Request the permission
    final result = await permission.request();
    return result.isGranted;
  }

  /// Check if a permission is permanently denied.
  /// Used to decide between showing request dialog vs settings redirect.
  Future<bool> isPermanentlyDenied(Permission permission) async {
    final status = await permission.status;
    return status.isPermanentlyDenied;
  }

  /// Check current status without requesting.
  Future<PermissionStatus> checkStatus(Permission permission) async {
    return permission.status;
  }

  /// Open device settings for this app.
  Future<bool> openSettings() async {
    return openAppSettings();
  }
}
```

### Provider

```dart
// lib/core/providers/permission_provider.dart

final permissionServiceProvider = Provider<PermissionService>((ref) {
  return PermissionService();
});
```

## Pre-Permission Explanation Screen

Show a custom screen BEFORE the system dialog to explain why the permission is needed. This increases grant rates significantly because users understand the context.

```dart
// lib/core/widgets/permission_request_view.dart

class PermissionRequestView extends StatelessWidget {
  final IconData icon;
  final String title;
  final String description;
  final String allowLabel;
  final String denyLabel;
  final VoidCallback onAllow;
  final VoidCallback onDeny;

  const PermissionRequestView({
    super.key,
    required this.icon,
    required this.title,
    required this.description,
    required this.allowLabel,
    required this.denyLabel,
    required this.onAllow,
    required this.onDeny,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            icon,
            size: 80,
            color: Theme.of(context).colorScheme.primary,
          ),
          const SizedBox(height: 24),
          Text(
            title,
            style: Theme.of(context).textTheme.headlineSmall,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 12),
          Text(
            description,
            style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 40),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: onAllow,
              child: Text(allowLabel),
            ),
          ),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: TextButton(
              onPressed: onDeny,
              child: Text(denyLabel),
            ),
          ),
        ],
      ),
    );
  }
}
```

## Complete Camera Permission Flow

End-to-end example: the user taps "Take Photo" and we handle every possible state.

```dart
// lib/features/profile/widgets/photo_picker.dart

class PhotoPicker extends ConsumerWidget {
  const PhotoPicker({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return IconButton(
      icon: const Icon(Icons.camera_alt),
      onPressed: () => _handleCameraPermission(context, ref),
    );
  }

  Future<void> _handleCameraPermission(
    BuildContext context,
    WidgetRef ref,
  ) async {
    final permissionService = ref.read(permissionServiceProvider);
    final status = await permissionService.checkStatus(Permission.camera);

    if (!context.mounted) return;

    switch (status) {
      case PermissionStatus.granted:
        // Already granted, open camera directly
        _openCamera(context);
        break;

      case PermissionStatus.denied:
        // Show explanation screen, then request
        final shouldRequest = await _showExplanation(context);
        if (shouldRequest && context.mounted) {
          final granted = await permissionService.request(Permission.camera);
          if (granted && context.mounted) {
            _openCamera(context);
          }
        }
        break;

      case PermissionStatus.permanentlyDenied:
        // Cannot request again, must go to settings
        if (context.mounted) {
          _showSettingsDialog(context, permissionService);
        }
        break;

      case PermissionStatus.restricted:
        // iOS: restricted by parental controls, cannot change
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(context.l10n.cameraRestricted)),
          );
        }
        break;

      default:
        // First time (undetermined) -- show explanation then request
        final shouldRequest = await _showExplanation(context);
        if (shouldRequest && context.mounted) {
          final granted = await permissionService.request(Permission.camera);
          if (granted && context.mounted) {
            _openCamera(context);
          }
        }
        break;
    }
  }

  Future<bool> _showExplanation(BuildContext context) async {
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (context) => PermissionRequestView(
        icon: Icons.camera_alt,
        title: context.l10n.cameraPermissionTitle,
        description: context.l10n.cameraPermissionDescription,
        allowLabel: context.l10n.allow,
        denyLabel: context.l10n.notNow,
        onAllow: () => Navigator.of(context).pop(true),
        onDeny: () => Navigator.of(context).pop(false),
      ),
    );

    return result ?? false;
  }

  void _showSettingsDialog(
    BuildContext context,
    PermissionService permissionService,
  ) {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(context.l10n.cameraAccessRequired),
        content: Text(context.l10n.cameraSettingsMessage),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: Text(context.l10n.cancel),
          ),
          FilledButton(
            onPressed: () {
              Navigator.of(context).pop();
              permissionService.openSettings();
            },
            child: Text(context.l10n.openSettings),
          ),
        ],
      ),
    );
  }

  void _openCamera(BuildContext context) {
    // Open camera or image picker
  }
}
```

## Location Permission (Two-Step)

Location has a special two-step flow: first request "While Using", then optionally upgrade to "Always" for background tracking.

```dart
Future<bool> requestLocationPermission({
  bool background = false,
}) async {
  // Step 1: Request "While Using" permission
  var status = await Permission.locationWhenInUse.status;

  if (!status.isGranted) {
    status = await Permission.locationWhenInUse.request();
    if (!status.isGranted) return false;
  }

  // Step 2: If background is needed, request "Always"
  if (background) {
    var bgStatus = await Permission.locationAlways.status;
    if (!bgStatus.isGranted) {
      bgStatus = await Permission.locationAlways.request();
      return bgStatus.isGranted;
    }
  }

  return true;
}
```

## Notification Permission

Android 13+ and iOS require explicit notification permission:

```dart
Future<bool> requestNotificationPermission() async {
  final status = await Permission.notification.request();
  return status.isGranted || status.isProvisional;
}
```

**iOS provisional notifications:** Users can receive notifications silently in the notification center without a dialog. Request with:

```dart
// iOS only: request provisional permission (no dialog)
// Notifications appear silently in notification center
final status = await Permission.notification.request();
// On iOS, this may return PermissionStatus.provisional
```

## Graceful Degradation

When a permission is denied, the feature should still work with reduced functionality:

```dart
// Example: task tracking without location
class TaskTrackingScreen extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final locationGranted = ref.watch(locationPermissionProvider);

    return Scaffold(
      body: locationGranted.when(
        data: (granted) {
          if (granted) {
            return const MapTrackingView(); // Full GPS tracking
          }
          return const ManualTrackingView(); // Manual step entry
        },
        loading: () => const LoadingView(),
        error: (_, __) => const ManualTrackingView(),
      ),
    );
  }
}
```

## Permission Provider for Reactive UI

```dart
// lib/core/providers/permission_providers.dart

final cameraPermissionProvider = FutureProvider<PermissionStatus>((ref) {
  return Permission.camera.status;
});

final locationPermissionProvider = FutureProvider<bool>((ref) async {
  final status = await Permission.locationWhenInUse.status;
  return status.isGranted;
});
```

## Checklist for Adding a New Permission

1. Add permission constant usage (`Permission.camera`, etc.)
2. Add usage description string in `ios/Runner/Info.plist`
3. Add `<uses-permission>` in `android/app/src/main/AndroidManifest.xml`
4. Add `permission_handler` setup in `ios/Podfile` (uncomment relevant permission)
5. Create pre-permission explanation UI with clear reason
6. Handle all states: granted, denied, permanentlyDenied, restricted
7. Implement graceful degradation for denied state
8. Add l10n strings for all permission-related messages
