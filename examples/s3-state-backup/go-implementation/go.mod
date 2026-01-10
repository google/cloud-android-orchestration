module github.com/google/cloud-android-orchestration/examples/s3-state-backup/go-implementation

go 1.22

require (
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.15.7
	github.com/aws/aws-sdk-go-v2/service/s3 v1.47.5
	github.com/gorilla/mux v1.8.1
)

// Note: These dependencies would be added to the android-cuttlefish repository's go.mod
// when integrating this code.
