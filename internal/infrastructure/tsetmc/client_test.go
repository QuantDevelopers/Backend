package tsetmc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	domaintsetmc "github.com/QuantDevelopers/Backend/internal/domain/tsetmc"
	infratsetmc "github.com/QuantDevelopers/Backend/internal/infrastructure/tsetmc"
	"github.com/stretchr/testify/suite"
)

type HTTPClientTestSuite struct {
	suite.Suite
}

func (s *HTTPClientTestSuite) TestSuccessAndPoliteHeaders() {
	var gotUA, gotReferer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotReferer = r.Header.Get("Referer")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := infratsetmc.NewHTTPClient(infratsetmc.ClientOptions{
		RateLimitRPS: 100,
		Burst:        10,
		MaxRetries:   3,
		BackoffBase:  time.Millisecond,
	}, nil)

	body, status, err := client.Get(context.Background(), srv.URL)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"ok":true}`, string(body))
	s.Contains(gotUA, "Mozilla/5.0")
	s.Equal("https://tsetmc.com/", gotReferer)
	s.Equal(0, client.ConsecutiveErrors())
}

func (s *HTTPClientTestSuite) TestRetryOn503ThenRecover() {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"recovered":true}`))
	}))
	defer srv.Close()

	client := infratsetmc.NewHTTPClient(infratsetmc.ClientOptions{
		RateLimitRPS:         100,
		Burst:                10,
		MaxRetries:           3,
		BackoffBase:          time.Millisecond,
		BackoffFactor:        1.5,
		MaxConsecutiveErrors: 5,
	}, nil)

	body, status, err := client.Get(context.Background(), srv.URL)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, status)
	s.JSONEq(`{"recovered":true}`, string(body))
	s.Equal(int32(3), attempts.Load())
	s.Equal(0, client.ConsecutiveErrors())
}

func (s *HTTPClientTestSuite) TestCircuitBreakerAnd4xxNoRetry() {
	var attempts404 atomic.Int32
	srv404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts404.Add(1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer srv404.Close()

	client := infratsetmc.NewHTTPClient(infratsetmc.ClientOptions{
		RateLimitRPS:         100,
		Burst:                10,
		MaxRetries:           3,
		BackoffBase:          time.Millisecond,
		MaxConsecutiveErrors: 3,
	}, nil)

	_, status, err := client.Get(context.Background(), srv404.URL)
	s.Require().Error(err)
	s.Equal(http.StatusNotFound, status)
	s.Equal(int32(1), attempts404.Load())

	srv500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv500.Close()

	_, _, err = client.Get(context.Background(), srv500.URL)
	s.Require().ErrorIs(err, domaintsetmc.ErrCircuitBreakerOpen)
	s.GreaterOrEqual(client.ConsecutiveErrors(), 3)

	// Immediate rejection without calling upstream until reset.
	_, _, err = client.Get(context.Background(), srv404.URL)
	s.Require().ErrorIs(err, domaintsetmc.ErrCircuitBreakerOpen)
	s.Equal(int32(1), attempts404.Load())

	client.ResetCircuitBreaker()
	s.Equal(0, client.ConsecutiveErrors())
}

func TestHTTPClientSuite(t *testing.T) {
	suite.Run(t, new(HTTPClientTestSuite))
}
