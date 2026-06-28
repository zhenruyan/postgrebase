package apis

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/daos"
	"github.com/zhenruyan/postgrebase/dbx"
	"github.com/zhenruyan/postgrebase/forms"
	"github.com/zhenruyan/postgrebase/models"
	"github.com/zhenruyan/postgrebase/replication"
	"github.com/zhenruyan/postgrebase/resolvers"
	"github.com/zhenruyan/postgrebase/tools/search"
	"github.com/zhenruyan/postgrebase/vector"
)

const expandQueryParam = "expand"

// bindRecordCrudApi registers the record crud api endpoints and
// the corresponding handlers.
func bindRecordCrudApi(app core.App, rg *echo.Group) {
	api := recordApi{app: app}

	subGroup := rg.Group(
		"/collections/:collection",
		ActivityLogger(app),
	)

	subGroup.GET("/records", api.list, LoadCollectionContext(app))
	subGroup.GET("/records/:id", api.view, LoadCollectionContext(app))
	subGroup.POST("/records", api.create, LoadCollectionContext(app, models.CollectionTypeBase, models.CollectionTypeAuth))
	subGroup.PATCH("/records/:id", api.update, LoadCollectionContext(app, models.CollectionTypeBase, models.CollectionTypeAuth))
	subGroup.DELETE("/records/:id", api.delete, LoadCollectionContext(app, models.CollectionTypeBase, models.CollectionTypeAuth))
}

func (api *recordApi) getCacheKey(collection *models.Collection, suffix string) string {
	return "pb_cache:" + collection.Id + ":" + suffix
}

type recordCacheEntry struct {
	ExpiresAt time.Time `json:"expiresAt"`
	Value     any       `json:"value"`
}

func (api *recordApi) cacheGetJSON(c echo.Context, key string, dest any) bool {
	if api.app.RedisCache() != nil {
		val, err := api.app.RedisCache().Get(c.Request().Context(), key).Result()
		return err == nil && json.Unmarshal([]byte(val), dest) == nil
	}

	raw := api.app.Cache().Get(key)
	if raw == nil {
		return false
	}
	entry, ok := raw.(recordCacheEntry)
	if ok {
		if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
			api.app.Cache().Remove(key)
			return false
		}
		raw = entry.Value
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return false
	}
	return json.Unmarshal(encoded, dest) == nil
}

func (api *recordApi) cacheSet(c echo.Context, key string, value any, ttl time.Duration) {
	if api.app.RedisCache() != nil {
		encoded, _ := json.Marshal(value)
		api.app.RedisCache().Set(c.Request().Context(), key, encoded, ttl)
		return
	}
	entry := recordCacheEntry{Value: value}
	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl)
	}
	api.app.Cache().Set(key, entry)
}

type recordApi struct {
	app core.App
}

func (api *recordApi) list(c echo.Context) error {
	collection, _ := c.Get(ContextCollectionKey).(*models.Collection)
	if collection == nil {
		return NewNotFoundError("", "Missing collection context.")
	}

	records := []*models.Record{}

	// --- Cache Read ---
	var cacheKey string
	canCache := (collection.ListCacheEnabled && c.QueryParams().Encode() == "") ||
		(collection.SearchCacheEnabled && c.QueryParams().Encode() != "")

	if canCache {
		cacheKey = api.getCacheKey(collection, "list:"+c.QueryParams().Encode())
		var cachedResult search.Result
		if api.cacheGetJSON(c, cacheKey, &cachedResult) {
			return c.JSON(http.StatusOK, cachedResult)
		}
	}

	// forbid users and guests to query special filter/sort fields
	if err := api.checkForForbiddenQueryFields(c); err != nil {
		return err
	}

	requestInfo := RequestInfo(c)

	if requestInfo.Admin == nil && collection.ListRule == nil {
		// only admins can access if the rule is nil
		return NewForbiddenError("Only admins can perform this action.", nil)
	}

	fieldsResolver := resolvers.NewRecordFieldResolver(
		api.app.Dao(),
		collection,
		requestInfo,
		// hidden fields are searchable only by admins
		requestInfo.Admin != nil,
	)

	searchProvider := search.NewProvider(fieldsResolver).
		Query(api.app.Dao().RecordQuery(collection))

	if requestInfo.Admin == nil && collection.ListRule != nil {
		searchProvider.AddFilter(search.FilterData(*collection.ListRule))
	}

	result, err := searchProvider.ParseAndExec(c.QueryParams().Encode(), &records)
	if err != nil {
		return NewBadRequestError("Invalid filter parameters.", err)
	}

	event := new(core.RecordsListEvent)
	event.HttpContext = c
	event.Collection = collection
	event.Records = records
	event.Result = result

	return api.app.OnRecordsListRequest().Trigger(event, func(e *core.RecordsListEvent) error {
		if e.HttpContext.Response().Committed {
			return nil
		}

		if err := EnrichRecords(e.HttpContext, api.app.Dao(), e.Records); err != nil && api.app.IsDebug() {
			log.Println(err)
		}

		// --- Cache Write ---
		if canCache && e.Result != nil {
			duration := time.Duration(collection.CacheDuration) * time.Second
			api.cacheSet(e.HttpContext, cacheKey, e.Result, duration)
		}

		return e.HttpContext.JSON(http.StatusOK, e.Result)
	})
}

