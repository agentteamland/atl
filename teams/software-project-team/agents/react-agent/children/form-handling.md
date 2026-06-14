---
knowledge-base-summary: "React Hook Form + Zod schema validation. Field components with error display. Multi-step forms. Async submission with loading state. Server-side error mapping. Form auto-save with debounce."
---
# Form Handling

React Hook Form for form state. Zod for validation. Typed end-to-end.

## Core Setup

```tsx
// Dependencies:
// npm install react-hook-form @hookform/resolvers zod
```

## Zod Schema for Form Validation

Define the schema first. The schema IS the single source of truth for both validation and TypeScript types:

```tsx
// src/features/users/types/users.schemas.ts

import { z } from 'zod';

export const createUserSchema = z.object({
  name: z
    .string()
    .min(2, 'Name must be at least 2 characters')
    .max(100, 'Name must be at most 100 characters'),

  email: z
    .string()
    .min(1, 'Email is required')
    .email('Invalid email address'),

  password: z
    .string()
    .min(8, 'Password must be at least 8 characters')
    .regex(/[A-Z]/, 'Password must contain at least one uppercase letter')
    .regex(/[0-9]/, 'Password must contain at least one number'),

  confirmPassword: z.string(),

  role: z.enum(['admin', 'member', 'viewer'], {
    errorMap: () => ({ message: 'Please select a role' }),
  }),

  isActive: z.boolean().default(true),

  bio: z.string().max(500).optional(),
}).refine((data) => data.password === data.confirmPassword, {
  message: 'Passwords do not match',
  path: ['confirmPassword'],
});

// Derive TypeScript type from schema
export type CreateUserForm = z.infer<typeof createUserSchema>;

// Update schema (all fields optional except what's required)
export const updateUserSchema = createUserSchema
  .omit({ password: true, confirmPassword: true })
  .partial()
  .required({ name: true, email: true });

export type UpdateUserForm = z.infer<typeof updateUserSchema>;
```

## useForm with zodResolver

```tsx
// src/features/users/pages/CreateUserPage.tsx

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useCreateUser } from '../api/users.api';
import { createUserSchema, type CreateUserForm } from '../types/users.schemas';
import { parseApiError } from '@/lib/api-error';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/Card';
import { toast } from 'sonner';

export function CreateUserPage() {
  const { t } = useTranslation('users');
  const navigate = useNavigate();
  const { mutate: createUser, isPending } = useCreateUser();

  const form = useForm<CreateUserForm>({
    resolver: zodResolver(createUserSchema),
    defaultValues: {
      name: '',
      email: '',
      password: '',
      confirmPassword: '',
      role: 'member',
      isActive: true,
      bio: '',
    },
  });

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = form;

  function onSubmit(data: CreateUserForm) {
    createUser(
      {
        name: data.name,
        email: data.email,
        password: data.password,
        role: data.role,
      },
      {
        onSuccess: (user) => {
          toast.success(t('created'));
          navigate(`/users/${user.id}`);
        },
        onError: (error) => {
          const apiError = parseApiError(error);

          // Map server validation errors to form fields
          if (apiError.errors) {
            Object.entries(apiError.errors).forEach(([field, messages]) => {
              setError(field as keyof CreateUserForm, {
                type: 'server',
                message: messages[0],
              });
            });
            return;
          }

          // Specific field errors
          if (apiError.statusCode === 409) {
            setError('email', {
              type: 'server',
              message: t('emailAlreadyExists'),
            });
            return;
          }

          toast.error(apiError.message);
        },
      }
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="mb-6 text-2xl font-bold text-gray-900 dark:text-white">
        {t('createUser')}
      </h1>

      <form onSubmit={handleSubmit(onSubmit)} noValidate>
        <Card>
          <CardContent className="space-y-4">
            {/* Name */}
            <Input
              label={t('fields.name')}
              error={errors.name?.message}
              {...register('name')}
              autoFocus
            />

            {/* Email */}
            <Input
              label={t('fields.email')}
              type="email"
              error={errors.email?.message}
              {...register('email')}
            />

            {/* Password */}
            <Input
              label={t('fields.password')}
              type="password"
              error={errors.password?.message}
              hint={t('fields.passwordHint')}
              {...register('password')}
            />

            {/* Confirm Password */}
            <Input
              label={t('fields.confirmPassword')}
              type="password"
              error={errors.confirmPassword?.message}
              {...register('confirmPassword')}
            />

            {/* Role (Select with Controller) */}
            <Select
              label={t('fields.role')}
              error={errors.role?.message}
              {...register('role')}
              options={[
                { value: 'admin', label: t('roles.admin') },
                { value: 'member', label: t('roles.member') },
                { value: 'viewer', label: t('roles.viewer') },
              ]}
            />

            {/* Bio (optional) */}
            <div className="space-y-1.5">
              <label
                htmlFor="bio"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                {t('fields.bio')}
              </label>
              <textarea
                id="bio"
                className="block w-full rounded-md border border-gray-300 p-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-white"
                rows={3}
                {...register('bio')}
              />
              {errors.bio && (
                <p className="text-sm text-red-600">{errors.bio.message}</p>
              )}
            </div>
          </CardContent>

          <CardFooter className="flex justify-end gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={() => navigate(-1)}
            >
              {t('cancel')}
            </Button>
            <Button type="submit" isLoading={isPending}>
              {t('create')}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </div>
  );
}
```

