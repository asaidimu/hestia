# Blobs API

## namespace

### List blob namespaces

**`POST`** `/system/blobs/namespace/list`

List blob namespaces

- **Handler:** `system:blobs:namespace:list`

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7e63-871e-357644d21499": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-70e8-bd7b-fc7fe2557d83"
      },
      "type": "object"
    }
  },
  "name": "NamespaceListOutput",
  "schemas": {
    "019fba9e-d801-70e8-bd7b-fc7fe2557d83": {
      "fields": {
        "019fba9e-d801-7e23-85ed-34f3bb6592a6": {
          "name": "namespaces",
          "schema": {
            "id": "019fba9e-d802-727f-9236-b9c8e5ee2a14"
          },
          "type": "array"
        }
      },
      "name": "NamespaceListDocument"
    },
    "019fba9e-d802-727f-9236-b9c8e5ee2a14": {
      "fields": {
        "019fba9e-d801-70a7-8d86-4af7acf8d998": {
          "name": "id",
          "type": "string"
        },
        "019fba9e-d802-7163-9bdf-e3a63a4d85ec": {
          "name": "display_name",
          "type": "string"
        },
        "019fba9e-d803-7c8f-b95b-751a38a872f4": {
          "name": "public",
          "type": "boolean"
        }
      },
      "name": "NamespaceView"
    }
  },
  "version": "1.0.0"
}
```

---

### Create a blob namespace

**`POST`** `/system/blobs/namespace/create/{ns}`

Create a blob namespace

- **Handler:** `system:blobs:namespace:create`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7390-a744-dd860b4bc834": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7e0f-b50b-b31ddf229cde"
      },
      "type": "object"
    },
    "019fba9e-d802-7fc4-af97-08858dc922d2": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d802-7b59-ad13-f5d8343f204b"
      },
      "type": "object"
    }
  },
  "name": "NsCreateInput",
  "schemas": {
    "019fba9e-d801-7e0f-b50b-b31ddf229cde": {
      "fields": {
        "019fba9e-d801-7805-84d5-81ffeadc3d63": {
          "name": "ns",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-7b59-ad13-f5d8343f204b": {
      "fields": {
        "019fba9e-d801-7dee-a540-8cdfb580596e": {
          "name": "display_name",
          "type": "string"
        },
        "019fba9e-d802-7ad7-8676-dd32e9bb844c": {
          "name": "public",
          "type": "boolean"
        }
      },
      "name": "payload"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7ac2-980d-cacc6e9c55b6": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-727f-9236-b9c8e5ee2a14"
      },
      "type": "object"
    }
  },
  "name": "NamespaceOutput",
  "schemas": {
    "019fba9e-d801-727f-9236-b9c8e5ee2a14": {
      "fields": {
        "019fba9e-d801-7b16-ae2b-4ffda57ff260": {
          "name": "id",
          "type": "string"
        },
        "019fba9e-d802-73ac-a488-2b2b0344527e": {
          "name": "display_name",
          "type": "string"
        },
        "019fba9e-d803-7673-941e-d9ccf88715dc": {
          "name": "public",
          "type": "boolean"
        }
      },
      "name": "NamespaceView"
    }
  },
  "version": "1.0.0"
}
```

---

### Delete a blob namespace

**`DELETE`** `/system/blobs/namespace/delete/{ns}`

Delete a blob namespace

