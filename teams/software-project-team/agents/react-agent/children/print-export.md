---
knowledge-base-summary: "PDF generation (react-pdf or html2pdf). CSV export from table data. Print-friendly view with @media print. Invoice/report template. Download trigger pattern."
---
# Print & Export: CSV, PDF, Print View

## Philosophy

Export is a client-side transformation of data already fetched from the API. We take the API response (React Query cache) and transform it into a downloadable format. No separate "export endpoint" on the API -- we use the same data the user sees on screen. For large datasets, the API may provide a dedicated export endpoint that returns a file stream.

## CSV Export

The most common export. Transform an array of objects into a CSV string, create a Blob, and trigger download.

```typescript
// src/lib/export-csv.ts

interface CsvColumn<T> {
  header: string;
  accessor: (row: T) => string | number;
}

/**
 * Export an array of data to CSV and trigger download.
 *
 * Usage:
 *   exportToCsv('orders-2024', orders, [
 *     { header: 'Order ID', accessor: (r) => r.id },
 *     { header: 'Customer', accessor: (r) => r.customerName },
 *     { header: 'Total', accessor: (r) => r.total },
 *     { header: 'Status', accessor: (r) => r.status },
 *   ]);
 */
export function exportToCsv<T>(
  filename: string,
  data: T[],
  columns: CsvColumn<T>[]
) {
  // Header row
  const header = columns.map((col) => escapeCsvField(col.header)).join(',');

  // Data rows
  const rows = data.map((row) =>
    columns.map((col) => escapeCsvField(String(col.accessor(row)))).join(',')
  );

  const csvContent = [header, ...rows].join('\n');

  // BOM for Excel UTF-8 compatibility
  const bom = '\uFEFF';
  const blob = new Blob([bom + csvContent], { type: 'text/csv;charset=utf-8;' });

  downloadBlob(blob, `${filename}.csv`);
}

function escapeCsvField(field: string): string {
  if (field.includes(',') || field.includes('"') || field.includes('\n')) {
    return `"${field.replace(/"/g, '""')}"`;
  }
  return field;
}
```

## Download Trigger (Shared Utility)

Creates a temporary link element, clicks it, then cleans up. Works for any Blob.

```typescript
// src/lib/download.ts

/**
 * Trigger a file download from a Blob.
 * Creates a temporary <a> element, clicks it, then revokes the URL.
 */
export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.style.display = 'none';
  document.body.appendChild(link);
  link.click();

  // Cleanup
  setTimeout(() => {
    URL.revokeObjectURL(url);
    document.body.removeChild(link);
  }, 100);
}

/**
 * Download a file from a URL (e.g., API-generated file).
 * For large exports where the API returns a file stream.
 */
export async function downloadFromUrl(url: string, filename: string) {
  const response = await fetch(url);
  const blob = await response.blob();
  downloadBlob(blob, filename);
}
```

## PDF Generation

Two approaches depending on complexity:

### Approach 1: html2canvas + jsPDF (Simple -- screenshot-based)

Best for: printing what the user sees on screen. Quick and simple.

```bash
npm install html2canvas jspdf
```

```typescript
// src/lib/export-pdf-simple.ts
import html2canvas from 'html2canvas';
import { jsPDF } from 'jspdf';

/**
 * Capture a DOM element as PDF.
 * The element is rendered as an image and placed on a PDF page.
 *
 * Usage:
 *   const elementRef = useRef<HTMLDivElement>(null);
 *   exportElementToPdf(elementRef.current, 'invoice-123');
 */
export async function exportElementToPdf(
  element: HTMLElement,
  filename: string,
  orientation: 'portrait' | 'landscape' = 'portrait'
) {
  const canvas = await html2canvas(element, {
    scale: 2,             // higher resolution
    useCORS: true,        // allow cross-origin images
    logging: false,
    backgroundColor: '#ffffff',
  });

  const imgData = canvas.toDataURL('image/png');
  const pdf = new jsPDF(orientation, 'mm', 'a4');

  const pdfWidth = pdf.internal.pageSize.getWidth();
  const pdfHeight = (canvas.height * pdfWidth) / canvas.width;

  pdf.addImage(imgData, 'PNG', 0, 0, pdfWidth, pdfHeight);
  pdf.save(`${filename}.pdf`);
}
```

### Approach 2: @react-pdf/renderer (Complex -- programmatic layout)

Best for: invoices, reports, documents with precise layout control. Renders React components directly to PDF.

```bash
npm install @react-pdf/renderer
```

```typescript
// src/components/export/invoice-pdf.tsx
import {
  Document,
  Page,
  View,
  Text,
  StyleSheet,
  Font,
  pdf,
} from '@react-pdf/renderer';
import { downloadBlob } from '@/lib/download';

// Register a custom font (optional)
Font.register({
  family: 'Inter',
  src: '/fonts/Inter-Regular.ttf',
});

