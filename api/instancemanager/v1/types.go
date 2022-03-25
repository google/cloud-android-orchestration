package v1

type InsertHostRequest struct {
	// [REQUIRED]
	CVDInfo *CVDInfo `json:"cvd_info"`
	// [REQUIRED]
	HostInfo *HostInfo `json:"host_info"`
}

type CVDInfo struct {
	// [REQUIRED] A number that uniquely identifies the a set of builds of different targets.
	BuildID string `json:"build_id"`
	// [REQUIRED] A string to determine the specific product and flavor from the set of builds, e.g. aosp_cf_x86_64_phone-userdebug
	Target string
	// The number of CVDs to create. Use this field if creating more than one instance.
	InstancesNumber int `json:"instances_number"`
}

type HostInfo struct {
	// Required if using GCP.
	GCP *GCPInstance `json:"gcp"`
}

type GCPInstance struct {
	// [REQUIRED]
	DiskSizeGB int `json:"disk_size_gb"`
	// [REQUIRED] More info about this field here for https://cloud.google.com/compute/docs/reference/rest/v1/instances/insert#request-body
	MachineType string `json:"machine_type"`
	// More info about this field here for https://cloud.google.com/compute/docs/reference/rest/v1/instances/insert#request-body
	MinCPUPlatform string `json:"min_cpu_platform"`
}

type Operation struct {
	Name string
	// Service-specific metadata associated with the operation.  It typically
	// contains progress information and common metadata such as create time.
	Metadata interface{} `json:",omitempty"`
	// If the value is `false`, it means the operation is still in progress.
	// If `true`, the operation is completed, and either `error` or `response` is
	// available.
	Done bool
	// Result will contain either an error or a result object but never both.
	Result *Result `json:",omitempty"`
}

type Result struct {
	Error        Error
	ResultObject interface{}
}

type Error struct {
	Code    string
	Message string
}

//
// Helper methods to easy how to test nil or default zero values in nested structs.
//
func (r *InsertHostRequest) GetCVDInfo() *CVDInfo {
	if r == nil {
		return nil
	}
	return r.CVDInfo
}

func (i *CVDInfo) GetBuildID() string {
	if i == nil {
		return ""
	}
	return i.BuildID
}

func (i *CVDInfo) GetTarget() string {
	if i == nil {
		return ""
	}
	return i.Target
}

func (r *InsertHostRequest) GetHostInfo() *HostInfo {
	if r == nil {
		return nil
	}
	return r.HostInfo
}

func (i *HostInfo) GetGCP() *GCPInstance {
	if i == nil {
		return nil
	}
	return i.GCP
}

func (i *GCPInstance) GetDiskSizeGB() int {
	if i == nil {
		return 0
	}
	return i.DiskSizeGB
}

func (i *GCPInstance) GetMachineType() string {
	if i == nil {
		return ""
	}
	return i.MachineType
}
