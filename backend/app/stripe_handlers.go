package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"example/my-go-api/app/config"
	"example/my-go-api/app/models"
	"example/my-go-api/auth"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v79"
	portal "github.com/stripe/stripe-go/v79/billingportal/session"
	"github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/webhook"
)

// CreateCheckoutSession starts a Stripe Checkout Session for the authenticated user.
func CreateCheckoutSession(c *gin.Context) {
	claims, ok := auth.ClaimsFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing auth context"})
		return
	}

	stripeCustomerID, err := ensureStripeCustomer(c.Request.Context(), claims.Subject)
	if err != nil {
		log.Printf("ensureStripeCustomer failed for sub=%s: %v", claims.Subject, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare billing"})
		return
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("stripe checkout config load failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "billing not configured"})
		return
	}

	priceID := cfg.Stripe.PriceIDProMonthly
	frontendURL := strings.TrimRight(cfg.Stripe.FrontendURL, "/")
	if priceID == "" || frontendURL == "" {
		log.Printf("missing Stripe config: price_id=%t frontend_url=%t", priceID != "", frontendURL != "")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "billing not configured"})
		return
	}

	params := &stripe.CheckoutSessionParams{
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer: stripe.String(stripeCustomerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(frontendURL + "/billing/success"),
		CancelURL:  stripe.String(frontendURL + "/billing/cancel"),
	}

	sess, err := session.New(params)
	if err != nil {
		log.Printf("stripe checkout session failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create checkout session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": sess.URL})
}

// StripeWebhook handles Stripe subscription events and updates user plans.
func StripeWebhook(c *gin.Context) {
	const maxBodyBytes = int64(65536)
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodyBytes))
	if err != nil {
		log.Printf("stripe webhook read failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	sigHeader := c.GetHeader("Stripe-Signature")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("stripe webhook config load failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "webhook not configured"})
		return
	}

	endpointSecret := cfg.Stripe.WebhookSecret
	if endpointSecret == "" {
		log.Printf("stripe webhook secret missing")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "webhook not configured"})
		return
	}

	event, err := webhook.ConstructEventWithOptions(
		body,
		sigHeader,
		endpointSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		log.Printf("stripe webhook signature failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "signature verification failed"})
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			log.Printf("stripe session unmarshal failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session payload"})
			return
		}
		customerID := ""
		if sess.Customer != nil {
			customerID = sess.Customer.ID
		}
		if customerID == "" {
			log.Printf("stripe session missing customer id")
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing customer id"})
			return
		}

		if err := updateUserPlanByStripeCustomer(c.Request.Context(), customerID, models.PlanPro); err != nil {
			log.Printf("stripe plan upgrade failed customer=%s err=%v", customerID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Printf("stripe subscription unmarshal failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription payload"})
			return
		}
		customerID := ""
		if sub.Customer != nil {
			customerID = sub.Customer.ID
		}
		if customerID == "" {
			log.Printf("stripe subscription missing customer id")
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing customer id"})
			return
		}

		if err := updateUserPlanByStripeCustomer(c.Request.Context(), customerID, models.PlanFree); err != nil {
			log.Printf("stripe plan downgrade failed customer=%s err=%v", customerID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
			return
		}
	default:
		// Intentionally ignore unhandled events.
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// CreatePortalSession creates a Stripe Customer Portal session for the authenticated user.
func CreatePortalSession(c *gin.Context) {
	claims, ok := auth.ClaimsFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing auth context"})
		return
	}
	if claims.Subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing auth0 sub"})
		return
	}
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db not initialized"})
		return
	}

	var stripeCustomerID sql.NullString
	err := db.QueryRowContext(
		c.Request.Context(),
		`
			SELECT stripe_customer_id
			FROM users
			WHERE auth0_sub = $1;
		`,
		claims.Subject,
	).Scan(&stripeCustomerID)
	if err != nil {
		log.Printf("portal lookup failed sub=%s err=%v", claims.Subject, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load customer"})
		return
	}
	if !stripeCustomerID.Valid || stripeCustomerID.String == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stripe customer missing for user"})
		return
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("portal config load failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "billing not configured"})
		return
	}

	frontendURL := strings.TrimRight(cfg.Stripe.FrontendURL, "/")
	if frontendURL == "" {
		log.Printf("missing Stripe config: frontend_url=false")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "billing not configured"})
		return
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(stripeCustomerID.String),
		ReturnURL: stripe.String(frontendURL + "/settings/billing"),
	}

	sess, err := portal.New(params)
	if err != nil {
		log.Printf("stripe portal session failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create portal session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": sess.URL})
}

type updatePlanRequest struct {
	Plan models.Plan `json:"plan"`
}

// UpdateUserPlan sets the authenticated user's plan to the requested value.
func UpdateUserPlan(c *gin.Context) {
	claims, ok := auth.ClaimsFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing auth context"})
		return
	}
	if claims.Subject == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing auth0 sub"})
		return
	}
	var req updatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Plan != models.PlanPro && req.Plan != models.PlanFree {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan"})
		return
	}
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db not initialized"})
		return
	}

	_, err := db.ExecContext(
		c.Request.Context(),
		`
			UPDATE users
			SET plan = $1
			WHERE auth0_sub = $2;
		`,
		req.Plan,
		claims.Subject,
	)
	if err != nil {
		log.Printf("update plan failed sub=%s err=%v", claims.Subject, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func updateUserPlanByStripeCustomer(ctx context.Context, stripeCustomerID string, plan models.Plan) error {
	if db == nil {
		return errors.New("db not initialized")
	}
	if stripeCustomerID == "" {
		return errors.New("missing stripe customer id")
	}
	_, err := db.ExecContext(
		ctx,
		`
			UPDATE users
			SET plan = $1
			WHERE stripe_customer_id = $2;
		`,
		plan,
		stripeCustomerID,
	)
	return err
}
