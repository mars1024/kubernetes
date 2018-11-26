package throttle

// DefaultThrottlePriority choose which container to throttle.
type DefaultThrottlePriority struct{}

var _ ContainerThrottlePriority = new(DefaultThrottlePriority)

// SelectCouldThrottleContainer just return the biggest overratio container to return.
func (p *DefaultThrottlePriority) SelectCouldThrottleContainer(inputData InputData, containerUIDs []*ParamRef) (*ParamRef, error) {
	var maxOverRatio float64
	var tmp *ParamRef
	for _, param := range containerUIDs {
		spec, ok := param.SpecValues.(float64)
		if ok {
			if spec == 0 {
				continue
			}
		}
		ratio := param.CurrentValues.(float64) / spec
		if ratio > maxOverRatio {
			tmp = &ParamRef{Name: param.Name, SpecValues: param.SpecValues, CurrentValues: param.CurrentValues, HistoryValues: param.HistoryValues}
			maxOverRatio = ratio
		}
	}
	return tmp, nil
}
