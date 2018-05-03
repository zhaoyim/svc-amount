package api

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"fmt"

	"github.com/zonesan/clog"
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1"
)

var (
	dataFoundryHostAddr string
	dataFoundryToken    string
	dataFoundryUser     string
	dataFoundryPass     string
	oClient             *DataFoundryClient
)

type DataFoundryClient struct {
	host        string
	oapiURL     string
	kapiURL     string
	username    string
	password    string
	bearerToken atomic.Value
	token       atomic.Value
}

// BackingService describe a service
type BackingService struct {
	unversioned.TypeMeta `json:",inline"`
	// Spec defines the behavior of the BackingService
	Spec                 BackingServiceSpec `json:"spec,omitempty" description:"spec defines the behavior of the BackingService"`
}

// BackingServiceSpec describe a service spec
type BackingServiceSpec struct {
	// service name
	Name                       string `json:"name, omitempty"`
	// service spec objects metadata
	BackingServiceSpecMetadata `json:"metadata, omitempty"`
}

// BackingServiceSpecMetadata describe a service spec metadata
type BackingServiceSpecMetadata struct {
	// display name
	DisplayName         string `json:"displayName, omitempty"`
	// documentation Url
	DocumentationUrl    string `json:"documentationUrl, omitempty"`
	// image url
	ImageUrl            string `json:"imageUrl, omitempty"`
	// description
	LongDescription     string `json:"longDescription, omitempty"`
	// provider display name
	ProviderDisplayName string `json:"providerDisplayName, omitempty"`
	// support url
	SupportUrl          string `json:"supportUrl, omitempty"`
	// service type, which used to distinguish the service
	Type                string `json:"type, omitempty"`
}

// BackingServiceInstance describe a BackingServiceInstance
type BackingServiceInstance struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	kapi.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the Namespace.
	Spec BackingServiceInstanceSpec `json:"spec,omitempty" description:"spec defines the behavior of the BackingServiceInstance"`

	// Status describes the current status of a Namespace
}

// BackingServiceInstanceSpec describes the attributes on a BackingServiceInstance
type BackingServiceInstanceSpec struct {
	// description of an instance.
	InstanceProvisioning `json:"provisioning, omitempty"`
	// id of an instance
	Binding []InstanceBinding `json:"binding, omitempty"`
	// binding number of an instance
	Bound      int    `json:"bound, omitempty"`
	InstanceID string `json:"instance_id, omitempty"`
	// tags of an instance
}

// InstanceBinding describe an instance binding.
type InstanceBinding struct {
	// bound time of an instance binding
	BoundTime *unversioned.Time `json:"bound_time,omitempty"`
	// bind uid of an instance binding
	BindUuid string `json:"bind_uuid, omitempty"`
	// deploymentconfig of an binding.
	BindDeploymentConfig string `json:"bind_deploymentconfig,omitempty"`
	// bind to hadoopuser
	BindHadoopUser string `json:"bind_hadoop_user,omitempty"`
	// credentials of an instance binding
	Credentials map[string]string `json:"credentials, omitempty"`
}

// InstanceProvisioning describe an InstanceProvisioning detail
type InstanceProvisioning struct {
	// dashboard url of an instance
	DashboardUrl string `json:"dashboard_url, omitempty"`
	// bs name of an instance
	BackingServiceName string `json:"backingservice_name, omitempty"`
	// bs id of an instance
	BackingServiceSpecID string `json:"backingservice_spec_id, omitempty"`
	// bs plan id of an instance
	BackingServicePlanGuid string `json:"backingservice_plan_guid, omitempty"`
	// bs plan name of an instance
	BackingServicePlanName string `json:"backingservice_plan_name, omitempty"`
	// parameters of an instance
	Parameters map[string]string `json:"parameters, omitempty"`
	// credentials of an instance
	Creds map[string]string `json:"credentials, omitempty"`
}

func DFClient() *DataFoundryClient {
	return NewDataFoundryTokenClient(dataFoundryToken)
}

