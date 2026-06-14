---
knowledge-base-summary: "Drag & drop zone (react-dropzone). Progress bar for large files. File preview (image thumbnail, file icon). Multi-file upload. Size/type validation (UX-only). Upload to API via FormData."
---
# File Upload

## Philosophy

React is a BRIDGE. File upload collects files from the user and sends them to the API. Validation in React is UX-only (show a warning before uploading) -- the API does the real validation. react-dropzone handles the drag-and-drop zone. Axios provides upload progress tracking. Image files get a preview via `URL.createObjectURL`. Multi-file uploads are queued and uploaded one by one (or in parallel with a concurrency limit).

## Dependencies

```json
{
  "react-dropzone": "^14.x",
  "axios": "^1.x",
  "@tanstack/react-query": "^5.x"
}
```

## File Upload Hook

```typescript
// hooks/use-file-upload.ts
import { useState, useCallback } from 'react';
import { api } from '@/lib/api-client';

interface UploadableFile {
  id: string;
  file: File;
  preview?: string;
  progress: number;
  status: 'pending' | 'uploading' | 'success' | 'error';
  error?: string;
}

interface UseFileUploadOptions {
  endpoint: string;
  maxFileSize?: number;       // bytes, default 10MB
  acceptedTypes?: string[];   // MIME types
  maxFiles?: number;
  onSuccess?: (file: UploadableFile, response: unknown) => void;
  onError?: (file: UploadableFile, error: Error) => void;
}

export function useFileUpload({
  endpoint,
  maxFileSize = 10 * 1024 * 1024,
  acceptedTypes,
  maxFiles = 10,
  onSuccess,
  onError,
}: UseFileUploadOptions) {
  const [files, setFiles] = useState<UploadableFile[]>([]);

  // Add files (from dropzone or file input)
  const addFiles = useCallback(
    (newFiles: File[]) => {
      const uploadableFiles: UploadableFile[] = newFiles
        .slice(0, maxFiles - files.length)
        .map((file) => ({
          id: `${file.name}-${Date.now()}-${Math.random().toString(36).slice(2)}`,
          file,
          preview: file.type.startsWith('image/')
            ? URL.createObjectURL(file)
            : undefined,
          progress: 0,
          status: 'pending' as const,
        }));

      setFiles((prev) => [...prev, ...uploadableFiles]);
      return uploadableFiles;
    },
    [files.length, maxFiles]
  );

  // Validate a file (UX-only)
  const validateFile = useCallback(
    (file: File): string | null => {
      if (maxFileSize && file.size > maxFileSize) {
        return `File size exceeds ${formatBytes(maxFileSize)}`;
      }
      if (acceptedTypes && !acceptedTypes.includes(file.type)) {
        return `File type ${file.type} is not accepted`;
      }
      return null;
    },
    [maxFileSize, acceptedTypes]
  );

  // Upload a single file
  const uploadFile = useCallback(
    async (uploadableFile: UploadableFile) => {
      setFiles((prev) =>
        prev.map((f) =>
          f.id === uploadableFile.id ? { ...f, status: 'uploading', progress: 0 } : f
        )
      );

      const formData = new FormData();
      formData.append('file', uploadableFile.file);

      try {
        const response = await api.post(endpoint, formData, {
          headers: { 'Content-Type': 'multipart/form-data' },
          onUploadProgress: (event) => {
            const progress = event.total
              ? Math.round((event.loaded / event.total) * 100)
              : 0;

            setFiles((prev) =>
              prev.map((f) =>
                f.id === uploadableFile.id ? { ...f, progress } : f
              )
            );
          },
        });

        setFiles((prev) =>
          prev.map((f) =>
            f.id === uploadableFile.id
              ? { ...f, status: 'success', progress: 100 }
              : f
          )
        );

        onSuccess?.(uploadableFile, response.data);
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Upload failed');

        setFiles((prev) =>
          prev.map((f) =>
            f.id === uploadableFile.id
              ? { ...f, status: 'error', error: error.message }
              : f
          )
        );

        onError?.(uploadableFile, error);
      }
    },
    [endpoint, onSuccess, onError]
  );

  // Upload all pending files
  const uploadAll = useCallback(async () => {
    const pending = files.filter((f) => f.status === 'pending');
    // Upload sequentially to avoid overwhelming the server
    for (const file of pending) {
      await uploadFile(file);
    }
  }, [files, uploadFile]);

  // Remove a file from the list
  const removeFile = useCallback((id: string) => {
    setFiles((prev) => {
      const file = prev.find((f) => f.id === id);
      if (file?.preview) {
        URL.revokeObjectURL(file.preview);
      }
      return prev.filter((f) => f.id !== id);
    });
  }, []);

  // Clear all files
  const clearFiles = useCallback(() => {
    files.forEach((f) => {
      if (f.preview) URL.revokeObjectURL(f.preview);
    });
    setFiles([]);
  }, [files]);

  return {
    files,
    addFiles,
    validateFile,
    uploadFile,
    uploadAll,
    removeFile,
    clearFiles,
    hasFiles: files.length > 0,
    isUploading: files.some((f) => f.status === 'uploading'),
    allComplete: files.length > 0 && files.every((f) => f.status === 'success'),
  };
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}
```

