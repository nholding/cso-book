package repository

import (
	"context"
	"database/sql"
	"fmt"
	//	"strings"
	"time"

	"github.com/nholding/cso-book/internal/period/domain"
	"github.com/nholding/cso-book/internal/platform/awsclient"
)

// PeriodRepository defines the interface for storing and retrieving Periods from a persistence layer
type PeriodRepository interface {
	// SavePeriods persists Periods. NOTE: ChildPeriodIDs are NOT stored in the DB.
	SavePeriods(ctx context.Context, periods []domain.Period) error

	// GetAllPeriods retrieves all Periods from the DB
	GetAllPeriods(ctx context.Context) ([]domain.Period, error)

	FindByID(ctx context.Context, id string) (*domain.Period, error)
}

type RdsPeriodRepository struct {
	db *sql.DB
}

func NewRdsPeriodRepository(cfg *awsclient.Config) (*RdsPeriodRepository, error) {
	rdsClient, err := cfg.NewRDSClient()
	if err != nil {
		return nil, fmt.Errorf("failed creating the AWS RDS Client: %v", err)
	}

	return &RdsPeriodRepository{db: rdsClient.Client}, nil
}

// SavePeriods Inserts a slice of Periods into the database.
// Will fail if a period with the same ID already exists. This method does NOT touch existing records.
// It assumes the Periods do NOT exist yet in the DB!
//
// Example:
//
//	ctx := context.TODO()
//	err := repo.SavePeriods(ctx, []*domain.Period{period1, period2})
func (p *RdsPeriodRepository) SavePeriods(ctx context.Context, periods []*domain.Period) error {
	if len(periods) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO periods (
			id, name, calendar, granularity, parent_period_id, start_date, end_date,
			audit_created_by, audit_created_at, audit_updated_by, audit_updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, $11)
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, p := range periods {
		if p == nil {
			continue
		}

		if err := p.Validate(); err != nil {
			return fmt.Errorf("period %s validation failed: %w", p.ID, err)
		}

		_, err := stmt.ExecContext(ctx,
			p.ID,
			p.Name,
			p.Calendar,
			p.Granularity,
			p.ParentPeriodID,
			p.StartDate,
			p.EndDate,
			p.AuditInfo.CreatedBy,
			p.AuditInfo.CreatedAt,
			p.AuditInfo.UpdatedBy,
			p.AuditInfo.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert period %s: %w", p.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdatePeriods updates a slice of existing Periods in the database.
// Will fail if a period does NOT exist in the DB.
func (p *RdsPeriodRepository) UpdatePeriods(ctx context.Context, periods []*domain.Period) error {
	if len(periods) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for _, p := range periods {
		query := `
			UPDATE periods
			SET name=$1, granularity=$2, parent_period_id=$3, start_date=$4, end_date=$5, audit_user=$6, audit_updated_at=$7
			WHERE id=$8
		`
		res, err := tx.ExecContext(ctx, query,
			p.Name,
			string(p.Granularity),
			p.ParentPeriodID,
			p.StartDate,
			p.EndDate,
			p.AuditInfo.CreatedBy,
			time.Now().UTC(),
			p.ID,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update period %s: %w", p.ID, err)
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			tx.Rollback()
			return fmt.Errorf("period %s does not exist", p.ID)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit update transaction: %w", err)
	}

	return nil
}

// GetAllPeriods retrieves all periods from the DB
// This is called at startup to populate the in-memory PeriodStore
func (r *RdsPeriodRepository) GetAllPeriods(ctx context.Context) ([]*domain.Period, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, granularity, parent_period_id, start_date, end_date FROM periods`)
	if err != nil {
		return nil, fmt.Errorf("failed to query periods: %w", err)
	}
	defer rows.Close()

	var periods []*domain.Period
	for rows.Next() {
		p := &domain.Period{}
		var granularity string
		if err := rows.Scan(&p.ID, &p.Name, &granularity, &p.ParentPeriodID, &p.StartDate, &p.EndDate); err != nil {
			return nil, fmt.Errorf("failed to scan period row: %w", err)
		}
		p.Granularity = domain.PeriodGranularity(granularity)
		periods = append(periods, p)
	}
	return periods, nil
}

// FindByID retrieves a single period by ID
func (r *RdsPeriodRepository) FindByID(ctx context.Context, id string) (*domain.Period, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, granularity, parent_period_id, start_date, end_date FROM periods WHERE id=$1`, id)

	var p domain.Period
	var granularity string
	if err := row.Scan(&p.ID, &p.Name, &granularity, &p.ParentPeriodID, &p.StartDate, &p.EndDate); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to scan period: %w", err)
	}
	p.Granularity = domain.PeriodGranularity(granularity)
	return &p, nil
}
