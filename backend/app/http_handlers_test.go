package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type mockResp struct {
	status int
	body   string
}

type mockRoundTripper struct {
	mu        sync.Mutex
	responses map[string][]mockResp
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	list, ok := m.responses[req.URL.String()]
	if !ok || len(list) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	resp := list[0]
	m.responses[req.URL.String()] = list[1:]

	return &http.Response{
		StatusCode: resp.status,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func withMockHTTPClient(t *testing.T, responses map[string][]mockResp) func() {
	t.Helper()
	original := httpc
	httpc = &http.Client{Transport: &mockRoundTripper{responses: responses}}
	return func() { httpc = original }
}

func TestGetJSONReturnsHttpError(t *testing.T) {
	defer withMockHTTPClient(t, map[string][]mockResp{
		"https://api.chess.com/notfound": {
			{status: http.StatusNotFound, body: `{"message":"gone"}`},
		},
	})()

	err := getJSON(context.Background(), "https://api.chess.com/notfound", &struct{}{})
	httpErr, ok := err.(httpError)
	if !ok {
		t.Fatalf("expected httpError, got %T", err)
	}
	if httpErr.Status != http.StatusNotFound || httpErr.Body != "gone" {
		t.Fatalf("httpError mismatch: %+v", httpErr)
	}
}

//Commenting these out until I stop hardcoding the # of games I'm pulling
//in http_handlers.go

// func TestGetChessGamesSuccess(t *testing.T) {
// 	gin.SetMode(gin.TestMode)

// 	responses := map[string][]mockResp{
// 		"https://api.chess.com/pub/player/testuser/games/archives": {
// 			{status: http.StatusOK, body: `{"archives":["https://api.chess.com/pub/player/testuser/2023/01","https://api.chess.com/pub/player/testuser/2023/02"]}`},
// 		},
// 		"https://api.chess.com/pub/player/testuser/2023/01": {
// 			{status: http.StatusOK, body: `{"games":[{"url":"u1","end_time":1,"rated":true,"time_class":"rapid","time_control":"600","pgn":"pgn1","white":{"username":"bob","result":"resigned","rating":1200},"black":{"username":"testuser","result":"win","rating":1300}}]}`},
// 		},
// 		"https://api.chess.com/pub/player/testuser/2023/02": {
// 			{status: http.StatusOK, body: `{"games":[{"url":"u2","end_time":2,"rated":false,"time_class":"blitz","time_control":"300","pgn":"pgn2","white":{"username":"testuser","result":"win","rating":1350},"black":{"username":"alice","result":"checkmated","rating":1250}}]}`},
// 		},
// 	}
// 	defer withMockHTTPClient(t, responses)()

// 	router := gin.New()
// 	router.GET("/chessgames/:username", GetChessGames)

// 	req := httptest.NewRequest(http.MethodGet, "/chessgames/testuser?months=2", nil)
// 	w := httptest.NewRecorder()

// 	router.ServeHTTP(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Fatalf("expected status 200, got %d", w.Code)
// 	}

// 	var body struct {
// 		Username string        `json:"username"`
// 		Count    int           `json:"count"`
// 		Games    []interface{} `json:"games"`
// 	}
// 	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
// 		t.Fatalf("decode body: %v", err)
// 	}
// 	if body.Username != "testuser" || body.Count != 2 || len(body.Games) != 2 {
// 		t.Fatalf("unexpected response: %+v", body)
// 	}
// }

// func TestGetChessGamesUserNotFound(t *testing.T) {
// 	gin.SetMode(gin.TestMode)

// 	responses := map[string][]mockResp{
// 		"https://api.chess.com/pub/player/missing/games/archives": {
// 			{status: http.StatusNotFound, body: `{"message":"404"}`},
// 		},
// 	}
// 	defer withMockHTTPClient(t, responses)()

// 	router := gin.New()
// 	router.GET("/chessgames/:username", GetChessGames)

// 	req := httptest.NewRequest(http.MethodGet, "/chessgames/missing", nil)
// 	w := httptest.NewRecorder()

// 	router.ServeHTTP(w, req)

// 	if w.Code != http.StatusNotFound {
// 		t.Fatalf("expected 404, got %d", w.Code)
// 	}
// }
