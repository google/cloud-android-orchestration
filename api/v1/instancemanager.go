package v1

type CreateHostRequest struct {
	// [REQUIRED]
	HostInstance *HostInstance `json:"host_instance"`
}

type HostInstance struct {
	// [Output Only] Instance name.
	Name string `json:"name,omitempty"`
	// [Output Only] Boot disk size in GB.
	BootDiskSizeGB int64 `json:"boot_disk_size_gb,omitempty"`
	// GCP specific properties.
	GCP *GCPInstance `json:"gcp,omitempty"`
}

type GCPInstance struct {
	// [REQUIRED] Specifies the machine type of the VM Instance.
	// Check https://cloud.google.com/compute/docs/regions-zones#available for available values.
	MachineType string `json:"machine_type"`
	// Specifies a minimum CPU platform for the VM instance.
	MinCPUPlatform string `json:"min_cpu_platform"`
}

type Operation struct {
	Name string `json:"name"`
	// Service-specific metadata associated with the operation.  It typically
	// contains progress information and common metadata such as create time.
	Metadata interface{} `json:"metadata,omitempty"`
	// If the value is `false`, it means the operation is still in progress.
	// If `true`, the operation is completed, and either `error` or `response` is
	// available.
	Done bool `json:"done"`
}

type OperationResult struct {
	// The error result of the operation in case of failure or cancellation.
	Error *Error `json:"error,omitempty"`
	// The expected response of the operation in case of success.  If the original method returns
	// no data on success, such as `Delete`, this field will be empty, hence omitted. If the original
	// method is standard: `Get`/`Create`/`Update`, the response should be the relevant resource
	// encoded in JSON format.
	Response string `json:"response,omitempty"`
}

type ListHostsResponse struct {
	Items []*HostInstance `json:"items"`
	// This token allows you to get the next page of results for list requests.
	// If the number of results is larger than maxResults, use the `nextPageToken`
	// as a value for the query parameter `pageToken` in the next list request.
	// Subsequent list requests will have their own `nextPageToken` to continue
	// paging through out all the results.
	NextPageToken string `json:"nextPageToken,omitempty"`
}
