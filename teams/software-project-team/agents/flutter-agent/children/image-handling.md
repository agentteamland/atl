---
knowledge-base-summary: "CachedNetworkImage for remote images. Placeholder and error widgets. Image picker (camera + gallery). Image compression before upload. Asset images vs network images. Avatar widget pattern. Image carousel/gallery."
---
# Image Handling

## Philosophy

Flutter is a BRIDGE. Images are either fetched from the API (network), bundled with the app (assets), or picked from the user's device (camera/gallery). All image processing decisions (resizing, cropping, watermarking) happen server-side. Client-side compression before upload is the only exception -- it reduces upload time and bandwidth.

## Dependencies

```yaml
dependencies:
  cached_network_image: ^3.4.0
  image_picker: ^1.1.0
  flutter_image_compress: ^2.3.0
  dio: ^5.7.0
```

## CachedNetworkImage

All remote images use `CachedNetworkImage`. Never use `Image.network()` directly -- it has no cache, no placeholder, no error handling.

```dart
import 'package:cached_network_image/cached_network_image.dart';

// Basic usage
CachedNetworkImage(
  imageUrl: 'https://api.example.com/images/task-123.jpg',
  placeholder: (context, url) => const _ImagePlaceholder(),
  errorWidget: (context, url, error) => const _ImageError(),
  fit: BoxFit.cover,
)

// With specific dimensions (prevents layout shift)
CachedNetworkImage(
  imageUrl: imageUrl,
  width: 120,
  height: 120,
  fit: BoxFit.cover,
  placeholder: (context, url) => const _ImagePlaceholder(),
  errorWidget: (context, url, error) => const _ImageError(),
)
```

### Placeholder and Error Widgets

```dart
class _ImagePlaceholder extends StatelessWidget {
  const _ImagePlaceholder();

  @override
  Widget build(BuildContext context) {
    return Container(
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: Center(
        child: CircularProgressIndicator(
          strokeWidth: 2,
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      ),
    );
  }
}

class _ImageError extends StatelessWidget {
  const _ImageError();

  @override
  Widget build(BuildContext context) {
    return Container(
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: Icon(
        Icons.broken_image_outlined,
        color: Theme.of(context).colorScheme.onSurfaceVariant,
      ),
    );
  }
}
```

## Asset Images

Bundled with the app. Used for logos, illustrations, icons, onboarding images.

```dart
// From assets directory
Image.asset(
  'assets/images/onboarding_1.png',
  width: 200,
  fit: BoxFit.contain,
)

// With asset constants (prevent typos)
class AppAssets {
  AppAssets._();

  static const String logo = 'assets/images/logo.png';
  static const String onboarding1 = 'assets/images/onboarding_1.png';
  static const String onboarding2 = 'assets/images/onboarding_2.png';
  static const String emptyState = 'assets/images/empty_state.svg';
  static const String errorIllustration = 'assets/images/error.svg';
}

// Usage
Image.asset(AppAssets.logo, width: 120)
```

Register in `pubspec.yaml`:
```yaml
flutter:
  assets:
    - assets/images/
```

## Image Picker

Pick images from camera or gallery. Returns a file path.

```dart
class ImagePickerService {
  final ImagePicker _picker = ImagePicker();

  /// Pick a single image from gallery or camera.
  Future<File?> pickImage({
    required ImageSource source,
    int maxWidth = 1920,
    int maxHeight = 1920,
    int quality = 85,
  }) async {
    try {
      final xFile = await _picker.pickImage(
        source: source,
        maxWidth: maxWidth.toDouble(),
        maxHeight: maxHeight.toDouble(),
        imageQuality: quality,
      );

      if (xFile == null) return null;
      return File(xFile.path);
    } catch (e) {
      // Permission denied or camera unavailable
      return null;
    }
  }

  /// Pick multiple images from gallery.
  Future<List<File>> pickMultipleImages({
    int maxWidth = 1920,
    int maxHeight = 1920,
    int quality = 85,
    int? limit,
  }) async {
    try {
      final xFiles = await _picker.pickMultiImage(
        maxWidth: maxWidth.toDouble(),
        maxHeight: maxHeight.toDouble(),
        imageQuality: quality,
        limit: limit,
      );

      return xFiles.map((xFile) => File(xFile.path)).toList();
    } catch (e) {
      return [];
    }
  }
}
```

### Image Source Selection Bottom Sheet

