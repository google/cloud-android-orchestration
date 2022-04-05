package v1

type CreateHostRequest struct {
	// [REQUIRED]
	CreateHostInstanceRequest *CreateHostInstanceRequest `json:"create_host_instance_request"`
	//
	CreateCVDRequest *CreateCVDRequest `json:"create_cvd_request"`
}

type CreateCVDRequest struct {
	// [REQUIRED] The Android build identifier.
	BuildID string `json:"build_id"`
	// [REQUIRED] A string to determine the specific product and flavor from the set of builds, e.g. aosp_cf_x86_64_phone-userdebug.
	Target string `json:"target"`
	// The number of CVDs to create. Use this field if creating more than one instance.
	InstancesCount int `json:"instances_number"`
}

type CreateHostInstanceRequest struct {
	// Required if using GCP.
	GCP *GCPInstance `json:"gcp"`
}

type GCPInstance struct {
	// [REQUIRED]
	DiskSizeGB int `json:"disk_size_gb"`
	// [REQUIRED] More info about this field in https://cloud.google.com/compute/docs/reference/rest/v1/instances/insert#request-body
	MachineType string `json:"machine_type"`
	// More info about this field in https://cloud.google.com/compute/docs/reference/rest/v1/instances/insert#request-body
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
	// Result will contain either an error or a result object but never both.
	Result *Result `json:"result,omitempty"`
}

type Result struct {
	Error        Error       `json:"error,omitempty"`
	ResultObject interface{} `json:"result,omitempty"`
}

type Error struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
