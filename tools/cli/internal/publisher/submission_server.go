package publisher

import (
	"context"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gowebpki/jcs"
)

func DecodeSubmission(raw []byte) (SubmissionInput, error) {
	var input SubmissionInput
	if len(raw) > MaxRequestBytes {
		return input, failure(TooLarge, nil)
	}
	if _, err := jcs.Transform(raw); err != nil {
		return input, invalid("submission")
	}
	if decodeStrict(raw, &input) != nil {
		return input, invalid("submission")
	}
	return input, nil
}
func (h *localHandler) serveSubmission(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" && !(strings.HasSuffix(r.URL.Path, "/requests") && r.Method == http.MethodGet) {
		writeError(w, invalid("query"))
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/submissions"), "/")
	var out any
	var err error
	status := http.StatusOK
	if len(parts) == 3 && parts[0] == "" && parts[2] == "requests" && r.Method == http.MethodGet {
		page, size, pageErr := marketPage(r.URL.RawQuery)
		err = pageErr
		if err == nil {
			out, err = h.engine.RequestTraces("submissions", parts[1], page, size)
		}
	} else if r.URL.Path == "/v1/submissions" && r.Method == http.MethodGet {
		out, err = h.engine.ListSubmissions()
	} else if r.URL.Path == "/v1/submissions" && r.Method == http.MethodPost {
		var input SubmissionInput
		input, err = submissionBody(w, r)
		if err == nil {
			out, err = h.engine.StartSubmission(context.Background(), input)
			status = http.StatusAccepted
		}
	} else if len(parts) == 2 && parts[0] == "" && r.Method == http.MethodGet {
		out, err = h.engine.GetSubmission(parts[1])
	} else if r.URL.Path == "/v1/submissions/preview" && r.Method == http.MethodPost {
		var input SubmissionInput
		input, err = submissionBody(w, r)
		if err == nil {
			out, err = h.engine.PreviewSubmission(input)
		}
	} else if r.URL.Path == "/v1/submissions/prepare" && r.Method == http.MethodPost {
		var input struct {
			ConnectionID string `json:"connection_id"`
			VersionID    string `json:"version_id"`
		}
		err = localBody(w, r, &input)
		if err == nil {
			ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
			defer cancel()
			out, err = h.engine.PrepareSubmission(ctx, input.ConnectionID, input.VersionID)
		}
	} else if len(parts) == 3 && parts[0] == "" && r.Method == http.MethodPost && (parts[2] == "resume" || parts[2] == "cancel") {
		if r.ContentLength != 0 {
			err = invalid("body")
		} else if parts[2] == "resume" {
			out, err = h.engine.ResumeSubmission(context.Background(), parts[1])
			status = http.StatusAccepted
		} else {
			out, err = h.engine.CancelSubmission(parts[1])
		}
	} else {
		err = failure(NotFound, nil)
	}
	if err != nil {
		writeError(w, AsError(err))
		return
	}
	writeData(w, status, out)
}

func submissionBody(w http.ResponseWriter, r *http.Request) (SubmissionInput, error) {
	kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || kind != "application/json" {
		return SubmissionInput{}, failure(MediaType, nil)
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxRequestBytes))
	if err != nil {
		return SubmissionInput{}, failure(TooLarge, nil)
	}
	return DecodeSubmission(raw)
}
