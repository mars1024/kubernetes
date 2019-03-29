package antitamper

import (
	"github.com/golang/glog"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/client-go/listers/core/v1"
	"strings"
)

// IsMultiTenancyWiseAdmin judges if a user is privileged as a global admin
func IsMultiTenancyWiseAdmin(username string) bool {
	switch username {
	case "system:admin", "kubeapiserver":
		return true
	case "system:kube-scheduler", "system:kube-controller-manager", "system:apiserver":
		return true
	default:
		return false
	}
}

func doFilter(input []string, filterRules []string) []string {
	var passed []string
	for _, item := range input {
		if passesAnyRule(item, filterRules) {
			passed = append(passed, item)
		}
	}
	return passed
}

func passesAnyRule(item string, filterRules []string) bool {
	for _, rule := range filterRules {
		if matchesRule(item, rule) {
			return true
		}
	}
	return false
}

func matchesRule(item string, rule string) bool {
	if rule == "*" {
		return true
	}
	if strings.HasSuffix(rule, "*") {
		return strings.HasPrefix(item, rule[:len(rule)-1])
	}
	if strings.HasPrefix(rule, "*") {
		return strings.HasSuffix(item, rule[1:])
	}
	return item == rule
}

func getChangedKeys(a map[string]string, b map[string]string) []string {
	var changedKeysMap = make(map[string]bool)
	for key := range a {
		if a[key] != b[key] {
			changedKeysMap[key] = true
		}
	}
	for key := range b {
		if a[key] != b[key] {
			changedKeysMap[key] = true
		}
	}

	var changedKeys []string
	for key := range changedKeysMap {
		changedKeys = append(changedKeys, key)
	}
	return changedKeys
}

func matchesResourceIdentifier(group string, version string, kind string, name string, namespace string, identifier ResourceIdentifier) bool {
	return (identifier.name == nil || matchesRule(name, *identifier.name)) &&
		(identifier.namespace == nil || matchesRule(namespace, *identifier.namespace)) &&
		(identifier.group == nil || matchesRule(group, *identifier.group)) &&
		(identifier.version == nil || matchesRule(version, *identifier.version)) &&
		(identifier.kind == nil || matchesRule(kind, *identifier.kind))
}

func verboseLogIfNecessary(message string, resourceName string) {
	if strings.HasPrefix(resourceName, "anti-tamper-test") {
		glog.V(4).Infof("[anti-tamper ac] %s: %s", resourceName, message)
	}
}

var any = "*"

func makeResourceIdentifier(name string, namespace string, group string, version string, kind string) ResourceIdentifier {
	resourceIdentifier := ResourceIdentifier{}
	if name != any {
		resourceIdentifier.name = &name
	}
	if namespace != any {
		resourceIdentifier.namespace = &namespace
	}
	if group != any {
		resourceIdentifier.group = &group
	}
	if version != any {
		resourceIdentifier.version = &version
	}
	if kind != any {
		resourceIdentifier.kind = &kind
	}
	return resourceIdentifier
}

func isTypeNamespace(group string, version string, kind string) bool {
	return group == "" && version == "v1" && kind == "Namespace"
}

type validateContext struct {
	attributes   admission.Attributes
	errorMessage string
	namespaceLister *v1.NamespaceLister
}