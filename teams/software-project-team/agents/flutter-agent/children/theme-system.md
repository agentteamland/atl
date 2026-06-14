---
knowledge-base-summary: "Material 3, ColorScheme.fromSeed, light/dark mode. AppTheme factory, AppColors palette, AppTypography with Google Fonts. ThemeMode switching (system/light/dark). Persisting theme preference. Never hardcode colors — always use Theme.of(context)."
---
# Theme System (Material 3)

Material 3 with `ColorScheme.fromSeed`. Light and dark themes. Never hardcode colors, sizes, or text styles in widgets. Everything comes from `Theme.of(context)`.

---

## AppTheme Factory

Central theme definition. Two factories: `light()` and `dark()`.

```dart
// lib/core/theme/app_theme.dart
import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:example_app/core/theme/app_colors.dart';

class AppTheme {
  AppTheme._();

  static ThemeData light() {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: AppColors.brandPrimary,
      brightness: Brightness.light,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: colorScheme,
      textTheme: _buildTextTheme(colorScheme),
      appBarTheme: _buildAppBarTheme(colorScheme),
      cardTheme: _buildCardTheme(colorScheme),
      inputDecorationTheme: _buildInputTheme(colorScheme),
      filledButtonTheme: _buildFilledButtonTheme(colorScheme),
      outlinedButtonTheme: _buildOutlinedButtonTheme(colorScheme),
      navigationBarTheme: _buildNavigationBarTheme(colorScheme),
      dividerTheme: DividerThemeData(color: colorScheme.outlineVariant),
      scaffoldBackgroundColor: colorScheme.surface,
    );
  }

  static ThemeData dark() {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: AppColors.brandPrimary,
      brightness: Brightness.dark,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: colorScheme,
      textTheme: _buildTextTheme(colorScheme),
      appBarTheme: _buildAppBarTheme(colorScheme),
      cardTheme: _buildCardTheme(colorScheme),
      inputDecorationTheme: _buildInputTheme(colorScheme),
      filledButtonTheme: _buildFilledButtonTheme(colorScheme),
      outlinedButtonTheme: _buildOutlinedButtonTheme(colorScheme),
      navigationBarTheme: _buildNavigationBarTheme(colorScheme),
      dividerTheme: DividerThemeData(color: colorScheme.outlineVariant),
      scaffoldBackgroundColor: colorScheme.surface,
    );
  }

  // --- Text Theme ---
  static TextTheme _buildTextTheme(ColorScheme colorScheme) {
    return GoogleFonts.interTextTheme().copyWith(
      displayLarge: GoogleFonts.inter(
        fontSize: 57,
        fontWeight: FontWeight.w400,
        color: colorScheme.onSurface,
      ),
      displayMedium: GoogleFonts.inter(
        fontSize: 45,
        fontWeight: FontWeight.w400,
        color: colorScheme.onSurface,
      ),
      headlineLarge: GoogleFonts.inter(
        fontSize: 32,
        fontWeight: FontWeight.w600,
        color: colorScheme.onSurface,
      ),
      headlineMedium: GoogleFonts.inter(
        fontSize: 28,
        fontWeight: FontWeight.w600,
        color: colorScheme.onSurface,
      ),
      titleLarge: GoogleFonts.inter(
        fontSize: 22,
        fontWeight: FontWeight.w600,
        color: colorScheme.onSurface,
      ),
      titleMedium: GoogleFonts.inter(
        fontSize: 16,
        fontWeight: FontWeight.w600,
        color: colorScheme.onSurface,
      ),
      titleSmall: GoogleFonts.inter(
        fontSize: 14,
        fontWeight: FontWeight.w600,
        color: colorScheme.onSurface,
      ),
      bodyLarge: GoogleFonts.inter(
        fontSize: 16,
        fontWeight: FontWeight.w400,
        color: colorScheme.onSurface,
      ),
      bodyMedium: GoogleFonts.inter(
        fontSize: 14,
        fontWeight: FontWeight.w400,
        color: colorScheme.onSurface,
      ),
      bodySmall: GoogleFonts.inter(
        fontSize: 12,
        fontWeight: FontWeight.w400,
        color: colorScheme.onSurfaceVariant,
      ),
      labelLarge: GoogleFonts.inter(
        fontSize: 14,
        fontWeight: FontWeight.w600,
        color: colorScheme.onSurface,
      ),
      labelSmall: GoogleFonts.inter(
        fontSize: 11,
        fontWeight: FontWeight.w500,
        color: colorScheme.onSurfaceVariant,
      ),
    );
  }

  // --- AppBar ---
  static AppBarTheme _buildAppBarTheme(ColorScheme colorScheme) {
    return AppBarTheme(
      centerTitle: false,
      elevation: 0,
      scrolledUnderElevation: 1,
      backgroundColor: colorScheme.surface,
      foregroundColor: colorScheme.onSurface,
      surfaceTintColor: colorScheme.surfaceTint,
    );
  }

  // --- Card ---
  static CardTheme _buildCardTheme(ColorScheme colorScheme) {
    return CardTheme(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: colorScheme.outlineVariant),
      ),
      color: colorScheme.surface,
    );
  }

  // --- Input ---
  static InputDecorationTheme _buildInputTheme(ColorScheme colorScheme) {
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(12),
      borderSide: BorderSide(color: colorScheme.outline),
    );

    return InputDecorationTheme(
      filled: true,
      fillColor: colorScheme.surfaceContainerHighest.withValues(alpha: 0.3),
      border: border,
      enabledBorder: border,
      focusedBorder: border.copyWith(
        borderSide: BorderSide(color: colorScheme.primary, width: 2),
      ),
      errorBorder: border.copyWith(
        borderSide: BorderSide(color: colorScheme.error),
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
    );
  }

  // --- Filled Button ---
  static FilledButtonThemeData _buildFilledButtonTheme(ColorScheme colorScheme) {
    return FilledButtonThemeData(
      style: FilledButton.styleFrom(
        minimumSize: const Size(double.infinity, 52),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
        textStyle: GoogleFonts.inter(
          fontSize: 16,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  // --- Outlined Button ---
  static OutlinedButtonThemeData _buildOutlinedButtonTheme(ColorScheme colorScheme) {
    return OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        minimumSize: const Size(double.infinity, 52),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
        ),
        side: BorderSide(color: colorScheme.outline),
        textStyle: GoogleFonts.inter(
          fontSize: 16,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  // --- Navigation Bar ---
  static NavigationBarThemeData _buildNavigationBarTheme(ColorScheme colorScheme) {
    return NavigationBarThemeData(
      elevation: 1,
      backgroundColor: colorScheme.surface,
      indicatorColor: colorScheme.secondaryContainer,
      labelTextStyle: WidgetStatePropertyAll(
        GoogleFonts.inter(
          fontSize: 12,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}
```