- **Handler:** `system:blobs:namespace:delete`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7654-8428-b04badd6c3ce": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7c6f-bf23-9646652b223a"
      },
      "type": "object"
    }
  },
  "name": "NsInput",
  "schemas": {
    "019fba9e-d801-7c6f-bf23-9646652b223a": {
      "fields": {
        "019fba9e-d801-70dd-a256-9bfe791c62f3": {
          "name": "ns",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

---

## blob

### List blobs in a namespace

**`POST`** `/system/blobs/blob/list/{ns}`

List blobs in a namespace

- **Handler:** `system:blobs:blob:list`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-766a-8f85-8e051151e23f": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7eab-bf7b-fd1a4188c6b9"
      },
      "type": "object"
    },
    "019fba9e-d802-7741-9bf5-0b7c881d9e90": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d802-7767-b652-70a79e219646"
      },
      "type": "object"
    }
  },
  "name": "BlobListInput",
  "schemas": {
    "019fba9e-d801-7eab-bf7b-fd1a4188c6b9": {
      "fields": {
        "019fba9e-d801-7cb7-a18e-f2b9139c8c8f": {
          "name": "ns",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-7767-b652-70a79e219646": {
      "fields": {
        "019fba9e-d801-7d31-9473-44f0f436957e": {
          "name": "prefix",
          "type": "string"
        },
        "019fba9e-d802-7800-af3a-c2237f43aa13": {
          "name": "limit",
          "type": "integer"
        }
      },
      "name": "payload"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7e5d-9337-ae4a5ebb1e79": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-73f8-bd9a-e620df3c6701"
      },
      "type": "object"
    }
  },
  "name": "BlobListOutput",
  "schemas": {
    "019fba9e-d801-73f8-bd9a-e620df3c6701": {
      "fields": {
        "019fba9e-d801-7616-a2d4-346b7207dcff": {
          "name": "blobs",
          "schema": {
            "id": "019fba9e-d802-74be-8fb0-129efe6a873b"
          },
          "type": "array"
        }
      },
      "name": "BlobListDocument"
    },
    "019fba9e-d802-74be-8fb0-129efe6a873b": {
      "fields": {
        "019fba9e-d801-7cfe-9765-6a68bffe732c": {
          "name": "key",
          "type": "string"
        },
        "019fba9e-d802-7e6a-97e0-b8650a7fc1d4": {
          "name": "namespace_id",
          "type": "string"
        },
        "019fba9e-d803-72f6-9737-00df21055cf6": {
          "name": "content_type",
          "type": "string"
        },
        "019fba9e-d804-7078-bec4-2665283fe1de": {
          "name": "size",
          "type": "integer"
        },
        "019fba9e-d805-7f76-897e-61c83dbd5261": {
          "name": "created_at",
          "type": "string"
        },
        "019fba9e-d806-74bd-bc6e-dd936d68b15e": {
          "name": "updated_at",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d807-750d-8f28-1f7b53368145": {
          "name": "custom",
          "schema": {
            "type": "string"
          },
          "type": "record"
        }
      },
      "name": "BlobMetaView"
    }
  },
  "version": "1.0.0"
}
```

---

### Get blob metadata

**`POST`** `/system/blobs/blob/head/{ns}/{key}`

Get blob metadata

- **Handler:** `system:blobs:blob:head`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-74b9-ab36-c27f41f126f5": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-76da-addb-d775bd8c940d"
      },
      "type": "object"
    }
  },
  "name": "BlobKeyInput",
  "schemas": {
    "019fba9e-d801-76da-addb-d775bd8c940d": {
      "fields": {
        "019fba9e-d801-7697-b17b-314aa45241ec": {
          "name": "ns",
          "type": "string"
        },
        "019fba9e-d802-760e-9e2b-f96395b86758": {
          "name": "key",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-78da-9a6c-60fd21c5bbec": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-74be-8fb0-129efe6a873b"
      },
      "type": "object"
    }
  },
  "name": "BlobMetaOutput",
  "schemas": {
    "019fba9e-d801-74be-8fb0-129efe6a873b": {
      "fields": {
        "019fba9e-d801-75f4-984a-2706cc8a7d4d": {
          "name": "key",
          "type": "string"
        },
        "019fba9e-d802-73fa-b87e-3d6248077553": {
          "name": "namespace_id",
          "type": "string"
        },
        "019fba9e-d803-7597-9f83-96222681e790": {
          "name": "content_type",
          "type": "string"
        },
        "019fba9e-d804-7514-a2e7-619685124e57": {
          "name": "size",
          "type": "integer"
        },
        "019fba9e-d805-7095-bec3-e5e5b583ec19": {
          "name": "created_at",
          "type": "string"
        },
        "019fba9e-d806-71b4-9f81-cd68e9df98be": {
          "name": "updated_at",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d807-7a18-851f-d52d1d71a661": {
          "name": "custom",
          "schema": {
            "type": "string"
          },
          "type": "record"
        }
      },
      "name": "BlobMetaView"
    }
  },
  "version": "1.0.0"
}
```

---

### Upload a blob

**`POST`** `/system/blobs/blob/upload/{ns}/{key}`

Upload a blob

