package handler_v1

import (
	"encoding/json"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"math/rand"
	"net/http"
	"time"
)

type Handler struct {
	temporalClient   client.Client
	temporalWorkflow Workflow
	config           Config
}

type Workflow struct {
}

type Order struct {
	ID        uuid.UUID
	ProductID uuid.UUID
	Quantity  int
}

// WorkflowVersionOverrideFlagValue represents a single configuration for overriding a workflow's version.
// - OverriddenVersion: Specifies the version of the workflow logic to use.
// - RolloutWeight: A relative weight used to determine the probability of selecting this version.
type WorkflowVersionOverrideFlagValue struct {
	OverriddenVersion int `json:"overriddenVersion"` // Version of the workflow logic
	RolloutWeight     int `json:"rolloutWeight"`     // Relative weight (e.g., 10, 20, etc.)
}

// WorkflowVersionOverrideFlags is a map where the key is the workflow flag name which represent the workflow changeID, and the value is
// a slice of WorkflowVersionOverrideFlagValue defining the version rollout configuration.
type WorkflowVersionOverrideFlags map[string][]WorkflowVersionOverrideFlagValue

// Config holds the application configuration, including workflow version override flag settings.
// - WorkflowVersionOverrideFlags: A map of feature flag names to their respective rollout configurations.
// - Other fields can be added for additional application configurations.
type Config struct {
	WorkflowVersionOverrideFlags WorkflowVersionOverrideFlags `json:"WorkflowVersionOverrideFlags"`
	// Add other configuration fields here as needed
}

// RandomizeWorkflowVersionOverride processes all version override flags and returns a map with selected versions.
func (f WorkflowVersionOverrideFlags) RandomizeWorkflowVersionOverride() map[string]int {
	results := make(map[string]int)

	for featureFlagName, values := range f {
		totalWeight := 0
		for _, value := range values {
			totalWeight += value.RolloutWeight
		}

		// Skip flags with no valid weights
		if totalWeight == 0 {
			continue
		}

		// Randomize selection based on weights
		rand.Seed(time.Now().UnixNano())
		randomPick := rand.Intn(totalWeight)

		cumulativeWeight := 0
		for _, value := range values {
			cumulativeWeight += value.RolloutWeight
			if randomPick < cumulativeWeight {
				results[featureFlagName] = value.OverriddenVersion
				break
			}
		}
	}

	return results
}

// OrderWorkflow processes an order using versioned workflow logic.
// It uses `workflow.GetVersion` to handle versioning and override logic.
func (w *Workflow) OrderWorkflow(ctx workflow.Context, versionOverride map[string]int, order Order) error {
	logger := workflow.GetLogger(ctx)

	// Log the start of the workflow with order details
	logger.Info("Starting OrderWorkflow",
		"order_id", order.ID,
		"product_id", order.ProductID,
		"quantity", order.Quantity,
	)

	// Process the order with the logic v2
	if err := w.ProcessOrderV2(ctx, order); err != nil {
		logger.Error("Failed to process order", "order_id", order.ID, "error", err)
		return err
	}

	// Log successful completion
	logger.Info("Order processed successfully", "order_id", order.ID)
	return nil
}

func (*Workflow) ProcessOrder(ctx workflow.Context, order Order) error {
	return nil
}

func (*Workflow) ProcessOrderV2(ctx workflow.Context, order Order) error {
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

	workflowRun, err := h.temporalClient.ExecuteWorkflow(
		r.Context(),
		options,
		h.temporalWorkflow.OrderWorkflow,
		h.config.WorkflowVersionOverrideFlags.RandomizeWorkflowVersionOverride(),
		order,
	)
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