const styles = StyleSheet.create({
  page: {
    padding: 40,
    fontSize: 10,
    fontFamily: 'Inter',
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 30,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
  },
  subtitle: {
    fontSize: 10,
    color: '#6b7280',
    marginTop: 4,
  },
  table: {
    marginTop: 20,
  },
  tableHeader: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: '#e5e7eb',
    paddingBottom: 8,
    marginBottom: 8,
  },
  tableRow: {
    flexDirection: 'row',
    paddingVertical: 6,
    borderBottomWidth: 1,
    borderBottomColor: '#f3f4f6',
  },
  colProduct: { width: '40%' },
  colQty: { width: '15%', textAlign: 'right' },
  colPrice: { width: '20%', textAlign: 'right' },
  colTotal: { width: '25%', textAlign: 'right' },
  totalsSection: {
    marginTop: 20,
    alignItems: 'flex-end',
  },
  totalRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    width: 200,
    paddingVertical: 4,
  },
  totalLabel: {
    fontWeight: 'bold',
  },
  footer: {
    position: 'absolute',
    bottom: 30,
    left: 40,
    right: 40,
    textAlign: 'center',
    color: '#9ca3af',
    fontSize: 8,
  },
});

interface InvoiceData {
  invoiceNumber: string;
  date: string;
  customerName: string;
  items: { product: string; quantity: number; price: number }[];
  subtotal: number;
  tax: number;
  total: number;
}

function InvoicePdfDocument({ data }: { data: InvoiceData }) {
  return (
    <Document>
      <Page size="A4" style={styles.page}>
        {/* Header */}
        <View style={styles.header}>
          <View>
            <Text style={styles.title}>INVOICE</Text>
            <Text style={styles.subtitle}>#{data.invoiceNumber}</Text>
          </View>
          <View>
            <Text>{data.date}</Text>
            <Text style={styles.subtitle}>{data.customerName}</Text>
          </View>
        </View>

        {/* Table */}
        <View style={styles.table}>
          <View style={styles.tableHeader}>
            <Text style={styles.colProduct}>Product</Text>
            <Text style={styles.colQty}>Qty</Text>
            <Text style={styles.colPrice}>Price</Text>
            <Text style={styles.colTotal}>Total</Text>
          </View>

          {data.items.map((item, i) => (
            <View key={i} style={styles.tableRow}>
              <Text style={styles.colProduct}>{item.product}</Text>
              <Text style={styles.colQty}>{item.quantity}</Text>
              <Text style={styles.colPrice}>${item.price.toFixed(2)}</Text>
              <Text style={styles.colTotal}>
                ${(item.quantity * item.price).toFixed(2)}
              </Text>
            </View>
          ))}
        </View>

        {/* Totals */}
        <View style={styles.totalsSection}>
          <View style={styles.totalRow}>
            <Text>Subtotal</Text>
            <Text>${data.subtotal.toFixed(2)}</Text>
          </View>
          <View style={styles.totalRow}>
            <Text>Tax</Text>
            <Text>${data.tax.toFixed(2)}</Text>
          </View>
          <View style={styles.totalRow}>
            <Text style={styles.totalLabel}>Total</Text>
            <Text style={styles.totalLabel}>${data.total.toFixed(2)}</Text>
          </View>
        </View>

        {/* Footer */}
        <Text style={styles.footer}>
          Thank you for your business.
        </Text>
      </Page>
    </Document>
  );
}

/**
 * Generate and download an invoice PDF.
 */
export async function downloadInvoicePdf(data: InvoiceData) {
  const blob = await pdf(<InvoicePdfDocument data={data} />).toBlob();
  downloadBlob(blob, `invoice-${data.invoiceNumber}.pdf`);
}
```

## Print View

Browser's native print dialog using `window.print()`. CSS `@media print` rules hide navigation and show only content.

```css
/* src/styles/print.css — imported in main CSS file */

@media print {
  /* Hide non-content elements */
  nav,
  .sidebar,
  .header,
  .no-print,
  button,
  .toast-container {
    display: none !important;
  }

  /* Reset body */
  body {
    background: white !important;
    color: black !important;
    font-size: 12pt;
  }

  /* Content fills the page */
  .print-content {
    width: 100% !important;
    margin: 0 !important;
    padding: 0 !important;
  }

  /* Page breaks */
  .print-page-break {
    page-break-before: always;
  }

  /* Don't break inside these elements */
  table, figure, .kpi-card {
    page-break-inside: avoid;
  }

  /* Show URLs after links */
  a[href]::after {
    content: ' (' attr(href) ')';
    font-size: 9pt;
    color: #666;
  }
}
```

### Print Button Component

```typescript
// src/components/print-button.tsx

interface PrintButtonProps {
  className?: string;
}

