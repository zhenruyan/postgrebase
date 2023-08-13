/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const collection = new Collection({
    "id": "ihsrpg105je3z6b",
    "created": "2023-08-13 20:51:56.785Z",
    "updated": "2023-08-13 20:51:56.785Z",
    "name": "asd",
    "type": "base",
    "system": false,
    "schema": [
      {
        "system": false,
        "id": "def4ucuy",
        "name": "aa",
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
    "indexes": [],
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
  const collection = dao.findCollectionByNameOrId("ihsrpg105je3z6b");

  return dao.deleteCollection(collection);
})
