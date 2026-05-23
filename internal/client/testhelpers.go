package client

// NewAPIClientForTest creates an APIClient that targets baseURL instead of the
// production API. For use in tests only.
func NewAPIClientForTest(tokens TokenProvider, baseURL string) *APIClient {
	return &APIClient{tokens: tokens, base: baseURL}
}

// NewPageClientForTest creates a PageClient that routes all requests to baseURL
// instead of courses.fit.cvut.cz. For use in tests only.
func NewPageClientForTest(tokens TokenProvider, baseURL string) *PageClient {
	return &PageClient{tokens: tokens, base: baseURL}
}
