---
knowledge-base-summary: "Login → register → forgot password → email verification. Secure token storage (flutter_secure_storage). Auto-login on app start (check stored refresh token). Token refresh interceptor in Dio. Logout (clear tokens, navigate to login). Multi-profile selection."
---
# Auth Flow

## Philosophy

Flutter is a BRIDGE. Authentication logic lives in the API. Flutter's role:
1. Collect credentials from the user.
2. Send them to the API.
3. Store the returned tokens securely.
4. Attach the access token to every API request.
5. Handle token expiry by refreshing automatically.
6. Clear tokens on logout.

No password hashing, no token generation, no session management in Flutter.

## Dependencies

```yaml
dependencies:
  dio: ^5.7.0
  flutter_secure_storage: ^9.2.0
  riverpod: ^2.5.0
```

## Token Storage

Tokens are sensitive. Store them encrypted using flutter_secure_storage. NEVER use SharedPreferences for tokens.

```dart
class TokenStorage {
  static const _accessTokenKey = 'access_token';
  static const _refreshTokenKey = 'refresh_token';

  final FlutterSecureStorage _storage;

  TokenStorage({FlutterSecureStorage? storage})
      : _storage = storage ?? const FlutterSecureStorage(
          aOptions: AndroidOptions(encryptedSharedPreferences: true),
          iOptions: IOSOptions(accessibility: KeychainAccessibility.first_unlock),
        );

  Future<String?> get accessToken => _storage.read(key: _accessTokenKey);
  Future<String?> get refreshToken => _storage.read(key: _refreshTokenKey);

  Future<void> saveTokens({
    required String accessToken,
    required String refreshToken,
  }) async {
    await Future.wait([
      _storage.write(key: _accessTokenKey, value: accessToken),
      _storage.write(key: _refreshTokenKey, value: refreshToken),
    ]);
  }

  Future<void> clearTokens() async {
    await Future.wait([
      _storage.delete(key: _accessTokenKey),
      _storage.delete(key: _refreshTokenKey),
    ]);
  }

  Future<bool> hasTokens() async {
    final refresh = await refreshToken;
    return refresh != null && refresh.isNotEmpty;
  }
}
```

## Auth State

```dart
sealed class AuthState {
  const AuthState();
}

class AuthInitial extends AuthState {
  const AuthInitial();
}

class AuthLoading extends AuthState {
  const AuthLoading();
}

class AuthAuthenticated extends AuthState {
  final User user;
  final List<UserProfile>? profiles; // For multi-profile SaaS
  const AuthAuthenticated(this.user, {this.profiles});
}

class AuthProfileSelection extends AuthState {
  final User user;
  final List<UserProfile> profiles;
  const AuthProfileSelection(this.user, this.profiles);
}

class AuthUnauthenticated extends AuthState {
  final String? message; // Optional logout reason
  const AuthUnauthenticated({this.message});
}
```

## Auth Notifier

