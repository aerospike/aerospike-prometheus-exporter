package dataprovider

import (
	aero "github.com/aerospike/aerospike-client-go/v6"
)

// Inherits DataProvider interface
type SystemInfoProvider struct {
}

func (asm SystemInfoProvider) RequestInfo(infoKeys []string) (map[string]string, error) {
	return nil, nil
}

func (asm SystemInfoProvider) FetchUsersDetails() (bool, []*aero.UserRoles, error) {
	return false, nil, nil
}
