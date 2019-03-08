/*
Copyright 2019 The Alipay Authors. All Rights Reserved.
*/

package apis

const (
	// Node Mandatory Label for ant.
	LabelAppEnv       = MandatoryAlipayPrefix + "/app-env"
	LabelAppLogicPool = MandatoryAlipayPrefix + "/app-logic-pool"
	LabelServerOwner  = MandatoryAlipayPrefix + "/server-owner"
	LabelServerOps    = MandatoryAlipayPrefix + "/server-ops"

	// Node labels extended by Alipay.com.
	// Please read the [Common Node API]: https://yuque.antfin-inc.com/sigma.pouch/sigma3.x/tsmf32
	LabelUplinkHostname = MetaAlipayPrefix + "/uplink-hostname"
	LabelUplinkIP       = MetaAlipayPrefix + "/uplink-ip"
	LabelUplinkSN       = MetaAlipayPrefix + "/uplink-sn"
	LabelModel          = MetaAlipayPrefix + "/model"

	// LabelIsColocation is flag to identity whether this is colocation node.
	// Value is "true"/"flase".
	LabelIsColocation = MetaAlipayPrefix + "/is-colocation"

	// LabelResourceConfigName is the config name of resource for node.
	LabelResourceConfigName = MetaAlipayPrefix + "/resource-config-name"

	// LabelIDCManagerState indicates node state in armory. e.g. IdcManagerState=READY
	LabelIDCManagerState = MetaAlipayPrefix + "/idc-manager-state"
)

// NodeCondition. Too many vendor files if v1.NodeCondition is used, so we use string instead.
// do not forget type conversions.
const (
	// NodeConditionArmoryUnset indicates whether the Node Armory info is correctly Sync.
	NodeConditionArmoryUnset = "ArmoryUnset"

	// NodeConditionIPPressure indicates whether the Node IP is not sufficient.
	NodeConditionIPPressure = "IPPressure"
)

// Node Taint keys.
const (
	// NodeTaintArmoryUnset will be added when node's logical-info(Armory-info) has not been set
	NodeTaintArmoryUnset = NodeAlipayPrefix + "/armory-unset"

	// NodeTaintIPPressure will be added when node's IP Pool is not sufficient
	NodeTaintIPPressure = NodeAlipayPrefix + "/ip-pressure"

	// NodeTaintInitial will be added when node first created
	NodeTaintInitial = NodeAlipayPrefix + "/initial"

	// NodeTaintNotReady will be added when node has not reached the final state
	NodeTaintNotReady = NodeAlipayPrefix + "/not-ready"
)
