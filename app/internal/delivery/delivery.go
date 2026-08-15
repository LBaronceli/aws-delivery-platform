package delivery

import "slices"

// Status is the current stage of a delivery.
type Status string

const (
	StatusCreated   Status = "created"
	StatusScheduled Status = "scheduled"
	StatusPickedUp  Status = "picked_up"
	StatusInTransit Status = "in_transit"
	StatusDelivered Status = "delivered"
	StatusCancelled Status = "cancelled"
)

var validStatuses = []Status{
	StatusCreated,
	StatusScheduled,
	StatusPickedUp,
	StatusInTransit,
	StatusDelivered,
	StatusCancelled,
}

// Delivery represents a shipment between two locations.
type Delivery struct {
	ID          string `json:"id"`
	Pickup      string `json:"pickup"`
	Destination string `json:"destination"`
	Status      Status `json:"status"`
}

// Update contains the fields that can be changed on a delivery.
type Update struct {
	Pickup      *string
	Destination *string
	Status      *Status
}

// IsValidStatus reports whether status belongs to the delivery lifecycle.
func IsValidStatus(status Status) bool {
	return slices.Contains(validStatuses, status)
}
