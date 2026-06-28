package replication

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zhenruyan/postgrebase/daos"
	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/tools/security"
	"github.com/zhenruyan/postgrebase/tools/types"
	"github.com/zhenruyan/postgrebase/vector"
)

const (
	OperationSchemaCollectionUpsert  = "schema.collection_upsert"
	OperationSchemaCollectionDelete  = "schema.collection_delete"
	OperationSchemaCollectionsImport = "schema.collections_import"
	OperationRecordCreate            = "record.create"
	OperationRecordUpdate            = "record.update"
	OperationRecordDelete            = "record.delete"
	OperationAdminUpsert             = "admin.upsert"
	OperationAdminDelete             = "admin.delete"
)

type CollectionUpsertPayload struct {
	Collection *models.Collection `json:"collection"`
	IsNew      bool               `json:"isNew"`
}

type CollectionDeletePayload struct {
	Collection *models.Collection `json:"collection"`
}

type CollectionsImportPayload struct {
	Collections   []*models.Collection `json:"collections"`
	DeleteMissing bool                 `json:"deleteMissing"`
}

type RecordUpsertPayload struct {
	CollectionID string         `json:"collectionId"`
	Data         map[string]any `json:"data"`
}

type RecordDeletePayload struct {
	CollectionID string `json:"collectionId"`
	RecordID     string `json:"recordId"`
}

type AdminUpsertPayload struct {
	Admin           *models.Admin  `json:"admin"`
	TokenKey        string         `json:"tokenKey"`
	PasswordHash    string         `json:"passwordHash"`
	LastResetSentAt types.DateTime `json:"lastResetSentAt"`
	IsNew           bool           `json:"isNew"`
}

type AdminDeletePayload struct {
	Admin *models.Admin `json:"admin"`
}

func NewCollectionUpsertOperation(collection *models.Collection, isNew bool) (vector.ReplicatedOperation, error) {
	if collection == nil {
		return vector.ReplicatedOperation{}, errors.New("collection is required")
	}

	stabilizeCollection(collection, isNew)
	payload, err := json.Marshal(CollectionUpsertPayload{
		Collection: collection,
		IsNew:      isNew,
	})
	if err != nil {
		return vector.ReplicatedOperation{}, err
	}

	return sqliteOperation(OperationSchemaCollectionUpsert, payload), nil
}

func NewCollectionDeleteOperation(collection *models.Collection) (vector.ReplicatedOperation, error) {
	if collection == nil {
		return vector.ReplicatedOperation{}, errors.New("collection is required")
	}

	payload, err := json.Marshal(CollectionDeletePayload{Collection: collection})
	if err != nil {
		return vector.ReplicatedOperation{}, err
	}

	return sqliteOperation(OperationSchemaCollectionDelete, payload), nil
}

func NewCollectionsImportOperation(collections []*models.Collection, deleteMissing bool) (vector.ReplicatedOperation, error) {
	if len(collections) == 0 {
		return vector.ReplicatedOperation{}, errors.New("collections are required")
	}
	for _, collection := range collections {
		stabilizeCollection(collection, collection.IsNew())
	}

	payload, err := json.Marshal(CollectionsImportPayload{
		Collections:   collections,
		DeleteMissing: deleteMissing,
	})
	if err != nil {
		return vector.ReplicatedOperation{}, err
	}

	return sqliteOperation(OperationSchemaCollectionsImport, payload), nil
}

func NewRecordUpsertOperation(record *models.Record, isNew bool) (vector.ReplicatedOperation, error) {
	if record == nil {
		return vector.ReplicatedOperation{}, errors.New("record is required")
	}
	if record.Collection() == nil {
		return vector.ReplicatedOperation{}, errors.New("record collection is required")
	}

	stabilizeModel(record, isNew)
	payload, err := json.Marshal(RecordUpsertPayload{
		CollectionID: record.Collection().Id,
		Data:         record.ColumnValueMap(),
	})
	if err != nil {
		return vector.ReplicatedOperation{}, err
	}

	operationType := OperationRecordUpdate
	if isNew {
		operationType = OperationRecordCreate
	}
	return sqliteOperation(operationType, payload), nil
}

