---
knowledge-base-summary: "Form widget, TextFormField, validation (UX-only). FormState management. Error display patterns (inline, snackbar, dialog). Async form submission with loading state. Multi-step forms. Form auto-save."
---
# Form Handling

## Philosophy

Flutter is a BRIDGE. Form validation in Flutter is UX-only: check if a field is empty, hint at expected format (email shape, minimum length). Real validation -- uniqueness, authorization, business rules -- happens in the API. Display API errors via snackbar or inline mapping.

## Core Pattern: Form + GlobalKey

Every form uses a `Form` widget with a `GlobalKey<FormState>` for validation control.

```dart
class _LoginScreenState extends State<LoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  void _submit() {
    if (_formKey.currentState?.validate() ?? false) {
      // All UX validations passed -- send to API
    }
  }

  @override
  Widget build(BuildContext context) {
    return Form(
      key: _formKey,
      child: Column(
        children: [
          TextFormField(
            controller: _emailController,
            validator: (value) {
              if (value == null || value.isEmpty) return 'Email is required';
              if (!value.contains('@')) return 'Enter a valid email';
              return null;
            },
          ),
          TextFormField(
            controller: _passwordController,
            validator: (value) {
              if (value == null || value.isEmpty) return 'Password is required';
              if (value.length < 6) return 'At least 6 characters';
              return null;
            },
          ),
        ],
      ),
    );
  }
}
```

## UX-Only Validation Rules

These validators run client-side for instant feedback. They do NOT replace server validation.

```dart
/// Collection of UX-only validators.
/// Real validation (uniqueness, auth, business rules) is in the API.
class AppValidators {
  AppValidators._();

  static FormFieldValidator<String> required(String fieldName) {
    return (value) {
      if (value == null || value.trim().isEmpty) {
        return '$fieldName is required';
      }
      return null;
    };
  }

  static FormFieldValidator<String> email() {
    return (value) {
      if (value == null || value.isEmpty) return 'Email is required';
      // Simple format check -- API validates deliverability
      final emailRegex = RegExp(r'^[^@\s]+@[^@\s]+\.[^@\s]+$');
      if (!emailRegex.hasMatch(value)) return 'Enter a valid email address';
      return null;
    };
  }

  static FormFieldValidator<String> minLength(int min) {
    return (value) {
      if (value != null && value.length < min) {
        return 'Must be at least $min characters';
      }
      return null;
    };
  }

  static FormFieldValidator<String> match(
    TextEditingController other,
    String message,
  ) {
    return (value) {
      if (value != other.text) return message;
      return null;
    };
  }

  /// Combine multiple validators. First error wins.
  static FormFieldValidator<String> compose(
    List<FormFieldValidator<String>> validators,
  ) {
    return (value) {
      for (final validator in validators) {
        final error = validator(value);
        if (error != null) return error;
      }
      return null;
    };
  }
}
```

**Usage:**

```dart
TextFormField(
  controller: _emailController,
  validator: AppValidators.email(),
)

TextFormField(
  controller: _passwordController,
  validator: AppValidators.compose([
    AppValidators.required('Password'),
    AppValidators.minLength(6),
  ]),
)

TextFormField(
  controller: _confirmPasswordController,
  validator: AppValidators.compose([
    AppValidators.required('Confirm password'),
    AppValidators.match(_passwordController, 'Passwords do not match'),
  ]),
)
```

## Error Display Patterns

### Inline Errors (under field)

TextFormField shows errors automatically when validator returns a string. The `errorText` parameter on `InputDecoration` can also be set manually for API-returned field errors.

```dart
// Map API field errors to controllers
void _handleApiError(Map<String, String> fieldErrors) {
  setState(() {
    _emailError = fieldErrors['email'];
    _passwordError = fieldErrors['password'];
  });
}

// In build:
AppInput(
  label: 'Email',
  controller: _emailController,
  error: _emailError, // From API response
)
```

### Snackbar for API Errors

General API errors (network, server, unknown) use snackbar.

```dart
void _showError(BuildContext context, String message) {
  ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(
      content: Text(message),
      backgroundColor: Theme.of(context).colorScheme.error,
      behavior: SnackBarBehavior.floating,
      action: SnackBarAction(
        label: 'Dismiss',
        textColor: Theme.of(context).colorScheme.onError,
        onPressed: () {
          ScaffoldMessenger.of(context).hideCurrentSnackBar();
        },
      ),
    ),
  );
}
```

## Form Submission with Loading State

Button shows spinner and is disabled during API call. Prevents double submission.

