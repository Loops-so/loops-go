# loops-go

[![Go Reference](https://pkg.go.dev/badge/github.com/loops-so/loops-go.svg)](https://pkg.go.dev/github.com/loops-so/loops-go)

Go SDK for the [Loops](https://loops.so) API.

## Install

```sh
go get github.com/loops-so/loops-go
```

## Quickstart

```go
package main

import (
    "log"

    loops "github.com/loops-so/loops-go"
)

func main() {
    client := loops.NewClient("YOUR_API_KEY")

    err := client.SendEvent(loops.SendEventRequest{
        Email:     "user@example.com",
        EventName: "signup",
        EventProperties: map[string]any{
            "plan": "pro",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

## Client options

```go
client := loops.NewClient("YOUR_API_KEY",
    loops.WithBaseURL("https://app.loops.so/api/v1"),
    loops.WithHTTPClient(myHTTPClient),
    loops.WithUserAgent("my-app/1.0"),
    loops.WithLogger(os.Stderr), // verbose request/response logging
)
```

## Supported endpoints

- API key — `GetAPIKey`
- Contacts — `CreateContact`, `UpdateContact`, `DeleteContact`, `FindContacts`, `CheckContactSuppression`, `RemoveContactSuppression`
- Contact properties — `ListContactProperties`, `CreateContactProperty`
- Mailing lists — `ListMailingLists`
- Events — `SendEvent`
- Transactional — `SendTransactional`, `ListTransactional`
- Email messages — `GetEmailMessage`, `UpdateEmailMessage`
- Campaigns — `CreateCampaign`, `UpdateCampaign`, `GetCampaign`, `ListCampaigns`

Full reference: [pkg.go.dev/github.com/loops-so/loops-go](https://pkg.go.dev/github.com/loops-so/loops-go).

## Errors

API errors are returned as `*loops.APIError` with `StatusCode` and `Message`:

```go
if err := client.SendEvent(req); err != nil {
    var apiErr *loops.APIError
    if errors.As(err, &apiErr) {
        log.Printf("loops api error %d: %s", apiErr.StatusCode, apiErr.Message)
    }
    return err
}
```

## Retries

Requests are automatically retried with exponential backoff and jitter on `429` and `5xx` responses (up to 2 retries).

## Idempotency

`SendEvent` and `SendTransactional` accept an `IdempotencyKey` field, which is sent as the `Idempotency-Key` header.

## Pagination

`ListTransactional` and `ListCampaigns` return a single page of results along with a `*Pagination` value. Pass a `PaginationParams` to control page size and cursor:

```go
items, page, err := client.ListTransactional(loops.PaginationParams{PerPage: "50"})
if err != nil {
    log.Fatal(err)
}
// page.NextCursor is "" when there are no more pages.
```

To fetch every page, use the generic `Paginate` helper:

```go
all, err := loops.Paginate(func(cursor string) ([]loops.TransactionalEmail, *loops.Pagination, error) {
    return client.ListTransactional(loops.PaginationParams{Cursor: cursor})
})
```

## License

MIT
