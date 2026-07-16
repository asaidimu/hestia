# Blob Storage

Namespaced file (blob) storage with metadata.

## Namespaces

Blobs are organized into namespaces. Each namespace can have its own configuration.

```ts
// Create a namespace
await api.blobs.createNamespace({
  name: "user-avatars",
  publicRead: true,
})

// List namespaces
const { namespaces } = await api.blobs.listNamespaces()

// Delete namespace
await api.blobs.deleteNamespace("user-avatars")
```

## Uploading & Downloading

```ts
// Upload a blob
const blob = await api.blobs.upload("user-avatars", "avatar-123.jpg", fileData, {
  contentType: "image/jpeg",
  metadata: { userId: "123" },
})

// Download
const { data, contentType } = await api.blobs.download("user-avatars", "avatar-123.jpg")

// List blobs in namespace
const { blobs } = await api.blobs.listBlobs("user-avatars")

// Get blob metadata
const meta = await api.blobs.meta("user-avatars", "avatar-123.jpg")

// Delete
await api.blobs.delete("user-avatars", "avatar-123.jpg")
```

## API Endpoints

| Method | Route | Description |
|---|---|---|
| `POST` | `/system/blobs/namespace` | Create namespace |
| `POST` | `/system/blobs/namespace/query` | List namespaces |
| `DELETE` | `/system/blobs/namespace/{ns}` | Delete namespace |
| `GET` | `/system/blobs/blob/{key}/{ns}` | Download blob |
| `POST` | `/system/blobs/blob/{key}/{ns}` | Upload blob |
| `POST` | `/system/blobs/blob/{ns}/query` | List blobs |
| `POST` | `/system/blobs/blob/{ns}/{key}/query` | Get blob metadata |
| `PATCH` | `/system/blobs/blob/{ns}/{key}` | Update blob metadata |
| `DELETE` | `/system/blobs/blob/{ns}/{key}` | Delete blob |