```dart
class AuthNotifier extends StateNotifier<AuthState> {
  final ApiClient _api;
  final TokenStorage _tokenStorage;

  AuthNotifier(this._api, this._tokenStorage) : super(const AuthInitial());

  /// Called on app startup. Check for stored tokens and auto-login.
  Future<void> initialize() async {
    final hasTokens = await _tokenStorage.hasTokens();

    if (!hasTokens) {
      state = const AuthUnauthenticated();
      return;
    }

    // Try to refresh the token (validates the session)
    try {
      await _refreshToken();
      final user = await _fetchCurrentUser();
      state = AuthAuthenticated(user);
    } catch (_) {
      // Token expired or invalid -- force re-login
      await _tokenStorage.clearTokens();
      state = const AuthUnauthenticated();
    }
  }

  /// Login with email and password.
  Future<void> login({
    required String email,
    required String password,
  }) async {
    state = const AuthLoading();

    try {
      final response = await _api.post('/auth/login', data: {
        'email': email,
        'password': password,
      });

      final data = response.data as Map<String, dynamic>;
      await _tokenStorage.saveTokens(
        accessToken: data['accessToken'] as String,
        refreshToken: data['refreshToken'] as String,
      );

      final user = User.fromJson(data['user'] as Map<String, dynamic>);

      // Multi-profile check
      final profiles = (data['profiles'] as List?)
          ?.map((p) => UserProfile.fromJson(p as Map<String, dynamic>))
          .toList();

      if (profiles != null && profiles.length > 1) {
        state = AuthProfileSelection(user, profiles);
      } else {
        state = AuthAuthenticated(user, profiles: profiles);
      }
    } on ApiException catch (e) {
      state = AuthUnauthenticated(message: e.message);
      rethrow; // Let the UI handle field-level errors
    }
  }

  /// Register a new account.
  Future<void> register({
    required String name,
    required String email,
    required String password,
  }) async {
    state = const AuthLoading();

    try {
      final response = await _api.post('/auth/register', data: {
        'name': name,
        'email': email,
        'password': password,
      });

      final data = response.data as Map<String, dynamic>;
      await _tokenStorage.saveTokens(
        accessToken: data['accessToken'] as String,
        refreshToken: data['refreshToken'] as String,
      );

      final user = User.fromJson(data['user'] as Map<String, dynamic>);
      state = AuthAuthenticated(user);
    } on ApiException catch (e) {
      state = AuthUnauthenticated(message: e.message);
      rethrow;
    }
  }

  /// Select a profile (multi-profile SaaS flow).
  Future<void> selectProfile(UserProfile profile) async {
    try {
      await _api.post('/auth/select-profile', data: {
        'profileId': profile.id,
      });

      if (state is AuthProfileSelection) {
        final currentState = state as AuthProfileSelection;
        state = AuthAuthenticated(currentState.user);
      }
    } on ApiException {
      rethrow;
    }
  }

  /// Logout. Clear tokens, reset state, invalidate providers.
  Future<void> logout() async {
    try {
      // Tell API to invalidate the refresh token
      await _api.post('/auth/logout');
    } catch (_) {
      // Even if API call fails, clear local state
    }

    await _tokenStorage.clearTokens();
    state = const AuthUnauthenticated();
  }

  /// Refresh the access token using the stored refresh token.
  Future<void> _refreshToken() async {
    final refreshToken = await _tokenStorage.refreshToken;
    if (refreshToken == null) throw Exception('No refresh token');

    final response = await _api.post('/auth/refresh', data: {
      'refreshToken': refreshToken,
    });

    final data = response.data as Map<String, dynamic>;
    await _tokenStorage.saveTokens(
      accessToken: data['accessToken'] as String,
      refreshToken: data['refreshToken'] as String,
    );
  }

  Future<User> _fetchCurrentUser() async {
    final response = await _api.get('/auth/me');
    return User.fromJson(response.data as Map<String, dynamic>);
  }
}
```

## Riverpod Providers

```dart
final tokenStorageProvider = Provider<TokenStorage>((ref) {
  return TokenStorage();
});

final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  final api = ref.read(apiClientProvider);
  final tokenStorage = ref.read(tokenStorageProvider);
  return AuthNotifier(api, tokenStorage);
});
```

## Dio Auth Interceptor

Attaches access token to every request. On 401, refreshes the token and retries the original request. If refresh fails, logs out.

```dart
class AuthInterceptor extends Interceptor {
  final TokenStorage _tokenStorage;
  final Dio _dio; // The main Dio instance (for retry)
  final Dio _refreshDio; // Separate Dio for refresh (avoids interceptor loop)
  final Ref _ref;

  bool _isRefreshing = false;
  final _pendingRequests = <({RequestOptions options, ErrorInterceptorHandler handler})>[];

  AuthInterceptor({
    required TokenStorage tokenStorage,
    required Dio dio,
    required Ref ref,
  })  : _tokenStorage = tokenStorage,
        _dio = dio,
        _ref = ref,
        _refreshDio = Dio(BaseOptions(baseUrl: dio.options.baseUrl));

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final token = await _tokenStorage.accessToken;
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
    if (err.response?.statusCode != 401) {
      return handler.next(err);
    }

    // Already refreshing -- queue this request
    if (_isRefreshing) {
      _pendingRequests.add((options: err.requestOptions, handler: handler));
      return;
    }

    _isRefreshing = true;

    try {
      // Refresh the token
      final refreshToken = await _tokenStorage.refreshToken;
      if (refreshToken == null) throw Exception('No refresh token');

      final response = await _refreshDio.post('/auth/refresh', data: {
        'refreshToken': refreshToken,
      });

      final data = response.data as Map<String, dynamic>;
      final newAccessToken = data['accessToken'] as String;
      final newRefreshToken = data['refreshToken'] as String;

      await _tokenStorage.saveTokens(
        accessToken: newAccessToken,
        refreshToken: newRefreshToken,
      );

      // Retry original request with new token
      err.requestOptions.headers['Authorization'] = 'Bearer $newAccessToken';
      final retryResponse = await _dio.fetch(err.requestOptions);
      handler.resolve(retryResponse);

      // Retry all pending requests
      for (final pending in _pendingRequests) {
        pending.options.headers['Authorization'] = 'Bearer $newAccessToken';
        try {
          final res = await _dio.fetch(pending.options);
          pending.handler.resolve(res);
        } catch (e) {
          pending.handler.reject(
            DioException(requestOptions: pending.options, error: e),
          );
        }
      }
    } catch (_) {
      // Refresh failed -- force logout
      await _tokenStorage.clearTokens();
      _ref.read(authProvider.notifier).logout();
      handler.next(err);

      // Reject all pending requests
      for (final pending in _pendingRequests) {
        pending.handler.reject(
          DioException(
            requestOptions: pending.options,
            error: 'Session expired',
          ),
        );
      }
    } finally {
      _isRefreshing = false;
      _pendingRequests.clear();
    }
  }
}
```

