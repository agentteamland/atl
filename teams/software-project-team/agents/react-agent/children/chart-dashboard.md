---
knowledge-base-summary: "Recharts for data visualization. KPI cards with trend indicators. Dashboard grid layout. Real-time chart updates via WebSocket. Common chart types: line, bar, pie, area. Responsive chart sizing."
---
# Chart & Dashboard: Data Visualization

## Philosophy

Dashboards display data from the API. Charts are visual representations of API responses -- nothing is calculated in the frontend. The API provides aggregated data, the frontend renders it. Real-time updates come through SignalR invalidation (refetch on event), not by pushing chart data through WebSocket.

## Package

```bash
npm install recharts
```

Recharts is composable, React-native, and built on D3. Each chart is a tree of React components -- no imperative API.

## KPI Card Component

The fundamental dashboard building block. Shows a single metric with trend direction.

```typescript
// src/components/dashboard/kpi-card.tsx
import { ArrowUp, ArrowDown, Minus, type LucideIcon } from 'lucide-react';

interface KpiCardProps {
  label: string;
  value: string | number;
  trend?: {
    value: number;       // percentage change
    direction: 'up' | 'down' | 'flat';
  };
  icon?: LucideIcon;
  loading?: boolean;
}

export function KpiCard({ label, value, trend, icon: Icon, loading }: KpiCardProps) {
  if (loading) {
    return <KpiCardSkeleton />;
  }

  const trendConfig = {
    up: { icon: ArrowUp, color: 'text-green-600', bg: 'bg-green-50' },
    down: { icon: ArrowDown, color: 'text-red-600', bg: 'bg-red-50' },
    flat: { icon: Minus, color: 'text-gray-500', bg: 'bg-gray-50' },
  };

  const trendStyle = trend ? trendConfig[trend.direction] : null;
  const TrendIcon = trendStyle?.icon;

  return (
    <div className="rounded-xl border bg-card p-6 shadow-sm">
      <div className="flex items-center justify-between">
        <p className="text-sm font-medium text-muted-foreground">{label}</p>
        {Icon && (
          <div className="rounded-md bg-primary/10 p-2">
            <Icon className="h-4 w-4 text-primary" />
          </div>
        )}
      </div>

      <div className="mt-3 flex items-end gap-3">
        <p className="text-3xl font-bold tracking-tight">{value}</p>

        {trend && trendStyle && TrendIcon && (
          <span
            className={`inline-flex items-center gap-0.5 rounded-full px-2 py-0.5 text-xs font-medium ${trendStyle.color} ${trendStyle.bg}`}
          >
            <TrendIcon className="h-3 w-3" />
            {Math.abs(trend.value)}%
          </span>
        )}
      </div>
    </div>
  );
}

function KpiCardSkeleton() {
  return (
    <div className="rounded-xl border bg-card p-6 shadow-sm animate-pulse">
      <div className="flex items-center justify-between">
        <div className="h-4 w-24 rounded bg-muted" />
        <div className="h-8 w-8 rounded-md bg-muted" />
      </div>
      <div className="mt-3">
        <div className="h-8 w-32 rounded bg-muted" />
      </div>
    </div>
  );
}
```

## Line Chart Component

Reusable line chart wrapper. API returns the data, this component renders it.

```typescript
// src/components/dashboard/line-chart-card.tsx
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';

interface LineChartSeries {
  dataKey: string;
  color: string;
  name: string;
}

interface LineChartCardProps {
  title: string;
  data: Record<string, unknown>[];
  xAxisKey: string;
  series: LineChartSeries[];
  loading?: boolean;
  height?: number;
}

export function LineChartCard({
  title,
  data,
  xAxisKey,
  series,
  loading,
  height = 300,
}: LineChartCardProps) {
  if (loading) {
    return <ChartSkeleton title={title} height={height} />;
  }

  return (
    <div className="rounded-xl border bg-card p-6 shadow-sm">
      <h3 className="mb-4 text-sm font-medium text-muted-foreground">{title}</h3>

      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={data} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
          <XAxis
            dataKey={xAxisKey}
            tick={{ fontSize: 12 }}
            className="text-muted-foreground"
          />
          <YAxis
            tick={{ fontSize: 12 }}
            className="text-muted-foreground"
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'hsl(var(--card))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '8px',
              fontSize: '12px',
            }}
          />
          <Legend />
          {series.map((s) => (
            <Line
              key={s.dataKey}
              type="monotone"
              dataKey={s.dataKey}
              stroke={s.color}
              name={s.name}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
```

## Bar Chart Component

