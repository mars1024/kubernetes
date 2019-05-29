/*
Copyright 2018 The Alipay Authors.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NotificationTemplate describes how to generate notifications for specific alerts.
type NotificationTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotificationTemplateSpec   `json:"spec"`
	Status NotificationTemplateStatus `json:"status"`
}

// NotificationTemplateSpec defines the template of a kind of alerts.
type NotificationTemplateSpec struct {
	AlertName string `json:"alertName"`
	Template  string `json:"template"`
}

// NotificationTemplateStatus describes whether or not the template is accepted.
type NotificationTemplateStatus struct {
	// Accepted checks if the template is accepted.
	Accepted bool `json:"accepted"`
	// Reason explain why the template is unacceptable.
	Reason string `json:"reason"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NotificationTemplateList is a list of NotificationTemplate objects.
type NotificationTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items individual CustomResourceDefinitions
	Items []NotificationTemplate `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NotificationReceiver describes a notification receiver.
type NotificationReceiver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NotificationReceiverSpec `json:"spec"`
}

// NotificationReceiverSpec describes the information of a receiver.
type NotificationReceiverSpec struct {
	// UID is the unique id of the receiver.
	UID string `json:"uid"`
	// Empno is the employee number.
	// Required.
	Empno string `json:"empno"`
	// Name is the real name of the receiver.
	Name string `json:"name"`
	// NickName is the nick name of the receiver.
	NickName string `json:"nickName"`
	// PhoneNumber is the phone number of the receiver.
	// Required.
	PhoneNumber string `json:"phoneNumber"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NotificationReceiverList is a list of NotificationReceiver objects.
type NotificationReceiverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items individual CustomResourceDefinitions
	Items []NotificationReceiver `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NotificationChannel defines a notification channel derived from existing channels.
type NotificationChannel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotificationChannelSpec   `json:"spec"`
	Status NotificationChannelStatus `json:"status"`
}

// NotificationChannelSpec defines a notification channel.
type NotificationChannelSpec struct {
	// LayoutTemplate is the root template of all alert templates.
	LayoutTemplate string `json:"layoutTemplate"`
	// DefaultTemplate is the default template for the alerts which have no corresponding
	// notification template.
	DefaultTemplate string `json:"defaultTemplate"`
	// TemplateSelector selects notification templates for this channel.
	// If this field is nil, it selects no templates.
	// If this field is zero value, it selects all templates.
	TemplateSelector *metav1.LabelSelector `json:"templateSelector"`

	// The following targets are mutually exclusive. Only one can be set.
	// If more than one target are filled in, select the one which has the highest priority.
	// Priority:
	// DingTalkTarget > SMSTarget > PhonecallTarget

	// DingTalk contains the information to send maseeages via dingtalk.
	DingTalk *DingTalkTarget `json:"dingtalk,omitempty"`
	// SMS contains the information to send messages via SMS service.
	SMS *SMSTarget `json:"sms,omitempty"`
	// Phonecall contains the information to send messages via phone call.
	Phonecall *PhonecallTarget `json:"phonecall,omitempty"`
}

// DingTalkTarget describes a dingtalk target sender.
type DingTalkTarget struct {
	// Token is a dingtalk robot token.
	Token string `json:"token"`
}

// SMSTarget describes a sms target sender.
type SMSTarget struct {
	// UserName of GOC authrization.
	UserName string `json:"username"`
	// Code of GOC authrization.
	Code string `json:"code"`
	// IDPrefix prepends a string to every message ID.
	IDPrefix string `json:"idPrefix"`
}

// PhonecallTarget describes a phone call target sender.
type PhonecallTarget struct {
	// UserName of GOC authrization.
	UserName string `json:"username"`
	// Code of GOC authrization.
	Code string `json:"code"`
	// Brief is a format string for generating brief phonecall.
	// e.g. Please read sms with number %s
	// The %s is the number of sms.
	Brief string `json:"brief"`
	// IDPrefix prepends a string to every message ID.
	IDPrefix string `json:"idPrefix"`
}

// NotificationChannelStatus describes whether or not the channel is accepted.
type NotificationChannelStatus struct {
	// Accepted checks if the template is accepted.
	Accepted bool `json:"accepted"`
	// Reason explain why the template is unacceptable.
	Reason string `json:"reason"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NotificationChannelList is a list of NotificationChannel objects.
type NotificationChannelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items individual CustomResourceDefinitions
	Items []NotificationChannel `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NotificationGroup binds receivers and a notification channel.
type NotificationGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotificationGroupSpec   `json:"spec"`
	Status NotificationGroupStatus `json:"status"`
}

// NotificationGroupSpec defines the relationships between receivers and a channel.
type NotificationGroupSpec struct {
	Channel   string         `json:"channel"`
	Receivers GroupReceivers `json:"receivers"`
	Alerts    GroupAlerts    `json:"alerts"`
}

// GroupReceivers contains all receivers for a group.
// The result of this struct is the union set of the two fields.
// If a receiver name is not defined, it will be ignored.
type GroupReceivers struct {
	// Names is an array of exact receiver names.
	Names []string `json:"names"`
	// Selector selects receivers by labels.
	// If this field is nil, it selects no receivers.
	// If this field is zero value, it selects all receivers.
	Selector *metav1.LabelSelector `json:"selector"`
}

// GroupAlerts contains all alerts for a group.
// The result of this struct is the union set of the two fields.
// If an alert name is not defined, it will be ignored.
type GroupAlerts struct {
	// Names is an array of exact alert names.
	Names []string `json:"names"`
	// Regexes is an array of regexes to match alerts.
	Regexes []string `json:"regexes"`
}

// NotificationGroupStatus describes
type NotificationGroupStatus struct {
	// Receivers shows all valid receivers.
	Receivers []string `json:"receivers"`
	// Accepted checks if the template is accepted.
	Accepted bool `json:"accepted"`
	// Reason explain why the template is unacceptable.
	Reason string `json:"reason"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NotificationGroupList is a list of NotificationGroup objects.
type NotificationGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items individual CustomResourceDefinitions
	Items []NotificationGroup `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterScrapeConfig defines common target components of all clusters.
// ClusterScrapeConfig shares common spec with ScrapeConfig.
// All namespace settings in `Spec` are ignored.
// Field `Spec.Selector.Hosts` is ignored.
type ClusterScrapeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScrapeConfigSpec   `json:"spec"`
	Status ScrapeConfigStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterScrapeConfigList is a list of ClusterScrapeConfig objects.
type ClusterScrapeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items individual CustomResourceDefinitions
	Items []ClusterScrapeConfig `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScrapeConfig contains configs for scraping metrics from remote targets.
type ScrapeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScrapeConfigSpec   `json:"spec"`
	Status ScrapeConfigStatus `json:"status"`
}

// ScrapeConfigSpec defines target infomations.
type ScrapeConfigSpec struct {
	// Interval is scraping interval.
	Interval metav1.Duration `json:"interval,omitempty"`
	// Timeout is the longest time for a scrape request.
	Timeout metav1.Duration `json:"timeout,omitempty"`
	// Port is listening port of hosts.
	Port uint16 `json:"port"`
	// MetricsPath defines the URL path of targets.
	MetricsPath string `json:"metricsPath,omitempty"`
	// TLSConfig contains configs for HTTPS.
	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`
	// Authorization supports bearer token and basic auth.
	Authorization *Authorization `json:"authorization,omitempty"`
	// Selector selects a list of hosts with same config.
	Selector Selector `json:"selector"`
	// Labels to add or overwrite for each metric scraped from hosts.
	Labels map[string]string `json:"labels,omitempty"`
}

// ScrapeConfigStatus describes whether or not the scrape config is accepted.
type ScrapeConfigStatus struct {
	// Accepted checks if the scrape config is accepted.
	Accepted bool `json:"accepted"`
	// Reason explain why the scrape config is unacceptable.
	Reason string `json:"reason,omitempty"`
}

// TLSConfig describes TLS config for target servers.
type TLSConfig struct {
	// ServerName is the domain name of target server.
	ServerName string `json:"serverName,omitempty"`
	// Insecure indicates that the target server is insecure and
	// scraper should skip cert verification. If this field is true,
	// CA should be empty.
	Insecure bool `json:"insecure"`
	// ClientCert is client certificate and private key.
	// If this is needed, the secret must have keys named `tls.crt` and `tls.key`.
	ClientCert *Secret `json:"clientCert,omitempty"`
	// CACert is CA certificate.
	// If this is needed, the secret must have a key named `tls.ca`.
	CACert *Secret `json:"caCert,omitempty"`
}

// Authorization describes authorization info for target servers.
type Authorization struct {
	// BearerToken is bearer token of OAuth.
	// If this is needed, the secret must have a key named `token`.
	BearerToken *Secret `json:"bearerToken,omitempty"`
	// BasicAuth is basic auth of HTTP.
	// If this is needed, the secret must have keys named `username` and `password`.
	BasicAuth *Secret `json:"basicAuth,omitempty"`
}

// Secret keys.
const (
	SecretKeyTLSKey         = "tls.key"
	SecretKeyTLSCertificate = "tls.crt"
	SecretKeyTLSCA          = "tls.ca"
	SecretKeyToken          = "token"
	SecretKeyUsername       = "username"
	SecretKeyPassword       = "password"
)

// Secret is a kubernetes secret.
type Secret struct {
	// Name is the name of this secret.
	Name string `json:"name"`
	// KeyMap is a mapping of needed keys and real secret keys.
	KeyMap map[string]string `json:"keyMap,omitempty"`
}

// Selector selects hosts for scraping metrics.
type Selector struct {
	// Hosts is a list of IPs of hosts.
	Hosts []string `json:"hosts,omitempty"`
	// Match is a selector for selecting pods from kuernetes.
	Match map[string]string `json:"match,omitempty"`
}

// BasicAuth contains username and password of basic auth of HTTP.
type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScrapeConfigList is a list of ScrapeConfig objects.
type ScrapeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items individual CustomResourceDefinitions
	Items []ScrapeConfig `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MonitoringRule describes rules of records and alerts.
type MonitoringRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MonitoringRuleSpec   `json:"spec"`
	Status MonitoringRuleStatus `json:"status"`
}

// MonitoringRuleSpec contains a set of rules to calculate records and
// trigger alerts. All records are evaluated before than alerts.
type MonitoringRuleSpec struct {
	// Interval is evaluation interval.
	Interval metav1.Duration `json:"interval"`
	// Records is a list of record targets for generating new time
	// serials. Rules are evaluated one by one.
	Records []RecordTarget `json:"records,omitempty"`
	// Alerts is a list of alert targets for triggering alerts.
	Alerts []AlertTarget `json:"alerts,omitempty"`
}

// MonitoringRuleStatus describes whether or not the rule is accepted.
type MonitoringRuleStatus struct {
	// Accepted checks if the rule is accepted.
	Accepted bool `json:"accepted"`
	// Reason explain why the rule is unacceptable.
	Reason string `json:"reason,omitempty"`
}

// RecordTarget describes a record target.
type RecordTarget struct {
	// Name is the name of records which are generated by this target.
	Name string `json:"name"`
	// Expr is the expression to evaluate.
	Expr string `json:"expr"`
	// Labels to add or overwrite before storing the result.
	Labels map[string]string `json:"labels,omitempty"`
}

// AlertTarget describes a alert target.
type AlertTarget struct {

	// Name is the name of alerts which are triggered by this target.
	Name string `json:"name"`
	// For indicates that the alert is firing if this target has been triggered for this long.
	For metav1.Duration `json:"for"`
	// Expr is the expression  to evaluate.
	Expr string `json:"expr"`
	// Labels to add or overwrite for each alert.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to add to each alert.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MonitoringRuleList is a list of MonitoringRule objects.
type MonitoringRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items individual CustomResourceDefinitions
	Items []MonitoringRule `json:"items"`
}
