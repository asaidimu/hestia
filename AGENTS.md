## Persistence Layer
This codebase relies on **go-anansi** for database operations and its proprietary query language.

* **Local Development Path:** `~/projects/go-anansi` for reference.
When working with schemas: 
1. Prefer writing schemas in json. The guide meta schema is at `~/projects/go-anansi/core/schema/meta/schema.json`
2. Schemas that contribute towards collection go into `./internal/app/**/schema/*.schema.json"` as plain json files
3. Schemas that describe DTOs go into `./internal/app/**/schema.go"` as 
```go 

var mySchema = []byte(`{
  "version": "1.0.",
  "name": "Example",
  "description": "Example schema — replace with your own",
  "fields": {
    "FieldIdForName": {
      "name": "name",
      "required": true,
      "type": "string"
    }
  }
}`)
```
4. 

## IAM Layer

Identity and Access Management is handled via **go-iam**.

* **Local Development Path:** `~/projects/go-iam`

## Test Server

A live, auto-reloading test server runs continuously at `./cmd/test-server` on port **8070**.

Because operations often require authentication, you must first establish a session using the following endpoint:

* **Endpoint:** `POST /api/system/session`
* **Payload:**
```json
{
  "email": "admin@test.local",
  "password": "password123"
}

```
## Discovering Commands

To discover and understand all available registered commands within the system, query the documentation endpoint:

* **Endpoint:** `GET /api/system/core/docs`

---

## Making Anansi Schema Changes

When updating the database schema, strictly adhere to the following workflow:

1. **Edit the Schema:** Modify the schema files directly.
* *Do not* alter existing IDs.
* Only modify field properties.
* If introducing a new field, set its ID to match the field's name.
* If deleting a field, remove it's entry completely preserving the other fields.


2. **Preview Changes:** Validate and preview the migration by running:
```bash
anansi schema migrate --dry-run

```


3. **Generate Migration:** Finalize and generate the required migration files by running:
```bash
anansi schema migrate

```
