package solver

import (
	"github.com/kong/deck/crud"
	"github.com/kong/deck/diff"
	"github.com/kong/deck/konnect"
	"github.com/kong/deck/state"
	"reflect"
)

// serviceVersionCRUD implements crud.Actions interface.
type serviceVersionCRUD struct {
	client *konnect.Client
}

func serviceVersionFromStruct(arg diff.Event) *state.ServiceVersion {
	sv, ok := arg.Obj.(*state.ServiceVersion)
	if !ok {
		panic("unexpected type, expected *state.ServiceVersion")
	}
	return sv
}

func oldServiceVersionFromStruct(arg diff.Event) *state.ServiceVersion {
	sv, ok := arg.OldObj.(*state.ServiceVersion)
	if !ok {
		panic("unexpected type, expected *state.ServiceVersion")
	}
	return sv
}

// Create creates a Service version in Konnect.
// The arg should be of type diff.Event, containing the service to be created,
// else the function will panic.
// It returns a the created *state.ServiceVersion.
func (s *serviceVersionCRUD) Create(arg ...crud.Arg) (crud.Arg, error) {
	event := eventFromArg(arg[0])
	sv := serviceVersionFromStruct(event)
	createdSV, err := s.client.ServiceVersions.Create(nil, &sv.ServiceVersion)
	if err != nil {
		return nil, err
	}
	if sv.ControlPlaneServiceRelation != nil {
		_, err := s.client.ControlPlaneRelations.Create(nil, &konnect.ControlPlaneServiceRelationCreateRequest{
			ServiceVersionID:     *createdSV.ID,
			ControlPlaneEntityID: *sv.ControlPlaneServiceRelation.ControlPlaneEntityID,
		})
		if err != nil {
			return nil, err
		}
	}
	return &state.ServiceVersion{ServiceVersion: *createdSV}, nil
}

// Delete deletes a Service version in Konnect.
// The arg should be of type diff.Event, containing the service to be deleted,
// else the function will panic.
// It returns a the deleted *state.ServiceVersion.
func (s *serviceVersionCRUD) Delete(arg ...crud.Arg) (crud.Arg, error) {
	event := eventFromArg(arg[0])
	sv := serviceVersionFromStruct(event)
	err := s.client.ServiceVersions.Delete(nil, sv.ID)
	if err != nil {
		return nil, err
	}
	return sv, nil
}

// Update updates a Service version in Konnect.
// The arg should be of type diff.Event, containing the service to be updated,
// else the function will panic.
// It returns a the updated *state.ServiceVersion.
func (s *serviceVersionCRUD) Update(arg ...crud.Arg) (crud.Arg, error) {
	var (
		err       error
		updatedSV *konnect.ServiceVersion
	)
	event := eventFromArg(arg[0])
	version := serviceVersionFromStruct(event)
	oldVersion := oldServiceVersionFromStruct(event)

	// if there is a change in service version entity, make a PATCH
	if !version.EqualWithOpts(oldVersion, false, true, true) {
		versionCopy := &state.ServiceVersion{ServiceVersion: *version.DeepCopy()}
		versionCopy.ControlPlaneServiceRelation = nil
		versionCopy.ServicePackage = nil
		updatedSV, err = s.client.ServiceVersions.Update(nil, &versionCopy.ServiceVersion)
		if err != nil {
			return nil, err
		}
	} else {
		updatedSV = &version.ServiceVersion
	}

	// When a service versions update is detected, it could be due to changes in
	// control-plane-entity and service-version relations
	// This is possible only during update events
	err = s.relationCRUD(&version.ServiceVersion, &oldVersion.ServiceVersion)
	if err != nil {
		return nil, err
	}

	return &state.ServiceVersion{ServiceVersion: *updatedSV}, nil
}

func (s *serviceVersionCRUD) relationCRUD(version,
	oldVersion *konnect.ServiceVersion) error {
	var err error

	if version.ControlPlaneServiceRelation != nil &&
		oldVersion.ControlPlaneServiceRelation == nil {
		// no version existed before, create a new relation
		_, err = s.client.ControlPlaneRelations.Create(nil, &konnect.ControlPlaneServiceRelationCreateRequest{
			ServiceVersionID:     *version.ID,
			ControlPlaneEntityID: *version.ControlPlaneServiceRelation.ControlPlaneEntityID,
		})
	} else if version.ControlPlaneServiceRelation == nil && oldVersion.
		ControlPlaneServiceRelation != nil {
		// version doesn't need to exist anymore, delete it
		err = s.client.ControlPlaneRelations.Delete(nil,
			oldVersion.ControlPlaneServiceRelation.ID)
	} else if !reflect.DeepEqual(version.ControlPlaneServiceRelation,
		oldVersion.ControlPlaneServiceRelation) {
		// relations are different, update it
		_, err = s.client.ControlPlaneRelations.Update(nil,
			&konnect.ControlPlaneServiceRelationUpdateRequest{
				ID: *oldVersion.ControlPlaneServiceRelation.ID,
				ControlPlaneServiceRelationCreateRequest: konnect.ControlPlaneServiceRelationCreateRequest{
					ServiceVersionID:     *version.ID,
					ControlPlaneEntityID: *version.ControlPlaneServiceRelation.ControlPlaneEntityID,
				},
			})
	}
	return err
}
