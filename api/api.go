package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/zonesan/clog"
)

type Driver interface {
	UsageAmount(svc string, bsi *BackingServiceInstance, req *http.Request) (*svcAmountList, error)
	RestartInstance(bsi *BackingServiceInstance, req *http.Request) error
}

type ServiceDefault struct{}

var ErrNotSupport = errors.New("not supported.")

func (*ServiceDefault) UsageAmount(svc string, bsi *BackingServiceInstance, req *http.Request) (*svcAmountList, error) {
	return nil, ErrNotSupport
}

func (*ServiceDefault) RestartInstance(bsi *BackingServiceInstance, req *http.Request) error {
	return ErrNotSupport
}

type Agent struct {
	driver  Driver
	service string
	req     *http.Request
}

type AmountAgent struct {
	driver   Driver
	services []string
}

var drivers = make(map[string]*AmountAgent)

func ListDrivers() {
	for driver, agent := range drivers {
		clog.Infof("%-10v: %v", driver, agent.services)
	}
}

func register(name string, services []string, driver Driver) {
	if driver == nil {
		panic("driver: Register driver is nil!")
	}
	if _, dup := drivers[name]; dup {
		panic("driver: Register called twice for driver" + name)
	}
	drivers[name] = &AmountAgent{driver: driver, services: services}
}

func findDriver(service string) (*Agent, error) {
	for driver, agent := range drivers {
		for _, svc := range agent.services {
			if strings.ToLower(svc) == strings.ToLower(service) {
				return NewAgent(service, driver)
			}
		}
	}
	return nil, fmt.Errorf("unsupported service '%s'", service)
}

func NewAgent(svc, name string) (*Agent, error) {
	driver, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf("Can't find agent %s", name)
	}
	return &Agent{driver: driver.driver, service: svc}, nil
}

func (agent *Agent) GetAmount(name string, bsi *BackingServiceInstance) (*svcAmountList, error) {
	return agent.driver.UsageAmount(agent.service, bsi, agent.req)
}

func (agent *Agent) RestartInstance(bsi *BackingServiceInstance) error {
	return agent.driver.RestartInstance(bsi, agent.req)
}