func NewDataFoundryTokenClient(token string) *DataFoundryClient {

	if oClient != nil {
		return oClient
	}
	// host = setBaseUrl(host)
	oClient = &DataFoundryClient{
		host:     dataFoundryHostAddr,
		username: dataFoundryUser,
		password: dataFoundryPass,
		oapiURL:  dataFoundryHostAddr + "/oapi/v1",
		kapiURL:  dataFoundryHostAddr + "/api/v1",
	}

	oClient.setBearerToken("Bearer " + token)
	oClient.setToken(token)

	go oClient.updateBearerToken(time.Hour)

	return oClient
}

func (c *DataFoundryClient) setBearerToken(token string) {
	c.bearerToken.Store(token)
}

func (c *DataFoundryClient) setToken(token string) {
	c.token.Store(token)
}

func (c *DataFoundryClient) BearerToken() string {
	//return oc.bearerToken
	return c.bearerToken.Load().(string)
}

func (c *DataFoundryClient) Token() string {
	return c.token.Load().(string)
}

func (c *DataFoundryClient) GetServiceInstance(ns, name string) (*BackingServiceInstance, error) {
	uri := "/namespaces/" + ns + "/backingserviceinstances/" + name
	bsi := new(BackingServiceInstance)
	err := c.OGet(uri, bsi)
	clog.Tracef("%#v", bsi)
	return bsi, err
}

func (c *DataFoundryClient) GetService(ns, name string) (*kapi.Service, error) {
	uri := "/namespaces/" + ns + "/services/" + name
	svc := new(kapi.Service)
	err := c.KGet(uri, svc)
	clog.Trace(svc)
	return svc, err
}

func (c *DataFoundryClient) GetBackingServices(ns, name string) (*BackingService, error) {
	uri := "/namespaces/" + ns + "/backingservices/" + name
	svc := new(BackingService)
	err := c.OGet(uri, svc)
	clog.Trace(svc)
	return svc, err
}

func (c *DataFoundryClient) ListPods(ns, queryParam string) (*kapi.PodList, error) {
	uri := fmt.Sprintf("/namespaces/%s/pods?%s", ns, queryParam)
	pods := &kapi.PodList{}
	err := c.KGet(uri, pods)
	if err != nil {
		clog.Error(err)
		return nil, err
	}

	return pods, err

}
func (c *DataFoundryClient) DeletetPods(ns, queryParam string) (*kapi.PodList, error) {
	uri := fmt.Sprintf("/namespaces/%s/pods?%s", ns, queryParam)
	pods := &kapi.PodList{}
	err := c.KDelete(uri, pods)
	if err != nil {
		clog.Error(err)
		return nil, err
	}
	return pods, err

}
func (c *DataFoundryClient) ExecCommand(ns, pod, cmd string, args ...string) (interface{}, error) {
	// wsd -insecureSkipVerify \
	// -url \
	// 'wss://10.247.32.17/api/v1/namespaces/service-brokers/pods/sb-ceeajasbecimq-rbbtmq-a4zgh/exec?command=df&command=%2fvar%2flib%2frabbitmq&acss_token=22736IIO7vk_lD1_Bq_rktRQzP7JZzDgjk66-4DjLHk' \
	// -origin https://10.247.32.17
	values := url.Values{}
	values.Set("command", cmd)
	command := values.Encode()

	for _, arg := range args {
		values.Set("command", arg)
		cmdArg := values.Encode()
		command = command + "&" + cmdArg
	}
	uri := fmt.Sprintf("%s/namespaces/%s/pods/%s/exec?%s&access_token=%s",
		c.kapiURL, ns, pod, command, c.Token())
	clog.Debug(uri)

	u, err := url.Parse(uri)
	if err != nil {
		clog.Error(err)
		return nil, err
	}
	u.Scheme = "wss"
	url := u.String()
	origin := c.host
	clog.Debugf("url: %s, origin: %s", url, origin)

	return ws(url, origin, args[len(args)-1])

}

func (c *DataFoundryClient) OGet(uri string, into interface{}) error {
	return doRequest("GET", c.oapiURL+uri, nil, into, c.BearerToken(), nil)
}

