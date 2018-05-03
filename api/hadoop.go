package api

import (
	"net/http"
	"os"
	"strings"

	"fmt"

	"github.com/zonesan/clog"
)

type RemoteURI interface {
	URI() (string, error)
}

type Hadoop struct {
	ServiceDefault
	BaseURL string
	Params  interface{}
}

var hadoopBaseURL string
var servicesStr string

func (h *Hadoop) UsageAmount(svc string, bsi *BackingServiceInstance, req *http.Request) (*svcAmountList, error) {
	// uri := fmt.Sprintf("%s/%s/%s", h.BaseURL, svc, bsi.Spec.InstanceID)
	uri, err := h.GetRequestURI(svc, bsi)
	if err != nil {
		clog.Error(err)
		return nil, err
	}

	amounts, err := h.getAmountFromRemote(hadoopBaseURL+uri, req)
	if err != nil {
		clog.Error(err)
	}
	return amounts, err

	// amounts := &svcAmountList{Items: []svcAmount{
	// 	{Name: "RegionsQuota", Used: "300", Size: "500"},
	// 	{Name: "TablesQuotaa", Used: "20", Size: "100", Desc: "HBase命名空间的表数目"},
	// 	{Name: svc, Used: bsi.Spec.BackingServiceName, Desc: "faked response from hadoop."}}}

	// return amounts
}

func (h *Hadoop) getAmountFromRemote(uri string, r *http.Request) (*svcAmountList, error) {
	result := new(svcAmountList)
	err := doRequest("GET", uri, nil, result, "", r.Header)
	return result, err
}

func (h *Hadoop) GetRequestURI(svc string, bsi *BackingServiceInstance) (string, error) {
	var remote RemoteURI

	oc := DFClient()
	svc1, err1 := oc.GetBackingServices("openshift", svc)
	clog.Trace(err1)
	svcType := strings.ToLower(svc1.Spec.BackingServiceSpecMetadata.Type)

	switch svcType {
	case "spark", "mapreduce":
		// on async mode we need to bind instance first.

		// for _, binding := range bsi.Spec.Binding {
		// 	if len(binding.BindHadoopUser) > 0 {
		// 		remote = &yarnQueue{cred: binding.Credentials, svc: svc}
		// 		break
		// 	}
		// }
		// if remote == nil {
		// 	return "", fmt.Errorf("%s %s is not bound yet", svc, bsi.Name)
		// }
		remote = &yarnQueue{cred: bsi.Spec.Creds, svc: svc, svcType: svcType}
	case "hdfs":
		remote = &hdfsPath{cred: bsi.Spec.Creds, svc: svc, svcType: svcType}
	case "hive":
		remote = &hiveDB{cred: bsi.Spec.Creds, svc: svc, svcType: svcType}
	case "hbase":
		remote = &hbaseNS{cred: bsi.Spec.Creds, svc: svc, svcType: svcType}
	case "kafka":
		remote = &kafkaTopic{cred: bsi.Spec.Creds, svc: svc, svcType: svcType}
	case "mongodb", "greenplum":
		remote = &dbName{cred: bsi.Spec.Creds, svc: svc}
	default:
		return "", fmt.Errorf("unknown service '%v' or not supported", svc)
	}

	return remote.URI()
}

type yarnQueue struct {
	cred    map[string]string
	svc     string
	svcType string
}

// since queue is start with root., we remove root. prefix by using queue[5:]

func (yarn *yarnQueue) URI() (uri string, err error) {
	queue, ok := yarn.cred["Yarn_Queue"]
	if !ok {
		return "", fmt.Errorf("Yarn_Queue value is empty.")
	}

	uri = fmt.Sprintf("/%s/%s?service=%s", yarn.svcType, queue[5:], strings.ToLower(yarn.svc))
	return
}

type hdfsPath struct {
	cred    map[string]string
	svc     string
	svcType string
}

func (hdfs *hdfsPath) URI() (uri string, err error) {
	path, ok := hdfs.cred["HDFS_Path"]
	if !ok {
		return "", fmt.Errorf("HDFS_Path value is empty")
	}
	uri = fmt.Sprintf("/%s?service=%s&path=%s", hdfs.svcType, strings.ToLower(hdfs.svc), path)
	return uri, nil
}

type dbName struct {
	cred map[string]string
	svc  string
}

func (db *dbName) URI() (uri string, err error) {
	name, ok := db.cred["name"]
	if !ok {
		return "", fmt.Errorf("%v db name is empty", db.svc)
	}
	uri = fmt.Sprintf("/%s/%s", db.svc, name)
	return uri, nil
}

type hiveDB struct {
	cred    map[string]string
	svc     string
	svcType string
}

func (hive *hiveDB) URI() (uri string, err error) {
	hivedb, ok := hive.cred["Hive_Database"]

	if !ok {
		return "", fmt.Errorf("%v Hive_Database value is empty", hive.svc)
	}

	uri = fmt.Sprintf("/%s/%s?service=%s", hive.svcType, hivedb, strings.ToLower(hive.svc))

	return
}

type hbaseNS struct {
	cred    map[string]string
	svc     string
	svcType string
}

func (hbase *hbaseNS) URI() (uri string, err error) {
	ns, ok := hbase.cred["HBase_NameSpace"]
	if !ok {
		return "", fmt.Errorf("%v HBase_NameSpace is empty", hbase.svc)
	}

	uri = fmt.Sprintf("/%s/%s?service=%s", hbase.svcType, ns, strings.ToLower(hbase.svc))
	return uri, nil
}

type kafkaTopic struct {
	cred    map[string]string
	svc     string
	svcType string
}

func (kafka *kafkaTopic) URI() (uri string, err error) {
	topic, ok := kafka.cred["topic"]
	if !ok {
		return "", fmt.Errorf("%v topic is empty", kafka.svc)
	}

	uri = fmt.Sprintf("/%s/%s?service=%s", kafka.svcType, topic, strings.ToLower(kafka.svc))
	return uri, nil
}

func init() {

	hadoopBaseURL = os.Getenv("HADOOP_AMOUNT_BASEURL")
	if len(hadoopBaseURL) == 0 {
		clog.Fatal("HADOOP_AMOUNT_BASEURL must be specified.")
	}
	hadoopBaseURL = httpsAddr(hadoopBaseURL)
	clog.Debug("hadoop amount base url:", hadoopBaseURL)

	servicesStr = os.Getenv("OCDP_SERVICES_LIST")
	// remove the spaces in the env service list
	// hdfs, hbase, hive, mapreduce, spark, kafka, hdfs_cluster2, hbase_cluster2,
	// hive_cluster2, mapreduce_cluster2, spark_cluster2, kafka_cluster2
	services := strings.Split(strings.Replace(servicesStr, " ", "", -1), ",")
	hadoop := &Hadoop{BaseURL: hadoopBaseURL}
	register("hadoop", services, hadoop)

	// since hadoop and rds is the same api.
	hdpservices := []string{"mongodb", "greenplum"}
	register("rds", hdpservices, hadoop)
}