## Field Components with register()

For simple inputs, `register()` works directly:

```tsx
// register() returns { name, ref, onChange, onBlur }
<Input
  label="Email"
  type="email"
  error={errors.email?.message}
  {...register('email')}
/>
```

## Controlled Components with Controller

For complex inputs (date pickers, rich text, custom selects), use `Controller`:

```tsx
import { Controller, useForm } from 'react-hook-form';
import { DatePicker } from '@/components/ui/DatePicker';

function EventForm() {
  const { control, handleSubmit, formState: { errors } } = useForm<EventForm>({
    resolver: zodResolver(eventSchema),
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      {/* Controlled DatePicker */}
      <Controller
        name="startDate"
        control={control}
        render={({ field }) => (
          <DatePicker
            label="Start Date"
            value={field.value}
            onChange={field.onChange}
            onBlur={field.onBlur}
            error={errors.startDate?.message}
          />
        )}
      />

      {/* Controlled Toggle */}
      <Controller
        name="isPublic"
        control={control}
        render={({ field }) => (
          <Toggle
            label="Public Event"
            checked={field.value}
            onChange={field.onChange}
          />
        )}
      />
    </form>
  );
}
```

## Server Error Mapping

When the API returns field-level errors, map them to the form:

```tsx
// API response: { errors: { email: ["Email already taken"], name: ["Too short"] } }

function onError(error: unknown) {
  const apiError = parseApiError(error);

  if (apiError.errors) {
    // Map each server error to its form field
    Object.entries(apiError.errors).forEach(([field, messages]) => {
      form.setError(field as keyof FormValues, {
        type: 'server',
        message: messages[0],
      });
    });
  } else {
    // Generic error — show toast
    toast.error(apiError.message);
  }
}
```

## Multi-Step Form Pattern

```tsx
// src/features/onboarding/pages/OnboardingPage.tsx

import { useState } from 'react';
import { useForm, FormProvider } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { onboardingSchema, type OnboardingForm } from '../types/onboarding.schemas';

const STEPS = [
  { id: 'account', title: 'Account', component: AccountStep },
  { id: 'profile', title: 'Profile', component: ProfileStep },
  { id: 'preferences', title: 'Preferences', component: PreferencesStep },
] as const;

export function OnboardingPage() {
  const [currentStep, setCurrentStep] = useState(0);

  const methods = useForm<OnboardingForm>({
    resolver: zodResolver(onboardingSchema),
    mode: 'onTouched', // Validate on blur for multi-step
    defaultValues: {
      // Account
      email: '',
      password: '',
      // Profile
      name: '',
      company: '',
      // Preferences
      language: 'en',
      notifications: true,
    },
  });

  const CurrentStepComponent = STEPS[currentStep].component;

  async function handleNext() {
    // Validate only current step's fields
    const fieldsToValidate = getStepFields(currentStep);
    const isValid = await methods.trigger(fieldsToValidate);

    if (isValid) {
      setCurrentStep((prev) => Math.min(prev + 1, STEPS.length - 1));
    }
  }

  function handleBack() {
    setCurrentStep((prev) => Math.max(prev - 1, 0));
  }

  function onSubmit(data: OnboardingForm) {
    // Final submission — all steps validated
    console.log('Complete onboarding:', data);
  }

  return (
    <div className="mx-auto max-w-xl">
      {/* Step indicator */}
      <StepIndicator steps={STEPS} currentStep={currentStep} />

      {/* Form wraps all steps */}
      <FormProvider {...methods}>
        <form onSubmit={methods.handleSubmit(onSubmit)}>
          <CurrentStepComponent />

          <div className="mt-6 flex justify-between">
            <Button
              type="button"
              variant="outline"
              onClick={handleBack}
              disabled={currentStep === 0}
            >
              Back
            </Button>

            {currentStep < STEPS.length - 1 ? (
              <Button type="button" onClick={handleNext}>
                Next
              </Button>
            ) : (
              <Button type="submit">
                Complete
              </Button>
            )}
          </div>
        </form>
      </FormProvider>
    </div>
  );
}

// Helper: which fields to validate per step
function getStepFields(step: number): (keyof OnboardingForm)[] {
  switch (step) {
    case 0: return ['email', 'password'];
    case 1: return ['name', 'company'];
    case 2: return ['language', 'notifications'];
    default: return [];
  }
}
```