func (api *recordApi) view(c echo.Context) error {
	collection, _ := c.Get(ContextCollectionKey).(*models.Collection)
	if collection == nil {
		return NewNotFoundError("", "Missing collection context.")
	}

	recordId := c.PathParam("id")
	if recordId == "" {
		return NewNotFoundError("", nil)
	}

	// --- Cache Read ---
	var cacheKey string
	canCache := collection.CacheEnabled
	if canCache {
		cacheKey = api.getCacheKey(collection, "view:"+recordId)
		var cachedRecord models.Record
		if api.cacheGetJSON(c, cacheKey, &cachedRecord) {
			return c.JSON(http.StatusOK, cachedRecord)
		}
	}

	requestInfo := RequestInfo(c)

	if requestInfo.Admin == nil && collection.ViewRule == nil {
		// only admins can access if the rule is nil
		return NewForbiddenError("Only admins can perform this action.", nil)
	}

	ruleFunc := func(q *dbx.SelectQuery) error {
		if requestInfo.Admin == nil && collection.ViewRule != nil && *collection.ViewRule != "" {
			resolver := resolvers.NewRecordFieldResolver(api.app.Dao(), collection, requestInfo, true)
			expr, err := search.FilterData(*collection.ViewRule).BuildExpr(resolver)
			if err != nil {
				return err
			}
			resolver.UpdateQuery(q)
			q.AndWhere(expr)
		}
		return nil
	}

	record, fetchErr := api.app.Dao().FindRecordById(collection.Id, recordId, ruleFunc)
	if fetchErr != nil || record == nil {
		return NewNotFoundError("", fetchErr)
	}

	event := new(core.RecordViewEvent)
	event.HttpContext = c
	event.Collection = collection
	event.Record = record

	return api.app.OnRecordViewRequest().Trigger(event, func(e *core.RecordViewEvent) error {
		if e.HttpContext.Response().Committed {
			return nil
		}

		if err := EnrichRecord(e.HttpContext, api.app.Dao(), e.Record); err != nil && api.app.IsDebug() {
			log.Println(err)
		}

		// --- Cache Write ---
		if canCache && e.Record != nil {
			duration := time.Duration(collection.CacheDuration) * time.Second
			api.cacheSet(e.HttpContext, cacheKey, e.Record, duration)
		}

		return e.HttpContext.JSON(http.StatusOK, e.Record)
	})
}

