package coderd

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/google/uuid"

	"github.com/coder/coder/database"
	"github.com/coder/coder/httpapi"
)

// CreateParameterValueRequest is used to create a new parameter value for a scope.
type CreateParameterValueRequest struct {
	Name              string                              `json:"name" validate:"required"`
	SourceValue       string                              `json:"source_value" validate:"required"`
	SourceScheme      database.ParameterSourceScheme      `json:"source_scheme" validate:"oneof=data,required"`
	DestinationScheme database.ParameterDestinationScheme `json:"destination_scheme" validate:"oneof=environment_variable provisioner_variable,required"`
	DestinationValue  string                              `json:"destination_value" validate:"required"`
}

// ParameterSchema represents a parameter parsed from project version source.
type ParameterSchema struct {
	ID                       uuid.UUID                           `json:"id"`
	CreatedAt                time.Time                           `json:"created_at"`
	Name                     string                              `json:"name"`
	Description              string                              `json:"description,omitempty"`
	DefaultSourceScheme      database.ParameterSourceScheme      `json:"default_source_scheme,omitempty"`
	DefaultSourceValue       string                              `json:"default_source_value,omitempty"`
	AllowOverrideSource      bool                                `json:"allow_override_source"`
	DefaultDestinationScheme database.ParameterDestinationScheme `json:"default_destination_scheme,omitempty"`
	DefaultDestinationValue  string                              `json:"default_destination_value,omitempty"`
	AllowOverrideDestination bool                                `json:"allow_override_destination"`
	DefaultRefresh           string                              `json:"default_refresh"`
	RedisplayValue           bool                                `json:"redisplay_value"`
	ValidationError          string                              `json:"validation_error,omitempty"`
	ValidationCondition      string                              `json:"validation_condition,omitempty"`
	ValidationTypeSystem     database.ParameterTypeSystem        `json:"validation_type_system,omitempty"`
	ValidationValueType      string                              `json:"validation_value_type,omitempty"`
}

// ParameterValue represents a set value for the scope.
type ParameterValue struct {
	ID                uuid.UUID                           `json:"id"`
	Name              string                              `json:"name"`
	CreatedAt         time.Time                           `json:"created_at"`
	UpdatedAt         time.Time                           `json:"updated_at"`
	Scope             database.ParameterScope             `json:"scope"`
	ScopeID           string                              `json:"scope_id"`
	SourceScheme      database.ParameterSourceScheme      `json:"source_scheme"`
	DestinationScheme database.ParameterDestinationScheme `json:"destination_scheme"`
	DestinationValue  string                              `json:"destination_value"`
}

// Abstracts creating parameters into a single request/response format.
// Callers are in charge of validating the requester has permissions to
// perform the creation.
func postParameterValueForScope(rw http.ResponseWriter, r *http.Request, db database.Store, scope database.ParameterScope, scopeID string) {
	var createRequest CreateParameterValueRequest
	if !httpapi.Read(rw, r, &createRequest) {
		return
	}
	parameterValue, err := db.InsertParameterValue(r.Context(), database.InsertParameterValueParams{
		ID:                uuid.New(),
		Name:              createRequest.Name,
		CreatedAt:         database.Now(),
		UpdatedAt:         database.Now(),
		Scope:             scope,
		ScopeID:           scopeID,
		SourceScheme:      createRequest.SourceScheme,
		SourceValue:       createRequest.SourceValue,
		DestinationScheme: createRequest.DestinationScheme,
		DestinationValue:  createRequest.DestinationValue,
	})
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("insert parameter value: %s", err),
		})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(rw, r, parameterValue)
}

// Abstracts returning parameters for a scope into a standardized
// request/response format. Callers are responsible for checking
// requester permissions.
func parametersForScope(rw http.ResponseWriter, r *http.Request, db database.Store, req database.GetParameterValuesByScopeParams) {
	parameterValues, err := db.GetParameterValuesByScope(r.Context(), req)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
		parameterValues = []database.ParameterValue{}
	}
	if err != nil {
		httpapi.Write(rw, http.StatusInternalServerError, httpapi.Response{
			Message: fmt.Sprintf("get parameter values: %s", err),
		})
		return
	}

	apiParameterValues := make([]ParameterValue, 0, len(parameterValues))
	for _, parameterValue := range parameterValues {
		apiParameterValues = append(apiParameterValues, convertParameterValue(parameterValue))
	}

	render.Status(r, http.StatusOK)
	render.JSON(rw, r, apiParameterValues)
}

func convertParameterSchema(parameter database.ParameterSchema) ParameterSchema {
	return ParameterSchema{
		ID:                       parameter.ID,
		CreatedAt:                parameter.CreatedAt,
		Name:                     parameter.Name,
		Description:              parameter.Description,
		DefaultSourceScheme:      parameter.DefaultSourceScheme,
		DefaultSourceValue:       parameter.DefaultSourceValue.String,
		AllowOverrideSource:      parameter.AllowOverrideSource,
		DefaultDestinationScheme: parameter.DefaultDestinationScheme,
		DefaultDestinationValue:  parameter.DefaultDestinationValue.String,
		AllowOverrideDestination: parameter.AllowOverrideDestination,
		DefaultRefresh:           parameter.DefaultRefresh,
		RedisplayValue:           parameter.RedisplayValue,
		ValidationError:          parameter.ValidationError,
		ValidationCondition:      parameter.ValidationCondition,
		ValidationTypeSystem:     parameter.ValidationTypeSystem,
		ValidationValueType:      parameter.ValidationValueType,
	}
}

func convertParameterValue(parameterValue database.ParameterValue) ParameterValue {
	return ParameterValue{
		ID:                parameterValue.ID,
		Name:              parameterValue.Name,
		CreatedAt:         parameterValue.CreatedAt,
		UpdatedAt:         parameterValue.UpdatedAt,
		Scope:             parameterValue.Scope,
		ScopeID:           parameterValue.ScopeID,
		SourceScheme:      parameterValue.SourceScheme,
		DestinationScheme: parameterValue.DestinationScheme,
		DestinationValue:  parameterValue.DestinationValue,
	}
}