func (c *DataFoundryClient) OPost(uri string, body, into interface{}) error {
	return doRequest("POST", c.oapiURL+uri, body, into, c.BearerToken(), nil)
}
func (c *DataFoundryClient) ODelete(uri string, into interface{}) error {
	return doRequest("DELETE", c.oapiURL+uri, &kapi.DeleteOptions{}, into, c.BearerToken(), nil)
}

func (c *DataFoundryClient) KGet(uri string, into interface{}) error {
	return doRequest("GET", c.kapiURL+uri, nil, into, c.BearerToken(), nil)
}

func (c *DataFoundryClient) KPost(uri string, body, into interface{}) error {
	return doRequest("POST", c.kapiURL+uri, body, into, c.BearerToken(), nil)
}

func (c *DataFoundryClient) KDelete(uri string, into interface{}) error {
	return doRequest("DELETE", c.kapiURL+uri, &kapi.DeleteOptions{}, into, c.BearerToken(), nil)
}

func (oc *DataFoundryClient) updateBearerToken(durPhase time.Duration) {
	for {

		// clog.Debugf("Request bearer token from: %v(%v) ", oc.name, oc.host)

		token, err := RequestToken(oc.host, oc.username, oc.password)
		if err != nil {
			clog.Error("RequestToken error, try in 15 seconds. error detail: ", err)

			time.Sleep(15 * time.Second)
		} else {

			oc.setBearerToken("Bearer " + token)
			oc.setToken(token)

			clog.Infof("[%v] [%v]", oc.host, token)

			// durPhase is to avoid mulitple OCs updating tokens at the same time
			time.Sleep(3*time.Hour + durPhase)
			durPhase = 0
		}
	}
}

func RequestToken(host, username, password string) (token string, err error) {

	tr := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		//RoundTrip:       roundTrip,
	}

	var DefaultTransport http.RoundTripper = tr

	oauthUrl := httpsAddr(host) + "/oauth/authorize?client_id=openshift-challenging-client&response_type=token"

	req, _ := http.NewRequest("HEAD", oauthUrl, nil)
	req.SetBasicAuth(username, password)

	resp, err := DefaultTransport.RoundTrip(req)

	//resp, err := client.Do(req)
	if err != nil {
		clog.Error(err)
		return "", err
	} else {
		defer resp.Body.Close()
		location, err := resp.Location()
		if err == nil {
			//fmt.Println("resp", url.Fragment)
			fragments := strings.Split(location.Fragment, "&")
			//n := proc(m)
			n := func(s []string) map[string]string {
				m := map[string]string{}
				for _, v := range s {
					n := strings.Split(v, "=")
					m[n[0]] = n[1]
				}
				return m
			}(fragments)

			//r, _ := json.Marshal(n)

			// return string(r), nil
			return n["access_token"], nil
		}
	}
	return token, err
}

func init() {
	dataFoundryHostAddr = os.Getenv("DATAFOUNDRY_API_SERVER")
	if len(dataFoundryHostAddr) == 0 {
		clog.Fatal("DATAFOUNDRY_API_SERVER must be specified.")
	}
	dataFoundryHostAddr = httpsAddr(dataFoundryHostAddr)
	clog.Debug("datafoundry api server:", dataFoundryHostAddr)

	dataFoundryToken = os.Getenv("DATAFOUNDRY_API_TOKEN")
	if len(dataFoundryToken) == 0 {
		clog.Fatal("DATAFOUNDRY_API_TOKEN must be specified.")
	}
	clog.Debug("datafoundry api token:", "*HIDDEN*") // dataFoundryToken)

	dataFoundryUser = os.Getenv("DATAFOUNDRY_ADMIN_USER")
	if len(dataFoundryUser) == 0 {
		clog.Fatal("DATAFOUNDRY_ADMIN_USER must be specified.")
	}
	dataFoundryPass = os.Getenv("DATAFOUNDRY_ADMIN_PASS")
	if len(dataFoundryPass) == 0 {
		clog.Fatal("DATAFOUNDRY_ADMIN_PASS must be specified.")
	}

	oClient = NewDataFoundryTokenClient(dataFoundryToken)
}