- **Handler:** `system:blobs:blob:upload`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-73a2-9005-e6f15ab198e1": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-74c4-af9a-ee8cc97679d5"
      },
      "type": "object"
    },
    "019fba9e-d802-7772-9c4f-679163bfa9be": {
      "name": "headers",
      "schema": {
        "id": "019fba9e-d802-7179-8627-ad5cb7d48779"
      },
      "type": "object"
    },
    "019fba9e-d803-7295-bdf9-184478420124": {
      "name": "modifiers",
      "schema": {
        "id": "019fba9e-d803-795d-a731-a883903a2692"
      },
      "type": "object"
    },
    "019fba9e-d804-7ca2-9935-355a5223ee6a": {
      "name": "payload",
      "type": "bytes"
    }
  },
  "name": "BlobUploadInput",
  "schemas": {
    "019fba9e-d801-74c4-af9a-ee8cc97679d5": {
      "fields": {
        "019fba9e-d801-7cd0-813a-149b7404378e": {
          "name": "ns",
          "type": "string"
        },
        "019fba9e-d802-7aab-bcf1-ebca45737a7a": {
          "name": "key",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-7179-8627-ad5cb7d48779": {
      "fields": {
        "019fba9e-d801-777d-945e-9a94cdfb6048": {
          "name": "content_type",
          "type": "string"
        }
      },
      "name": "headers"
    },
    "019fba9e-d803-795d-a731-a883903a2692": {
      "fields": {
        "019fba9e-d801-7849-ad61-cb22b1b851b6": {
          "name": "overwrite",
          "type": "string"
        }
      },
      "name": "modifiers"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-78da-9a6c-60fd21c5bbec": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-74be-8fb0-129efe6a873b"
      },
      "type": "object"
    }
  },
  "name": "BlobMetaOutput",
  "schemas": {
    "019fba9e-d801-74be-8fb0-129efe6a873b": {
      "fields": {
        "019fba9e-d801-75f4-984a-2706cc8a7d4d": {
          "name": "key",
          "type": "string"
        },
        "019fba9e-d802-73fa-b87e-3d6248077553": {
          "name": "namespace_id",
          "type": "string"
        },
        "019fba9e-d803-7597-9f83-96222681e790": {
          "name": "content_type",
          "type": "string"
        },
        "019fba9e-d804-7514-a2e7-619685124e57": {
          "name": "size",
          "type": "integer"
        },
        "019fba9e-d805-7095-bec3-e5e5b583ec19": {
          "name": "created_at",
          "type": "string"
        },
        "019fba9e-d806-71b4-9f81-cd68e9df98be": {
          "name": "updated_at",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d807-7a18-851f-d52d1d71a661": {
          "name": "custom",
          "schema": {
            "type": "string"
          },
          "type": "record"
        }
      },
      "name": "BlobMetaView"
    }
  },
  "version": "1.0.0"
}
```

---

### Download a blob

**`GET`** `/system/blobs/blob/download/{ns}/{key}`

Download a blob

- **Handler:** `system:blobs:blob:download`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-74b9-ab36-c27f41f126f5": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-76da-addb-d775bd8c940d"
      },
      "type": "object"
    }
  },
  "name": "BlobKeyInput",
  "schemas": {
    "019fba9e-d801-76da-addb-d775bd8c940d": {
      "fields": {
        "019fba9e-d801-7697-b17b-314aa45241ec": {
          "name": "ns",
          "type": "string"
        },
        "019fba9e-d802-760e-9e2b-f96395b86758": {
          "name": "key",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

---

### Delete a blob

**`DELETE`** `/system/blobs/blob/delete/{ns}/{key}`

Delete a blob

- **Handler:** `system:blobs:blob:delete`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-74b9-ab36-c27f41f126f5": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-76da-addb-d775bd8c940d"
      },
      "type": "object"
    }
  },
  "name": "BlobKeyInput",
  "schemas": {
    "019fba9e-d801-76da-addb-d775bd8c940d": {
      "fields": {
        "019fba9e-d801-7697-b17b-314aa45241ec": {
          "name": "ns",
          "type": "string"
        },
        "019fba9e-d802-760e-9e2b-f96395b86758": {
          "name": "key",
          "type": "string"
        }
      },
      "name": "arguments"
    }
  },
  "version": "1.0.0"
}
```

---

### Update blob metadata

**`PATCH`** `/system/blobs/blob/update/{ns}/{key}`

Update blob metadata