## Complete FileUpload Component

```typescript
// components/ui/file-upload.tsx
import { useCallback } from 'react';
import { useDropzone } from 'react-dropzone';
import { Upload, X, FileText, Image, CheckCircle2, AlertCircle, Loader2 } from 'lucide-react';
import clsx from 'clsx';
import { useFileUpload } from '@/hooks/use-file-upload';
import { toast } from '@/lib/toast';

interface FileUploadProps {
  endpoint: string;
  accept?: Record<string, string[]>;  // react-dropzone accept format
  maxFileSize?: number;
  maxFiles?: number;
  onAllComplete?: () => void;
}

export function FileUpload({
  endpoint,
  accept,
  maxFileSize = 10 * 1024 * 1024,
  maxFiles = 10,
  onAllComplete,
}: FileUploadProps) {
  const {
    files,
    addFiles,
    validateFile,
    uploadAll,
    removeFile,
    isUploading,
    allComplete,
  } = useFileUpload({
    endpoint,
    maxFileSize,
    maxFiles,
    onSuccess: () => {},
    onError: (file, error) => {
      toast.error(`Failed to upload ${file.file.name}`);
    },
  });

  const onDrop = useCallback(
    (acceptedFiles: File[]) => {
      const validFiles: File[] = [];

      for (const file of acceptedFiles) {
        const error = validateFile(file);
        if (error) {
          toast.error(`${file.name}: ${error}`);
        } else {
          validFiles.push(file);
        }
      }

      if (validFiles.length > 0) {
        addFiles(validFiles);
      }
    },
    [addFiles, validateFile]
  );

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept,
    maxFiles: maxFiles - files.length,
    disabled: isUploading,
  });

  return (
    <div className="space-y-4">
      {/* Drop Zone */}
      <div
        {...getRootProps()}
        className={clsx(
          'flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 transition-colors cursor-pointer',
          isDragActive
            ? 'border-blue-500 bg-blue-50'
            : 'border-gray-300 hover:border-gray-400 hover:bg-gray-50',
          isUploading && 'opacity-50 cursor-not-allowed'
        )}
      >
        <input {...getInputProps()} />
        <Upload className="h-8 w-8 text-gray-400 mb-3" />
        {isDragActive ? (
          <p className="text-sm text-blue-600 font-medium">Drop files here</p>
        ) : (
          <>
            <p className="text-sm text-gray-600">
              <span className="font-medium text-blue-600">Click to upload</span> or drag and drop
            </p>
            <p className="mt-1 text-xs text-gray-400">
              Max {formatBytes(maxFileSize)} per file, up to {maxFiles} files
            </p>
          </>
        )}
      </div>

      {/* File List */}
      {files.length > 0 && (
        <div className="space-y-2">
          {files.map((file) => (
            <FileItem
              key={file.id}
              file={file}
              onRemove={() => removeFile(file.id)}
            />
          ))}

          {/* Upload Button */}
          <div className="flex justify-end gap-2 pt-2">
            <button
              onClick={uploadAll}
              disabled={isUploading || allComplete}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:opacity-50 inline-flex items-center gap-2"
            >
              {isUploading ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" /> Uploading...
                </>
              ) : allComplete ? (
                <>
                  <CheckCircle2 className="h-4 w-4" /> All uploaded
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4" /> Upload {files.filter((f) => f.status === 'pending').length} files
                </>
              )}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// --- File Item ---
function FileItem({
  file,
  onRemove,
}: {
  file: { id: string; file: File; preview?: string; progress: number; status: string; error?: string };
  onRemove: () => void;
}) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-gray-200 p-3">
      {/* Preview or icon */}
      {file.preview ? (
        <img
          src={file.preview}
          alt={file.file.name}
          className="h-12 w-12 rounded-md object-cover"
        />
      ) : (
        <div className="flex h-12 w-12 items-center justify-center rounded-md bg-gray-100">
          <FileText className="h-6 w-6 text-gray-400" />
        </div>
      )}

      {/* File info + progress */}
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-gray-900 truncate">{file.file.name}</p>
        <p className="text-xs text-gray-500">{formatBytes(file.file.size)}</p>

        {/* Progress bar */}
        {file.status === 'uploading' && (
          <div className="mt-1 h-1.5 w-full rounded-full bg-gray-200">
            <div
              className="h-1.5 rounded-full bg-blue-600 transition-all duration-300"
              style={{ width: `${file.progress}%` }}
            />
          </div>
        )}

        {/* Error message */}
        {file.status === 'error' && (
          <p className="mt-1 text-xs text-red-500">{file.error}</p>
        )}
      </div>

      {/* Status icon */}
      <div className="shrink-0">
        {file.status === 'success' && <CheckCircle2 className="h-5 w-5 text-green-500" />}
        {file.status === 'error' && <AlertCircle className="h-5 w-5 text-red-500" />}
        {file.status === 'uploading' && (
          <span className="text-xs text-gray-500">{file.progress}%</span>
        )}
      </div>

      {/* Remove button */}
      {file.status !== 'uploading' && (
        <button
          onClick={onRemove}
          className="shrink-0 rounded-md p-1 text-gray-400 hover:text-gray-600 hover:bg-gray-100"
          aria-label={`Remove ${file.file.name}`}
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}
```

