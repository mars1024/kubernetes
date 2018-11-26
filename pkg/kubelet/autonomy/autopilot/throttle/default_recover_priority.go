package throttle

// DefaultRecoverPriority just select the index 0 return.
type DefaultRecoverPriority struct{}

var _ ContainerThrottlePriority = new(DefaultRecoverPriority)

// SelectCouldThrottleContainer return the first element.
func (p *DefaultRecoverPriority) SelectCouldThrottleContainer(inputData InputData, containerUIDs []*ParamRef) (*ParamRef, error) {
	if len(containerUIDs) > 0 {
		return containerUIDs[0], nil
	}
	return nil, nil
}