```dart
class _LoginFormState extends State<LoginForm> {
  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _isSubmitting = false;
  bool _obscurePassword = true;
  String? _emailError;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    // Clear previous API errors
    setState(() => _emailError = null);

    if (!(_formKey.currentState?.validate() ?? false)) return;

    setState(() => _isSubmitting = true);

    try {
      await ref.read(authProvider.notifier).login(
        email: _emailController.text.trim(),
        password: _passwordController.text,
      );
      // Navigation happens via auth state listener (go_router redirect)
    } on ApiException catch (e) {
      if (!mounted) return;

      if (e.fieldErrors != null) {
        // Map API field errors to inline display
        setState(() {
          _emailError = e.fieldErrors!['email'];
        });
      } else {
        // General error -> snackbar
        _showError(context, e.message);
      }
    } finally {
      if (mounted) {
        setState(() => _isSubmitting = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Form(
      key: _formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AppInput(
            label: context.l10n.email,
            controller: _emailController,
            hint: 'you@example.com',
            prefixIcon: Icons.email_outlined,
            keyboardType: TextInputType.emailAddress,
            textInputAction: TextInputAction.next,
            error: _emailError,
            validator: AppValidators.email(),
          ),
          const SizedBox(height: 16),
          AppInput(
            label: context.l10n.password,
            controller: _passwordController,
            obscureText: _obscurePassword,
            prefixIcon: Icons.lock_outlined,
            suffixIcon: _obscurePassword
                ? Icons.visibility
                : Icons.visibility_off,
            onSuffixTap: () {
              setState(() => _obscurePassword = !_obscurePassword);
            },
            textInputAction: TextInputAction.done,
            onSubmitted: (_) => _submit(),
            validator: AppValidators.compose([
              AppValidators.required(context.l10n.password),
              AppValidators.minLength(6),
            ]),
          ),
          const SizedBox(height: 24),
          AppButton(
            label: context.l10n.login,
            onPressed: _submit,
            isLoading: _isSubmitting,
            fullWidth: true,
            size: AppButtonSize.large,
          ),
        ],
      ),
    );
  }
}
```

## Custom Form Fields

### Date Picker Field

```dart
class AppDateField extends StatelessWidget {
  const AppDateField({
    super.key,
    required this.label,
    required this.value,
    required this.onChanged,
    this.firstDate,
    this.lastDate,
    this.error,
  });

  final String label;
  final DateTime? value;
  final ValueChanged<DateTime> onChanged;
  final DateTime? firstDate;
  final DateTime? lastDate;
  final String? error;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          label,
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                fontWeight: FontWeight.w500,
              ),
        ),
        const SizedBox(height: 6),
        InkWell(
          onTap: () async {
            final picked = await showDatePicker(
              context: context,
              initialDate: value ?? DateTime.now(),
              firstDate: firstDate ?? DateTime(1900),
              lastDate: lastDate ?? DateTime(2100),
            );
            if (picked != null) onChanged(picked);
          },
          borderRadius: BorderRadius.circular(8),
          child: InputDecorator(
            decoration: InputDecoration(
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
              ),
              errorText: error,
              suffixIcon: const Icon(Icons.calendar_today),
              contentPadding: const EdgeInsets.symmetric(
                horizontal: 16,
                vertical: 14,
              ),
            ),
            child: Text(
              value != null
                  ? DateFormat.yMMMd().format(value!)
                  : 'Select a date',
              style: TextStyle(
                color: value != null
                    ? Theme.of(context).colorScheme.onSurface
                    : Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
          ),
        ),
      ],
    );
  }
}
```

### Dropdown Field

```dart
class AppDropdownField<T> extends StatelessWidget {
  const AppDropdownField({
    super.key,
    required this.label,
    required this.value,
    required this.items,
    required this.onChanged,
    required this.itemLabel,
    this.hint,
    this.error,
  });

  final String label;
  final T? value;
  final List<T> items;
  final ValueChanged<T?> onChanged;
  final String Function(T) itemLabel;
  final String? hint;
  final String? error;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          label,
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                fontWeight: FontWeight.w500,
              ),
        ),
        const SizedBox(height: 6),
        DropdownButtonFormField<T>(
          value: value,
          items: items
              .map((item) => DropdownMenuItem(
                    value: item,
                    child: Text(itemLabel(item)),
                  ))
              .toList(),
          onChanged: onChanged,
          decoration: InputDecoration(
            hintText: hint,
            errorText: error,
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
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

## Form Auto-Save (Debounced)

For forms that save automatically as the user types (e.g., notes, profile bio).

```dart
class _AutoSaveFormState extends State<AutoSaveForm> {
  final _controller = TextEditingController();
  Timer? _debounce;

