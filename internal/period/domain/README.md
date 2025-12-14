# Period

## Period init
Check-before-create

If periods for startYear exist, we skip creation. ✅

Otherwise, call GeneratePeriods to generate years, quarters, months.

Database insert

Inserts each period individually (could batch for performance).

Fields inserted: id, name, granularity, parent_period_id, start_date, end_date, created_by.

Load all periods into memory

Regardless of whether new periods were generated or already exist.

Returns a PeriodStore instance.

This avoids repeated queries to RDS during normal operations.

Use case

At application startup, call InitializePeriods.

Then all trade calculations (BreakDownTradePeriodRange, CreateTradeBreakdowns) can use PeriodStore in memory.

Chronological order

PeriodStore gua

## Start inclusive, end exclusive
This project stores all period date ranges (years, quarters, months) using the industry-standard start-inclusive, end-exclusive format:
```go
[start_date, end_date]
```

This means:
* start_date is included
* end_date is excluded
* The period covers all timestamps up to the day before end_date

Example:

| Period    | Stored Start | Stored End | Actual Meaning             |
| --------- | ------------ | ---------- | -------------------------- |
| 2026 year | 2026-01-01   | 2027-01-01 | Jan 1 → Dec 31 (full year) |
| 2026-FEB  | 2026-02-01   | 2026-03-01 | Feb 1 → Feb 28/29          |
| Q1 2026   | 2026-01-01   | 2026-04-01 | Jan 1 → Mar 31             |

Although the end date appears “one day too far”, it is **correct** and **intentional**.

Using exclusive end dates avoids multiple categories of bugs and avoids the following problems:
* Off-by-one errors
* Overlapping periods
* Gaps between consecutive periods
* Complicated “end of month” logic
* Leap year inconsistencie
* SQL queriers simpler (no need for BETWEEN)
