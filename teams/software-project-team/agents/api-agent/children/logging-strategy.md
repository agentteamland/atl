---
knowledge-base-summary: "You can't set breakpoints in production — logs are your debugger. Every handler tells its story through logs. Two logs per step: (1) what I'm about to do + with what data, (2) what I did + result. No performance concern — pipeline is non-blocking (Channel.TryWrite = nanoseconds). Log generously; don't be afraid."
---
# Logging Strategy: Virtual Debug

## Core Philosophy

You cannot set breakpoints in production — logs are your debugger. Every handler tells its story through logs. Someone reading them after the fact should be able to answer "what happened, with what data, where did it fail" from the logs alone, without knowing the code at all.

**No performance concern.** The log pipeline works non-blocking: `_logger.Log*()` → Serilog → `BoundedChannel.TryWrite()` (nanoseconds) → separate thread batch publish → RMQ. The total overhead on the handler is on the order of microseconds. Even if you write 50 log lines, it's immeasurably small compared to DB queries. If the channel fills up (10k capacity), `DropOldest` — it never blocks, at worst it loses logs.

**Therefore: log generously, don't be afraid, don't be stingy.**

## Two Log Rule for Every Business Step

For every meaningful business step in the handler:

1. **Entry log:** "I will perform this operation with the data I have"
2. **Result log:** "I performed this operation with this data, result is positive/negative"

On negative results, exception information and relevant context data are also added to the log.

## Concrete Pattern

```csharp
public async Task<CreateOrderResponse> Handle(
    CreateOrderCommand request, CancellationToken ct)
{
    _logger.LogInformation(
        "CreateOrder started for customer {CustomerId}, cart {CartId}",
        request.CustomerId, request.CartId);

    // 1. Customer check
    var customer = await _db.Customers.FindAsync(request.CustomerId, ct)
        ?? throw new NotFoundException(nameof(Customer), request.CustomerId);
    _logger.LogInformation(
        "Customer resolved: {CustomerId}, email {Email}, status {Status}",
        customer.Id, customer.Email, customer.Status);

    // 2. Cart loading
    var cart = await _db.Carts
        .Include(c => c.Items)
        .FirstOrDefaultAsync(c => c.Id == request.CartId, ct)
        ?? throw new NotFoundException(nameof(Cart), request.CartId);
    _logger.LogInformation(
        "Cart loaded: {CartId}, {ItemCount} items, subtotal {Subtotal}",
        cart.Id, cart.Items.Count, cart.Subtotal);

    // 3. Stock check (separate for each product)
    foreach (var item in cart.Items)
    {
        var product = await _db.Products.FindAsync(item.ProductId, ct)
            ?? throw new NotFoundException(nameof(Product), item.ProductId);

        if (product.Stock < item.Quantity)
        {
            _logger.LogWarning(
                "Stock insufficient for product {ProductId} ({ProductName}): "
                + "requested {RequestedQty}, available {AvailableStock}",
                product.Id, product.Name, item.Quantity, product.Stock);
            throw new ValidationException(
                $"Insufficient stock for {product.Name}");
        }

        _logger.LogInformation(
            "Stock OK for product {ProductId}: {RequestedQty}/{AvailableStock}",
            product.Id, item.Quantity, product.Stock);
    }

    // 4. Price verification
    var calculatedTotal = cart.Items.Sum(i => i.UnitPrice * i.Quantity);
    _logger.LogInformation(
        "Price verification: calculated {Calculated}, submitted {Submitted}, "
        + "tax {Tax}, discount {Discount}",
        calculatedTotal, request.SubmittedTotal,
        request.TaxAmount, request.DiscountAmount);

    if (calculatedTotal != request.SubmittedTotal)
    {
        _logger.LogWarning(
            "Price mismatch: calculated {Calculated} vs submitted {Submitted}",
            calculatedTotal, request.SubmittedTotal);
        throw new ValidationException("Price mismatch detected");
    }

    // 5. Order creation
    var order = new Order
    {
        CustomerId = customer.Id,
        Total = calculatedTotal,
    };
    _db.Orders.Add(order);
    await _db.SaveChangesAsync(ct);

    _logger.LogInformation(
        "Order {OrderId} created: customer {CustomerId}, "
        + "{ItemCount} items, total {Total}",
        order.Id, customer.Id, cart.Items.Count, order.Total);

    // 6. Order items
    foreach (var item in cart.Items)
    {
        var orderItem = new OrderItem
        {
            OrderId = order.Id,
            ProductId = item.ProductId,
            Quantity = item.Quantity,
            UnitPrice = item.UnitPrice,
        };
        _db.OrderItems.Add(orderItem);
    }
    await _db.SaveChangesAsync(ct);

    _logger.LogInformation(
        "Order items saved: {ItemCount} items for order {OrderId}",
        cart.Items.Count, order.Id);

    return new CreateOrderResponse(order.Id, order.Total, order.CreatedAt);
}
```

## Log Level Rules

| Level | When |
|-------|------|
| `Information` | Normal flow: step start, successful result, important data points |
| `Warning` | Unexpected but recoverable situation: insufficient stock, price mismatch, rate limit |
| `Error` | Unrecoverable error: before throwing an exception or in a catch block |
| `Debug` | Temporary during development — not carried to production |

## Structured Logging Rules

- **Always use message templates**, NEVER string interpolation:
  ```csharp
  // ✅ Correct — indexed as Serilog property
  _logger.LogInformation("Order {OrderId} created", order.Id);
  
  // ❌ Wrong — plain string, not filterable in Elasticsearch
  _logger.LogInformation($"Order {order.Id} created");
  ```

- **Property names are PascalCase** and meaningful: `{OrderId}`, `{CustomerId}`, `{ItemCount}` — not `{id}`, `{x}`, `{count}`.

- **Sensitive data is never logged:** Fields like password, token, credit card, API key are never written to logs. RmqLogSink does PII masking, but the first line of defense is the handler — never pass sensitive fields to the log at all.

## Logging Checklist for Handlers

Follow this checklist when writing a new handler:

- [ ] At handler start: "starting" log with request parameters
- [ ] After each DB query/external call: result summary log
- [ ] At each validation/check step: success → Info, failure → Warning
- [ ] At handler end: overall result log (created IDs, counts, totals)
- [ ] Before throwing exception: Warning or Error with context
- [ ] Sensitive data check: no password, token, key in logs

