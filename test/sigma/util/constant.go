package util

var (
	// TestDataDir test data dir path, which contains sigma test files.
	TestDataDir string

	// SigmaPauseImage is the pause image for sigma 2e2 test,
	// use a private pause image for speed and security reason.
	SigmaPauseImage string

	// for ant e2e use
	AlipayCertPath       string
	AlipayAdapterAddress string
	ArmoryUser           string
	ArmoryKey            string
)
