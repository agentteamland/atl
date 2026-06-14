---
knowledge-base-summary: "One-to-one, one-to-many, many-to-many configuration patterns. Navigation properties. Cascade delete behavior selection guide (Cascade, Restrict, SetNull, NoAction). Owned types for value objects. Table splitting for wide entities."
---
# Relationships

## One-to-Many

The most common relationship. Parent has a collection, child has a FK.

```csharp
// Domain entities
public class Customer : BaseEntity
{
    public string Name { get; set; } = default!;
    public ICollection<Order> Orders { get; set; } = new List<Order>();
}

public class Order : BaseEntity
{
    public Guid CustomerId { get; set; }
    public Customer Customer { get; set; } = default!;
}

// Configuration (in OrderConfiguration)
builder.HasOne(x => x.Customer)
    .WithMany(x => x.Orders)
    .HasForeignKey(x => x.CustomerId)
    .HasConstraintName("fk_orders_customers")
    .OnDelete(DeleteBehavior.Restrict);

// FK index — MANDATORY
builder.HasIndex(x => x.CustomerId)
    .HasDatabaseName("ix_orders_customer_id");
```

### Which Side Gets the Configuration?

Configure the relationship on the **dependent** (child) side — the entity that has the FK. `OrderConfiguration` configures the `Order → Customer` relationship, not `CustomerConfiguration`.

## One-to-One

Rare. One entity "extends" another. The dependent entity holds the FK.

```csharp
// Domain
public class User : BaseEntity
{
    public UserProfile? Profile { get; set; }
}

public class UserProfile : BaseEntity
{
    public Guid UserId { get; set; }
    public User User { get; set; } = default!;
    public string Bio { get; set; } = default!;
    public string? AvatarUrl { get; set; }
}

// Configuration (in UserProfileConfiguration)
builder.HasOne(x => x.User)
    .WithOne(x => x.Profile)
    .HasForeignKey<UserProfile>(x => x.UserId)
    .HasConstraintName("fk_user_profiles_users")
    .OnDelete(DeleteBehavior.Cascade);

// Unique index on FK — ensures one-to-one at DB level
builder.HasIndex(x => x.UserId)
    .HasDatabaseName("ix_user_profiles_user_id")
    .IsUnique();
```

## Many-to-Many

### With Explicit Join Entity (Preferred)

When the join table has additional data (e.g., assigned date, role in relationship):

```csharp
// Domain
public class Product : BaseEntity
{
    public ICollection<ProductCategory> ProductCategories { get; set; } = new List<ProductCategory>();
}

public class Category : BaseEntity
{
    public ICollection<ProductCategory> ProductCategories { get; set; } = new List<ProductCategory>();
}

public class ProductCategory : BaseEntity
{
    public Guid ProductId { get; set; }
    public Product Product { get; set; } = default!;
    public Guid CategoryId { get; set; }
    public Category Category { get; set; } = default!;
    public int DisplayOrder { get; set; }
}

// Configuration (in ProductCategoryConfiguration)
builder.HasKey(x => new { x.ProductId, x.CategoryId }); // composite PK

builder.HasOne(x => x.Product)
    .WithMany(x => x.ProductCategories)
    .HasForeignKey(x => x.ProductId)
    .HasConstraintName("fk_product_categories_products")
    .OnDelete(DeleteBehavior.Cascade);

builder.HasOne(x => x.Category)
    .WithMany(x => x.ProductCategories)
    .HasForeignKey(x => x.CategoryId)
    .HasConstraintName("fk_product_categories_categories")
    .OnDelete(DeleteBehavior.Cascade);

// Indexes on FK columns
builder.HasIndex(x => x.ProductId)
    .HasDatabaseName("ix_product_categories_product_id");

builder.HasIndex(x => x.CategoryId)
    .HasDatabaseName("ix_product_categories_category_id");
```

### With Skip Navigation (EF Core Implicit Join)

When the join table has no extra data — EF Core manages the join table automatically:

```csharp
// Domain
public class Student : BaseEntity
{
    public ICollection<Course> Courses { get; set; } = new List<Course>();
}

public class Course : BaseEntity
{
    public ICollection<Student> Students { get; set; } = new List<Student>();
}

// Configuration (in StudentConfiguration or CourseConfiguration — pick one)
builder.HasMany(x => x.Courses)
    .WithMany(x => x.Students)
    .UsingEntity(j => j.ToTable("student_courses"));
```

**Prefer the explicit join entity.** It gives more control, supports additional columns, and follows the explicit-over-implicit principle.

## Cascade Delete Behavior

### Decision Guide

| Scenario | Behavior | Example |
|----------|----------|---------|
| Child cannot exist without parent | `Cascade` | OrderItem when Order is deleted |
| Child should remain when parent is deleted | `Restrict` | Order when Customer is deleted |
| FK should be set to null when parent is deleted | `SetNull` | Comment.AuthorId when User is deleted |
| Let the database handle it (avoid) | `NoAction` | Rarely used |

