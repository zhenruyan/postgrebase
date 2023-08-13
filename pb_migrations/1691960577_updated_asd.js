/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ihsrpg105je3z6b")

  collection.indexes = []

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ihsrpg105je3z6b")

  collection.indexes = [
    "CREATE INDEX \"idx_luB5JZt\" ON \"asd\" (\"aa\")"
  ]

  return dao.saveCollection(collection)
})
