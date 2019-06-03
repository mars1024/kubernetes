package v1

const (
	// Component Name, such as apiserver, etcd, scheduler, etc.
	LabelComponentName = "cluster.kok.sigma.alipay.com/component-name"

	// App Name of some Component Name, such as etcd1.
	LabelAppName = "cluster.kok.sigma.alipay.com/app-name"

	// Version of kok-operator processed the Cluster CR
	LabelKokVersion = "cluster.kok.sigma.alipay.com/kok-version"
)

const (
	AnnotationLastUpdateUsername = "cluster.kok.sigma.alipay.com/last-update-username"

	// Cluster Name, used in Namespace only. 用于标记这个Namespace是属于某个Cluster的，否则都会忽略
	AnnotationClusterName = "cluster.kok.sigma.alipay.com/name"

	// 集群目前etcd的实例名
	AnnotationEtcdNames = "cluster.kok.sigma.alipay.com/etcd-names"

	// PKI Secret证书过期时间
	AnnotationPKIExpireTime = "cluster.kok.sigma.alipay.com/pki-expire-time"

	// Master Component 模板版本
	AnnotationMasterComponentTemplateVersion = "cluster.kok.sigma.alipay.com/master-component-template-version"

	// Deployment 或者 Pod 引用的 PKI Secret列表， ","隔开
	AnnotationPKISecretReference = "cluster.kok.sigma.alipay.com/pki-secrets"

	// Secret PKI reference hash
	AnnotationVersionSecret = "cluster.kok.sigma.alipay.com/pki-secret-hash"

	// Custom Component 发布文件的hash
	AnnotationCustomComponentHashLayout = "cluster.kok.sigma.alipay.com/custom-component-%s-hash"

	// Master Component 发布文件的hash
	AnnotationMasterComponentHashLayout = "cluster.kok.sigma.alipay.com/master-component-%s-hash"
)
