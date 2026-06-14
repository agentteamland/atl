---
knowledge-base-summary: "In-app browser for 3D Secure payments, OAuth flows, legal pages, external content. WebView widget configuration. JavaScript bridge for communication. Callback URL interception (payment result, auth code). When to use in-app WebView vs external browser. Security considerations."
---
# WebView

## Overview

In-app browser for controlled web flows: 3D Secure payment verification, OAuth with external providers, legal pages (terms, privacy), and external content that must stay within the app. For general external links (blog posts, social media), use `url_launcher` to open the system browser instead.

## When to Use WebView vs External Browser

| Scenario | Use | Reason |
|----------|-----|--------|
| 3D Secure payment | WebView | Need to intercept callback URL and extract result |
| OAuth (Google, Apple) | WebView | Need to intercept redirect URI and extract auth code |
| Terms of Service, Privacy Policy | WebView | Keep user in-app, no need for callback |
| Blog post, external article | External browser (`url_launcher`) | No control needed, user can manage tabs |
| Support/help page | WebView | In-app experience, back button returns to app |
| Partner/third-party form | WebView | Need to detect completion and return |

## Dependency

```yaml
# pubspec.yaml
dependencies:
  webview_flutter: ^4.0.0

# For url_launcher (external links)
  url_launcher: ^6.0.0
```

### Platform Configuration

**iOS:** Add to `ios/Runner/Info.plist`:

```xml
<key>NSAppTransportSecurity</key>
<dict>
    <key>NSAllowsArbitraryLoads</key>
    <true/>
</dict>
```

**Android:** Add to `android/app/src/main/AndroidManifest.xml`:

```xml
<application
    android:usesCleartextTraffic="true">
```

## Basic WebView Screen

```dart
// lib/core/widgets/webview_screen.dart

import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

class WebViewScreen extends StatefulWidget {
  final String url;
  final String title;

  const WebViewScreen({
    super.key,
    required this.url,
    required this.title,
  });

  @override
  State<WebViewScreen> createState() => _WebViewScreenState();
}

class _WebViewScreenState extends State<WebViewScreen> {
  late final WebViewController _controller;
  bool _isLoading = true;
  double _progress = 0;

  @override
  void initState() {
    super.initState();
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setNavigationDelegate(
        NavigationDelegate(
          onPageStarted: (url) {
            setState(() => _isLoading = true);
          },
          onPageFinished: (url) {
            setState(() => _isLoading = false);
          },
          onProgress: (progress) {
            setState(() => _progress = progress / 100);
          },
          onWebResourceError: (error) {
            // Handle page load errors
            setState(() => _isLoading = false);
          },
        ),
      )
      ..loadRequest(Uri.parse(widget.url));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.title),
        bottom: _isLoading
            ? PreferredSize(
                preferredSize: const Size.fromHeight(2),
                child: LinearProgressIndicator(value: _progress),
              )
            : null,
      ),
      body: WebViewWidget(controller: _controller),
    );
  }
}
```

## 3D Secure Payment Flow

The most common WebView use case. The payment provider gives a URL to verify the card. After verification, they redirect to a callback URL with the result. The app intercepts this redirect and extracts the payment result.

### Flow

```
1. API returns 3DS verification URL + callback URL pattern
2. App opens WebView with verification URL
3. User completes 3DS challenge in WebView
4. Payment provider redirects to callback URL with result parameters
5. App detects callback URL → extracts result → closes WebView
6. App sends result to API for confirmation
```

### Implementation