---

## AppColors (Raw Color Palette)

Raw colors that feed into `ColorScheme.fromSeed`. These are brand colors and semantic colors that exist outside of the Material color system.

```dart
// lib/core/theme/app_colors.dart
import 'package:flutter/material.dart';

abstract class AppColors {
  // --- Brand ---
  static const brandPrimary = Color(0xFF4CAF50);   // Green (friendly-nature)
  static const brandSecondary = Color(0xFF2196F3);  // Blue (sky/activity)

  // --- Semantic (outside Material system) ---
  static const success = Color(0xFF4CAF50);
  static const warning = Color(0xFFFFC107);
  static const info = Color(0xFF2196F3);

  // --- Neutral ---
  static const neutral50 = Color(0xFFFAFAFA);
  static const neutral100 = Color(0xFFF5F5F5);
  static const neutral200 = Color(0xFFEEEEEE);
  static const neutral300 = Color(0xFFE0E0E0);
  static const neutral400 = Color(0xFFBDBDBD);
  static const neutral500 = Color(0xFF9E9E9E);
  static const neutral600 = Color(0xFF757575);
  static const neutral700 = Color(0xFF616161);
  static const neutral800 = Color(0xFF424242);
  static const neutral900 = Color(0xFF212121);

  // --- Task Status ---
  static const statusPlanned = Color(0xFF90CAF9);
  static const statusActive = Color(0xFF66BB6A);
  static const statusCompleted = Color(0xFF4CAF50);
  static const statusCancelled = Color(0xFFEF5350);
}
```