func (api *recordApi) create(c echo.Context) error {
	collection, _ := c.Get(ContextCollectionKey).(*models.Collection)
	if collection == nil {
		return NewNotFoundError("", "Missing collection context.")
	}

	requestInfo := RequestInfo(c)

	if requestInfo.Admin == nil && collection.CreateRule == nil {
		// only admins can access if the rule is nil
		return NewForbiddenError("Only admins can perform this action.", nil)
	}

	hasFullManageAccess := requestInfo.Admin != nil

	// temporary save the record and check it against the create rule
	if requestInfo.Admin == nil && collection.CreateRule != nil {
		testRecord := models.NewRecord(collection)

		// replace modifiers fields so that the resolved value is always
		// available when accessing requestInfo.Data using just the field name
		if requestInfo.HasModifierDataKeys() {
			requestInfo.Data = testRecord.ReplaceModifers(requestInfo.Data)
		}

		testForm := forms.NewRecordUpsert(api.app, testRecord)
		testForm.SetFullManageAccess(true)
		if err := testForm.LoadRequest(c.Request(), ""); err != nil {
			return NewBadRequestError("Failed to load the submitted data due to invalid formatting.", err)
		}

		createRuleFunc := func(q *dbx.SelectQuery) error {
			if *collection.CreateRule == "" {
				return nil // no create rule to resolve
			}

			resolver := resolvers.NewRecordFieldResolver(api.app.Dao(), collection, requestInfo, true)
			expr, err := search.FilterData(*collection.CreateRule).BuildExpr(resolver)
			if err != nil {
				return err
			}
			resolver.UpdateQuery(q)
			q.AndWhere(expr)
			return nil
		}

		testErr := testForm.DrySubmit(func(txDao *daos.Dao) error {
			foundRecord, err := txDao.FindRecordById(collection.Id, testRecord.Id, createRuleFunc)
			if err != nil {
				return fmt.Errorf("DrySubmit create rule failure: %w", err)
			}
			hasFullManageAccess = hasAuthManageAccess(txDao, foundRecord, requestInfo)
			return nil
		})

		if testErr != nil {
			return NewBadRequestError("Failed to create record.", testErr)
		}
	}

	record := models.NewRecord(collection)
	form := forms.NewRecordUpsert(api.app, record)
	form.SetFullManageAccess(hasFullManageAccess)
	if api.app.IsSQLiteCluster() {
		form.SetSaveFunc(api.saveRecord)
	}

	// load request
	if err := form.LoadRequest(c.Request(), ""); err != nil {
		return NewBadRequestError("Failed to load the submitted data due to invalid formatting.", err)
	}

	event := new(core.RecordCreateEvent)
	event.HttpContext = c
	event.Collection = collection
	event.Record = record
	event.UploadedFiles = form.FilesToUpload()

	// create the record
	return form.Submit(func(next forms.InterceptorNextFunc[*models.Record]) forms.InterceptorNextFunc[*models.Record] {
		return func(m *models.Record) error {
			event.Record = m

			return api.app.OnRecordBeforeCreateRequest().Trigger(event, func(e *core.RecordCreateEvent) error {
				if err := next(e.Record); err != nil {
					return NewBadRequestError("Failed to create record.", err)
				}

				if err := EnrichRecord(e.HttpContext, api.app.Dao(), e.Record); err != nil && api.app.IsDebug() {
					log.Println(err)
				}

				if vectorManager := api.app.VectorManager(); vectorManager != nil {
					vectorManager.TriggerRecordEmbedding(e.Record)
				}

				return api.app.OnRecordAfterCreateRequest().Trigger(event, func(e *core.RecordCreateEvent) error {
					if e.HttpContext.Response().Committed {
						return nil
					}

					return e.HttpContext.JSON(http.StatusOK, e.Record)
				})
			})
		}
	})
}

func (api *recordApi) update(c echo.Context) error {
	collection, _ := c.Get(ContextCollectionKey).(*models.Collection)
	if collection == nil {
		return NewNotFoundError("", "Missing collection context.")
	}

	recordId := c.PathParam("id")
	if recordId == "" {
		return NewNotFoundError("", nil)
	}

	requestInfo := RequestInfo(c)

	if requestInfo.Admin == nil && collection.UpdateRule == nil {
		// only admins can access if the rule is nil
		return NewForbiddenError("Only admins can perform this action.", nil)
	}

	// eager fetch the record so that the modifier field values are replaced
	// and available when accessing requestInfo.Data using just the field name
	if requestInfo.HasModifierDataKeys() {
		record, err := api.app.Dao().FindRecordById(collection.Id, recordId)
		if err != nil || record == nil {
			return NewNotFoundError("", err)
		}
		requestInfo.Data = record.ReplaceModifers(requestInfo.Data)
	}

	ruleFunc := func(q *dbx.SelectQuery) error {
		if requestInfo.Admin == nil && collection.UpdateRule != nil && *collection.UpdateRule != "" {
			resolver := resolvers.NewRecordFieldResolver(api.app.Dao(), collection, requestInfo, true)
			expr, err := search.FilterData(*collection.UpdateRule).BuildExpr(resolver)
			if err != nil {
				return err
			}
			resolver.UpdateQuery(q)
			q.AndWhere(expr)
		}
		return nil
	}

	// fetch record
	record, fetchErr := api.app.Dao().FindRecordById(collection.Id, recordId, ruleFunc)
	if fetchErr != nil || record == nil {
		return NewNotFoundError("", fetchErr)
	}

	form := forms.NewRecordUpsert(api.app, record)
	form.SetFullManageAccess(requestInfo.Admin != nil || hasAuthManageAccess(api.app.Dao(), record, requestInfo))
	if api.app.IsSQLiteCluster() {
		form.SetSaveFunc(api.saveRecord)
	}

	// load request
	if err := form.LoadRequest(c.Request(), ""); err != nil {
		return NewBadRequestError("Failed to load the submitted data due to invalid formatting.", err)
	}

	event := new(core.RecordUpdateEvent)
	event.HttpContext = c
	event.Collection = collection
	event.Record = record
	event.UploadedFiles = form.FilesToUpload()

	// update the record
	return form.Submit(func(next forms.InterceptorNextFunc[*models.Record]) forms.InterceptorNextFunc[*models.Record] {
		return func(m *models.Record) error {
			event.Record = m

			return api.app.OnRecordBeforeUpdateRequest().Trigger(event, func(e *core.RecordUpdateEvent) error {
				if err := next(e.Record); err != nil {
					return NewBadRequestError("Failed to update record.", err)
				}

				if err := EnrichRecord(e.HttpContext, api.app.Dao(), e.Record); err != nil && api.app.IsDebug() {
					log.Println(err)
				}

				if vectorManager := api.app.VectorManager(); vectorManager != nil {
					vectorManager.TriggerRecordEmbedding(e.Record)
				}

				return api.app.OnRecordAfterUpdateRequest().Trigger(event, func(e *core.RecordUpdateEvent) error {
					if e.HttpContext.Response().Committed {
						return nil
					}

					return e.HttpContext.JSON(http.StatusOK, e.Record)
				})
			})
		}
	})
}

