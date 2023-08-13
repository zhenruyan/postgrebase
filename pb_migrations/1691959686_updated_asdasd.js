/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("2i8e0tbmfevmlgl")

  collection.indexes = [
    "CREATE INDEX \"idx_bbEdMHK\" ON \"asdasd\" (\"a1\")",
    "CREATE INDEX \"idx_Yk0deUn\" ON \"asdasd\" (\"a1\")"
  ]

  return dao.saveCollection(collection)
}, (db) => {
  const dao = new Dao(db)
  const collection = dao.findCollectionByNameOrId("2i8e0tbmfevmlgl")

  collection.indexes = [
    "CREATE INDEX \"idx_bbEdMHK\" ON \"asdasd\" (\"a1\")"
  ]

  return dao.saveCollection(collection)
})