```typescript
// src/components/dashboard/bar-chart-card.tsx
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';

interface BarChartCardProps {
  title: string;
  data: Record<string, unknown>[];
  xAxisKey: string;
  barKey: string;
  barColor?: string;
  loading?: boolean;
  height?: number;
}

export function BarChartCard({
  title,
  data,
  xAxisKey,
  barKey,
  barColor = 'hsl(var(--primary))',
  loading,
  height = 300,
}: BarChartCardProps) {
  if (loading) {
    return <ChartSkeleton title={title} height={height} />;
  }

  return (
    <div className="rounded-xl border bg-card p-6 shadow-sm">
      <h3 className="mb-4 text-sm font-medium text-muted-foreground">{title}</h3>

      <ResponsiveContainer width="100%" height={height}>
        <BarChart data={data} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
          <XAxis dataKey={xAxisKey} tick={{ fontSize: 12 }} />
          <YAxis tick={{ fontSize: 12 }} />
          <Tooltip
            contentStyle={{
              backgroundColor: 'hsl(var(--card))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '8px',
            }}
          />
          <Bar
            dataKey={barKey}
            fill={barColor}
            radius={[4, 4, 0, 0]}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
```

## Pie Chart Component

```typescript
// src/components/dashboard/pie-chart-card.tsx
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts';

interface PieChartItem {
  name: string;
  value: number;
  color: string;
}

interface PieChartCardProps {
  title: string;
  data: PieChartItem[];
  loading?: boolean;
  height?: number;
}

export function PieChartCard({ title, data, loading, height = 300 }: PieChartCardProps) {
  if (loading) {
    return <ChartSkeleton title={title} height={height} />;
  }

  return (
    <div className="rounded-xl border bg-card p-6 shadow-sm">
      <h3 className="mb-4 text-sm font-medium text-muted-foreground">{title}</h3>

      <ResponsiveContainer width="100%" height={height}>
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius={60}
            outerRadius={100}
            paddingAngle={2}
            dataKey="value"
          >
            {data.map((entry, index) => (
              <Cell key={index} fill={entry.color} />
            ))}
          </Pie>
          <Tooltip />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
```

## Chart Skeleton (Shared Loading State)

```typescript
// src/components/dashboard/chart-skeleton.tsx
interface ChartSkeletonProps {
  title: string;
  height: number;
}

export function ChartSkeleton({ title, height }: ChartSkeletonProps) {
  return (
    <div className="rounded-xl border bg-card p-6 shadow-sm">
      <h3 className="mb-4 text-sm font-medium text-muted-foreground">{title}</h3>
      <div
        className="animate-pulse rounded-lg bg-muted"
        style={{ height: `${height}px` }}
      />
    </div>
  );
}
```

## Date Range Picker for Filtering

Dashboard data is filtered by date range. The API receives `from` and `to` parameters.

```typescript
// src/components/dashboard/date-range-filter.tsx
import { useState } from 'react';

interface DateRange {
  from: string; // ISO date string
  to: string;
}

type Preset = '7d' | '30d' | '90d' | 'custom';

const presets: { key: Preset; label: string; days: number }[] = [
  { key: '7d', label: 'Last 7 days', days: 7 },
  { key: '30d', label: 'Last 30 days', days: 30 },
  { key: '90d', label: 'Last 90 days', days: 90 },
];

interface DateRangeFilterProps {
  value: DateRange;
  onChange: (range: DateRange) => void;
}

export function DateRangeFilter({ value, onChange }: DateRangeFilterProps) {
  const [activePreset, setActivePreset] = useState<Preset>('30d');

  const handlePreset = (preset: (typeof presets)[number]) => {
    setActivePreset(preset.key);
    const to = new Date();
    const from = new Date();
    from.setDate(from.getDate() - preset.days);
    onChange({
      from: from.toISOString().split('T')[0],
      to: to.toISOString().split('T')[0],
    });
  };

  return (
    <div className="flex items-center gap-2">
      {presets.map((preset) => (
        <button
          key={preset.key}
          onClick={() => handlePreset(preset)}
          className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
            activePreset === preset.key
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground hover:bg-muted/80'
          }`}
        >
          {preset.label}
        </button>
      ))}

      <div className="ml-2 flex items-center gap-1">
        <input
          type="date"
          value={value.from}
          onChange={(e) => {
            setActivePreset('custom');
            onChange({ ...value, from: e.target.value });
          }}
          className="rounded-md border bg-background px-2 py-1.5 text-sm"
        />
        <span className="text-muted-foreground">-</span>
        <input
          type="date"
          value={value.to}
          onChange={(e) => {
            setActivePreset('custom');
            onChange({ ...value, to: e.target.value });
          }}
          className="rounded-md border bg-background px-2 py-1.5 text-sm"
        />
      </div>
    </div>
  );
}
```

## Dashboard API Hook

```typescript
// src/features/dashboard/hooks/use-dashboard-stats.ts
import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api-client';

interface DashboardStats {
  totalOrders: number;
  totalRevenue: number;
  activeUsers: number;
  conversionRate: number;
  ordersTrend: number;       // percentage change from previous period
  revenueTrend: number;
  revenueByDay: { date: string; revenue: number; orders: number }[];
  ordersByStatus: { name: string; value: number; color: string }[];
  topProducts: { name: string; sales: number }[];
}