- **Handler:** `system:blobs:blob:update`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7dd5-999d-38e72708de4c": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7259-b136-9e7e894160ad"
      },
      "type": "object"
    },
    "019fba9e-d802-7601-ae6a-4e384dcfa21b": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d802-7d82-8e3f-bb1adb25b935"
      },
      "type": "object"
    }
  },
  "name": "BlobUpdateInput",
  "schemas": {
    "019fba9e-d801-7259-b136-9e7e894160ad": {
      "fields": {
        "019fba9e-d801-7409-bca3-4d981ab43c3a": {
          "name": "ns",
          "type": "string"
        },
        "019fba9e-d802-78a2-89b0-94e74afbef8a": {
          "name": "key",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-7d82-8e3f-bb1adb25b935": {
      "fields": {
        "019fba9e-d801-7503-8bfb-d447eb40335b": {
          "name": "custom",
          "schema": {
            "type": "string"
          },
          "type": "record"
        }
      },
      "name": "payload"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-78da-9a6c-60fd21c5bbec": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-74be-8fb0-129efe6a873b"
      },
      "type": "object"
    }
  },
  "name": "BlobMetaOutput",
  "schemas": {
    "019fba9e-d801-74be-8fb0-129efe6a873b": {
      "fields": {
        "019fba9e-d801-75f4-984a-2706cc8a7d4d": {
          "name": "key",
          "type": "string"
        },
        "019fba9e-d802-73fa-b87e-3d6248077553": {
          "name": "namespace_id",
          "type": "string"
        },
        "019fba9e-d803-7597-9f83-96222681e790": {
          "name": "content_type",
          "type": "string"
        },
        "019fba9e-d804-7514-a2e7-619685124e57": {
          "name": "size",
          "type": "integer"
        },
        "019fba9e-d805-7095-bec3-e5e5b583ec19": {
          "name": "created_at",
          "type": "string"
        },
        "019fba9e-d806-71b4-9f81-cd68e9df98be": {
          "name": "updated_at",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d807-7a18-851f-d52d1d71a661": {
          "name": "custom",
          "schema": {
            "type": "string"
          },
          "type": "record"
        }
      },
      "name": "BlobMetaView"
    }
  },
  "version": "1.0.0"
}
```

---

### Begin a resumable blob upload

**`POST`** `/system/blobs/blob/begin/{ns}`

Begin a resumable blob upload

- **Handler:** `system:blobs:blob:begin`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7abf-a1ea-aa8391bccf22": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7726-9944-0122ba7e58d7"
      },
      "type": "object"
    },
    "019fba9e-d802-7985-accf-9fff855aa3dc": {
      "name": "modifiers",
      "schema": {
        "id": "019fba9e-d802-7f42-803a-b8c4d7753995"
      },
      "type": "object"
    },
    "019fba9e-d803-79b1-ba57-c7e1e6c354f7": {
      "name": "payload",
      "schema": {
        "id": "019fba9e-d803-74f6-9ff1-4527e68777f7"
      },
      "type": "object"
    }
  },
  "name": "BlobBeginInput",
  "schemas": {
    "019fba9e-d801-7726-9944-0122ba7e58d7": {
      "fields": {
        "019fba9e-d801-762f-adde-841e0a9017ea": {
          "name": "ns",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-7f42-803a-b8c4d7753995": {
      "fields": {
        "019fba9e-d801-7f61-9cbd-2667bf731550": {
          "name": "overwrite",
          "type": "string"
        }
      },
      "name": "modifiers"
    },
    "019fba9e-d803-74f6-9ff1-4527e68777f7": {
      "fields": {
        "019fba9e-d801-76bc-b3a9-a82a6f82184e": {
          "name": "key",
          "type": "string"
        },
        "019fba9e-d802-71da-baea-6a746c5def3e": {
          "name": "size",
          "type": "integer"
        },
        "019fba9e-d803-7d7f-8c03-f55d21e234dd": {
          "name": "content_type",
          "type": "string"
        },
        "019fba9e-d804-73f0-a6c9-f7ed19d528fe": {
          "name": "block_size",
          "type": "integer"
        }
      },
      "name": "payload"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-71e8-8148-2d133e8d52d7": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-721c-9a45-d28e4a897dd3"
      },
      "type": "object"
    }
  },
  "name": "UploadBeginOutput",
  "schemas": {
    "019fba9e-d801-721c-9a45-d28e4a897dd3": {
      "fields": {
        "019fba9e-d801-7adc-8493-6411d54ab167": {
          "name": "session_id",
          "type": "string"
        },
        "019fba9e-d802-780c-a355-e03839cbb7da": {
          "name": "key",
          "type": "string"
        },
        "019fba9e-d803-7a68-9975-eb998e5f90cc": {
          "name": "offset",
          "type": "integer"
        },
        "019fba9e-d804-780e-9874-14408a977529": {
          "name": "block_size",
          "type": "integer"
        }
      },
      "name": "UploadBeginDocument"
    }
  },
  "version": "1.0.0"
}
```

