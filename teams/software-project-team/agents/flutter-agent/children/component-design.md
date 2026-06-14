---
knowledge-base-summary: "Reusable widget philosophy. Small, focused widgets with props. Prop extension pattern: required vs optional parameters. Composition over inheritance. Common components: AppButton, AppCard, AppInput, AppImage. Consistent API across components."
---
# Component Design

## Philosophy

Flutter is a BRIDGE. Components display data and collect input. They do NOT contain business logic, calculations, or validation beyond basic UX checks (empty field, format hint). Every component is a small, focused widget with a clear API (props).

## Rules

1. **Small and focused** -- one widget does one thing. A button is a button, not a button-with-form-logic.
2. **Composition over inheritance** -- combine widgets, don't extend them. Use parameters to change behavior.
3. **Required vs optional** -- required params for identity (label, onPressed), optional params for variants (color, size, icon).
4. **App prefix** -- all custom components start with `App`: AppButton, AppCard, AppInput, AppAvatar, AppBadge.
5. **Extract rule** -- if you copy-paste a widget, extract it immediately. No exceptions.
6. **No business logic** -- components receive data via props and report events via callbacks. Never fetch data inside a component.

## Prop Extension Pattern

Same component, different looks. Instead of creating `PrimaryButton`, `SecondaryButton`, `OutlineButton`, create one `AppButton` with a `variant` parameter.

```dart
// WRONG: separate classes
class PrimaryButton extends StatelessWidget { ... }
class SecondaryButton extends StatelessWidget { ... }

// RIGHT: one class, variant prop
AppButton(label: 'Save', variant: AppButtonVariant.primary)
AppButton(label: 'Cancel', variant: AppButtonVariant.outline)
```

## Standard Component Library

### AppButton

Full-featured button with variants, loading state, icon support, and disabled state.

```dart
enum AppButtonVariant { primary, secondary, outline, text }
enum AppButtonSize { small, medium, large }

class AppButton extends StatelessWidget {
  const AppButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.variant = AppButtonVariant.primary,
    this.size = AppButtonSize.medium,
    this.isLoading = false,
    this.isDisabled = false,
    this.icon,
    this.fullWidth = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final AppButtonVariant variant;
  final AppButtonSize size;
  final bool isLoading;
  final bool isDisabled;
  final IconData? icon;
  final bool fullWidth;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final effectiveOnPressed = (isLoading || isDisabled) ? null : onPressed;

    final child = isLoading
        ? SizedBox(
            width: _iconSize,
            height: _iconSize,
            child: CircularProgressIndicator(
              strokeWidth: 2,
              color: _foregroundColor(theme),
            ),
          )
        : Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (icon != null) ...[
                Icon(icon, size: _iconSize),
                const SizedBox(width: 8),
              ],
              Text(label),
            ],
          );

    final style = _buildStyle(theme);

    Widget button;
    switch (variant) {
      case AppButtonVariant.primary:
        button = FilledButton(
          onPressed: effectiveOnPressed,
          style: style,
          child: child,
        );
      case AppButtonVariant.secondary:
        button = FilledButton.tonal(
          onPressed: effectiveOnPressed,
          style: style,
          child: child,
        );
      case AppButtonVariant.outline:
        button = OutlinedButton(
          onPressed: effectiveOnPressed,
          style: style,
          child: child,
        );
      case AppButtonVariant.text:
        button = TextButton(
          onPressed: effectiveOnPressed,
          style: style,
          child: child,
        );
    }

    if (fullWidth) {
      return SizedBox(width: double.infinity, child: button);
    }
    return button;
  }

  double get _iconSize {
    return switch (size) {
      AppButtonSize.small => 16.0,
      AppButtonSize.medium => 20.0,
      AppButtonSize.large => 24.0,
    };
  }

  double get _fontSize {
    return switch (size) {
      AppButtonSize.small => 12.0,
      AppButtonSize.medium => 14.0,
      AppButtonSize.large => 16.0,
    };
  }

  EdgeInsets get _padding {
    return switch (size) {
      AppButtonSize.small =>
        const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      AppButtonSize.medium =>
        const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
      AppButtonSize.large =>
        const EdgeInsets.symmetric(horizontal: 28, vertical: 16),
    };
  }

  Color _foregroundColor(ThemeData theme) {
    return switch (variant) {
      AppButtonVariant.primary => theme.colorScheme.onPrimary,
      AppButtonVariant.secondary => theme.colorScheme.onSecondaryContainer,
      AppButtonVariant.outline => theme.colorScheme.primary,
      AppButtonVariant.text => theme.colorScheme.primary,
    };
  }

  ButtonStyle _buildStyle(ThemeData theme) {
    return ButtonStyle(
      padding: WidgetStatePropertyAll(_padding),
      textStyle: WidgetStatePropertyAll(
        TextStyle(fontSize: _fontSize, fontWeight: FontWeight.w600),
      ),
      shape: WidgetStatePropertyAll(
        RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
    );
  }
}
```