Use these ONLY for values that don't map to Material's ColorScheme (e.g., status indicators, gradient backgrounds). For everything else, use `colorScheme.primary`, `colorScheme.error`, etc.

---

## Context Extensions

Convenient access to theme properties from any widget:

```dart
// lib/core/extensions/context_extensions.dart (add to existing file)
import 'package:flutter/material.dart';

extension ThemeExtension on BuildContext {
  ThemeData get theme => Theme.of(this);
  ColorScheme get colors => theme.colorScheme;
  TextTheme get text => theme.textTheme;
  bool get isDarkMode => theme.brightness == Brightness.dark;
}
```

### Usage

```dart
@override
Widget build(BuildContext context) {
  return Container(
    color: context.colors.surface,
    child: Text(
      'Hello',
      style: context.text.titleLarge?.copyWith(
        color: context.colors.primary,
      ),
    ),
  );
}
```

---

## ThemeMode Switching

Three modes: system (default), light, dark. Persisted with SharedPreferences.

```dart
// lib/core/providers/theme_provider.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

final themeModeProvider = StateNotifierProvider<ThemeModeNotifier, ThemeMode>((ref) {
  return ThemeModeNotifier();
});

class ThemeModeNotifier extends StateNotifier<ThemeMode> {
  ThemeModeNotifier() : super(ThemeMode.system) {
    _loadSavedTheme();
  }

  static const _key = 'theme_mode';

  Future<void> _loadSavedTheme() async {
    final prefs = await SharedPreferences.getInstance();
    final value = prefs.getString(_key);
    if (value != null) {
      state = ThemeMode.values.byName(value);
    }
  }

  Future<void> setThemeMode(ThemeMode mode) async {
    state = mode;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_key, mode.name);
  }
}
```

### Using in MaterialApp

```dart
class App extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
      theme: AppTheme.light(),
      darkTheme: AppTheme.dark(),
      themeMode: ref.watch(themeModeProvider),
      // ...
    );
  }
}
```

### Theme Picker Widget

```dart
class ThemePicker extends ConsumerWidget {
  const ThemePicker({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final currentMode = ref.watch(themeModeProvider);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(context.l10n.settings_theme, style: context.text.titleMedium),
        const SizedBox(height: 8),
        SegmentedButton<ThemeMode>(
          segments: [
            ButtonSegment(
              value: ThemeMode.system,
              label: Text(context.l10n.settings_theme_system),
              icon: const Icon(Icons.settings_suggest),
            ),
            ButtonSegment(
              value: ThemeMode.light,
              label: Text(context.l10n.settings_theme_light),
              icon: const Icon(Icons.light_mode),
            ),
            ButtonSegment(
              value: ThemeMode.dark,
              label: Text(context.l10n.settings_theme_dark),
              icon: const Icon(Icons.dark_mode),
            ),
          ],
          selected: {currentMode},
          onSelectionChanged: (selection) {
            ref.read(themeModeProvider.notifier).setThemeMode(selection.first);
          },
        ),
      ],
    );
  }
}
```

---

## Widget Theming Examples

### Styled Card

```dart
class TaskCard extends StatelessWidget {
  const TaskCard({super.key, required this.task});
  final Task task;

  @override
  Widget build(BuildContext context) {
    return Card(
      // Uses CardTheme from AppTheme automatically
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              task.title,
              style: context.text.titleMedium,
            ),
            const SizedBox(height: 4),
            Text(
              context.l10n.task_distance_km(task.distanceKm.toStringAsFixed(1)),
              style: context.text.bodyMedium?.copyWith(
                color: context.colors.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 12),
            _StatusChip(status: task.status),
          ],
        ),
      ),
    );
  }
}
```

### Status Chip with Semantic Colors

