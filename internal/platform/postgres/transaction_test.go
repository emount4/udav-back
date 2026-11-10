package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emount4/udav-back/internal/core/apperror"
	"github.com/emount4/udav-back/internal/core/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgproto3"
)

func TestWithinTxCommitsQueriesOnOneConnection(t *testing.T) {
	pool := openTestPool(t,
		command("begin", "BEGIN", 'T'),
		command(testUpdate, "UPDATE 2", 'T'),
		command("commit", "COMMIT", 'I'),
	)

	var manager db.Transactor = pool
	ctx := t.Context()
	err := manager.WithinTx(ctx, func(tx db.DBTX) error {
		result, err := tx.Exec(ctx, testUpdate)
		if err == nil && (result == nil || result.RowsAffected() != 2) {
			t.Error("unexpected update result")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWithinTxRollsBackOnErrorOrCancellation(t *testing.T) {
	callbackErr := apperror.New(apperror.Conflict, "PROFILE_VERSION_CONFLICT", "Профиль уже изменился")
	for _, test := range []struct {
		name   string
		cancel bool
		want   error
		stage  string
	}{
		{"callback error", false, callbackErr, "execute postgres transaction"},
		{"canceled context", true, context.Canceled, "interrupted before commit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool := openTestPool(t, command("begin", "BEGIN", 'T'), command("rollback", "ROLLBACK", 'I'))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			err := pool.WithinTx(ctx, func(db.DBTX) error {
				if test.cancel {
					cancel()
					return nil
				}
				return test.want
			})
			if !errors.Is(err, test.want) || !strings.Contains(err.Error(), test.stage) {
				t.Fatalf("got %v, want %q wrapping %v", err, test.stage, test.want)
			}
			if !test.cancel {
				var expected *apperror.Error
				if !errors.As(err, &expected) || expected != callbackErr {
					t.Fatalf("application error lost: %v", err)
				}
			}
		})
	}
}

func TestWithinTxRollsBackOnPanic(t *testing.T) {
	pool := openTestPool(t, command("begin", "BEGIN", 'T'), command("rollback", "ROLLBACK", 'I'))
	defer func() {
		if got := recover(); got != "callback panic" {
			t.Fatalf("unexpected panic: %v", got)
		}
	}()
	_ = pool.WithinTx(t.Context(), func(db.DBTX) error { panic("callback panic") })
}

func TestWithinTxPreservesCallbackErrorWhenRollbackFails(t *testing.T) {
	pool := openTestPool(t, command("begin", "BEGIN", 'T'), testQuery{
		sql:      "rollback",
		txStatus: 'I',
		response: []pgproto3.BackendMessage{&pgproto3.ErrorResponse{
			Severity: "ERROR", Code: "08006", Message: "rollback failed",
		}},
	})
	callbackErr := errors.New("reject update")
	err := pool.WithinTx(t.Context(), func(db.DBTX) error { return callbackErr })
	if !errors.Is(err, callbackErr) || !strings.Contains(err.Error(), "rollback postgres transaction") {
		t.Fatalf("expected callback and rollback errors, got %v", err)
	}
}

func TestWithinTxReportsCommitFailure(t *testing.T) {
	pool := openTestPool(t, command("begin", "BEGIN", 'T'), command("commit", "ROLLBACK", 'I'))
	err := pool.WithinTx(t.Context(), func(db.DBTX) error { return nil })
	if !errors.Is(err, pgx.ErrTxCommitRollback) {
		t.Fatalf("expected commit failure, got %v", err)
	}
}
