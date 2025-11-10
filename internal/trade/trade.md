# Trade

There are two “views” of the same trade:
- Commercial/contract view → One trade record (e.g., “Sale of Crude, Q1 2026, 10,000 MT at 3.5 EUR/MT per month”).
- Operational/payment view → Three monthly payment obligations (January, February, March).

You want to:
- Keep one TradeID (because it’s one negotiated deal),
- But still be able to analyze and report monthly (because payments happen monthly).

# Design

Think of your trading data model as having two layers:

- Trade (the deal itself): A single Trade record represents one negotiated purchase or sale, no matter whether it’s for one month, a quarter, or a year.
- TradePeriod (the monthly breakdown): Each Trade is linked to one or more TradePeriod records. They show how the trade’s economics are distributed across months.

Trade:
Each trade references:
- Which Period it covers (e.g., Q1-2026, 2026, or 2026-01),
- The Product, Currency, Volume, and Fee per month,
- Counterparty and other metadata.

For example:

```
Trade{
    ID: "T001",
    Type: "Sale",
    ProductID: "Crude",
    PeriodID: "Q1-2026",
    Volume: 10000,
    FeePerMT: 3.50,
    Currency: "EUR",
}
```
This is one trade. If it’s for Q1 2026, that’s the full deal — one unique trade ID.

TradePeriod:
You do not create multiple trades for a quarterly deal — you create one Trade, and then 3 TradePeriod entries (one per month in Q1).

For example:
```
TradePeriod{
    ID: "TP001", TradeID: "T001", PeriodID: "2026-01", MonthlyTotal: 35_000,
}
TradePeriod{
    ID: "TP002", TradeID: "T001", PeriodID: "2026-02", MonthlyTotal: 35_000,
}
TradePeriod{
    ID: "TP003", TradeID: "T001", PeriodID: "2026-03", MonthlyTotal: 35_000,
}
```

These “belong” to the same trade, but let you:
- Track monthly payments and receipts,
- Filter by month,
- Aggregate by quarter or year.