func (api *recordApi) delete(c echo.Context) error {
	collection, _ := c.Get(ContextCollectionKey).(*models.Collection)
	if collection == nil {
		return NewNotFoundError("", "Missing collection context.")
	}

	recordId := c.PathParam("id")
	if recordId == "" {
		return NewNotFoundError("", nil)
	}

	requestInfo := RequestInfo(c)

	if requestInfo.Admin == nil && collection.DeleteRule == nil {
		// only admins can access if the rule is nil
		return NewForbiddenError("Only admins can perform this action.", nil)
	}

	ruleFunc := func(q *dbx.SelectQuery) error {
		if requestInfo.Admin == nil && collection.DeleteRule != nil && *collection.DeleteRule != "" {
			resolver := resolvers.NewRecordFieldResolver(api.app.Dao(), collection, requestInfo, true)
			expr, err := search.FilterData(*collection.DeleteRule).BuildExpr(resolver)
			if err != nil {
				return err
			}
			resolver.UpdateQuery(q)
			q.AndWhere(expr)
		}
		return nil
	}

	record, fetchErr := api.app.Dao().FindRecordById(collection.Id, recordId, ruleFunc)
	if fetchErr != nil || record == nil {
		return NewNotFoundError("", fetchErr)
	}

	event := new(core.RecordDeleteEvent)
	event.HttpContext = c
	event.Collection = collection
	event.Record = record

	return api.app.OnRecordBeforeDeleteRequest().Trigger(event, func(e *core.RecordDeleteEvent) error {
		// delete the record
		if err := api.deleteRecord(e.Record); err != nil {
			return NewBadRequestError("Failed to delete record. Make sure that the record is not part of a required relation reference.", err)
		}

		return api.app.OnRecordAfterDeleteRequest().Trigger(event, func(e *core.RecordDeleteEvent) error {
			if e.HttpContext.Response().Committed {
				return nil
			}

			return e.HttpContext.NoContent(http.StatusNoContent)
		})
	})
}

func (api *recordApi) saveRecord(dao *daos.Dao, record *models.Record) error {
	if !api.app.IsSQLiteCluster() {
		return dao.SaveRecord(record)
	}

	op, err := replication.NewRecordUpsertOperation(record, record.IsNew())
	if err != nil {
		return err
	}
	if err := api.proposeSQLiteOperation(op); err != nil {
		return err
	}
	record.MarkAsNotNew()
	return nil
}

func (api *recordApi) deleteRecord(record *models.Record) error {
	if !api.app.IsSQLiteCluster() {
		return api.app.Dao().DeleteRecord(record)
	}

	op, err := replication.NewRecordDeleteOperation(record)
	if err != nil {
		return err
	}
	return api.proposeSQLiteOperation(op)
}

func (api *recordApi) proposeSQLiteOperation(op vector.ReplicatedOperation) error {
	manager := api.app.VectorManager()
	if manager == nil || manager.Coordinator() == nil {
		return NewApiError(http.StatusServiceUnavailable, "SQLite cluster coordinator is not enabled.", nil)
	}
	_, err := manager.Coordinator().ProposeReplicated(op)
	return err
}

func (api *recordApi) checkForForbiddenQueryFields(c echo.Context) error {
	admin, _ := c.Get(ContextAdminKey).(*models.Admin)
	if admin != nil {
		return nil // admins are allowed to query everything
	}

	decodedQuery := c.QueryParam(search.FilterQueryParam) + c.QueryParam(search.SortQueryParam)
	forbiddenFields := []string{"@collection.", "@request."}

	for _, field := range forbiddenFields {
		if strings.Contains(decodedQuery, field) {
			return NewForbiddenError("Only admins can filter by @collection and @request query params", nil)
		}
	}

	return nil
}