```dart
// lib/features/payment/screens/payment_3ds_screen.dart

import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

class Payment3dsResult {
  final bool success;
  final String? transactionId;
  final String? errorMessage;

  const Payment3dsResult({
    required this.success,
    this.transactionId,
    this.errorMessage,
  });
}

class Payment3dsScreen extends StatefulWidget {
  final String verificationUrl;
  final String callbackUrlPrefix;

  const Payment3dsScreen({
    super.key,
    required this.verificationUrl,
    required this.callbackUrlPrefix,
  });

  @override
  State<Payment3dsScreen> createState() => _Payment3dsScreenState();
}

class _Payment3dsScreenState extends State<Payment3dsScreen> {
  late final WebViewController _controller;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setNavigationDelegate(
        NavigationDelegate(
          onPageStarted: (url) {
            setState(() => _isLoading = true);
          },
          onPageFinished: (url) {
            setState(() => _isLoading = false);
          },
          onNavigationRequest: (request) {
            // Intercept callback URL
            if (request.url.startsWith(widget.callbackUrlPrefix)) {
              _handleCallback(request.url);
              return NavigationDecision.prevent; // Do not navigate
            }

            // Restrict to allowed domains
            final uri = Uri.parse(request.url);
            if (_isAllowedDomain(uri.host)) {
              return NavigationDecision.navigate;
            }

            return NavigationDecision.prevent;
          },
        ),
      )
      ..loadRequest(Uri.parse(widget.verificationUrl));
  }

  bool _isAllowedDomain(String host) {
    const allowedDomains = [
      'example-app.com',
      'bank.example.com',
      'secure.payment-provider.com',
      'acs.3dsecure.com',
    ];
    return allowedDomains.any(
      (domain) => host == domain || host.endsWith('.$domain'),
    );
  }

  void _handleCallback(String callbackUrl) {
    final uri = Uri.parse(callbackUrl);
    final status = uri.queryParameters['status'];
    final transactionId = uri.queryParameters['transaction_id'];
    final error = uri.queryParameters['error'];

    final result = Payment3dsResult(
      success: status == 'success',
      transactionId: transactionId,
      errorMessage: error,
    );

    // Return result to the calling screen
    Navigator.of(context).pop(result);
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      // Prevent accidental back during payment
      canPop: false,
      onPopInvokedWithResult: (didPop, result) {
        if (!didPop) {
          _showCancelConfirmation();
        }
      },
      child: Scaffold(
        appBar: AppBar(
          title: Text(context.l10n.paymentVerification),
          leading: IconButton(
            icon: const Icon(Icons.close),
            onPressed: _showCancelConfirmation,
          ),
        ),
        body: Stack(
          children: [
            WebViewWidget(controller: _controller),
            if (_isLoading)
              const Center(child: CircularProgressIndicator()),
          ],
        ),
      ),
    );
  }

  void _showCancelConfirmation() {
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(context.l10n.cancelPayment),
        content: Text(context.l10n.cancelPaymentMessage),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(), // Close dialog
            child: Text(context.l10n.continuePayment),
          ),
          TextButton(
            onPressed: () {
              Navigator.of(context).pop(); // Close dialog
              Navigator.of(this.context).pop(
                const Payment3dsResult(success: false),
              ); // Close WebView
            },
            style: TextButton.styleFrom(
              foregroundColor: Theme.of(context).colorScheme.error,
            ),
            child: Text(context.l10n.cancelPayment),
          ),
        ],
      ),
    );
  }
}
```

### Calling the 3DS Screen

```dart
// From the payment screen:
Future<void> _processPayment() async {
  // 1. Send payment to API, get 3DS URL
  final threeDsData = await ref.read(paymentRepositoryProvider)
      .initiatePayment(amount: 19.99, currency: 'TRY');

  if (!mounted) return;

  // 2. Open 3DS WebView
  final result = await Navigator.of(context).push<Payment3dsResult>(
    MaterialPageRoute(
      builder: (context) => Payment3dsScreen(
        verificationUrl: threeDsData.verificationUrl,
        callbackUrlPrefix: threeDsData.callbackUrl,
      ),
    ),
  );

  if (!mounted) return;

  // 3. Handle result
  if (result != null && result.success) {
    // Confirm with API
    await ref.read(paymentRepositoryProvider).confirmPayment(
      transactionId: result.transactionId!,
    );
    if (mounted) {
      showSuccessSnackbar(context, context.l10n.paymentSuccessful);
      context.pop();
    }
  } else {
    showErrorSnackbar(
      context,
      result?.errorMessage ?? context.l10n.paymentFailed,
    );
  }
}
```

## OAuth WebView Flow

Similar to 3DS but for authentication with external providers:

```dart
// lib/features/auth/screens/oauth_webview_screen.dart

class OAuthWebViewScreen extends StatefulWidget {
  final String authUrl;
  final String redirectUri;

  const OAuthWebViewScreen({
    super.key,
    required this.authUrl,
    required this.redirectUri,
  });

  @override
  State<OAuthWebViewScreen> createState() => _OAuthWebViewScreenState();
}

class _OAuthWebViewScreenState extends State<OAuthWebViewScreen> {
  late final WebViewController _controller;
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setNavigationDelegate(
        NavigationDelegate(
          onPageStarted: (_) => setState(() => _isLoading = true),
          onPageFinished: (_) => setState(() => _isLoading = false),
          onNavigationRequest: (request) {
            if (request.url.startsWith(widget.redirectUri)) {
              final uri = Uri.parse(request.url);
              final code = uri.queryParameters['code'];
              final error = uri.queryParameters['error'];

              if (code != null) {
                Navigator.of(context).pop(code); // Return auth code
              } else {
                Navigator.of(context).pop(null); // Auth failed
              }
              return NavigationDecision.prevent;
            }
            return NavigationDecision.navigate;
          },
        ),
      )
      ..loadRequest(Uri.parse(widget.authUrl));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(context.l10n.signIn),
      ),
      body: Stack(
        children: [
          WebViewWidget(controller: _controller),
          if (_isLoading)
            const Center(child: CircularProgressIndicator()),
        ],
      ),
    );
  }
}
```

