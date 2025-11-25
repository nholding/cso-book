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
