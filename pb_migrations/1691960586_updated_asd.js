/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ihsrpg105je3z6b")

  collection.indexes = [
    "CREATE INDEX \"idx_BxbyVbe\" ON \"asd\" (\n  \"aa\",\n  \"created\",\n  \"updated\"\n)"
  ]

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("ihsrpg105je3z6b")

  collection.indexes = []

  return dao.saveCollection(collection)
})
