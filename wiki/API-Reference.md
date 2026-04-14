# API Reference

Base URL: `http://localhost:8070`

All endpoints under `/api/v1/` require the header `X-API-KEY` set to the value of the `API_KEY_SECRET` environment variable.

---

## Health Check

```
GET /health
```

No authentication required.

**Response** `200 OK`
```json
{
  "status": "OK",
  "message": "Kliops API Gateway is running"
}
```

---

## Upload DCE Document

```
POST /api/v1/upload
```

Upload a DCE file to MinIO. Max file size: 50 MB. Content type is auto-detected.

**Request**: `multipart/form-data`

| Field    | Type | Required | Description            |
|----------|------|----------|------------------------|
| document | file | yes      | The DCE file to upload |

**Response** `200 OK`
```json
{
  "status": "success",
  "path": "minio://dce-entrants/filename.pdf"
}
```

**Errors**
- `400` file too heavy, invalid request, missing field, or invalid filename
- `500` upload failure

---

## Query Price

```
GET /api/v1/price?source={source}&code={code}
```

Retrieve a unit price for a given article code from one of three pricing backends.

| Parameter | Type   | Required | Values                    |
|-----------|--------|----------|---------------------------|
| source    | string | yes      | `excel`, `postgres`, `erp`|
| code      | string | yes      | Article code              |

**Response** `200 OK`
```json
{
  "source": "excel",
  "code_article": "ART-001",
  "prix": 42.50
}
```

**Errors**
- `400` missing source or code parameter
- `500` internal server error (details logged server-side)

---

## Upload Archive ZIP

```
POST /api/v1/ingest/archive
```

Upload a ZIP archive containing a `manifest.csv` and associated PDF files. Triggers asynchronous processing via RabbitMQ.

**Request**: `multipart/form-data`

| Field   | Type | Required | Description                          |
|---------|------|----------|--------------------------------------|
| archive | file | yes      | ZIP file containing manifest and PDFs|

The `manifest.csv` must contain columns: `id_projet`, `titre`, `client`, `statut`, `fichier_dce`, `fichier_memoire`.

**Response** `202 Accepted`
```json
{
  "message": "Archive received, processing started asynchronously"
}
```

**Errors**
- `400` invalid form or non-ZIP file
- `500` processing failure

---

## Upload Mercuriale (Price List)

```
POST /api/v1/ingest/mercuriale
```

Upload an XLSX pricing spreadsheet. Stored as `mercuriale_current.xlsx` in the `kliops-config` MinIO bucket.

**Request**: `multipart/form-data`

| Field      | Type | Required | Description       |
|------------|------|----------|-------------------|
| excel_file | file | yes      | .xlsx price file  |

**Response** `200 OK`
```json
{
  "message": "Mercuriale updated successfully",
  "path": "minio://kliops-config/mercuriale_current.xlsx"
}
```

---

## Upload Template DOCX

```
POST /api/v1/ingest/template
```

Upload a DOCX company charter template. Stored as `template_charte.docx` in the `kliops-config` MinIO bucket.

**Request**: `multipart/form-data`

| Field         | Type | Required | Description        |
|---------------|------|----------|--------------------|
| template_file | file | yes      | .docx template file|

**Response** `200 OK`
```json
{
  "message": "Template DOCX uploaded successfully",
  "path": "minio://kliops-config/template_charte.docx"
}
```
