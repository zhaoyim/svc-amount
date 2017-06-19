package api

type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason,omitempty"`
	status  int    `json:"status,omitempty"`
	//Data    interface{} `json:"data,omitempty"`
}

type svcAmount struct {
	Name      string `json:"name"`
	Used      string `json:"used"`
	Size      string `json:"size,omitempty"`
	Available string `json:"available,omitempty"`
	Desc      string `json:"desc,omitempty"`
}

type svcAmountList struct {
	Items []svcAmount `json:"items"`
}

var (
	rds       = []string{"mongodb", "greenplum"}
	hadoop    = []string{"hbase", "hive", "hdfs", "kafka", "spark"}
	container = []string{"neo4j", "rabbitmq"}
	services  = map[string][]string{
		"rds":       []string{"mongodb", "greenplum"},
		"hadoop":    []string{"hbase", "hive", "hdfs", "kafka", "spark"},
		"container": []string{"neo4j", "rabbitmq"}}
)

type svcUsageAmount interface {
	UsageAmount(params interface{}) *svcAmountList
}
