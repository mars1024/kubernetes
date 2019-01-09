package ase

var AseManagedClusterConfigMapName = "ase-managed-cluster"
var AseLogAgentContainerName = "ase-log-agent"

// logTail Env
var AliyunLogtailUserId = "ALIYUN_LOGTAIL_USER_ID"
var AliyunLogtailUserDefinedId = "ALIYUN_LOGTAIL_USER_DEFINED_ID"
var AliyunLogtailConfig = "ALIYUN_LOGTAIL_CONFIG"

var EnvPodName = "POD_NAME"
var EnvVarSignedUrl = "ASE_SIGNED_FILE_DOWNLOAD_URL"
var EnvAseSystemUrl = "ASE_SYSTEM_URL"

var InteropFasLogsRootPath = "/home/admin/fas-logs/"
var InteropFasLogsOnFasAgent = "on-fasagent"

var getSignedDownloadUrlPath = "/privateapi/ase/v1/filestorage/getSignedDownloadUrl"

var SpecialConfigMapNames = []string{AseManagedClusterConfigMapName, AseLogAgentContainerName}

func IsSpecialConfigMap(name string) bool {
	for _, val := range SpecialConfigMapNames {
		if name == val {
			return true
		}
	}
	return false
}