export function useDashboardStats(from: string, to: string) {
  return useQuery({
    queryKey: ['dashboard-stats', from, to],
    queryFn: () =>
      api.get<DashboardStats>('/api/dashboard/stats', {
        params: { from, to },
      }).then((r) => r.data),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}
```

## Dashboard Page (Full Example)

```typescript
// src/features/dashboard/pages/dashboard-page.tsx
import { useState } from 'react';
import { ShoppingCart, DollarSign, Users, TrendingUp } from 'lucide-react';
import { KpiCard } from '@/components/dashboard/kpi-card';
import { LineChartCard } from '@/components/dashboard/line-chart-card';
import { BarChartCard } from '@/components/dashboard/bar-chart-card';
import { PieChartCard } from '@/components/dashboard/pie-chart-card';
import { DateRangeFilter } from '@/components/dashboard/date-range-filter';
import { useDashboardStats } from '../hooks/use-dashboard-stats';
import { useRealtimeInvalidation } from '@/hooks/use-realtime-invalidation';

function getDefaultRange() {
  const to = new Date();
  const from = new Date();
  from.setDate(from.getDate() - 30);
  return {
    from: from.toISOString().split('T')[0],
    to: to.toISOString().split('T')[0],
  };
}

export function Component() {
  const [dateRange, setDateRange] = useState(getDefaultRange);
  const { data, isLoading } = useDashboardStats(dateRange.from, dateRange.to);

  // Real-time: refetch dashboard stats when a new order arrives
  useRealtimeInvalidation([
    { event: 'OrderCreated', queryKeys: [['dashboard-stats']] },
    { event: 'OrderStatusChanged', queryKeys: [['dashboard-stats']] },
  ]);

  const trendDirection = (val: number) =>
    val > 0 ? 'up' : val < 0 ? 'down' : 'flat';

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <DateRangeFilter value={dateRange} onChange={setDateRange} />
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label="Total Orders"
          value={data?.totalOrders.toLocaleString() ?? '-'}
          trend={data ? { value: data.ordersTrend, direction: trendDirection(data.ordersTrend) } : undefined}
          icon={ShoppingCart}
          loading={isLoading}
        />
        <KpiCard
          label="Revenue"
          value={data ? `$${data.totalRevenue.toLocaleString()}` : '-'}
          trend={data ? { value: data.revenueTrend, direction: trendDirection(data.revenueTrend) } : undefined}
          icon={DollarSign}
          loading={isLoading}
        />
        <KpiCard
          label="Active Users"
          value={data?.activeUsers.toLocaleString() ?? '-'}
          icon={Users}
          loading={isLoading}
        />
        <KpiCard
          label="Conversion Rate"
          value={data ? `${data.conversionRate}%` : '-'}
          icon={TrendingUp}
          loading={isLoading}
        />
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Revenue over time - takes 2 columns */}
        <div className="lg:col-span-2">
          <LineChartCard
            title="Revenue & Orders Over Time"
            data={data?.revenueByDay ?? []}
            xAxisKey="date"
            series={[
              { dataKey: 'revenue', color: 'hsl(var(--primary))', name: 'Revenue' },
              { dataKey: 'orders', color: '#f97316', name: 'Orders' },
            ]}
            loading={isLoading}
          />
        </div>

        {/* Order status distribution */}
        <PieChartCard
          title="Orders by Status"
          data={data?.ordersByStatus ?? []}
          loading={isLoading}
        />
      </div>

      {/* Top Products */}
      <BarChartCard
        title="Top Products by Sales"
        data={data?.topProducts ?? []}
        xAxisKey="name"
        barKey="sales"
        loading={isLoading}
        height={250}
      />
    </div>
  );
}
```

## Responsive Dashboard Grid

The dashboard uses CSS grid with `auto-fit` for responsive behavior. No media query breakpoints needed for the grid itself -- Tailwind's responsive prefixes handle column counts.

```css
/* Pattern: grid-cols-1 on mobile, 2 on tablet, 4 on desktop */
grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4

/* Charts: 1 column mobile, 3 columns desktop with span-2 for wide charts */
grid grid-cols-1 gap-4 lg:grid-cols-3
```

## Rules

1. **API provides aggregated data.** Never calculate sums, averages, or percentages in the frontend. The API returns ready-to-display values.
2. **Always wrap charts in `ResponsiveContainer`.** This makes charts resize with their parent container. Never use fixed width.
3. **Loading skeleton for every chart.** Never show an empty chart area. Show a pulsing placeholder while data loads.
4. **Date range is a query parameter.** Changing the date range changes the React Query key, which triggers a refetch.
5. **Real-time via invalidation.** When a SignalR event arrives (e.g., new order), invalidate the dashboard query. Don't push chart data through WebSocket.
6. **KPI cards are typed.** The trend direction (up/down/flat) and value are computed from the API response, not hardcoded.
