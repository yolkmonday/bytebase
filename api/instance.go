package api

import (
	"context"
	"encoding/json"

	"github.com/bytebase/bytebase/plugin/db"
)

// Instance is the API message for an instance.
type Instance struct {
	ID int `jsonapi:"primary,instance"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	EnvironmentID int
	Environment   *Environment `jsonapi:"relation,environment"`
	// Anomalies are stored in a separate table, but just return here for convenience
	AnomalyList []*Anomaly `jsonapi:"relation,anomaly"`

	// Domain specific fields
	Name          string  `jsonapi:"attr,name"`
	Engine        db.Type `jsonapi:"attr,engine"`
	EngineVersion string  `jsonapi:"attr,engineVersion"`
	ExternalLink  string  `jsonapi:"attr,externalLink"`
	Host          string  `jsonapi:"attr,host"`
	Port          string  `jsonapi:"attr,port"`
	Username      string  `jsonapi:"attr,username"`
	// Password is not returned to the client
	Password string
}

// InstanceCreate is the API message for creating an instance.
type InstanceCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	EnvironmentID int `jsonapi:"attr,environmentId"`

	// Domain specific fields
	Name         string  `jsonapi:"attr,name"`
	Engine       db.Type `jsonapi:"attr,engine"`
	ExternalLink string  `jsonapi:"attr,externalLink"`
	Host         string  `jsonapi:"attr,host"`
	Port         string  `jsonapi:"attr,port"`
	Username     string  `jsonapi:"attr,username"`
	Password     string  `jsonapi:"attr,password"`
}

// InstanceFind is the API message for finding instances.
type InstanceFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus
}

func (find *InstanceFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// InstancePatch is the API message for patching an instance.
type InstancePatch struct {
	ID int `jsonapi:"primary,instancePatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name             *string `jsonapi:"attr,name"`
	EngineVersion    *string
	ExternalLink     *string `jsonapi:"attr,externalLink"`
	Host             *string `jsonapi:"attr,host"`
	Port             *string `jsonapi:"attr,port"`
	Username         *string `jsonapi:"attr,username"`
	Password         *string `jsonapi:"attr,password"`
	UseEmptyPassword bool    `jsonapi:"attr,useEmptyPassword"`
}

// InstanceMigrationSchemaStatus is the schema status for instance migration.
type InstanceMigrationSchemaStatus string

const (
	// InstanceMigrationSchemaUnknown is the UNKNOWN InstanceMigrationSchemaStatus.
	InstanceMigrationSchemaUnknown InstanceMigrationSchemaStatus = "UNKNOWN"
	// InstanceMigrationSchemaOK is the OK InstanceMigrationSchemaStatus.
	InstanceMigrationSchemaOK InstanceMigrationSchemaStatus = "OK"
	// InstanceMigrationSchemaNotExist is the NOT_EXIST InstanceMigrationSchemaStatus.
	InstanceMigrationSchemaNotExist InstanceMigrationSchemaStatus = "NOT_EXIST"
)

func (e InstanceMigrationSchemaStatus) String() string {
	switch e {
	case InstanceMigrationSchemaUnknown:
		return "UNKNOWN"
	case InstanceMigrationSchemaOK:
		return "OK"
	case InstanceMigrationSchemaNotExist:
		return "NOT_EXIST"
	}
	return "UNKNOWN"
}

// InstanceMigration is the API message for instance migration.
type InstanceMigration struct {
	Status InstanceMigrationSchemaStatus `jsonapi:"attr,status"`
	Error  string                        `jsonapi:"attr,error"`
}

// MigrationHistory is stored in the instance instead of our own data file, so the field
// format is a bit different from the standard format
type MigrationHistory struct {
	ID int `jsonapi:"primary,migrationHistory"`

	// Standard fields
	Creator   string `jsonapi:"attr,creator"`
	CreatedTs int64  `jsonapi:"attr,createdTs"`
	Updater   string `jsonapi:"attr,updater"`
	UpdatedTs int64  `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	ReleaseVersion    string             `jsonapi:"attr,releaseVersion"`
	Database          string             `jsonapi:"attr,database"`
	Engine            db.MigrationEngine `jsonapi:"attr,engine"`
	Type              db.MigrationType   `jsonapi:"attr,type"`
	Status            db.MigrationStatus `jsonapi:"attr,status"`
	Version           string             `jsonapi:"attr,version"`
	Description       string             `jsonapi:"attr,description"`
	Statement         string             `jsonapi:"attr,statement"`
	Schema            string             `jsonapi:"attr,schema"`
	SchemaPrev        string             `jsonapi:"attr,schemaPrev"`
	ExecutionDuration int                `jsonapi:"attr,executionDuration"`
	// This is a string instead of int as the issue id may come from other issue tracking system in the future
	IssueID string `jsonapi:"attr,issueId"`
	Payload string `jsonapi:"attr,payload"`
}

// InstanceService is the service for instances.
type InstanceService interface {
	// CreateInstance should also create the * database and the admin data source.
	CreateInstance(ctx context.Context, create *InstanceCreate) (*Instance, error)
	FindInstanceList(ctx context.Context, find *InstanceFind) ([]*Instance, error)
	FindInstance(ctx context.Context, find *InstanceFind) (*Instance, error)
	PatchInstance(ctx context.Context, patch *InstancePatch) (*Instance, error)
}