## Usage Examples

```typescript
// Basic usage -- accept images only
<FileUpload
  endpoint="/api/uploads/images"
  accept={{ 'image/*': ['.png', '.jpg', '.jpeg', '.webp'] }}
  maxFileSize={5 * 1024 * 1024}
  maxFiles={5}
/>

// Documents
<FileUpload
  endpoint="/api/uploads/documents"
  accept={{
    'application/pdf': ['.pdf'],
    'application/msword': ['.doc', '.docx'],
  }}
  maxFileSize={20 * 1024 * 1024}
  maxFiles={3}
/>

// Avatar upload (single file)
<FileUpload
  endpoint="/api/users/me/avatar"
  accept={{ 'image/*': ['.png', '.jpg', '.jpeg'] }}
  maxFileSize={2 * 1024 * 1024}
  maxFiles={1}
  onAllComplete={() => {
    queryClient.invalidateQueries({ queryKey: ['user', 'me'] });
    toast.success('Avatar updated');
  }}
/>
```

## Single Image Upload with Preview

For cases like avatar or cover photo where you need a compact single-file upload.

```typescript
// components/ui/image-upload.tsx
import { useCallback, useState } from 'react';
import { useDropzone } from 'react-dropzone';
import { Camera, X, Loader2 } from 'lucide-react';
import { api } from '@/lib/api-client';
import { toast } from '@/lib/toast';

interface ImageUploadProps {
  currentImage?: string;
  endpoint: string;
  onSuccess: (url: string) => void;
  size?: 'sm' | 'md' | 'lg';
}

const sizeMap = {
  sm: 'h-16 w-16',
  md: 'h-24 w-24',
  lg: 'h-32 w-32',
};

export function ImageUpload({
  currentImage,
  endpoint,
  onSuccess,
  size = 'md',
}: ImageUploadProps) {
  const [preview, setPreview] = useState<string | null>(null);
  const [isUploading, setIsUploading] = useState(false);

  const onDrop = useCallback(
    async (acceptedFiles: File[]) => {
      const file = acceptedFiles[0];
      if (!file) return;

      // Show preview immediately
      const previewUrl = URL.createObjectURL(file);
      setPreview(previewUrl);
      setIsUploading(true);

      try {
        const formData = new FormData();
        formData.append('file', file);
        const response = await api.post(endpoint, formData, {
          headers: { 'Content-Type': 'multipart/form-data' },
        });
        onSuccess(response.data.url);
        toast.success('Image uploaded');
      } catch {
        toast.error('Failed to upload image');
        setPreview(null);
      } finally {
        setIsUploading(false);
        URL.revokeObjectURL(previewUrl);
      }
    },
    [endpoint, onSuccess]
  );

  const { getRootProps, getInputProps } = useDropzone({
    onDrop,
    accept: { 'image/*': ['.png', '.jpg', '.jpeg', '.webp'] },
    maxFiles: 1,
    maxSize: 5 * 1024 * 1024,
    disabled: isUploading,
  });

  const displayImage = preview ?? currentImage;

  return (
    <div
      {...getRootProps()}
      className={clsx(
        'relative rounded-full overflow-hidden cursor-pointer group',
        sizeMap[size]
      )}
    >
      <input {...getInputProps()} />

      {displayImage ? (
        <img src={displayImage} alt="Upload" className="h-full w-full object-cover" />
      ) : (
        <div className="h-full w-full bg-gray-200 flex items-center justify-center">
          <Camera className="h-6 w-6 text-gray-400" />
        </div>
      )}

      {/* Hover overlay */}
      <div className="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity">
        {isUploading ? (
          <Loader2 className="h-6 w-6 text-white animate-spin" />
        ) : (
          <Camera className="h-6 w-6 text-white" />
        )}
      </div>
    </div>
  );
}
```

## Key Rules

1. **UX-only validation** -- file size and type checks in React are hints for the user. The API performs the real validation and may reject files that pass client checks.
2. **FormData for upload** -- always use `multipart/form-data` content type. Append the file with `formData.append('file', file)`.
3. **Axios onUploadProgress** -- provides real-time progress percentage. Update state on each progress event.
4. **URL.createObjectURL for preview** -- creates a temporary URL for image preview. Must call `URL.revokeObjectURL` on cleanup to prevent memory leaks.
5. **Sequential uploads** -- upload files one by one (or with a concurrency limit). Avoid uploading 10 files simultaneously.
6. **Remove button during pending/error** -- users can remove files that haven't started or failed. Disable remove while uploading.
7. **react-dropzone for drag & drop** -- provides `getRootProps`, `getInputProps`, `isDragActive`. Handles both click and drag interactions.
8. **Progress bar in file item** -- thin bar below filename, filled with blue based on progress percentage.
9. **Status tracking per file** -- each file has its own status: pending, uploading, success, error. Never batch statuses.
10. **Toast on error** -- file upload failure shows a toast. The file item also shows an error state with the message.
