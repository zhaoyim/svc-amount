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

	switch svc {
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
		remote = &yarnQueue{cred: bsi.Spec.Creds, svc: svc}
	case "mongodb", "greenplum":
		remote = &dbName{cred: bsi.Spec.Creds, svc: svc}
	case "hdfs":
		remote = &hdfsPath{cred: bsi.Spec.Creds, svc: svc}
	case "hive":
		remote = &hiveDB{cred: bsi.Spec.Creds, svc: svc}
	case "hbase":
		remote = &hbaseNS{cred: bsi.Spec.Creds, svc: svc}
	case "kafka":
		remote = &kafkaTopic{cred: bsi.Spec.Creds, svc: svc}
	default:
		return "", fmt.Errorf("unknown service '%v' or not supported", svc)
	}

	return remote.URI()
}

type yarnQueue struct {
	cred map[string]string
	svc  string
}

// since queue is start with root., we remove root. prefix by using queue[5:]

func (yarn *yarnQueue) URI() (uri string, err error) {
	queue, ok := yarn.cred["Yarn_Queue"]
	if !ok {
		return "", fmt.Errorf("Yarn Queue value is empty.")
	}
	uri = fmt.Sprintf("/%s/%s", yarn.svc, queue[5:])
	return
}

type hdfsPath struct {
	cred map[string]string
	svc  string
}

func (hdfs *hdfsPath) URI() (uri string, err error) {
	path, ok := hdfs.cred["HDFS_Path"]
	if !ok {
		return "", fmt.Errorf("HDFS Path value is empty")
	}
	uri = fmt.Sprintf("/%s?path=%s", hdfs.svc, path)
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
	cred map[string]string
	svc  string
}

func (hive *hiveDB) URI() (uri string, err error) {
	credStr, ok := hive.cred["Hive_Database"]
	if !ok {
		return "", fmt.Errorf("%v Hive database value is empty", hive.svc)
	}

	hivedb := strings.Split(credStr, ":")
	if len(hivedb) != 2 {
		return "", fmt.Errorf("Hive database '%v' is invalid", credStr)
	}

	db, queue := hivedb[0], hivedb[1]

	// remove root. prefix
	uri = fmt.Sprintf("/%s/%s?queue=%s", hive.svc, db, queue[5:])

	return
}

type hbaseNS struct {
	cred map[string]string
	svc  string
}

func (hbase *hbaseNS) URI() (uri string, err error) {
	ns, ok := hbase.cred["HBase_NameSpace"]
	if !ok {
		return "", fmt.Errorf("%v namespace is empty", hbase.svc)
	}

	uri = fmt.Sprintf("/%s/%s", hbase.svc, ns)
	return uri, nil
}

type kafkaTopic struct {
	cred map[string]string
	svc  string
}

func (kafka *kafkaTopic) URI() (uri string, err error) {
	topic, ok := kafka.cred["topic"]
	if !ok {
		return "", fmt.Errorf("%v topic is empty", kafka.svc)
	}

	uri = fmt.Sprintf("/%s/%s", kafka.svc, topic)
	return uri, nil
}

func init() {

	hadoopBaseURL = os.Getenv("HADOOP_AMOUNT_BASEURL")
	if len(hadoopBaseURL) == 0 {
		clog.Fatal("HADOOP_AMOUNT_BASEURL must be specified.")
	}
	hadoopBaseURL = httpsAddr(hadoopBaseURL)
	clog.Debug("hadoop amount base url:", hadoopBaseURL)

	services := []string{"hbase", "hive", "hdfs", "kafka", "spark", "mapreduce"}
	hadoop := &Hadoop{BaseURL: hadoopBaseURL}
	register("hadoop", services, hadoop)

	// since hadoop and rds is the same api.
	hdpservices := []string{"mongodb", "greenplum"}
	register("rds", hdpservices, hadoop)
}
