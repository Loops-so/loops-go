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
- Audience segments — `GetAudienceSegment`, `ListAudienceSegments`
- Events — `SendEvent`
- Transactional — `SendTransactional`, `ListTransactionals`, `CreateTransactional`, `GetTransactional`, `UpdateTransactional`, `EnsureTransactionalDraft`, `PublishTransactional`
- Transactional groups — `CreateTransactionalGroup`, `GetTransactionalGroup`, `UpdateTransactionalGroup`, `ListTransactionalGroups`
- Email messages — `GetEmailMessage`, `UpdateEmailMessage`, `PreviewEmailMessage`
- Campaigns — `CreateCampaign`, `UpdateCampaign`, `GetCampaign`, `ListCampaigns`
- Campaign groups — `CreateCampaignGroup`, `GetCampaignGroup`, `UpdateCampaignGroup`, `ListCampaignGroups`
- Components — `GetComponent`, `ListComponents`
- Themes — `GetTheme`, `ListThemes`
- Uploads — `Upload`, `CreateUpload`, `CompleteUpload`

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

`ListTransactionals` and `ListCampaigns` return a single page of results along with a `*Pagination` value. Pass a `PaginationParams` to control page size and cursor:

```go
items, page, err := client.ListTransactionals(loops.PaginationParams{PerPage: "50"})
if err != nil {
    log.Fatal(err)
}
// page.NextCursor is "" when there are no more pages.
```

To fetch every page, use the generic `Paginate` helper:

```go
all, err := loops.Paginate(func(cursor string) ([]loops.Transactional, *loops.Pagination, error) {
    return client.ListTransactionals(loops.PaginationParams{Cursor: cursor})
})
```

## Uploads

`Upload` is a one-call helper that requests a presigned URL, uploads the bytes, and finalizes the asset. Supported content types are `image/jpeg`, `image/png`, `image/gif` and `image/webp`, up to 4 MB.

```go
f, err := os.Open("hero.png")
if err != nil {
    log.Fatal(err)
}
defer f.Close()
stat, err := f.Stat()
if err != nil {
    log.Fatal(err)
}

asset, err := client.Upload(loops.UploadRequest{
    EmailMessageID: "em_abc123",
    ContentType:    "image/png",
    ContentLength:  stat.Size(),
    Body:           f,
})
if err != nil {
    log.Fatal(err)
}
// asset.FinalURL is the public URL to reference in your email LMX.
```

For finer control, call `CreateUpload` and `CompleteUpload` directly and do the `PUT` to the presigned URL yourself.

## Escape hatch

`Client.Do` is a low-level entry point for endpoints that do not yet have a dedicated method. It shares the client's auth, User-Agent, retries and backoff. The body is JSON-encoded when non-nil; pass `json.RawMessage` to forward a pre-encoded payload. The raw `*http.Response` is returned so the caller decides how to handle non-2xx — close the body when done.

```go
resp, err := client.Do(ctx, http.MethodPost, "/contacts/create", map[string]any{
    "email": "user@example.com",
})
if err != nil {
    return err
}
defer resp.Body.Close()
```

## License

MIT
