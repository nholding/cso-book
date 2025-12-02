# Trade Management System

## Overview

The **Trade Management System** models commodity trading activities, such as purchases and sales of fossil oil products, by splitting trades into **monthly breakdowns** for precise reporting and auditing. A trade can represent a purchase or sale that spans across one or more months, quarters, or even full years.

Each trade is represented by a **parent trade** (either a `Purchase` or a `Sale`), and the system breaks down the trade into **TradeBreakdowns** that reflect the monthly periods associated with the trade.

---

## Relationship Between Parent Trade and Breakdown

- **Parent Trade**: 
    - The core entity representing a **Purchase** or **Sale**. 
    - This contains essential information about the trade, including the price, volume, currency, and status of the trade. 
    - The status of a trade (e.g., Draft, Confirmed, Canceled) is tracked only at the parent level.
  
- **TradeBreakdown**:
    - A breakdown corresponds to **one or more periods** that a trade spans. For example, a `Purchase` trade for Q1 would result in multiple breakdowns—one for each month in Q1 (January, February, March).
    - Each breakdown represents a **full-month trade volume** with the same price and other parameters as the parent trade, divided across the relevant months or periods.
    - Trade breakdowns are automatically created when a trade is registered, ensuring that any trade spanning multiple periods (like months or quarters) is appropriately reflected in the system.

The breakdowns of a trade are **not standalone** but are instead **linked to the parent trade** through the `ParentTradeID`. This ensures that each breakdown can be traced back to the original parent trade for audit and reporting purposes.

### Example:

- A **Purchase** for Q1 would create:
  - A parent `Purchase` record that contains the overall trade details (e.g., price, volume, status).
  - Three breakdowns (one for each month in Q1: January, February, and March), each representing the same trade value but for different periods.

---

## Trade Status Handling

- **Option 2** was selected for the **status design**.
  - The **Status** field is stored only at the **parent trade level** (`Purchase` or `Sale`) and not in the breakdowns. 
  - The `TradeBreakdown` type does **not** include a `Status` field because it would be redundant—status changes should only apply to the parent trade, not individual breakdowns. A status change (e.g., from "Draft" to "Confirmed") applies uniformly across all breakdowns of a trade.
  - **Status Audit**: While the `TradeBreakdown` does not store the status, **status changes** of the parent trade are tracked using a **TradeStatusHistory** record. This records changes like:
    - Status transitions (Draft → Confirmed → Canceled)
    - The time of the status change
    - The user who performed the change
    - The reason for the change, if applicable (especially useful for cancellations)

---

## Why Status is Not in TradeBreakdowns

- **Centralized Status Management**: 
    - The status of a trade is meant to represent the **state of the entire trade** and not just a specific part (i.e., one period). This is why we only track the `Status` at the parent level.
    - By removing `Status` from the `TradeBreakdown`, we ensure that there is a **single point of truth** for the trade status and avoid the complexity of managing individual breakdown statuses.

- **Status Audit**: 
    - Changes to the trade's status are separately tracked in a **`TradeStatusHistory`** entity, which logs the **old status**, **new status**, **time of change**, and **reason for the change**.
    - This allows for full traceability and auditability, ensuring that any changes in the trade status can be reviewed for regulatory or business purposes.

---

## Storing Trades and TradeBreakdowns to Persistent Storage

### Trade Persistence Strategy

- The system stores both the **parent trade** (i.e., `Purchase` or `Sale`) and its corresponding **TradeBreakdowns** in a persistent database, such as **AWS RDS** or another relational database.
- **Parent Trade** and **TradeBreakdowns** are stored separately, but they are linked via the `ParentTradeID`. This enables efficient querying and reporting while ensuring that data consistency is maintained.
  - **Parent Trade**: Contains the overall details of the trade, such as price, volume, status, and audit information.
  - **TradeBreakdown**: Contains period-specific details for the trade, such as volume and price for each month or period.

### Workflow for Storing Trades

1. **Creating a New Trade**:
   - When a new `Purchase` or `Sale` is created, the trade is saved to the database as a **parent trade**.
   - The system automatically generates **TradeBreakdowns** for the corresponding periods (e.g., months or quarters) based on the trade's time span (e.g., a trade for Q1 would generate breakdowns for January, February, and March).
   - Both the parent trade and its breakdowns are stored in the database. Breakdown records reference the `ParentTradeID` to link them to the parent trade.

2. **Updating a Trade**:
   - When a trade is updated (e.g., price changes, status changes), the parent trade record is updated first.
   - If any changes affect the breakdowns (e.g., price change, volume adjustment), the breakdowns are updated as well.
   - If the trade’s status changes (e.g., from Draft → Confirmed), this is tracked via a **TradeStatusHistory** record, which logs the old and new status and the reason for the change.
   - Both the updated parent trade and the breakdowns are saved back to the database. Breakdown records are always kept in sync with the parent trade to maintain consistency.

3. **Cancelling a Trade**:
   - When a trade is canceled (e.g., rejected by the buyer), the status of the parent trade is updated to **Canceled**.
   - A **TradeStatusHistory** record is created to document the cancellation, including the reason for the cancellation and the time it occurred.
   - If necessary, breakdown records associated with the canceled trade are updated, though in most cases, the cancellation status will be recorded at the parent level only.

4. **Trade Lookup and Auditing**:
   - Both **parent trades** and **TradeBreakdowns** can be retrieved via SQL queries or database calls. The `ParentTradeID` is used to filter and aggregate all breakdowns associated with a specific parent trade.
   - The system supports full auditability, allowing users to track all changes to a trade, including status changes, price adjustments, and cancellations.

---

## Design Notes

- **Trade Creation**: When a new `Purchase` or `Sale` is created, the system automatically generates the relevant **TradeBreakdowns** based on the trade's period range (e.g., monthly, quarterly).
  
- **Updates to Trade**: When a trade is updated (e.g., price changes, status updates), the parent trade is updated first, and its associated breakdowns are then updated to reflect the changes. Breakdown updates are usually not needed for changes that do not affect the period or volume (e.g., status updates).

- **Audit Information**: All trades and breakdowns include an `AuditInfo` field, which tracks the user who created or updated the trade and when these actions occurred. The audit info is carried forward from the parent trade to each breakdown.

---

## Future Considerations

- **Position Management**: The system could later incorporate features for aggregating trades across multiple periods to calculate **overall positions** and **P&L** (profit and loss) for different periods (e.g., quarterly or yearly exposure).
  
- **Status Transitions**: Additional status transitions (e.g., Draft → Confirmed → Executed) may be introduced as business requirements evolve.

- **Trade Cancellation and Reversal**: The system handles trade cancellations by updating the `Status` of the parent trade (e.g., Draft → Canceled). Breakdown data can also be adjusted if necessary for accurate reporting.

---

## Conclusion

This design ensures **clean separation of concerns** between the **parent trade** (which holds the overall trade details and status) and the **breakdowns** (which hold period-specific trade details). This structure simplifies tracking, auditing, and aggregating trades over time, while also ensuring flexibility for status updates and trade cancellations.
