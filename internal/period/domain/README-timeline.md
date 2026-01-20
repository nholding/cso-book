# Period mental model

1. Gregorian calendar is generated first
   └─ Years → Quarters → Months (atomic facts)

2. Fiscal calendars are generated second
   └─ Years → Quarters (overlays)
      └─ reference existing months by ID

* Gregorian months must exist first
* Fiscal generation depends on them
* Startup enforces this order

# Period Model: Calendar vs Fiscal Coexistence

**Trades should always be booked against Gregorian periods. Fiscal years should never be trade anchors — they are reporting and aggregation lenses.**

**Fiscal periods exist for reporting, aggregation, and slicing — not for trade definition. Think of fiscal year as a view, not a dimension.** 

This system models time using Gregorian calendar periods as the single source of truth, with fiscal calendars implemented as overlays on top of those periods.

This design is intentional and critical for:
* Commodity trading
* Risk aggregation
* Delivery schedules
* P&L attribution
* Regulatory reporting

The core principle is:
Gregorian months are the atomic unit of time.
Fiscal calendars do not own time — they group time.

## Core Design Principles

[^1]: Gregorian months are atomic
* All trades ultimately resolve to a set of months
* No trade ever resolves to “partial” months
[^2]: Calendar (CAL) and Fiscal (FY) calendars coexist
* They are independent hierarchies
* They may reference the same months by date
* They must never cross-link hierarchically
[^3]: Fiscal calendars are overlays
* No fiscal months exist
* Fiscal years and quarters reuse existing Gregorian months
* This avoids duplication and ambiguity
[^4]: All validation is fail-fast
* Structural integrity is enforced at startup
* If validation fails, the application must not start

### Gregorian Calendar example
```
CALENDAR (CAL)
│
├── 2026
│   ├── 2026-Q1
│   │   ├── 2026-JAN
│   │   ├── 2026-FEB
│   │   └── 2026-MAR
│   ├── 2026-Q2
│   │   ├── 2026-APR
│   │   ├── 2026-MAY
│   │   └── 2026-JUN
│   ├── 2026-Q3
│   └── 2026-Q4
```

### Fiscal Calendar (FY – April Start Example)
```
FISCAL (FY)
│
├── FY2026
│   ├── FY2026-Q1
│   │   ├── 2026-APR
│   │   ├── 2026-MAY
│   │   └── 2026-JUN
│   ├── FY2026-Q2
│   │   ├── 2026-JUL
│   │   ├── 2026-AUG
│   │   └── 2026-SEP
│   ├── FY2026-Q3
│   │   ├── 2026-OCT
│   │   ├── 2026-NOV
│   │   └── 2026-DEC
│   └── FY2026-Q4
│       ├── 2027-JAN
│       ├── 2027-FEB
│       └── 2027-MAR
```

🔑 Important:
The months (2026-APR, 2027-JAN, etc.) are the same objects in memory and in the database.

## Calendar Isolation Invariant

A period may only have a parent with the same calendar type.

| Child Period | Allowed Parent | Forbidden parent |
| ------------- | ------------- | -----------------|
| 2026-JAN (CAL) | 2026-Q1 (CAL) | FY2026-Q4 (FY) |
| FY2026-Q1 (FY | FY2026 (FY) | 2026 (CAL) |

This prevents ambiguity and ensures safe aggregation.


