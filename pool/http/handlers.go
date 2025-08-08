package http

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"pool/core"
	"pool/logs"
	"pool/metrics"
	"pool/security"
	"pool/utils"
)

type StandardResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type SharePayload struct {
	MinerID  string  `json:"miner_id"`
	JobID    string  `json:"job_id"`
	Nonce    string  `json:"nonce"`
	Hash     string  `json:"hash"`
	HashRate float64 `json:"hashrate"`
}

type PayoutRequest struct {
	To     string `json:"to"`
	Amount string `json:"amount"`
}

type RewardSender interface {
	SendReward(to string, amount *big.Int) (string, error)
}

var paymentClient RewardSender

func SetPaymentClient(rs RewardSender) {
	paymentClient = rs
}

type ShareSaver interface {
	SaveShare(ctx context.Context, s *core.Share) error
}

type BalanceStore interface {
	AddBalance(ctx context.Context, minerID string, delta float64) error
	GetBalance(ctx context.Context, minerID string) (float64, error)
	ResetBalance(ctx context.Context, minerID string) error
}

func respond(ctx context.Context, w http.ResponseWriter, code int, data StandardResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func SubmitShareHandler(pool *core.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqID := utils.GetRequestID(ctx)

		logger := logs.WithFields(map[string]interface{}{
			"request_id": reqID,
			"path":       r.URL.Path,
			"method":     r.Method,
		})

		if r.Method != http.MethodPost {
			respond(ctx, w, http.StatusMethodNotAllowed, StandardResponse{
				Status:  "error",
				Message: "Method Not Allowed",
			})
			return
		}

		var payload SharePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			logger.Warn("Invalid JSON structure")
			respond(ctx, w, http.StatusBadRequest, StandardResponse{
				Status:  "error",
				Message: "Invalid JSON structure",
			})
			return
		}

		if payload.MinerID == "" || payload.JobID == "" || payload.Nonce == "" {
			logger.Warn("Missing required fields")
			respond(ctx, w, http.StatusBadRequest, StandardResponse{
				Status:  "error",
				Message: "Missing required fields",
			})
			return
		}

		if security.IsFraudulentNonce(payload.MinerID, payload.Nonce) {
			logger.Warn("Suspicious share blocked")
			metrics.SharesInvalid.Inc()
			respond(ctx, w, http.StatusForbidden, StandardResponse{
				Status:  "error",
				Message: "Suspicious activity detected",
			})
			return
		}

		share := &core.Share{
			JobID:     payload.JobID,
			WorkerID:  payload.MinerID,
			Nonce:     payload.Nonce,
			Hash:      payload.Hash,
			Diff:      payload.HashRate,
			Timestamp: time.Now(),
		}

		processor := core.NewShareProcessor(core.NewDefaultShareValidator(pool.ShareStore), pool.ShareStore)
		start := time.Now()
		result := processor.Process(*share, "0000", 30*time.Second)
		metrics.ValidationDuration.Observe(time.Since(start).Seconds())

		if !result.Valid {
			logger.Error("Share rejected: " + result.Description)
			metrics.SharesInvalid.Inc()
			respond(ctx, w, http.StatusForbidden, StandardResponse{
				Status:  "error",
				Message: result.Description,
			})
			return
		}

		if err := pool.Engine.SubmitShare(core.WorkerIdentifier(payload.MinerID), share); err != nil {
			logger.Error("Failed to submit share to engine: " + err.Error())
			respond(ctx, w, http.StatusInternalServerError, StandardResponse{
				Status:  "error",
				Message: "Failed to submit share",
			})
			return
		}

		if saver, ok := pool.ShareStore.(ShareSaver); ok {
			if err := saver.SaveShare(ctx, share); err != nil {
				logger.Error("Failed to save share to DB: " + err.Error())
			}
		}

		if bal, ok := pool.ShareStore.(BalanceStore); ok {
			if err := bal.AddBalance(ctx, payload.MinerID, payload.HashRate); err != nil {
				logger.Error("Failed to update miner balance: " + err.Error())
			}
		}

		logger.Info("Valid share accepted")
		metrics.SharesValid.Inc()
		respond(ctx, w, http.StatusOK, StandardResponse{
			Status:  "success",
			Message: "Share accepted",
		})
	}
}

func GetPoolStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := utils.GetRequestID(ctx)

	logger := logs.WithFields(map[string]interface{}{
		"request_id": reqID,
		"path":       r.URL.Path,
		"method":     r.Method,
	})

	if r.Method != http.MethodGet {
		logger.Warn("Method Not Allowed")
		respond(ctx, w, http.StatusMethodNotAllowed, StandardResponse{
			Status:  "error",
			Message: "Method Not Allowed",
		})
		return
	}

	stats := core.GetCurrentPoolStats()
	logger.Info("Pool stats retrieved")
	respond(ctx, w, http.StatusOK, StandardResponse{
		Status: "success",
		Data:   stats,
	})
}

func TestPayoutHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := utils.GetRequestID(ctx)

	logger := logs.WithFields(map[string]interface{}{
		"request_id": reqID,
		"path":       r.URL.Path,
		"method":     r.Method,
	})

	if r.Method != http.MethodPost {
		logger.Warn("Method Not Allowed")
		respond(ctx, w, http.StatusMethodNotAllowed, StandardResponse{
			Status:  "error",
			Message: "Method Not Allowed",
		})
		return
	}

	if paymentClient == nil {
		logger.Error("Payment client not initialized")
		respond(ctx, w, http.StatusInternalServerError, StandardResponse{
			Status:  "error",
			Message: "Payment client not initialized",
		})
		return
	}

	var req PayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Invalid JSON structure")
		respond(ctx, w, http.StatusBadRequest, StandardResponse{
			Status:  "error",
			Message: "Invalid JSON structure",
		})
		return
	}

	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		logger.Warn("Invalid amount format")
		respond(ctx, w, http.StatusBadRequest, StandardResponse{
			Status:  "error",
			Message: "Invalid amount format",
		})
		return
	}

	txHash, err := paymentClient.SendReward(req.To, amount)
	if err != nil {
		logger.Error("Payout transaction failed: " + err.Error())
		respond(ctx, w, http.StatusInternalServerError, StandardResponse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	logger.Info("Payout transaction sent successfully")
	respond(ctx, w, http.StatusOK, StandardResponse{
		Status: "success",
		Data:   map[string]string{"txHash": txHash},
	})
}