### Cascade

Child is automatically deleted when parent is deleted.

```csharp
builder.HasOne(x => x.Order)
    .WithMany(x => x.Items)
    .HasForeignKey(x => x.OrderId)
    .OnDelete(DeleteBehavior.Cascade);
```

**Use when:** Child has no meaning without parent (OrderItem without Order).

### Restrict

Prevents parent deletion if children exist. Throws exception.

```csharp
builder.HasOne(x => x.Customer)
    .WithMany(x => x.Orders)
    .HasForeignKey(x => x.CustomerId)
    .OnDelete(DeleteBehavior.Restrict);
```

**Use when:** Children are important entities that should not be silently deleted. The handler must explicitly handle the children first.

### SetNull

Sets the FK to null when parent is deleted. FK column must be nullable.

```csharp
// Entity
public class Comment : BaseEntity
{
    public Guid? AuthorId { get; set; }  // nullable FK
    public User? Author { get; set; }
}

// Configuration
builder.HasOne(x => x.Author)
    .WithMany(x => x.Comments)
    .HasForeignKey(x => x.AuthorId)
    .OnDelete(DeleteBehavior.SetNull);
```

**Use when:** Child should survive parent deletion but lose the reference (e.g., comments remain after user is deleted, shown as "deleted user").

### NoAction

Database takes no action. Application must handle consistency.

**Use when:** Almost never. Prefer Restrict or SetNull.

## Owned Types (Value Objects)

For value objects that don't have their own table — they are stored as columns in the parent table.

```csharp
// Domain
public class Address
{
    public string Street { get; set; } = default!;
    public string City { get; set; } = default!;
    public string PostalCode { get; set; } = default!;
    public string Country { get; set; } = default!;
}

public class Customer : BaseEntity
{
    public string Name { get; set; } = default!;
    public Address ShippingAddress { get; set; } = default!;
    public Address? BillingAddress { get; set; }
}

// Configuration (in CustomerConfiguration)
builder.OwnsOne(x => x.ShippingAddress, address =>
{
    address.Property(a => a.Street)
        .HasColumnName("shipping_street")
        .IsRequired()
        .HasMaxLength(200);
    address.Property(a => a.City)
        .HasColumnName("shipping_city")
        .IsRequired()
        .HasMaxLength(100);
    address.Property(a => a.PostalCode)
        .HasColumnName("shipping_postal_code")
        .IsRequired()
        .HasMaxLength(20);
    address.Property(a => a.Country)
        .HasColumnName("shipping_country")
        .IsRequired()
        .HasMaxLength(100);
});

builder.OwnsOne(x => x.BillingAddress, address =>
{
    address.Property(a => a.Street)
        .HasColumnName("billing_street")
        .HasMaxLength(200);
    // ... same pattern
});
```

Result: The `customers` table has columns `shipping_street`, `shipping_city`, etc. No separate table.

## Table Splitting

Two entities share the same physical table. Useful for separating frequently-accessed columns from rarely-accessed ones.

```csharp
// Domain
public class Order : BaseEntity
{
    public string OrderNumber { get; set; } = default!;
    public decimal TotalAmount { get; set; }
    public OrderDetail Detail { get; set; } = default!;
}

public class OrderDetail
{
    public Guid Id { get; set; }
    public string? InternalNotes { get; set; }
    public string? ShippingInstructions { get; set; }
    public string? CustomFields { get; set; } // large jsonb
}

// Configuration
// OrderConfiguration
builder.HasOne(x => x.Detail)
    .WithOne()
    .HasForeignKey<OrderDetail>(x => x.Id);

builder.ToTable("orders");

// OrderDetailConfiguration
builder.ToTable("orders"); // same table!
```

## Self-Referencing Relationships

Entity references itself (e.g., categories with subcategories, organizational hierarchy).

```csharp
// Domain
public class Category : BaseEntity
{
    public string Name { get; set; } = default!;
    public Guid? ParentCategoryId { get; set; }
    public Category? ParentCategory { get; set; }
    public ICollection<Category> SubCategories { get; set; } = new List<Category>();
}

// Configuration
builder.HasOne(x => x.ParentCategory)
    .WithMany(x => x.SubCategories)
    .HasForeignKey(x => x.ParentCategoryId)
    .HasConstraintName("fk_categories_parent")
    .OnDelete(DeleteBehavior.Restrict); // prevent accidental tree deletion

builder.HasIndex(x => x.ParentCategoryId)
    .HasDatabaseName("ix_categories_parent_category_id");
```

## Relationship Checklist

For every new relationship:

- [ ] FK property defined on the dependent entity
- [ ] Navigation properties on both sides (if bidirectional)
- [ ] `OnDelete` behavior explicitly set
- [ ] FK index created
- [ ] Constraint name follows `fk_{table}_{ref}` convention
- [ ] Cascade behavior documented (why this choice)