```dart
Future<ImageSource?> showImageSourceSheet(BuildContext context) {
  return showModalBottomSheet<ImageSource>(
    context: context,
    builder: (context) => SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ListTile(
            leading: const Icon(Icons.camera_alt),
            title: Text(context.l10n.takePhoto),
            onTap: () => Navigator.pop(context, ImageSource.camera),
          ),
          ListTile(
            leading: const Icon(Icons.photo_library),
            title: Text(context.l10n.chooseFromGallery),
            onTap: () => Navigator.pop(context, ImageSource.gallery),
          ),
          const SizedBox(height: 8),
        ],
      ),
    ),
  );
}
```

## Image Compression Before Upload

Compress images before sending to API. Reduces upload time and bandwidth.

```dart
class ImageCompressor {
  /// Compress an image file. Returns a new compressed file.
  static Future<File> compress(
    File file, {
    int quality = 80,
    int minWidth = 1024,
    int minHeight = 1024,
  }) async {
    final result = await FlutterImageCompress.compressAndGetFile(
      file.path,
      _targetPath(file.path),
      quality: quality,
      minWidth: minWidth,
      minHeight: minHeight,
      format: CompressFormat.jpeg,
    );

    if (result == null) return file; // Compression failed, return original
    return File(result.path);
  }

  static String _targetPath(String originalPath) {
    final dir = path.dirname(originalPath);
    final ext = path.extension(originalPath);
    final name = path.basenameWithoutExtension(originalPath);
    return '$dir/${name}_compressed$ext';
  }
}
```

## Upload to API (Multipart)

Upload images as multipart/form-data using Dio.

```dart
class ImageUploadService {
  final ApiClient _api;

  ImageUploadService(this._api);

  /// Upload a single image. Returns the URL from the API response.
  Future<String> uploadImage(
    File file, {
    String fieldName = 'image',
    void Function(int sent, int total)? onProgress,
  }) async {
    // Compress before upload
    final compressed = await ImageCompressor.compress(file);

    final formData = FormData.fromMap({
      fieldName: await MultipartFile.fromFile(
        compressed.path,
        filename: path.basename(compressed.path),
        contentType: DioMediaType('image', 'jpeg'),
      ),
    });

    final response = await _api.post(
      '/uploads/image',
      data: formData,
      options: Options(contentType: 'multipart/form-data'),
      onSendProgress: onProgress,
    );

    return response.data['url'] as String;
  }

  /// Upload multiple images. Returns list of URLs.
  Future<List<String>> uploadMultipleImages(
    List<File> files, {
    void Function(int current, int total)? onFileProgress,
  }) async {
    final urls = <String>[];

    for (var i = 0; i < files.length; i++) {
      final url = await uploadImage(files[i]);
      urls.add(url);
      onFileProgress?.call(i + 1, files.length);
    }

    return urls;
  }
}
```

### Upload with Progress Indicator

```dart
class _ImageUploadWidgetState extends State<ImageUploadWidget> {
  double _progress = 0;
  bool _isUploading = false;

  Future<void> _pickAndUpload() async {
    final source = await showImageSourceSheet(context);
    if (source == null) return;

    final file = await ref.read(imagePickerProvider).pickImage(source: source);
    if (file == null) return;

    setState(() {
      _isUploading = true;
      _progress = 0;
    });

    try {
      final url = await ref.read(imageUploadProvider).uploadImage(
        file,
        onProgress: (sent, total) {
          setState(() => _progress = sent / total);
        },
      );

      widget.onUploaded(url);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Upload failed: ${e.toString()}')),
        );
      }
    } finally {
      if (mounted) setState(() => _isUploading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: _isUploading ? null : _pickAndUpload,
      child: Container(
        width: 120,
        height: 120,
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: Theme.of(context).colorScheme.outline),
        ),
        child: _isUploading
            ? Center(
                child: CircularProgressIndicator(value: _progress),
              )
            : const Icon(Icons.add_a_photo, size: 32),
      ),
    );
  }
}
```

## AppAvatar Widget

Circular image with fallback to initials. See `component-design.md` for the full implementation.

```dart
// Quick reference usage:

// With network image
AppAvatar(
  imageUrl: user.avatarUrl,
  name: user.fullName,
  size: 48,
)

// Without image (shows initials)
AppAvatar(
  name: 'John Doe',
  size: 56,
)

// With online indicator badge
AppAvatar(
  imageUrl: user.avatarUrl,
  name: user.fullName,
  badge: Container(
    width: 14,
    height: 14,
    decoration: BoxDecoration(
      color: Colors.green,
      shape: BoxShape.circle,
      border: Border.all(color: Colors.white, width: 2),
    ),
  ),
)
```

## Image Carousel

Swipeable image gallery with indicator dots.

