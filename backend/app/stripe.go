package app

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"example/my-go-api/app/config"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/customer"
)

// InitStripe wires the Stripe API key from the environment.
func InitStripe() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config for stripe: %v", err)
	}
	stripe.Key = cfg.Stripe.SecretKey
}

// ensureStripeCustomer finds or creates a Stripe Customer for the given user.
// It uses users.stripe_customer_id when present, otherwise creates a new customer
// with metadata auth0_sub = <auth0Sub>, then stores that in the users table.
func ensureStripeCustomer(ctx context.Context, auth0Sub string) (string, error) {
	if db == nil {
		return "", errors.New("db not initialized")
	}
	if auth0Sub == "" {
		return "", errors.New("missing auth0 sub")
	}

	var stripeID sql.NullString
	err := db.QueryRowContext(
		ctx,
		`
			SELECT stripe_customer_id
			FROM users
			WHERE auth0_sub = $1;
		`,
		auth0Sub,
	).Scan(&stripeID)
	if err != nil {
		return "", err
	}

	if stripeID.Valid && stripeID.String != "" {
		return stripeID.String, nil
	}

	params := &stripe.CustomerParams{
		Metadata: map[string]string{
			"auth0_sub": auth0Sub,
		},
	}
	cust, err := customer.New(params)
	if err != nil {
		return "", err
	}

	_, err = db.ExecContext(
		ctx,
		`
			UPDATE users
			SET stripe_customer_id = $1
			WHERE auth0_sub = $2;
		`,
		cust.ID,
		auth0Sub,
	)
	if err != nil {
		return "", err
	}

	return cust.ID, nil
}