```dart
class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.status});
  final TaskStatus status;

  @override
  Widget build(BuildContext context) {
    final (color, label) = switch (status) {
      TaskStatus.planned => (AppColors.statusPlanned, context.l10n.task_status_planned),
      TaskStatus.active => (AppColors.statusActive, context.l10n.task_status_active),
      TaskStatus.completed => (AppColors.statusCompleted, context.l10n.task_status_completed),
      TaskStatus.cancelled => (AppColors.statusCancelled, context.l10n.task_status_cancelled),
    };

    return Chip(
      label: Text(label, style: context.text.labelSmall),
      backgroundColor: color.withValues(alpha: 0.15),
      side: BorderSide.none,
      padding: const EdgeInsets.symmetric(horizontal: 4),
    );
  }
}
```

### Input Field

```dart
// Uses InputDecorationTheme from AppTheme automatically
TextFormField(
  decoration: InputDecoration(
    labelText: context.l10n.login_email_label,
    hintText: context.l10n.login_email_hint,
    prefixIcon: const Icon(Icons.email_outlined),
  ),
  keyboardType: TextInputType.emailAddress,
)
```

### Full-Width Button

```dart
// Uses FilledButtonThemeData from AppTheme automatically
FilledButton(
  onPressed: _onSubmit,
  child: Text(context.l10n.login_button_submit),
)
```

---

## Custom Theme Extensions

For app-specific tokens that don't fit into Material's ColorScheme:

```dart
// lib/core/theme/custom_theme.dart
import 'package:flutter/material.dart';

@immutable
class AppCustomColors extends ThemeExtension<AppCustomColors> {
  const AppCustomColors({
    required this.success,
    required this.warning,
    required this.info,
  });

  final Color success;
  final Color warning;
  final Color info;

  @override
  AppCustomColors copyWith({Color? success, Color? warning, Color? info}) {
    return AppCustomColors(
      success: success ?? this.success,
      warning: warning ?? this.warning,
      info: info ?? this.info,
    );
  }

  @override
  AppCustomColors lerp(AppCustomColors? other, double t) {
    if (other is! AppCustomColors) return this;
    return AppCustomColors(
      success: Color.lerp(success, other.success, t)!,
      warning: Color.lerp(warning, other.warning, t)!,
      info: Color.lerp(info, other.info, t)!,
    );
  }

  static const light = AppCustomColors(
    success: AppColors.success,
    warning: AppColors.warning,
    info: AppColors.info,
  );

  static const dark = AppCustomColors(
    success: Color(0xFF81C784),
    warning: Color(0xFFFFD54F),
    info: Color(0xFF64B5F6),
  );
}

// Add to ThemeData in AppTheme:
// extensions: [AppCustomColors.light] or [AppCustomColors.dark]

// Access:
// Theme.of(context).extension<AppCustomColors>()!.success
```

---

## Anti-Patterns

### 1. Hardcoded colors
```dart
// BAD
Container(color: Color(0xFF4CAF50))
Text('Hello', style: TextStyle(color: Colors.blue))
// GOOD
Container(color: context.colors.primary)
Text('Hello', style: context.text.bodyLarge)
```

### 2. Hardcoded text styles
```dart
// BAD
Text('Title', style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold))
// GOOD
Text('Title', style: context.text.headlineMedium)
```

### 3. Colors.grey instead of ColorScheme
```dart
// BAD
Icon(Icons.info, color: Colors.grey)
// GOOD
Icon(Icons.info, color: context.colors.onSurfaceVariant)
```

### 4. Direct Theme.of(context) everywhere
```dart
// BAD (verbose)
final color = Theme.of(context).colorScheme.primary;
final style = Theme.of(context).textTheme.bodyLarge;
// GOOD (extensions)
final color = context.colors.primary;
final style = context.text.bodyLarge;
```

### 5. Forgetting dark mode
```dart
// BAD: looks good in light mode, invisible in dark mode
Container(color: Colors.white) // white on white in dark mode
// GOOD
Container(color: context.colors.surface) // adapts to light/dark
```

### 6. Opacity with deprecated withOpacity
```dart
// DEPRECATED
color.withOpacity(0.5)
// GOOD
color.withValues(alpha: 0.5)
```
