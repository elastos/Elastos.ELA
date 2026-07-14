// Copyright (c) 2026 The Elastos Foundation
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/stretchr/testify/assert"
)

// TestEnforceCrossChainUTXORestrictionHeightIgnoresLocalOverrides verifies
// every supported network receives its coordinated consensus value.
func TestEnforceCrossChainUTXORestrictionHeightIgnoresLocalOverrides(t *testing.T) {
	testCases := []struct {
		name      string
		activeNet string
		expected  uint32
	}{
		{
			name:     "mainnet",
			expected: config.MainNetCrossChainUTXORestrictionHeight,
		},
		{
			name:      "named mainnet",
			activeNet: "mainnet",
			expected:  config.MainNetCrossChainUTXORestrictionHeight,
		},
		{
			name:      "testnet",
			activeNet: "testnet",
			expected:  config.DisabledCrossChainUTXORestrictionHeight,
		},
		{
			name:      "regnet",
			activeNet: "regnet",
			expected:  config.DisabledCrossChainUTXORestrictionHeight,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			configuration := config.GetDefaultParams()
			configuration.ActiveNet = testCase.activeNet
			configuration.CrossChainUTXORestrictionHeight = 0

			enforceCrossChainUTXORestrictionHeight(configuration)

			assert.Equal(t, testCase.expected,
				configuration.CrossChainUTXORestrictionHeight)
		})
	}
}

// TestSetupConfigIgnoresCrossChainUTXORestrictionHeightOverride verifies a
// config-file value cannot alter the mainnet consensus activation height.
func TestSetupConfigIgnoresCrossChainUTXORestrictionHeightOverride(t *testing.T) {
	originalDefaultParams := config.DefaultParams
	originalParameters := config.Parameters
	t.Cleanup(func() {
		config.DefaultParams = originalDefaultParams
		config.Parameters = originalParameters
	})

	configPath := filepath.Join(t.TempDir(), "config.json")
	configContents := []byte(`{
  "Configuration": {
    "CrossChainUTXORestrictionHeight": 0
  }
}`)
	assert.NoError(t, os.WriteFile(configPath, configContents, 0o600))

	config.DefaultParams = *config.GetDefaultParams()
	config.DefaultParams.Conf = configPath

	configuration := NewSettings().SetupConfig(false, "", "")

	assert.Equal(t, config.MainNetCrossChainUTXORestrictionHeight,
		configuration.CrossChainUTXORestrictionHeight)
}