func NewRecordDeleteOperation(record *models.Record) (vector.ReplicatedOperation, error) {
	if record == nil {
		return vector.ReplicatedOperation{}, errors.New("record is required")
	}
	if record.Collection() == nil {
		return vector.ReplicatedOperation{}, errors.New("record collection is required")
	}
	if record.Id == "" {
		return vector.ReplicatedOperation{}, errors.New("record id is required")
	}

	payload, err := json.Marshal(RecordDeletePayload{
		CollectionID: record.Collection().Id,
		RecordID:     record.Id,
	})
	if err != nil {
		return vector.ReplicatedOperation{}, err
	}

	return sqliteOperation(OperationRecordDelete, payload), nil
}

func NewAdminUpsertOperation(admin *models.Admin, isNew bool) (vector.ReplicatedOperation, error) {
	if admin == nil {
		return vector.ReplicatedOperation{}, errors.New("admin is required")
	}

	stabilizeModel(admin, isNew)
	payload, err := json.Marshal(AdminUpsertPayload{
		Admin:           admin,
		TokenKey:        admin.TokenKey,
		PasswordHash:    admin.PasswordHash,
		LastResetSentAt: admin.LastResetSentAt,
		IsNew:           isNew,
	})
	if err != nil {
		return vector.ReplicatedOperation{}, err
	}

	return sqliteOperation(OperationAdminUpsert, payload), nil
}

func NewAdminDeleteOperation(admin *models.Admin) (vector.ReplicatedOperation, error) {
	if admin == nil {
		return vector.ReplicatedOperation{}, errors.New("admin is required")
	}

	payload, err := json.Marshal(AdminDeletePayload{Admin: admin})
	if err != nil {
		return vector.ReplicatedOperation{}, err
	}

	return sqliteOperation(OperationAdminDelete, payload), nil
}

func Apply(dao *daos.Dao, op vector.ReplicatedOperation) error {
	if dao == nil {
		return errors.New("dao is required")
	}

	switch op.Type {
	case OperationSchemaCollectionUpsert:
		var payload CollectionUpsertPayload
		if err := json.Unmarshal(op.Payload, &payload); err != nil {
			return err
		}
		if payload.Collection == nil {
			return errors.New("collection payload is required")
		}
		if payload.IsNew {
			payload.Collection.MarkAsNew()
		} else {
			payload.Collection.MarkAsNotNew()
		}
		created := payload.Collection.Created
		updated := payload.Collection.Updated
		if err := dao.SaveCollection(payload.Collection); err != nil {
			return err
		}
		return restoreCollectionTimestamps(dao, payload.Collection, created, updated)

	case OperationSchemaCollectionDelete:
		var payload CollectionDeletePayload
		if err := json.Unmarshal(op.Payload, &payload); err != nil {
			return err
		}
		if payload.Collection == nil {
			return errors.New("collection payload is required")
		}
		payload.Collection.MarkAsNotNew()
		return dao.DeleteCollection(payload.Collection)

	case OperationSchemaCollectionsImport:
		var payload CollectionsImportPayload
		if err := json.Unmarshal(op.Payload, &payload); err != nil {
			return err
		}
		return dao.RunInTransaction(func(txDao *daos.Dao) error {
			return txDao.ImportCollections(payload.Collections, payload.DeleteMissing, nil)
		})

	case OperationRecordCreate, OperationRecordUpdate:
		var payload RecordUpsertPayload
		if err := json.Unmarshal(op.Payload, &payload); err != nil {
			return err
		}
		record, err := recordFromPayload(dao, payload.CollectionID, payload.Data)
		if err != nil {
			return err
		}
		if op.Type == OperationRecordCreate {
			record.MarkAsNew()
		} else {
			record.MarkAsNotNew()
		}
		created := record.Created
		updated := record.Updated
		if err := dao.SaveRecord(record); err != nil {
			return err
		}
		return restoreRecordTimestamps(dao, record, created, updated)

	case OperationRecordDelete:
		var payload RecordDeletePayload
		if err := json.Unmarshal(op.Payload, &payload); err != nil {
			return err
		}
		record, err := dao.FindRecordById(payload.CollectionID, payload.RecordID)
		if err != nil {
			return err
		}
		record.MarkAsNotNew()
		return dao.DeleteRecord(record)

	case OperationAdminUpsert:
		var payload AdminUpsertPayload
		if err := json.Unmarshal(op.Payload, &payload); err != nil {
			return err
		}
		if payload.Admin == nil {
			return errors.New("admin payload is required")
		}
		payload.Admin.TokenKey = payload.TokenKey
		payload.Admin.PasswordHash = payload.PasswordHash
		payload.Admin.LastResetSentAt = payload.LastResetSentAt
		if payload.IsNew {
			payload.Admin.MarkAsNew()
		} else {
			payload.Admin.MarkAsNotNew()
		}
		created := payload.Admin.Created
		updated := payload.Admin.Updated
		if err := dao.SaveAdmin(payload.Admin); err != nil {
			return err
		}
		return restoreModelTimestamps(dao, payload.Admin, created, updated)

	case OperationAdminDelete:
		var payload AdminDeletePayload
		if err := json.Unmarshal(op.Payload, &payload); err != nil {
			return err
		}
		if payload.Admin == nil {
			return errors.New("admin payload is required")
		}
		payload.Admin.MarkAsNotNew()
		return dao.DeleteAdmin(payload.Admin)

	default:
		return fmt.Errorf("unknown replicated operation type %q", op.Type)
	}
}

