package resources

import (
	"errors"

	"github.com/dscof/terraform-provider-quartermaster/internal/client"
)

func isNotFound(err error) bool {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}
	return false
}

func mapAPIError(err error) string {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 400:
			return "Validation error: " + apiErr.Message
		case 404:
			return "Resource not found: " + apiErr.Message
		case 409:
			return "Conflict: " + apiErr.Message
		}
	}
	return err.Error()
}
