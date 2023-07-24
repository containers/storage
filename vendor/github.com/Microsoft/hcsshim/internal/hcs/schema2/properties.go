/*
 * HCS API
 *
 * No description provided (generated by Swagger Codegen https://github.com/swagger-api/swagger-codegen)
 *
 * API version: 2.1
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package hcsschema

import (
	v1 "github.com/containerd/cgroups/v3/cgroup1/stats"
)

type Properties struct {
	Id string `json:"Id,omitempty"`

	SystemType string `json:"SystemType,omitempty"`

	RuntimeOsType string `json:"RuntimeOsType,omitempty"`

	Name string `json:"Name,omitempty"`

	Owner string `json:"Owner,omitempty"`

	RuntimeId string `json:"RuntimeId,omitempty"`

	RuntimeTemplateId string `json:"RuntimeTemplateId,omitempty"`

	State string `json:"State,omitempty"`

	Stopped bool `json:"Stopped,omitempty"`

	ExitType string `json:"ExitType,omitempty"`

	Memory *MemoryInformationForVm `json:"Memory,omitempty"`

	Statistics *Statistics `json:"Statistics,omitempty"`

	ProcessList []ProcessDetails `json:"ProcessList,omitempty"`

	TerminateOnLastHandleClosed bool `json:"TerminateOnLastHandleClosed,omitempty"`

	HostingSystemId string `json:"HostingSystemId,omitempty"`

	SharedMemoryRegionInfo []SharedMemoryRegionInfo `json:"SharedMemoryRegionInfo,omitempty"`

	GuestConnectionInfo *GuestConnectionInfo `json:"GuestConnectionInfo,omitempty"`

	// Metrics is not part of the API for HCS but this is used for LCOW v2 to
	// return the full cgroup metrics from the guest.
	Metrics *v1.Metrics `json:"LCOWMetrics,omitempty"`
}
