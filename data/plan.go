package data

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Plan struct {
	ID                  int
	PlanName            string
	PlanAmount          int
	PlanAmountFormatted string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (p *Plan) GetAll() ([]*Plan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, plan_name, plan_amount, created_at, updated_at
	from plans order by id`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*Plan

	for rows.Next() {
		var plan Plan
		err := rows.Scan(
			&plan.ID,
			&plan.PlanName,
			&plan.PlanAmount,
			&plan.CreatedAt,
			&plan.UpdatedAt,
		)

		plan.PlanAmountFormatted = plan.AmountForDisplay()
		if err != nil {
			log.Println("Error scanning", err)
			return nil, err
		}

		plans = append(plans, &plan)
	}

	return plans, nil
}

func (p *Plan) GetById(id int) (*Plan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, plan_name, plan_amount, created_at, updated_at from plans where id = $1`

	var plan Plan
	row := db.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&plan.ID,
		&plan.PlanName,
		&plan.PlanAmount,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	plan.PlanAmountFormatted = plan.AmountForDisplay()

	println("INI AMOUNT FORMATTED : ", plan.PlanAmountFormatted)

	return &plan, nil
}

func (p *Plan) SubscribeUserToPlan(user User, plan Plan) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// delete existing plan, if any
	stmt := `delete from user_plans where user_id = $1`
	_, err := db.ExecContext(ctx, stmt, user.ID)
	if err != nil {
		return err
	}

	// subscribe to new plan
	stmt = `insert into user_plans (user_id, plan_id, created_at, updated_at)
			values ($1, $2, $3, $4)`

	_, err = db.ExecContext(ctx, stmt, user.ID, plan.ID, time.Now(), time.Now())
	if err != nil {
		return err
	}
	return nil
}

// helper buat convert int ke format rupiah
func (p *Plan) AmountForDisplay() string {
	amount := float64(p.PlanAmount) / 1000.0
	return fmt.Sprintf("Rp%.3f", amount)
}
