package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/common"
)

// DeploymentConfig is the API message for deployment configurations.
type DeploymentConfig struct {
	ID int `jsonapi:"primary,deploymentConfig"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	ProjectID int
	Project   *Project `jsonapi:"relation,project"`

	// Domain specific fields
	Name string `jsonapi:"attr,name"`
	// Payload encapsulates DeploymentSchedule in json string format. We use json instead jsonapi because this configuration isn't queryable as HTTP format.
	Payload string `jsonapi:"attr,payload"`
}

// DeploymentSchedule is the API message for deployment schedule.
type DeploymentSchedule struct {
	Deployments []*Deployment `json:"deployments"`
}

// Deployment is the API message for deployment.
type Deployment struct {
	Spec *DeploymentSpec `json:"spec"`
}

// DeploymentSpec is the API message for deployment specification.
type DeploymentSpec struct {
	Selector *LabelSelector `json:"selector"`
}

// LabelSelector is the API message for label selector.
type LabelSelector struct {
	// MatchExpressions is a list of label selector requirements. The requirements are ANDed.
	MatchExpressions []*LabelSelectorRequirement `json:"matchExpressions"`
}

// OperatorType is the type of label selector requirement operator.
// Valid operators are In, Exists.
// Note: NotIn and DoesNotExist are not supported initially.
type OperatorType string

const (
	// InOperatorType is the operator type for In.
	InOperatorType OperatorType = "In"
	// ExistsOperatorType is the operator type for Exists.
	ExistsOperatorType OperatorType = "Exists"
)

// LabelSelectorRequirement is the API message for label selector.
type LabelSelectorRequirement struct {
	// Key is the label key that the selector applies to.
	Key string `json:"key"`

	// Operator represents a key's relationship to a set of values.
	Operator OperatorType `json:"operator"`

	// Values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.
	Values []string `json:"values"`
}

// DeploymentConfigFind is the find request for deployment configs.
type DeploymentConfigFind struct {
	ID *int

	// Related fields
	ProjectID *int
}

// DeploymentConfigUpsert is the message to upsert a deployment configuration.
// NOTE: We use PATCH for Upsert, this is inspired by https://google.aip.dev/134#patch-and-put
type DeploymentConfigUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	// CreatorID is the ID of the creator.
	UpdaterID int

	// Related fields
	ProjectID int

	// Domain specific fields
	Name    string `jsonapi:"attr,name"`
	Payload string `jsonapi:"attr,payload"`
}

// DeploymentConfigService is the service for deployment configurations.
type DeploymentConfigService interface {
	// FindDeploymentConfig finds the deployment configuration in a project.
	FindDeploymentConfig(ctx context.Context, find *DeploymentConfigFind) (*DeploymentConfig, error)
	// UpsertDeploymentConfig upserts a deployment configuration to a project.
	UpsertDeploymentConfig(ctx context.Context, upsert *DeploymentConfigUpsert) (*DeploymentConfig, error)
}

// ValidateAndGetDeploymentSchedule validates and returns the deployment schedule.
// Note: this validation only checks whether the payloads is a valid json, however, invalid field name errors are ignored.
func ValidateAndGetDeploymentSchedule(payload string) (*DeploymentSchedule, error) {
	schedule := &DeploymentSchedule{}
	if err := json.Unmarshal([]byte(payload), schedule); err != nil {
		return nil, err
	}

	for _, d := range schedule.Deployments {
		for _, e := range d.Spec.Selector.MatchExpressions {
			switch e.Operator {
			case InOperatorType:
				if len(e.Values) <= 0 {
					return nil, common.Errorf(common.Invalid, fmt.Errorf("expression key %q with %q operator should have at least one value", e.Key, e.Operator))
				}
			case ExistsOperatorType:
				if len(e.Values) > 0 {
					return nil, common.Errorf(common.Invalid, fmt.Errorf("expression key %q with %q operator shouldn't have values", e.Key, e.Operator))
				}
			default:
				return nil, common.Errorf(common.Invalid, fmt.Errorf("expression key %q has invalid operator %q", e.Key, e.Operator))
			}
		}
	}
	return schedule, nil
}
