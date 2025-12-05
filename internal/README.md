# 🏗️ Project Architecture Overview

This project follows a **domain-oriented architecture** inspired by Clean and Hexagonal Architecture, but adapted to be idiomatic for Go.  
The goal is to keep domain logic pure, avoid circular dependencies, and prevent the emergence of a dangerous *“god package.”*

## Why Domain-Oriented Structure?

Most Go projects use *technical layering*:

```bash
models/
services/
repositories/
controllers/
util/
```

That approach **scales badly** and inevitably produces a *god package*—a directory where everything ends up, usually `models/` or `utils/`.
In contrast, **domain-oriented** structure groups code by business concept:

```bash
period/
company/
trade/
pricing/
.../
```

Each domain is self-contained, easy to navigate, and free from cross-domain clutter.

Benefits:

- Clear boundaries  
- Testable domain logic  
- No circular imports  
- Replaceable infrastructure (DB, S3, etc.)  
- Easy to scale as more domains are added  

## Layer Responsibilities

### **1. domain/**
Pure business logic:
- Entities
- Value objects
- Period hierarchy generation
- Date breakdown logic
- Invariants and validation

**No AWS SDK, no SQL, no HTTP.**

This is the *core* of the system and should remain free of external dependencies.

### **2. repository/**
Holds **interfaces**, not implementations:

```go
type PeriodRepository interface {
    Save(ctx context.Context, p *Period) error
    FindByID(ctx context.Context, id string) (*Period, error)
    ListAll(ctx context.Context) ([]*Period, error)
}
````

These interfaces are depended on by:
* Services
* Domain logic (only when needed)
* But they have no dependencies back.

### **3. service/**
This is where workflows live.

Examples:
* Load all periods from DB → build in-memory cache
* Validate period ranges
* Create new company / validate uniqueness
* Compute price adjustments

Service layer coordinates logic such as:

```go 
err := periodService.PreloadCache(ctx)
````

Responsibilities:
* Orchestrate across repositories
* Apply business rules
* Check invariants
* Perform multi-step workflows

Dependencies:
✔ domain
✔ repository interfaces
❌ No AWS
❌ No SQL
❌ No infra packages

### **4. infrastructure/**
Real Implementation

This layer satisfies the repository interfaces. Here is where you use:
* SQL (PostgreSQL)
* AWS RDS IAM authentication
* AWS S3 SDK
* DynamoDB (future)
* HTTP clients (future)

Example:

```go
type PeriodRepositoryPostgres struct {
    db *sql.DB
}

func (r *PeriodRepositoryPostgres) FindByID(ctx context.Context, id string) (*period.Period, error) {
    // SELECT ... FROM periods WHERE id=$1
}
```

Responsibilities
* Translate DB rows ↔ domain entities
* Handle IAM auth tokens
* Handle S3 I/O
* Perform migrations (optional)

Dependencies:
✔ database/sql
✔ AWS SDK
✔ platform/aws
❌ domain should NOT depend on this layer

### **5. platform/**
Some things aren’t domain-specific:
* AWS config loader
* RDS IAM auth client
* S3 clients
* Secrets manager client
* Kubernetes IRSA support
* Logging (optional)

These belong in:

internal/platform/aws/
internal/platform/db/
internal/platform/log/

These are shared by all domains and avoid duplication. The dependency direction always flows into infrastructure, never from it.