## JavaScript Bridge

For bidirectional communication between Flutter and web content. Use cases: embedded forms that need to pass data back, custom web components that trigger native actions.

```dart
class InteractiveWebViewScreen extends StatefulWidget {
  final String url;

  const InteractiveWebViewScreen({super.key, required this.url});

  @override
  State<InteractiveWebViewScreen> createState() =>
      _InteractiveWebViewScreenState();
}

class _InteractiveWebViewScreenState extends State<InteractiveWebViewScreen> {
  late final WebViewController _controller;

  @override
  void initState() {
    super.initState();
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..addJavaScriptChannel(
        'FlutterBridge',
        onMessageReceived: (message) {
          _handleWebMessage(message.message);
        },
      )
      ..loadRequest(Uri.parse(widget.url));
  }

  void _handleWebMessage(String message) {
    // Parse message from web page
    // Web page calls: FlutterBridge.postMessage('{"action":"close","data":"..."}')
    try {
      final data = jsonDecode(message) as Map<String, dynamic>;
      final action = data['action'] as String;

      switch (action) {
        case 'close':
          Navigator.of(context).pop(data['data']);
          break;
        case 'share':
          // Trigger native share sheet
          Share.share(data['data'] as String);
          break;
        case 'navigate':
          // Navigate to a native screen
          context.push(data['route'] as String);
          break;
      }
    } catch (e) {
      // Invalid message format, ignore
    }
  }

  /// Send data from Flutter to the web page
  Future<void> _sendToWeb(String action, Map<String, dynamic> data) async {
    final json = jsonEncode({'action': action, 'data': data});
    await _controller.runJavaScript(
      'window.handleFlutterMessage($json)',
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(context.l10n.webContent)),
      body: WebViewWidget(controller: _controller),
    );
  }
}
```

### Web Side (JavaScript)

The web page communicates with Flutter via the registered channel:

```javascript
// Send message to Flutter
FlutterBridge.postMessage(JSON.stringify({
  action: 'close',
  data: { result: 'success', value: 42 }
}));

// Receive message from Flutter
window.handleFlutterMessage = function(message) {
  const { action, data } = message;
  if (action === 'setTheme') {
    document.body.classList.toggle('dark', data.dark);
  }
};
```

## Security

### Domain Restriction

Always restrict which domains the WebView can navigate to:

```dart
onNavigationRequest: (request) {
  final uri = Uri.parse(request.url);
  if (_isAllowedDomain(uri.host)) {
    return NavigationDecision.navigate;
  }
  // Open disallowed domains in external browser instead
  launchUrl(Uri.parse(request.url), mode: LaunchMode.externalApplication);
  return NavigationDecision.prevent;
},
```

### Clear WebView Data

After sensitive flows (payment, auth), clear cookies and cache:

```dart
void _clearWebViewData() async {
  final cookieManager = WebViewCookieManager();
  await cookieManager.clearCookies();

  // Clear cache via controller
  await _controller.clearCache();
  await _controller.clearLocalStorage();
}
```

## External Links Helper

For links that should open in the system browser:

```dart
// lib/core/utils/url_utils.dart

import 'package:url_launcher/url_launcher.dart';

Future<void> openExternalUrl(String url) async {
  final uri = Uri.parse(url);
  if (await canLaunchUrl(uri)) {
    await launchUrl(uri, mode: LaunchMode.externalApplication);
  }
}

Future<void> openInAppUrl(BuildContext context, {
  required String url,
  required String title,
}) async {
  Navigator.of(context).push(
    MaterialPageRoute(
      builder: (context) => WebViewScreen(url: url, title: title),
    ),
  );
}
```

## Key Rules

1. **WebView for flow control.** Use when you need to intercept a callback URL (payment, OAuth).
2. **External browser for general links.** Blog posts, social media, documentation.
3. **Always restrict domains.** Never allow the WebView to navigate freely.
4. **Always show loading indicator.** Users must see that something is happening while the page loads.
5. **Block back button during sensitive flows.** Payment and auth flows should show a confirmation dialog before closing.
6. **Clear data after sensitive flows.** Cookies and cache from payment/auth should not persist.
7. **JavaScript bridge for bidirectional communication.** Use named channels, not eval.
