package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Payment Service is healthy")
	})
	router.HandleFunc("/accounts", createStripeAccountHandler).Methods("POST")
	router.HandleFunc("/accounts/{id}/onboarding", getStripeOnboardingLinkHandler).Methods("GET")

	// TODO: Implement Stripe Connect handlers
	// TODO: Implement TSE (Technical Security Device) integration

	log.Printf("Payment Service starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}






func createStripeAccountHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Mock Stripe Connect Account Creation
	accountID := "acct_mock_" + input.UserID
	log.Printf("Created mock Stripe Connect account %s for user %s", accountID, input.UserID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"stripe_account_id": accountID,
		"status":            "created",
	})
}

func getStripeOnboardingLinkHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := vars["id"]

	// Mock Onboarding Link
	link := "https://connect.stripe.com/setup/s/mock_" + accountID
	log.Printf("Generated mock onboarding link for account %s", accountID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url": link,
	})
}