## Token Refresh Flow

```
Request → 200 OK → done
Request → 401 →
  ├── Is refreshing? → queue request, wait
  └── Not refreshing →
        ├── POST /auth/refresh → 200 → save new tokens → retry original → retry queued
        └── POST /auth/refresh → fail → clear tokens → logout → reject all
```

## go_router Auth Redirect

```dart
GoRouter(
  refreshListenable: GoRouterRefreshStream(ref.watch(authProvider.notifier).stream),
  redirect: (context, state) {
    final auth = ref.read(authProvider);
    final isOnAuthPage = state.matchedLocation.startsWith('/auth');

    if (auth is AuthUnauthenticated || auth is AuthInitial) {
      return isOnAuthPage ? null : '/auth/login';
    }

    if (auth is AuthProfileSelection) {
      return '/auth/select-profile';
    }

    if (auth is AuthAuthenticated && isOnAuthPage) {
      return '/home';
    }

    return null;
  },
  routes: [
    GoRoute(path: '/auth/login', builder: (_, __) => const LoginScreen()),
    GoRoute(path: '/auth/register', builder: (_, __) => const RegisterScreen()),
    GoRoute(path: '/auth/select-profile', builder: (_, __) => const ProfileSelectionScreen()),
    GoRoute(path: '/home', builder: (_, __) => const HomeScreen()),
  ],
);
```

## Auto-Login Flow

```
App starts
  → AuthNotifier.initialize()
  → Has stored refresh token?
    ├── No → state = Unauthenticated → go_router redirects to /auth/login
    └── Yes → POST /auth/refresh
        ├── Success → GET /auth/me → state = Authenticated → go_router redirects to /home
        └── Failure → clear tokens → state = Unauthenticated → go_router redirects to /auth/login
```

```dart
class App extends ConsumerWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return FutureBuilder(
      future: ref.read(authProvider.notifier).initialize(),
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const MaterialApp(
            home: Scaffold(body: Center(child: CircularProgressIndicator())),
          );
        }

        return MaterialApp.router(
          routerConfig: ref.read(routerProvider),
          // ...
        );
      },
    );
  }
}
```

## Logout

Logout must:
1. Call API to invalidate the refresh token server-side.
2. Clear tokens from secure storage.
3. Set auth state to Unauthenticated (triggers go_router redirect).
4. Invalidate all providers that hold user-specific data.

```dart
// In a settings screen:
AppButton(
  label: context.l10n.logout,
  onPressed: () async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(context.l10n.logoutConfirmTitle),
        content: Text(context.l10n.logoutConfirmMessage),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(context.l10n.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(context.l10n.logout),
          ),
        ],
      ),
    );

    if (confirmed == true) {
      await ref.read(authProvider.notifier).logout();
      // go_router redirect handles navigation to login
    }
  },
  variant: AppButtonVariant.outline,
  icon: Icons.logout,
)
```

## Multi-Profile Selection (SaaS)

After login, if the user has multiple profiles (e.g., personal + company), show a selection screen before proceeding to home.

```dart
class ProfileSelectionScreen extends ConsumerWidget {
  const ProfileSelectionScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authProvider);

    if (auth is! AuthProfileSelection) {
      return const SizedBox.shrink();
    }

    return Scaffold(
      appBar: AppBar(title: Text(context.l10n.selectProfile)),
      body: ListView.builder(
        padding: const EdgeInsets.all(16),
        itemCount: auth.profiles.length,
        itemBuilder: (context, index) {
          final profile = auth.profiles[index];
          return AppCard(
            onTap: () {
              ref.read(authProvider.notifier).selectProfile(profile);
            },
            child: Row(
              children: [
                AppAvatar(
                  imageUrl: profile.avatarUrl,
                  name: profile.name,
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        profile.name,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      Text(
                        profile.role,
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    ],
                  ),
                ),
                const Icon(Icons.chevron_right),
              ],
            ),
          );
        },
      ),
    );
  }
}
```

## Key Rules

1. **Tokens in flutter_secure_storage** -- never SharedPreferences, never in memory only.
2. **Separate Dio instance for refresh** -- prevents interceptor infinite loop.
3. **Queue concurrent 401s** -- only one refresh at a time, others wait.
4. **Refresh failure = forced logout** -- no retry loops.
5. **go_router redirect** -- single source of truth for auth navigation.
6. **Always confirm logout** -- dialog before clearing tokens.
7. **mounted check** -- check before setState in async auth callbacks.
