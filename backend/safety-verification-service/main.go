package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type VerificationRequest struct {
	DriverID   string `json:"driver_id"`
	DocumentID string `json:"document_id"`
	DocType    string `json:"doc_type"` // e.g., "P-Schein", "ID", "Insurance"
}

type VerificationStatus struct {
	DriverID string `json:"driver_id"`
	Status   string `json:"status"` // "pending", "approved", "rejected"
	Message  string `json:"message"`
}

var log = logrus.New()

func init() {
	log.Out = os.Stdout
	log.SetFormatter(&logrus.JSONFormatter{})
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/health", HealthHandler).Methods("GET")
	r.HandleFunc("/verify", VerifyHandler).Methods("POST")
	r.HandleFunc("/status/{driver_id}", StatusHandler).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	log.Infof("Safety & Verification Service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func VerifyHandler(w http.ResponseWriter, r *http.Request) {
	var req VerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Infof("Received verification request for driver %s, type %s", req.DriverID, req.DocType)

	// Mock logic for compliance with German PBefG
	status := VerificationStatus{
		DriverID: req.DriverID,
		Status:   "pending",
		Message:  "Verification initiated. Compliance check with PBefG in progress.",
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(status)
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	driverID := vars["driver_id"]

	log.Infof("Checking verification status for driver %s", driverID)

	// Mock response
	status := VerificationStatus{
		DriverID: driverID,
		Status:   "approved",
		Message:  "All documents verified. Compliant with German regulations.",
	}

	json.NewEncoder(w).Encode(status)
}