  @override
  void initState() {
    super.initState();
    _controller.text = widget.initialValue;
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _controller.dispose();
    super.dispose();
  }

  void _onChanged(String value) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 800), () {
      _save(value);
    });
  }

  Future<void> _save(String value) async {
    try {
      await ref.read(profileProvider.notifier).updateBio(value);
      // Optionally show a subtle "Saved" indicator
    } on ApiException catch (e) {
      if (mounted) _showError(context, e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    return AppInput(
      label: 'Bio',
      controller: _controller,
      hint: 'Tell us about yourself...',
      maxLines: 4,
      maxLength: 300,
      onChanged: _onChanged,
    );
  }
}
```

## Multi-Step Form with PageView

For registration, onboarding, or any flow with multiple pages.

```dart
class MultiStepForm extends StatefulWidget {
  const MultiStepForm({super.key});

  @override
  State<MultiStepForm> createState() => _MultiStepFormState();
}

class _MultiStepFormState extends State<MultiStepForm> {
  final _pageController = PageController();
  final _formKeys = List.generate(3, (_) => GlobalKey<FormState>());
  int _currentStep = 0;

  // Controllers for all steps
  final _nameController = TextEditingController();
  final _emailController = TextEditingController();
  final _phoneController = TextEditingController();

  @override
  void dispose() {
    _pageController.dispose();
    _nameController.dispose();
    _emailController.dispose();
    _phoneController.dispose();
    super.dispose();
  }

  void _nextStep() {
    if (!(_formKeys[_currentStep].currentState?.validate() ?? false)) return;

    if (_currentStep < _formKeys.length - 1) {
      _pageController.nextPage(
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeInOut,
      );
      setState(() => _currentStep++);
    } else {
      _submit();
    }
  }

  void _previousStep() {
    if (_currentStep > 0) {
      _pageController.previousPage(
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeInOut,
      );
      setState(() => _currentStep--);
    }
  }

  Future<void> _submit() async {
    // Collect all form data and send to API
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        // Step indicator
        _StepIndicator(
          totalSteps: _formKeys.length,
          currentStep: _currentStep,
        ),
        const SizedBox(height: 24),

        // Pages
        Expanded(
          child: PageView(
            controller: _pageController,
            physics: const NeverScrollableScrollPhysics(),
            children: [
              _buildStep1(),
              _buildStep2(),
              _buildStep3(),
            ],
          ),
        ),

        // Navigation buttons
        Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              if (_currentStep > 0)
                AppButton(
                  label: 'Back',
                  onPressed: _previousStep,
                  variant: AppButtonVariant.outline,
                ),
              const Spacer(),
              AppButton(
                label: _currentStep == _formKeys.length - 1
                    ? 'Submit'
                    : 'Next',
                onPressed: _nextStep,
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildStep1() {
    return Form(
      key: _formKeys[0],
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: AppInput(
          label: 'Full Name',
          controller: _nameController,
          validator: AppValidators.required('Name'),
          textInputAction: TextInputAction.next,
        ),
      ),
    );
  }

  Widget _buildStep2() {
    return Form(
      key: _formKeys[1],
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: AppInput(
          label: 'Email',
          controller: _emailController,
          validator: AppValidators.email(),
          keyboardType: TextInputType.emailAddress,
        ),
      ),
    );
  }

  Widget _buildStep3() {
    return Form(
      key: _formKeys[2],
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: AppInput(
          label: 'Phone',
          controller: _phoneController,
          validator: AppValidators.required('Phone'),
          keyboardType: TextInputType.phone,
        ),
      ),
    );
  }
}

class _StepIndicator extends StatelessWidget {
  const _StepIndicator({required this.totalSteps, required this.currentStep});

  final int totalSteps;
  final int currentStep;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: List.generate(totalSteps, (index) {
        final isActive = index <= currentStep;
        return Container(
          width: 32,
          height: 4,
          margin: const EdgeInsets.symmetric(horizontal: 4),
          decoration: BoxDecoration(
            color: isActive
                ? theme.colorScheme.primary
                : theme.colorScheme.surfaceContainerHighest,
            borderRadius: BorderRadius.circular(2),
          ),
        );
      }),
    );
  }
}
```

## Key Rules

1. **Validation is UX-only** -- empty check, format hint, min length. Never check business rules (e.g., "email already taken" is an API concern).
2. **Always disable submit button during loading** -- prevent double submission.
3. **Always dispose controllers** -- memory leak prevention.
4. **API errors go to snackbar** (general) or **inline** (field-specific).
5. **mounted check** -- always check `if (mounted)` before calling `setState` in async callbacks.
6. **i18n** -- all form labels and error messages through localization.