**Usage:**

```dart
// Primary with loading
AppButton(
  label: 'Save',
  onPressed: _handleSave,
  isLoading: isSaving,
)

// Outline with icon
AppButton(
  label: 'Add Item',
  onPressed: _handleAdd,
  variant: AppButtonVariant.outline,
  icon: Icons.add,
)

// Full width large
AppButton(
  label: 'Continue',
  onPressed: _handleContinue,
  size: AppButtonSize.large,
  fullWidth: true,
)

// Disabled text button
AppButton(
  label: 'Skip',
  onPressed: null,
  variant: AppButtonVariant.text,
  isDisabled: true,
)
```

### AppInput

Text input with label, hint, error message, prefix/suffix icon, and obscure toggle for passwords.

```dart
class AppInput extends StatelessWidget {
  const AppInput({
    super.key,
    required this.label,
    this.controller,
    this.hint,
    this.error,
    this.prefixIcon,
    this.suffixIcon,
    this.onSuffixTap,
    this.obscureText = false,
    this.keyboardType,
    this.textInputAction,
    this.maxLines = 1,
    this.maxLength,
    this.enabled = true,
    this.autofocus = false,
    this.onChanged,
    this.onSubmitted,
    this.validator,
    this.focusNode,
  });

  final String label;
  final TextEditingController? controller;
  final String? hint;
  final String? error;
  final IconData? prefixIcon;
  final IconData? suffixIcon;
  final VoidCallback? onSuffixTap;
  final bool obscureText;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final int maxLines;
  final int? maxLength;
  final bool enabled;
  final bool autofocus;
  final ValueChanged<String>? onChanged;
  final ValueChanged<String>? onSubmitted;
  final FormFieldValidator<String>? validator;
  final FocusNode? focusNode;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          label,
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.w500,
            color: error != null
                ? theme.colorScheme.error
                : theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 6),
        TextFormField(
          controller: controller,
          obscureText: obscureText,
          keyboardType: keyboardType,
          textInputAction: textInputAction,
          maxLines: maxLines,
          maxLength: maxLength,
          enabled: enabled,
          autofocus: autofocus,
          focusNode: focusNode,
          onChanged: onChanged,
          onFieldSubmitted: onSubmitted,
          validator: validator,
          decoration: InputDecoration(
            hintText: hint,
            errorText: error,
            prefixIcon: prefixIcon != null ? Icon(prefixIcon) : null,
            suffixIcon: suffixIcon != null
                ? IconButton(
                    icon: Icon(suffixIcon),
                    onPressed: onSuffixTap,
                  )
                : null,
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
            ),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: theme.colorScheme.outline),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(
                color: theme.colorScheme.primary,
                width: 2,
              ),
            ),
            errorBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide(color: theme.colorScheme.error),
            ),
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 16,
              vertical: 14,
            ),
          ),
        ),
      ],
    );
  }
}
```

**Usage:**

```dart
// Email input with icon
AppInput(
  label: 'Email',
  controller: _emailController,
  hint: 'you@example.com',
  prefixIcon: Icons.email_outlined,
  keyboardType: TextInputType.emailAddress,
  textInputAction: TextInputAction.next,
  error: _emailError,
)

// Password input with toggle
AppInput(
  label: 'Password',
  controller: _passwordController,
  obscureText: !_showPassword,
  prefixIcon: Icons.lock_outlined,
  suffixIcon: _showPassword ? Icons.visibility_off : Icons.visibility,
  onSuffixTap: () => setState(() => _showPassword = !_showPassword),
)

// Multiline textarea
AppInput(
  label: 'Notes',
  controller: _notesController,
  hint: 'Enter your notes...',
  maxLines: 4,
  maxLength: 500,
)
```

### AppCard

Container for grouped content with optional header and action.

