package libfirewallsso

import (
	"fmt"
	"github.com/inverse-inc/packetfence/go/pfconfigdriver"
)

// Basic interface that all FirewallSSO must implement
type FirewallSSOInt interface {
	Start(info map[string]string, timeout int) bool
	Stop(info map[string]string) bool
	GetFirewallSSO() *FirewallSSO
	IsRoleBased() bool
	MatchesRole(info map[string]string) bool
}

// Basic struct for all firewalls
type FirewallSSO struct {
	PfconfigMethod string `val:"hash_element"`
	PfconfigNS     string `val:"config::Firewall_SSO"`
	PfconfigHashNS string `val:"-"`
	RoleBasedFirewallSSO
	pfconfigdriver.TypedConfig
	Networks     []string `json:"networks"`
	CacheUpdates string   `json:"cache_updates"`
}

// Get the base firewall SSO object
// This is used so that all structs including FirewallSSO have access to FirewallSSO via the FirewallSSOInt interface
func (fw *FirewallSSO) GetFirewallSSO() *FirewallSSO {
	return fw
}

// Whether or not this firewall is role based.
// Meant to be overriden if necessary by structs including FirewallSSO
func (fw *FirewallSSO) IsRoleBased() bool {
	return true
}

// Start method that will be called on every SSO called via ExecuteStart
func (fw *FirewallSSO) Start(info map[string]string, timeout int) bool {
	return true
}

// Struct to be combined with another one when the firewall SSO should only be for certain roles
type RoleBasedFirewallSSO struct {
	Roles []string `json:"categories"`
}

// Is the role in info["role"] part of the roles that are configured for the SSO
func (rbf *RoleBasedFirewallSSO) MatchesRole(info map[string]string) bool {
	userRole := info["role"]
	for _, role := range rbf.Roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// Execute an SSO request on the specified firewall
// Makes sure to call FirewallSSO.Start and to validate the role if necessary
func ExecuteStart(fw FirewallSSOInt, info map[string]string, timeout int) bool {
	if fw.IsRoleBased() && !fw.MatchesRole(info) {
		fmt.Printf("Not sending SSO for user device %s since it doesn't match the role \n", info["role"])
		return false
	}
	parentResult := fw.GetFirewallSSO().Start(info, timeout)
	childResult := fw.Start(info, timeout)
	return parentResult && childResult
}