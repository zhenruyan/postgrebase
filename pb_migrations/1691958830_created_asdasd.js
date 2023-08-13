/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const collection = new Collection({
    "id": "2i8e0tbmfevmlgl",
    "created": "2023-08-13 20:33:50.616Z",
    "updated": "2023-08-13 20:33:50.616Z",
    "name": "asdasd",
    "type": "base",
    "system": false,
    "schema": [
      {
        "system": false,
        "id": "847lxpwh",
        "name": "a1",
        "type": "text",
        "required": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      }
    ],
    "indexes": [
      "CREATE INDEX \"idx_bbEdMHK\" ON \"asdasd\" (\"a1\")"
    ],
    "listRule": null,
    "viewRule": null,
    "createRule": null,
    "updateRule": null,
    "deleteRule": null,
    "options": {}
  });

  return Dao(db).saveCollection(collection);
}, (db) => {
  const dao = new Dao(db);
  const collection = dao.findCollectionByNameOrId("2i8e0tbmfevmlgl");

  return dao.deleteCollection(collection);
})