---

### Upload a chunk of a resumable blob upload

**`POST`** `/system/blobs/blob/chunk/{ns}`

Upload a chunk of a resumable blob upload

- **Handler:** `system:blobs:blob:chunk`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7016-84a0-ac3a3a9c6f9a": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7f90-8745-1c34559d8563"
      },
      "type": "object"
    },
    "019fba9e-d802-750f-b18a-5c57e49315e5": {
      "name": "headers",
      "schema": {
        "id": "019fba9e-d802-7b5c-aa44-47a9be49b695"
      },
      "type": "object"
    },
    "019fba9e-d803-7d75-9fe0-dd6d92869440": {
      "name": "payload",
      "type": "bytes"
    }
  },
  "name": "BlobChunkInput",
  "schemas": {
    "019fba9e-d801-7f90-8745-1c34559d8563": {
      "fields": {
        "019fba9e-d801-74d6-81ec-74b94cc1f385": {
          "name": "ns",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-7b5c-aa44-47a9be49b695": {
      "fields": {
        "019fba9e-d801-7fec-b63d-27c9b7e74793": {
          "name": "session_id",
          "type": "string"
        },
        "019fba9e-d802-7669-86e8-840571ce18d1": {
          "name": "offset",
          "type": "string"
        },
        "019fba9e-d803-7aa2-8748-479f5e70dea7": {
          "name": "sha256",
          "type": "string"
        }
      },
      "name": "headers"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-77a9-96b3-89a2e11770b0": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-779f-a576-c9a4c7eab600"
      },
      "type": "object"
    }
  },
  "name": "UploadChunkOutput",
  "schemas": {
    "019fba9e-d801-779f-a576-c9a4c7eab600": {
      "fields": {
        "019fba9e-d801-774e-bde9-ec5edc30a890": {
          "name": "total",
          "type": "integer"
        }
      },
      "name": "UploadChunkDocument"
    }
  },
  "version": "1.0.0"
}
```

---

### Complete a resumable blob upload

**`POST`** `/system/blobs/blob/complete/{ns}`

Complete a resumable blob upload

- **Handler:** `system:blobs:blob:complete`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-776a-832f-4af3b8a7d3fb": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7fd1-ae8a-8c11cc9b8030"
      },
      "type": "object"
    },
    "019fba9e-d802-7ba4-a9f2-4728527d346b": {
      "name": "headers",
      "schema": {
        "id": "019fba9e-d802-7edd-9eb7-dd402d2426d9"
      },
      "type": "object"
    },
    "019fba9e-d803-7aa6-87f6-9e129acf0889": {
      "name": "modifiers",
      "schema": {
        "id": "019fba9e-d803-7716-a513-3eddbdea325a"
      },
      "type": "object"
    }
  },
  "name": "BlobCompleteInput",
  "schemas": {
    "019fba9e-d801-7fd1-ae8a-8c11cc9b8030": {
      "fields": {
        "019fba9e-d801-7b0c-b8a4-083387c82062": {
          "name": "ns",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-7edd-9eb7-dd402d2426d9": {
      "fields": {
        "019fba9e-d801-745c-aa50-89d9bf839395": {
          "name": "session_id",
          "type": "string"
        }
      },
      "name": "headers"
    },
    "019fba9e-d803-7716-a513-3eddbdea325a": {
      "fields": {
        "019fba9e-d801-76cb-960f-fc045bd502ec": {
          "name": "overwrite",
          "type": "string"
        }
      },
      "name": "modifiers"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-78da-9a6c-60fd21c5bbec": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-74be-8fb0-129efe6a873b"
      },
      "type": "object"
    }
  },
  "name": "BlobMetaOutput",
  "schemas": {
    "019fba9e-d801-74be-8fb0-129efe6a873b": {
      "fields": {
        "019fba9e-d801-75f4-984a-2706cc8a7d4d": {
          "name": "key",
          "type": "string"
        },
        "019fba9e-d802-73fa-b87e-3d6248077553": {
          "name": "namespace_id",
          "type": "string"
        },
        "019fba9e-d803-7597-9f83-96222681e790": {
          "name": "content_type",
          "type": "string"
        },
        "019fba9e-d804-7514-a2e7-619685124e57": {
          "name": "size",
          "type": "integer"
        },
        "019fba9e-d805-7095-bec3-e5e5b583ec19": {
          "name": "created_at",
          "type": "string"
        },
        "019fba9e-d806-71b4-9f81-cd68e9df98be": {
          "name": "updated_at",
          "nullable": true,
          "type": "string"
        },
        "019fba9e-d807-7a18-851f-d52d1d71a661": {
          "name": "custom",
          "schema": {
            "type": "string"
          },
          "type": "record"
        }
      },
      "name": "BlobMetaView"
    }
  },
  "version": "1.0.0"
}
```