```dart
class AppCard extends StatelessWidget {
  const AppCard({
    super.key,
    required this.child,
    this.title,
    this.trailing,
    this.onTap,
    this.padding,
    this.margin,
  });

  final Widget child;
  final String? title;
  final Widget? trailing;
  final VoidCallback? onTap;
  final EdgeInsets? padding;
  final EdgeInsets? margin;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    Widget content = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        if (title != null) ...[
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                title!,
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
              if (trailing != null) trailing!,
            ],
          ),
          const SizedBox(height: 12),
        ],
        child,
      ],
    );

    return Card(
      margin: margin ?? const EdgeInsets.symmetric(vertical: 4),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: padding ?? const EdgeInsets.all(16),
          child: content,
        ),
      ),
    );
  }
}
```

### AppAvatar

Circular avatar with image, fallback to initials, and optional badge.

```dart
class AppAvatar extends StatelessWidget {
  const AppAvatar({
    super.key,
    this.imageUrl,
    this.name,
    this.size = 40,
    this.badge,
  });

  final String? imageUrl;
  final String? name;
  final double size;
  final Widget? badge;

  String get _initials {
    if (name == null || name!.isEmpty) return '?';
    final parts = name!.trim().split(' ');
    if (parts.length >= 2) {
      return '${parts.first[0]}${parts.last[0]}'.toUpperCase();
    }
    return parts.first[0].toUpperCase();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    Widget avatar;
    if (imageUrl != null && imageUrl!.isNotEmpty) {
      avatar = CircleAvatar(
        radius: size / 2,
        backgroundImage: CachedNetworkImageProvider(imageUrl!),
        backgroundColor: theme.colorScheme.surfaceContainerHighest,
      );
    } else {
      avatar = CircleAvatar(
        radius: size / 2,
        backgroundColor: theme.colorScheme.primaryContainer,
        child: Text(
          _initials,
          style: TextStyle(
            fontSize: size * 0.36,
            fontWeight: FontWeight.w600,
            color: theme.colorScheme.onPrimaryContainer,
          ),
        ),
      );
    }

    if (badge == null) return avatar;

    return Stack(
      children: [
        avatar,
        Positioned(right: 0, bottom: 0, child: badge!),
      ],
    );
  }
}
```

### AppBadge

Small label for status, count, or category.

```dart
enum AppBadgeVariant { info, success, warning, error }

class AppBadge extends StatelessWidget {
  const AppBadge({
    super.key,
    required this.label,
    this.variant = AppBadgeVariant.info,
    this.icon,
  });

  final String label;
  final AppBadgeVariant variant;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = _colors(theme);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: colors.$1,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 12, color: colors.$2),
            const SizedBox(width: 4),
          ],
          Text(
            label,
            style: theme.textTheme.labelSmall?.copyWith(
              color: colors.$2,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }

  (Color background, Color foreground) _colors(ThemeData theme) {
    return switch (variant) {
      AppBadgeVariant.info => (
        theme.colorScheme.primaryContainer,
        theme.colorScheme.onPrimaryContainer,
      ),
      AppBadgeVariant.success => (
        theme.colorScheme.tertiaryContainer,
        theme.colorScheme.onTertiaryContainer,
      ),
      AppBadgeVariant.warning => (
        Colors.orange.shade100,
        Colors.orange.shade900,
      ),
      AppBadgeVariant.error => (
        theme.colorScheme.errorContainer,
        theme.colorScheme.onErrorContainer,
      ),
    };
  }
}
```

## When to Extract a Widget

Extract when:
- You copy-paste the same widget structure a second time
- A build method exceeds ~50 lines
- A widget has its own local state (needs StatefulWidget)
- A section of UI can be described with a single noun ("user card", "price badge")

Do NOT extract when:
- The widget is used only once and is <20 lines
- Extracting would require passing 10+ parameters (the cure is worse than the disease)
- The widget is tightly coupled to a specific screen's layout

## File Organization

```
lib/
  shared/
    widgets/
      app_button.dart
      app_card.dart
      app_input.dart
      app_avatar.dart
      app_badge.dart
      app_empty_state.dart
      app_error_view.dart
      app_loading_view.dart
  features/
    tasks/
      widgets/           ← Feature-specific widgets (not reusable)
        task_summary_card.dart
        task_route_map.dart
```

- **shared/widgets/** -- reusable across all features (App prefix)
- **features/{name}/widgets/** -- feature-specific widgets (no App prefix, feature-specific naming)
