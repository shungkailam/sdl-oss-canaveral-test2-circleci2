package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

// ServiceTypeDefault is the default service type
const ServiceTypeDefault = "IoT"

// Placeholder for service type
type serviceType struct {
	name   string
	regexp *regexp.Regexp
}

var serviceTypeMap map[string]*serviceType

func init() {
	// Create the service type mapping
	serviceTypeMap = map[string]*serviceType{
		`(?i)iot`:          &serviceType{name: ServiceTypeDefault},
		`(?i)paas`:         &serviceType{name: "PaaS"},
		`(?i)gray-?matter`: &serviceType{name: "GrayMatter"},
		`(?i)karbon`:       &serviceType{name: "Karbon"},
	}
	for reg, sType := range serviceTypeMap {
		sType.regexp = regexp.MustCompile(reg)
	}
}

// GetServices get service provided
func (dbAPI *dbObjectModelAPI) GetServices(context context.Context, clue string) (model.Service, error) {
	serviceType := ServiceTypeDefault
	for _, sType := range serviceTypeMap {
		if sType.regexp.MatchString(clue) {
			serviceType = sType.name
			break
		}
	}
	resp := model.Service{
		ServiceType: serviceType,
	}
	return resp, nil
}

// GetServicesW get service provided, write output into writer
func (dbAPI *dbObjectModelAPI) GetServicesW(context context.Context, w io.Writer, req *http.Request) error {
	clue := ""
	if *config.Cfg.ServiceType != "" {
		clue = *config.Cfg.ServiceType
	} else {
		if req != nil {
			clue = req.Header.Get("X-Forwarded-Host")
		}
	}
	service, err := dbAPI.GetServices(context, clue)
	if err == nil {
		err = json.NewEncoder(w).Encode(service)
	}
	return err
}

// GetServicesInternal returns the service directly for internal consumption
func (dbAPI *dbObjectModelAPI) GetServicesInternal(ctx context.Context, req *http.Request) (model.Service, error) {
	clue := ""
	if *config.Cfg.ServiceType != "" {
		clue = *config.Cfg.ServiceType
	} else if req != nil {
		clue = req.Header.Get("X-Forwarded-Host")
	}
	return dbAPI.GetServices(ctx, clue)
}

// GetServiceLandingURL returns the service landing URL from the request
func (dbAPI *dbObjectModelAPI) GetServiceLandingURL(ctx context.Context, req *http.Request) string {
	host := req.Header.Get("X-Forwarded-Host")
	return fmt.Sprintf("https://%s/login?mynutanix=true", host)
}
