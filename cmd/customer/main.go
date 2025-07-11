package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"go-shopping-poc/cmd/customer/api"
	"go-shopping-poc/domain/customer"
	"go-shopping-poc/pkg/apiutil"
	"go-shopping-poc/pkg/config"
	"go-shopping-poc/pkg/logging"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logging.SetLevel("DEBUG")
	logging.Info("Customer service started...")

	// Load configuration

	envFile := config.ResolveEnvFile()
	cfg := config.Load(envFile)

	logging.Info("Configuration loaded from %s", envFile)
	//logging.Info("Config: %v", cfg)

	// Connect to Postgres
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" && cfg.GetCustomerDBURL() != "" {
		dbURL = cfg.GetCustomerDBURL()
	}
	if dbURL == "" {
		logging.Error("DATABASE_URL not set")
	}
	dbpool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		logging.Error("Failed to connect to DB: %v", err)
	}
	defer dbpool.Close()

	queries := customer.New(dbpool)

	mux := http.NewServeMux()
	mux.Handle("/customer", apiutil.JWTMiddleware([]byte("your-keycloak-public-key"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetCustomer(w, r, queries)
		case http.MethodPost:
			handleAddCustomer(w, r, queries)
		case http.MethodPut:
			handleUpdateCustomer(w, r, queries)
		default:
			apiutil.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})))

	addr := ":8080"
	logging.Info("Listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logging.Error("Server error: %v", err)
	}
}

func handleGetCustomer(w http.ResponseWriter, r *http.Request, queries *customer.Queries) {
	id := r.URL.Query().Get("id")
	if id == "" {
		apiutil.Error(w, http.StatusBadRequest, "missing id")
		return
	}
	id = uuid.MustParse(id).String() // Ensure id is a valid UUID
	pg_id := pgtype.UUID{Bytes: uuid.MustParse(id), Valid: true}
	// Fetch customer
	dbCustomer, err := queries.GetCustomer(r.Context(), pg_id)
	if err != nil {
		apiutil.Error(w, http.StatusNotFound, "customer not found")
		return
	}
	// Fetch related data (adjust query names as needed)
	dbAddresses, _ := queries.GetCustomerAddresses(r.Context(), dbCustomer.Customerid)
	dbCreditCards, _ := queries.GetCustomerCreditCards(r.Context(), dbCustomer.Customerid)
	dbStatuses, _ := queries.GetCustomerStatuses(r.Context(), dbCustomer.Customerid)

	apiCustomer := api.ConvertCustomerDBToAPI(dbCustomer, dbAddresses, dbCreditCards, dbStatuses)
	apiutil.JSON(w, http.StatusOK, apiCustomer)
}

func handleAddCustomer(w http.ResponseWriter, r *http.Request, queries *customer.Queries) {
	var req api.Customer
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	dbAddCustomer := api.ConvertCustomerAPIToDB(req)
	dbAddAddresses := api.ConvertAddressesAPIToDB(req.Addresses)
	dbAddCreditCards := api.ConvertCreditCardsAPIToDB(req.CreditCards)
	dbAddStatuses := api.ConvertStatusesAPIToDB(req.Statuses)

	// Insert customer (adjust query name as needed)
	id, err := queries.AddCustomer(r.Context(), dbAddCustomer)
	if err != nil {
		apiutil.Error(w, http.StatusInternalServerError, "failed to add customer")
		return
	}
	apiutil.JSON(w, http.StatusCreated, map[string]interface{}{"id": id})
}

func handleUpdateCustomer(w http.ResponseWriter, r *http.Request, queries *customer.Queries) {
	var req api.Customer
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiutil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	dbCustomer := api.ConvertCustomerAPIToDB(req)
	// Update customer (adjust query name as needed)
	err := queries.UpdateCustomer(r.Context(), dbCustomer)
	if err != nil {
		apiutil.Error(w, http.StatusInternalServerError, "failed to update customer")
		return
	}
	apiutil.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
