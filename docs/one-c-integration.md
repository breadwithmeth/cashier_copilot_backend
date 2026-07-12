# 1C integration

All 1C requests are authenticated by `X-1C-Key` or `X-One-C-Key`.
The value must match `ONE_C_API_KEY`.

Each request is bound to a workplace:

- `organizationCode` is optional when `ONE_C_ORGANIZATION_CODE` is configured.
- `storeCode` is required.
- `workplaceExternalId` is required and must match `workplaces.external_id` in that store.

## Event

`POST /api/integrations/1c/events`

For `eventType: "PRODUCT_SCANNED"` the request is saved to `product_scans`.
Other event types are saved to `external_events`.

```json
{
  "storeCode": "store-1",
  "workplaceExternalId": "pos-1",
  "externalEventId": "scan-000001",
  "eventType": "PRODUCT_SCANNED",
  "occurredAt": "2026-07-13T10:15:30.000Z",
  "externalReceiptId": "receipt-000001",
  "barcode": "4870000000012",
  "productName": "Milk",
  "quantity": 1,
  "price": 650,
  "currency": "KZT",
  "payload": {}
}
```

## Receipt

`POST /api/integrations/1c/receipts`

Receipts are saved to `receipts`, receipt lines are saved to `receipt_items`,
and the backend creates or updates a linked `sale_sessions` row.

```json
{
  "storeCode": "store-1",
  "workplaceExternalId": "pos-1",
  "externalReceiptId": "receipt-000001",
  "externalOrderId": "order-000001",
  "occurredAt": "2026-07-13T10:16:00.000Z",
  "cashierExternalId": "cashier-1",
  "paymentMethod": "CARD",
  "items": [
    {
      "lineNumber": 1,
      "externalProductId": "product-000001",
      "barcode": "4870000000012",
      "productName": "Milk",
      "quantity": 1,
      "price": 650,
      "lineTotal": 650,
      "discountAmount": 0,
      "isContainer": false
    }
  ],
  "totals": {
    "operationType": "SALE",
    "receiptStatus": "CLOSED",
    "amount": 650,
    "paidAmount": 650,
    "changeAmount": 0,
    "bonusAmount": 0,
    "discountAmount": 0,
    "currency": "KZT"
  }
}
```

`paymentMethod` can be `CASH`, `CARD`, `BONUS`, or `MIXED`.
`operationType` can be `SALE`, `RETURN`, `CANCEL`, or `STORNO`.
`receiptStatus` can be `OPEN`, `CLOSED`, `CANCELLED`, or `RETURNED`.

If analytics events already contain the same `externalReceiptId` or `externalOrderId`,
the receipt endpoint links them by updating `receipt_id`, `external_receipt_id`,
and `external_order_id`.

## Tables

1C integration writes to these tables:

- `product_scans` - real-time product scan events.
- `receipts` - receipt header, payment and operation data.
- `receipt_items` - normalized receipt lines.
- `sale_sessions` - sale/service session linked to the receipt.
- `external_events` - fallback table for non-product-scan external events.
- `integration_errors` - expected table for import and mapping errors.