### Step Component (uses FormProvider context)

```tsx
// src/features/onboarding/components/AccountStep.tsx

import { useFormContext } from 'react-hook-form';
import { Input } from '@/components/ui/Input';
import type { OnboardingForm } from '../types/onboarding.schemas';

export function AccountStep() {
  const { register, formState: { errors } } = useFormContext<OnboardingForm>();

  return (
    <div className="space-y-4">
      <Input
        label="Email"
        type="email"
        error={errors.email?.message}
        {...register('email')}
      />
      <Input
        label="Password"
        type="password"
        error={errors.password?.message}
        {...register('password')}
      />
    </div>
  );
}
```

## Select Component Pattern

```tsx
// src/components/ui/Select.tsx

import { forwardRef } from 'react';
import { cn } from '@/lib/cn';

interface SelectOption {
  value: string;
  label: string;
}

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  error?: string;
  options: SelectOption[];
  placeholder?: string;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ label, error, options, placeholder, className, id, ...props }, ref) => {
    const selectId = id || props.name;

    return (
      <div className="space-y-1.5">
        {label && (
          <label htmlFor={selectId} className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {label}
          </label>
        )}
        <select
          ref={ref}
          id={selectId}
          className={cn(
            'block w-full rounded-md border bg-white px-3 py-2 text-sm shadow-sm',
            'focus:outline-none focus:ring-2',
            'dark:bg-gray-900 dark:text-white',
            error
              ? 'border-red-500 focus:ring-red-200'
              : 'border-gray-300 focus:ring-primary-200 dark:border-gray-600',
            className
          )}
          aria-invalid={error ? 'true' : undefined}
          {...props}
        >
          {placeholder && (
            <option value="" disabled>
              {placeholder}
            </option>
          )}
          {options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        {error && (
          <p className="text-sm text-red-600 dark:text-red-400" role="alert">
            {error}
          </p>
        )}
      </div>
    );
  }
);

Select.displayName = 'Select';
```

## Form Patterns Summary

| Pattern | When |
|---------|------|
| `register()` | Simple inputs (text, email, password, textarea) |
| `Controller` | Complex inputs (date picker, rich text, custom toggle) |
| `FormProvider` + `useFormContext` | Multi-step forms, deeply nested field components |
| `setError(field, { message })` | Mapping server errors to specific fields |
| `form.trigger(fields)` | Validate specific fields (e.g., per step in multi-step) |
| `form.reset()` | Reset form after successful submission |
| `form.watch(field)` | React to field value changes (conditional rendering) |

## Rules

1. **Zod schema is the source of truth** — derive TypeScript types with `z.infer`
2. **Always use `zodResolver`** — no manual validation logic in components
3. **`noValidate` on form element** — prevent browser's built-in validation
4. **Server errors mapped to fields** with `setError()` — not just toast
5. **Multi-step: validate per step** with `trigger()` — don't wait until submit
6. **`defaultValues` always defined** — prevents uncontrolled-to-controlled warnings
7. **Loading state on submit button** — `isLoading={isPending}` from mutation
8. **Form validation is UX only** — real validation happens in the API
