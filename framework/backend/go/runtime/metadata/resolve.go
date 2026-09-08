package metadata

import "context"

// resolvePages never treats a truncated first page as a complete directory.
func resolvePages[T any](ctx context.Context, fetch func(int) (*Page[T], error), matches func(T) bool) (*T, error) {
	var seen int64
	for pageNumber := 1; ; pageNumber++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := fetch(pageNumber)
		if err != nil {
			return nil, err
		}
		if page == nil || page.Pagination.Page != pageNumber || page.Pagination.PageSize <= 0 {
			return nil, &Error{Code: CodeDecodeFailed, Message: "metadata.pagination_invalid"}
		}
		for i := range page.Items {
			if matches(page.Items[i]) {
				return &page.Items[i], nil
			}
		}
		seen += int64(len(page.Items))
		if seen >= page.Pagination.Total {
			return nil, &Error{Code: CodeNotFound, Message: "metadata.not_found"}
		}
		if len(page.Items) == 0 {
			return nil, &Error{Code: CodeDecodeFailed, Message: "metadata.pagination_incomplete"}
		}
	}
}
