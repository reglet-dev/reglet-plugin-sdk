package ports

import "github.com/reglet-dev/reglet-sdk/go/domain/entities"

// Policy enforces capability grants against runtime requests.
type Policy interface {
	CheckNetwork(req entities.NetworkRequest, grants *entities.GrantSet) bool
	CheckFileSystem(req entities.FileSystemRequest, grants *entities.GrantSet) bool
	CheckEnvironment(req entities.EnvironmentRequest, grants *entities.GrantSet) bool
	CheckExec(req entities.ExecCapabilityRequest, grants *entities.GrantSet) bool
	CheckKeyValue(req entities.KeyValueRequest, grants *entities.GrantSet) bool

	// Evaluate methods return the decision without side effects (like logging denials).
	EvaluateNetwork(req entities.NetworkRequest, grants *entities.GrantSet) bool
	EvaluateFileSystem(req entities.FileSystemRequest, grants *entities.GrantSet) bool
	EvaluateEnvironment(req entities.EnvironmentRequest, grants *entities.GrantSet) bool
	EvaluateExec(req entities.ExecCapabilityRequest, grants *entities.GrantSet) bool
	EvaluateKeyValue(req entities.KeyValueRequest, grants *entities.GrantSet) bool
}
