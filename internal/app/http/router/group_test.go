package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/emount4/udav-back/internal/app/http/middleware"
	"github.com/emount4/udav-back/internal/app/http/router"
	"github.com/emount4/udav-back/internal/app/http/testsupport"
)

func TestGroupsPreserveRoutePatternAndMiddlewareOrder(t *testing.T) {
	root := router.New(testsupport.Logger{})
	var calls []string
	record := func(name string) middleware.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	v1 := root.Group("/v1", record("version"))
	users := v1.Group("/users", record("feature"))
	users.Handle(http.MethodGet, "/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "handler")
		if r.Pattern != "GET /v1/users/{id}" || r.PathValue("id") != "player" {
			t.Fatalf("unexpected pattern or path value: %q %q", r.Pattern, r.PathValue("id"))
		}
		w.WriteHeader(http.StatusNoContent)
	}), record("route"))

	handler := middleware.Chain(root, record("global"))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/users/player", nil))
	want := []string{"global", "version", "feature", "route", "handler"}
	if recorder.Code != http.StatusNoContent || !slices.Equal(calls, want) {
		t.Fatalf("got status %d and chain %v, want %v", recorder.Code, calls, want)
	}
}

type testRoleKey struct{}

func TestGroupsScopeAuthenticationAndAccessChecks(t *testing.T) {
	root := router.New(testsupport.Logger{})
	authenticate := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := r.Header.Get("X-Test-Role")
			if role == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), testRoleKey{}, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	requireAdmin := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Context().Value(testRoleKey{}) != "admin" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	v1 := root.Group("/v1")
	private := v1.Group("", authenticate)
	private.Handle(http.MethodGet, "/me", ok)
	private.Group("/admin").Handle(http.MethodGet, "", ok, requireAdmin)
	v1.Handle(http.MethodGet, "/public", ok)
	root.Group("/v2").Handle(http.MethodGet, "/me", ok)

	for _, test := range []struct {
		path string
		role string
		want int
	}{
		{"/v1/admin", "", http.StatusUnauthorized},
		{"/v1/admin", "member", http.StatusForbidden},
		{"/v1/admin", "admin", http.StatusNoContent},
		{"/v1/me", "", http.StatusUnauthorized},
		{"/v1/me", "member", http.StatusNoContent},
		{"/v1/public", "", http.StatusNoContent},
		{"/v2/me", "", http.StatusNoContent},
	} {
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		req.Header.Set("X-Test-Role", test.role)
		recorder := httptest.NewRecorder()
		root.ServeHTTP(recorder, req)
		if recorder.Code != test.want {
			t.Errorf("%s with role %q returned %d, want %d", test.path, test.role, recorder.Code, test.want)
		}
	}
}
