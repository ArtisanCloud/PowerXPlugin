package publisher

import (
	"context"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Local mutations enqueue work. Long uploads never run inside the loopback
// server's short request budget. The protected binding route never returns secrets.
func (h *localHandler) serveMarket(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/"), "/")
	if r.URL.RawQuery != "" && !(len(parts) == 3 && parts[0] == "markets" && (parts[2] == "versions" || parts[2] == "publications") && r.Method == http.MethodGet) && !(len(parts) == 3 && parts[0] == "uploads" && parts[2] == "requests" && r.Method == http.MethodGet) {
		writeError(w, invalid("query"))
		return
	}
	read := r.Method == http.MethodGet
	write := r.Method == http.MethodPost
	var value any
	var err error
	status := http.StatusOK
	if len(parts) == 3 && parts[0] == "uploads" && parts[2] == "requests" && read {
		page, size, pageErr := marketPage(r.URL.RawQuery)
		err = pageErr
		if err == nil {
			value, err = h.engine.RequestTraces("uploads", parts[1], page, size)
		}
	} else if len(parts) == 1 && parts[0] == "markets" && read {
		value, err = h.engine.ListMarkets()
	} else if len(parts) == 1 && parts[0] == "markets" && write {
		var input MarketInput
		err = localBody(w, r, &input)
		if err == nil {
			value, err = h.engine.CreateMarket(input)
			status = http.StatusCreated
		}
	} else if len(parts) == 2 && parts[0] == "markets" && read {
		value, err = h.engine.GetMarket(parts[1])
	} else if len(parts) == 3 && parts[0] == "markets" && parts[2] == "versions" && read {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		page, size, pageErr := marketPage(r.URL.RawQuery)
		if pageErr != nil {
			err = pageErr
		} else {
			value, err = h.engine.MarketVersions(ctx, parts[1], page, size)
		}
	} else if len(parts) == 3 && parts[0] == "markets" && parts[2] == "publications" && read {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		page, size, pageErr := marketPage(r.URL.RawQuery)
		if pageErr != nil {
			err = pageErr
		} else {
			value, err = h.engine.MarketPublications(ctx, parts[1], page, size)
		}
	} else if len(parts) == 3 && parts[0] == "markets" && parts[2] == "bind" && write {
		var input struct {
			Token string `json:"token"`
		}
		err = localBody(w, r, &input)
		if err == nil {
			ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
			defer cancel()
			value, err = h.engine.BindMarket(ctx, parts[1], input.Token)
			input.Token = ""
		}
	} else if len(parts) == 3 && parts[0] == "markets" && write && (parts[2] == "verify" || parts[2] == "disconnect") {
		if r.ContentLength != 0 {
			err = invalid("body")
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
			defer cancel()
			if parts[2] == "verify" {
				value, err = h.engine.VerifyMarket(ctx, parts[1])
			} else {
				value, err = h.engine.DisconnectMarket(ctx, parts[1])
			}
		}
	} else if len(parts) == 1 && parts[0] == "uploads" && read {
		value, err = h.engine.ListUploads()
	} else if len(parts) == 1 && parts[0] == "uploads" && write {
		var input UploadInput
		err = localBody(w, r, &input)
		if err == nil {
			value, err = h.engine.StartUpload(context.Background(), input)
			status = http.StatusAccepted
		}
	} else if len(parts) == 2 && parts[0] == "uploads" && read {
		value, err = h.engine.GetUpload(parts[1])
	} else if len(parts) == 3 && parts[0] == "uploads" && write && (parts[2] == "resume" || parts[2] == "cancel") {
		if r.ContentLength != 0 {
			err = invalid("body")
		} else if parts[2] == "resume" {
			value, err = h.engine.ResumeUpload(context.Background(), parts[1])
			status = http.StatusAccepted
		} else {
			value, err = h.engine.CancelUpload(parts[1])
		}
	} else {
		err = failure(NotFound, nil)
	}
	if err != nil {
		writeError(w, AsError(err))
		return
	}
	writeData(w, status, value)
}
func localBody(w http.ResponseWriter, r *http.Request, out any) error {
	t, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || t != "application/json" {
		return failure(MediaType, nil)
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxRequestBytes))
	if err != nil {
		return failure(TooLarge, nil)
	}
	if decodeStrict(b, out) != nil {
		return invalid("body")
	}
	return nil
}

func marketPage(raw string) (int, int, error) {
	q, err := url.ParseQuery(raw)
	if err != nil {
		return 0, 0, invalid("pagination")
	}
	page, size := 1, 20
	for k, v := range q {
		if (k != "page" && k != "size") || len(v) != 1 {
			return 0, 0, invalid("pagination")
		}
		n, err := strconv.Atoi(v[0])
		if err != nil || n < 1 || strconv.Itoa(n) != v[0] {
			return 0, 0, invalid("pagination")
		}
		if k == "page" {
			page = n
		} else {
			size = n
		}
	}
	if page > 1000000 || size > 100 {
		return 0, 0, invalid("pagination")
	}
	return page, size, nil
}