```dart
class AppImageCarousel extends StatefulWidget {
  const AppImageCarousel({
    super.key,
    required this.imageUrls,
    this.height = 250,
    this.borderRadius = 12,
    this.onTap,
  });

  final List<String> imageUrls;
  final double height;
  final double borderRadius;
  final void Function(int index)? onTap;

  @override
  State<AppImageCarousel> createState() => _AppImageCarouselState();
}

class _AppImageCarouselState extends State<AppImageCarousel> {
  final _pageController = PageController();
  int _currentPage = 0;

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (widget.imageUrls.isEmpty) return const SizedBox.shrink();

    if (widget.imageUrls.length == 1) {
      return _buildSingleImage(widget.imageUrls.first);
    }

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        SizedBox(
          height: widget.height,
          child: PageView.builder(
            controller: _pageController,
            itemCount: widget.imageUrls.length,
            onPageChanged: (page) => setState(() => _currentPage = page),
            itemBuilder: (context, index) {
              return GestureDetector(
                onTap: () => widget.onTap?.call(index),
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 4),
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(widget.borderRadius),
                    child: CachedNetworkImage(
                      imageUrl: widget.imageUrls[index],
                      fit: BoxFit.cover,
                      placeholder: (_, __) => const _ImagePlaceholder(),
                      errorWidget: (_, __, ___) => const _ImageError(),
                    ),
                  ),
                ),
              );
            },
          ),
        ),
        const SizedBox(height: 8),
        _PageIndicator(
          count: widget.imageUrls.length,
          current: _currentPage,
        ),
      ],
    );
  }

  Widget _buildSingleImage(String url) {
    return GestureDetector(
      onTap: () => widget.onTap?.call(0),
      child: SizedBox(
        height: widget.height,
        width: double.infinity,
        child: ClipRRect(
          borderRadius: BorderRadius.circular(widget.borderRadius),
          child: CachedNetworkImage(
            imageUrl: url,
            fit: BoxFit.cover,
            placeholder: (_, __) => const _ImagePlaceholder(),
            errorWidget: (_, __, ___) => const _ImageError(),
          ),
        ),
      ),
    );
  }
}

class _PageIndicator extends StatelessWidget {
  const _PageIndicator({required this.count, required this.current});

  final int count;
  final int current;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: List.generate(count, (index) {
        final isActive = index == current;
        return AnimatedContainer(
          duration: const Duration(milliseconds: 200),
          margin: const EdgeInsets.symmetric(horizontal: 3),
          width: isActive ? 20 : 8,
          height: 8,
          decoration: BoxDecoration(
            color: isActive
                ? theme.colorScheme.primary
                : theme.colorScheme.surfaceContainerHighest,
            borderRadius: BorderRadius.circular(4),
          ),
        );
      }),
    );
  }
}
```

**Usage:**

```dart
AppImageCarousel(
  imageUrls: task.photoUrls,
  height: 200,
  onTap: (index) {
    // Open full-screen image viewer
    context.push('/image-viewer', extra: {
      'urls': task.photoUrls,
      'initialIndex': index,
    });
  },
)
```

## Full-Screen Image Viewer

For tapping on an image to view it full-screen with pinch-to-zoom.

```dart
class FullScreenImageViewer extends StatelessWidget {
  const FullScreenImageViewer({
    super.key,
    required this.imageUrls,
    this.initialIndex = 0,
  });

  final List<String> imageUrls;
  final int initialIndex;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        foregroundColor: Colors.white,
        elevation: 0,
      ),
      body: PageView.builder(
        controller: PageController(initialPage: initialIndex),
        itemCount: imageUrls.length,
        itemBuilder: (context, index) {
          return InteractiveViewer(
            minScale: 0.5,
            maxScale: 4.0,
            child: Center(
              child: CachedNetworkImage(
                imageUrl: imageUrls[index],
                fit: BoxFit.contain,
                placeholder: (_, __) => const CircularProgressIndicator(
                  color: Colors.white,
                ),
                errorWidget: (_, __, ___) => const Icon(
                  Icons.broken_image,
                  color: Colors.white54,
                  size: 64,
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}
```

## Key Rules

1. **Always CachedNetworkImage** -- never `Image.network()`.
2. **Always provide placeholder and error widgets** -- no blank spaces.
3. **Compress before upload** -- reduce bandwidth, not quality (quality: 80 is fine).
4. **Multipart/form-data for upload** -- never base64 in JSON body.
5. **Asset constants class** -- prevent typos, single source of truth.
6. **Permission handling** -- camera and gallery permissions must be handled (see `permissions.md`).
7. **mounted check** -- always check before setState after async image operations.