export function PrintButton({ className }: PrintButtonProps) {
  return (
    <button
      onClick={() => window.print()}
      className={`flex items-center gap-2 rounded-md border px-3 py-2 text-sm hover:bg-muted ${className}`}
    >
      <PrinterIcon className="h-4 w-4" />
      Print
    </button>
  );
}
```

### Print-Optimized Report Component

A separate component designed for print. Rendered invisibly on screen, visible only during print.

```typescript
// src/components/export/printable-report.tsx
import { forwardRef } from 'react';

interface ReportData {
  title: string;
  dateRange: string;
  generatedAt: string;
  sections: {
    heading: string;
    content: React.ReactNode;
  }[];
}

export const PrintableReport = forwardRef<HTMLDivElement, { data: ReportData }>(
  ({ data }, ref) => {
    return (
      <div ref={ref} className="hidden print:block print-content">
        {/* Report Header */}
        <div className="mb-8 border-b pb-4">
          <h1 className="text-2xl font-bold">{data.title}</h1>
          <p className="text-sm text-gray-500">
            Period: {data.dateRange} | Generated: {data.generatedAt}
          </p>
        </div>

        {/* Sections */}
        {data.sections.map((section, i) => (
          <div key={i} className="mb-6">
            <h2 className="mb-3 text-lg font-semibold">{section.heading}</h2>
            {section.content}
          </div>
        ))}

        {/* Footer */}
        <div className="mt-8 border-t pt-4 text-center text-xs text-gray-400">
          Confidential - For internal use only
        </div>
      </div>
    );
  }
);
```

## Export Button with Format Selection

A dropdown button that lets the user choose between CSV, PDF, or Print.

```typescript
// src/components/export-menu.tsx
import { useState, useRef } from 'react';
import { Download, FileSpreadsheet, FileText, Printer } from 'lucide-react';

interface ExportMenuProps {
  onCsv: () => void;
  onPdf: () => void;
  onPrint: () => void;
}

export function ExportMenu({ onCsv, onPdf, onPrint }: ExportMenuProps) {
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const options = [
    { label: 'Export CSV', icon: FileSpreadsheet, action: onCsv },
    { label: 'Export PDF', icon: FileText, action: onPdf },
    { label: 'Print', icon: Printer, action: onPrint },
  ];

  return (
    <div className="relative" ref={menuRef}>
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm hover:bg-muted"
      >
        <Download className="h-4 w-4" />
        Export
      </button>

      {open && (
        <div className="absolute right-0 top-full z-10 mt-1 w-44 rounded-md border bg-card py-1 shadow-lg">
          {options.map((opt) => (
            <button
              key={opt.label}
              onClick={() => { opt.action(); setOpen(false); }}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-muted"
            >
              <opt.icon className="h-4 w-4 text-muted-foreground" />
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
```

## Page-Level Usage Example

```typescript
// src/features/orders/pages/orders-page.tsx
import { ExportMenu } from '@/components/export-menu';
import { exportToCsv } from '@/lib/export-csv';
import { exportElementToPdf } from '@/lib/export-pdf-simple';
import { useOrders } from '../hooks/use-orders';

export function Component() {
  const { data: orders } = useOrders();
  const tableRef = useRef<HTMLDivElement>(null);

  const handleCsvExport = () => {
    if (!orders) return;
    exportToCsv('orders', orders, [
      { header: 'Order ID', accessor: (r) => r.id },
      { header: 'Customer', accessor: (r) => r.customerName },
      { header: 'Date', accessor: (r) => r.createdAt },
      { header: 'Status', accessor: (r) => r.status },
      { header: 'Total', accessor: (r) => r.total },
    ]);
  };

  const handlePdfExport = async () => {
    if (tableRef.current) {
      await exportElementToPdf(tableRef.current, 'orders-report');
    }
  };

  return (
    <div className="space-y-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Orders</h1>
        <ExportMenu
          onCsv={handleCsvExport}
          onPdf={handlePdfExport}
          onPrint={() => window.print()}
        />
      </div>

      <div ref={tableRef}>
        {/* Order table */}
      </div>
    </div>
  );
}
```

## Rules

1. **CSV uses BOM.** The `\uFEFF` byte-order mark ensures Excel opens UTF-8 CSV files correctly (especially for non-ASCII characters).
2. **Escape CSV fields.** Fields containing commas, quotes, or newlines must be wrapped in double quotes with internal quotes doubled.
3. **Revoke object URLs.** After triggering a download, always call `URL.revokeObjectURL()` to free memory.
4. **Print CSS is additive.** The `@media print` stylesheet hides navigation and chrome. Content styles are preserved.
5. **Two PDF paths.** Use html2canvas+jsPDF for quick "screenshot" exports. Use @react-pdf/renderer for structured documents (invoices, reports).
6. **No separate export API.** Export uses the same data the user sees. For very large datasets (10k+ rows), the API may provide a streaming export endpoint.