func sqliteOperation(operationType string, payload []byte) vector.ReplicatedOperation {
	return vector.ReplicatedOperation{
		ID:        security.NewUUIDString(),
		Kind:      vector.ReplicatedOperationKindSQLite,
		Type:      operationType,
		Strict:    true,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}
}

func stabilizeCollection(collection *models.Collection, isNew bool) {
	stabilizeModel(collection, isNew)
}

func stabilizeModel(model models.Model, isNew bool) {
	if model == nil {
		return
	}
	if !model.HasId() {
		model.RefreshId()
	}
	now := types.NowDateTime()
	if model.GetCreated().IsZero() {
		switch m := model.(type) {
		case *models.Collection:
			m.Created = now
		case *models.Record:
			m.Created = now
		case *models.Admin:
			m.Created = now
		}
	}
	if model.GetUpdated().IsZero() || !isNew {
		switch m := model.(type) {
		case *models.Collection:
			m.Updated = now
		case *models.Record:
			m.Updated = now
		case *models.Admin:
			m.Updated = now
		}
	}
}

func recordFromPayload(dao *daos.Dao, collectionID string, data map[string]any) (*models.Record, error) {
	if collectionID == "" {
		return nil, errors.New("record collection id is required")
	}
	if data == nil {
		return nil, errors.New("record data is required")
	}

	collection, err := dao.FindCollectionByNameOrId(collectionID)
	if err != nil {
		return nil, err
	}
	record := models.NewRecord(collection)
	record.Load(data)
	return record, nil
}

func restoreRecordTimestamps(dao *daos.Dao, record *models.Record, created, updated types.DateTime) error {
	return restoreModelTimestamps(dao, record, created, updated)
}

func restoreCollectionTimestamps(dao *daos.Dao, collection *models.Collection, created, updated types.DateTime) error {
	return restoreModelTimestamps(dao, collection, created, updated)
}

func restoreModelTimestamps(dao *daos.Dao, model models.Model, created, updated types.DateTime) error {
	if model == nil || model.GetId() == "" {
		return nil
	}
	if created.IsZero() && updated.IsZero() {
		return nil
	}

	values := map[string]any{}
	if !created.IsZero() {
		values["created"] = created
		switch m := model.(type) {
		case *models.Collection:
			m.Created = created
		case *models.Record:
			m.Created = created
		case *models.Admin:
			m.Created = created
		}
	}
	if !updated.IsZero() {
		values["updated"] = updated
		switch m := model.(type) {
		case *models.Collection:
			m.Updated = updated
		case *models.Record:
			m.Updated = updated
		case *models.Admin:
			m.Updated = updated
		}
	}

	_, err := dao.DB().Update(model.TableName(), values, dbx.HashExp{"id": model.GetId()}).Execute()
	return err
}
