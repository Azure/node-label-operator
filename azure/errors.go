package azure

import (
	"github.com/Azure/go-autorest/autorest"
)

func IsNotFound(err error) bool {
	if derr, ok := err.(autorest.DetailedError); ok && derr.StatusCode == 404 {
		return true
	}
	return false
}