---

### Report progress of a resumable blob upload

**`POST`** `/system/blobs/blob/progress/{ns}`

Report progress of a resumable blob upload

- **Handler:** `system:blobs:blob:progress`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-7f69-a91c-3b498787e794": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7e0b-8797-0995a0a4f197"
      },
      "type": "object"
    },
    "019fba9e-d802-74e6-a179-b5fddfa3b28a": {
      "name": "modifiers",
      "schema": {
        "id": "019fba9e-d802-74ee-837d-e0198f825490"
      },
      "type": "object"
    }
  },
  "name": "BlobProgressInput",
  "schemas": {
    "019fba9e-d801-7e0b-8797-0995a0a4f197": {
      "fields": {
        "019fba9e-d801-7f65-ae62-3b841e402563": {
          "name": "ns",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-74ee-837d-e0198f825490": {
      "fields": {
        "019fba9e-d801-79ed-b239-ff3710453fd2": {
          "name": "session_id",
          "type": "string"
        }
      },
      "name": "modifiers"
    }
  },
  "version": "1.0.0"
}
```

#### Response

```json
{
  "fields": {
    "019fba9e-d801-7689-b65c-a74f4afab002": {
      "name": "document",
      "schema": {
        "id": "019fba9e-d801-74e5-a552-065ca377290a"
      },
      "type": "object"
    }
  },
  "name": "UploadProgressOutput",
  "schemas": {
    "019fba9e-d801-74e5-a552-065ca377290a": {
      "fields": {
        "019fba9e-d801-7608-b593-c81ac27f3532": {
          "name": "total",
          "type": "integer"
        },
        "019fba9e-d802-7a6b-9576-a53ec0b4e1dc": {
          "name": "ranges",
          "schema": {
            "id": "019fba9e-d802-70ba-a26b-b5e736b4c1ad"
          },
          "type": "array"
        },
        "019fba9e-d803-70d9-a597-7c6aa2584813": {
          "name": "block_size",
          "type": "integer"
        },
        "019fba9e-d804-7b04-a7cf-6c0a9ec1285e": {
          "name": "expected_size",
          "type": "integer"
        }
      },
      "name": "UploadProgressDocument"
    },
    "019fba9e-d802-70ba-a26b-b5e736b4c1ad": {
      "fields": {
        "019fba9e-d801-777d-92e7-6e59ee283531": {
          "name": "start",
          "type": "integer"
        },
        "019fba9e-d802-7879-bcdb-0739efaea3fa": {
          "name": "end",
          "type": "integer"
        }
      },
      "name": "ByteRange"
    }
  },
  "version": "1.0.0"
}
```

---

### Abort a resumable blob upload

**`POST`** `/system/blobs/blob/abort/{ns}`

Abort a resumable blob upload

- **Handler:** `system:blobs:blob:abort`

#### Request Body

```json
{
  "fields": {
    "019fba9e-d801-747d-b2f9-bfe1b7ebc168": {
      "name": "arguments",
      "schema": {
        "id": "019fba9e-d801-7882-becd-c5f36cbd9e03"
      },
      "type": "object"
    },
    "019fba9e-d802-75f2-905e-1d5973792710": {
      "name": "headers",
      "schema": {
        "id": "019fba9e-d802-7db2-81e4-67e2b2305b77"
      },
      "type": "object"
    }
  },
  "name": "BlobAbortInput",
  "schemas": {
    "019fba9e-d801-7882-becd-c5f36cbd9e03": {
      "fields": {
        "019fba9e-d801-7188-94c2-0e0c32da32a9": {
          "name": "ns",
          "type": "string"
        }
      },
      "name": "arguments"
    },
    "019fba9e-d802-7db2-81e4-67e2b2305b77": {
      "fields": {
        "019fba9e-d801-7b44-aef0-6846342315a2": {
          "name": "session_id",
          "type": "string"
        }
      },
      "name": "headers"
    }
  },
  "version": "1.0.0"
}
```

---
