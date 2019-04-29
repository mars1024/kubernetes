package api

const (
	// EnvIsSidecar used to indicates this is a sidecar container
	EnvIsSidecar = "IS_SIDECAR"
	// EnvIgnoreReady used to indicates should ignore this container ready when generate container ready condition
	EnvIgnoreReady = "SIGMA_IGNORE_READY"
)
