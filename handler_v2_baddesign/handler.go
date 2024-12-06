package handler_v2_baddesign

import (
	"encoding/json"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"net/http"
)

type Handler struct {
	temporalClient   client.Client
	temporalWorkflow Workflow
}

type Workflow struct {
}

type Order struct {
	ID        uuid.UUID
	ProductID uuid.UUID
	Quantity  int
}

// OrderWorkflow is the workflow definition that processes the order
// It now uses `workflow.GetVersion` to handle versioning
func (w *Workflow) OrderWorkflow(ctx workflow.Context, order Order) error {
	logger := workflow.GetLogger(ctx)

	// Define the versioning for the workflow
	version := workflow.GetVersion(ctx, "OrderWorkflowVersion", workflow.DefaultVersion, 2)

	// Log the order details and the version
	logger.Info("Starting OrderWorkflow", "order_id", order.ID, "product_id", order.ProductID, "quantity", order.Quantity, "version", version)

	// Execute the workflow based on the version
	switch version {
	case workflow.DefaultVersion:
		// Default version: Basic processing logic
		if err := w.ProcessOrder(ctx, order); err != nil {
			logger.Error("Failed to process order (default version)", "error", err)
			return err
		}
	case 2:
		// Version 2: Enhanced processing logic (for example, adding new features)
		if err := w.ProcessOrderV2(ctx, order); err != nil {
			logger.Error("Failed to process order (V2)", "error", err)
			return err
		}
	default:
		// Use NewContinueAsNewError to continue the workflow with a new version
		return workflow.NewContinueAsNewError(ctx, w.OrderWorkflow, order)
	}

	logger.Info("Order processed successfully", "order_id", order.ID, "version", version)
	return nil
}

func (w *Workflow) ProcessOrder(ctx workflow.Context, order Order) error {
	return nil
}

func (w *Workflow) ProcessOrderV2(ctx workflow.Context, order Order) error {
	return nil
}

func (h *Handler) OrderHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProductID uuid.UUID `json:"product_id"` // Expect product ID as part of the request body
		Quantity  int       `json:"quantity"`   // Expect quantity as part of the request body
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}

	// Generate a new UUID for the Order ID
	orderID := uuid.New()

	order := Order{
		ID:        orderID,       // Assign the generated UUID to the order ID
		ProductID: req.ProductID, // Use the provided product ID
		Quantity:  req.Quantity,  // Use the provided quantity
	}

	// Start the workflow with the generated UUID and the order data
	options := client.StartWorkflowOptions{
		TaskQueue: "OrderTaskQueue",
	}

	workflowRun, err := h.temporalClient.ExecuteWorkflow(r.Context(), options, h.temporalWorkflow.OrderWorkflow, order)
	if err != nil {
		http.Error(w, `{"error": "Failed to start workflow"}`, http.StatusInternalServerError)
		return
	}

	// Prepare the response
	response := struct {
		OrderID    string `json:"order_id"`
		RunID      string `json:"run_id"`
		WorkflowID string `json:"workflow_id"`
	}{
		OrderID:    orderID.String(),
		RunID:      workflowRun.GetRunID(),
		WorkflowID: workflowRun.GetID(),
	}

	// Send the response in JSON format
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, `{"error": "Failed to write response"}`, http.StatusInternalServerError)
	}
}
