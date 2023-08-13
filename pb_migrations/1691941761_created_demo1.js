/// <reference path="../pb_data/types.d.ts" />
migrate((db) => {
  const collection = new Collection({
    "id": "g9e1erpsjjo17le",
    "created": "2023-08-13 15:49:21.334Z",
    "updated": "2023-08-13 15:49:21.334Z",
    "name": "demo1",
    "type": "base",
    "system": false,
    "schema": [
      {
        "system": false,
        "id": "cohiwjzj",
        "name": "a1",
        "type": "text",
        "required": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "system": false,
        "id": "ebmt1o4o",
        "name": "a2",
        "type": "text",
        "required": false,
        "unique": false,
        "options": {
          "min": null,
          "max": null,
          "pattern": ""
        }
      },
      {
        "system": false,
        "id": "37yqha49",
        "name": "a23",
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
  const collection = dao.findCollectionByNameOrId("g9e1erpsjjo17le");

  return dao.deleteCollection(collection);
})
