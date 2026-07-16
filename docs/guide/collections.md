# Dynamic Collections

Collections provide schema-less document CRUD — similar to Firestore or MongoDB collections.

## Creating a Collection

```
POST /system/collections/collection
```

```json
{
  "name": "products",
  "schema": {
    "fields": {
      "title": { "type": "string" },
      "price": { "type": "number" },
      "inStock": { "type": "boolean" },
      "tags": { "type": "array", "items": { "type": "string" } }
    }
  }
}
```

## CRUD Operations

Collections expose standard document CRUD endpoints:

| Method | Route | Description |
|---|---|---|
| `POST` | `/system/collections/document/{name}/query` | Query documents |
| `GET` | `/system/collections/document/{doc_id}/{name}` | Get document |
| `POST` | `/system/collections/document/{name}` | Create document |
| `PATCH` | `/system/collections/document/{name}/{doc_id}` | Update document |
| `DELETE` | `/system/collections/document/{name}/{doc_id}` | Delete document |

## Client SDK

```ts
const products = api.collection<ProductDoc>("products")

// Create
const created = await products.create({
  title: "Widget", price: 9.99, inStock: true, tags: ["gadget"]
})

// Query
const { data, page } = await products.find({
  pagination: { type: "offset", offset: 0, limit: 20 },
  filters: { price: { $gte: 5 } },
})

// Read
const doc = await products.read(created._id_)

// Update
await products.update(created._id_, { price: 7.99 })

// Delete
await products.delete(created._id_)
```
