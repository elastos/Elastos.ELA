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

// TestEnforceCrossChainUTXORestrictionHeightsIgnoresLocalOverrides verifies
// every supported network receives its coordinated consensus values.
func TestEnforceCrossChainUTXORestrictionHeightsIgnoresLocalOverrides(t *testing.T) {
	testCases := []struct {
		name                string
		activeNet           string
		expectedFreeze      uint32
		expectedRestriction uint32
	}{
		{
			name:                "mainnet",
			expectedFreeze:      config.MainNetCrossChainUTXOFreezeHeight,
			expectedRestriction: config.MainNetCrossChainUTXORestrictionHeight,
		},
		{
			name:                "named mainnet",
			activeNet:           "mainnet",
			expectedFreeze:      config.MainNetCrossChainUTXOFreezeHeight,
			expectedRestriction: config.MainNetCrossChainUTXORestrictionHeight,
		},
		{
			name:                "testnet",
			activeNet:           "testnet",
			expectedFreeze:      config.DisabledCrossChainUTXORestrictionHeight,
			expectedRestriction: config.DisabledCrossChainUTXORestrictionHeight,
		},
		{
			name:                "regnet",
			activeNet:           "regnet",
			expectedFreeze:      config.DisabledCrossChainUTXORestrictionHeight,
			expectedRestriction: config.DisabledCrossChainUTXORestrictionHeight,
		},
		{
			name:                "unknown network",
			activeNet:           "private-net",
			expectedFreeze:      config.DisabledCrossChainUTXORestrictionHeight,
			expectedRestriction: config.DisabledCrossChainUTXORestrictionHeight,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			configuration := config.GetDefaultParams()
			configuration.ActiveNet = testCase.activeNet
			configuration.CrossChainUTXOFreezeHeight = 0
			configuration.CrossChainUTXORestrictionHeight = 0

			enforceCrossChainUTXORestrictionHeights(configuration)

			assert.Equal(t, testCase.expectedFreeze,
				configuration.CrossChainUTXOFreezeHeight)
			assert.Equal(t, testCase.expectedRestriction,
				configuration.CrossChainUTXORestrictionHeight)
			if testCase.expectedFreeze != config.DisabledCrossChainUTXORestrictionHeight {
				assert.Less(t, configuration.CrossChainUTXOFreezeHeight,
					configuration.CrossChainUTXORestrictionHeight)
			}
		})
	}
}

// TestSetupConfigIgnoresCrossChainUTXOHeightOverrides verifies config-file
// values cannot alter either coordinated mainnet consensus height.
func TestSetupConfigIgnoresCrossChainUTXOHeightOverrides(t *testing.T) {
	originalDefaultParams := config.DefaultParams
	originalParameters := config.Parameters
	t.Cleanup(func() {
		config.DefaultParams = originalDefaultParams
		config.Parameters = originalParameters
	})

	configPath := filepath.Join(t.TempDir(), "config.json")
	configContents := []byte(`{
  "Configuration": {
    "CrossChainUTXOFreezeHeight": 0,
    "CrossChainUTXORestrictionHeight": 0
  }
}`)
	assert.NoError(t, os.WriteFile(configPath, configContents, 0o600))

	config.DefaultParams = *config.GetDefaultParams()
	config.DefaultParams.Conf = configPath

	configuration := NewSettings().SetupConfig(false, "", "")

	assert.Equal(t, config.MainNetCrossChainUTXOFreezeHeight,
		configuration.CrossChainUTXOFreezeHeight)
	assert.Equal(t, config.MainNetCrossChainUTXORestrictionHeight,
		configuration.CrossChainUTXORestrictionHeight)
}
