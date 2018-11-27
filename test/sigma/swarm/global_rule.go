package swarm

import (
	"encoding/json"
	"fmt"
	"time"
)

// AddP0M0Rules add p0m0 rules
func AddP0M0Rules(key string, rule string) error {
	return etcdPut(key, rule)
}

// PutMonoAPPRule put app rules for mono app.
func PutMonoAPPDURule(appname, du string) error {
	for _, v := range Site {
		key := fmt.Sprintf("/applications/schedulerules/globalrules/%s", v)
		globalRule := &GlobalRules{
			Monopolize: MonopolizeDecs{
				AppConstraints: []string{appname},
				DUConstraints:  []string{du},
			},
		}
		buf, _ := json.Marshal(globalRule)
		err := etcdPut(key, string(buf))
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateSigmaGlobalConfig update global rules
func UpdateSigmaGlobalConfig(globalRule *GlobalRules) error {
	for _, v := range Site {
		err := EtcdPut(fmt.Sprintf("/applications/schedulerules/globalrules/%s", v), globalRule)
		if err != nil {
			return err
		}
	}
	return nil
}

// RemoveSigmaGlobal put app rules for mono app.
func RemoveSigmaGlobal() error {
	// globalRule CAN NOT be deleted, scheduler process MUST NEED it.
	globalRule := &GlobalRules{
		UpdateTime: time.Now().Format(time.RFC3339),
	}
	UpdateSigmaGlobalConfig(globalRule)
	return nil
}
