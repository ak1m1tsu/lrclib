package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ak1m1tsu/lrclib/internal/errs"
)

func TestKindExitCode(t *testing.T) {
	tests := []struct {
		kind     errs.Kind
		wantCode int
	}{
		{errs.KindNotFound, 1},
		{errs.KindNetwork, 2},
		{errs.KindRateLimited, 3},
		{errs.KindInternal, 4},
		{errs.KindBadInput, 5},
	}

	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			if got := tt.kind.ExitCode(); got != tt.wantCode {
				t.Errorf("ExitCode() = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

func TestKindString(t *testing.T) {
	tests := []struct {
		kind errs.Kind
		want string
	}{
		{errs.KindNotFound, "not_found"},
		{errs.KindNetwork, "network"},
		{errs.KindRateLimited, "rate_limited"},
		{errs.KindInternal, "internal"},
		{errs.KindBadInput, "bad_input"},
	}

	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestAppError_Error(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := errs.New(errs.KindNotFound, "lyrics not found")
		if err.Error() != "lyrics not found" {
			t.Errorf("unexpected Error(): %q", err.Error())
		}
	})

	t.Run("with cause", func(t *testing.T) {
		cause := fmt.Errorf("connection refused")
		err := errs.Network("request failed", cause)
		want := "request failed: connection refused"
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})
}

func TestAppError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := errs.Internal("something went wrong", cause)

	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the wrapped cause")
	}
}

func TestAppError_Is(t *testing.T) {
	err := errs.NotFound("track missing")

	if !errors.Is(err, errs.New(errs.KindNotFound, "")) {
		t.Error("errors.Is should match same Kind")
	}

	if errors.Is(err, errs.New(errs.KindNetwork, "")) {
		t.Error("errors.Is should not match different Kind")
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name string
		err  *errs.AppError
		kind errs.Kind
	}{
		{"NotFound", errs.NotFound("msg"), errs.KindNotFound},
		{"Network", errs.Network("msg", nil), errs.KindNetwork},
		{"RateLimited", errs.RateLimited("msg"), errs.KindRateLimited},
		{"Internal", errs.Internal("msg", nil), errs.KindInternal},
		{"BadInput", errs.BadInput("msg"), errs.KindBadInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Kind != tt.kind {
				t.Errorf("got kind %v, want %v", tt.err.Kind, tt.kind)
			}
		})
	}
}
