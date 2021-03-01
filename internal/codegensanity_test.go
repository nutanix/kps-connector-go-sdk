// Copyright (c) 2021 Nutanix, Inc.
package internal

import (
	"testing"

	"github.com/nutanix/kps-connector-go-sdk/connector/v1"
)

func TestCodeGenSanity(t *testing.T) {
	// Force 'init' call on connector/v1/connector.pb.go to ensure init does not panic
	// in generated code
	t.Run("make sure init does not panic", func(t *testing.T) {
		_ = connector.Status{}
	})
}
